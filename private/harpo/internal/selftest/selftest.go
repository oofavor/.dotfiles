package selftest

import (
	"bufio"
	"bytes"
	"context"
	"crypto/tls"
	"encoding/base64"
	"encoding/binary"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"net/netip"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"time"

	"harpo/internal/apps"
	"harpo/internal/config"
	"harpo/internal/probe"
	"harpo/internal/sub"
	"harpo/internal/system"
	"harpo/internal/xray"
)

// Status represents a single test outcome.
type Status string

const (
	StatusPass Status = "PASS"
	StatusFail Status = "FAIL"
	StatusSkip Status = "SKIP"
)

// TestResult captures one executed test case.
type TestResult struct {
	Suite      string
	Name       string
	Status     Status
	Detail     string
	DurationMS int64
}

// Event is emitted incrementally during a test run so both CLI and TUI can stream live output.
type Event struct {
	SuiteTitle string
	Result     *TestResult
	Done       bool
	Summary    Summary
}

// Summary holds aggregate counters.
type Summary struct {
	Passed  int
	Failed  int
	Skipped int
	Total   int
	Results []TestResult
}

type skipErr struct{ msg string }

func (e skipErr) Error() string { return e.msg }

func skip(format string, args ...any) error {
	return skipErr{msg: fmt.Sprintf(format, args...)}
}

// Options configures a selftest run.
type Options struct {
	Fast    bool
	Verbose bool
	OnEvent func(Event)
}

// RunInternalProbe handles `harpo _probe <mode> ...` used inside isp-isolated.slice
// so penetration tests run natively in Go without requiring Python or Bash /dev/tcp.
func RunInternalProbe(args []string) int {
	if len(args) == 0 {
		return 2
	}
	switch args[0] {
	case "tcp4", "tcp6":
		if len(args) < 4 {
			return 2
		}
		host := args[1]
		port := args[2]
		ms, _ := strconv.Atoi(args[3])
		if ms <= 0 {
			ms = 600
		}
		addr := net.JoinHostPort(host, port)
		c, err := net.DialTimeout(args[0], addr, time.Duration(ms)*time.Millisecond)
		if err != nil {
			fmt.Printf("BLOCKED:%v\n", err)
			return 42
		}
		_ = c.Close()
		fmt.Println("LEAKED")
		return 0

	case "echo":
		if len(args) < 4 {
			return 2
		}
		network, host, port := args[1], args[2], args[3]
		c, err := net.DialTimeout(network, net.JoinHostPort(host, port), 1500*time.Millisecond)
		if err != nil {
			fmt.Fprintf(os.Stderr, "dial failed: %v\n", err)
			return 1
		}
		defer c.Close()
		_ = c.SetDeadline(time.Now().Add(2 * time.Second))
		if _, err := c.Write([]byte("ok")); err != nil {
			return 1
		}
		buf := make([]byte, 32)
		n, err := c.Read(buf)
		if err != nil || string(buf[:n]) != "ECHO:ok" {
			fmt.Fprintf(os.Stderr, "unexpected echo %q err=%v\n", string(buf[:n]), err)
			return 1
		}
		fmt.Println("LOOPBACK_OK")
		return 0

	case "dns":
		q := buildDNSQuery("api.anthropic.com")
		for _, target := range args[1:] {
			c, err := net.DialTimeout("udp4", net.JoinHostPort(target, "53"), 500*time.Millisecond)
			if err != nil {
				continue
			}
			_ = c.SetDeadline(time.Now().Add(500 * time.Millisecond))
			_, _ = c.Write(q)
			buf := make([]byte, 512)
			n, err := c.Read(buf)
			_ = c.Close()
			if err == nil && n > 0 {
				fmt.Printf("DNS_LEAK:%s:%d\n", target, n)
				return 0
			}
		}
		fmt.Println("ALL_DNS_BLOCKED")
		return 42

	case "quic":
		c, err := net.DialTimeout("udp4", "1.1.1.1:443", 500*time.Millisecond)
		if err != nil {
			fmt.Printf("BLOCKED:%v\n", err)
			return 42
		}
		defer c.Close()
		_ = c.SetDeadline(time.Now().Add(500 * time.Millisecond))
		pkt := make([]byte, 1201)
		pkt[0] = 0xc0
		_, _ = c.Write(pkt)
		buf := make([]byte, 2048)
		n, err := c.Read(buf)
		if err == nil && n > 0 {
			fmt.Printf("UDP_LEAK:%d\n", n)
			return 0
		}
		fmt.Printf("BLOCKED:%v\n", err)
		return 42

	case "http-get":
		// _probe http-get <http|socks5> <port> <url>
		if len(args) < 4 {
			return 2
		}
		kind := args[1]
		port, _ := strconv.Atoi(args[2])
		targetURL := args[3]
		var client *http.Client
		var tr *http.Transport
		var err error
		if kind == "socks5" {
			client, tr, err = probe.HTTPClientForSOCKS5(port, 10*time.Second)
			if err != nil {
				fmt.Fprintf(os.Stderr, "%v\n", err)
				return 1
			}
		} else {
			client, tr = probe.HTTPClientForProxy(port, 10*time.Second)
		}
		defer tr.CloseIdleConnections()
		t0 := time.Now()
		resp, err := client.Get(targetURL)
		if err != nil {
			fmt.Fprintf(os.Stderr, "%v\n", err)
			return 1
		}
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
		_ = resp.Body.Close()
		if len(args) >= 5 && args[4] == "meta" {
			fmt.Printf("%d %d %d\n", resp.StatusCode, resp.ProtoMajor, time.Since(t0).Milliseconds())
		} else {
			fmt.Print(strings.TrimSpace(string(body)))
		}
		return 0

	case "keepalive":
		if len(args) < 2 {
			return 2
		}
		port, _ := strconv.Atoi(args[1])
		conn, err := net.DialTimeout("tcp4", fmt.Sprintf("127.0.0.1:%d", port), 10*time.Second)
		if err != nil {
			fmt.Fprintf(os.Stderr, "dial: %v\n", err)
			return 1
		}
		defer conn.Close()
		_ = conn.SetDeadline(time.Now().Add(10 * time.Second))
		if _, err := conn.Write([]byte("CONNECT api.anthropic.com:443 HTTP/1.1\r\nHost: api.anthropic.com:443\r\n\r\n")); err != nil {
			fmt.Fprintf(os.Stderr, "connect write: %v\n", err)
			return 1
		}
		br := bufio.NewReader(conn)
		statusLine, err := br.ReadString('\n')
		if err != nil || !strings.Contains(statusLine, "200") {
			fmt.Fprintf(os.Stderr, "connect hdr: %q err=%v\n", statusLine, err)
			return 1
		}
		for {
			l, err := br.ReadString('\n')
			if err != nil || l == "\r\n" {
				break
			}
		}
		tlsConn := tls.Client(conn, &tls.Config{ServerName: "api.anthropic.com", MinVersion: tls.VersionTLS12})
		if err := tlsConn.Handshake(); err != nil {
			fmt.Fprintf(os.Stderr, "tls: %v\n", err)
			return 1
		}
		defer tlsConn.Close()
		reqBytes := []byte("GET / HTTP/1.1\r\nHost: api.anthropic.com\r\nConnection: keep-alive\r\n\r\n")
		_ = tlsConn.SetDeadline(time.Now().Add(8 * time.Second))
		if _, err := tlsConn.Write(reqBytes); err != nil {
			fmt.Fprintf(os.Stderr, "req1 write: %v\n", err)
			return 1
		}
		buf := make([]byte, 4096)
		n, err := tlsConn.Read(buf)
		if err != nil || !bytes.Contains(buf[:n], []byte("HTTP/1.1")) {
			fmt.Fprintf(os.Stderr, "req1 read: %v\n", err)
			return 1
		}
		time.Sleep(20 * time.Second)
		t0 := time.Now()
		_ = tlsConn.SetDeadline(time.Now().Add(8 * time.Second))
		if _, err := tlsConn.Write(reqBytes); err != nil {
			fmt.Fprintf(os.Stderr, "req2 write: %v\n", err)
			return 1
		}
		n2, err := tlsConn.Read(buf)
		if err != nil || !bytes.Contains(buf[:n2], []byte("HTTP/1.1")) {
			fmt.Fprintf(os.Stderr, "stale socket after 20s idle: %v\n", err)
			return 1
		}
		fmt.Printf("REUSED_OK %dms\n", time.Since(t0).Milliseconds())
		return 0

	case "env-check":
		cg, _ := os.ReadFile("/proc/self/cgroup")
		out, _ := json.Marshal(map[string]string{
			"cg":          string(cg),
			"TZ":          os.Getenv("TZ"),
			"HTTP_PROXY":  os.Getenv("HTTP_PROXY"),
			"HTTPS_PROXY": os.Getenv("HTTPS_PROXY"),
			"ALL_PROXY":   os.Getenv("ALL_PROXY"),
			"NO_PROXY":    os.Getenv("NO_PROXY"),
		})
		fmt.Println(string(out))
		return 0

	case "echo-args":
		wd, _ := os.Getwd()
		stdinBytes, _ := io.ReadAll(os.Stdin)
		out, _ := json.Marshal(map[string]any{
			"cwd":   wd,
			"args":  args[1:],
			"stdin": strings.TrimSpace(string(stdinBytes)),
		})
		fmt.Println(string(out))
		return 0
	}
	return 2
}

