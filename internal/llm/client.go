package llm

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"ai-meeting-go/internal/evaluation"
)

type QuestionDraft struct {
	Number     string `json:"number"`
	Content    string `json:"content"`
	Suggestion string `json:"suggestion"`
}
type ResumeAnalysis struct {
	Direction   string          `json:"direction"`
	ResumeScore int             `json:"resume_score"`
	Questions   []QuestionDraft `json:"questions"`
}
type Client interface {
	AnalyzeResume(context.Context, ResumeInput) (ResumeAnalysis, error)
	EvaluateAnswer(context.Context, AnswerInput) (evaluation.Result, error)
	GenerateFollowUp(context.Context, FollowUpInput) (FollowUpResult, error)
}

type OpenAICompatible struct {
	mode, baseURL, apiKey, model string
	client                       *http.Client
}

func New(mode, baseURL, apiKey, model string) Client {
	return NewConfigured(Options{Mode: mode, OpenAIBaseURL: baseURL, OpenAIAPIKey: apiKey, OpenAIModel: model})
}

func newOpenAICompatible(mode, baseURL, apiKey, model string) *OpenAICompatible {
	return &OpenAICompatible{mode: mode, baseURL: strings.TrimRight(baseURL, "/"), apiKey: apiKey, model: model, client: &http.Client{Timeout: 45 * time.Second}}
}

func (c *OpenAICompatible) AnalyzeResume(ctx context.Context, input ResumeInput) (ResumeAnalysis, error) {
	if c.mode == "mock" {
		return ResumeAnalysis{Direction: "Go 后端开发", ResumeScore: 78, Questions: []QuestionDraft{{"1", "请介绍你在项目中承担的核心职责。", "说明个人贡献和结果"}, {"2", "为什么使用 Redis 保存会话热状态？", "说明数据边界和恢复"}, {"3", "如何避免同一答案被重复评分？", "说明幂等键和唯一约束"}, {"4", "如何处理大模型输出格式不稳定？", "说明校验与修复"}, {"5", "如何设计系统的降级路径？", "说明 Redis 故障场景"}}}, nil
	}
	prompt := "分析以下简历，返回严格JSON：{direction:string,resume_score:0-100,questions:[{number,content,suggestion}]}，生成5道与经历相关的问题。简历：\n" + truncate(input.Text, 20000)
	raw, err := c.complete(ctx, prompt)
	if err != nil {
		return ResumeAnalysis{}, err
	}
	var out ResumeAnalysis
	if err := decodeStructured(raw, &out); err != nil {
		return ResumeAnalysis{}, err
	}
	if out.ResumeScore < 0 || out.ResumeScore > 100 || len(out.Questions) == 0 {
		return ResumeAnalysis{}, fmt.Errorf("invalid resume analysis")
	}
	return out, nil
}
func (c *OpenAICompatible) EvaluateAnswer(ctx context.Context, input AnswerInput) (evaluation.Result, error) {
	if c.mode == "mock" {
		score := 75
		if len([]rune(input.Answer)) < 30 {
			score = 55
		}
		return evaluation.Result{Score: score, Feedback: "回答覆盖了主要思路，可以进一步补充量化结果和异常场景。", MissingPoints: []string{"量化结果"}, FollowUpNeeded: score < 60, FollowUpQuestion: "请结合一次具体故障说明你的处理过程。"}, nil
	}
	prompt := "评价面试回答，返回严格JSON：{score:0-100,feedback:string,missing_points:string[],follow_up_needed:boolean,follow_up_question:string}。问题：" + input.Question + "\n回答：" + input.Answer + "\n简历上下文：" + truncate(input.ResumeContext, 20000)
	raw, err := c.complete(ctx, prompt)
	if err != nil {
		return evaluation.Result{}, err
	}
	var out evaluation.Result
	if err := decodeStructured(raw, &out); err != nil {
		return out, err
	}
	if out.Score < 0 || out.Score > 100 || strings.TrimSpace(out.Feedback) == "" {
		return out, fmt.Errorf("invalid evaluation")
	}
	return out, nil
}

func (c *OpenAICompatible) GenerateFollowUp(ctx context.Context, input FollowUpInput) (FollowUpResult, error) {
	if c.mode == "mock" {
		return FollowUpResult{Question: "请结合一次具体故障说明你的处理过程。"}, nil
	}
	prompt := "根据当前面试问题和回答生成一条简短追问，返回严格JSON：{ask_to_user:string,end_interview:boolean}。问题：" + input.Question + "\n回答：" + input.Answer + "\n简历上下文：" + truncate(input.ResumeContext, 20000)
	raw, err := c.complete(ctx, prompt)
	if err != nil {
		return FollowUpResult{}, err
	}
	var out struct {
		Question     string `json:"ask_to_user"`
		EndInterview bool   `json:"end_interview"`
	}
	if err := decodeStructured(raw, &out); err != nil {
		return FollowUpResult{}, err
	}
	if !out.EndInterview && strings.TrimSpace(out.Question) == "" {
		return FollowUpResult{}, fmt.Errorf("invalid follow-up")
	}
	return FollowUpResult{Question: strings.TrimSpace(out.Question), EndInterview: out.EndInterview}, nil
}

