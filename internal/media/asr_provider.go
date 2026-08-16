package media

import (
	"context"
	"crypto/hmac"
	"crypto/sha1"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"net/url"
	"sort"
	"strconv"
	"strings"
	"sync/atomic"
	"time"

	"github.com/coder/websocket"
	"github.com/google/uuid"
)

const defaultASTURL = "wss://office-api-ast-dx.iflyaisol.com/ast/communicate/v1"

type ASREvent struct {
	Type          string `json:"type"`
	Code          string `json:"code"`
	SessionID     string `json:"sessionId"`
	Message       string `json:"message,omitempty"`
	DisplayText   string `json:"displayText,omitempty"`
	CommittedText string `json:"committedText,omitempty"`
	LiveText      string `json:"liveText,omitempty"`
	Revision      int64  `json:"revision,omitempty"`
	ResultStatus  int    `json:"resultStatus,omitempty"`
	UpdateAction  string `json:"updateAction,omitempty"`
	Timestamp     int64  `json:"timestamp"`
	SegmentID     int    `json:"segmentId,omitempty"`
	SentenceSeq   int    `json:"sentenceSeq,omitempty"`
	SegmentText   string `json:"segmentText,omitempty"`
	PGS           string `json:"pgs,omitempty"`
	Range         []int  `json:"rg,omitempty"`
	Begin         *int   `json:"bg,omitempty"`
	End           *int   `json:"ed,omitempty"`
	IsFinalPacket bool   `json:"isFinalPacket,omitempty"`
	Retryable     bool   `json:"retryable"`
}

type ASRProvider interface {
	Run(context.Context, string, <-chan []byte, chan<- ASREvent) error
}

type XunfeiASTClient struct {
	AppID, APIKey, APISecret string
	Endpoint                 string
	Now                      func() time.Time
}

func (x *XunfeiASTClient) Run(ctx context.Context, sessionID string, audio <-chan []byte, events chan<- ASREvent) error {
	if strings.TrimSpace(x.AppID) == "" || strings.TrimSpace(x.APIKey) == "" || strings.TrimSpace(x.APISecret) == "" {
		return errors.New("Xunfei AST credentials are not configured")
	}
	signedURL, err := x.signedURL(sessionID, uuid.NewString())
	if err != nil {
		return err
	}
	conn, _, err := websocket.Dial(ctx, signedURL, nil)
	if err != nil {
		return fmt.Errorf("connect Xunfei AST: %w", err)
	}
	defer conn.CloseNow()
	errCh := make(chan error, 2)
	go func() {
		for chunk := range audio {
			if len(chunk) == 0 {
				continue
			}
			if err := conn.Write(ctx, websocket.MessageBinary, chunk); err != nil {
				errCh <- err
				conn.CloseNow()
				return
			}
			select {
			case <-time.After(40 * time.Millisecond):
			case <-ctx.Done():
				errCh <- ctx.Err()
				conn.CloseNow()
				return
			}
		}
		end, _ := json.Marshal(map[string]any{"end": true, "sessionId": sessionID})
		if err := conn.Write(ctx, websocket.MessageText, end); err != nil {
			errCh <- err
			conn.CloseNow()
			return
		}
		errCh <- nil
	}()
	assembler := NewASTAssembler()
	var fallback atomic.Int64
	for {
		select {
		case err := <-errCh:
			if err != nil {
				return fmt.Errorf("send AST audio: %w", err)
			}
		default:
		}
		kind, payload, err := conn.Read(ctx)
		if err != nil {
			return fmt.Errorf("read Xunfei AST: %w", err)
		}
		if kind != websocket.MessageText {
			continue
		}
		segment, finalPacket, providerErr := parseASTPacket(payload, &fallback)
		if providerErr != nil {
			return providerErr
		}
		if segment.Text != "" {
			snapshot, changed := assembler.Apply(segment, finalPacket)
			if changed {
				eventType, action := "transcription", "replace"
				if finalPacket {
					eventType, action = "final", "archive"
				}
				events <- ASREvent{Type: eventType, Code: "OK", SessionID: sessionID, DisplayText: snapshot.DisplayText, CommittedText: snapshot.CommittedText, LiveText: snapshot.LiveText, Revision: snapshot.Revision, ResultStatus: boolInt(finalPacket), UpdateAction: action, Timestamp: time.Now().UnixMilli(), SegmentID: segment.ID, SentenceSeq: segment.ID, SegmentText: segment.Text, PGS: segment.PGS, Range: segment.Range, Begin: segment.Begin, End: segment.End, IsFinalPacket: finalPacket}
			}
		}
		if finalPacket {
			_ = conn.Close(websocket.StatusNormalClosure, "completed")
			return nil
		}
	}
}

