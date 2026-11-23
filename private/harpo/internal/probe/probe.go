package probe

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"

	xproxy "golang.org/x/net/proxy"

	"harpo/internal/xray"
)

// Result holds telemetry returned by probing an HTTP proxy port.
type Result struct {
	IP          string
	Country     string
	City        string
	Org         string
	Timezone    string
	AnthropicMS int // 0 if unreachable or not tested
	Err         error
}

// Location formats city and country cleanly.
func (r Result) Location() string {
	parts := make([]string, 0, 2)
	if r.City != "" {
		parts = append(parts, r.City)
	}
	if r.Country != "" {
		parts = append(parts, r.Country)
	}
	return strings.Join(parts, ", ")
}

// HTTPClientForProxy builds an http.Client routed strictly through http://127.0.0.1:<port>.
func HTTPClientForProxy(httpPort int, timeout time.Duration) (*http.Client, *http.Transport) {
	proxyURL, _ := url.Parse(fmt.Sprintf("http://127.0.0.1:%d", httpPort))
	tr := &http.Transport{
		Proxy:                 http.ProxyURL(proxyURL),
		DialContext:           (&net.Dialer{Timeout: timeout}).DialContext,
		TLSHandshakeTimeout:   timeout,
		ResponseHeaderTimeout: timeout,
		ForceAttemptHTTP2:     true,
	}
	return &http.Client{Transport: tr, Timeout: timeout}, tr
}

// HTTPClientForSOCKS5 builds an http.Client routed through socks5h://127.0.0.1:<port> (remote DNS).
func HTTPClientForSOCKS5(socksPort int, timeout time.Duration) (*http.Client, *http.Transport, error) {
	dialer, err := xproxy.SOCKS5("tcp", fmt.Sprintf("127.0.0.1:%d", socksPort), nil, &net.Dialer{Timeout: timeout})
	if err != nil {
		return nil, nil, err
	}
	tr := &http.Transport{
		Proxy: nil,
		DialContext: func(ctx context.Context, network, addr string) (net.Conn, error) {
			if cd, ok := dialer.(xproxy.ContextDialer); ok {
				return cd.DialContext(ctx, network, addr)
			}
			return dialer.Dial(network, addr)
		},
		TLSHandshakeTimeout:   timeout,
		ResponseHeaderTimeout: timeout,
		ForceAttemptHTTP2:     true,
	}
	return &http.Client{Transport: tr, Timeout: timeout}, tr, nil
}

// ProbeHTTPPort queries exit IP, geolocation, timezone, and Anthropic API RTT via a local HTTP proxy.
func ProbeHTTPPort(ctx context.Context, httpPort int, checkAnthropic bool) Result {
	client, tr := HTTPClientForProxy(httpPort, 6*time.Second)
	defer tr.CloseIdleConnections()

	var res Result
	for _, geoURL := range []string{"https://ipinfo.io/json", "https://ipapi.co/json"} {
		req, err := http.NewRequestWithContext(ctx, http.MethodGet, geoURL, nil)
		if err != nil {
			continue
		}
		req.Header.Set("User-Agent", "harpo/1.0")
		resp, err := client.Do(req)
		if err != nil {
			continue
		}
		body, readErr := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
		_ = resp.Body.Close()
		if readErr != nil || resp.StatusCode != http.StatusOK {
			continue
		}
		var data map[string]any
		if err := json.Unmarshal(body, &data); err != nil {
			continue
		}
		ip := strings.TrimSpace(strVal(data["ip"]))
		if ip != "" {
			res.IP = ip
			res.Country = strings.TrimSpace(firstNonEmpty(strVal(data["country"]), strVal(data["country_code"])))
			res.City = strings.TrimSpace(strVal(data["city"]))
			res.Org = strings.TrimSpace(strVal(data["org"]))
			res.Timezone = strings.TrimSpace(strVal(data["timezone"]))
			break
		}
	}

	if res.IP == "" {
		req, err := http.NewRequestWithContext(ctx, http.MethodGet, "https://api.ipify.org", nil)
		if err == nil {
			if resp, err := client.Do(req); err == nil {
				body, _ := io.ReadAll(io.LimitReader(resp.Body, 256))
				_ = resp.Body.Close()
				if resp.StatusCode == http.StatusOK {
					res.IP = strings.TrimSpace(string(body))
				}
			}
		}
	}

	if res.IP != "" && checkAnthropic {
		t0 := time.Now()
		req, err := http.NewRequestWithContext(ctx, http.MethodGet, "https://api.anthropic.com/", nil)
		if err == nil {
			if resp, err := client.Do(req); err == nil {
				_, _ = io.Copy(io.Discard, io.LimitReader(resp.Body, 4096))
				_ = resp.Body.Close()
				ms := int(time.Since(t0).Milliseconds())
				if ms <= 0 {
					ms = 1
				}
				res.AnthropicMS = ms
			}
		}
	}

	return res
}

