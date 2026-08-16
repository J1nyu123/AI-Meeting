package main

import (
	"log/slog"
	"os"

	"ai-meeting-go/internal/config"
	"ai-meeting-go/internal/job"
	"ai-meeting-go/internal/llm"
	"ai-meeting-go/internal/resume"
	"ai-meeting-go/internal/storage"
	"github.com/hibiken/asynq"
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
	client := llm.NewConfigured(llm.Options{
		Mode: cfg.LLMMode, OpenAIBaseURL: cfg.LLMBaseURL, OpenAIAPIKey: cfg.LLMAPIKey, OpenAIModel: cfg.LLMModel,
		XingChen: llm.XingChenConfig{
			BaseURL:    cfg.XingChenBaseURL,
			Question:   llm.WorkflowConfig{APIKey: cfg.XingChenQuestionAPIKey, APISecret: cfg.XingChenQuestionAPISecret, FlowID: cfg.XingChenQuestionFlowID},
			Evaluation: llm.WorkflowConfig{APIKey: cfg.XingChenEvaluationAPIKey, APISecret: cfg.XingChenEvaluationAPISecret, FlowID: cfg.XingChenEvaluationFlowID},
			Asking:     llm.WorkflowConfig{APIKey: cfg.XingChenAskingAPIKey, APISecret: cfg.XingChenAskingAPISecret, FlowID: cfg.XingChenAskingFlowID},
		},
	})
	redisClient := storage.OpenRedis(cfg.RedisAddr, cfg.RedisPassword)
	defer redisClient.Close()
	processor := job.NewProcessor(db, cfg.StoragePath, resume.TextPDFExtractor{}, client, redisClient)
	mux := asynq.NewServeMux()
	processor.Register(mux)
	server := asynq.NewServer(asynq.RedisClientOpt{Addr: cfg.RedisAddr, Password: cfg.RedisPassword}, asynq.Config{Concurrency: 4, Queues: map[string]int{"default": 1}})
	slog.Info("worker_started")
	if err := server.Run(mux); err != nil {
		slog.Error("worker_stopped", "error", err)
		os.Exit(1)
	}
}
