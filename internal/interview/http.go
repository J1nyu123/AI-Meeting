package interview

import (
	"strconv"

	"ai-meeting-go/internal/auth"
	"ai-meeting-go/internal/platform/httpx"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

type Handler struct {
	service *Service
	db      *gorm.DB
}

func NewHandler(service *Service, db *gorm.DB) *Handler { return &Handler{service: service, db: db} }
func (h *Handler) Create(c *gin.Context) {
	session, err := h.service.Create(c, auth.UserID(c))
	if err != nil {
		httpx.Fail(c, err)
		return
	}
	httpx.Created(c, session)
}
func (h *Handler) State(c *gin.Context) {
	view, degraded, err := h.service.State(c, auth.UserID(c), c.Param("id"))
	if err != nil {
		httpx.Fail(c, err)
		return
	}
	if degraded {
		c.Header("X-Degraded-Mode", "mysql")
	}
	httpx.OK(c, view)
}
func (h *Handler) Answer(c *gin.Context) {
	var req struct {
		QuestionNumber string `json:"questionNumber" binding:"required"`
		AnswerContent  string `json:"answerContent" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		httpx.Fail(c, httpx.BadRequest("INVALID_REQUEST", "questionNumber and answerContent are required"))
		return
	}
	result, degraded, err := h.service.Answer(c, auth.UserID(c), c.Param("id"), c.GetHeader("Idempotency-Key"), req.QuestionNumber, req.AnswerContent)
	if err != nil {
		httpx.Fail(c, err)
		return
	}
	if degraded {
		c.Header("X-Degraded-Mode", "mysql")
	}
	httpx.OK(c, result)
}
func (h *Handler) Delete(c *gin.Context) {
	degraded, err := h.service.Delete(c, auth.UserID(c), c.Param("id"))
	if err != nil {
		httpx.Fail(c, err)
		return
	}
	if degraded {
		c.Header("X-Degraded-Mode", "mysql")
	}
	httpx.OK(c, gin.H{"sessionId": c.Param("id")})
}
func (h *Handler) List(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	size, _ := strconv.Atoi(c.DefaultQuery("size", "10"))
	if page < 1 {
		page = 1
	}
	if size < 1 || size > 100 {
		size = 10
	}
	var total int64
	var records []Session
	q := h.db.WithContext(c).Where("user_id=?", auth.UserID(c))
	q.Model(&Session{}).Count(&total)
	q.Order("created_at DESC").Offset((page - 1) * size).Limit(size).Find(&records)
	httpx.OK(c, gin.H{"records": records, "total": total, "page": page, "size": size})
}
