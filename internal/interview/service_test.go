package interview

import (
	"context"
	"errors"
	"os"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"ai-meeting-go/internal/evaluation"
	"ai-meeting-go/internal/llm"
	"ai-meeting-go/internal/resume"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

type fixedLLM struct{ result evaluation.Result }

type stubRuntimeStateStore struct {
	cached RuntimeState
	set    RuntimeState
}

func (s *stubRuntimeStateStore) Get(context.Context, string) (RuntimeState, error) {
	return s.cached, nil
}

func (s *stubRuntimeStateStore) Set(_ context.Context, state RuntimeState) error {
	s.set = state
	return nil
}

func (f fixedLLM) AnalyzeResume(context.Context, llm.ResumeInput) (llm.ResumeAnalysis, error) {
	return llm.ResumeAnalysis{}, nil
}
func (f fixedLLM) EvaluateAnswer(context.Context, llm.AnswerInput) (evaluation.Result, error) {
	return f.result, nil
}
func (f fixedLLM) GenerateFollowUp(context.Context, llm.FollowUpInput) (llm.FollowUpResult, error) {
	return llm.FollowUpResult{Question: f.result.FollowUpQuestion}, nil
}

type failingLLM struct{}

func (failingLLM) AnalyzeResume(context.Context, llm.ResumeInput) (llm.ResumeAnalysis, error) {
	return llm.ResumeAnalysis{}, errors.New("not used")
}
func (failingLLM) EvaluateAnswer(context.Context, llm.AnswerInput) (evaluation.Result, error) {
	return evaluation.Result{}, errors.New("llm timeout")
}
func (failingLLM) GenerateFollowUp(context.Context, llm.FollowUpInput) (llm.FollowUpResult, error) {
	return llm.FollowUpResult{}, errors.New("llm timeout")
}

type trackingLLM struct {
	evaluationResult evaluation.Result
	followUpResult   llm.FollowUpResult
	followUpErr      error
	answerInput      llm.AnswerInput
	followUpInput    llm.FollowUpInput
	evaluationCalls  int
	followUpCalls    int
}

func (*trackingLLM) AnalyzeResume(context.Context, llm.ResumeInput) (llm.ResumeAnalysis, error) {
	return llm.ResumeAnalysis{}, nil
}
func (f *trackingLLM) EvaluateAnswer(_ context.Context, input llm.AnswerInput) (evaluation.Result, error) {
	f.answerInput = input
	f.evaluationCalls++
	return f.evaluationResult, nil
}
func (f *trackingLLM) GenerateFollowUp(_ context.Context, input llm.FollowUpInput) (llm.FollowUpResult, error) {
	f.followUpInput = input
	f.followUpCalls++
	return f.followUpResult, f.followUpErr
}

func interviewDB(t *testing.T) *gorm.DB {
	db, err := gorm.Open(sqlite.Open("file:"+uuid.NewString()+"?mode=memory&cache=shared"), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, db.AutoMigrate(&Session{}, &Question{}, &Turn{}, &AnswerAttempt{}, &resume.Asset{}))
	require.NoError(t, db.Exec("CREATE TABLE analysis_jobs (session_id TEXT NOT NULL)").Error)
	require.NoError(t, db.Exec("CREATE TABLE interview_reports (session_id TEXT NOT NULL)").Error)
	return db
}

func TestDeleteCompletedInterviewRemovesRelatedDataAndFile(t *testing.T) {
	db := interviewDB(t)
	storage := resume.LocalStorage{Root: t.TempDir()}
	storageKey, _, _, err := storage.Save(strings.NewReader("%PDF-1.4 resume"), "resume.pdf")
	require.NoError(t, err)
	session := Session{ID: uuid.NewString(), UserID: 7, Status: StatusCompleted}
	require.NoError(t, db.Create(&session).Error)
	require.NoError(t, db.Create(&Question{ID: uuid.NewString(), SessionID: session.ID, Number: "1"}).Error)
	require.NoError(t, db.Create(&Turn{SessionID: session.ID, Sequence: 1, RequestID: "delete-request", QuestionNumber: "1"}).Error)
	require.NoError(t, db.Create(&AnswerAttempt{SessionID: session.ID, UserID: 7, IdempotencyKey: "delete-request"}).Error)
	require.NoError(t, db.Create(&resume.Asset{ID: uuid.NewString(), SessionID: session.ID, StorageKey: storageKey}).Error)
	require.NoError(t, db.Exec("INSERT INTO analysis_jobs (session_id) VALUES (?)", session.ID).Error)
	require.NoError(t, db.Exec("INSERT INTO interview_reports (session_id) VALUES (?)", session.ID).Error)

	service := NewService(db, NewRuntimeStore(nil), fixedLLM{}).WithFileStorage(storage)
	degraded, err := service.Delete(t.Context(), 7, session.ID)
	require.NoError(t, err)
	require.True(t, degraded)

	for _, table := range []string{"interview_reports", "interview_turns", "answer_attempts", "interview_questions", "analysis_jobs", "resume_assets"} {
		var count int64
		require.NoError(t, db.Table(table).Where("session_id = ?", session.ID).Count(&count).Error)
		require.Zero(t, count, table)
	}
	var sessionCount int64
	require.NoError(t, db.Table("interview_sessions").Where("id = ?", session.ID).Count(&sessionCount).Error)
	require.Zero(t, sessionCount)
	_, err = os.Stat(storage.Root + string(os.PathSeparator) + storageKey)
	require.ErrorIs(t, err, os.ErrNotExist)
}

func TestDeleteFailedInterviewSucceeds(t *testing.T) {
	db := interviewDB(t)
	session := Session{ID: uuid.NewString(), UserID: 7, Status: StatusFailed}
	require.NoError(t, db.Create(&session).Error)

	_, err := NewService(db, NewRuntimeStore(nil), fixedLLM{}).Delete(t.Context(), 7, session.ID)
	require.NoError(t, err)
	var count int64
	require.NoError(t, db.Model(&Session{}).Where("id = ?", session.ID).Count(&count).Error)
	require.Zero(t, count)
}

func TestDeleteInterviewRejectsNonTerminalAndUnknownSessions(t *testing.T) {
	for _, status := range []Status{StatusCreated, StatusAnalyzing, StatusReady, StatusInProgress} {
		t.Run(string(status), func(t *testing.T) {
			db := interviewDB(t)
			session := Session{ID: uuid.NewString(), UserID: 7, Status: status}
			require.NoError(t, db.Create(&session).Error)
			_, err := NewService(db, NewRuntimeStore(nil), fixedLLM{}).Delete(t.Context(), 7, session.ID)
			require.ErrorContains(t, err, "only completed or failed")
			require.NoError(t, db.First(&Session{}, "id = ?", session.ID).Error)
		})
	}

	db := interviewDB(t)
	session := Session{ID: uuid.NewString(), UserID: 8, Status: StatusCompleted}
	require.NoError(t, db.Create(&session).Error)
	service := NewService(db, NewRuntimeStore(nil), fixedLLM{})
	_, err := service.Delete(t.Context(), 7, session.ID)
	require.ErrorContains(t, err, "interview not found")
	_, err = service.Delete(t.Context(), 7, uuid.NewString())
	require.ErrorContains(t, err, "interview not found")
}

func TestAnswerUsesResumeContextAndIndependentFollowUpAgent(t *testing.T) {
	db := interviewDB(t)
	session := Session{ID: uuid.NewString(), UserID: 7, Status: StatusReady, CurrentQuestionNumber: "1"}
	require.NoError(t, db.Create(&session).Error)
	require.NoError(t, db.Create(&Question{ID: uuid.NewString(), SessionID: session.ID, Number: "1", Content: "如何设计恢复？", Ordinal: 1}).Error)
	require.NoError(t, db.Create(&resume.Asset{ID: uuid.NewString(), SessionID: session.ID, StorageKey: "resume.pdf", ExtractedText: "候选人的 Redis 项目经历"}).Error)
	client := &trackingLLM{
		evaluationResult: evaluation.Result{Score: 55, Feedback: "需要补充", MissingPoints: []string{"恢复过程"}, FollowUpNeeded: true, FollowUpQuestion: "评分官建议追问"},
		followUpResult:   llm.FollowUpResult{Question: "提问官生成的追问？"},
	}
	service := NewService(db, NewRuntimeStore(nil), client)

	first, _, err := service.Answer(t.Context(), 7, session.ID, "request-agent", "1", "我的回答")
	require.NoError(t, err)
	require.NotNil(t, first.NextQuestion)
	require.Equal(t, "提问官生成的追问？", first.NextQuestion.Content)
	require.Equal(t, session.ID, client.answerInput.SessionID)
	require.Equal(t, "候选人的 Redis 项目经历", client.answerInput.ResumeContext)
	require.Equal(t, "如何设计恢复？", client.followUpInput.Question)
	require.Equal(t, "候选人的 Redis 项目经历", client.followUpInput.ResumeContext)
	require.Equal(t, 0, client.followUpInput.Current)
	require.Equal(t, 2, client.followUpInput.Maximum)

	replayed, _, err := service.Answer(t.Context(), 7, session.ID, "request-agent", "1", "我的回答")
	require.NoError(t, err)
	require.Equal(t, first, replayed)
	require.Equal(t, 1, client.evaluationCalls)
	require.Equal(t, 1, client.followUpCalls)
}

func TestAnswerFallsBackToScorerQuestionWhenAskingFails(t *testing.T) {
	db := interviewDB(t)
	session := Session{ID: uuid.NewString(), UserID: 7, Status: StatusReady, CurrentQuestionNumber: "1"}
	require.NoError(t, db.Create(&session).Error)
	require.NoError(t, db.Create(&Question{ID: uuid.NewString(), SessionID: session.ID, Number: "1", Content: "问题", Ordinal: 1}).Error)
	client := &trackingLLM{
		evaluationResult: evaluation.Result{Score: 55, Feedback: "需要补充", FollowUpNeeded: true, FollowUpQuestion: "评分官的降级追问？"},
		followUpErr:      errors.New("asking unavailable"),
	}
	service := NewService(db, NewRuntimeStore(nil), client)

	response, _, err := service.Answer(t.Context(), 7, session.ID, "request-fallback", "1", "回答")
	require.NoError(t, err)
	require.NotNil(t, response.NextQuestion)
	require.Equal(t, "评分官的降级追问？", response.NextQuestion.Content)
}

func TestAnswerIsIdempotentAndDegradesWithoutRedis(t *testing.T) {
	db := interviewDB(t)
	session := Session{ID: uuid.NewString(), UserID: 7, Status: StatusReady, CurrentQuestionNumber: "1"}
	require.NoError(t, db.Create(&session).Error)
	require.NoError(t, db.Create(&[]Question{
		{ID: uuid.NewString(), SessionID: session.ID, Number: "1", Content: "first", Ordinal: 1},
		{ID: uuid.NewString(), SessionID: session.ID, Number: "2", Content: "second", Ordinal: 2},
	}).Error)
	service := NewService(db, NewRuntimeStore(nil), fixedLLM{evaluation.Result{Score: 80, Feedback: "good"}})

	first, degraded, err := service.Answer(t.Context(), 7, session.ID, "request-1", "1", "a detailed answer")
	require.NoError(t, err)
	require.True(t, degraded)
	require.Equal(t, "2", first.NextQuestion.Number)
	require.Equal(t, 80, first.TotalScore)
	var storedAttempt AnswerAttempt
	require.NoError(t, db.Where("session_id = ? AND idempotency_key = ?", session.ID, "request-1").First(&storedAttempt).Error)
	require.Equal(t, "SUCCEEDED", storedAttempt.Status)

	replayed, _, err := service.Answer(t.Context(), 7, session.ID, "request-1", "1", "a detailed answer")
	require.NoError(t, err)
	require.Equal(t, first, replayed)
	var turns int64
	require.NoError(t, db.Model(&Turn{}).Where("session_id = ?", session.ID).Count(&turns).Error)
	require.EqualValues(t, 1, turns)
}

func TestStateRehydratesFromMySQLWithoutRedis(t *testing.T) {
	db := interviewDB(t)
	session := Session{ID: uuid.NewString(), UserID: 9, Status: StatusInProgress, CurrentQuestionNumber: "3-F1", CurrentMainIndex: 3, FollowUpCount: 1, TotalScore: 165, TurnSequence: 12, Version: 9}
	require.NoError(t, db.Create(&session).Error)
	require.NoError(t, db.Create(&Question{ID: uuid.NewString(), SessionID: session.ID, Number: "3-F1", Content: "question", Ordinal: 3, IsFollowUp: true}).Error)
	for sequence := 1; sequence <= 12; sequence++ {
		require.NoError(t, db.Create(&Turn{SessionID: session.ID, Sequence: int64(sequence), QuestionNumber: "1", QuestionContent: "question", AnswerContent: "answer", Score: 55, Feedback: "feedback"}).Error)
	}
	service := NewService(db, NewRuntimeStore(nil), fixedLLM{})
	state, degraded, err := service.State(t.Context(), 9, session.ID)
	require.NoError(t, err)
	require.True(t, degraded)
	require.Equal(t, int64(9), state.Version)
	require.Equal(t, "3-F1", state.CurrentQuestion.Number)
	require.Equal(t, 1, state.Progress.FollowUpCount)
	require.Equal(t, 165, state.Progress.TotalScore)
	require.Len(t, state.RecentTurns, 12)
	require.Equal(t, int64(1), state.RecentTurns[0].Sequence)
	require.Equal(t, int64(12), state.RecentTurns[11].Sequence)
}

func TestStateRejectsCachedRuntimeWithStaleVersion(t *testing.T) {
	session := Session{ID: uuid.NewString(), Status: StatusInProgress, CurrentQuestionNumber: "3-F1", CurrentMainIndex: 3, FollowUpCount: 1, TotalScore: 165, TurnSequence: 7, Version: 9}
	cached := RuntimeState{SessionID: session.ID, Status: StatusInProgress, CurrentQuestionNumber: "2", CurrentMainIndex: 2, FollowUpCount: 0, TotalScore: 110, TurnSequence: 3, Version: 5}
	store := &stubRuntimeStateStore{cached: cached}

	state, rebuilt, err := Rehydrate(t.Context(), store, session)

	require.NoError(t, err)
	require.True(t, rebuilt)
	require.Equal(t, int64(9), state.Version)
	require.Equal(t, "3-F1", state.CurrentQuestionNumber)
	require.Equal(t, 1, state.FollowUpCount)
	require.Equal(t, int64(7), state.TurnSequence)
	require.Equal(t, state, store.set)
}

func TestConcurrentAnswersOnlyAdvanceOnceInMySQLFallback(t *testing.T) {
	db := interviewDB(t)
	sqlDB, err := db.DB()
	require.NoError(t, err)
	sqlDB.SetMaxOpenConns(1)
	session := Session{ID: uuid.NewString(), UserID: 11, Status: StatusReady, CurrentQuestionNumber: "1"}
	require.NoError(t, db.Create(&session).Error)
	require.NoError(t, db.Create(&[]Question{
		{ID: uuid.NewString(), SessionID: session.ID, Number: "1", Content: "first", Ordinal: 1},
		{ID: uuid.NewString(), SessionID: session.ID, Number: "2", Content: "second", Ordinal: 2},
	}).Error)
	service := NewService(db, NewRuntimeStore(nil), fixedLLM{evaluation.Result{Score: 70, Feedback: "ok"}})
	var successes atomic.Int32
	var group sync.WaitGroup
	for index := 0; index < 20; index++ {
		group.Add(1)
		go func(index int) {
			defer group.Done()
			_, _, answerErr := service.Answer(context.Background(), 11, session.ID, "concurrent-"+string(rune('a'+index)), "1", "answer")
			if answerErr == nil {
				successes.Add(1)
			}
		}(index)
	}
	group.Wait()
	require.EqualValues(t, 1, successes.Load())
	var turns int64
	require.NoError(t, db.Model(&Turn{}).Where("session_id = ?", session.ID).Count(&turns).Error)
	require.EqualValues(t, 1, turns)
}

func TestCreateStateAndAnswerValidation(t *testing.T) {
	db := interviewDB(t)
	service := NewService(db, NewRuntimeStore(nil), fixedLLM{})
	session, err := service.Create(t.Context(), 42)
	require.NoError(t, err)
	require.Equal(t, StatusCreated, session.Status)

	_, _, err = service.State(t.Context(), 99, session.ID)
	require.Error(t, err)
	_, _, err = service.Answer(t.Context(), 42, session.ID, "", "1", "answer")
	require.Error(t, err)
	_, _, err = service.Answer(t.Context(), 42, session.ID, "key", "1", "")
	require.Error(t, err)
	_, _, err = service.Answer(t.Context(), 42, session.ID, "key", "1", strings.Repeat("字", 5001))
	require.Error(t, err)

	_, _, err = service.Answer(t.Context(), 42, session.ID, "inactive", "1", "answer")
	require.Error(t, err)
	var attempt AnswerAttempt
	require.NoError(t, db.Where("idempotency_key=?", "inactive").First(&attempt).Error)
	require.Equal(t, "FAILED", attempt.Status)
}

func TestAnswerRejectsStaleMissingQuestionAndLLMFailure(t *testing.T) {
	tests := []struct {
		name      string
		question  string
		seed      bool
		client    llm.Client
		errorCode string
	}{
		{"stale", "2", true, fixedLLM{}, "REQUEST_ABORTED"},
		{"missing question", "1", false, fixedLLM{}, "REQUEST_ABORTED"},
		{"llm failure", "1", true, failingLLM{}, "AI_EVALUATION_FAILED"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			db := interviewDB(t)
			session := Session{ID: uuid.NewString(), UserID: 7, Status: StatusReady, CurrentQuestionNumber: "1"}
			require.NoError(t, db.Create(&session).Error)
			if tt.seed {
				require.NoError(t, db.Create(&Question{ID: uuid.NewString(), SessionID: session.ID, Number: "1", Content: "question", Ordinal: 1}).Error)
			}
			service := NewService(db, NewRuntimeStore(nil), tt.client)
			_, _, err := service.Answer(t.Context(), 7, session.ID, "request", tt.question, "answer")
			require.Error(t, err)
			var attempt AnswerAttempt
			require.NoError(t, db.Where("idempotency_key=?", "request").First(&attempt).Error)
			require.Equal(t, tt.errorCode, attempt.ErrorCode)
		})
	}
}

