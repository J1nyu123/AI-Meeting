package report

import (
	"context"
	"encoding/json"
	"time"

	"ai-meeting-go/internal/auth"
	"ai-meeting-go/internal/interview"
	"ai-meeting-go/internal/platform/httpx"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type Report struct {
	ID              string               `gorm:"primaryKey;size:36" json:"id"`
	SessionID       string               `gorm:"size:36;uniqueIndex" json:"sessionId"`
	UserID          uint64               `json:"-"`
	InterviewScore  int                  `json:"interviewScore"`
	ResumeScore     int                  `json:"resumeScore"`
	CompositeScore  int                  `json:"compositeScore"`
	OverallComment  string               `json:"overallComment"`
	Highlights      json.RawMessage      `gorm:"type:json" json:"highlights"`
	ImprovementTips json.RawMessage      `gorm:"type:json" json:"improvementTips"`
	NextActions     json.RawMessage      `gorm:"type:json" json:"nextActions"`
	RadarMetrics    json.RawMessage      `gorm:"type:json" json:"radarMetrics"`
	CreatedAt       time.Time            `json:"createdAt"`
	UpdatedAt       time.Time            `json:"updatedAt"`
	Turns           []interview.TurnView `gorm:"-" json:"turns"`
}

func (Report) TableName() string { return "interview_reports" }

type Service struct{ db *gorm.DB }

func NewService(db *gorm.DB) *Service { return &Service{db: db} }
func (s *Service) Generate(ctx context.Context, userID uint64, sessionID string) (Report, error) {
	var session interview.Session
	if err := s.db.WithContext(ctx).Where("id=? AND user_id=?", sessionID, userID).First(&session).Error; err != nil {
		return Report{}, httpx.NotFound("INTERVIEW_NOT_FOUND", "interview not found")
	}
	var turns []interview.Turn
	s.db.WithContext(ctx).Where("session_id=?", sessionID).Order("sequence").Find(&turns)
	mainCount, total := 0, 0
	for _, t := range turns {
		if !t.IsFollowUp {
			mainCount++
			total += t.Score
		}
	}
	score := 0
	if mainCount > 0 {
		score = total / mainCount
	}
	composite := (score*8 + session.ResumeScore*2) / 10
	highlights, _ := json.Marshal([]string{"完成了结构化模拟面试", "能够围绕项目经历回答问题"})
	tips, _ := json.Marshal([]string{"回答时补充量化结果", "主动覆盖异常和降级场景"})
	actions, _ := json.Marshal([]string{"复盘低分题目", "使用 STAR 法重新组织答案"})
	radar, _ := json.Marshal([]map[string]any{{"label": "专业能力", "value": score}, {"label": "项目表达", "value": min(100, score+5)}, {"label": "问题分析", "value": max(0, score-5)}, {"label": "简历质量", "value": session.ResumeScore}})
	record := Report{ID: uuid.NewString(), SessionID: sessionID, UserID: userID, InterviewScore: score, ResumeScore: session.ResumeScore, CompositeScore: composite, OverallComment: "已完成本次模拟面试，请结合逐题反馈继续完善项目表达。", Highlights: highlights, ImprovementTips: tips, NextActions: actions, RadarMetrics: radar}
	if err := s.db.WithContext(ctx).Clauses(clause.OnConflict{Columns: []clause.Column{{Name: "session_id"}}, DoUpdates: clause.AssignmentColumns([]string{"interview_score", "resume_score", "composite_score", "overall_comment", "highlights", "improvement_tips", "next_actions", "radar_metrics", "updated_at"})}).Create(&record).Error; err != nil {
		return Report{}, err
	}
	return s.Get(ctx, userID, sessionID)
}
func (s *Service) Get(ctx context.Context, userID uint64, sessionID string) (Report, error) {
	var r Report
	if err := s.db.WithContext(ctx).Where("session_id=? AND user_id=?", sessionID, userID).First(&r).Error; err != nil {
		return Report{}, httpx.NotFound("REPORT_NOT_FOUND", "report not found")
	}
	var turns []interview.Turn
	s.db.WithContext(ctx).Where("session_id=?", sessionID).Order("sequence").Find(&turns)
	for _, t := range turns {
		r.Turns = append(r.Turns, interview.TurnView{Sequence: t.Sequence, QuestionNumber: t.QuestionNumber, Question: t.QuestionContent, Answer: t.AnswerContent, Score: t.Score, Feedback: t.Feedback, IsFollowUp: t.IsFollowUp})
	}
	return r, nil
}
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}

type Handler struct {
	service *Service
	db      *gorm.DB
}

func NewHandler(service *Service, db *gorm.DB) *Handler { return &Handler{service: service, db: db} }
func (h *Handler) Finish(c *gin.Context) {
	uid := auth.UserID(c)
	now := time.Now()
	var session interview.Session
	if err := h.db.WithContext(c).Where("id = ? AND user_id = ?", c.Param("id"), uid).First(&session).Error; err != nil {
		httpx.Fail(c, httpx.NotFound("INTERVIEW_NOT_FOUND", "interview not found"))
		return
	}
	if session.Status == interview.StatusCompleted {
		record, err := h.service.Generate(c, uid, c.Param("id"))
		if err != nil {
			httpx.Fail(c, err)
			return
		}
		httpx.OK(c, record)
		return
	}
	result := h.db.WithContext(c).Model(&interview.Session{}).Where("id=? AND user_id=? AND status IN ?", c.Param("id"), uid, []interview.Status{interview.StatusReady, interview.StatusInProgress}).Updates(map[string]any{"status": interview.StatusCompleted, "completed_at": now, "version": gorm.Expr("version + 1")})
	if result.Error != nil {
		httpx.Fail(c, result.Error)
		return
	}
	if result.RowsAffected == 0 {
		httpx.Fail(c, httpx.Conflict("INTERVIEW_NOT_ACTIVE", "interview is not active"))
		return
	}
	record, err := h.service.Generate(c, uid, c.Param("id"))
	if err != nil {
		httpx.Fail(c, err)
		return
	}
	httpx.OK(c, record)
}
func (h *Handler) Get(c *gin.Context) {
	record, err := h.service.Get(c, auth.UserID(c), c.Param("id"))
	if err != nil {
		httpx.Fail(c, err)
		return
	}
	httpx.OK(c, record)
}
