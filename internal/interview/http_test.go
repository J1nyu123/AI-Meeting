package interview

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"ai-meeting-go/internal/platform/httpx"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

func TestDeleteHTTPContract(t *testing.T) {
	gin.SetMode(gin.TestMode)
	db := interviewDB(t)
	session := Session{ID: uuid.NewString(), UserID: 7, Status: StatusCompleted}
	require.NoError(t, db.Create(&session).Error)

	handler := NewHandler(NewService(db, NewRuntimeStore(nil), fixedLLM{}), db)
	router := gin.New()
	router.Use(httpx.RequestMiddleware(), func(c *gin.Context) {
		c.Set("user_id", uint64(7))
		c.Next()
	})
	router.DELETE("/api/v1/interviews/:id", handler.Delete)

	request := httptest.NewRequest(http.MethodDelete, "/api/v1/interviews/"+session.ID, nil)
	response := httptest.NewRecorder()
	router.ServeHTTP(response, request)

	require.Equal(t, http.StatusOK, response.Code)
	require.Equal(t, "mysql", response.Header().Get("X-Degraded-Mode"))
	var envelope struct {
		Success bool `json:"success"`
		Data    struct {
			SessionID string `json:"sessionId"`
		} `json:"data"`
	}
	require.NoError(t, json.Unmarshal(response.Body.Bytes(), &envelope))
	require.True(t, envelope.Success)
	require.Equal(t, session.ID, envelope.Data.SessionID)
}