func buildDNSQuery(domain string) []byte {
	var buf bytes.Buffer
	buf.Write([]byte{0x12, 0x34, 0x01, 0x00})
	_ = binary.Write(&buf, binary.BigEndian, uint16(1))
	_ = binary.Write(&buf, binary.BigEndian, uint16(0))
	_ = binary.Write(&buf, binary.BigEndian, uint16(0))
	_ = binary.Write(&buf, binary.BigEndian, uint16(0))
	for _, part := range strings.Split(domain, ".") {
		buf.WriteByte(byte(len(part)))
		buf.WriteString(part)
	}
	buf.WriteByte(0)
	_ = binary.Write(&buf, binary.BigEndian, uint16(1)) // A
	_ = binary.Write(&buf, binary.BigEndian, uint16(1)) // IN
	return buf.Bytes()
}

type runner struct {
	ctx     context.Context
	opts    Options
	suite   string
	results []TestResult
	paths   config.Paths
	exe     string
	xrayBin string
}

func (r *runner) setSuite(title string) {
	r.suite = title
	if r.opts.OnEvent != nil {
		r.opts.OnEvent(Event{SuiteTitle: title})
	}
}

func (r *runner) test(name string, fn func() (string, error)) {
	if r.ctx.Err() != nil {
		return
	}
	t0 := time.Now()
	detail, err := fn()
	ms := time.Since(t0).Milliseconds()
	res := TestResult{
		Suite:      r.suite,
		Name:       name,
		DurationMS: ms,
	}
	switch {
	case err == nil:
		res.Status = StatusPass
		res.Detail = detail
	case errors.As(err, new(skipErr)):
		res.Status = StatusSkip
		res.Detail = err.Error()
	default:
		res.Status = StatusFail
		res.Detail = err.Error()
	}
	r.results = append(r.results, res)
	if r.opts.OnEvent != nil {
		cp := res
		r.opts.OnEvent(Event{Result: &cp})
	}
}

func (r *runner) inSlice(timeout time.Duration, env []string, cmdArgs ...string) (string, string, int) {
	cCtx, cancel := context.WithTimeout(r.ctx, timeout)
	defer cancel()
	var cmd *exec.Cmd
	if runtime.GOOS == "windows" {
		if len(cmdArgs) == 0 {
			return "", "empty command", -1
		}
		cmd = exec.CommandContext(cCtx, cmdArgs[0], cmdArgs[1:]...)
	} else {
		sd := config.ResolveBin("HARPO_SYSTEMD_RUN", "", "systemd-run")
		full := append([]string{"--user", "--scope", "--quiet", "--collect", "--slice=" + config.SliceUnit, "--"}, cmdArgs...)
		cmd = exec.CommandContext(cCtx, sd, full...)
	}
	if env != nil {
		cmd.Env = env
	}
	var outB, errB bytes.Buffer
	cmd.Stdout = &outB
	cmd.Stderr = &errB
	err := cmd.Run()
	rc := 0
	if err != nil {
		if ee, ok := err.(*exec.ExitError); ok {
			rc = ee.ExitCode()
		} else {
			rc = -1
		}
	}
	return outB.String(), errB.String(), rc
}

func runCmd(ctx context.Context, timeout time.Duration, env []string, dir, stdin string, name string, args ...string) (string, string, int) {
	cCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()
	cmd := exec.CommandContext(cCtx, name, args...)
	if env != nil {
		cmd.Env = env
	}
	if dir != "" {
		cmd.Dir = dir
	}
	if stdin != "" {
		cmd.Stdin = strings.NewReader(stdin)
	}
	var outB, errB bytes.Buffer
	cmd.Stdout = &outB
	cmd.Stderr = &errB
	err := cmd.Run()
	rc := 0
	if err != nil {
		if ee, ok := err.(*exec.ExitError); ok {
			rc = ee.ExitCode()
		} else {
			rc = -1
		}
	}
	return outB.String(), errB.String(), rc
}

// RunAll executes all 6 verification suites non-destructively.
func RunAll(ctx context.Context, opts Options) Summary {
	exe, _ := os.Executable()
	r := &runner{
		ctx:     ctx,
		opts:    opts,
		paths:   config.DefaultPaths(),
		exe:     exe,
		xrayBin: config.ResolveXrayBin(),
	}

	r.runSuite1()
	r.runSuite2()
	r.runSuite3()
	r.runSuite4()
	r.runSuite5()
	r.runSuite6()

	var sum Summary
	sum.Results = r.results
	sum.Total = len(r.results)
	for _, res := range r.results {
		switch res.Status {
		case StatusPass:
			sum.Passed++
		case StatusFail:
			sum.Failed++
		case StatusSkip:
			sum.Skipped++
		}
	}
	if opts.OnEvent != nil {
		opts.OnEvent(Event{Done: true, Summary: sum})
	}
	return sum
}

