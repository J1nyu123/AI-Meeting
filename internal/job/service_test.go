package job

import (
	"context"
	"errors"
	"path/filepath"
	"testing"

	"ai-meeting-go/internal/evaluation"
	"ai-meeting-go/internal/interview"
	"ai-meeting-go/internal/llm"
	"ai-meeting-go/internal/resume"
	"github.com/google/uuid"
	"github.com/hibiken/asynq"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

type fakeExtractor struct {
	text string
	err  error
}

func (f fakeExtractor) Extract(string) (string, error) { return f.text, f.err }

type fakeLLM struct {
	analysis llm.ResumeAnalysis
	err      error
	inspect  func(llm.ResumeInput)
}

func (f fakeLLM) AnalyzeResume(_ context.Context, input llm.ResumeInput) (llm.ResumeAnalysis, error) {
	if f.inspect != nil {
		f.inspect(input)
	}
	return f.analysis, f.err
}
func (fakeLLM) EvaluateAnswer(context.Context, llm.AnswerInput) (evaluation.Result, error) {
	return evaluation.Result{}, nil
}
func (fakeLLM) GenerateFollowUp(context.Context, llm.FollowUpInput) (llm.FollowUpResult, error) {
	return llm.FollowUpResult{}, nil
}

func jobDB(t *testing.T) *gorm.DB {
	db, err := gorm.Open(sqlite.Open("file:"+uuid.NewString()+"?mode=memory&cache=shared"), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, db.AutoMigrate(&interview.Session{}, &interview.Question{}, &resume.Asset{}, &AnalysisJob{}))
	return db
}

func seedJob(t *testing.T, db *gorm.DB) (interview.Session, AnalysisJob) {
	session := interview.Session{ID: uuid.NewString(), UserID: 7, Status: interview.StatusAnalyzing, Version: 1}
	require.NoError(t, db.Create(&session).Error)
	require.NoError(t, db.Create(&resume.Asset{ID: uuid.NewString(), SessionID: session.ID, OriginalName: "candidate.pdf", StorageKey: "resume.pdf"}).Error)
	record := AnalysisJob{ID: uuid.NewString(), SessionID: session.ID, UserID: session.UserID, Status: "QUEUED"}
	require.NoError(t, db.Create(&record).Error)
	return session, record
}

func taskFor(jobID string) *asynq.Task {
	return asynq.NewTask(TaskAnalyzeResume, []byte(`{"jobId":"`+jobID+`"}`))
}

func TestProcessorCompletesAndCreatesQuestions(t *testing.T) {
	db := jobDB(t)
	session, record := seedJob(t, db)
	storagePath := t.TempDir()
	client := fakeLLM{
		analysis: llm.ResumeAnalysis{Direction: "Go", ResumeScore: 88, Questions: []llm.QuestionDraft{{Number: "1", Content: "Q1"}, {Content: "Q2"}}},
		inspect: func(input llm.ResumeInput) {
			require.Equal(t, session.ID, input.SessionID)
			require.Equal(t, "extracted resume", input.Text)
			require.Equal(t, filepath.Join(storagePath, "resume.pdf"), input.FilePath)
			require.Equal(t, "candidate.pdf", input.OriginalName)
		},
	}
	processor := NewProcessor(db, storagePath, fakeExtractor{text: "extracted resume"}, client, nil)
	require.NoError(t, processor.Handle(t.Context(), taskFor(record.ID)))

	require.NoError(t, db.First(&record, "id=?", record.ID).Error)
	require.Equal(t, "COMPLETED", record.Status)
	require.Equal(t, 100, record.Progress)
	require.Equal(t, 1, record.Attempts)
	require.NoError(t, db.First(&session, "id=?", session.ID).Error)
	require.Equal(t, interview.StatusReady, session.Status)
	require.Equal(t, "Go", session.Direction)
	require.Equal(t, 88, session.ResumeScore)
	require.Equal(t, "1", session.CurrentQuestionNumber)
	var questions []interview.Question
	require.NoError(t, db.Where("session_id=?", session.ID).Order("ordinal").Find(&questions).Error)
	require.Len(t, questions, 2)
	require.Equal(t, "2", questions[1].Number)

	// A completed task is safe to replay and does not increment attempts.
	require.NoError(t, processor.Handle(t.Context(), taskFor(record.ID)))
	require.NoError(t, db.First(&record, "id=?", record.ID).Error)
	require.Equal(t, 1, record.Attempts)
}

func TestProcessorFailurePaths(t *testing.T) {
	t.Run("deleted job is treated as cancelled", func(t *testing.T) {
		processor := NewProcessor(jobDB(t), t.TempDir(), fakeExtractor{}, fakeLLM{}, nil)
		require.NoError(t, processor.Handle(t.Context(), taskFor(uuid.NewString())))
	})

	t.Run("invalid payload", func(t *testing.T) {
		processor := NewProcessor(jobDB(t), t.TempDir(), fakeExtractor{}, fakeLLM{}, nil)
		require.Error(t, processor.Handle(t.Context(), asynq.NewTask(TaskAnalyzeResume, []byte("{"))))
	})

	t.Run("ocr required", func(t *testing.T) {
		db := jobDB(t)
		session, record := seedJob(t, db)
		processor := NewProcessor(db, t.TempDir(), fakeExtractor{err: errors.New("PDF_OCR_REQUIRED")}, fakeLLM{}, nil)
		err := processor.Handle(t.Context(), taskFor(record.ID))
		require.ErrorIs(t, err, asynq.SkipRetry)
		require.NoError(t, db.First(&record, "id=?", record.ID).Error)
		require.Equal(t, "FAILED", record.Status)
		require.Equal(t, "PDF_OCR_REQUIRED", record.ErrorCode)
		require.NoError(t, db.First(&session, "id=?", session.ID).Error)
		require.Equal(t, interview.StatusFailed, session.Status)
	})

	t.Run("llm failure remains retryable", func(t *testing.T) {
		db := jobDB(t)
		session, record := seedJob(t, db)
		processor := NewProcessor(db, t.TempDir(), fakeExtractor{text: "resume"}, fakeLLM{err: errors.New("timeout")}, nil)
		err := processor.Handle(t.Context(), taskFor(record.ID))
		require.EqualError(t, err, "timeout")
		require.NoError(t, db.First(&record, "id=?", record.ID).Error)
		require.Equal(t, "LLM_ANALYSIS_FAILED", record.ErrorCode)
		require.NoError(t, db.First(&session, "id=?", session.ID).Error)
		require.Equal(t, interview.StatusFailed, session.Status)

		processor.llm = fakeLLM{analysis: llm.ResumeAnalysis{Direction: "Go", ResumeScore: 80, Questions: []llm.QuestionDraft{{Number: "1", Content: "Q1"}}}}
		require.NoError(t, processor.Handle(t.Context(), taskFor(record.ID)))
		require.NoError(t, db.First(&record, "id=?", record.ID).Error)
		require.Equal(t, "COMPLETED", record.Status)
		require.Equal(t, 2, record.Attempts)
		require.NoError(t, db.First(&session, "id=?", session.ID).Error)
		require.Equal(t, interview.StatusReady, session.Status)
	})
}
