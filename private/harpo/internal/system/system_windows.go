//go:build windows

package system

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
	"time"

	"harpo/internal/apps"
	"harpo/internal/config"
	"harpo/internal/probe"
)

func runHiddenCmd(ctx context.Context, name string, args ...string) (string, error) {
	cmd := exec.CommandContext(ctx, name, args...)
	cmd.SysProcAttr = &syscall.SysProcAttr{HideWindow: true, CreationFlags: 0x08000000}
	out, err := cmd.CombinedOutput()
	return strings.TrimSpace(string(out)), err
}

func runHiddenPS(ctx context.Context, script string) (string, error) {
	return runHiddenCmd(ctx, "powershell.exe", "-NoProfile", "-NonInteractive", "-WindowStyle", "Hidden", "-Command", script)
}

func psQuote(s string) string {
	return "'" + strings.ReplaceAll(s, "'", "''") + "'"
}

func isAdmin(ctx context.Context) bool {
	_, err := runHiddenCmd(ctx, "net", "session")
	return err == nil
}

// ServiceActive reports Windows proxy, logon Scheduled Task, or Windows Firewall kill-switch state.
func ServiceActive(unit string, isUser bool) string {
	switch unit {
	case config.ProxyUnit:
		if probe.IsPortListening(config.HTTPPort) && probe.IsPortListening(config.SocksPort) {
			return "active"
		}
		return "inactive"
	case config.AnchorUnit:
		ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
		defer cancel()
		if _, err := runHiddenCmd(ctx, "schtasks", "/Query", "/TN", config.WinProxyTask); err == nil {
			return "active"
		}
		return "inactive"
	case config.FirewallUnit:
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if probe.CheckKillSwitch(ctx).Blocked {
			return "active"
		}
		return "inactive"
	default:
		return "unknown"
	}
}

// RestartService restarts the managed hidden xray.exe proxy on Windows (zero UAC required).
func RestartService(unit string, isUser bool) error {
	p := config.DefaultPaths()
	switch unit {
	case config.ProxyUnit, config.AnchorUnit:
		stopManagedXray(p)
		if _, err := os.Stat(p.CfgPath); err != nil {
			return fmt.Errorf("missing %s — activate a profile first", p.CfgPath)
		}
		// Give _proxy-daemon (if running) a brief window to respawn automatically; otherwise start xray directly.
		deadline := time.Now().Add(700 * time.Millisecond)
		for time.Now().Before(deadline) {
			if probe.IsPortListening(config.HTTPPort) && probe.IsPortListening(config.SocksPort) {
				return nil
			}
			time.Sleep(100 * time.Millisecond)
		}
		return startDetachedXray(p)
	default:
		return fmt.Errorf("cannot restart %s without elevation", unit)
	}
}

func stopManagedXray(p config.Paths) {
	if raw := config.ReadTrimmed(p.ProxyPID); raw != "" {
		if pid, err := strconv.Atoi(raw); err == nil && pid > 0 {
			ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
			_, _ = runHiddenCmd(ctx, "taskkill", "/F", "/PID", strconv.Itoa(pid), "/T")
			cancel()
		}
		_ = os.Remove(p.ProxyPID)
	}
	// If another xray.exe is still holding our exact config.json, terminate it cleanly via WMI/CIM.
	ctx, cancel := context.WithTimeout(context.Background(), 4*time.Second)
	defer cancel()
	ps := fmt.Sprintf(`
Get-CimInstance Win32_Process -Filter "Name='xray.exe'" -ErrorAction SilentlyContinue |
  Where-Object { $_.CommandLine -like '*%s*' } |
  ForEach-Object { Stop-Process -Id $_.ProcessId -Force -ErrorAction SilentlyContinue }
`, strings.ReplaceAll(p.CfgPath, "'", "''"))
	_, _ = runHiddenPS(ctx, ps)
	time.Sleep(200 * time.Millisecond)
}

func openLogFile(logPath string) (*os.File, error) {
	if err := os.MkdirAll(filepath.Dir(logPath), 0o755); err != nil {
		return nil, err
	}
	if fi, err := os.Stat(logPath); err == nil && fi.Size() > 2<<20 {
		_ = os.Rename(logPath, logPath+".old")
	}
	return os.OpenFile(logPath, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o600)
}