// ---------------------------------------------------------------------------
// Suite 1: Static Config & Security Hygiene
// ---------------------------------------------------------------------------
func (r *runner) runSuite1() {
	r.setSuite("Suite 1: Static Config & Security Hygiene")

	r.test("Config directory mode is 0700", func() (string, error) {
		fi, err := os.Stat(r.paths.CfgDir)
		if err != nil || !fi.IsDir() {
			return "", fmt.Errorf("%s missing", r.paths.CfgDir)
		}
		if runtime.GOOS == "windows" {
			return "", skip("Windows NTFS user-profile ACLs apply instead of POSIX 0700 bits")
		}
		mode := fi.Mode().Perm()
		if mode != 0o700 {
			return "", fmt.Errorf("mode is %04o, expected 0700", mode)
		}
		return "0700", nil
	})

	r.test("All files in config dir are user-only (0600/0700)", func() (string, error) {
		if runtime.GOOS == "windows" {
			return "", skip("Windows NTFS user-profile ACLs apply instead of POSIX 0600 bits")
		}
		entries, err := os.ReadDir(r.paths.CfgDir)
		if err != nil {
			return "", err
		}
		count := 0
		for _, e := range entries {
			if e.IsDir() {
				continue
			}
			fi, err := e.Info()
			if err != nil {
				return "", err
			}
			mode := fi.Mode().Perm()
			if mode&0o077 != 0 {
				return "", fmt.Errorf("%s is accessible by group/other (%04o)", e.Name(), mode)
			}
			count++
		}
		if count == 0 {
			return "", errors.New("no files found in config dir")
		}
		return fmt.Sprintf("%d files strictly user-only", count), nil
	})

	r.test("config.json invariants (127.0.0.1 only, no direct outbound, AsIs, keepalive)", func() (string, error) {
		b, err := os.ReadFile(r.paths.CfgPath)
		if err != nil {
			return "", err
		}
		var cfg map[string]any
		if err := json.Unmarshal(b, &cfg); err != nil {
			return "", err
		}
		inbounds, _ := cfg["inbounds"].([]any)
		byPort := map[int]map[string]any{}
		for _, raw := range inbounds {
			ib, _ := raw.(map[string]any)
			if fmt.Sprint(ib["listen"]) != "127.0.0.1" {
				return "", fmt.Errorf("inbound %v listens on %v (must be 127.0.0.1)", ib["tag"], ib["listen"])
			}
			if p, ok := ib["port"].(float64); ok {
				byPort[int(p)] = ib
			}
		}
		socksIB, okS := byPort[config.SocksPort]
		httpIB, okH := byPort[config.HTTPPort]
		if !okS || !okH {
			return "", fmt.Errorf("missing expected ports %d/%d", config.SocksPort, config.HTTPPort)
		}
		if fmt.Sprint(socksIB["protocol"]) != "socks" || fmt.Sprint(httpIB["protocol"]) != "http" {
			return "", errors.New("unexpected inbound protocols")
		}
		if st, _ := socksIB["settings"].(map[string]any); st["udp"] != false {
			return "", errors.New("SOCKS5 UDP should be disabled")
		}

		outbounds, _ := cfg["outbounds"].([]any)
		if len(outbounds) != 1 {
			return "", fmt.Errorf("expected exactly 1 outbound, found %d", len(outbounds))
		}
		ob, _ := outbounds[0].(map[string]any)
		proto := fmt.Sprint(ob["protocol"])
		if proto == "freedom" || proto == "direct" {
			return "", fmt.Errorf("forbidden direct outbound protocol: %s", proto)
		}
		ss, _ := ob["streamSettings"].(map[string]any)
		so, _ := ss["sockopt"].(map[string]any)
		idle, _ := so["tcpKeepAliveIdle"].(float64)
		itv, _ := so["tcpKeepAliveInterval"].(float64)
		nodelay, _ := so["tcpNoDelay"].(bool)
		if idle <= 0 || itv <= 0 || !nodelay {
			return "", errors.New("missing keepalive or tcpNoDelay=true in sockopt")
		}
		routing, _ := cfg["routing"].(map[string]any)
		if fmt.Sprint(routing["domainStrategy"]) != "AsIs" {
			return "", fmt.Errorf("expected domainStrategy=AsIs, got %v", routing["domainStrategy"])
		}
		return fmt.Sprintf("1 outbound (%s), AsIs, keepalive=%.0fs/%.0fs", proto, idle, itv), nil
	})

	r.test("config.json validates cleanly with xray run -test", func() (string, error) {
		if err := xray.ValidateConfigFile(r.ctx, r.xrayBin, r.paths.CfgPath); err != nil {
			return "", err
		}
		return "Configuration OK", nil
	})

	r.test("expected-ip is a valid public global IP", func() (string, error) {
		ipStr := config.ReadTrimmed(r.paths.ExpectedIP)
		if ipStr == "" {
			return "", fmt.Errorf("missing or empty %s", r.paths.ExpectedIP)
		}
		addr, err := netip.ParseAddr(ipStr)
		if err != nil {
			return "", err
		}
		if !addr.IsGlobalUnicast() || addr.IsPrivate() || addr.IsLoopback() || addr.IsLinkLocalUnicast() {
			return "", fmt.Errorf("expected-ip %s is not a public global IP", ipStr)
		}
		kind := "IPv4"
		if addr.Is6() {
			kind = "IPv6"
		}
		return "public " + kind, nil
	})

	r.test("Zero runtime secrets/IPs leaked into tracked dotfiles", func() (string, error) {
		dotDir := os.Getenv("HARPO_DOTFILES")
		if dotDir == "" {
			dotDir = filepath.Join(r.paths.Home, ".dotfiles")
		}
		if _, err := os.Stat(filepath.Join(dotDir, ".git")); err != nil {
			return "", skip("Dotfiles git repo not present")
		}
		b, err := os.ReadFile(r.paths.CfgPath)
		if err != nil {
			return "", skip("config.json not present")
		}
		var cfg map[string]any
		if err := json.Unmarshal(b, &cfg); err != nil {
			return "", err
		}
		type secret struct{ label, val string }
		var secrets []secret
		if eip := config.ReadTrimmed(r.paths.ExpectedIP); eip != "" {
			secrets = append(secrets, secret{"expected-ip", eip})
		}
		if obs, _ := cfg["outbounds"].([]any); len(obs) > 0 {
			ob, _ := obs[0].(map[string]any)
			if st, _ := ob["settings"].(map[string]any); st != nil {
				if id, _ := st["id"].(string); id != "" {
					secrets = append(secrets, secret{"vless-uuid", id})
				}
				if addr, _ := st["address"].(string); addr != "" {
					secrets = append(secrets, secret{"relay-addr", addr})
				}
			}
			if ss, _ := ob["streamSettings"].(map[string]any); ss != nil {
				if rs, _ := ss["realitySettings"].(map[string]any); rs != nil {
					for _, k := range []string{"password", "publicKey", "shortId"} {
						if v, _ := rs[k].(string); len(v) >= 6 {
							secrets = append(secrets, secret{"reality-" + k, v})
						}
					}
				}
			}
		}

		out, _, rc := runCmd(r.ctx, 5*time.Second, nil, "", "", "git", "-C", dotDir, "ls-files")
		if rc != 0 {
			return "", skip("git ls-files unavailable")
		}
		files := strings.Split(strings.TrimSpace(out), "\n")
		var leaks []string
		for _, rel := range files {
			if rel == "" {
				continue
			}
			p := filepath.Join(dotDir, rel)
			fi, err := os.Stat(p)
			if err != nil || fi.IsDir() || fi.Size() > 2<<20 {
				continue
			}
			content, err := os.ReadFile(p)
			if err != nil {
				continue
			}
			for _, s := range secrets {
				if s.val != "" && bytes.Contains(content, []byte(s.val)) {
					leaks = append(leaks, fmt.Sprintf("%s found in %s", s.label, rel))
				}
			}
		}
		if len(leaks) > 0 {
			return "", errors.New(strings.Join(leaks, "; "))
		}
		return fmt.Sprintf("checked %d secret values across %d tracked files", len(secrets), len(files)), nil
	})
}

// ---------------------------------------------------------------------------
// Suite 2: Systemd Services, Cgroup Pinning & Listening Sockets
// ---------------------------------------------------------------------------
func (r *runner) runSuite2() {
	if runtime.GOOS == "windows" {
		r.setSuite("Suite 2: Proxy Daemon, Scheduled Task, Firewall & Sockets")

		r.test("Harpo xray.exe proxy is active on :10808/:10809", func() (string, error) {
			if st := system.ServiceActive(config.ProxyUnit, true); st != "active" {
				return "", fmt.Errorf("proxy is %q (expected active)", st)
			}
			return "xray.exe listening on 127.0.0.1:10808 & :10809", nil
		})

		r.test("Logon Scheduled Task (HarpoProxy) is registered", func() (string, error) {
			if st := system.ServiceActive(config.AnchorUnit, true); st != "active" {
				return "", skip("HarpoProxy logon task not registered yet (run `harpo setup`)")
			}
			return config.WinProxyTask + " registered (ONLOGON)", nil
		})

		r.test("Windows Defender Firewall kill-switch status", func() (string, error) {
			ks := probe.CheckKillSwitch(r.ctx)
			if !ks.Blocked {
				return "", skip("soft mode active: %s", ks.Detail)
			}
			return ks.Detail, nil
		})

		r.test("Proxy ports 10808/10809 bound exclusively to 127.0.0.1", func() (string, error) {
			socks, err := probe.ParseProcNetTCP()
			if err != nil {
				return "", err
			}
			found := map[int]string{}
			for _, s := range socks {
				if s.Port == config.SocksPort || s.Port == config.HTTPPort {
					if s.Addr != "127.0.0.1" {
						return "", fmt.Errorf("port %d bound to %s, expected 127.0.0.1", s.Port, s.Addr)
					}
					found[s.Port] = fmt.Sprintf("%s:%d", s.Addr, s.Port)
				}
			}
			if _, ok := found[config.SocksPort]; !ok {
				return "", fmt.Errorf("missing listener on 127.0.0.1:%d", config.SocksPort)
			}
			if _, ok := found[config.HTTPPort]; !ok {
				return "", fmt.Errorf("missing listener on 127.0.0.1:%d", config.HTTPPort)
			}
			return "127.0.0.1:10808 & 127.0.0.1:10809 only", nil
		})
		return
	}

	r.setSuite("Suite 2: Systemd Services, Cgroup Pinning & Sockets")

	r.test("All 3 systemd units are active", func() (string, error) {
		for _, u := range []struct {
			name string
			user bool
		}{
			{config.ProxyUnit, true},
			{config.AnchorUnit, true},
			{config.FirewallUnit, false},
		} {
			st := system.ServiceActive(u.name, u.user)
			if st != "active" {
				return "", fmt.Errorf("%s is %q (expected active)", u.name, st)
			}
		}
		return fmt.Sprintf("%s, %s, %s all active", config.ProxyUnit, config.AnchorUnit, config.FirewallUnit), nil
	})

	r.test("User lingering is enabled (survives logout / boot)", func() (string, error) {
		uname := os.Getenv("USER")
		if uname == "" {
			return "", skip("USER env unset")
		}
		out, _, _ := runCmd(r.ctx, 5*time.Second, nil, "", "", "loginctl", "show-user", uname, "-p", "Linger", "--value")
		if strings.TrimSpace(out) != "yes" {
			return "", fmt.Errorf("Linger=%q (expected yes)", strings.TrimSpace(out))
		}
		return fmt.Sprintf("Linger=yes for %s", uname), nil
	})

	r.test("harpo.slice/harpo-isolated.slice exists and is pinned by anchor PID", func() (string, error) {
		uid := os.Getuid()
		cg := fmt.Sprintf("/sys/fs/cgroup/user.slice/user-%d.slice/user@%d.service/harpo.slice/%s", uid, uid, config.SliceUnit)
		if fi, err := os.Stat(cg); err != nil || !fi.IsDir() {
			return "", fmt.Errorf("cgroup dir missing: %s", cg)
		}
		anchor := filepath.Join(cg, config.AnchorUnit, "cgroup.procs")
		b, err := os.ReadFile(anchor)
		if err != nil {
			return "", fmt.Errorf("anchor cgroup.procs missing: %w", err)
		}
		pids := strings.Fields(string(b))
		if len(pids) == 0 {
			return "", fmt.Errorf("no live anchor PID in %s", config.SliceUnit)
		}
		return fmt.Sprintf("anchor PID %s pinning slice", pids[0]), nil
	})

	r.test("Kernel cgroup firewall backend is ready (xt_cgroup or nftables)", func() (string, error) {
		if b, err := os.ReadFile("/proc/modules"); err == nil && strings.Contains(string(b), "xt_cgroup ") {
			return "xt_cgroup loaded", nil
		}
		if _, err := exec.LookPath("nft"); err == nil {
			return "nftables cgroupv2 backend", nil
		}
		return "", errors.New("neither xt_cgroup kernel module nor nft binary found")
	})

	r.test("Proxy ports 10808/10809 bound exclusively to 127.0.0.1", func() (string, error) {
		socks, err := probe.ParseProcNetTCP()
		if err != nil {
			return "", err
		}
		found := map[int]string{}
		for _, s := range socks {
			if s.Port == config.SocksPort || s.Port == config.HTTPPort {
				if s.Addr != "127.0.0.1" {
					return "", fmt.Errorf("port %d bound to %s, expected 127.0.0.1", s.Port, s.Addr)
				}
				found[s.Port] = fmt.Sprintf("%s:%d", s.Addr, s.Port)
			}
		}
		if _, ok := found[config.SocksPort]; !ok {
			return "", fmt.Errorf("missing listener on 127.0.0.1:%d", config.SocksPort)
		}
		if _, ok := found[config.HTTPPort]; !ok {
			return "", fmt.Errorf("missing listener on 127.0.0.1:%d", config.HTTPPort)
		}
		return "127.0.0.1:10808 & 127.0.0.1:10809 only", nil
	})
}

