package httpx

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"
)

type Envelope struct {
	Success   bool    `json:"success"`
	Code      string  `json:"code"`
	Message   *string `json:"message"`
	Data      any     `json:"data"`
	RequestID string  `json:"requestId"`
}

type AppError struct {
	Status        int
	Code, Message string
	Cause         error
}

func (e *AppError) Error() string { return e.Message }
func (e *AppError) Unwrap() error { return e.Cause }
func BadRequest(code, message string) *AppError {
	return &AppError{Status: http.StatusBadRequest, Code: code, Message: message}
}
func Unauthorized(message string) *AppError {
	return &AppError{Status: http.StatusUnauthorized, Code: "UNAUTHORIZED", Message: message}
}
func Forbidden(message string) *AppError {
	return &AppError{Status: http.StatusForbidden, Code: "FORBIDDEN", Message: message}
}
func NotFound(code, message string) *AppError {
	return &AppError{Status: http.StatusNotFound, Code: code, Message: message}
}
func Conflict(code, message string) *AppError {
	return &AppError{Status: http.StatusConflict, Code: code, Message: message}
}

func OK(c *gin.Context, data any) {
	c.JSON(http.StatusOK, Envelope{Success: true, Code: "OK", Data: data, RequestID: RequestID(c)})
}
func Created(c *gin.Context, data any) {
	c.JSON(http.StatusCreated, Envelope{Success: true, Code: "OK", Data: data, RequestID: RequestID(c)})
}
func Accepted(c *gin.Context, data any) {
	c.JSON(http.StatusAccepted, Envelope{Success: true, Code: "OK", Data: data, RequestID: RequestID(c)})
}
func Fail(c *gin.Context, err error) {
	var appErr *AppError
	if !errors.As(err, &appErr) {
		appErr = &AppError{Status: 500, Code: "INTERNAL_ERROR", Message: "internal server error", Cause: err}
	}
	message := appErr.Message
	c.AbortWithStatusJSON(appErr.Status, Envelope{Success: false, Code: appErr.Code, Message: &message, Data: nil, RequestID: RequestID(c)})
}

func RequestID(c *gin.Context) string {
	value, _ := c.Get("request_id")
	id, _ := value.(string)
	return id
}
