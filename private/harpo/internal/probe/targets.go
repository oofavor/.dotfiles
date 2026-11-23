package probe

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"harpo/internal/config"
)

// DomainTarget is a user-configurable probe destination saved in ~/.config/harpo/targets.json.
type DomainTarget struct {
	Name string `json:"name"`
	URL  string `json:"url"`
}

// DomainProbeResult holds the result of an explicit on-demand probe for one DomainTarget.
type DomainProbeResult struct {
	StatusCode int
	Proto      string // e.g. "h2", "h1.1"
	RTTMS      int
	Verdict    string // "OK", "Reachable", "CF Challenge", "Blocked", "Timeout", "Error"
	Healthy    bool
	PortMode   string // "http" or "socks5"
	Err        error
}

// DefaultTargets returns the AI & Streaming presets seeded when targets.json does not exist yet.
func DefaultTargets() []DomainTarget {
	return []DomainTarget{
		{Name: "Anthropic", URL: "https://api.anthropic.com/"},
		{Name: "Google Gemini", URL: "https://generativelanguage.googleapis.com/"},
		{Name: "OpenAI", URL: "https://api.openai.com/"},
		{Name: "YouTube", URL: "https://www.youtube.com/"},
		{Name: "GitHub", URL: "https://api.github.com/"},
		{Name: "Cloudflare", URL: "https://1.1.1.1/cdn-cgi/trace"},
	}
}

// TargetsPath returns ~/.config/harpo/targets.json.
func TargetsPath(p config.Paths) string {
	return filepath.Join(p.CfgDir, "targets.json")
}

// LoadTargets loads targets.json, falling back to DefaultTargets if not yet created.
func LoadTargets(p config.Paths) []DomainTarget {
	b, err := os.ReadFile(TargetsPath(p))
	if err != nil {
		return DefaultTargets()
	}
	var list []DomainTarget
	if err := json.Unmarshal(b, &list); err != nil {
		return DefaultTargets()
	}
	out := make([]DomainTarget, 0, len(list))
	for _, t := range list {
		if strings.TrimSpace(t.URL) != "" {
			if strings.TrimSpace(t.Name) == "" {
				t.Name = t.URL
			}
			out = append(out, t)
		}
	}
	return out
}

// SaveTargets saves targets.json atomically with mode 0600.
func SaveTargets(p config.Paths, list []DomainTarget) error {
	if list == nil {
		list = []DomainTarget{}
	}
	b, err := json.MarshalIndent(list, "", "  ")
	if err != nil {
		return err
	}
	return config.WriteSecretText(TargetsPath(p), string(b)+"\n")
}

// NormalizeTargetURL ensures a user-typed domain or URL has an https:// scheme.
func NormalizeTargetURL(raw string) string {
	s := strings.TrimSpace(raw)
	if s == "" {
		return ""
	}
	if !strings.Contains(s, "://") {
		s = "https://" + s
	}
	return s
}

// ProbeDomain performs an explicit on-demand HTTPS GET through either HTTP (:10809) or SOCKS5h (:10808).
func ProbeDomain(ctx context.Context, targetURL string, useSocks5 bool) DomainProbeResult {
	portMode := "http"
	var (
		client *http.Client
		tr     *http.Transport
		err    error
	)
	if useSocks5 {
		portMode = "socks5"
		client, tr, err = HTTPClientForSOCKS5(config.SocksPort, 8*time.Second)
		if err != nil {
			return DomainProbeResult{PortMode: portMode, Verdict: "Error", Err: err}
		}
	} else {
		client, tr = HTTPClientForProxy(config.HTTPPort, 8*time.Second)
	}
	defer tr.CloseIdleConnections()
	// Don't follow endless redirects when checking reachability.
	client.CheckRedirect = func(*http.Request, []*http.Request) error {
		return http.ErrUseLastResponse
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, targetURL, nil)
	if err != nil {
		return DomainProbeResult{PortMode: portMode, Verdict: "Invalid URL", Err: err}
	}
	req.Header.Set("User-Agent", "Mozilla/5.0 (X11; Linux x86_64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/131.0.0.0 Safari/537.36")

	t0 := time.Now()
	resp, err := client.Do(req)
	rtt := int(time.Since(t0).Milliseconds())
	if rtt <= 0 {
		rtt = 1
	}
	if err != nil {
		verdict := "Unreachable"
		if strings.Contains(strings.ToLower(err.Error()), "timeout") || ctx.Err() == context.DeadlineExceeded {
			verdict = "Timeout"
		}
		return DomainProbeResult{PortMode: portMode, RTTMS: rtt, Verdict: verdict, Err: err}
	}
	bodyBytes, _ := io.ReadAll(io.LimitReader(resp.Body, 8192))
	_ = resp.Body.Close()

	proto := fmt.Sprintf("h%d.%d", resp.ProtoMajor, resp.ProtoMinor)
	if resp.ProtoMajor == 2 {
		proto = "h2"
	} else if resp.ProtoMajor == 3 {
		proto = "h3"
	}

	verdict, healthy := classifyHTTPResponse(resp.StatusCode, resp.Header, string(bodyBytes))
	return DomainProbeResult{
		StatusCode: resp.StatusCode,
		Proto:      proto,
		RTTMS:      rtt,
		Verdict:    verdict,
		Healthy:    healthy,
		PortMode:   portMode,
	}
}

func classifyHTTPResponse(code int, hdr http.Header, body string) (string, bool) {
	lowBody := strings.ToLower(body)
	cfMitigated := strings.EqualFold(hdr.Get("cf-mitigated"), "challenge") ||
		strings.Contains(lowBody, "just a moment") ||
		strings.Contains(lowBody, "cf-browser-verification")

	switch {
	case cfMitigated:
		return "CF Challenge", false
	case code >= 200 && code < 400:
		return "OK", true
	case code == 401 || code == 404 || code == 405:
		// API root endpoints (Anthropic, OpenAI, Gemini) commonly return 401/404/405 when reachable.
		return "Reachable", true
	case code == 403:
		if strings.Contains(lowBody, "country") || strings.Contains(lowBody, "region") || strings.Contains(lowBody, "unavailable") || strings.Contains(lowBody, "not available") {
			return "Geo-Blocked", false
		}
		return "Forbidden (403)", false
	case code == 429:
		return "Rate-Limited", false
	case code >= 500:
		return fmt.Sprintf("Server %d", code), false
	default:
		return fmt.Sprintf("HTTP %d", code), false
	}
}

// DomainBatchEvent is emitted as each target finishes during concurrent probing.
type DomainBatchEvent struct {
	Index  int
	Result DomainProbeResult
}

// ProbeDomainsConcurrent probes all given targets in parallel (only called on explicit user request).
func ProbeDomainsConcurrent(ctx context.Context, targets []DomainTarget, useSocks5 bool, onResult func(DomainBatchEvent)) {
	var wg sync.WaitGroup
	sem := make(chan struct{}, 6)
	for i, t := range targets {
		wg.Add(1)
		sem <- struct{}{}
		go func(idx int, urlStr string) {
			defer wg.Done()
			defer func() { <-sem }()
			cCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
			defer cancel()
			res := ProbeDomain(cCtx, urlStr, useSocks5)
			if onResult != nil {
				onResult(DomainBatchEvent{Index: idx, Result: res})
			}
		}(i, t.URL)
	}
	wg.Wait()
}
