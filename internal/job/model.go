package job

import "time"

type AnalysisJob struct {
	ID            string    `gorm:"primaryKey;size:36" json:"jobId"`
	SessionID     string    `json:"sessionId"`
	UserID        uint64    `json:"-"`
	Status        string    `json:"status"`
	Progress      int       `json:"progress"`
	Stage         string    `json:"stage"`
	PromptVersion string    `json:"promptVersion"`
	Attempts      int       `json:"attempts"`
	ErrorCode     string    `json:"errorCode,omitempty"`
	ErrorMessage  string    `json:"errorMessage,omitempty"`
	CreatedAt     time.Time `json:"createdAt"`
	UpdatedAt     time.Time `json:"updatedAt"`
}

func (AnalysisJob) TableName() string { return "analysis_jobs" }
