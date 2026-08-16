package report

import (
	"testing"

	"ai-meeting-go/internal/interview"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func TestGenerateReportIncludesOnlyMainQuestionInAverage(t *testing.T) {
	db, err := gorm.Open(sqlite.Open("file:"+uuid.NewString()+"?mode=memory&cache=shared"), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, db.AutoMigrate(&interview.Session{}, &interview.Turn{}, &Report{}))
	session := interview.Session{ID: uuid.NewString(), UserID: 3, Status: interview.StatusCompleted, ResumeScore: 80}
	require.NoError(t, db.Create(&session).Error)
	require.NoError(t, db.Create(&[]interview.Turn{
		{SessionID: session.ID, Sequence: 1, RequestID: "a", Score: 60, IsFollowUp: false},
		{SessionID: session.ID, Sequence: 2, RequestID: "b", Score: 10, IsFollowUp: true},
		{SessionID: session.ID, Sequence: 3, RequestID: "c", Score: 80, IsFollowUp: false},
	}).Error)
	report, err := NewService(db).Generate(t.Context(), 3, session.ID)
	require.NoError(t, err)
	require.Equal(t, 70, report.InterviewScore)
	require.Equal(t, 72, report.CompositeScore)
	require.Len(t, report.Turns, 3)

	_, err = NewService(db).Get(t.Context(), 999, session.ID)
	require.Error(t, err)
	_, err = NewService(db).Generate(t.Context(), 999, session.ID)
	require.Error(t, err)
}
