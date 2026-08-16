package job

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"ai-meeting-go/internal/interview"
	"ai-meeting-go/internal/llm"
	"ai-meeting-go/internal/resume"
	"github.com/google/uuid"
	"github.com/hibiken/asynq"
	"github.com/redis/go-redis/v9"
	"gorm.io/gorm"
)

const TaskAnalyzeResume = "resume:analyze"

type analyzePayload struct {
	JobID string `json:"jobId"`
}
type Publisher struct{ client *asynq.Client }

func NewPublisher(opt asynq.RedisClientOpt) *Publisher {
	return &Publisher{client: asynq.NewClient(opt)}
}
func (p *Publisher) Close() error { return p.client.Close() }
func (p *Publisher) Publish(ctx context.Context, jobID string) error {
	payload, _ := json.Marshal(analyzePayload{jobID})
	_, err := p.client.EnqueueContext(ctx, asynq.NewTask(TaskAnalyzeResume, payload), asynq.MaxRetry(3), asynq.Timeout(2*time.Minute))
	return err
}

type Processor struct {
	db          *gorm.DB
	storagePath string
	extractor   resume.PDFExtractor
	llm         llm.Client
	events      *redis.Client
}

func NewProcessor(db *gorm.DB, path string, extractor resume.PDFExtractor, client llm.Client, events *redis.Client) *Processor {
	return &Processor{db: db, storagePath: path, extractor: extractor, llm: client, events: events}
}
func (p *Processor) Register(mux *asynq.ServeMux) { mux.HandleFunc(TaskAnalyzeResume, p.Handle) }
func (p *Processor) Handle(ctx context.Context, task *asynq.Task) error {
	var payload analyzePayload
	if err := json.Unmarshal(task.Payload(), &payload); err != nil {
		return fmt.Errorf("decode payload: %w", err)
	}
	var record AnalysisJob
	if err := p.db.WithContext(ctx).First(&record, "id=?", payload.JobID).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil
		}
		return err
	}
	if record.Status == "COMPLETED" {
		return nil
	}
	p.db.WithContext(ctx).Model(&record).Update("attempts", gorm.Expr("attempts + 1"))
	// A retryable dependency failure temporarily marks the session FAILED for clients.
	// When Asynq retries the same job, move only that session back into ANALYZING.
	p.db.WithContext(ctx).Model(&interview.Session{}).Where("id = ? AND status = ?", record.SessionID, interview.StatusFailed).Update("status", interview.StatusAnalyzing)
	p.update(ctx, &record, "PROCESSING", 20, "extracting_pdf", "", "")
	var asset resume.Asset
	if err := p.db.WithContext(ctx).Where("session_id=?", record.SessionID).First(&asset).Error; err != nil {
		return err
	}
	resumePath := filepath.Join(p.storagePath, asset.StorageKey)
	text, err := p.extractor.Extract(resumePath)
	if err != nil {
		p.fail(ctx, &record, "PDF_OCR_REQUIRED", err.Error())
		return asynq.SkipRetry
	}
	p.db.WithContext(ctx).Model(&asset).Update("extracted_text", text)
	p.update(ctx, &record, "PROCESSING", 55, "calling_llm", "", "")
	analysis, err := p.llm.AnalyzeResume(ctx, llm.ResumeInput{
		SessionID:    record.SessionID,
		Text:         text,
		FilePath:     resumePath,
		OriginalName: asset.OriginalName,
	})
	if err != nil {
		p.fail(ctx, &record, "LLM_ANALYSIS_FAILED", err.Error())
		return err
	}
	err = p.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var session interview.Session
		if err := tx.Where("id=? AND user_id=?", record.SessionID, record.UserID).First(&session).Error; err != nil {
			return err
		}
		if session.Status != interview.StatusAnalyzing {
			return nil
		}
		for i, draft := range analysis.Questions {
			number := draft.Number
			if number == "" {
				number = fmt.Sprint(i + 1)
			}
			q := interview.Question{ID: uuid.NewString(), SessionID: session.ID, Number: number, Content: draft.Content, Suggestion: draft.Suggestion, Ordinal: i + 1, CreatedAt: time.Now()}
			if err := tx.Create(&q).Error; err != nil {
				return err
			}
		}
		first := ""
		if len(analysis.Questions) > 0 {
			first = analysis.Questions[0].Number
			if first == "" {
				first = "1"
			}
		}
		return tx.Model(&session).Updates(map[string]any{"status": interview.StatusReady, "direction": analysis.Direction, "resume_score": analysis.ResumeScore, "current_question_number": first, "version": gorm.Expr("version + 1")}).Error
	})
	if err != nil {
		return err
	}
	p.update(ctx, &record, "COMPLETED", 100, "completed", "", "")
	return nil
}
func (p *Processor) update(ctx context.Context, j *AnalysisJob, status string, progress int, stage, code, message string) {
	result := p.db.WithContext(ctx).Model(j).Updates(map[string]any{"status": status, "progress": progress, "stage": stage, "error_code": code, "error_message": message})
	if result.Error == nil && p.events != nil {
		_ = p.events.Publish(ctx, eventChannel(j.ID), status).Err()
	}
}
func (p *Processor) fail(ctx context.Context, j *AnalysisJob, code, message string) {
	p.update(ctx, j, "FAILED", 100, "failed", code, message)
	p.db.WithContext(ctx).Model(&interview.Session{}).Where("id=?", j.SessionID).Update("status", interview.StatusFailed)
}

var _ = os.ErrNotExist

func eventChannel(jobID string) string { return "analysis_job:" + jobID }