func startDetachedXray(p config.Paths) error {
	xrayBin := config.ResolveXrayBin()
	if _, err := exec.LookPath(xrayBin); err != nil {
		if _, statErr := os.Stat(xrayBin); statErr != nil {
			return fmt.Errorf("xray.exe not found (%s) — place xray.exe in PATH or next to harpo.exe", xrayBin)
		}
	}
	logF, err := openLogFile(p.ProxyLog)
	if err != nil {
		return fmt.Errorf("open log %s: %w", p.ProxyLog, err)
	}
	defer logF.Close()

	_, _ = fmt.Fprintf(logF, "\n--- [%s] harpo starting %s run -c %s ---\n", time.Now().Format(time.RFC3339), xrayBin, p.CfgPath)

	cmd := exec.Command(xrayBin, "run", "-c", p.CfgPath)
	cmd.Stdout = logF
	cmd.Stderr = logF
	// CREATE_NO_WINDOW (0x08000000) | DETACHED_PROCESS (0x00000008)
	cmd.SysProcAttr = &syscall.SysProcAttr{
		HideWindow:    true,
		CreationFlags: 0x08000000 | 0x00000008,
	}
	if err := cmd.Start(); err != nil {
		return fmt.Errorf("start xray.exe: %w", err)
	}
	_ = config.WriteSecretText(p.ProxyPID, fmt.Sprintf("%d\n", cmd.Process.Pid))
	_ = cmd.Process.Release()
	return nil
}

// RunProxyDaemon keeps hidden xray.exe running and auto-restarts it if it crashes.
// Invoked silently at Windows logon by the per-user Scheduled Task `HarpoProxy`.
func RunProxyDaemon(ctx context.Context, p config.Paths) error {
	for {
		select {
		case <-ctx.Done():
			return nil
		default:
		}
		if _, err := os.Stat(p.CfgPath); err != nil {
			time.Sleep(5 * time.Second)
			continue
		}
		if probe.IsPortListening(config.HTTPPort) && probe.IsPortListening(config.SocksPort) {
			time.Sleep(2 * time.Second)
			continue
		}
		xrayBin := config.ResolveXrayBin()
		logF, err := openLogFile(p.ProxyLog)
		if err != nil {
			time.Sleep(5 * time.Second)
			continue
		}
		_, _ = fmt.Fprintf(logF, "\n--- [%s] harpo daemon starting %s ---\n", time.Now().Format(time.RFC3339), xrayBin)
		cmd := exec.CommandContext(ctx, xrayBin, "run", "-c", p.CfgPath)
		cmd.Stdout = logF
		cmd.Stderr = logF
		cmd.SysProcAttr = &syscall.SysProcAttr{
			HideWindow:    true,
			CreationFlags: 0x08000000,
		}
		if err := cmd.Start(); err != nil {
			_, _ = fmt.Fprintf(logF, "[%s] failed to start xray.exe: %v\n", time.Now().Format(time.RFC3339), err)
			_ = logF.Close()
			time.Sleep(5 * time.Second)
			continue
		}
		_ = config.WriteSecretText(p.ProxyPID, fmt.Sprintf("%d\n", cmd.Process.Pid))
		_ = cmd.Wait()
		_ = logF.Close()
		time.Sleep(2 * time.Second)
	}
}

// JournalLogs reads the tail of %LOCALAPPDATA%\harpo\logs\xray.log on Windows.
func JournalLogs(unit string, isUser bool, lines int) string {
	if lines <= 0 {
		lines = 50
	}
	p := config.DefaultPaths()
	b, err := os.ReadFile(p.ProxyLog)
	if err != nil {
		if os.IsNotExist(err) {
			return fmt.Sprintf("No proxy logs yet (%s). Activate a node or press R to start xray.exe.", p.ProxyLog)
		}
		return fmt.Sprintf("Failed to read %s: %v", p.ProxyLog, err)
	}
	all := strings.Split(strings.TrimRight(string(b), "\r\n"), "\n")
	if len(all) > lines {
		all = all[len(all)-lines:]
	}
	return strings.TrimSpace(strings.Join(all, "\n"))
}

