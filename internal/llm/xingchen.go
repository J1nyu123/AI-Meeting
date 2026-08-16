package llm

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"ai-meeting-go/internal/evaluation"
)

const defaultXingChenBaseURL = "https://xingchen-api.xf-yun.com"

type WorkflowConfig struct {
	APIKey    string
	APISecret string
	FlowID    string
}

func (c WorkflowConfig) Complete() bool {
	return strings.TrimSpace(c.APIKey) != "" && strings.TrimSpace(c.APISecret) != "" && strings.TrimSpace(c.FlowID) != ""
}

type XingChenConfig struct {
	BaseURL    string
	Question   WorkflowConfig
	Evaluation WorkflowConfig
	Asking     WorkflowConfig
}

type ResumeInput struct {
	SessionID    string
	Text         string
	FilePath     string
	OriginalName string
}

type AnswerInput struct {
	SessionID     string
	Question      string
	Answer        string
	ResumeContext string
}

type FollowUpInput struct {
	SessionID     string
	Question      string
	Answer        string
	ResumeContext string
	Current       int
	Maximum       int
}

type FollowUpResult struct {
	Question     string
	EndInterview bool
}

type XingChen struct {
	config XingChenConfig
	client *http.Client
}

func NewXingChen(config XingChenConfig) *XingChen {
	baseURL := normalizeXingChenBaseURL(config.BaseURL)
	if baseURL == "" {
		baseURL = defaultXingChenBaseURL
	}
	config.BaseURL = baseURL
	return &XingChen{config: config, client: &http.Client{Timeout: 45 * time.Second}}
}

func normalizeXingChenBaseURL(value string) string {
	baseURL := strings.TrimRight(strings.TrimSpace(value), "/")
	baseURL = strings.Replace(baseURL, "http(s)://", "https://", 1)
	for _, suffix := range []string{"/workflow/v1/chat/completions", "/workflow/v1/upload_file"} {
		baseURL = strings.TrimSuffix(baseURL, suffix)
	}
	return strings.TrimRight(baseURL, "/")
}

func (c *XingChen) AnalyzeResume(ctx context.Context, input ResumeInput) (ResumeAnalysis, error) {
	if !c.config.Question.Complete() {
		return ResumeAnalysis{}, fmt.Errorf("xingchen question workflow is not configured")
	}
	fileURL, err := c.uploadFile(ctx, input.FilePath, input.OriginalName, c.config.Question)
	if err != nil {
		return ResumeAnalysis{}, fmt.Errorf("upload resume: %w", err)
	}
	prompt := "Extract technical interview questions from the uploaded resume. Return JSON only with keys questions, sugest, type, and resumeScore."
	raw, err := c.chat(ctx, input.SessionID, prompt, map[string]any{"USER_FILE": fileURL}, c.config.Question)
	if err != nil {
		return ResumeAnalysis{}, err
	}
	return parseXingChenResume(raw)
}

func (c *XingChen) EvaluateAnswer(ctx context.Context, input AnswerInput) (evaluation.Result, error) {
	if !c.config.Evaluation.Complete() {
		return evaluation.Result{}, fmt.Errorf("xingchen evaluation workflow is not configured")
	}
	raw, err := c.chat(ctx, input.SessionID+"_score", input.Answer, map[string]any{
		"question":       input.Question,
		"resume_context": truncate(input.ResumeContext, 20000),
	}, c.config.Evaluation)
	if err != nil {
		return evaluation.Result{}, err
	}
	return parseXingChenEvaluation(raw)
}

func (c *XingChen) GenerateFollowUp(ctx context.Context, input FollowUpInput) (FollowUpResult, error) {
	if !c.config.Asking.Complete() {
		return FollowUpResult{}, fmt.Errorf("xingchen asking workflow is not configured")
	}
	raw, err := c.chat(ctx, input.SessionID, input.Answer, map[string]any{
		"mode":            "FOLLOW_UP",
		"question":        input.Question,
		"resume_context":  truncate(input.ResumeContext, 20000),
		"follow_up_count": input.Current,
		"max_follow_up":   input.Maximum,
	}, c.config.Asking)
	if err != nil {
		return FollowUpResult{}, err
	}
	return parseXingChenFollowUp(raw)
}