// ---------------------------------------------------------------------------
// Suite 3: Kernel Kill-Switch Penetration & Loopback Isolation
// ---------------------------------------------------------------------------
func (r *runner) runSuite3() {
	if runtime.GOOS == "windows" {
		r.setSuite("Suite 3: Windows Firewall Kill-Switch & Loopback Verification")

		r.test("Windows Defender Firewall service & outbound rules active", func() (string, error) {
			ks := probe.CheckKillSwitch(r.ctx)
			if !ks.Blocked {
				return "", skip("soft mode (%s)", ks.Detail)
			}
			return ks.Detail, nil
		})

		r.test("All firewall-enabled wrapped .exe apps have active WFP block rules", func() (string, error) {
			list, err := apps.Load(r.paths)
			if err != nil {
				return "", err
			}
			shimsDir := apps.ShimsDir(r.paths)
			checked := 0
			var missing []string
			for _, a := range list {
				if !a.WantsFirewall() {
					continue
				}
				realBin, err := apps.ResolveRealBinary(a.TargetCommand(), shimsDir)
				if err != nil || !strings.EqualFold(filepath.Ext(realBin), ".exe") {
					continue
				}
				checked++
				if !probe.HasAppFirewallRule(r.ctx, realBin) {
					missing = append(missing, a.Name)
				}
			}
			if checked == 0 {
				return "", skip("no wrapped .exe apps registered yet")
			}
			if len(missing) > 0 {
				return "", skip("missing WFP rules for: %s (run Admin: harpo apply-firewall)", strings.Join(missing, ", "))
			}
			return fmt.Sprintf("%d wrapped .exe app(s) protected by WFP", checked), nil
		})

		r.test("Localhost IPv4 (127.0.0.1) remains reachable for proxy & dev servers", func() (string, error) {
			ln, port, err := startEchoServer("tcp4", "127.0.0.1:0")
			if err != nil {
				return "", err
			}
			defer ln.Close()
			out, errOut, rc := r.inSlice(5*time.Second, nil, r.exe, "_probe", "echo", "tcp4", "127.0.0.1", strconv.Itoa(port))
			if rc != 0 || !strings.Contains(out, "LOOPBACK_OK") {
				return "", fmt.Errorf("127.0.0.1 blocked: rc=%d err=%s", rc, strings.TrimSpace(errOut))
			}
			return fmt.Sprintf("127.0.0.1:%d reachable", port), nil
		})

		r.test("Localhost IPv6 (::1) remains reachable for dev servers", func() (string, error) {
			ln, port, err := startEchoServer("tcp6", "[::1]:0")
			if err != nil {
				return "", skip("IPv6 loopback unavailable: %v", err)
			}
			defer ln.Close()
			out, errOut, rc := r.inSlice(5*time.Second, nil, r.exe, "_probe", "echo", "tcp6", "::1", strconv.Itoa(port))
			if rc != 0 || !strings.Contains(out, "LOOPBACK_OK") {
				return "", fmt.Errorf("::1 blocked: rc=%d err=%s", rc, strings.TrimSpace(errOut))
			}
			return fmt.Sprintf("[::1]:%d reachable", port), nil
		})
		return
	}

	r.setSuite("Suite 3: Kernel Kill-Switch Penetration & Loopback Isolation")

	assertTCPBlocked := func(family, host string, port int) (string, error) {
		out, errOut, rc := r.inSlice(5*time.Second, nil, r.exe, "_probe", family, host, strconv.Itoa(port), "600")
		if rc != 42 || !strings.Contains(out, "BLOCKED:") || strings.Contains(out, "LEAKED") {
			return "", fmt.Errorf("expected BLOCKED for %s:%d, got rc=%d out=%q err=%q", host, port, rc, strings.TrimSpace(out), strings.TrimSpace(errOut))
		}
		return strings.TrimPrefix(strings.TrimSpace(out), "BLOCKED:"), nil
	}

	r.test("Direct IPv4 TCP to 1.1.1.1:443 is kernel-dropped", func() (string, error) {
		return assertTCPBlocked("tcp4", "1.1.1.1", 443)
	})

	r.test("Direct IPv4 TCP to Anthropic (160.79.104.10:443) is kernel-dropped", func() (string, error) {
		return assertTCPBlocked("tcp4", "160.79.104.10", 443)
	})

	r.test("Direct IPv4 TCP to relay IP is kernel-dropped", func() (string, error) {
		b, err := os.ReadFile(r.paths.CfgPath)
		if err != nil {
			return "", err
		}
		var cfg map[string]any
		if err := json.Unmarshal(b, &cfg); err != nil {
			return "", err
		}
		obs, _ := cfg["outbounds"].([]any)
		if len(obs) == 0 {
			return "", errors.New("no outbound in config.json")
		}
		ob, _ := obs[0].(map[string]any)
		host, port := xray.ExtractHostPort(ob)
		if host == "" {
			return "", errors.New("no outbound host in config.json")
		}
		if port == 0 {
			port = 443
		}
		return assertTCPBlocked("tcp4", host, port)
	})

	r.test("Direct IPv4 TCP to LAN gateway is kernel-dropped", func() (string, error) {
		gw := findLocalGateway(r.ctx)
		if gw == "" {
			return "", skip("No default IPv4 gateway found")
		}
		return assertTCPBlocked("tcp4", gw, 80)
	})

	r.test("Direct UDP:53 DNS queries (1.1.1.1, 8.8.8.8, router) are kernel-dropped", func() (string, error) {
		targets := []string{"1.1.1.1", "8.8.8.8"}
		if ns := findResolvNameserver(); ns != "" && ns != "127.0.0.1" && ns != "1.1.1.1" && ns != "8.8.8.8" {
			targets = append(targets, ns)
		}
		args := append([]string{r.exe, "_probe", "dns"}, targets...)
		out, errOut, rc := r.inSlice(5*time.Second, nil, args...)
		if rc != 42 || !strings.Contains(out, "ALL_DNS_BLOCKED") {
			return "", fmt.Errorf("DNS leak detected: rc=%d out=%q err=%q", rc, out, errOut)
		}
		return "blocked UDP:53 to " + strings.Join(targets, ", "), nil
	})

	r.test("Direct UDP:443 (QUIC) packets are kernel-dropped", func() (string, error) {
		out, errOut, rc := r.inSlice(5*time.Second, nil, r.exe, "_probe", "quic")
		if rc != 42 || strings.Contains(out, "UDP_LEAK") {
			return "", fmt.Errorf("UDP/QUIC leak: rc=%d out=%q err=%q", rc, out, errOut)
		}
		return "UDP:443 dropped", nil
	})

	r.test("Non-127.0.0.1 loopback (127.0.0.2 / 127.0.0.53 stub DNS) is kernel-dropped", func() (string, error) {
		ln, port, err := startEchoServer("tcp4", "127.0.0.2:0")
		if err != nil {
			return "", err
		}
		defer ln.Close()
		// First prove host outside slice CAN reach 127.0.0.2
		c, err := net.DialTimeout("tcp4", fmt.Sprintf("127.0.0.2:%d", port), time.Second)
		if err != nil {
			return "", fmt.Errorf("host could not reach local 127.0.0.2 echo server: %w", err)
		}
		_, _ = c.Write([]byte("hi"))
		buf := make([]byte, 16)
		n, _ := c.Read(buf)
		_ = c.Close()
		if string(buf[:n]) != "ECHO:hi" {
			return "", errors.New("echo server sanity check failed")
		}
		return assertTCPBlocked("tcp4", "127.0.0.2", port)
	})

	r.test("Direct IPv6 TCP to [2606:4700:4700::1111]:443 is kernel-dropped", func() (string, error) {
		return assertTCPBlocked("tcp6", "2606:4700:4700::1111", 443)
	})

	r.test("Nested & background child processes inherit cgroup kill-switch", func() (string, error) {
		script := fmt.Sprintf(
			`grep -q %s /proc/self/cgroup || exit 10; `+
				`(grep -q %s /proc/self/cgroup || exit 11; %q _probe tcp4 1.1.1.1 443 400 >/dev/null; test $? -eq 42 || exit 12) & wait $!`,
			config.SliceUnit, config.SliceUnit, r.exe,
		)
		_, errOut, rc := r.inSlice(5*time.Second, nil, "sh", "-c", script)
		if rc != 0 {
			return "", fmt.Errorf("child/bg escaped isolation (rc=%d): %s", rc, strings.TrimSpace(errOut))
		}
		return "nested & background children stay in " + config.SliceUnit + " & blocked", nil
	})

	r.test("Localhost IPv4 (127.0.0.1) remains reachable for proxy & dev servers", func() (string, error) {
		ln, port, err := startEchoServer("tcp4", "127.0.0.1:0")
		if err != nil {
			return "", err
		}
		defer ln.Close()
		out, errOut, rc := r.inSlice(5*time.Second, nil, r.exe, "_probe", "echo", "tcp4", "127.0.0.1", strconv.Itoa(port))
		if rc != 0 || !strings.Contains(out, "LOOPBACK_OK") {
			return "", fmt.Errorf("127.0.0.1 blocked: rc=%d err=%s", rc, strings.TrimSpace(errOut))
		}
		return fmt.Sprintf("127.0.0.1:%d reachable", port), nil
	})

	r.test("Localhost IPv6 (::1) remains reachable for dev servers", func() (string, error) {
		ln, port, err := startEchoServer("tcp6", "[::1]:0")
		if err != nil {
			return "", skip("IPv6 loopback unavailable: %v", err)
		}
		defer ln.Close()
		out, errOut, rc := r.inSlice(5*time.Second, nil, r.exe, "_probe", "echo", "tcp6", "::1", strconv.Itoa(port))
		if rc != 0 || !strings.Contains(out, "LOOPBACK_OK") {
			return "", fmt.Errorf("::1 blocked: rc=%d err=%s", rc, strings.TrimSpace(errOut))
		}
		return fmt.Sprintf("[::1]:%d reachable", port), nil
	})
}

