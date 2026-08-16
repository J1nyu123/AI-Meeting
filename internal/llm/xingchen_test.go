package llm

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestXingChenUsesIndependentWorkflows(t *testing.T) {
	t.Helper()
	var uploadSeen bool
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/workflow/v1/upload_file":
			require.Equal(t, "Bearer question-key:question-secret", r.Header.Get("Authorization"))
			require.NoError(t, r.ParseMultipartForm(1<<20))
			file, _, err := r.FormFile("file")
			require.NoError(t, err)
			defer file.Close()
			data, err := io.ReadAll(file)
			require.NoError(t, err)
			require.Equal(t, "%PDF-test", string(data))
			uploadSeen = true
			writeJSON(t, w, map[string]any{"code": 0, "data": map[string]string{"url": "https://files.example/resume.pdf"}})
		case "/workflow/v1/chat/completions":
			var body struct {
				FlowID     string         `json:"flow_id"`
				ChatID     string         `json:"chat_id"`
				Stream     bool           `json:"stream"`
				History    []any          `json:"history"`
				Parameters map[string]any `json:"parameters"`
			}
			require.NoError(t, json.NewDecoder(r.Body).Decode(&body))
			require.False(t, body.Stream)
			require.Empty(t, body.History)
			switch body.FlowID {
			case "question-flow":
				require.Equal(t, "Bearer question-key:question-secret", r.Header.Get("Authorization"))
				require.Equal(t, "session-1", body.ChatID)
				require.Equal(t, "https://files.example/resume.pdf", body.Parameters["USER_FILE"])
				writeChatContent(t, w, `{"result":{"questions":["Q1","Q2"],"sugest":["S1","S2"],"type":"Go","resumeScore":"88"}}`)
			case "evaluation-flow":
				require.Equal(t, "Bearer evaluation-key:evaluation-secret", r.Header.Get("Authorization"))
				require.Equal(t, "session-1_score", body.ChatID)
				require.Equal(t, "Q1", body.Parameters["question"])
				require.Equal(t, "resume context", body.Parameters["resume_context"])
				writeChatContent(t, w, `{"score":55,"feedback":"需要补充","missing_points":["指标"],"follow_up_needed":true,"follow_up_question":"评分官建议"}`)
			case "asking-flow":
				require.Equal(t, "Bearer asking-key:asking-secret", r.Header.Get("Authorization"))
				require.Equal(t, "FOLLOW_UP", body.Parameters["mode"])
				require.Equal(t, float64(1), body.Parameters["follow_up_count"])
				writeChatContent(t, w, `{"ask_to_user":"请说明故障恢复过程？","end_interview":false}`)
			default:
				t.Fatalf("unexpected flow id %q", body.FlowID)
			}
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	client := NewXingChen(XingChenConfig{
		BaseURL:    server.URL,
		Question:   WorkflowConfig{APIKey: "question-key", APISecret: "question-secret", FlowID: "question-flow"},
		Evaluation: WorkflowConfig{APIKey: "evaluation-key", APISecret: "evaluation-secret", FlowID: "evaluation-flow"},
		Asking:     WorkflowConfig{APIKey: "asking-key", APISecret: "asking-secret", FlowID: "asking-flow"},
	})
	path := filepath.Join(t.TempDir(), "resume.pdf")
	require.NoError(t, os.WriteFile(path, []byte("%PDF-test"), 0600))

	analysis, err := client.AnalyzeResume(t.Context(), ResumeInput{SessionID: "session-1", FilePath: path, OriginalName: "resume.pdf"})
	require.NoError(t, err)
	require.True(t, uploadSeen)
	require.Equal(t, 88, analysis.ResumeScore)
	require.Equal(t, "Go", analysis.Direction)
	require.Equal(t, []QuestionDraft{{"1", "Q1", "S1"}, {"2", "Q2", "S2"}}, analysis.Questions)

	evaluated, err := client.EvaluateAnswer(t.Context(), AnswerInput{SessionID: "session-1", Question: "Q1", Answer: "answer", ResumeContext: "resume context"})
	require.NoError(t, err)
	require.Equal(t, 55, evaluated.Score)
	require.True(t, evaluated.FollowUpNeeded)

	followUp, err := client.GenerateFollowUp(t.Context(), FollowUpInput{SessionID: "session-1", Question: "Q1", Answer: "answer", ResumeContext: "resume context", Current: 1, Maximum: 2})
	require.NoError(t, err)
	require.Equal(t, "请说明故障恢复过程？", followUp.Question)
}

func TestXingChenValidationAndErrors(t *testing.T) {
	require.Equal(t, "https://xingchen-api.xf-yun.com", normalizeXingChenBaseURL("http(s)://xingchen-api.xf-yun.com/workflow/v1/chat/completions"))
	require.Equal(t, "https://xingchen-api.xf-yun.com", normalizeXingChenBaseURL("https://xingchen-api.xf-yun.com/workflow/v1/upload_file"))

	client := NewXingChen(XingChenConfig{})
	_, err := client.AnalyzeResume(t.Context(), ResumeInput{})
	require.ErrorContains(t, err, "not configured")
	_, err = client.EvaluateAnswer(t.Context(), AnswerInput{})
	require.ErrorContains(t, err, "not configured")
	_, err = client.GenerateFollowUp(t.Context(), FollowUpInput{})
	require.ErrorContains(t, err, "not configured")

	_, err = parseXingChenEvaluation([]byte(`{"code":0,"choices":[{"message":{"content":"{\"score\":101,\"feedback\":\"x\"}"}}]}`))
	require.ErrorContains(t, err, "invalid xingchen evaluation")
	_, err = parseXingChenResume([]byte(`{"code":7,"message":"workflow failed"}`))
	require.Error(t, err)

	followUp, err := parseXingChenFollowUp([]byte(`{"content":"直接追问内容"}`))
	require.NoError(t, err)
	require.Equal(t, "直接追问内容", followUp.Question)
}

func writeChatContent(t *testing.T, w http.ResponseWriter, content string) {
	t.Helper()
	writeJSON(t, w, map[string]any{"code": 0, "choices": []any{map[string]any{"message": map[string]any{"content": content}}}})
}

func writeJSON(t *testing.T, w http.ResponseWriter, value any) {
	t.Helper()
	w.Header().Set("Content-Type", "application/json")
	require.NoError(t, json.NewEncoder(w).Encode(value))
}