func (c *XingChen) uploadFile(ctx context.Context, path, originalName string, workflow WorkflowConfig) (string, error) {
	file, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer file.Close()
	if strings.TrimSpace(originalName) == "" {
		originalName = filepath.Base(path)
	}
	var body bytes.Buffer
	writer := multipart.NewWriter(&body)
	part, err := writer.CreateFormFile("file", filepath.Base(originalName))
	if err != nil {
		return "", err
	}
	if _, err := io.Copy(part, file); err != nil {
		return "", err
	}
	if err := writer.Close(); err != nil {
		return "", err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.config.BaseURL+"/workflow/v1/upload_file", &body)
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", writer.FormDataContentType())
	setXingChenAuthorization(req, workflow)
	response, err := c.client.Do(req)
	if err != nil {
		return "", err
	}
	defer response.Body.Close()
	data, err := io.ReadAll(io.LimitReader(response.Body, 2<<20))
	if err != nil {
		return "", err
	}
	if response.StatusCode < 200 || response.StatusCode >= 300 {
		return "", fmt.Errorf("xingchen upload status %d", response.StatusCode)
	}
	var envelope struct {
		Code    int    `json:"code"`
		Message string `json:"message"`
		Data    struct {
			URL string `json:"url"`
		} `json:"data"`
	}
	if err := json.Unmarshal(data, &envelope); err != nil {
		return "", fmt.Errorf("invalid xingchen upload response: %w", err)
	}
	if envelope.Code != 0 {
		return "", fmt.Errorf("xingchen upload code %d: %s", envelope.Code, strings.TrimSpace(envelope.Message))
	}
	if strings.TrimSpace(envelope.Data.URL) == "" {
		return "", fmt.Errorf("xingchen upload response missing file url")
	}
	return envelope.Data.URL, nil
}

func (c *XingChen) chat(ctx context.Context, chatID, input string, parameters map[string]any, workflow WorkflowConfig) ([]byte, error) {
	values := make(map[string]any, len(parameters)+1)
	values["AGENT_USER_INPUT"] = input
	for key, value := range parameters {
		if value != nil {
			values[key] = value
		}
	}
	payload, err := json.Marshal(map[string]any{
		"flow_id":    workflow.FlowID,
		"uid":        "ai-meeting-go",
		"stream":     false,
		"chat_id":    chatID,
		"history":    []any{},
		"parameters": values,
	})
	if err != nil {
		return nil, err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.config.BaseURL+"/workflow/v1/chat/completions", bytes.NewReader(payload))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "text/event-stream")
	setXingChenAuthorization(req, workflow)
	response, err := c.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer response.Body.Close()
	data, err := io.ReadAll(io.LimitReader(response.Body, 2<<20))
	if err != nil {
		return nil, err
	}
	if response.StatusCode < 200 || response.StatusCode >= 300 {
		return nil, fmt.Errorf("xingchen chat status %d", response.StatusCode)
	}
	var envelope map[string]any
	if err := json.Unmarshal(data, &envelope); err != nil {
		return nil, fmt.Errorf("invalid xingchen chat response: %w", err)
	}
	if code, ok := integerValue(envelope["code"]); ok && code != 0 {
		return nil, fmt.Errorf("xingchen workflow code %d: %s", code, stringValue(envelope["message"]))
	}
	return data, nil
}

func setXingChenAuthorization(req *http.Request, workflow WorkflowConfig) {
	req.Header.Set("Authorization", "Bearer "+workflow.APIKey+":"+workflow.APISecret)
}

func parseXingChenResume(raw []byte) (ResumeAnalysis, error) {
	object, err := xingChenStructuredObject(raw, "questions")
	if err != nil {
		return ResumeAnalysis{}, err
	}
	questions := questionDrafts(object["questions"])
	suggestions := stringList(firstValue(object, "sugest", "suggestions"))
	for index := range questions {
		if questions[index].Number == "" {
			questions[index].Number = strconv.Itoa(index + 1)
		}
		if questions[index].Suggestion == "" && index < len(suggestions) {
			questions[index].Suggestion = suggestions[index]
		}
	}
	score, ok := integerValue(firstValue(object, "resumeScore", "resume_score", "score"))
	if !ok || score < 0 || score > 100 || len(questions) == 0 {
		return ResumeAnalysis{}, fmt.Errorf("invalid xingchen resume analysis")
	}
	direction := stringValue(firstValue(object, "type", "direction", "resumeType"))
	return ResumeAnalysis{Direction: direction, ResumeScore: score, Questions: questions}, nil
}

func parseXingChenEvaluation(raw []byte) (evaluation.Result, error) {
	object, err := xingChenStructuredObject(raw, "score")
	if err != nil {
		return evaluation.Result{}, err
	}
	score, ok := integerValue(object["score"])
	feedback := stringValue(object["feedback"])
	if !ok || score < 0 || score > 100 || feedback == "" {
		return evaluation.Result{}, fmt.Errorf("invalid xingchen evaluation")
	}
	return evaluation.Result{
		Score:            score,
		Feedback:         feedback,
		MissingPoints:    stringList(object["missing_points"]),
		FollowUpNeeded:   booleanValue(object["follow_up_needed"]),
		FollowUpQuestion: stringValue(object["follow_up_question"]),
	}, nil
}

func parseXingChenFollowUp(raw []byte) (FollowUpResult, error) {
	content, err := xingChenContent(raw)
	if err != nil {
		return FollowUpResult{}, err
	}
	var object map[string]any
	if decodeStructured(content, &object) == nil {
		if nested := findObjectWithKey(object, "ask_to_user"); nested != nil {
			object = nested
		}
		result := FollowUpResult{Question: stringValue(object["ask_to_user"]), EndInterview: booleanValue(object["end_interview"])}
		if result.EndInterview || result.Question != "" {
			return result, nil
		}
	}
	question := strings.TrimSpace(content)
	if question == "" {
		return FollowUpResult{}, fmt.Errorf("invalid xingchen follow-up")
	}
	return FollowUpResult{Question: question}, nil
}

