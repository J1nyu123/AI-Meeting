package interview

import "time"

type Status string

const (
	StatusCreated    Status = "CREATED"
	StatusAnalyzing  Status = "ANALYZING"
	StatusReady      Status = "READY"
	StatusInProgress Status = "IN_PROGRESS"
	StatusCompleted  Status = "COMPLETED"
	StatusFailed     Status = "FAILED"
)

type Session struct {
	ID                    string     `gorm:"primaryKey;size:36" json:"sessionId"`
	UserID                uint64     `json:"-"`
	Status                Status     `gorm:"size:24" json:"status"`
	Direction             string     `json:"direction"`
	ResumeScore           int        `json:"resumeScore"`
	CurrentQuestionNumber string     `json:"currentQuestionNumber"`
	CurrentMainIndex      int        `json:"currentMainIndex"`
	FollowUpCount         int        `json:"followUpCount"`
	TotalScore            int        `json:"totalScore"`
	TurnSequence          int64      `json:"turnSequence"`
	Version               int64      `json:"version"`
	StartedAt             *time.Time `json:"startedAt"`
	CompletedAt           *time.Time `json:"completedAt"`
	CreatedAt             time.Time  `json:"createdAt"`
	UpdatedAt             time.Time  `json:"updatedAt"`
}

func (Session) TableName() string { return "interview_sessions" }

type Question struct {
	ID           string `gorm:"primaryKey;size:36"`
	SessionID    string
	Number       string
	Content      string
	Suggestion   string
	IsFollowUp   bool
	ParentNumber string
	Ordinal      int
	CreatedAt    time.Time
}

func (Question) TableName() string { return "interview_questions" }

type Turn struct {
	ID              uint64 `gorm:"primaryKey"`
	SessionID       string
	Sequence        int64
	RequestID       string
	QuestionNumber  string
	QuestionContent string
	AnswerContent   string
	Score           int
	Feedback        string
	MissingPoints   []byte `gorm:"type:json"`
	IsFollowUp      bool
	CreatedAt       time.Time
}

func (Turn) TableName() string { return "interview_turns" }

type AnswerAttempt struct {
	ID             uint64 `gorm:"primaryKey"`
	SessionID      string
	UserID         uint64
	IdempotencyKey string
	QuestionNumber string
	Status         string
	LeaseUntil     *time.Time
	ResponseJSON   []byte `gorm:"type:json"`
	ErrorCode      string
	CreatedAt      time.Time
	UpdatedAt      time.Time
}

func (AnswerAttempt) TableName() string { return "answer_attempts" }

type RuntimeState struct {
	SessionID             string    `json:"sessionId"`
	Status                Status    `json:"status"`
	CurrentQuestionNumber string    `json:"currentQuestionNumber"`
	CurrentMainIndex      int       `json:"currentMainIndex"`
	FollowUpCount         int       `json:"followUpCount"`
	TotalScore            int       `json:"totalScore"`
	TurnSequence          int64     `json:"turnSequence"`
	Version               int64     `json:"version"`
	LastAppliedRequestID  string    `json:"lastAppliedRequestId"`
	UpdatedAt             time.Time `json:"updatedAt"`
}

type QuestionView struct {
	Number     string `json:"number"`
	Content    string `json:"content"`
	IsFollowUp bool   `json:"isFollowUp"`
}
type StateView struct {
	SessionID       string        `json:"sessionId"`
	Status          Status        `json:"status"`
	CanResume       bool          `json:"canResume"`
	Version         int64         `json:"version"`
	ResumeScore     int           `json:"resumeScore"`
	InterviewType   string        `json:"interviewType"`
	ResumeFileURL   string        `json:"resumeFileUrl,omitempty"`
	CurrentQuestion *QuestionView `json:"currentQuestion"`
	Progress        Progress      `json:"progress"`
	RecentTurns     []TurnView    `json:"recentTurns"`
}
type Progress struct {
	MainCompleted int   `json:"mainCompleted"`
	MainTotal     int64 `json:"mainTotal"`
	FollowUpCount int   `json:"followUpCount"`
	TotalScore    int   `json:"totalScore"`
}
type TurnView struct {
	Sequence       int64  `json:"sequence"`
	QuestionNumber string `json:"questionNumber"`
	Question       string `json:"question"`
	Answer         string `json:"answer"`
	Score          int    `json:"score"`
	Feedback       string `json:"feedback"`
	IsFollowUp     bool   `json:"isFollowUp"`
}
type AnswerResponse struct {
	QuestionNumber string        `json:"questionNumber"`
	Score          int           `json:"score"`
	TotalScore     int           `json:"totalScore"`
	Feedback       string        `json:"feedback"`
	MissingPoints  []string      `json:"missingPoints"`
	NextQuestion   *QuestionView `json:"nextQuestion"`
	FollowUpNeeded bool          `json:"followUpNeeded"`
	FollowUpCount  int           `json:"followUpCount"`
	Finished       bool          `json:"finished"`
	Version        int64         `json:"version"`
}
