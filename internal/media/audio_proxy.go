package media

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"strings"
	"time"
)

type HostResolver interface {
	LookupIPAddr(context.Context, string) ([]net.IPAddr, error)
}
type AudioProxy interface {
	Open(context.Context, string, string) (*http.Response, error)
}
type SecureAudioProxy struct {
	Resolver HostResolver
	Dialer   *net.Dialer
}

func (p *SecureAudioProxy) Open(ctx context.Context, rawURL, rangeHeader string) (*http.Response, error) {
	parsed, ips, err := p.validate(ctx, rawURL)
	if err != nil {
		return nil, err
	}
	dialer := p.Dialer
	if dialer == nil {
		dialer = &net.Dialer{Timeout: 10 * time.Second}
	}
	port := parsed.Port()
	if port == "" {
		port = "443"
	}
	target := net.JoinHostPort(ips[0].String(), port)
	transport := &http.Transport{Proxy: nil, DialContext: func(ctx context.Context, network, address string) (net.Conn, error) {
		return dialer.DialContext(ctx, network, target)
	}, TLSHandshakeTimeout: 10 * time.Second, ResponseHeaderTimeout: 30 * time.Second}
	client := &http.Client{Transport: transport, CheckRedirect: func(_ *http.Request, _ []*http.Request) error { return errors.New("audio redirects are not allowed") }}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, parsed.String(), nil)
	if err != nil {
		return nil, err
	}
	if strings.TrimSpace(rangeHeader) != "" {
		req.Header.Set("Range", rangeHeader)
	}
	resp, err := client.Do(req)
	if err != nil {
		transport.CloseIdleConnections()
		return nil, err
	}
	resp.Body = &closingBody{ReadCloser: resp.Body, closeTransport: transport.CloseIdleConnections}
	return resp, nil
}
func (p *SecureAudioProxy) validate(ctx context.Context, rawURL string) (*url.URL, []net.IP, error) {
	parsed, err := url.Parse(strings.TrimSpace(rawURL))
	if err != nil || parsed.Scheme != "https" || parsed.Hostname() == "" || parsed.User != nil {
		return nil, nil, errors.New("unsafe TTS audio URL")
	}
	resolver := p.Resolver
	if resolver == nil {
		resolver = net.DefaultResolver
	}
	addresses, err := resolver.LookupIPAddr(ctx, parsed.Hostname())
	if err != nil || len(addresses) == 0 {
		return nil, nil, errors.New("unable to resolve TTS audio host")
	}
	ips := make([]net.IP, 0, len(addresses))
	for _, address := range addresses {
		if unsafeIP(address.IP) {
			continue
		}
		ips = append(ips, address.IP)
	}
	if len(ips) == 0 {
		return nil, nil, fmt.Errorf("unsafe TTS audio host")
	}
	return parsed, ips, nil
}

type closingBody struct {
	io.ReadCloser
	closeTransport func()
}

func (b *closingBody) Close() error { err := b.ReadCloser.Close(); b.closeTransport(); return err }
func unsafeIP(ip net.IP) bool {
	return ip == nil || ip.IsLoopback() || ip.IsPrivate() || ip.IsUnspecified() || ip.IsLinkLocalUnicast() || ip.IsLinkLocalMulticast() || ip.IsMulticast()
}
