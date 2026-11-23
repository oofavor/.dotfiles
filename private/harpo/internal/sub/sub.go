package sub

import (
	"context"
	"encoding/base64"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"strings"
	"time"

	"harpo/internal/config"
	"harpo/internal/xray"
)

// DecodeBody decodes a plain-text or base64-encoded subscription payload into URI lines.
func DecodeBody(raw string) []string {
	text := strings.TrimSpace(raw)
	if text == "" {
		return nil
	}
	if strings.Contains(text, "vless://") ||
		strings.Contains(text, "vmess://") ||
		strings.Contains(text, "trojan://") ||
		strings.Contains(text, "hysteria") {
		return nonEmptyLines(text)
	}

	compact := strings.Join(strings.Fields(text), "")
	// Normalize padding for standard/URL-safe encodings.
	if rem := len(compact) % 4; rem != 0 {
		compact += strings.Repeat("=", 4-rem)
	}
	decoders := []*base64.Encoding{
		base64.StdEncoding,
		base64.URLEncoding,
		base64.RawStdEncoding,
		base64.RawURLEncoding,
	}
	for _, enc := range decoders {
		in := compact
		if enc == base64.RawStdEncoding || enc == base64.RawURLEncoding {
			in = strings.TrimRight(compact, "=")
		}
		decoded, err := enc.DecodeString(in)
		if err != nil {
			continue
		}
		s := string(decoded)
		if strings.Contains(s, "://") {
			return nonEmptyLines(s)
		}
	}
	return nil
}

func nonEmptyLines(s string) []string {
	rawLines := strings.Split(s, "\n")
	out := make([]string, 0, len(rawLines))
	for _, line := range rawLines {
		if l := strings.TrimSpace(line); l != "" {
			out = append(out, l)
		}
	}
	return out
}

// Fetch downloads a subscription URL (trying direct first, then any listening local HTTP proxy)
// and parses all VLESS profiles, returning the parsed profiles and count of skipped entries.
func Fetch(ctx context.Context, subURL string) ([]xray.Profile, int, error) {
	proxies := []string{""}
	for _, port := range []int{config.HTTPPort, 2080, 7897, 7890} {
		c, err := net.DialTimeout("tcp4", fmt.Sprintf("127.0.0.1:%d", port), 200*time.Millisecond)
		if err == nil {
			_ = c.Close()
			proxies = append(proxies, fmt.Sprintf("http://127.0.0.1:%d", port))
		}
	}

	var body string
	var lastErr error
	for _, proxyStr := range proxies {
		tr := &http.Transport{
			DialContext: (&net.Dialer{Timeout: 8 * time.Second}).DialContext,
		}
		if proxyStr != "" {
			pURL, err := url.Parse(proxyStr)
			if err == nil {
				tr.Proxy = http.ProxyURL(pURL)
			}
		} else {
			tr.Proxy = nil // Explicit direct fetch, ignoring env proxies.
		}
		client := &http.Client{
			Transport: tr,
			Timeout:   15 * time.Second,
		}
		req, err := http.NewRequestWithContext(ctx, http.MethodGet, subURL, nil)
		if err != nil {
			return nil, 0, fmt.Errorf("invalid subscription URL: %w", err)
		}
		req.Header.Set("User-Agent", "v2rayN/6.45")

		resp, err := client.Do(req)
		if err != nil {
			lastErr = err
			tr.CloseIdleConnections()
			continue
		}
		b, readErr := io.ReadAll(io.LimitReader(resp.Body, 4<<20))
		_ = resp.Body.Close()
		tr.CloseIdleConnections()

		if readErr != nil {
			lastErr = readErr
			continue
		}
		if resp.StatusCode < 200 || resp.StatusCode >= 300 {
			lastErr = fmt.Errorf("HTTP %d", resp.StatusCode)
			continue
		}
		if s := strings.TrimSpace(string(b)); s != "" {
			body = s
			break
		}
		lastErr = fmt.Errorf("empty response body")
	}

	if body == "" {
		return nil, 0, fmt.Errorf("failed to fetch subscription: %v", lastErr)
	}

	lines := DecodeBody(body)
	profiles := make([]xray.Profile, 0, len(lines))
	skipped := 0
	for _, line := range lines {
		if strings.HasPrefix(line, "vless://") {
			p, err := xray.ParseVLESSURI(line)
			if err != nil {
				skipped++
				continue
			}
			p.Source = subURL
			profiles = append(profiles, *p)
		} else if strings.Contains(line, "://") {
			skipped++
		}
	}
	return profiles, skipped, nil
}