// ---------------------------------------------------------------------------
// Suite 4: End-to-End Proxy Routing, Remote DNS, Keep-Alive & Burst
// ---------------------------------------------------------------------------
func (r *runner) runSuite4() {
	r.setSuite("Suite 4: End-to-End Proxy Routing, Remote DNS, Keep-Alive & Burst")
	expectedIP := config.ReadTrimmed(r.paths.ExpectedIP)

	r.test("HTTP proxy (:10809) exits via expected-ip on 3 endpoints (remote DNS)", func() (string, error) {
		if expectedIP == "" {
			return "", errors.New("expected-ip unset")
		}
		for _, u := range []string{"https://api.ipify.org", "https://api64.ipify.org", "https://ifconfig.me/ip"} {
			out, errOut, rc := r.inSlice(12*time.Second, nil, r.exe, "_probe", "http-get", "http", strconv.Itoa(config.HTTPPort), u)
			if rc != 0 {
				return "", fmt.Errorf("HTTP proxy failed on %s: %s", u, strings.TrimSpace(errOut))
			}
			got := strings.TrimSpace(out)
			if got != expectedIP {
				return "", fmt.Errorf("%s returned %q, expected %q", u, got, expectedIP)
			}
		}
		return fmt.Sprintf("3 endpoints (ipify v4/v6 & ifconfig.me) all == %s", expectedIP), nil
	})

	r.test("SOCKS5h proxy (:10808) exits via expected-ip (remote DNS)", func() (string, error) {
		if expectedIP == "" {
			return "", errors.New("expected-ip unset")
		}
		out, errOut, rc := r.inSlice(12*time.Second, nil, r.exe, "_probe", "http-get", "socks5", strconv.Itoa(config.SocksPort), "https://api.ipify.org")
		if rc != 0 {
			return "", fmt.Errorf("SOCKS5h failed: %s", strings.TrimSpace(errOut))
		}
		if strings.TrimSpace(out) != expectedIP {
			return "", fmt.Errorf("SOCKS5h returned %q, expected %q", strings.TrimSpace(out), expectedIP)
		}
		return fmt.Sprintf("SOCKS5h == %s (with local DNS blocked)", expectedIP), nil
	})

	r.test("Anthropic API reachable over HTTP/2 on both :10809 and :10808", func() (string, error) {
		var parts []string
		for _, p := range []struct {
			label, kind string
			port        int
		}{
			{"HTTP", "http", config.HTTPPort},
			{"SOCKS5h", "socks5", config.SocksPort},
		} {
			out, errOut, rc := r.inSlice(12*time.Second, nil, r.exe, "_probe", "http-get", p.kind, strconv.Itoa(p.port), "https://api.anthropic.com/", "meta")
			if rc != 0 {
				return "", fmt.Errorf("Anthropic via %s failed: %s", p.label, strings.TrimSpace(errOut))
			}
			fields := strings.Fields(out)
			if len(fields) < 3 {
				return "", fmt.Errorf("unexpected meta output: %q", out)
			}
			code, ver, ms := fields[0], fields[1], fields[2]
			if code != "200" && code != "401" && code != "403" && code != "404" && code != "405" {
				return "", fmt.Errorf("unexpected HTTP status %s", code)
			}
			if ver != "2" {
				return "", fmt.Errorf("expected HTTP/2, got HTTP/%s", ver)
			}
			parts = append(parts, fmt.Sprintf("%s=h%s %sms", p.label, ver, ms))
		}
		return strings.Join(parts, ", "), nil
	})

	r.test("Bun runtime fetch() inside harpo-isolated.slice exits via expected-ip", func() (string, error) {
		bun, err := exec.LookPath("bun")
		if err != nil {
			return "", skip("bun not installed")
		}
		js := `const r = await fetch("https://api.ipify.org", { signal: AbortSignal.timeout(8000) }); console.log((await r.text()).trim());`
		env := append(os.Environ(),
			fmt.Sprintf("HTTP_PROXY=http://127.0.0.1:%d", config.HTTPPort),
			fmt.Sprintf("HTTPS_PROXY=http://127.0.0.1:%d", config.HTTPPort),
		)
		out, errOut, rc := r.inSlice(12*time.Second, env, bun, "-e", js)
		if rc != 0 {
			return "", fmt.Errorf("bun fetch failed: %s", strings.TrimSpace(errOut))
		}
		if strings.TrimSpace(out) != expectedIP {
			return "", fmt.Errorf("bun got %q, expected %q", strings.TrimSpace(out), expectedIP)
		}
		return fmt.Sprintf("Bun fetch inside slice == %s", expectedIP), nil
	})

	r.test("10 concurrent HTTPS requests to Anthropic succeed without timeouts", func() (string, error) {
		const n = 10
		var wg sync.WaitGroup
		var mu sync.Mutex
		var errs []string
		var lats []int64

		client, tr := probe.HTTPClientForProxy(config.HTTPPort, 10*time.Second)
		defer tr.CloseIdleConnections()

		for i := 0; i < n; i++ {
			wg.Add(1)
			go func(idx int) {
				defer wg.Done()
				t0 := time.Now()
				req, _ := http.NewRequestWithContext(r.ctx, http.MethodGet, "https://api.anthropic.com/", nil)
				resp, err := client.Do(req)
				dt := time.Since(t0).Milliseconds()
				mu.Lock()
				defer mu.Unlock()
				if err != nil {
					errs = append(errs, fmt.Sprintf("#%d: %v", idx, err))
					return
				}
				_, _ = io.Copy(io.Discard, io.LimitReader(resp.Body, 4096))
				_ = resp.Body.Close()
				if resp.StatusCode != 200 && resp.StatusCode != 401 && resp.StatusCode != 403 && resp.StatusCode != 404 && resp.StatusCode != 405 {
					errs = append(errs, fmt.Sprintf("#%d: HTTP %d", idx, resp.StatusCode))
					return
				}
				lats = append(lats, dt)
			}(i)
		}
		wg.Wait()
		if len(errs) > 0 {
			return "", fmt.Errorf("burst failures (%d/%d): %s", len(errs), n, strings.Join(errs, "; "))
		}
		var sum, maxL int64
		for _, l := range lats {
			sum += l
			if l > maxL {
				maxL = l
			}
		}
		return fmt.Sprintf("%d/%d OK, avg=%dms, max=%dms", n, n, sum/int64(len(lats)), maxL), nil
	})

	r.test("Keep-alive TLS socket survives 20s idle past 15s keepalive threshold", func() (string, error) {
		if r.opts.Fast {
			return "", skip("--fast enabled (skips 20s idle keep-alive probe)")
		}
		out, errOut, rc := r.inSlice(35*time.Second, nil, r.exe, "_probe", "keepalive", strconv.Itoa(config.HTTPPort))
		if rc != 0 || !strings.Contains(out, "REUSED_OK") {
			return "", fmt.Errorf("keep-alive failed: %s %s", strings.TrimSpace(out), strings.TrimSpace(errOut))
		}
		return fmt.Sprintf("same TCP/TLS socket survived 20s idle (%s)", strings.TrimSpace(out)), nil
	})
}

