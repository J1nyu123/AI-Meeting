package job

import (
	"bytes"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"net/textproto"
	"testing"

	"ai-meeting-go/internal/interview"
	"ai-meeting-go/internal/platform/httpx"
	"ai-meeting-go/internal/resume"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

func uploadRequest(t *testing.T, path, filename, contentType string, content []byte) *http.Request {
	var body bytes.Buffer
	w := multipart.NewWriter(&body)
	header := make(textproto.MIMEHeader)
	header.Set("Content-Disposition", `form-data; name="resume"; filename="`+filename+`"`)
	header.Set("Content-Type", contentType)
	part, err := w.CreatePart(header)
	require.NoError(t, err)
	_, err = part.Write(content)
	require.NoError(t, err)
	require.NoError(t, w.Close())
	req := httptest.NewRequest(http.MethodPost, path, &body)
	req.Header.Set("Content-Type", w.FormDataContentType())
	return req
}

func TestUploadRejectsOwnershipTypeSignatureAndSize(t *testing.T) {
	gin.SetMode(gin.TestMode)
	db := jobDB(t)
	session := interview.Session{ID: uuid.NewString(), UserID: 7, Status: interview.StatusCreated}
	require.NoError(t, db.Create(&session).Error)
	handler := NewHandler(db, resume.LocalStorage{Root: t.TempDir()}, nil, nil, 16)
	router := gin.New()
	router.Use(httpx.RequestMiddleware(), func(c *gin.Context) { c.Set("user_id", uint64(7)); c.Next() })
	router.POST("/interviews/:id/resume", handler.Upload)

	tests := []struct {
		name     string
		session  string
		filename string
		mime     string
		content  []byte
		status   int
		code     string
	}{
		{"ownership", uuid.NewString(), "resume.pdf", "application/pdf", []byte("%PDF-ok"), http.StatusNotFound, "INTERVIEW_NOT_FOUND"},
		{"type", session.ID, "resume.txt", "text/plain", []byte("plain"), http.StatusBadRequest, "INVALID_PDF"},
		{"signature", session.ID, "resume.pdf", "application/pdf", []byte("plain text"), http.StatusBadRequest, "INVALID_PDF"},
		{"size", session.ID, "resume.pdf", "application/pdf", []byte("%PDF-this is larger than sixteen bytes"), http.StatusBadRequest, "RESUME_TOO_LARGE"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := uploadRequest(t, "/interviews/"+tt.session+"/resume", tt.filename, tt.mime, tt.content)
			recorder := httptest.NewRecorder()
			router.ServeHTTP(recorder, req)
			require.Equal(t, tt.status, recorder.Code)
			require.Contains(t, recorder.Body.String(), tt.code)
		})
	}
}

func TestJobStatusAndSSEReconnectReturnAuthoritativeFinalState(t *testing.T) {
	gin.SetMode(gin.TestMode)
	db := jobDB(t)
	record := AnalysisJob{ID: uuid.NewString(), SessionID: uuid.NewString(), UserID: 7, Status: "COMPLETED", Progress: 100, Stage: "completed"}
	require.NoError(t, db.Create(&record).Error)
	handler := NewHandler(db, resume.LocalStorage{Root: t.TempDir()}, nil, nil, 1024)
	router := gin.New()
	router.Use(httpx.RequestMiddleware(), func(c *gin.Context) { c.Set("user_id", uint64(7)); c.Next() })
	router.GET("/jobs/:jobId", handler.Get)
	router.GET("/jobs/:jobId/events", handler.Events)

	statusReq := httptest.NewRequest(http.MethodGet, "/jobs/"+record.ID, nil)
	statusRecorder := httptest.NewRecorder()
	router.ServeHTTP(statusRecorder, statusReq)
	require.Equal(t, http.StatusOK, statusRecorder.Code)
	require.Contains(t, statusRecorder.Body.String(), `"status":"COMPLETED"`)

	eventReq := httptest.NewRequest(http.MethodGet, "/jobs/"+record.ID+"/events", nil)
	eventReq.Header.Set("Last-Event-ID", "previous-event")
	eventRecorder := httptest.NewRecorder()
	router.ServeHTTP(eventRecorder, eventReq)
	require.Equal(t, http.StatusOK, eventRecorder.Code)
	require.Equal(t, "text/event-stream", eventRecorder.Header().Get("Content-Type"))
	require.Contains(t, eventRecorder.Body.String(), "event: status")
	require.Contains(t, eventRecorder.Body.String(), `"status":"COMPLETED"`)

	missingReq := httptest.NewRequest(http.MethodGet, "/jobs/"+uuid.NewString(), nil)
	missingRecorder := httptest.NewRecorder()
	router.ServeHTTP(missingRecorder, missingReq)
	require.Equal(t, http.StatusNotFound, missingRecorder.Code)
}