// DetectFirewallBackend identifies Windows Defender Firewall (WFP).
func DetectFirewallBackend() string {
	return "Windows Firewall (WFP)"
}

// SetupUserUnits registers the per-user logon Scheduled Task `HarpoProxy` (hidden, zero UAC),
// starts xray.exe if config.json exists, and prepends the shims folder to User PATH.
func SetupUserUnits(p config.Paths) ([]string, error) {
	if err := config.EnsureCfgDir(p.CfgDir); err != nil {
		return nil, err
	}
	if err := os.MkdirAll(p.LogsDir, 0o755); err != nil {
		return nil, err
	}
	if err := os.MkdirAll(apps.ShimsDir(p), 0o755); err != nil {
		return nil, err
	}

	harpoExe := apps.ResolveHarpoTarget(p)
	vbsPath := filepath.Join(p.DataDir, "start-proxy.vbs")
	vbsBody := fmt.Sprintf("CreateObject(\"WScript.Shell\").Run \"\"\"%s\"\" _proxy-daemon\", 0, False\r\n", harpoExe)
	if err := os.WriteFile(vbsPath, []byte(vbsBody), 0o644); err != nil {
		return nil, fmt.Errorf("write %s: %w", vbsPath, err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	taskCmd := fmt.Sprintf(`wscript.exe "%s"`, vbsPath)
	out, err := runHiddenCmd(ctx, "schtasks", "/Create", "/TN", config.WinProxyTask, "/SC", "ONLOGON", "/TR", taskCmd, "/F")
	if err != nil {
		return nil, fmt.Errorf("register Scheduled Task %s: %w (%s)", config.WinProxyTask, err, out)
	}

	written := []string{
		vbsPath,
		"Scheduled Task: " + config.WinProxyTask + " (ONLOGON, hidden, no UAC)",
	}

	// Start proxy immediately if config.json exists and isn't already running.
	if _, err := os.Stat(p.CfgPath); err == nil && !probe.IsPortListening(config.HTTPPort) {
		if startErr := startDetachedXray(p); startErr == nil {
			written = append(written, "Started hidden xray.exe proxy")
		}
	}

	if pathFiles, err := apps.InstallShellPATH(p); err == nil {
		written = append(written, pathFiles...)
	}
	return written, nil
}

// SetupFirewallUnit enables Windows Defender Firewall and applies per-.exe outbound block rules for wrapped apps.
func SetupFirewallUnit(ctx context.Context, targetUser string) (string, error) {
	used, err := ApplyFirewall(ctx, targetUser, "auto")
	if err != nil {
		return "", err
	}
	return "Configured " + used + " (rules persist automatically across Windows reboots).", nil
}

// ApplyFirewall installs per-.exe Windows Defender Firewall outbound block rules for all wrapped apps
// (blocking all IPv4 & IPv6 destinations except 127.0.0.1 and ::1). Pass forceBackend="clear" to remove rules.
func ApplyFirewall(ctx context.Context, targetUser string, forceBackend string) (string, error) {
	if !isAdmin(ctx) {
		return "", errors.New("apply-firewall must be run in an Administrator terminal (Right-click Terminal/PowerShell -> Run as administrator)")
	}

	// Clear existing Harpo rules first (idempotent).
	clearPS := fmt.Sprintf(`Get-NetFirewallRule -DisplayName '%s*' -ErrorAction SilentlyContinue | Remove-NetFirewallRule -ErrorAction SilentlyContinue`, config.WinFirewallPrefix)
	if out, err := runHiddenPS(ctx, clearPS); err != nil {
		return "", fmt.Errorf("clear existing firewall rules: %w (%s)", err, out)
	}

	if strings.EqualFold(strings.TrimSpace(forceBackend), "clear") {
		return "Windows Firewall (all Harpo rules removed)", nil
	}

	// Ensure Windows Defender Firewall is enabled on all profiles.
	_, _ = runHiddenPS(ctx, `Set-NetFirewallProfile -Profile Domain,Public,Private -Enabled True -ErrorAction SilentlyContinue`)

	paths := resolveUserPathsForAdmin(targetUser)
	list, err := apps.Load(paths)
	if err != nil {
		return "", fmt.Errorf("load %s: %w", apps.RegistryPath(paths), err)
	}

	shimsDir := apps.ShimsDir(paths)
	seenExe := map[string]bool{}
	applied := 0
	var skipped []string

	for _, a := range list {
		if !a.WantsFirewall() {
			continue
		}
		realBin, err := apps.ResolveRealBinary(a.TargetCommand(), shimsDir)
		if err != nil {
			skipped = append(skipped, fmt.Sprintf("%s (binary not found)", a.Name))
			continue
		}
		if !strings.EqualFold(filepath.Ext(realBin), ".exe") {
			skipped = append(skipped, fmt.Sprintf("%s (%s is not an .exe — use native .exe or soft mode)", a.Name, filepath.Base(realBin)))
			continue
		}
		key := strings.ToLower(filepath.Clean(realBin))
		if seenExe[key] {
			continue
		}
		seenExe[key] = true

		ruleV4 := config.WinFirewallPrefix + a.Name + " (IPv4)"
		ruleV6 := config.WinFirewallPrefix + a.Name + " (IPv6)"

		// Block EVERY non-loopback IPv4 & IPv6 address; 127.0.0.1 and ::1 stay open for local proxy & dev servers.
		ps := fmt.Sprintf(`
New-NetFirewallRule -DisplayName %s -Description 'Harpo Kill-Switch: blocks direct IPv4 outbound (loopback 127.0.0.1 allowed)' -Direction Outbound -Action Block -Program %s -RemoteAddress @('0.0.0.0-127.0.0.0','127.0.0.2-255.255.255.255') -Profile Any -Enabled True | Out-Null
New-NetFirewallRule -DisplayName %s -Description 'Harpo Kill-Switch: blocks direct IPv6 outbound (loopback ::1 allowed)' -Direction Outbound -Action Block -Program %s -RemoteAddress @('::/128','::2-ffff:ffff:ffff:ffff:ffff:ffff:ffff:ffff') -Profile Any -Enabled True | Out-Null
`, psQuote(ruleV4), psQuote(realBin), psQuote(ruleV6), psQuote(realBin))

		if out, err := runHiddenPS(ctx, ps); err != nil {
			return "", fmt.Errorf("add firewall rule for %s (%s): %w (%s)", a.Name, realBin, err, out)
		}
		applied++
	}

	if applied == 0 {
		note := "Windows Firewall enabled (0 app rules — wrap an .exe app with `harpo wrap <app>` then re-run `harpo apply-firewall`)"
		if len(skipped) > 0 {
			note += "; skipped: " + strings.Join(skipped, ", ")
		}
		return note, nil
	}
	res := fmt.Sprintf("Windows Firewall (%d app(s) / %d outbound block rules)", applied, applied*2)
	if len(skipped) > 0 {
		res += " — skipped: " + strings.Join(skipped, ", ")
	}
	return res, nil
}

func resolveUserPathsForAdmin(targetUser string) config.Paths {
	p := config.DefaultPaths()
	if strings.TrimSpace(targetUser) == "" {
		return p
	}
	// If elevated under a separate Admin account, allow targeting C:\Users\<targetUser>\AppData\Roaming\harpo.
	sysDrive := os.Getenv("SystemDrive")
	if sysDrive == "" {
		sysDrive = "C:"
	}
	candHome := filepath.Join(sysDrive+`\`, "Users", targetUser)
	candCfg := filepath.Join(candHome, "AppData", "Roaming", "harpo")
	if fi, err := os.Stat(candCfg); err == nil && fi.IsDir() {
		p.Home = candHome
		p.CfgDir = candCfg
		p.DataDir = filepath.Join(candHome, "AppData", "Local", "harpo")
		p.CfgPath = filepath.Join(candCfg, "config.json")
		p.ExpectedIP = filepath.Join(candCfg, "expected-ip")
		p.ActiveProfile = filepath.Join(candCfg, "active-profile")
	}
	return p
}
