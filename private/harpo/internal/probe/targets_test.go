package probe

import (
	"net/http"
	"path/filepath"
	"testing"

	"harpo/internal/config"
)

func TestTargetsPersistenceAndDefaults(t *testing.T) {
	td := t.TempDir()
	p := config.Paths{Home: td, CfgDir: filepath.Join(td, "harpo")}

	defs := LoadTargets(p)
	if len(defs) < 5 {
		t.Fatalf("expected default AI/streaming presets, got %d", len(defs))
	}

	custom := append(defs, DomainTarget{Name: "Example", URL: NormalizeTargetURL("example.com")})
	if custom[len(custom)-1].URL != "https://example.com" {
		t.Fatalf("unexpected normalized URL: %s", custom[len(custom)-1].URL)
	}
	if err := SaveTargets(p, custom); err != nil {
		t.Fatalf("SaveTargets: %v", err)
	}
	reloaded := LoadTargets(p)
	if len(reloaded) != len(custom) || reloaded[len(reloaded)-1].Name != "Example" {
		t.Fatalf("reloaded targets mismatch: %+v", reloaded)
	}
}

func TestClassifyHTTPResponse(t *testing.T) {
	h := http.Header{}
	if v, ok := classifyHTTPResponse(200, h, "hello"); !ok || v != "OK" {
		t.Fatalf("200 -> %q %v", v, ok)
	}
	if v, ok := classifyHTTPResponse(404, h, "{}"); !ok || v != "Reachable" {
		t.Fatalf("404 API root -> %q %v", v, ok)
	}
	if v, ok := classifyHTTPResponse(403, h, "Sorry, not available in your country"); ok || v != "Geo-Blocked" {
		t.Fatalf("403 geo -> %q %v", v, ok)
	}
	cfh := http.Header{"Cf-Mitigated": []string{"challenge"}}
	if v, ok := classifyHTTPResponse(403, cfh, "Just a moment..."); ok || v != "CF Challenge" {
		t.Fatalf("CF -> %q %v", v, ok)
	}
}