type Options struct {
	Mode          string
	OpenAIBaseURL string
	OpenAIAPIKey  string
	OpenAIModel   string
	XingChen      XingChenConfig
}

type configuredClient struct {
	mode     string
	mock     *OpenAICompatible
	openAI   *OpenAICompatible
	xingChen *XingChen
	config   XingChenConfig
}

func NewConfigured(options Options) Client {
	mode := strings.ToLower(strings.TrimSpace(options.Mode))
	if mode == "" {
		mode = "mock"
	}
	client := &configuredClient{
		mode:     mode,
		mock:     newOpenAICompatible("mock", "", "", ""),
		openAI:   newOpenAICompatible("openai", options.OpenAIBaseURL, options.OpenAIAPIKey, options.OpenAIModel),
		xingChen: NewXingChen(options.XingChen),
		config:   options.XingChen,
	}
	if mode == "xingchen" {
		client.warnMissingWorkflow("question", options.XingChen.Question)
		client.warnMissingWorkflow("evaluation", options.XingChen.Evaluation)
		client.warnMissingWorkflow("asking", options.XingChen.Asking)
	}
	return client
}

func (c *configuredClient) AnalyzeResume(ctx context.Context, input ResumeInput) (ResumeAnalysis, error) {
	switch c.mode {
	case "xingchen":
		if c.config.Question.Complete() {
			return c.xingChen.AnalyzeResume(ctx, input)
		}
		return c.mock.AnalyzeResume(ctx, input)
	case "openai":
		return c.openAI.AnalyzeResume(ctx, input)
	default:
		return c.mock.AnalyzeResume(ctx, input)
	}
}

func (c *configuredClient) EvaluateAnswer(ctx context.Context, input AnswerInput) (evaluation.Result, error) {
	switch c.mode {
	case "xingchen":
		if c.config.Evaluation.Complete() {
			return c.xingChen.EvaluateAnswer(ctx, input)
		}
		return c.mock.EvaluateAnswer(ctx, input)
	case "openai":
		return c.openAI.EvaluateAnswer(ctx, input)
	default:
		return c.mock.EvaluateAnswer(ctx, input)
	}
}

func (c *configuredClient) GenerateFollowUp(ctx context.Context, input FollowUpInput) (FollowUpResult, error) {
	switch c.mode {
	case "xingchen":
		if c.config.Asking.Complete() {
			return c.xingChen.GenerateFollowUp(ctx, input)
		}
		return FollowUpResult{}, fmt.Errorf("xingchen asking workflow is not configured")
	case "openai":
		return c.openAI.GenerateFollowUp(ctx, input)
	default:
		return c.mock.GenerateFollowUp(ctx, input)
	}
}

func (c *configuredClient) warnMissingWorkflow(name string, workflow WorkflowConfig) {
	if workflow.Complete() {
		return
	}
	missing := make([]string, 0, 3)
	if strings.TrimSpace(workflow.APIKey) == "" {
		missing = append(missing, "api_key")
	}
	if strings.TrimSpace(workflow.APISecret) == "" {
		missing = append(missing, "api_secret")
	}
	if strings.TrimSpace(workflow.FlowID) == "" {
		missing = append(missing, "flow_id")
	}
	slog.Warn("xingchen_workflow_fallback", "workflow", name, "missing", strings.Join(missing, ","))
}
func (c *OpenAICompatible) complete(ctx context.Context, prompt string) (string, error) {
	body, _ := json.Marshal(map[string]any{"model": c.model, "temperature": 0.2, "response_format": map[string]string{"type": "json_object"}, "messages": []map[string]string{{"role": "user", "content": prompt}}})
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+"/chat/completions", bytes.NewReader(body))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+c.apiKey)
	resp, err := c.client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	data, _ := io.ReadAll(io.LimitReader(resp.Body, 2<<20))
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return "", fmt.Errorf("llm status %d: %s", resp.StatusCode, string(data))
	}
	var decoded struct {
		Choices []struct {
			Message struct {
				Content string `json:"content"`
			} `json:"message"`
		} `json:"choices"`
	}
	if err := json.Unmarshal(data, &decoded); err != nil || len(decoded.Choices) == 0 {
		return "", fmt.Errorf("invalid llm response")
	}
	return decoded.Choices[0].Message.Content, nil
}
func decodeStructured(raw string, out any) error {
	raw = strings.TrimSpace(raw)
	raw = strings.TrimPrefix(raw, "```json")
	raw = strings.TrimPrefix(raw, "```")
	raw = strings.TrimSuffix(raw, "```")
	start, end := strings.Index(raw, "{"), strings.LastIndex(raw, "}")
	if start < 0 || end < start {
		return fmt.Errorf("json object missing")
	}
	return json.Unmarshal([]byte(raw[start:end+1]), out)
}
func truncate(s string, n int) string {
	r := []rune(s)
	if len(r) <= n {
		return s
	}
	return string(r[:n])
}