func (x *XunfeiASTClient) signedURL(sessionID, nonce string) (string, error) {
	now := time.Now
	if x.Now != nil {
		now = x.Now
	}
	endpoint := x.Endpoint
	if endpoint == "" {
		endpoint = defaultASTURL
	}
	params := map[string]string{"appId": strings.TrimSpace(x.AppID), "accessKeyId": strings.TrimSpace(x.APIKey), "audio_encode": "pcm_s16le", "lang": "autodialect", "samplerate": "16000", "sessionId": sessionID, "utc": now().Format("2006-01-02T15:04:05-0700"), "uuid": strings.ReplaceAll(nonce, "-", "")}
	canonical := canonicalQuery(params)
	mac := hmac.New(sha1.New, []byte(strings.TrimSpace(x.APISecret)))
	_, _ = mac.Write([]byte(canonical))
	params["signature"] = base64.StdEncoding.EncodeToString(mac.Sum(nil))
	return endpoint + "?" + canonicalQuery(params), nil
}

func canonicalQuery(params map[string]string) string {
	keys := make([]string, 0, len(params))
	for key := range params {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	parts := make([]string, 0, len(keys))
	for _, key := range keys {
		parts = append(parts, url.QueryEscape(key)+"="+url.QueryEscape(params[key]))
	}
	return strings.Join(parts, "&")
}

func parseASTPacket(payload []byte, fallback *atomic.Int64) (ASTSegment, bool, error) {
	var root map[string]any
	if err := json.Unmarshal(payload, &root); err != nil {
		return ASTSegment{}, false, errors.New("invalid AST JSON response")
	}
	if strings.EqualFold(stringValue(root["action"]), "error") {
		return ASTSegment{}, false, fmt.Errorf("AST business failure: code=%s", stringValue(root["code"]))
	}
	st := extractST(root)
	data := objectValue(root["data"])
	id := intValue(data["seg_id"])
	if id == 0 {
		id = intValue(st["seg_id"])
	}
	if id == 0 {
		id = intValue(st["sn"])
	}
	if id == 0 {
		id = int(fallback.Add(1))
	} else {
		for {
			old := fallback.Load()
			if int64(id) <= old || fallback.CompareAndSwap(old, int64(id)) {
				break
			}
		}
	}
	var text strings.Builder
	for _, rt := range arrayValue(st["rt"]) {
		for _, ws := range arrayValue(objectValue(rt)["ws"]) {
			cw := arrayValue(objectValue(ws)["cw"])
			if len(cw) > 0 {
				text.WriteString(stringValue(objectValue(cw[0])["w"]))
			}
		}
	}
	segment := ASTSegment{ID: id, Text: text.String(), PGS: stringValue(st["pgs"]), Finalized: boolValue(data["ls"]) || boolValue(st["ls"])}
	rg := arrayValue(st["rg"])
	if len(rg) >= 2 {
		segment.Range = []int{intValue(rg[0]), intValue(rg[1])}
		if segment.Range[0] > segment.Range[1] {
			segment.Range[0], segment.Range[1] = segment.Range[1], segment.Range[0]
		}
	}
	if _, ok := st["bg"]; ok {
		v := intValue(st["bg"])
		segment.Begin = &v
	}
	if _, ok := st["ed"]; ok {
		v := intValue(st["ed"])
		segment.End = &v
	}
	return segment, segment.Finalized, nil
}

func extractST(root map[string]any) map[string]any {
	data := objectValue(root["data"])
	if st := objectValue(objectValue(data["cn"])["st"]); len(st) > 0 {
		return st
	}
	if st := objectValue(data["st"]); len(st) > 0 {
		return st
	}
	if st := objectValue(objectValue(root["cn"])["st"]); len(st) > 0 {
		return st
	}
	return objectValue(root["st"])
}
func objectValue(v any) map[string]any {
	result, _ := v.(map[string]any)
	if result == nil {
		return map[string]any{}
	}
	return result
}
func arrayValue(v any) []any { result, _ := v.([]any); return result }
func stringValue(v any) string {
	if v == nil {
		return ""
	}
	return fmt.Sprint(v)
}
func intValue(v any) int {
	switch n := v.(type) {
	case float64:
		return int(n)
	case int:
		return n
	case json.Number:
		i, _ := strconv.Atoi(n.String())
		return i
	default:
		i, _ := strconv.Atoi(fmt.Sprint(v))
		return i
	}
}
func boolValue(v any) bool { result, _ := v.(bool); return result }
func boolInt(v bool) int {
	if v {
		return 2
	}
	return 1
}
