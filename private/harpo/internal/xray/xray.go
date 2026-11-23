package xray

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"net/url"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"time"

	"harpo/internal/config"
)

// Profile represents a parsed VLESS proxy profile ready to build into an Xray config.
type Profile struct {
	Name     string
	Host     string
	Port     int
	Security string
	Network  string
	SNI      string
	Flow     string
	Source   string
	Outbound map[string]any
}

var defaultSockopt = map[string]any{
	"tcpKeepAliveIdle":     15,
	"tcpKeepAliveInterval": 5,
	"tcpNoDelay":           true,
}

func cloneMap(in map[string]any) map[string]any {
	if in == nil {
		return map[string]any{}
	}
	b, err := json.Marshal(in)
	if err != nil {
		return map[string]any{}
	}
	var out map[string]any
	if err := json.Unmarshal(b, &out); err != nil || out == nil {
		return map[string]any{}
	}
	return out
}

// ExtractHostPort extracts address and port from either flat settings or vnext[0].
func ExtractHostPort(ob map[string]any) (string, int) {
	settings, _ := ob["settings"].(map[string]any)
	if settings == nil {
		return "", 0
	}
	if addr, ok := settings["address"]; ok {
		return fmt.Sprint(addr), toInt(settings["port"])
	}
	if vnext, ok := settings["vnext"].([]any); ok && len(vnext) > 0 {
		if first, ok := vnext[0].(map[string]any); ok {
			return fmt.Sprint(first["address"]), toInt(first["port"])
		}
	}
	return "", 0
}

func toInt(v any) int {
	switch x := v.(type) {
	case int:
		return x
	case int64:
		return int(x)
	case float64:
		return int(x)
	case json.Number:
		n, _ := x.Int64()
		return int(n)
	case string:
		n, _ := strconv.Atoi(x)
		return n
	default:
		return 0
	}
}

// SetOutboundHost returns a deep copy of ob with the server host replaced.
func SetOutboundHost(ob map[string]any, newHost string) map[string]any {
	cp := cloneMap(ob)
	settings, ok := cp["settings"].(map[string]any)
	if !ok || settings == nil {
		settings = map[string]any{}
		cp["settings"] = settings
	}
	if _, hasAddr := settings["address"]; hasAddr {
		settings["address"] = newHost
	} else if vnext, ok := settings["vnext"].([]any); ok && len(vnext) > 0 {
		if first, ok := vnext[0].(map[string]any); ok {
			first["address"] = newHost
		}
	}
	return cp
}

// BuildConfig builds a full loopback-only, single-outbound Xray config without mutating outbound.
func BuildConfig(outbound map[string]any, socksPort, httpPort int) map[string]any {
	if socksPort == 0 {
		socksPort = config.SocksPort
	}
	if httpPort == 0 {
		httpPort = config.HTTPPort
	}
	ob := cloneMap(outbound)
	ob["tag"] = config.OutboundTag

	ss, ok := ob["streamSettings"].(map[string]any)
	if !ok || ss == nil {
		ss = map[string]any{}
		ob["streamSettings"] = ss
	}
	so, ok := ss["sockopt"].(map[string]any)
	if !ok || so == nil {
		so = map[string]any{}
		ss["sockopt"] = so
	}
	for k, v := range defaultSockopt {
		if _, exists := so[k]; !exists {
			so[k] = v
		}
	}

	return map[string]any{
		"log": map[string]any{"loglevel": "warning"},
		"inbounds": []any{
			map[string]any{
				"tag":      "socks-in",
				"listen":   "127.0.0.1",
				"port":     socksPort,
				"protocol": "socks",
				"settings": map[string]any{"auth": "noauth", "udp": false},
			},
			map[string]any{
				"tag":      "http-in",
				"listen":   "127.0.0.1",
				"port":     httpPort,
				"protocol": "http",
				"settings": map[string]any{},
			},
		},
		"outbounds": []any{ob},
		"routing": map[string]any{
			"domainStrategy": "AsIs",
			"rules":          []any{},
		},
	}
}

