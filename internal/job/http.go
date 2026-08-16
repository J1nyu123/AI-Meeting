package job

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"strings"
	"time"

	"ai-meeting-go/internal/auth"
	"ai-meeting-go/internal/interview"
	"ai-meeting-go/internal/platform/httpx"
	"ai-meeting-go/internal/resume"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
	"gorm.io/gorm"
)

type Handler struct {
	db        *gorm.DB
	storage   resume.FileStorage
	publisher *Publisher
	redis     *redis.Client
	maxBytes  int64
}

func NewHandler(db *gorm.DB, storage resume.FileStorage, publisher *Publisher, redisClient *redis.Client, max int64) *Handler {
	return &Handler{db: db, storage: storage, publisher: publisher, redis: redisClient, maxBytes: max}
}
func (h *Handler) Upload(c *gin.Context) {
	userID := auth.UserID(c)
	sessionID := c.Param("id")
	var session interview.Session
	if err := h.db.WithContext(c).Where("id=? AND user_id=?", sessionID, userID).First(&session).Error; err != nil {
		httpx.Fail(c, httpx.NotFound("INTERVIEW_NOT_FOUND", "interview not found"))
		return
	}
	if session.Status != interview.StatusCreated && session.Status != interview.StatusFailed {
		httpx.Fail(c, httpx.Conflict("INVALID_STATUS", "resume can only be uploaded before analysis"))
		return
	}
	c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, h.maxBytes+1<<20)
	file, header, err := c.Request.FormFile("resume")
	if err != nil {
		httpx.Fail(c, httpx.BadRequest("RESUME_REQUIRED", "resume PDF is required"))
		return
	}
	defer file.Close()
	if !strings.EqualFold(header.Header.Get("Content-Type"), "application/pdf") && !strings.HasSuffix(strings.ToLower(header.Filename), ".pdf") {
		httpx.Fail(c, httpx.BadRequest("INVALID_PDF", "only PDF files are accepted"))
		return
	}
	prefix := make([]byte, 5)
	n, err := io.ReadFull(file, prefix)
	if err != nil || n < 5 || !bytes.Equal(prefix, []byte("%PDF-")) {
		httpx.Fail(c, httpx.BadRequest("INVALID_PDF", "invalid PDF signature"))
		return
	}
	reader := io.MultiReader(bytes.NewReader(prefix), bufio.NewReader(file))
	key, size, digest, err := h.storage.Save(reader, header.Filename)
	if err != nil || size > h.maxBytes {
		httpx.Fail(c, httpx.BadRequest("RESUME_TOO_LARGE", "resume exceeds size limit"))
		return
	}
	asset := resume.Asset{ID: uuid.NewString(), SessionID: sessionID, OriginalName: header.Filename, StorageKey: key, ContentType: "application/pdf", SizeBytes: size, SHA256: digest, CreatedAt: time.Now()}
	record := AnalysisJob{ID: uuid.NewString(), SessionID: sessionID, UserID: userID, Status: "QUEUED", Stage: "queued", PromptVersion: "v1"}
	err = h.db.WithContext(c).Transaction(func(tx *gorm.DB) error {
		tx.Where("session_id=?", sessionID).Delete(&resume.Asset{})
		if err := tx.Create(&asset).Error; err != nil {
			return err
		}
		if err := tx.Create(&record).Error; err != nil {
			return err
		}
		return tx.Model(&session).Updates(map[string]any{"status": interview.StatusAnalyzing, "version": gorm.Expr("version + 1")}).Error
	})
	if err != nil {
		httpx.Fail(c, err)
		return
	}
	if err := h.publisher.Publish(c, record.ID); err != nil {
		h.db.WithContext(c).Model(&record).Updates(map[string]any{"status": "FAILED", "error_code": "QUEUE_UNAVAILABLE", "error_message": err.Error()})
		httpx.Fail(c, err)
		return
	}
	httpx.Accepted(c, record)
}
func (h *Handler) Get(c *gin.Context) {
	var record AnalysisJob
	if err := h.db.WithContext(c).Where("id=? AND user_id=?", c.Param("jobId"), auth.UserID(c)).First(&record).Error; err != nil {
		httpx.Fail(c, httpx.NotFound("JOB_NOT_FOUND", "job not found"))
		return
	}
	httpx.OK(c, record)
}
func (h *Handler) Events(c *gin.Context) {
	jobID := c.Param("jobId")
	userID := auth.UserID(c)
	c.Header("Content-Type", "text/event-stream")
	c.Header("Cache-Control", "no-cache")
	c.Header("Connection", "keep-alive")
	ticker := time.NewTicker(time.Second)
	defer ticker.Stop()
	var notifications <-chan *redis.Message
	var subscription *redis.PubSub
	if h.redis != nil {
		subscription = h.redis.Subscribe(c, eventChannel(jobID))
		defer subscription.Close()
		notifications = subscription.Channel()
	}
	last := ""
	for {
		var record AnalysisJob
		if err := h.db.WithContext(c).Where("id=? AND user_id=?", jobID, userID).First(&record).Error; err != nil {
			return
		}
		data, _ := json.Marshal(record)
		signature := fmt.Sprintf("%s:%d:%s", record.Status, record.Progress, record.UpdatedAt)
		if signature != last {
			fmt.Fprintf(c.Writer, "id: %d\nevent: status\ndata: %s\n\n", record.UpdatedAt.UnixMilli(), data)
			c.Writer.Flush()
			last = signature
		}
		if record.Status == "COMPLETED" || record.Status == "FAILED" {
			return
		}
		select {
		case <-c.Request.Context().Done():
			return
		case <-ticker.C:
		case <-notifications:
		}
	}
}

func (h *Handler) Preview(c *gin.Context) {
	var asset resume.Asset
	if err := h.db.WithContext(c).Table("resume_assets AS r").Select("r.*").Joins("JOIN interview_sessions AS s ON s.id = r.session_id").Where("r.session_id = ? AND s.user_id = ?", c.Param("id"), auth.UserID(c)).First(&asset).Error; err != nil {
		httpx.Fail(c, httpx.NotFound("RESUME_NOT_FOUND", "resume not found"))
		return
	}
	file, err := h.storage.Open(asset.StorageKey)
	if err != nil {
		httpx.Fail(c, err)
		return
	}
	defer file.Close()
	c.Header("Content-Type", "application/pdf")
	c.Header("Content-Disposition", fmt.Sprintf("inline; filename=%q", asset.OriginalName))
	_, _ = io.Copy(c.Writer, file)
}

var _ multipart.File