// ---------------------------------------------------------------------------
// Suite 5: harpo run / harpo-run Fail-Closed & Edge Cases
// ---------------------------------------------------------------------------
func (r *runner) runSuite5() {
	r.setSuite("Suite 5: harpo run / harpo-run Fail-Closed & Edge Cases")
	expectedIP := config.ReadTrimmed(r.paths.ExpectedIP)

	// Prefer testing `harpo-run` if present, otherwise test `harpo run` directly.
	runCmdPrefix := []string{r.exe, "run"}
	if harpoRun, err := exec.LookPath("harpo-run"); err == nil {
		runCmdPrefix = []string{harpoRun}
	}
	execRunner := func(timeout time.Duration, env []string, dir, stdin string, extra ...string) (string, string, int) {
		args := append(append([]string{}, runCmdPrefix[1:]...), extra...)
		return runCmd(r.ctx, timeout, env, dir, stdin, runCmdPrefix[0], args...)
	}

	r.test("harpo run / harpo-run with no args exits 2 with usage", func() (string, error) {
		_, errOut, rc := execRunner(5*time.Second, nil, "", "")
		if rc != 2 || !strings.Contains(errOut, "Usage:") {
			return "", fmt.Errorf("expected exit 2 + Usage, got rc=%d err=%q", rc, errOut)
		}
		return "exit 2 + usage", nil
	})

	r.test("Exports proxy env, synced TZ & enters sandbox", func() (string, error) {
		out, errOut, rc := execRunner(15*time.Second, nil, "", "", r.exe, "_probe", "env-check")
		if rc != 0 {
			return "", fmt.Errorf("runner failed: %s", strings.TrimSpace(errOut))
		}
		var data map[string]string
		if err := json.Unmarshal([]byte(strings.TrimSpace(out)), &data); err != nil {
			return "", err
		}
		if runtime.GOOS != "windows" && !strings.Contains(data["cg"], config.SliceUnit) {
			return "", fmt.Errorf("not in %s: %s", config.SliceUnit, data["cg"])
		}
		if data["HTTP_PROXY"] != fmt.Sprintf("http://127.0.0.1:%d", config.HTTPPort) ||
			data["HTTPS_PROXY"] != fmt.Sprintf("http://127.0.0.1:%d", config.HTTPPort) ||
			data["ALL_PROXY"] != fmt.Sprintf("socks5h://127.0.0.1:%d", config.SocksPort) {
			return "", fmt.Errorf("unexpected proxy env: %v", data)
		}
		if !strings.Contains(data["NO_PROXY"], "127.0.0.1") || data["TZ"] == "" {
			return "", fmt.Errorf("missing NO_PROXY or TZ: %v", data)
		}
		return fmt.Sprintf("TZ=%s, proxy env verified", data["TZ"]), nil
	})

	r.test("Nested runner invocation works without looping", func() (string, error) {
		var nested []string
		if runtime.GOOS == "windows" {
			nested = append(append([]string{}, runCmdPrefix...), "cmd.exe", "/c", "echo", "NESTED_OK")
		} else {
			nested = append(append([]string{}, runCmdPrefix...), "echo", "NESTED_OK")
		}
		out, errOut, rc := execRunner(15*time.Second, nil, "", "", nested...)
		if rc != 0 || strings.TrimSpace(out) != "NESTED_OK" {
			return "", fmt.Errorf("nested failed: rc=%d out=%q err=%q", rc, out, errOut)
		}
		return "nested run succeeded cleanly", nil
	})

	r.test("Preserves exit codes, quoted args, PWD & stdin", func() (string, error) {
		var rc int
		if runtime.GOOS == "windows" {
			_, _, rc = execRunner(15*time.Second, nil, "", "", "cmd.exe", "/c", "exit", "/b", "42")
		} else {
			_, _, rc = execRunner(15*time.Second, nil, "", "", "sh", "-c", "exit 42")
		}
		if rc != 42 {
			return "", fmt.Errorf("expected rc=42, got %d", rc)
		}
		td, err := os.MkdirTemp("", "harpo-pwd-*")
		if err != nil {
			return "", err
		}
		defer os.RemoveAll(td)
		out, errOut, rc2 := execRunner(15*time.Second, nil, td, "hello-stdin\n", r.exe, "_probe", "echo-args", "a b", `c"d`)
		if rc2 != 0 {
			return "", fmt.Errorf("echo-args failed: %s", errOut)
		}
		var data struct {
			Cwd   string   `json:"cwd"`
			Args  []string `json:"args"`
			Stdin string   `json:"stdin"`
		}
		if err := json.Unmarshal([]byte(strings.TrimSpace(out)), &data); err != nil {
			return "", err
		}
		if !strings.EqualFold(filepath.Clean(data.Cwd), filepath.Clean(td)) || len(data.Args) != 2 || data.Args[0] != "a b" || data.Args[1] != `c"d` || data.Stdin != "hello-stdin" {
			return "", fmt.Errorf("mismatch: %+v", data)
		}
		return "exit code 42, args with spaces/quotes, PWD & stdin preserved", nil
	})

	withFakeHome := func(setup func(cfgDir string) error) (string, string, int, string, error) {
		fakeHome, err := os.MkdirTemp("", "harpo-fakehome-*")
		if err != nil {
			return "", "", -1, "", err
		}
		cfgDir := filepath.Join(fakeHome, ".config", "harpo")
		if setup != nil {
			if err := os.MkdirAll(cfgDir, 0o700); err != nil {
				_ = os.RemoveAll(fakeHome)
				return "", "", -1, "", err
			}
			if err := setup(cfgDir); err != nil {
				_ = os.RemoveAll(fakeHome)
				return "", "", -1, "", err
			}
		}
		var env []string
		if runtime.GOOS == "windows" {
			env = replaceEnv(os.Environ(), "HARPO_CONFIG_DIR", cfgDir)
		} else {
			env = replaceEnv(os.Environ(), "HOME", fakeHome)
			env = removeEnv(env, "HARPO_CONFIG_DIR")
		}
		out, errOut, rc := execRunner(15*time.Second, env, "", "", r.exe, "_probe", "echo-args", "SHOULD_NOT_RUN")
		return out, errOut, rc, fakeHome, nil
	}

	r.test("Fails closed when expected-ip is missing", func() (string, error) {
		out, errOut, rc, fh, err := withFakeHome(nil)
		if fh != "" {
			defer os.RemoveAll(fh)
		}
		if err != nil {
			return "", err
		}
		if rc == 0 || strings.Contains(out, "SHOULD_NOT_RUN") || !strings.Contains(errOut, "Missing") {
			return "", fmt.Errorf("did not fail closed: rc=%d out=%q err=%q", rc, out, errOut)
		}
		return "refused to start (missing expected-ip)", nil
	})

	r.test("Fails closed when expected-ip is empty", func() (string, error) {
		out, errOut, rc, fh, err := withFakeHome(func(d string) error {
			return os.WriteFile(filepath.Join(d, "expected-ip"), []byte("   \n"), 0o600)
		})
		if fh != "" {
			defer os.RemoveAll(fh)
		}
		if err != nil {
			return "", err
		}
		if rc == 0 || strings.Contains(out, "SHOULD_NOT_RUN") || !strings.Contains(errOut, "empty") {
			return "", fmt.Errorf("did not fail closed: rc=%d out=%q err=%q", rc, out, errOut)
		}
		return "refused to start (empty expected-ip)", nil
	})

	r.test("Fails closed when expected-ip mismatches live exit IP", func() (string, error) {
		out, errOut, rc, fh, err := withFakeHome(func(d string) error {
			return os.WriteFile(filepath.Join(d, "expected-ip"), []byte("1.2.3.4\n"), 0o600)
		})
		if fh != "" {
			defer os.RemoveAll(fh)
		}
		if err != nil {
			return "", err
		}
		if rc == 0 || strings.Contains(out, "SHOULD_NOT_RUN") || !strings.Contains(errOut, "mismatch") {
			return "", fmt.Errorf("did not fail closed: rc=%d out=%q err=%q", rc, out, errOut)
		}
		return "refused to start (IP mismatch)", nil
	})

	r.test("Auto-syncs and caches timezone with 0600 permissions", func() (string, error) {
		if expectedIP == "" {
			return "", skip("expected-ip unset")
		}
		fakeHome, err := os.MkdirTemp("", "harpo-tz-*")
		if err != nil {
			return "", err
		}
		defer os.RemoveAll(fakeHome)
		d := filepath.Join(fakeHome, ".config", "harpo")
		if err := os.MkdirAll(d, 0o700); err != nil {
			return "", err
		}
		if err := os.WriteFile(filepath.Join(d, "expected-ip"), []byte(expectedIP+"\n"), 0o600); err != nil {
			return "", err
		}
		var env []string
		if runtime.GOOS == "windows" {
			env = replaceEnv(os.Environ(), "HARPO_CONFIG_DIR", d)
		} else {
			env = removeEnv(replaceEnv(os.Environ(), "HOME", fakeHome), "HARPO_CONFIG_DIR")
		}
		out, errOut, rc := execRunner(15*time.Second, env, "", "", r.exe, "_probe", "env-check")
		if rc != 0 {
			return "", fmt.Errorf("runner failed: %s", errOut)
		}
		var envData map[string]string
		if err := json.Unmarshal([]byte(strings.TrimSpace(out)), &envData); err != nil {
			return "", err
		}
		tz := strings.TrimSpace(envData["TZ"])
		if tz == "" {
			return "", errors.New("TZ empty")
		}
		tzFile := filepath.Join(d, "timezone")
		fi, err := os.Stat(tzFile)
		if err != nil {
			return "", errors.New("timezone cache file not created")
		}
		if runtime.GOOS != "windows" && fi.Mode().Perm() != 0o600 {
			return "", fmt.Errorf("timezone cache mode %04o, expected 0600", fi.Mode().Perm())
		}
		if config.ReadTrimmed(tzFile) != tz {
			return "", errors.New("cached timezone mismatch")
		}
		return fmt.Sprintf("synced TZ=%s, cached", tz), nil
	})

	r.test("claude wrapper works end-to-end with telemetry disabled (optional)", func() (string, error) {
		// Support both harpo-managed apps.json shim and legacy shell wrapper.
		if entry, _ := apps.Find(r.paths, "claude"); entry != nil && entry.Enabled {
			if !apps.HasShim(r.paths, "claude") {
				return "", errors.New("claude is in apps.json but shim symlink is missing (run `harpo shims sync`)")
			}
			for _, k := range []string{
				"CLAUDE_CODE_DISABLE_NONESSENTIAL_TRAFFIC",
				"DISABLE_TELEMETRY",
				"DISABLE_ERROR_REPORTING",
				"DO_NOT_TRACK",
			} {
				if entry.Env[k] != "1" {
					return "", fmt.Errorf("missing %s=1 in harpo apps.json entry for claude", k)
				}
			}
			out, errOut, rc := runCmd(r.ctx, 15*time.Second, nil, "", "", r.exe, "launch", "claude", "--version")
			if rc != 0 || !strings.Contains(out, "Claude Code") {
				return "", fmt.Errorf("harpo launch claude --version failed: %s %s", out, errOut)
			}
			return strings.TrimSpace(out) + " (harpo shim)", nil
		}

		claudeBin, err := exec.LookPath("claude")
		if err != nil {
			return "", skip("claude not installed on this host")
		}
		scriptBytes, err := os.ReadFile(claudeBin)
		if err != nil {
			return "", skip("claude binary is not a readable wrapper script")
		}
		script := string(scriptBytes)
		if !strings.Contains(script, "isp-run") && !strings.Contains(script, "harpo run") {
			return "", skip("claude is not wrapped with isp-run/harpo")
		}
		for _, needle := range []string{
			"CLAUDE_CODE_DISABLE_NONESSENTIAL_TRAFFIC=1",
			"DISABLE_TELEMETRY=1",
			"DISABLE_ERROR_REPORTING=1",
			"DO_NOT_TRACK=1",
		} {
			if !strings.Contains(script, needle) {
				return "", fmt.Errorf("missing %q in claude wrapper", needle)
			}
		}
		out, errOut, rc := runCmd(r.ctx, 15*time.Second, nil, "", "", claudeBin, "--version")
		if rc != 0 || !strings.Contains(out, "Claude Code") {
			return "", fmt.Errorf("claude --version failed: %s %s", out, errOut)
		}
		return strings.TrimSpace(out), nil
	})

	r.test("claude-web wrapper delegates to harpo/isp-run with anti-leak flags (optional)", func() (string, error) {
		if entry, _ := apps.Find(r.paths, "claude-web"); entry != nil && entry.Enabled {
			if entry.Mode != apps.ModeChromium && entry.Mode != apps.ModeElectron {
				return "", fmt.Errorf("expected chromium/electron mode for claude-web, got %s", entry.Mode)
			}
			argv, err := apps.BuildExecArgs(r.paths, *entry, "/usr/bin/chromium", nil)
			if err != nil {
				return "", err
			}
			joined := strings.Join(argv, " ")
			for _, flag := range []string{
				"--proxy-server=socks5://127.0.0.1:10808",
				"--host-resolver-rules=",
				"--force-webrtc-ip-handling-policy=disable_non_proxied_udp",
				"--disable-quic",
				"--user-data-dir=",
			} {
				if !strings.Contains(joined, flag) {
					return "", fmt.Errorf("missing %q in harpo claude-web args", flag)
				}
			}
			return "harpo shim + anti-leak Chromium flags verified", nil
		}

		cw, err := exec.LookPath("claude-web")
		if err != nil {
			return "", skip("claude-web not installed on this host")
		}
		b, err := os.ReadFile(cw)
		if err != nil {
			return "", err
		}
		script := string(b)
		for _, flag := range []string{
			`--proxy-server="socks5://127.0.0.1:10808"`,
			"--host-resolver-rules=",
			"--force-webrtc-ip-handling-policy=disable_non_proxied_udp",
			"--disable-quic",
			"--user-data-dir=",
		} {
			if !strings.Contains(script, flag) {
				return "", fmt.Errorf("missing %q in claude-web wrapper", flag)
			}
		}
		return "isp-run + anti-leak Chromium flags verified", nil
	})
}

