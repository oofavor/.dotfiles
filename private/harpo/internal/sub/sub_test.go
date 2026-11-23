package sub

import (
	"context"
	"encoding/base64"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestDecodeBodyVariants(t *testing.T) {
	sample := "vless://11111111-2222-3333-4444-555555555555@1.2.3.4:443#A\nhysteria2://x@1.2.3.4:443#B\n"
	if got := DecodeBody(sample); len(got) != 2 {
		t.Fatalf("plain: expected 2, got %d", len(got))
	}

	b64 := strings.TrimRight(base64.StdEncoding.EncodeToString([]byte(sample)), "=")
	var wrapped strings.Builder
	for i := 0; i < len(b64); i += 20 {
		end := i + 20
		if end > len(b64) {
			end = len(b64)
		}
		wrapped.WriteString(b64[i:end] + "\n")
	}
	if got := DecodeBody(wrapped.String()); len(got) != 2 {
		t.Fatalf("wrapped b64: expected 2, got %d", len(got))
	}

	ub64 := strings.TrimRight(base64.URLEncoding.EncodeToString([]byte(sample)), "=")
	if got := DecodeBody(ub64); len(got) != 2 {
		t.Fatalf("urlsafe b64: expected 2, got %d", len(got))
	}

	if got := DecodeBody("<html>404 Not Found</html>"); len(got) != 0 {
		t.Fatalf("expected 0 for html, got %d", len(got))
	}
}

func TestFetchMockServer(t *testing.T) {
	sample := "vless://11111111-2222-3333-4444-555555555555@198.51.100.1:443?security=tls&sni=a.test#One\n" +
		"vless://11111111-2222-3333-4444-555555555555@198.51.100.2:443?security=reality#MissingPBK\n" +
		"hysteria2://pw@198.51.100.3:443#SkipHysteria\n"
	payload := base64.StdEncoding.EncodeToString([]byte(sample))

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(payload))
	}))
	defer srv.Close()

	profs, skipped, err := Fetch(context.Background(), srv.URL)
	if err != nil {
		t.Fatalf("Fetch failed: %v", err)
	}
	if len(profs) != 1 || profs[0].Name != "One" || skipped != 2 {
		t.Fatalf("got %d profs, skipped=%d", len(profs), skipped)
	}
}
