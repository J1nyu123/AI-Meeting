package interview

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"ai-meeting-go/internal/evaluation"
	"ai-meeting-go/internal/llm"
	"ai-meeting-go/internal/platform/httpx"
	"ai-meeting-go/internal/resume"
	"github.com/google/uuid"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type Service struct {
	db      *gorm.DB
	runtime *RuntimeStore
	llm     llm.Client
	files   resume.FileStorage
	now     func() time.Time
}

func NewService(db *gorm.DB, runtime *RuntimeStore, client llm.Client) *Service {
	return &Service{db: db, runtime: runtime, llm: client, now: time.Now}
}
func (s *Service) WithFileStorage(files resume.FileStorage) *Service {
	s.files = files
	return s
}
func (s *Service) Create(ctx context.Context, userID uint64) (Session, error) {
	session := Session{ID: uuid.NewString(), UserID: userID, Status: StatusCreated}
	if err := s.db.WithContext(ctx).Create(&session).Error; err != nil {
		return Session{}, err
	}
	return session, nil
}

func (s *Service) State(ctx context.Context, userID uint64, id string) (StateView, bool, error) {
	var session Session
	if err := s.db.WithContext(ctx).Where("id = ? AND user_id = ?", id, userID).First(&session).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return StateView{}, true, httpx.NotFound("INTERVIEW_NOT_FOUND", "interview not found")
		}
		return StateView{}, true, err
	}
	state, degraded, err := Rehydrate(ctx, s.runtime, session)
	if err != nil {
		return StateView{}, degraded, err
	}
	var current *QuestionView
	if state.CurrentQuestionNumber != "" {
		var q Question
		if err := s.db.WithContext(ctx).Where("session_id=? AND number=?", id, state.CurrentQuestionNumber).First(&q).Error; err == nil {
			current = &QuestionView{Number: q.Number, Content: q.Content, IsFollowUp: q.IsFollowUp}
		}
	}
	var mainTotal int64
	s.db.WithContext(ctx).Model(&Question{}).Where("session_id=? AND is_follow_up=?", id, false).Count(&mainTotal)
	var turns []Turn
	s.db.WithContext(ctx).Where("session_id=?", id).Order("sequence ASC").Find(&turns)
	views := make([]TurnView, 0, len(turns))
	for _, t := range turns {
		views = append(views, TurnView{t.Sequence, t.QuestionNumber, t.QuestionContent, t.AnswerContent, t.Score, t.Feedback, t.IsFollowUp})
	}
	return StateView{SessionID: id, Status: state.Status, CanResume: state.Status == StatusReady || state.Status == StatusInProgress, Version: state.Version, ResumeScore: session.ResumeScore, InterviewType: session.Direction, ResumeFileURL: "/api/v1/interviews/" + id + "/resume", CurrentQuestion: current, Progress: Progress{state.CurrentMainIndex, mainTotal, state.FollowUpCount, state.TotalScore}, RecentTurns: views}, degraded, nil
}

func (s *Service) Delete(ctx context.Context, userID uint64, id string) (bool, error) {
	var storageKey string
	err := s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var session Session
		err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).Where("id = ? AND user_id = ?", id, userID).First(&session).Error
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return httpx.NotFound("INTERVIEW_NOT_FOUND", "interview not found")
		}
		if err != nil {
			return err
		}
		if session.Status != StatusCompleted && session.Status != StatusFailed {
			return httpx.Conflict("INTERVIEW_DELETE_NOT_ALLOWED", "only completed or failed interviews can be deleted")
		}

		var asset resume.Asset
		assetErr := tx.Select("storage_key").Where("session_id = ?", id).First(&asset).Error
		if assetErr == nil {
			storageKey = asset.StorageKey
		} else if !errors.Is(assetErr, gorm.ErrRecordNotFound) {
			return assetErr
		}

		for _, table := range []string{
			"interview_reports",
			"interview_turns",
			"answer_attempts",
			"interview_questions",
			"analysis_jobs",
			"resume_assets",
		} {
			if err := tx.Exec("DELETE FROM "+table+" WHERE session_id = ?", id).Error; err != nil {
				return err
			}
		}
		return tx.Where("id = ? AND user_id = ?", id, userID).Delete(&Session{}).Error
	})
	if err != nil {
		return false, err
	}

	if storageKey != "" && s.files != nil {
		if err := s.files.Delete(storageKey); err != nil {
			slog.Warn("interview_resume_cleanup_failed", "session_id", id, "storage_key", storageKey, "error", err)
		}
	}
	degraded := false
	if s.runtime != nil {
		if err := s.runtime.Delete(ctx, id); err != nil {
			degraded = true
			slog.Warn("interview_runtime_cleanup_failed", "session_id", id, "error", err)
		}
	}
	return degraded, nil
}