func xingChenStructuredObject(raw []byte, requiredKey string) (map[string]any, error) {
	content, err := xingChenContent(raw)
	if err != nil {
		return nil, err
	}
	var object map[string]any
	if err := decodeStructured(content, &object); err != nil {
		return nil, err
	}
	matched := findObjectWithKey(object, requiredKey)
	if matched == nil {
		return nil, fmt.Errorf("xingchen response missing %s", requiredKey)
	}
	return matched, nil
}

func xingChenContent(raw []byte) (string, error) {
	var root map[string]any
	if err := json.Unmarshal(raw, &root); err != nil {
		return "", err
	}
	if choices, ok := root["choices"].([]any); ok && len(choices) > 0 {
		if choice, ok := choices[0].(map[string]any); ok {
			for _, key := range []string{"message", "delta"} {
				if holder, ok := choice[key].(map[string]any); ok {
					if content := contentValue(holder["content"]); content != "" {
						return content, nil
					}
				}
			}
		}
	}
	if content := contentValue(root["content"]); content != "" {
		return content, nil
	}
	return "", fmt.Errorf("xingchen response missing content")
}

func contentValue(value any) string {
	if text, ok := value.(string); ok {
		return strings.TrimSpace(text)
	}
	if value != nil {
		encoded, err := json.Marshal(value)
		if err == nil {
			return string(encoded)
		}
	}
	return ""
}

func findObjectWithKey(value any, key string) map[string]any {
	switch typed := value.(type) {
	case map[string]any:
		if _, ok := typed[key]; ok {
			return typed
		}
		for _, child := range typed {
			if found := findObjectWithKey(child, key); found != nil {
				return found
			}
		}
	case []any:
		for _, child := range typed {
			if found := findObjectWithKey(child, key); found != nil {
				return found
			}
		}
	}
	return nil
}

func firstValue(object map[string]any, keys ...string) any {
	for _, key := range keys {
		if value, ok := object[key]; ok && value != nil {
			return value
		}
	}
	return nil
}

func integerValue(value any) (int, bool) {
	switch typed := value.(type) {
	case float64:
		return int(typed + 0.5), true
	case int:
		return typed, true
	case json.Number:
		parsed, err := strconv.ParseFloat(string(typed), 64)
		return int(parsed + 0.5), err == nil
	case string:
		parsed, err := strconv.ParseFloat(strings.TrimSpace(typed), 64)
		return int(parsed + 0.5), err == nil
	default:
		return 0, false
	}
}

func booleanValue(value any) bool {
	if boolean, ok := value.(bool); ok {
		return boolean
	}
	switch strings.ToLower(stringValue(value)) {
	case "true", "1", "yes":
		return true
	default:
		return false
	}
}

func stringValue(value any) string {
	if value == nil {
		return ""
	}
	if text, ok := value.(string); ok {
		return strings.TrimSpace(text)
	}
	return strings.TrimSpace(fmt.Sprint(value))
}

func stringList(value any) []string {
	var result []string
	switch typed := value.(type) {
	case []any:
		for _, item := range typed {
			if text := stringValue(item); text != "" {
				result = append(result, text)
			}
		}
	case []string:
		for _, item := range typed {
			if text := strings.TrimSpace(item); text != "" {
				result = append(result, text)
			}
		}
	case string:
		text := strings.TrimSpace(typed)
		if strings.HasPrefix(text, "[") {
			var array []any
			if json.Unmarshal([]byte(text), &array) == nil {
				return stringList(array)
			}
		}
		for _, item := range strings.FieldsFunc(text, func(r rune) bool { return r == ',' || r == ';' || r == '\n' || r == '，' || r == '；' }) {
			if item = strings.TrimSpace(item); item != "" {
				result = append(result, item)
			}
		}
	}
	return result
}

func questionDrafts(value any) []QuestionDraft {
	var result []QuestionDraft
	switch typed := value.(type) {
	case []any:
		for _, item := range typed {
			switch question := item.(type) {
			case string:
				if content := strings.TrimSpace(question); content != "" {
					result = append(result, QuestionDraft{Content: content})
				}
			case map[string]any:
				content := stringValue(firstValue(question, "content", "question", "title"))
				if content != "" {
					result = append(result, QuestionDraft{Number: stringValue(question["number"]), Content: content, Suggestion: stringValue(firstValue(question, "suggestion", "sugest"))})
				}
			}
		}
	case []string:
		for _, question := range typed {
			if content := strings.TrimSpace(question); content != "" {
				result = append(result, QuestionDraft{Content: content})
			}
		}
	case string:
		for _, question := range stringList(typed) {
			result = append(result, QuestionDraft{Content: question})
		}
	}
	return result
}