func TestFollowUpsAreCappedAndDoNotIncreaseMainScore(t *testing.T) {
	db := interviewDB(t)
	session := Session{ID: uuid.NewString(), UserID: 7, Status: StatusReady, CurrentQuestionNumber: "1"}
	require.NoError(t, db.Create(&session).Error)
	require.NoError(t, db.Create(&Question{ID: uuid.NewString(), SessionID: session.ID, Number: "1", Content: "main", Ordinal: 1}).Error)
	service := NewService(db, NewRuntimeStore(nil), fixedLLM{evaluation.Result{Score: 50, Feedback: "more", FollowUpNeeded: true, FollowUpQuestion: "details?"}})

	first, _, err := service.Answer(t.Context(), 7, session.ID, "one", "1", "answer")
	require.NoError(t, err)
	require.Equal(t, "1-F1", first.NextQuestion.Number)
	require.Equal(t, 50, first.TotalScore)
	second, _, err := service.Answer(t.Context(), 7, session.ID, "two", "1-F1", "answer")
	require.NoError(t, err)
	require.Equal(t, "1-F2", second.NextQuestion.Number)
	require.Equal(t, 50, second.TotalScore)
	third, _, err := service.Answer(t.Context(), 7, session.ID, "three", "1-F2", "answer")
	require.NoError(t, err)
	require.True(t, third.Finished)
	require.Nil(t, third.NextQuestion)
	require.Equal(t, 50, third.TotalScore)

	require.NoError(t, db.First(&session, "id=?", session.ID).Error)
	require.Equal(t, StatusCompleted, session.Status)
	require.Equal(t, 2, session.FollowUpCount)
	var turns []Turn
	require.NoError(t, db.Where("session_id=?", session.ID).Order("sequence").Find(&turns).Error)
	require.Len(t, turns, 3)
	require.False(t, turns[0].IsFollowUp)
	require.True(t, turns[1].IsFollowUp)
}