func (s *Service) Answer(ctx context.Context, userID uint64, sessionID, idempotencyKey, questionNumber, answer string) (AnswerResponse, bool, error) {
	if idempotencyKey == "" {
		return AnswerResponse{}, false, httpx.BadRequest("IDEMPOTENCY_KEY_REQUIRED", "Idempotency-Key header is required")
	}
	if len(answer) == 0 || len([]rune(answer)) > 5000 {
		return AnswerResponse{}, false, httpx.BadRequest("INVALID_ANSWER", "answer length must be 1-5000")
	}
	now := s.now()
	lease := now.Add(time.Minute)
	attempt := AnswerAttempt{SessionID: sessionID, UserID: userID, IdempotencyKey: idempotencyKey, QuestionNumber: questionNumber, Status: "PROCESSING", LeaseUntil: &lease}
	adoptExisting := func(existing AnswerAttempt) (AnswerResponse, bool, error) {
		if existing.Status == "SUCCEEDED" {
			var replay AnswerResponse
			if json.Unmarshal(existing.ResponseJSON, &replay) == nil {
				return replay, true, nil
			}
		}
		if existing.LeaseUntil != nil && existing.LeaseUntil.After(now) {
			return AnswerResponse{}, true, httpx.Conflict("ANSWER_IN_PROGRESS", "answer is already being processed")
		}
		if err := s.db.WithContext(ctx).Model(&existing).Updates(map[string]any{"status": "PROCESSING", "lease_until": lease, "error_code": ""}).Error; err != nil {
			return AnswerResponse{}, true, err
		}
		attempt = existing
		return AnswerResponse{}, false, nil
	}
	var existing AnswerAttempt
	lookupErr := s.db.WithContext(ctx).Where("session_id=? AND idempotency_key=?", sessionID, idempotencyKey).First(&existing).Error
	if lookupErr == nil {
		response, handled, err := adoptExisting(existing)
		if handled || err != nil {
			return response, false, err
		}
	} else if !errors.Is(lookupErr, gorm.ErrRecordNotFound) {
		return AnswerResponse{}, false, lookupErr
	} else if createErr := s.db.WithContext(ctx).Create(&attempt).Error; createErr != nil {
		if err := s.db.WithContext(ctx).Where("session_id=? AND idempotency_key=?", sessionID, idempotencyKey).First(&existing).Error; err != nil {
			return AnswerResponse{}, false, createErr
		}
		response, handled, err := adoptExisting(existing)
		if handled || err != nil {
			return response, false, err
		}
	}
	completedAttempt := false
	defer func() {
		if !completedAttempt && attempt.ID != 0 {
			s.failAttempt(context.Background(), attempt.ID, "REQUEST_ABORTED")
		}
	}()
	owner := uuid.NewString()
	locked, lockErr := s.runtime.Acquire(ctx, sessionID+":"+questionNumber, owner, 50*time.Second)
	degraded := lockErr != nil
	if lockErr == nil && !locked {
		return AnswerResponse{}, false, httpx.Conflict("QUESTION_LOCKED", "question is being evaluated")
	}
	if locked {
		defer s.runtime.Release(context.Background(), sessionID+":"+questionNumber, owner)
	}
	var session Session
	if err := s.db.WithContext(ctx).Where("id=? AND user_id=?", sessionID, userID).First(&session).Error; err != nil {
		return AnswerResponse{}, degraded, httpx.NotFound("INTERVIEW_NOT_FOUND", "interview not found")
	}
	if session.Status != StatusReady && session.Status != StatusInProgress {
		return AnswerResponse{}, degraded, httpx.Conflict("INTERVIEW_NOT_ACTIVE", "interview is not active")
	}
	if session.CurrentQuestionNumber != questionNumber {
		return AnswerResponse{}, degraded, httpx.Conflict("STALE_QUESTION", "submitted question is not current")
	}
	var question Question
	if err := s.db.WithContext(ctx).Where("session_id=? AND number=?", sessionID, questionNumber).First(&question).Error; err != nil {
		return AnswerResponse{}, degraded, httpx.NotFound("QUESTION_NOT_FOUND", "question not found")
	}
	resumeContext, err := s.loadResumeContext(ctx, sessionID)
	if err != nil {
		return AnswerResponse{}, degraded, err
	}
	evalCtx, cancel := context.WithTimeout(ctx, 45*time.Second)
	defer cancel()
	evaluated, err := s.llm.EvaluateAnswer(evalCtx, llm.AnswerInput{SessionID: sessionID, Question: question.Content, Answer: answer, ResumeContext: resumeContext})
	if err != nil {
		s.failAttempt(ctx, attempt.ID, "AI_EVALUATION_FAILED")
		completedAttempt = true
		return AnswerResponse{}, degraded, fmt.Errorf("evaluate answer: %w", err)
	}
	decision := evaluation.Decide(evaluated, session.FollowUpCount, 2)
	if decision.NeedFollowUp {
		generated, generateErr := s.llm.GenerateFollowUp(evalCtx, llm.FollowUpInput{
			SessionID: sessionID, Question: question.Content, Answer: answer, ResumeContext: resumeContext,
			Current: session.FollowUpCount, Maximum: 2,
		})
		if generateErr == nil && !generated.EndInterview && strings.TrimSpace(generated.Question) != "" {
			evaluated.FollowUpQuestion = generated.Question
		} else if generateErr != nil {
			slog.Warn("follow_up_generation_fallback", "session_id", sessionID, "error", generateErr)
		}
	}
	response := AnswerResponse{QuestionNumber: questionNumber, Score: evaluated.Score, Feedback: evaluated.Feedback, MissingPoints: evaluated.MissingPoints, FollowUpNeeded: decision.NeedFollowUp}
	err = s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var fresh Session
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).Where("id=? AND user_id=?", sessionID, userID).First(&fresh).Error; err != nil {
			return err
		}
		if fresh.Version != session.Version || fresh.CurrentQuestionNumber != questionNumber {
			return httpx.Conflict("SESSION_CHANGED", "interview state changed")
		}
		fresh.TurnSequence++
		if !question.IsFollowUp {
			fresh.TotalScore += evaluated.Score
			fresh.CurrentMainIndex = question.Ordinal
		}
		next, err := s.resolveNext(tx, &fresh, question, evaluated, decision)
		if err != nil {
			return err
		}
		fresh.Version++
		if fresh.Status == StatusReady {
			fresh.Status = StatusInProgress
			start := now
			fresh.StartedAt = &start
		}
		if next == nil {
			fresh.Status = StatusCompleted
			done := now
			fresh.CompletedAt = &done
		}
		missing, _ := json.Marshal(evaluated.MissingPoints)
		turn := Turn{SessionID: sessionID, Sequence: fresh.TurnSequence, RequestID: idempotencyKey, QuestionNumber: question.Number, QuestionContent: question.Content, AnswerContent: answer, Score: evaluated.Score, Feedback: evaluated.Feedback, MissingPoints: missing, IsFollowUp: question.IsFollowUp, CreatedAt: now}
		if err := tx.Create(&turn).Error; err != nil {
			return err
		}
		updated := tx.Model(&Session{}).Where("id=? AND version=?", fresh.ID, session.Version).Updates(map[string]any{"status": fresh.Status, "current_question_number": fresh.CurrentQuestionNumber, "current_main_index": fresh.CurrentMainIndex, "follow_up_count": fresh.FollowUpCount, "total_score": fresh.TotalScore, "turn_sequence": fresh.TurnSequence, "version": fresh.Version, "started_at": fresh.StartedAt, "completed_at": fresh.CompletedAt, "updated_at": now})
		if updated.Error != nil {
			return updated.Error
		}
		if updated.RowsAffected != 1 {
			return httpx.Conflict("SESSION_CHANGED", "interview state changed")
		}
		response.TotalScore = fresh.TotalScore
		response.FollowUpCount = fresh.FollowUpCount
		response.Finished = next == nil
		response.Version = fresh.Version
		if next != nil {
			response.NextQuestion = &QuestionView{next.Number, next.Content, next.IsFollowUp}
		}
		payload, _ := json.Marshal(response)
		return tx.Model(&AnswerAttempt{}).Where("id=?", attempt.ID).Updates(map[string]any{"status": "SUCCEEDED", "response_json": payload, "lease_until": nil, "updated_at": now}).Error
	})
	if err != nil {
		s.failAttempt(ctx, attempt.ID, "COMMIT_FAILED")
		completedAttempt = true
		return AnswerResponse{}, degraded, err
	}
	completedAttempt = true
	var saved Session
	s.db.WithContext(ctx).First(&saved, "id=?", sessionID)
	_ = s.runtime.Set(ctx, RuntimeState{SessionID: saved.ID, Status: saved.Status, CurrentQuestionNumber: saved.CurrentQuestionNumber, CurrentMainIndex: saved.CurrentMainIndex, FollowUpCount: saved.FollowUpCount, TotalScore: saved.TotalScore, TurnSequence: saved.TurnSequence, Version: saved.Version, LastAppliedRequestID: idempotencyKey, UpdatedAt: saved.UpdatedAt})
	return response, degraded, nil
}

