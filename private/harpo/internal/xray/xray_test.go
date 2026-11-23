package xray

import (
	"encoding/json"
	"testing"
)

func TestParseVLESSURIValidVariants(t *testing.T) {
	cases := []struct {
		name     string
		uri      string
		wantName string
		wantNet  string
		wantSec  string
		wantHost string
		wantPort int
	}{
		{
			name:     "reality tcp vision unicode",
			uri:      "vless://11111111-2222-3333-4444-555555555555@198.51.100.1:2053?type=tcp&security=reality&pbk=AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA&fp=firefox&sni=example.com&sid=aabbccdd&spx=%2Ftest&flow=xtls-rprx-vision#%F0%9F%A4%96%20Claude%20%E2%80%94%20Main",
			wantName: "🤖 Claude — Main",
			wantNet:  "raw",
			wantSec:  "reality",
			wantHost: "198.51.100.1",
			wantPort: 2053,
		},
		{
			name:     "tls ws alpn",
			uri:      "vless://11111111-2222-3333-4444-555555555555@example.com:443?type=ws&security=tls&sni=example.com&host=ws.example.com&path=%2Fws&alpn=h2,http/1.1&allowInsecure=1#WS-TLS",
			wantName: "WS-TLS",
			wantNet:  "ws",
			wantSec:  "tls",
			wantHost: "example.com",
			wantPort: 443,
		},
		{
			name:     "splithttp normalized to xhttp",
			uri:      "vless://11111111-2222-3333-4444-555555555555@example.com?type=splithttp&security=tls&path=%2Fxhttp&host=cdn.example.com#SplitHTTP",
			wantName: "SplitHTTP",
			wantNet:  "xhttp",
			wantSec:  "tls",
			wantHost: "example.com",
			wantPort: 443,
		},
		{
			name:     "ipv6 httpupgrade default port",
			uri:      "vless://11111111-2222-3333-4444-555555555555@[2001:db8::1]?type=httpupgrade&security=tls&sni=v6.example.com&path=%2Fup#IPv6-Up",
			wantName: "IPv6-Up",
			wantNet:  "httpupgrade",
			wantSec:  "tls",
			wantHost: "2001:db8::1",
			wantPort: 443,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			p, err := ParseVLESSURI(tc.uri)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if p.Name != tc.wantName || p.Network != tc.wantNet || p.Security != tc.wantSec || p.Host != tc.wantHost || p.Port != tc.wantPort {
				t.Fatalf("got %+v, want name=%q net=%q sec=%q host=%q port=%d", p, tc.wantName, tc.wantNet, tc.wantSec, tc.wantHost, tc.wantPort)
			}
		})
	}
}

func TestParseVLESSURIMalformed(t *testing.T) {
	bad := []string{
		"vmess://abc",
		"vless://@1.2.3.4:443#no-uuid",
		"vless://11111111-2222-3333-4444-555555555555@:443#no-host",
		"vless://11111111-2222-3333-4444-555555555555@1.2.3.4:443?security=reality#no-pbk",
	}
	for _, uri := range bad {
		if _, err := ParseVLESSURI(uri); err == nil {
			t.Errorf("expected error for malformed URI %q", uri)
		}
	}
}

func TestBuildConfigNonMutating(t *testing.T) {
	orig := map[string]any{
		"protocol": "vless",
		"tag":      "orig",
		"settings": map[string]any{"address": "1.2.3.4", "port": 443, "id": "u"},
	}
	before, _ := json.Marshal(orig)
	cfg := BuildConfig(orig, 10808, 10809)
	after, _ := json.Marshal(orig)
	if string(before) != string(after) {
		t.Fatalf("BuildConfig mutated input: before=%s after=%s", before, after)
	}
	obs := cfg["outbounds"].([]any)
	if len(obs) != 1 || obs[0].(map[string]any)["tag"] != "harpo-exit" {
		t.Fatalf("unexpected outbounds: %+v", obs)
	}
}
