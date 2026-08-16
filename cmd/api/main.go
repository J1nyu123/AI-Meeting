package main

import (
	"context"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"ai-meeting-go/internal/auth"
	"ai-meeting-go/internal/config"
	"ai-meeting-go/internal/interview"
	"ai-meeting-go/internal/job"
	"ai-meeting-go/internal/llm"
	"ai-meeting-go/internal/platform/httpx"
	"ai-meeting-go/internal/report"
	"ai-meeting-go/internal/resume"
	"ai-meeting-go/internal/storage"
	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"github.com/hibiken/asynq"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		slog.Error("configuration failed", "error", err)
		os.Exit(1)
	}
	db, err := storage.OpenMySQL(cfg.MySQLDSN)
	if err != nil {
		slog.Error("mysql failed", "error", err)
		os.Exit(1)
	}
	redisClient := storage.OpenRedis(cfg.RedisAddr, cfg.RedisPassword)
	defer redisClient.Close()
	authService := auth.NewService(db, cfg.JWTSecret, cfg.AccessTTL, cfg.RefreshTTL)
	authHandler := auth.NewHandler(authService, db)
	llmClient := llm.NewConfigured(llm.Options{
		Mode: cfg.LLMMode, OpenAIBaseURL: cfg.LLMBaseURL, OpenAIAPIKey: cfg.LLMAPIKey, OpenAIModel: cfg.LLMModel,
		XingChen: llm.XingChenConfig{
			BaseURL:    cfg.XingChenBaseURL,
			Question:   llm.WorkflowConfig{APIKey: cfg.XingChenQuestionAPIKey, APISecret: cfg.XingChenQuestionAPISecret, FlowID: cfg.XingChenQuestionFlowID},
			Evaluation: llm.WorkflowConfig{APIKey: cfg.XingChenEvaluationAPIKey, APISecret: cfg.XingChenEvaluationAPISecret, FlowID: cfg.XingChenEvaluationFlowID},
			Asking:     llm.WorkflowConfig{APIKey: cfg.XingChenAskingAPIKey, APISecret: cfg.XingChenAskingAPISecret, FlowID: cfg.XingChenAskingFlowID},
		},
	})
	runtime := interview.NewRuntimeStore(redisClient)
	fileStorage := resume.LocalStorage{Root: cfg.StoragePath}
	interviewService := interview.NewService(db, runtime, llmClient).WithFileStorage(fileStorage)
	interviewHandler := interview.NewHandler(interviewService, db)
	reportService := report.NewService(db)
	reportHandler := report.NewHandler(reportService, db)
	publisher := job.NewPublisher(asynq.RedisClientOpt{Addr: cfg.RedisAddr, Password: cfg.RedisPassword})
	defer publisher.Close()
	jobHandler := job.NewHandler(db, fileStorage, publisher, redisClient, cfg.MaxResumeBytes)
	r := gin.New()
	r.Use(httpx.RequestMiddleware(), httpx.Recovery(), cors.New(cors.Config{AllowOrigins: []string{"http://localhost:5173"}, AllowMethods: []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"}, AllowHeaders: []string{"Authorization", "Content-Type", "Idempotency-Key", "Last-Event-ID", "X-Request-ID"}, ExposeHeaders: []string{"X-Request-ID", "X-Degraded-Mode"}, AllowCredentials: true, MaxAge: 12 * time.Hour}))
	r.GET("/health/live", func(c *gin.Context) { httpx.OK(c, gin.H{"status": "UP"}) })
	r.GET("/metrics", gin.WrapH(promhttp.Handler()))
	r.GET("/health/ready", func(c *gin.Context) {
		sqlDB, _ := db.DB()
		ctx, cancel := context.WithTimeout(c, 2*time.Second)
		defer cancel()
		if err := sqlDB.PingContext(ctx); err != nil {
			httpx.Fail(c, err)
			return
		}
		httpx.OK(c, gin.H{"status": "UP"})
	})
	v1 := r.Group("/api/v1")
	auth.RegisterRoutes(v1.Group("/auth"), authHandler)
	secured := v1.Group("")
	secured.Use(authHandler.Middleware())
	secured.GET("/auth/me", authHandler.Me)
	secured.POST("/interviews", interviewHandler.Create)
	secured.GET("/interviews", interviewHandler.List)
	secured.DELETE("/interviews/:id", interviewHandler.Delete)
	secured.POST("/interviews/:id/resume", jobHandler.Upload)
	secured.GET("/interviews/:id/resume", jobHandler.Preview)
	secured.GET("/interviews/:id/state", interviewHandler.State)
	secured.POST("/interviews/:id/answers", interviewHandler.Answer)
	secured.POST("/interviews/:id/finish", reportHandler.Finish)
	secured.GET("/interviews/:id/report", reportHandler.Get)
	secured.GET("/jobs/:jobId", jobHandler.Get)
	secured.GET("/jobs/:jobId/events", jobHandler.Events)
	server := &http.Server{Addr: cfg.HTTPAddr, Handler: r, ReadHeaderTimeout: 5 * time.Second}
	go func() {
		slog.Info("api_started", "addr", cfg.HTTPAddr)
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			slog.Error("api_stopped", "error", err)
			os.Exit(1)
		}
	}()
	stop := make(chan os.Signal, 1)
	signal.Notify(stop, syscall.SIGINT, syscall.SIGTERM)
	<-stop
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	_ = server.Shutdown(ctx)
}