func strVal(v any) string {
	if v == nil {
		return ""
	}
	s, ok := v.(string)
	if ok {
		return s
	}
	return fmt.Sprint(v)
}

func firstNonEmpty(vals ...string) string {
	for _, v := range vals {
		if strings.TrimSpace(v) != "" {
			return v
		}
	}
	return ""
}

// ValidateAndProbeCandidate starts an ephemeral Xray instance for outbound and probes its live exit.
func ValidateAndProbeCandidate(ctx context.Context, xrayBin string, outbound map[string]any) (Result, error) {
	inst, err := xray.StartTemp(ctx, xrayBin, outbound)
	if err != nil {
		return Result{}, err
	}
	defer inst.Close()

	res := ProbeHTTPPort(ctx, inst.HTTPPort, false)
	if res.IP == "" {
		return res, fmt.Errorf("connected to xray, but live exit-IP probe failed (proxy unreachable or blocked)")
	}
	return res, nil
}

// BatchResult holds an individual profile probe result during concurrent batch probing.
type BatchResult struct {
	Index  int
	Result Result
	Err    error
}

// ProbeProfilesConcurrent tests multiple candidate profiles concurrently with bounded parallelism.
func ProbeProfilesConcurrent(ctx context.Context, xrayBin string, profiles []xray.Profile, maxWorkers int, onResult func(BatchResult)) {
	if maxWorkers <= 0 {
		maxWorkers = 4
	}
	sem := make(chan struct{}, maxWorkers)
	var wg sync.WaitGroup
	for i := range profiles {
		if ctx.Err() != nil {
			break
		}
		wg.Add(1)
		sem <- struct{}{}
		go func(idx int, ob map[string]any) {
			defer wg.Done()
			defer func() { <-sem }()
			pCtx, cancel := context.WithTimeout(ctx, 12*time.Second)
			defer cancel()
			res, err := ValidateAndProbeCandidate(pCtx, xrayBin, ob)
			if onResult != nil {
				onResult(BatchResult{Index: idx, Result: res, Err: err})
			}
		}(i, profiles[i].Outbound)
	}
	wg.Wait()
}

// KillSwitchStatus describes the live OS kill-switch check (cgroup v2 on Linux, Windows Firewall on Windows).
type KillSwitchStatus struct {
	Blocked bool
	Detail  string
	Err     error
}

// ListeningSocket represents a local listening TCP socket.
type ListeningSocket struct {
	Addr string
	Port int
	IPv6 bool
}

// IsPortListening checks if 127.0.0.1:<port> is listening using OS socket table with TCP dial fallback.
func IsPortListening(port int) bool {
	if socks, err := ParseProcNetTCP(); err == nil {
		for _, s := range socks {
			if s.Port == port && (s.Addr == "127.0.0.1" || s.Addr == "0.0.0.0" || s.Addr == "::1" || s.Addr == "::") {
				return true
			}
		}
	}
	c, err := net.DialTimeout("tcp4", fmt.Sprintf("127.0.0.1:%d", port), 150*time.Millisecond)
	if err == nil {
		_ = c.Close()
		return true
	}
	return false
}
