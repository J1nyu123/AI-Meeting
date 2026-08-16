package media

import (
	"io"
	"net/http"
	"strconv"
	"time"

	"ai-meeting-go/internal/auth"
	"ai-meeting-go/internal/platform/httpx"
	"github.com/gin-gonic/gin"
)

type TTSHandler struct {
	Service *TTSService
	Proxy   AudioProxy
}

func (h *TTSHandler) Create(c *gin.Context) {
	req, ok := bindTTSRequest(c)
	if !ok {
		return
	}
	task, err := h.Service.Create(c, auth.UserID(c), c.GetHeader("Idempotency-Key"), req)
	if err != nil {
		httpx.Fail(c, err)
		return
	}
	httpx.Accepted(c, task.Public())
}
func (h *TTSHandler) Get(c *gin.Context) {
	task, err := h.Service.Get(c, auth.UserID(c), c.Param("taskId"), true)
	if err != nil {
		httpx.Fail(c, err)
		return
	}
	httpx.OK(c, task.Public())
}
func (h *TTSHandler) Synthesize(c *gin.Context) {
	req, ok := bindTTSRequest(c)
	if !ok {
		return
	}
	task, err := h.Service.Create(c, auth.UserID(c), c.GetHeader("Idempotency-Key"), req)
	if err != nil {
		httpx.Fail(c, err)
		return
	}
	timeout := time.Duration(*defaults(req).TimeoutSeconds) * time.Second
	poll := time.Duration(*defaults(req).PollIntervalMS) * time.Millisecond
	task, err = h.Service.Wait(c, auth.UserID(c), task.ID, timeout, poll)
	if err != nil {
		httpx.Fail(c, err)
		return
	}
	httpx.OK(c, task.Public())
}
func (h *TTSHandler) Audio(c *gin.Context) {
	rawURL, _, err := h.Service.AudioURL(c, auth.UserID(c), c.Param("taskId"))
	if err != nil {
		httpx.Fail(c, err)
		return
	}
	response, err := h.Proxy.Open(c, rawURL, c.GetHeader("Range"))
	if err != nil {
		httpx.Fail(c, &httpx.AppError{Status: http.StatusBadGateway, Code: "TTS_AUDIO_PROXY_FAILED", Message: "unable to load TTS audio", Cause: err})
		return
	}
	defer response.Body.Close()
	for _, header := range []string{"Content-Type", "Content-Length", "Content-Range", "Accept-Ranges", "ETag", "Last-Modified"} {
		if value := response.Header.Get(header); value != "" {
			c.Header(header, value)
		}
	}
	status := response.StatusCode
	if status != http.StatusOK && status != http.StatusPartialContent {
		httpx.Fail(c, &httpx.AppError{Status: http.StatusBadGateway, Code: "TTS_AUDIO_UPSTREAM_FAILED", Message: "TTS audio provider returned status " + strconv.Itoa(status)})
		return
	}
	c.Status(status)
	_, _ = io.Copy(c.Writer, response.Body)
}
func bindTTSRequest(c *gin.Context) (TTSRequest, bool) {
	c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, 1<<20)
	var req TTSRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		httpx.Fail(c, httpx.BadRequest("INVALID_TTS_REQUEST", "invalid TTS request"))
		return TTSRequest{}, false
	}
	normalized := defaults(req)
	if err := validateTTSRequest(normalized); err != nil {
		httpx.Fail(c, err)
		return TTSRequest{}, false
	}
	return normalized, true
}
