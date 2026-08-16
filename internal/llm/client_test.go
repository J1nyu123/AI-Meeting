package llm

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestDecodeStructured(t *testing.T) {
	var out ResumeAnalysis
	require.NoError(t, decodeStructured("```json\n{\"direction\":\"Go\",\"resume_score\":80,\"questions\":[]}\n```", &out))
	require.Equal(t, 80, out.ResumeScore)
}

func TestMockClientFlows(t *testing.T) {
	c := New("mock", "", "", "")
	analysis, err := c.AnalyzeResume(t.Context(), ResumeInput{Text: "Go developer"})
	require.NoError(t, err)
	require.Len(t, analysis.Questions, 5)
	short, err := c.EvaluateAnswer(t.Context(), AnswerInput{Question: "question", Answer: "too short"})
	require.NoError(t, err)
	require.Equal(t, 55, short.Score)
	require.True(t, short.FollowUpNeeded)
	long, err := c.EvaluateAnswer(t.Context(), AnswerInput{Question: "question", Answer: "这是一个包含完整背景、方案、异常处理和量化结果的足够长回答，用于验证正常评分路径。"})
	require.NoError(t, err)
	require.Equal(t, 75, long.Score)
	followUp, err := c.GenerateFollowUp(t.Context(), FollowUpInput{})
	require.NoError(t, err)
	require.NotEmpty(t, followUp.Question)
}

func TestOpenAICompatibleValidationAndHTTPFailures(t *testing.T) {
	tests := []struct {
		name   string
		status int
		body   string
	}{
		{"server error", http.StatusTooManyRequests, `{"error":"rate limited"}`},
		{"invalid envelope", http.StatusOK, `{}`},
		{"invalid structured content", http.StatusOK, `{"choices":[{"message":{"content":"not json"}}]}`},
		{"out of range score", http.StatusOK, `{"choices":[{"message":{"content":"{\"score\":101,\"feedback\":\"x\"}"}}]}`},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				require.Equal(t, "/chat/completions", r.URL.Path)
				w.WriteHeader(tt.status)
				_, _ = w.Write([]byte(tt.body))
			}))
			defer server.Close()
			c := New("openai", server.URL, "test-key", "test-model")
			_, err := c.EvaluateAnswer(context.Background(), AnswerInput{Question: "q", Answer: "a"})
			require.Error(t, err)
		})
	}
}

func TestConfiguredXingChenFallsBackPerCapability(t *testing.T) {
	client := NewConfigured(Options{Mode: "xingchen"})
	analysis, err := client.AnalyzeResume(t.Context(), ResumeInput{})
	require.NoError(t, err)
	require.Len(t, analysis.Questions, 5)
	evaluated, err := client.EvaluateAnswer(t.Context(), AnswerInput{Answer: "short"})
	require.NoError(t, err)
	require.Equal(t, 55, evaluated.Score)
	_, err = client.GenerateFollowUp(t.Context(), FollowUpInput{})
	require.ErrorContains(t, err, "not configured")
}

func TestDecodeStructuredAndTruncateFailures(t *testing.T) {
	var out ResumeAnalysis
	require.Error(t, decodeStructured("no object", &out))
	require.Error(t, decodeStructured("{bad json}", &out))
	require.Equal(t, "你好", truncate("你好世界", 2))
	require.Equal(t, "short", truncate("short", 10))
}
