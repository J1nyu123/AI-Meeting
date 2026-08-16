package httpx

import (
	"log/slog"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

var httpRequests = promauto.NewCounterVec(prometheus.CounterOpts{Name: "ai_meeting_http_requests_total", Help: "HTTP requests by method, route and status."}, []string{"method", "route", "status"})
var httpDuration = promauto.NewHistogramVec(prometheus.HistogramOpts{Name: "ai_meeting_http_request_duration_seconds", Help: "HTTP request latency.", Buckets: prometheus.DefBuckets}, []string{"method", "route"})

func RequestMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		id := c.GetHeader("X-Request-ID")
		if id == "" {
			id = uuid.NewString()
		}
		c.Set("request_id", id)
		c.Header("X-Request-ID", id)
		started := time.Now()
		c.Next()
		route := c.FullPath()
		if route == "" {
			route = "unmatched"
		}
		httpRequests.WithLabelValues(c.Request.Method, route, http.StatusText(c.Writer.Status())).Inc()
		httpDuration.WithLabelValues(c.Request.Method, route).Observe(time.Since(started).Seconds())
		slog.Info("http_request", "request_id", id, "method", c.Request.Method, "path", c.Request.URL.Path, "status", c.Writer.Status(), "duration_ms", time.Since(started).Milliseconds())
	}
}

func Recovery() gin.HandlerFunc {
	return gin.CustomRecovery(func(c *gin.Context, recovered any) {
		slog.Error("panic_recovered", "request_id", RequestID(c), "error", recovered)
		message := "internal server error"
		c.AbortWithStatusJSON(http.StatusInternalServerError, Envelope{Success: false, Code: "INTERNAL_ERROR", Message: &message, RequestID: RequestID(c)})
	})
}
