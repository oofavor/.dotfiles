//go:build windows

package runner

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"
	"time"

	"harpo/internal/apps"
	"harpo/internal/config"
	"harpo/internal/probe"
)

func fail(msg string) int {
	fmt.Fprintf(os.Stderr, "harpo-run: %s\n", msg)
	// Show a native Windows popup when stderr is not attached to a console (e.g. Start Menu .lnk launch).
	if fi, err := os.Stderr.Stat(); err == nil && (fi.Mode()&os.ModeCharDevice) == 0 {
		safe := strings.ReplaceAll(msg, "'", "''")
		ps := fmt.Sprintf(`(New-Object -ComObject WScript.Shell).Popup('%s', 12, 'Harpo Kill-Switch', 16) | Out-Null`, safe)
		cmd := exec.Command("powershell.exe", "-NoProfile", "-NonInteractive", "-WindowStyle", "Hidden", "-Command", ps)
		cmd.SysProcAttr = &syscall.SysProcAttr{HideWindow: true, CreationFlags: 0x08000000}
		_ = cmd.Start()
	}
	return 1
}

// LaunchApp looks up a wrapped app in %APPDATA%\harpo\apps.json, resolves its real binary
// dynamically (skipping harpo shims), verifies proxy/kill-switch state, and executes it.
func LaunchApp(appName string, userArgs []string) int {
	paths := config.DefaultPaths()
	entry, err := apps.Find(paths, appName)
	if err != nil {
		return fail(fmt.Sprintf("Failed to load apps.json: %v", err))
	}
	if entry == nil {
		return fail(fmt.Sprintf("App %q is not registered in %s", appName, apps.RegistryPath(paths)))
	}

	shimsDir := apps.ShimsDir(paths)
	realBin, err := apps.ResolveRealBinary(entry.TargetCommand(), shimsDir)
	if err != nil {
		return fail(fmt.Sprintf("Cannot find real binary for %q (%s): %v", appName, entry.TargetCommand(), err))
	}

	isGUI := entry.Mode == apps.ModeChromium || entry.Mode == apps.ModeElectron
	if !entry.Enabled {
		return execChild(append([]string{realBin}, userArgs...), os.Environ(), isGUI)
	}

	finalArgv, err := apps.BuildExecArgs(paths, *entry, realBin, userArgs)
	if err != nil {
		return fail(fmt.Sprintf("Failed to prepare launch args for %s: %v", appName, err))
	}

	forceSoft := entry.NoFirewall || os.Getenv("HARPO_SOFT") == "1"
	forceStrict := !entry.NoFirewall && os.Getenv("HARPO_STRICT") == "1"
	return verifyAndExec(paths, finalArgv, entry.Env, forceSoft, forceStrict, isGUI)
}

// RunWithOptions runs an arbitrary command through Harpo's Windows fail-closed verifier.
func RunWithOptions(opts RunOptions) int {
	prog := "harpo run"
	if opts.InvokedAsAlias {
		prog = "harpo-run"
	}
	if len(opts.Args) == 0 {
		fmt.Fprintf(os.Stderr, "Usage: %s [--soft|--strict] [--electron|--chromium] <command> [args...]\n", prog)
		return 2
	}

	paths := config.DefaultPaths()
	realBin, err := apps.ResolveRealBinary(opts.Args[0], apps.ShimsDir(paths))
	if err != nil {
		return fail(fmt.Sprintf("Command not found: %s (%v)", opts.Args[0], err))
	}

	finalArgv := append([]string{realBin}, opts.Args[1:]...)
	isGUI := opts.Mode == apps.ModeChromium || opts.Mode == apps.ModeElectron
	if opts.Mode != "" && opts.Mode != apps.ModeCLI {
		tmpEntry := apps.AppEntry{Name: strings.TrimSuffix(filepath.Base(realBin), filepath.Ext(realBin)), Mode: opts.Mode}
		finalArgv, err = apps.BuildExecArgs(paths, tmpEntry, realBin, opts.Args[1:])
		if err != nil {
			return fail(err.Error())
		}
	}

	forceSoft := opts.ForceSoft || os.Getenv("HARPO_SOFT") == "1"
	forceStrict := opts.ForceStrict || os.Getenv("HARPO_STRICT") == "1"
	return verifyAndExec(paths, finalArgv, opts.ExtraEnv, forceSoft, forceStrict, isGUI)
}

