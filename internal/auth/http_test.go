package auth

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"ai-meeting-go/internal/platform/httpx"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

func authRouter(t *testing.T) *gin.Engine {
	gin.SetMode(gin.TestMode)
	service := testService(t)
	handler := NewHandler(service, service.db)
	router := gin.New()
	router.Use(httpx.RequestMiddleware(), httpx.Recovery())
	api := router.Group("/api/v1")
	RegisterRoutes(api.Group("/auth"), handler)
	secured := api.Group("")
	secured.Use(handler.Middleware())
	secured.GET("/auth/me", handler.Me)
	return router
}

func performJSON(router http.Handler, method, path string, body any, token string) *httptest.ResponseRecorder {
	data, _ := json.Marshal(body)
	req := httptest.NewRequest(method, path, bytes.NewReader(data))
	req.Header.Set("Content-Type", "application/json")
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, req)
	return recorder
}

func TestAuthHTTPFlowAndProtection(t *testing.T) {
	router := authRouter(t)

	bad := performJSON(router, http.MethodPost, "/api/v1/auth/register", map[string]string{"username": "candidate"}, "")
	require.Equal(t, http.StatusBadRequest, bad.Code)

	registered := performJSON(router, http.MethodPost, "/api/v1/auth/register", map[string]string{"username": "candidate", "password": "password123"}, "")
	require.Equal(t, http.StatusCreated, registered.Code)
	require.NotEmpty(t, registered.Header().Get("X-Request-ID"))

	wrong := performJSON(router, http.MethodPost, "/api/v1/auth/login", map[string]string{"username": "candidate", "password": "wrong"}, "")
	require.Equal(t, http.StatusUnauthorized, wrong.Code)
	login := performJSON(router, http.MethodPost, "/api/v1/auth/login", map[string]string{"username": "candidate", "password": "password123"}, "")
	require.Equal(t, http.StatusOK, login.Code)
	var envelope struct {
		Data TokenPair `json:"data"`
	}
	require.NoError(t, json.Unmarshal(login.Body.Bytes(), &envelope))
	require.NotEmpty(t, envelope.Data.AccessToken)
	require.NotEmpty(t, envelope.Data.RefreshToken)

	missing := performJSON(router, http.MethodGet, "/api/v1/auth/me", nil, "")
	require.Equal(t, http.StatusUnauthorized, missing.Code)
	wrongType := performJSON(router, http.MethodGet, "/api/v1/auth/me", nil, envelope.Data.RefreshToken)
	require.Equal(t, http.StatusUnauthorized, wrongType.Code)
	me := performJSON(router, http.MethodGet, "/api/v1/auth/me", nil, envelope.Data.AccessToken)
	require.Equal(t, http.StatusOK, me.Code)
	require.Contains(t, me.Body.String(), "candidate")

	refresh := performJSON(router, http.MethodPost, "/api/v1/auth/refresh", map[string]string{"refreshToken": envelope.Data.RefreshToken}, "")
	require.Equal(t, http.StatusOK, refresh.Code)
	var refreshed struct {
		Data TokenPair `json:"data"`
	}
	require.NoError(t, json.Unmarshal(refresh.Body.Bytes(), &refreshed))
	require.NotEqual(t, envelope.Data.RefreshToken, refreshed.Data.RefreshToken)

	logout := performJSON(router, http.MethodPost, "/api/v1/auth/logout", map[string]string{"refreshToken": refreshed.Data.RefreshToken}, "")
	require.Equal(t, http.StatusOK, logout.Code)
	rejected := performJSON(router, http.MethodPost, "/api/v1/auth/refresh", map[string]string{"refreshToken": refreshed.Data.RefreshToken}, "")
	require.Equal(t, http.StatusUnauthorized, rejected.Code)
}
