package media

import (
	"context"
	"github.com/stretchr/testify/require"
	"net"
	"testing"
)

type staticResolver []net.IP

func (r staticResolver) LookupIPAddr(context.Context, string) ([]net.IPAddr, error) {
	result := make([]net.IPAddr, len(r))
	for i, ip := range r {
		result[i] = net.IPAddr{IP: ip}
	}
	return result, nil
}
func TestSecureAudioProxyRejectsUnsafeTargets(t *testing.T) {
	for _, raw := range []string{"http://example.com/a.mp3", "https://user:pass@example.com/a.mp3"} {
		_, _, err := (&SecureAudioProxy{Resolver: staticResolver{net.ParseIP("8.8.8.8")}}).validate(context.Background(), raw)
		require.Error(t, err)
	}
	_, _, err := (&SecureAudioProxy{Resolver: staticResolver{net.ParseIP("127.0.0.1")}}).validate(context.Background(), "https://example.com/a.mp3")
	require.Error(t, err)
}
func TestSecureAudioProxyAcceptsPublicHTTPS(t *testing.T) {
	parsed, ips, err := (&SecureAudioProxy{Resolver: staticResolver{net.ParseIP("8.8.8.8")}}).validate(context.Background(), "https://audio.example.com/a.mp3")
	require.NoError(t, err)
	require.Equal(t, "audio.example.com", parsed.Hostname())
	require.Len(t, ips, 1)
}

func TestSecureAudioProxyIgnoresUnsafeAddressWhenSafeAddressExists(t *testing.T) {
	resolver := staticResolver{net.ParseIP("198.18.0.88"), net.ParseIP("fdfe:dcba:9876::d7")}
	_, ips, err := (&SecureAudioProxy{Resolver: resolver}).validate(context.Background(), "https://sgw-dx.xf-yun.com/a.mp3")
	require.NoError(t, err)
	require.Equal(t, []net.IP{net.ParseIP("198.18.0.88")}, ips)
}