// ParseVLESSURI parses a vless:// share URI into a Profile and Xray outbound map.
func ParseVLESSURI(raw string) (*Profile, error) {
	uri := strings.TrimSpace(raw)
	if !strings.HasPrefix(uri, "vless://") {
		return nil, errors.New("URI does not start with vless://")
	}
	u, err := url.Parse(uri)
	if err != nil {
		return nil, fmt.Errorf("invalid URI: %w", err)
	}
	uuid := ""
	if u.User != nil {
		uuid = u.User.Username()
	}
	host := u.Hostname()
	port := 443
	if p := u.Port(); p != "" {
		parsedPort, err := strconv.Atoi(p)
		if err != nil || parsedPort <= 0 || parsedPort > 65535 {
			return nil, fmt.Errorf("invalid port %q", p)
		}
		port = parsedPort
	}
	if uuid == "" || host == "" {
		return nil, errors.New("missing UUID or host in VLESS URI")
	}

	name := strings.TrimSpace(u.Fragment)
	if name == "" {
		name = fmt.Sprintf("%s:%d", host, port)
	}

	qs := u.Query()
	q := func(key, def string) string {
		if vals, ok := qs[key]; ok && len(vals) > 0 {
			return vals[0]
		}
		return def
	}

	netType := strings.ToLower(q("type", "tcp"))
	if netType == "" || netType == "tcp" {
		netType = "raw"
	}
	security := strings.ToLower(q("security", "none"))
	if security == "" {
		security = "none"
	}
	encryption := q("encryption", "none")
	if encryption == "" {
		encryption = "none"
	}
	flow := q("flow", "")

	sockoptCopy := cloneMap(defaultSockopt)
	streamSettings := map[string]any{
		"network":  netType,
		"security": security,
		"sockopt":  sockoptCopy,
	}

	switch security {
	case "reality":
		pbk := q("pbk", "")
		if pbk == "" {
			return nil, errors.New("Reality VLESS link is missing 'pbk' (public key)")
		}
		fp := q("fp", "firefox")
		if fp == "" {
			fp = "firefox"
		}
		spx := q("spx", "/")
		if spx == "" {
			spx = "/"
		}
		rs := map[string]any{
			"fingerprint": fp,
			"password":    pbk,
			"serverName":  q("sni", ""),
			"shortId":     q("sid", ""),
			"spiderX":     spx,
		}
		if pqv := q("pqv", ""); pqv != "" {
			rs["mldsa65Verify"] = pqv
		}
		streamSettings["realitySettings"] = rs
	case "tls":
		sni := q("sni", "")
		if sni == "" {
			sni = host
		}
		fp := q("fp", "firefox")
		if fp == "" {
			fp = "firefox"
		}
		ts := map[string]any{
			"serverName":  sni,
			"fingerprint": fp,
		}
		if alpnRaw := q("alpn", ""); alpnRaw != "" {
			var alpn []string
			for _, part := range strings.Split(alpnRaw, ",") {
				if p := strings.TrimSpace(part); p != "" {
					alpn = append(alpn, p)
				}
			}
			if len(alpn) > 0 {
				ts["alpn"] = alpn
			}
		}
		// Xray 26.11+ removed allowInsecure in favor of pcs/vcn; ignore legacy allowInsecure.
		if pcs := q("pcs", ""); pcs != "" {
			ts["pinnedPeerCertSha256"] = pcs
		}
		if vcn := q("vcn", ""); vcn != "" {
			ts["verifyPeerCertByName"] = vcn
		}
		streamSettings["tlsSettings"] = ts
	}

	switch netType {
	case "ws":
		wsPath := q("path", "/")
		if wsPath == "" {
			wsPath = "/"
		}
		wsCfg := map[string]any{"path": wsPath}
		if wsHost := q("host", ""); wsHost != "" {
			wsCfg["headers"] = map[string]any{"Host": wsHost}
		}
		streamSettings["wsSettings"] = wsCfg
	case "grpc":
		grpcCfg := map[string]any{"serviceName": q("serviceName", "")}
		if q("mode", "") == "multi" {
			grpcCfg["multiMode"] = true
		}
		streamSettings["grpcSettings"] = grpcCfg
	case "xhttp", "splithttp":
		streamSettings["network"] = "xhttp"
		netType = "xhttp"
		xPath := q("path", "/")
		if xPath == "" {
			xPath = "/"
		}
		xCfg := map[string]any{"path": xPath}
		if h := q("host", ""); h != "" {
			xCfg["host"] = h
		}
		if m := q("mode", ""); m != "" {
			xCfg["mode"] = m
		}
		streamSettings["xhttpSettings"] = xCfg
	case "httpupgrade":
		huPath := q("path", "/")
		if huPath == "" {
			huPath = "/"
		}
		huCfg := map[string]any{"path": huPath}
		if h := q("host", ""); h != "" {
			huCfg["host"] = h
		}
		streamSettings["httpupgradeSettings"] = huCfg
	}

	settings := map[string]any{
		"address":    host,
		"port":       port,
		"id":         uuid,
		"encryption": encryption,
	}
	if flow != "" {
		settings["flow"] = flow
	}

	outbound := map[string]any{
		"tag":            config.OutboundTag,
		"protocol":       "vless",
		"settings":       settings,
		"streamSettings": streamSettings,
	}

	sni := q("sni", "")
	if sni == "" {
		sni = "-"
	}
	flowDisp := flow
	if flowDisp == "" {
		flowDisp = "-"
	}

	return &Profile{
		Name:     name,
		Host:     host,
		Port:     port,
		Security: security,
		Network:  netType,
		SNI:      sni,
		Flow:     flowDisp,
		Outbound: outbound,
	}, nil
}