// ---------------------------------------------------------------------------
// Suite 6: Parser, Subscription Fetcher, Pre-Flight Probes & Rollback Safety
// ---------------------------------------------------------------------------
func (r *runner) runSuite6() {
	r.setSuite("Suite 6: Parser, Subscription Fetcher & Pre-Flight Probes")
	expectedIP := config.ReadTrimmed(r.paths.ExpectedIP)

	r.test("VLESS parser handles Reality/TLS/WS/gRPC/xHTTP/HTTPUpgrade/IPv6 & passes xray -test", func() (string, error) {
		uris := []string{
			"vless://11111111-2222-3333-4444-555555555555@198.51.100.1:2053?type=tcp&security=reality&pbk=AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA&fp=firefox&sni=example.com&sid=aabbccdd&spx=%2Ftest&flow=xtls-rprx-vision#%F0%9F%A4%96%20Claude%20%E2%80%94%20Main",
			"vless://11111111-2222-3333-4444-555555555555@example.com:443?type=ws&security=tls&sni=example.com&host=ws.example.com&path=%2Fws&alpn=h2,http/1.1&allowInsecure=1#WS-TLS",
			"vless://11111111-2222-3333-4444-555555555555@example.com:8443?type=grpc&security=tls&sni=example.com&serviceName=mygrpc&mode=multi#gRPC",
			"vless://11111111-2222-3333-4444-555555555555@example.com?type=splithttp&security=tls&path=%2Fxhttp&host=cdn.example.com#SplitHTTP",
			"vless://11111111-2222-3333-4444-555555555555@[2001:db8::1]?type=httpupgrade&security=tls&sni=v6.example.com&path=%2Fup#IPv6-Up",
		}
		var parsed []*xray.Profile
		for _, uri := range uris {
			p, err := xray.ParseVLESSURI(uri)
			if err != nil {
				return "", fmt.Errorf("parse %s: %w", uri, err)
			}
			parsed = append(parsed, p)
			cfg := xray.BuildConfig(p.Outbound, 19808, 19809)
			tf, err := os.CreateTemp("", "harpo-unit-*.json")
			if err != nil {
				return "", err
			}
			tfName := tf.Name()
			b, _ := json.Marshal(cfg)
			_, _ = tf.Write(b)
			_ = tf.Close()
			vErr := xray.ValidateConfigFile(r.ctx, r.xrayBin, tfName)
			_ = os.Remove(tfName)
			if vErr != nil {
				return "", vErr
			}
		}
		if parsed[0].Name != "🤖 Claude — Main" || parsed[0].Network != "raw" || parsed[3].Network != "xhttp" || parsed[4].Host != "2001:db8::1" || parsed[4].Port != 443 {
			return "", errors.New("parsed field assertions failed")
		}
		return fmt.Sprintf("%d transport/security variants all validated by xray -test", len(uris)), nil
	})

	r.test("VLESS parser rejects malformed URIs (wrong scheme, missing UUID/host/pbk)", func() (string, error) {
		bad := []string{
			"vmess://abc",
			"vless://@1.2.3.4:443#no-uuid",
			"vless://11111111-2222-3333-4444-555555555555@:443#no-host",
			"vless://11111111-2222-3333-4444-555555555555@1.2.3.4:443?security=reality#no-pbk",
		}
		for _, uri := range bad {
			if _, err := xray.ParseVLESSURI(uri); err == nil {
				return "", fmt.Errorf("expected error for %q", uri)
			}
		}
		return fmt.Sprintf("%d malformed URIs cleanly rejected", len(bad)), nil
	})

	r.test("Subscription decoder handles plain, unpadded/wrapped/urlsafe base64 & garbage", func() (string, error) {
		sample := "vless://11111111-2222-3333-4444-555555555555@1.2.3.4:443#A\nhysteria2://x@1.2.3.4:443#B\n"
		if len(sub.DecodeBody(sample)) != 2 {
			return "", errors.New("plain decode failed")
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
		if len(sub.DecodeBody(wrapped.String())) != 2 {
			return "", errors.New("wrapped b64 decode failed")
		}
		ub64 := strings.TrimRight(base64.URLEncoding.EncodeToString([]byte(sample)), "=")
		if len(sub.DecodeBody(ub64)) != 2 {
			return "", errors.New("urlsafe b64 decode failed")
		}
		if len(sub.DecodeBody("<html>404 Not Found</html>")) != 0 || len(sub.DecodeBody("")) != 0 {
			return "", errors.New("garbage not rejected")
		}
		return "plain, unpadded b64, wrapped b64, urlsafe b64 & garbage handled", nil
	})

	r.test("sub.Fetch parses mock HTTP base64 sub & counts skipped protocols", func() (string, error) {
		sample := "vless://11111111-2222-3333-4444-555555555555@198.51.100.1:443?security=tls&sni=a.test#One\n" +
			"vless://11111111-2222-3333-4444-555555555555@198.51.100.2:443?security=reality#MissingPBK\n" +
			"hysteria2://pw@198.51.100.3:443#SkipHysteria\n"
		payload := base64.StdEncoding.EncodeToString([]byte(sample))
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
			w.Header().Set("Content-Type", "text/plain")
			_, _ = w.Write([]byte(payload))
		}))
		defer srv.Close()

		profs, skipped, err := sub.Fetch(r.ctx, srv.URL+"/sub")
		if err != nil {
			return "", err
		}
		if len(profs) != 1 || profs[0].Name != "One" || skipped != 2 {
			return "", fmt.Errorf("got %d profiles skipped=%d", len(profs), skipped)
		}
		return "fetched mock b64 sub: 1 valid VLESS, 2 skipped", nil
	})

	r.test("BuildConfig is non-mutating and enforces single outbound + loopback", func() (string, error) {
		orig := map[string]any{"protocol": "vless", "tag": "orig", "settings": map[string]any{"address": "1.2.3.4", "port": 443, "id": "u"}}
		snap, _ := json.Marshal(orig)
		cfg := xray.BuildConfig(orig, config.SocksPort, config.HTTPPort)
		after, _ := json.Marshal(orig)
		if !bytes.Equal(snap, after) {
			return "", errors.New("BuildConfig mutated input map")
		}
		obs := cfg["outbounds"].([]any)
		if len(obs) != 1 || obs[0].(map[string]any)["tag"] != config.OutboundTag {
			return "", errors.New("unexpected outbounds")
		}
		return "pure, single harpo-exit outbound, 127.0.0.1 inbounds", nil
	})

	r.test("Pre-flight probe rejects invalid & unreachable outbounds with zero leaked temp files", func() (string, error) {
		before, _ := filepath.Glob(filepath.Join(os.TempDir(), "harpo-test-*.json"))
		if _, err := probe.ValidateAndProbeCandidate(r.ctx, r.xrayBin, map[string]any{"protocol": "not-a-protocol", "settings": map[string]any{}}); err == nil {
			return "", errors.New("expected invalid config failure")
		}
		deadPort, _ := xray.FindFreePort()
		deadProf, _ := xray.ParseVLESSURI(fmt.Sprintf("vless://11111111-2222-3333-4444-555555555555@127.0.0.1:%d?security=none#Dead", deadPort))
		pCtx, cancel := context.WithTimeout(r.ctx, 8*time.Second)
		_, err2 := probe.ValidateAndProbeCandidate(pCtx, r.xrayBin, deadProf.Outbound)
		cancel()
		if err2 == nil {
			return "", errors.New("expected unreachable failure")
		}
		after, _ := filepath.Glob(filepath.Join(os.TempDir(), "harpo-test-*.json"))
		if len(before) != len(after) {
			return "", fmt.Errorf("leaked temp files: before=%d after=%d", len(before), len(after))
		}
		return "invalid & unreachable rejected with zero leaked temp files", nil
	})

	r.test("Pre-flight probe succeeds live against current outbound on temp port", func() (string, error) {
		b, err := os.ReadFile(r.paths.CfgPath)
		if err != nil {
			return "", err
		}
		var cfg map[string]any
		if err := json.Unmarshal(b, &cfg); err != nil {
			return "", err
		}
		obs, _ := cfg["outbounds"].([]any)
		if len(obs) == 0 {
			return "", errors.New("no outbound")
		}
		res, err := probe.ValidateAndProbeCandidate(r.ctx, r.xrayBin, obs[0].(map[string]any))
		if err != nil {
			return "", err
		}
		if expectedIP != "" && res.IP != expectedIP {
			return "", fmt.Errorf("got %q, expected %q", res.IP, expectedIP)
		}
		return fmt.Sprintf("temp xray probe == %s (%s)", res.IP, res.Timezone), nil
	})

	r.test("WriteSecretText writes atomically with 0600 mode and no leftovers", func() (string, error) {
		td, err := os.MkdirTemp("", "harpo-secret-*")
		if err != nil {
			return "", err
		}
		defer os.RemoveAll(td)
		target := filepath.Join(td, "secret-test")
		if err := config.WriteSecretText(target, "top-secret\n"); err != nil {
			return "", err
		}
		if got := config.ReadTrimmed(target); got != "top-secret" {
			return "", fmt.Errorf("content mismatch: %q", got)
		}
		fi, err := os.Stat(target)
		if err != nil || (runtime.GOOS != "windows" && fi.Mode().Perm() != 0o600) {
			return "", fmt.Errorf("expected 0600, got %04o", fi.Mode().Perm())
		}
		entries, _ := os.ReadDir(td)
		if len(entries) != 1 {
			return "", fmt.Errorf("leftover temp files: %d", len(entries))
		}
		return "atomic write, zero temp leftovers", nil
	})

	r.test("harpo status CLI succeeds end-to-end", func() (string, error) {
		out, errOut, rc := runCmd(r.ctx, 20*time.Second, nil, "", "", r.exe, "status")
		if rc != 0 {
			return "", fmt.Errorf("harpo status exited %d: %s", rc, errOut)
		}
		if !strings.Contains(out, "MATCH") {
			return "", fmt.Errorf("expected MATCH in status output:\n%s", out)
		}
		if runtime.GOOS != "windows" && !strings.Contains(out, "BLOCKED") {
			return "", fmt.Errorf("expected BLOCKED + MATCH in status output:\n%s", out)
		}
		return "harpo status OK (MATCH)", nil
	})
}

