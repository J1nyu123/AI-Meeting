package media

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"
	"unicode/utf8"

	"ai-meeting-go/internal/platform/httpx"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

type TTSTask struct {
	ID              string     `gorm:"type:char(36);primaryKey" json:"taskId"`
	UserID          uint64     `gorm:"not null;uniqueIndex:uk_media_tts_user_idempotency" json:"-"`
	ProviderTaskID  string     `gorm:"size:128;not null;uniqueIndex" json:"-"`
	IdempotencyKey  string     `gorm:"size:128;not null;uniqueIndex:uk_media_tts_user_idempotency" json:"-"`
	SID             string     `gorm:"column:sid;size:128" json:"sid,omitempty"`
	TaskStatus      string     `gorm:"size:16;not null" json:"taskStatus"`
	ProviderCode    int        `gorm:"not null" json:"code"`
	ProviderMessage string     `gorm:"size:512" json:"message,omitempty"`
	AudioEncoding   string     `gorm:"size:16;not null" json:"audioEncoding"`
	SampleRate      int        `gorm:"not null" json:"sampleRate"`
	CreatedAt       time.Time  `json:"createdAt"`
	UpdatedAt       time.Time  `json:"updatedAt"`
	CompletedAt     *time.Time `json:"completedAt,omitempty"`
}

func (TTSTask) TableName() string { return "media_tts_tasks" }
func (t TTSTask) Completed() bool {
	return t.TaskStatus == "2" || t.TaskStatus == "4" || t.TaskStatus == "5"
}
func (t TTSTask) Success() bool { return t.TaskStatus == "5" && t.ProviderCode == 0 }
func (t TTSTask) Public() map[string]any {
	return map[string]any{"taskId": t.ID, "sid": t.SID, "taskStatus": t.TaskStatus, "code": t.ProviderCode, "message": t.ProviderMessage, "audioEncoding": t.AudioEncoding, "sampleRate": t.SampleRate, "completed": t.Completed(), "success": t.Success(), "audioPath": "/api/v1/media/tts/tasks/" + t.ID + "/audio"}
}

type TTSRequest struct {
	Text           string `json:"text"`
	VCN            string `json:"vcn"`
	Language       string `json:"language"`
	Speed          *int   `json:"speed"`
	Volume         *int   `json:"volume"`
	Pitch          *int   `json:"pitch"`
	Rhy            *int   `json:"rhy"`
	AudioEncoding  string `json:"audioEncoding"`
	SampleRate     *int   `json:"sampleRate"`
	TimeoutSeconds *int   `json:"timeoutSeconds"`
	PollIntervalMS *int   `json:"pollIntervalMs"`
}
type ProviderTTSTask struct {
	TaskID, SID, Status, Message, AudioURL string
	Code                                   int
}
type TTSProvider interface {
	Create(context.Context, TTSRequest) (ProviderTTSTask, error)
	Query(context.Context, string) (ProviderTTSTask, error)
}

type TTSService struct {
	db       *gorm.DB
	provider TTSProvider
	enabled  bool
	timeout  time.Duration
	now      func() time.Time
}

func NewTTSService(db *gorm.DB, provider TTSProvider, enabled bool, timeout time.Duration) *TTSService {
	return &TTSService{db: db, provider: provider, enabled: enabled, timeout: durationOr(timeout, 90*time.Second), now: time.Now}
}

