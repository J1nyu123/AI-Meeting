package auth

import (
	"net/http"
	"strings"

	"ai-meeting-go/internal/platform/httpx"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

type Handler struct {
	service *Service
	db      *gorm.DB
}

func NewHandler(service *Service, db *gorm.DB) *Handler { return &Handler{service: service, db: db} }

type credentials struct {
	Username string `json:"username" binding:"required"`
	Password string `json:"password" binding:"required"`
}
type refreshRequest struct {
	RefreshToken string `json:"refreshToken" binding:"required"`
}

func (h *Handler) Register(c *gin.Context) {
	var req credentials
	if err := c.ShouldBindJSON(&req); err != nil {
		httpx.Fail(c, httpx.BadRequest("INVALID_REQUEST", "username and password are required"))
		return
	}
	user, err := h.service.Register(c, req.Username, req.Password)
	if err != nil {
		httpx.Fail(c, err)
		return
	}
	httpx.Created(c, user)
}
func (h *Handler) Login(c *gin.Context) {
	var req credentials
	if err := c.ShouldBindJSON(&req); err != nil {
		httpx.Fail(c, httpx.BadRequest("INVALID_REQUEST", "username and password are required"))
		return
	}
	pair, err := h.service.Login(c, req.Username, req.Password)
	if err != nil {
		httpx.Fail(c, err)
		return
	}
	httpx.OK(c, pair)
}
func (h *Handler) Refresh(c *gin.Context) {
	var req refreshRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		httpx.Fail(c, httpx.BadRequest("INVALID_REQUEST", "refreshToken is required"))
		return
	}
	pair, err := h.service.Refresh(c, req.RefreshToken)
	if err != nil {
		httpx.Fail(c, err)
		return
	}
	httpx.OK(c, pair)
}
func (h *Handler) Logout(c *gin.Context) {
	var req refreshRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		httpx.Fail(c, httpx.BadRequest("INVALID_REQUEST", "refreshToken is required"))
		return
	}
	if err := h.service.Logout(c, req.RefreshToken); err != nil {
		httpx.Fail(c, err)
		return
	}
	httpx.OK(c, gin.H{"loggedOut": true})
}
func (h *Handler) Me(c *gin.Context) {
	id := UserID(c)
	var user User
	if err := h.db.WithContext(c).First(&user, id).Error; err != nil {
		httpx.Fail(c, httpx.NotFound("USER_NOT_FOUND", "user not found"))
		return
	}
	httpx.OK(c, user)
}

func (h *Handler) Middleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		raw := strings.TrimSpace(strings.TrimPrefix(c.GetHeader("Authorization"), "Bearer "))
		if raw == "" {
			httpx.Fail(c, httpx.Unauthorized("access token required"))
			return
		}
		id, err := h.service.ValidateAccess(raw)
		if err != nil {
			httpx.Fail(c, httpx.Unauthorized("invalid or expired access token"))
			return
		}
		c.Set("user_id", id)
		c.Next()
	}
}
func UserID(c *gin.Context) uint64 { value, _ := c.Get("user_id"); id, _ := value.(uint64); return id }

func RegisterRoutes(r *gin.RouterGroup, h *Handler) {
	r.POST("/register", h.Register)
	r.POST("/login", h.Login)
	r.POST("/refresh", h.Refresh)
	r.POST("/logout", h.Logout)
}

var _ = http.StatusOK