// ProfileFromOutbound constructs a Profile summary from a raw Xray outbound JSON object.
func ProfileFromOutbound(name, source string, ob map[string]any) Profile {
	host, port := ExtractHostPort(ob)
	if name == "" {
		name = fmt.Sprintf("%s:%d", host, port)
	}
	ss, _ := ob["streamSettings"].(map[string]any)
	sec, _ := ss["security"].(string)
	if sec == "" {
		sec = "none"
	}
	network, _ := ss["network"].(string)
	if network == "" {
		network = "raw"
	}
	sni := "-"
	if rs, ok := ss["realitySettings"].(map[string]any); ok {
		if s, _ := rs["serverName"].(string); s != "" {
			sni = s
		}
	} else if ts, ok := ss["tlsSettings"].(map[string]any); ok {
		if s, _ := ts["serverName"].(string); s != "" {
			sni = s
		}
	}
	flow := "-"
	if st, ok := ob["settings"].(map[string]any); ok {
		if f, _ := st["flow"].(string); f != "" {
			flow = f
		}
	}
	return Profile{
		Name:     name,
		Host:     host,
		Port:     port,
		Security: sec,
		Network:  network,
		SNI:      sni,
		Flow:     flow,
		Source:   source,
		Outbound: ob,
	}
}

// FindFreePort allocates an available ephemeral TCP port on 127.0.0.1.
func FindFreePort() (int, error) {
	ln, err := net.Listen("tcp4", "127.0.0.1:0")
	if err != nil {
		return 0, err
	}
	defer ln.Close()
	return ln.Addr().(*net.TCPAddr).Port, nil
}

// ResolveIPv4 resolves a hostname to its first IPv4 address.
func ResolveIPv4(ctx context.Context, host string) (string, error) {
	r := &net.Resolver{}
	ips, err := r.LookupIP(ctx, "ip4", host)
	if err != nil {
		return "", err
	}
	if len(ips) == 0 {
		return "", fmt.Errorf("no IPv4 addresses for %s", host)
	}
	return ips[0].String(), nil
}