func (s *TTSService) Create(ctx context.Context, userID uint64, key string, req TTSRequest) (TTSTask, error) {
	if !s.enabled {
		return TTSTask{}, httpx.Conflict("MEDIA_TTS_DISABLED", "remote text-to-speech is disabled")
	}
	key = strings.TrimSpace(key)
	if len(key) < 8 || len(key) > 128 {
		return TTSTask{}, httpx.BadRequest("INVALID_IDEMPOTENCY_KEY", "Idempotency-Key length must be 8-128")
	}
	req = defaults(req)
	if err := validateTTSRequest(req); err != nil {
		return TTSTask{}, err
	}
	placeholder := TTSTask{ID: uuid.NewString(), UserID: userID, ProviderTaskID: "pending:" + uuid.NewString(), IdempotencyKey: key, TaskStatus: "creating", AudioEncoding: req.AudioEncoding, SampleRate: *req.SampleRate, CreatedAt: s.now(), UpdatedAt: s.now()}
	if err := s.db.WithContext(ctx).Create(&placeholder).Error; err != nil {
		var existing TTSTask
		if findErr := s.db.WithContext(ctx).Where("user_id = ? AND idempotency_key = ?", userID, key).First(&existing).Error; findErr == nil {
			return existing, nil
		}
		return TTSTask{}, err
	}
	created, err := s.provider.Create(ctx, req)
	if err != nil {
		placeholder.TaskStatus = "4"
		placeholder.ProviderCode = -1
		placeholder.ProviderMessage = "provider task creation failed"
		now := s.now()
		placeholder.CompletedAt = &now
		_ = s.db.WithContext(ctx).Save(&placeholder).Error
		return TTSTask{}, fmt.Errorf("create TTS provider task: %w", err)
	}
	placeholder.ProviderTaskID = created.TaskID
	placeholder.SID = created.SID
	placeholder.TaskStatus = normalizeTaskStatus(created.Status, "1")
	placeholder.ProviderCode = created.Code
	placeholder.ProviderMessage = safeProviderMessage(created.Message)
	placeholder.UpdatedAt = s.now()
	if placeholder.Completed() {
		now := s.now()
		placeholder.CompletedAt = &now
	}
	if err := s.db.WithContext(ctx).Save(&placeholder).Error; err != nil {
		return TTSTask{}, err
	}
	return placeholder, nil
}
func (s *TTSService) Get(ctx context.Context, userID uint64, id string, refresh bool) (TTSTask, error) {
	var task TTSTask
	if err := s.db.WithContext(ctx).Where("id = ? AND user_id = ?", id, userID).First(&task).Error; err != nil {
		return TTSTask{}, httpx.NotFound("TTS_TASK_NOT_FOUND", "TTS task not found")
	}
	if !refresh || task.Completed() || strings.HasPrefix(task.ProviderTaskID, "pending:") {
		return task, nil
	}
	upstream, err := s.provider.Query(ctx, task.ProviderTaskID)
	if err != nil {
		return TTSTask{}, fmt.Errorf("query TTS provider task: %w", err)
	}
	task.TaskStatus = normalizeTaskStatus(upstream.Status, task.TaskStatus)
	task.ProviderCode = upstream.Code
	task.ProviderMessage = safeProviderMessage(upstream.Message)
	task.SID = upstream.SID
	task.UpdatedAt = s.now()
	if task.Completed() && task.CompletedAt == nil {
		now := s.now()
		task.CompletedAt = &now
	}
	if err := s.db.WithContext(ctx).Save(&task).Error; err != nil {
		return TTSTask{}, err
	}
	return task, nil
}
func (s *TTSService) Wait(ctx context.Context, userID uint64, id string, timeout time.Duration, poll time.Duration) (TTSTask, error) {
	deadline := time.NewTimer(timeout)
	defer deadline.Stop()
	ticker := time.NewTicker(poll)
	defer ticker.Stop()
	for {
		task, err := s.Get(ctx, userID, id, true)
		if err != nil {
			return TTSTask{}, err
		}
		if task.Completed() {
			if !task.Success() {
				return task, httpx.Conflict("TTS_SYNTHESIS_FAILED", "TTS synthesis failed")
			}
			return task, nil
		}
		select {
		case <-ctx.Done():
			return TTSTask{}, ctx.Err()
		case <-deadline.C:
			return TTSTask{}, &httpx.AppError{Status: 504, Code: "TTS_TIMEOUT", Message: "timed out waiting for TTS synthesis"}
		case <-ticker.C:
		}
	}
}
func (s *TTSService) AudioURL(ctx context.Context, userID uint64, id string) (string, TTSTask, error) {
	task, err := s.Get(ctx, userID, id, false)
	if err != nil {
		return "", TTSTask{}, err
	}
	if !task.Success() {
		return "", task, httpx.Conflict("TTS_AUDIO_NOT_READY", "TTS audio is not ready")
	}
	upstream, err := s.provider.Query(ctx, task.ProviderTaskID)
	if err != nil {
		return "", task, err
	}
	if strings.TrimSpace(upstream.AudioURL) == "" {
		return "", task, httpx.Conflict("TTS_AUDIO_NOT_READY", "TTS audio URL is unavailable")
	}
	return upstream.AudioURL, task, nil
}