func (s *Service) resolveNext(tx *gorm.DB, session *Session, current Question, result evaluation.Result, decision evaluation.Decision) (*Question, error) {
	if decision.NeedFollowUp {
		count := session.FollowUpCount + 1
		number := fmt.Sprintf("%s-F%d", baseNumber(current), count)
		content := sanitizeFollowUpQuestion(result.FollowUpQuestion)
		if content == "" {
			content = "请结合具体经历进一步说明实现细节和结果。"
		}
		q := Question{ID: uuid.NewString(), SessionID: session.ID, Number: number, Content: content, IsFollowUp: true, ParentNumber: baseNumber(current), Ordinal: current.Ordinal, CreatedAt: s.now()}
		if err := tx.Create(&q).Error; err != nil {
			return nil, err
		}
		session.FollowUpCount = count
		session.CurrentQuestionNumber = q.Number
		return &q, nil
	}
	var next Question
	err := tx.Where("session_id=? AND is_follow_up=? AND ordinal>?", session.ID, false, current.Ordinal).Order("ordinal ASC").First(&next).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		session.CurrentQuestionNumber = ""
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	session.FollowUpCount = 0
	session.CurrentQuestionNumber = next.Number
	return &next, nil
}

func (s *Service) loadResumeContext(ctx context.Context, sessionID string) (string, error) {
	var asset resume.Asset
	err := s.db.WithContext(ctx).Select("extracted_text").Where("session_id=?", sessionID).First(&asset).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return "", nil
	}
	if err != nil {
		return "", err
	}
	return asset.ExtractedText, nil
}

func sanitizeFollowUpQuestion(question string) string {
	question = strings.TrimSpace(question)
	runes := []rune(question)
	if len(runes) > 100 {
		question = strings.TrimSpace(string(runes[:100]))
	}
	return question
}
func baseNumber(q Question) string {
	if q.IsFollowUp && q.ParentNumber != "" {
		return q.ParentNumber
	}
	return q.Number
}
func (s *Service) failAttempt(ctx context.Context, id uint64, code string) {
	s.db.WithContext(ctx).Model(&AnswerAttempt{}).Where("id=?", id).Updates(map[string]any{"status": "FAILED", "error_code": code, "lease_until": nil})
}