// ValidateConfigFile executes `xray run -test -c <path>`.
func ValidateConfigFile(ctx context.Context, xrayBin, cfgPath string) error {
	cmd := exec.CommandContext(ctx, xrayBin, "run", "-test", "-c", cfgPath)
	configureHiddenCmd(cmd)
	out, err := cmd.CombinedOutput()
	if err != nil {
		lines := strings.Split(strings.TrimSpace(string(out)), "\n")
		msg := "xray config invalid"
		for i := len(lines) - 1; i >= 0; i-- {
			if s := strings.TrimSpace(lines[i]); s != "" {
				msg = s
				break
			}
		}
		return fmt.Errorf("config validation failed: %s", msg)
	}
	return nil
}

// TempInstance represents a running temporary Xray instance bound to ephemeral loopback ports.
type TempInstance struct {
	HTTPPort  int
	SocksPort int
	tmpPath   string
	cmd       *exec.Cmd
}

// StartTemp validates an outbound config and launches an isolated temporary Xray instance.
func StartTemp(ctx context.Context, xrayBin string, outbound map[string]any) (*TempInstance, error) {
	socksPort, err := FindFreePort()
	if err != nil {
		return nil, err
	}
	httpPort, err := FindFreePort()
	if err != nil {
		return nil, err
	}
	cfg := BuildConfig(outbound, socksPort, httpPort)
	b, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return nil, err
	}

	tmp, err := os.CreateTemp("", "harpo-test-*.json")
	if err != nil {
		return nil, err
	}
	tmpPath := tmp.Name()
	cleanupFile := func() { _ = os.Remove(tmpPath) }

	if err := tmp.Chmod(0o600); err != nil {
		_ = tmp.Close()
		cleanupFile()
		return nil, err
	}
	if _, err := tmp.Write(b); err != nil {
		_ = tmp.Close()
		cleanupFile()
		return nil, err
	}
	if err := tmp.Close(); err != nil {
		cleanupFile()
		return nil, err
	}

	if err := ValidateConfigFile(ctx, xrayBin, tmpPath); err != nil {
		cleanupFile()
		return nil, err
	}

	cmd := exec.Command(xrayBin, "run", "-c", tmpPath)
	configureHiddenCmd(cmd)
	var stderrBuf strings.Builder
	cmd.Stderr = &stderrBuf
	if err := cmd.Start(); err != nil {
		cleanupFile()
		return nil, fmt.Errorf("failed to start temp xray: %w", err)
	}

	inst := &TempInstance{
		HTTPPort:  httpPort,
		SocksPort: socksPort,
		tmpPath:   tmpPath,
		cmd:       cmd,
	}

	// Wait up to 3s for temp HTTP port to start accepting connections.
	doneCh := make(chan error, 1)
	go func() { doneCh <- cmd.Wait() }()

	deadline := time.Now().Add(3 * time.Second)
	ready := false
	for time.Now().Before(deadline) {
		select {
		case <-ctx.Done():
			inst.Close()
			return nil, ctx.Err()
		case err := <-doneCh:
			_ = os.Remove(tmpPath)
			msg := strings.TrimSpace(stderrBuf.String())
			if len(msg) > 200 {
				msg = msg[:200]
			}
			if msg == "" && err != nil {
				msg = err.Error()
			}
			return nil, fmt.Errorf("temp xray exited early: %s", msg)
		default:
		}
		c, dialErr := net.DialTimeout("tcp4", fmt.Sprintf("127.0.0.1:%d", httpPort), 100*time.Millisecond)
		if dialErr == nil {
			_ = c.Close()
			ready = true
			break
		}
		time.Sleep(80 * time.Millisecond)
	}

	if !ready {
		inst.Close()
		msg := strings.TrimSpace(stderrBuf.String())
		if len(msg) > 200 {
			msg = msg[:200]
		}
		return nil, fmt.Errorf("temp xray failed to bind port %d: %s", httpPort, msg)
	}
	return inst, nil
}

// Close terminates the temporary Xray process and deletes its temporary config file.
func (t *TempInstance) Close() {
	if t == nil {
		return
	}
	if t.cmd != nil && t.cmd.Process != nil {
		_ = t.cmd.Process.Kill()
		_, _ = t.cmd.Process.Wait()
	}
	if t.tmpPath != "" {
		_ = os.Remove(t.tmpPath)
	}
}