func defaults(req TTSRequest) TTSRequest {
	if req.VCN == "" {
		req.VCN = "x4_mingge"
	}
	if req.Language == "" {
		req.Language = "zh"
	}
	if req.Speed == nil {
		v := 50
		req.Speed = &v
	}
	if req.Volume == nil {
		v := 50
		req.Volume = &v
	}
	if req.Pitch == nil {
		v := 50
		req.Pitch = &v
	}
	if req.Rhy == nil {
		v := 0
		req.Rhy = &v
	}
	if req.AudioEncoding == "" {
		req.AudioEncoding = "lame"
	}
	if req.SampleRate == nil {
		v := 16000
		req.SampleRate = &v
	}
	if req.TimeoutSeconds == nil {
		v := 90
		req.TimeoutSeconds = &v
	}
	if req.PollIntervalMS == nil {
		v := 1500
		req.PollIntervalMS = &v
	}
	return req
}
func validateTTSRequest(req TTSRequest) error {
	if strings.TrimSpace(req.Text) == "" || !utf8.ValidString(req.Text) {
		return httpx.BadRequest("INVALID_TTS_TEXT", "text must be valid non-empty UTF-8")
	}
	if utf8.RuneCountInString(req.Text) > 100000 {
		return httpx.BadRequest("TTS_TEXT_TOO_LONG", "text must not exceed 100000 characters")
	}
	for name, value := range map[string]int{"speed": *req.Speed, "volume": *req.Volume, "pitch": *req.Pitch} {
		if value < 0 || value > 100 {
			return httpx.BadRequest("INVALID_TTS_PARAMETER", name+" must be between 0 and 100")
		}
	}
	if *req.Rhy != 0 && *req.Rhy != 1 {
		return httpx.BadRequest("INVALID_TTS_PARAMETER", "rhy must be 0 or 1")
	}
	allowed := map[string]bool{"raw": true, "pcm": true, "lame": true, "mp3": true, "speex": true, "opus": true}
	if !allowed[strings.ToLower(req.AudioEncoding)] {
		return httpx.BadRequest("INVALID_TTS_PARAMETER", "unsupported audioEncoding")
	}
	if *req.SampleRate != 8000 && *req.SampleRate != 16000 && *req.SampleRate != 24000 {
		return httpx.BadRequest("INVALID_TTS_PARAMETER", "sampleRate must be 8000, 16000 or 24000")
	}
	if *req.TimeoutSeconds < 10 || *req.TimeoutSeconds > 120 {
		return httpx.BadRequest("INVALID_TTS_PARAMETER", "timeoutSeconds must be between 10 and 120")
	}
	if *req.PollIntervalMS < 500 || *req.PollIntervalMS > 5000 {
		return httpx.BadRequest("INVALID_TTS_PARAMETER", "pollIntervalMs must be between 500 and 5000")
	}
	return nil
}
func normalizeTaskStatus(value, fallback string) string {
	value = strings.TrimSpace(value)
	switch value {
	case "1", "2", "3", "4", "5":
		return value
	}
	return fallback
}
func safeProviderMessage(value string) string {
	value = strings.TrimSpace(value)
	if len(value) > 512 {
		return value[:512]
	}
	return value
}

var _ = errors.Is