func TestAnswerLeaseCanBeRejectedOrTakenOver(t *testing.T) {
	for _, tc := range []struct {
		name       string
		lease      time.Time
		wantError  bool
		wantStatus string
	}{
		{"active lease", time.Now().Add(time.Minute), true, "PROCESSING"},
		{"expired lease", time.Now().Add(-time.Minute), false, "SUCCEEDED"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			db := interviewDB(t)
			session := Session{ID: uuid.NewString(), UserID: 7, Status: StatusReady, CurrentQuestionNumber: "1"}
			require.NoError(t, db.Create(&session).Error)
			require.NoError(t, db.Create(&Question{ID: uuid.NewString(), SessionID: session.ID, Number: "1", Content: "question", Ordinal: 1}).Error)
			attempt := AnswerAttempt{SessionID: session.ID, UserID: 7, IdempotencyKey: "same", QuestionNumber: "1", Status: "PROCESSING", LeaseUntil: &tc.lease}
			require.NoError(t, db.Create(&attempt).Error)
			service := NewService(db, NewRuntimeStore(nil), fixedLLM{evaluation.Result{Score: 90, Feedback: "good"}})
			_, _, err := service.Answer(t.Context(), 7, session.ID, "same", "1", "answer")
			if tc.wantError {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
			}
			require.NoError(t, db.First(&attempt, attempt.ID).Error)
			require.Equal(t, tc.wantStatus, attempt.Status)
		})
	}
}
