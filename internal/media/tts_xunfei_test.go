package media

import (
	"encoding/base64"
	"github.com/stretchr/testify/require"
	"net/url"
	"testing"
	"time"
)

func TestXunfeiTTSSigningIsDeterministic(t *testing.T) {
	client := XunfeiTTSClient{APIKey: "key", APISecret: "secret", Now: func() time.Time { return time.Date(2026, 8, 16, 0, 0, 0, 0, time.UTC) }}
	raw, date, auth, err := client.signedURL("/v1/private/dts_create")
	require.NoError(t, err)
	require.Equal(t, "Sun, 16 Aug 2026 00:00:00 GMT", date)
	require.NotEmpty(t, auth)
	parsed, _ := url.Parse(raw)
	require.Equal(t, auth, parsed.Query().Get("authorization"))
	require.NotContains(t, raw, "secret")
}
func TestDecodeBase64AudioURL(t *testing.T) {
	expected := "https://example.com/audio.mp3"
	require.Equal(t, expected, decodeBase64URL(base64.StdEncoding.EncodeToString([]byte(expected))))
}

func TestDecodeBase64AudioURLUpgradesTrustedXunfeiHost(t *testing.T) {
	raw := "http://sgw-dx.xf-yun.com/audio/file.mp3?token=signed"
	require.Equal(t, "https://sgw-dx.xf-yun.com/audio/file.mp3?token=signed", decodeBase64URL(base64.StdEncoding.EncodeToString([]byte(raw))))
}

func TestDecodeBase64AudioURLDoesNotUpgradeUntrustedHost(t *testing.T) {
	raw := "http://example.com/audio/file.mp3"
	require.Equal(t, raw, decodeBase64URL(base64.StdEncoding.EncodeToString([]byte(raw))))
}
