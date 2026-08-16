package media

import (
	"encoding/json"
	"net/url"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestASTSignedURLIsDeterministicAndDoesNotExposeSecret(t *testing.T) {
	client := XunfeiASTClient{AppID: "app", APIKey: "key", APISecret: "secret", Now: func() time.Time { return time.Date(2026, 8, 16, 12, 0, 0, 0, time.FixedZone("CST", 8*3600)) }}
	raw, err := client.signedURL("session", "uuid-value")
	require.NoError(t, err)
	parsed, err := url.Parse(raw)
	require.NoError(t, err)
	require.Equal(t, "app", parsed.Query().Get("appId"))
	require.Equal(t, "key", parsed.Query().Get("accessKeyId"))
	require.NotEmpty(t, parsed.Query().Get("signature"))
	require.NotContains(t, raw, "secret")
}

func TestParseASTPacketSupportsNestedLayoutAndRange(t *testing.T) {
	payload := map[string]any{"data": map[string]any{"seg_id": 8, "ls": true, "cn": map[string]any{"st": map[string]any{"pgs": "rpl", "rg": []int{5, 3}, "bg": 10, "ed": 20, "rt": []any{map[string]any{"ws": []any{map[string]any{"cw": []any{map[string]any{"w": "你好"}}}}}}}}}}
	raw, _ := json.Marshal(payload)
	var fallback atomic.Int64
	segment, finalPacket, err := parseASTPacket(raw, &fallback)
	require.NoError(t, err)
	require.True(t, finalPacket)
	require.Equal(t, 8, segment.ID)
	require.Equal(t, "你好", segment.Text)
	require.Equal(t, []int{3, 5}, segment.Range)
}

func TestParseASTPacketRejectsBusinessErrorWithoutLeakingPayload(t *testing.T) {
	var fallback atomic.Int64
	_, _, err := parseASTPacket([]byte(`{"action":"error","code":"101","desc":"sensitive transcript"}`), &fallback)
	require.Error(t, err)
	require.NotContains(t, err.Error(), "sensitive transcript")
}