func startEchoServer(network, addr string) (net.Listener, int, error) {
	ln, err := net.Listen(network, addr)
	if err != nil {
		return nil, 0, err
	}
	port := ln.Addr().(*net.TCPAddr).Port
	go func() {
		for {
			c, err := ln.Accept()
			if err != nil {
				return
			}
			go func(conn net.Conn) {
				defer conn.Close()
				_ = conn.SetDeadline(time.Now().Add(2 * time.Second))
				buf := make([]byte, 64)
				n, err := conn.Read(buf)
				if err == nil && n > 0 {
					_, _ = conn.Write(append([]byte("ECHO:"), buf[:n]...))
				}
			}(c)
		}
	}()
	return ln, port, nil
}

func findLocalGateway(ctx context.Context) string {
	out, _, rc := runCmd(ctx, 3*time.Second, nil, "", "", "ip", "route", "show", "default")
	if rc != 0 {
		return ""
	}
	fields := strings.Fields(out)
	for i := 0; i+1 < len(fields); i++ {
		if fields[i] == "via" {
			return fields[i+1]
		}
	}
	return ""
}

func findResolvNameserver() string {
	b, err := os.ReadFile("/etc/resolv.conf")
	if err != nil {
		return ""
	}
	for _, line := range strings.Split(string(b), "\n") {
		fields := strings.Fields(line)
		if len(fields) >= 2 && fields[0] == "nameserver" {
			return fields[1]
		}
	}
	return ""
}

func replaceEnv(env []string, key, val string) []string {
	out := make([]string, 0, len(env)+1)
	replaced := false
	prefix := key + "="
	for _, kv := range env {
		if strings.HasPrefix(kv, prefix) {
			out = append(out, prefix+val)
			replaced = true
		} else {
			out = append(out, kv)
		}
	}
	if !replaced {
		out = append(out, prefix+val)
	}
	return out
}

func removeEnv(env []string, key string) []string {
	out := make([]string, 0, len(env))
	prefix := key + "="
	for _, kv := range env {
		if !strings.HasPrefix(kv, prefix) {
			out = append(out, kv)
		}
	}
	return out
}