func verifyAndExec(paths config.Paths, finalArgv []string, extraEnv map[string]string, forceSoft, forceStrict, isGUI bool) int {
	// 1. Check Windows Defender Firewall per-exe kill-switch status (hard mode vs soft fallback).
	if !forceSoft {
		fwCtx, fwCancel := context.WithTimeout(context.Background(), 5*time.Second)
		hasRule := probe.HasAppFirewallRule(fwCtx, finalArgv[0])
		fwCancel()
		if !hasRule {
			if forceStrict {
				return fail(fmt.Sprintf("STRICT KILL-SWITCH — no active Windows Firewall block rule for %s. Run elevated: harpo apply-firewall", finalArgv[0]))
			}
			// Dual mode: warn clearly on stderr and fall back to soft proxy-only mode (still fail-closed on exit IP).
			fmt.Fprintf(os.Stderr, "harpo-run: [soft mode] no Windows Firewall rule for %s (run Admin 'harpo apply-firewall' for kernel kill-switch)\n", filepath.Base(finalArgv[0]))
		}
	}

	// 2. Require the dedicated local Harpo proxy (HTTP :10809, SOCKS5 :10808).
	if !probe.IsPortListening(config.HTTPPort) || !probe.IsPortListening(config.SocksPort) {
		return fail(fmt.Sprintf("Harpo proxy (127.0.0.1:%d/%d) is not listening — run `harpo setup` or press R in `harpo`", config.SocksPort, config.HTTPPort))
	}

	// 3. Live end-to-end exit IP verification + IP-synced timezone detection.
	if _, err := os.Stat(paths.ExpectedIP); err != nil {
		return fail(fmt.Sprintf("Missing %s — refusing to start without expected exit IP.", paths.ExpectedIP))
	}
	expectedIP := config.ReadTrimmed(paths.ExpectedIP)
	if expectedIP == "" {
		return fail(fmt.Sprintf("%s is empty — refusing to start.", paths.ExpectedIP))
	}

	ctx, cancel := context.WithTimeout(context.Background(), 12*time.Second)
	res := probe.ProbeHTTPPort(ctx, config.HTTPPort, false)
	cancel()

	if res.IP != expectedIP {
		return fail(fmt.Sprintf("Harpo proxy exit IP mismatch (expected '%s', got '%s') — refusing to start.", expectedIP, res.IP))
	}

	// Validate detected IANA timezone and cache it for fallback.
	resolvedTZ := ""
	if res.Timezone != "" && config.IsValidTimezone(res.Timezone) {
		resolvedTZ = res.Timezone
		if fi, err := os.Stat(paths.CfgDir); err == nil && fi.IsDir() {
			_ = config.WriteSecretText(paths.TimezonePath, resolvedTZ+"\n")
		}
	} else if cached := config.ReadTrimmed(paths.TimezonePath); cached != "" && config.IsValidTimezone(cached) {
		resolvedTZ = cached
	}

	// 4. Build sanitized proxy environment and launch target command.
	httpProxy := fmt.Sprintf("http://127.0.0.1:%d", config.HTTPPort)
	socksProxy := fmt.Sprintf("socks5h://127.0.0.1:%d", config.SocksPort)
	overrides := map[string]string{
		"HTTP_PROXY":  httpProxy,
		"HTTPS_PROXY": httpProxy,
		"ALL_PROXY":   socksProxy,
		"NO_PROXY":    "localhost,127.0.0.1,::1",
	}
	if resolvedTZ != "" {
		overrides["TZ"] = resolvedTZ
	}
	for k, v := range extraEnv {
		if strings.TrimSpace(k) != "" {
			overrides[k] = v
		}
	}

	env := mergeEnv(os.Environ(), overrides)
	return execChild(finalArgv, env, isGUI)
}

func execChild(argv []string, env []string, detachGUI bool) int {
	// Support .cmd / .bat targets seamlessly via cmd.exe /c.
	ext := strings.ToLower(filepath.Ext(argv[0]))
	var cmd *exec.Cmd
	if ext == ".cmd" || ext == ".bat" {
		cmdArgs := append([]string{"/c"}, argv...)
		cmd = exec.Command("cmd.exe", cmdArgs...)
	} else {
		cmd = exec.Command(argv[0], argv[1:]...)
	}
	cmd.Env = env

	if detachGUI {
		cmd.SysProcAttr = &syscall.SysProcAttr{CreationFlags: 0x00000008} // DETACHED_PROCESS
		if err := cmd.Start(); err != nil {
			return fail(fmt.Sprintf("Failed to launch %s: %v", argv[0], err))
		}
		return 0
	}

	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	// Forward Ctrl+C gracefully to the child process group instead of killing harpo prematurely.
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, os.Interrupt)
	defer signal.Stop(sigCh)
	go func() {
		for range sigCh {
			// Windows console shares Ctrl+C with attached child processes automatically.
		}
	}()

	err := cmd.Run()
	if err == nil {
		return 0
	}
	if exitErr, ok := err.(*exec.ExitError); ok {
		return exitErr.ExitCode()
	}
	return fail(fmt.Sprintf("Failed to exec %s: %v", argv[0], err))
}
