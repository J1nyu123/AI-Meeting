package config

import (
	"fmt"
	"os"
	"strconv"
	"time"
)

type Config struct {
	HTTPAddr                    string
	MySQLDSN                    string
	RedisAddr                   string
	RedisPassword               string
	JWTSecret                   string
	AccessTTL                   time.Duration
	RefreshTTL                  time.Duration
	StoragePath                 string
	MaxResumeBytes              int64
	LLMMode                     string
	LLMBaseURL                  string
	LLMAPIKey                   string
	LLMModel                    string
	XingChenBaseURL             string
	XingChenQuestionAPIKey      string
	XingChenQuestionAPISecret   string
	XingChenQuestionFlowID      string
	XingChenEvaluationAPIKey    string
	XingChenEvaluationAPISecret string
	XingChenEvaluationFlowID    string
	XingChenAskingAPIKey        string
	XingChenAskingAPISecret     string
	XingChenAskingFlowID        string
	XunfeiAppID                 string
	XunfeiAPIKey                string
	XunfeiAPISecret             string
	MediaASREnabled             bool
	MediaTTSEnabled             bool
	MediaASRMaxDuration         time.Duration
	MediaASRIdleTimeout         time.Duration
	MediaTTSTimeout             time.Duration
}

func Load() (Config, error) {
	accessTTL, err := time.ParseDuration(env("ACCESS_TOKEN_TTL", "15m"))
	if err != nil {
		return Config{}, fmt.Errorf("ACCESS_TOKEN_TTL: %w", err)
	}
	refreshTTL, err := time.ParseDuration(env("REFRESH_TOKEN_TTL", "168h"))
	if err != nil {
		return Config{}, fmt.Errorf("REFRESH_TOKEN_TTL: %w", err)
	}
	maxBytes, err := strconv.ParseInt(env("MAX_RESUME_BYTES", "10485760"), 10, 64)
	if err != nil {
		return Config{}, fmt.Errorf("MAX_RESUME_BYTES: %w", err)
	}
	cfg := Config{
		HTTPAddr: env("HTTP_ADDR", ":8080"), MySQLDSN: env("MYSQL_DSN", ""),
		RedisAddr: env("REDIS_ADDR", "127.0.0.1:6379"), RedisPassword: os.Getenv("REDIS_PASSWORD"),
		JWTSecret: env("JWT_SECRET", "development-only-change-this-secret"), AccessTTL: accessTTL,
		RefreshTTL: refreshTTL, StoragePath: env("FILE_STORAGE_PATH", "./data/uploads"),
		MaxResumeBytes: maxBytes, LLMMode: env("LLM_MODE", "mock"),
		LLMBaseURL: env("LLM_BASE_URL", "https://api.deepseek.com"), LLMAPIKey: os.Getenv("LLM_API_KEY"),
		LLMModel:                    env("LLM_MODEL", "deepseek-chat"),
		XingChenBaseURL:             env("XINGCHEN_BASE_URL", "https://xingchen-api.xf-yun.com"),
		XingChenQuestionAPIKey:      os.Getenv("XINGCHEN_QUESTION_API_KEY"),
		XingChenQuestionAPISecret:   os.Getenv("XINGCHEN_QUESTION_API_SECRET"),
		XingChenQuestionFlowID:      os.Getenv("XINGCHEN_QUESTION_FLOW_ID"),
		XingChenEvaluationAPIKey:    os.Getenv("XINGCHEN_EVALUATION_API_KEY"),
		XingChenEvaluationAPISecret: os.Getenv("XINGCHEN_EVALUATION_API_SECRET"),
		XingChenEvaluationFlowID:    os.Getenv("XINGCHEN_EVALUATION_FLOW_ID"),
		XingChenAskingAPIKey:        os.Getenv("XINGCHEN_ASKING_API_KEY"),
		XingChenAskingAPISecret:     os.Getenv("XINGCHEN_ASKING_API_SECRET"),
		XingChenAskingFlowID:        os.Getenv("XINGCHEN_ASKING_FLOW_ID"),
		XunfeiAppID:                 os.Getenv("XUNFEI_APP_ID"),
		XunfeiAPIKey:                os.Getenv("XUNFEI_API_KEY"),
		XunfeiAPISecret:             os.Getenv("XUNFEI_API_SECRET"),
	}
	if cfg.MediaASREnabled, err = strconv.ParseBool(env("MEDIA_ASR_ENABLED", "true")); err != nil {
		return Config{}, fmt.Errorf("MEDIA_ASR_ENABLED: %w", err)
	}
	if cfg.MediaTTSEnabled, err = strconv.ParseBool(env("MEDIA_TTS_ENABLED", "true")); err != nil {
		return Config{}, fmt.Errorf("MEDIA_TTS_ENABLED: %w", err)
	}
	if cfg.MediaASRMaxDuration, err = time.ParseDuration(env("MEDIA_ASR_MAX_DURATION", "15m")); err != nil {
		return Config{}, fmt.Errorf("MEDIA_ASR_MAX_DURATION: %w", err)
	}
	if cfg.MediaASRIdleTimeout, err = time.ParseDuration(env("MEDIA_ASR_IDLE_TIMEOUT", "45s")); err != nil {
		return Config{}, fmt.Errorf("MEDIA_ASR_IDLE_TIMEOUT: %w", err)
	}
	if cfg.MediaTTSTimeout, err = time.ParseDuration(env("MEDIA_TTS_TIMEOUT", "90s")); err != nil {
		return Config{}, fmt.Errorf("MEDIA_TTS_TIMEOUT: %w", err)
	}
	if cfg.MySQLDSN == "" {
		return Config{}, fmt.Errorf("MYSQL_DSN is required")
	}
	if len(cfg.JWTSecret) < 32 {
		return Config{}, fmt.Errorf("JWT_SECRET must contain at least 32 characters")
	}
	return cfg, nil
}

func env(key, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return fallback
}
