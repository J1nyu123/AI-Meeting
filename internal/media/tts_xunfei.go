package media

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

const xunfeiTTSHost = "api-dx.xf-yun.com"

type XunfeiTTSClient struct {
	AppID, APIKey, APISecret string
	BaseURL                  string
	HTTPClient               *http.Client
	Now                      func() time.Time
}

func (x *XunfeiTTSClient) Create(ctx context.Context, req TTSRequest) (ProviderTTSTask, error) {
	body := map[string]any{"header": map[string]any{"app_id": strings.TrimSpace(x.AppID)}, "parameter": map[string]any{"dts": map[string]any{"vcn": req.VCN, "language": req.Language, "speed": *req.Speed, "volume": *req.Volume, "pitch": *req.Pitch, "rhy": *req.Rhy, "audio": map[string]any{"encoding": req.AudioEncoding, "sample_rate": *req.SampleRate}, "pybuf": map[string]any{"encoding": "utf8", "compress": "raw", "format": "plain"}}}, "payload": map[string]any{"text": map[string]any{"encoding": "utf8", "compress": "raw", "format": "plain", "text": base64.StdEncoding.EncodeToString([]byte(req.Text))}}}
	return x.post(ctx, "/v1/private/dts_create", body)
}
func (x *XunfeiTTSClient) Query(ctx context.Context, taskID string) (ProviderTTSTask, error) {
	return x.post(ctx, "/v1/private/dts_query", map[string]any{"header": map[string]any{"app_id": strings.TrimSpace(x.AppID), "task_id": taskID}})
}

func (x *XunfeiTTSClient) post(ctx context.Context, path string, body any) (ProviderTTSTask, error) {
	if strings.TrimSpace(x.AppID) == "" || strings.TrimSpace(x.APIKey) == "" || strings.TrimSpace(x.APISecret) == "" {
		return ProviderTTSTask{}, fmt.Errorf("Xunfei TTS credentials are not configured")
	}
	raw, err := json.Marshal(body)
	if err != nil {
		return ProviderTTSTask{}, err
	}
	signedURL, date, authorization, err := x.signedURL(path)
	if err != nil {
		return ProviderTTSTask{}, err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, signedURL, bytes.NewReader(raw))
	if err != nil {
		return ProviderTTSTask{}, err
	}
	req.Host = xunfeiTTSHost
	req.Header.Set("Date", date)
	req.Header.Set("x-date", date)
	req.Header.Set("Authorization", authorization)
	req.Header.Set("Content-Type", "application/json")
	client := x.HTTPClient
	if client == nil {
		client = &http.Client{Timeout: 30 * time.Second}
	}
	resp, err := client.Do(req)
	if err != nil {
		return ProviderTTSTask{}, err
	}
	defer resp.Body.Close()
	limited, err := io.ReadAll(io.LimitReader(resp.Body, 2<<20))
	if err != nil {
		return ProviderTTSTask{}, err
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return ProviderTTSTask{}, fmt.Errorf("Xunfei TTS HTTP status %d", resp.StatusCode)
	}
	var payload struct {
		Header struct {
			Code       int    `json:"code"`
			Message    string `json:"message"`
			SID        string `json:"sid"`
			TaskID     string `json:"task_id"`
			TaskStatus any    `json:"task_status"`
		} `json:"header"`
		Payload struct {
			Audio struct {
				Audio string `json:"audio"`
			} `json:"audio"`
		} `json:"payload"`
	}
	if err := json.Unmarshal(limited, &payload); err != nil {
		return ProviderTTSTask{}, fmt.Errorf("invalid Xunfei TTS response")
	}
	result := ProviderTTSTask{TaskID: payload.Header.TaskID, SID: payload.Header.SID, Status: fmt.Sprint(payload.Header.TaskStatus), Code: payload.Header.Code, Message: payload.Header.Message, AudioURL: decodeBase64URL(payload.Payload.Audio.Audio)}
	if result.Code != 0 {
		return result, fmt.Errorf("Xunfei TTS business failure: code=%d", result.Code)
	}
	return result, nil
}
func (x *XunfeiTTSClient) signedURL(path string) (string, string, string, error) {
	now := time.Now
	if x.Now != nil {
		now = x.Now
	}
	date := now().UTC().Format(http.TimeFormat)
	origin := "host: " + xunfeiTTSHost + "\n" + "date: " + date + "\nPOST " + path + " HTTP/1.1"
	mac := hmac.New(sha256.New, []byte(strings.TrimSpace(x.APISecret)))
	_, _ = mac.Write([]byte(origin))
	signature := base64.StdEncoding.EncodeToString(mac.Sum(nil))
	authorizationOrigin := fmt.Sprintf(`api_key="%s", algorithm="hmac-sha256", headers="host date request-line", signature="%s"`, strings.TrimSpace(x.APIKey), signature)
	authorization := base64.StdEncoding.EncodeToString([]byte(authorizationOrigin))
	base := strings.TrimSuffix(x.BaseURL, "/")
	if base == "" {
		base = "https://" + xunfeiTTSHost
	}
	parsed, err := url.Parse(base + path)
	if err != nil {
		return "", "", "", err
	}
	query := parsed.Query()
	query.Set("host", xunfeiTTSHost)
	query.Set("date", date)
	query.Set("authorization", authorization)
	parsed.RawQuery = query.Encode()
	return parsed.String(), date, authorization, nil
}
func decodeBase64URL(value string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return ""
	}
	decoded, err := base64.StdEncoding.DecodeString(value)
	if err != nil {
		return upgradeXunfeiAudioURL(value)
	}
	return upgradeXunfeiAudioURL(string(decoded))
}

func upgradeXunfeiAudioURL(value string) string {
	parsed, err := url.Parse(strings.TrimSpace(value))
	if err == nil && parsed.Scheme == "http" && strings.EqualFold(parsed.Hostname(), "sgw-dx.xf-yun.com") {
		parsed.Scheme = "https"
		return parsed.String()
	}
	return value
}
