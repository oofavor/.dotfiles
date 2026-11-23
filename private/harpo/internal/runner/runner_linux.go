//go:build !windows

package runner

import (
	"context"
	"fmt"
	"net"
	"os"
	"os/exec"
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
	// Desktop notification if stderr is not a TTY (e.g. GUI app launcher like claude-desktop / claude-web).
	if fi, err := os.Stderr.Stat(); err == nil && (fi.Mode()&os.ModeCharDevice) == 0 {
		if notify, err := exec.LookPath("notify-send"); err == nil {
			_ = exec.Command(notify, "-u", "critical", "Harpo Kill-Switch", msg).Run()
		}
	}
	return 1
}

func inIsolatedSlice() bool {
	b, err := os.ReadFile("/proc/self/cgroup")
	if err != nil {
		return false
	}
	return strings.Contains(string(b), config.SliceUnit)
}

// LaunchApp looks up a wrapped app in ~/.config/harpo/apps.json, resolves its real binary
// dynamically (skipping harpo shims so updates never break), and runs it inside the kill-switch.
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

	// If isolation is disabled for this app, pass through cleanly to the real binary.
	if !entry.Enabled {
		argv := append([]string{realBin}, userArgs...)
		if err := syscall.Exec(realBin, argv, os.Environ()); err != nil {
			return fail(fmt.Sprintf("Failed to exec %s: %v", realBin, err))
		}
		return 0
	}

	// If not yet inside harpo-isolated.slice, enter slice by re-execing `harpo launch <appName> <userArgs...>`.
	if !inIsolatedSlice() {
		return execInSlice("launch", append([]string{appName}, userArgs...), false)
	}

	finalArgv, err := apps.BuildExecArgs(paths, *entry, realBin, userArgs)
	if err != nil {
		return fail(fmt.Sprintf("Failed to prepare launch args for %s: %v", appName, err))
	}

	return verifyAndExec(paths, finalArgv, entry.Env)
}

// RunWithOptions runs an arbitrary command inside harpo-isolated.slice with optional mode flags and env.
func RunWithOptions(opts RunOptions) int {
	prog := "harpo run"
	if opts.InvokedAsAlias {
		prog = "harpo-run"
	}
	if len(opts.Args) == 0 {
		fmt.Fprintf(os.Stderr, "Usage: %s [--electron|--chromium|--proxychains] <command> [args...]\n", prog)
		return 2
	}

	if !inIsolatedSlice() {
		reArgs := opts.Args
		if opts.Mode != "" && opts.Mode != apps.ModeCLI && !opts.InvokedAsAlias {
			reArgs = append([]string{"--" + string(opts.Mode)}, reArgs...)
		}
		return execInSlice("run", reArgs, opts.InvokedAsAlias)
	}

	paths := config.DefaultPaths()
	realBin, err := apps.ResolveRealBinary(opts.Args[0], apps.ShimsDir(paths))
	if err != nil {
		return fail(fmt.Sprintf("Command not found: %s (%v)", opts.Args[0], err))
	}

	finalArgv := append([]string{realBin}, opts.Args[1:]...)
	if opts.Mode != "" && opts.Mode != apps.ModeCLI {
		tmpEntry := apps.AppEntry{Name: filepath.Base(realBin), Mode: opts.Mode}
		finalArgv, err = apps.BuildExecArgs(paths, tmpEntry, realBin, opts.Args[1:])
		if err != nil {
			return fail(err.Error())
		}
	}

	return verifyAndExec(paths, finalArgv, opts.ExtraEnv)
}

func execInSlice(subcmd string, args []string, invokedAsAlias bool) int {
	exe, err := os.Executable()
	if err != nil {
		return fail(fmt.Sprintf("Unable to locate executable: %v", err))
	}
	systemdRun, err := exec.LookPath(config.ResolveBin("HARPO_SYSTEMD_RUN", "", "systemd-run"))
	if err != nil {
		return fail("systemd-run not found in PATH — refusing to start.")
	}
	reexecArgs := []string{
		systemdRun,
		"--user",
		"--scope",
		"--quiet",
		"--collect",
		"--slice=" + config.SliceUnit,
		"--",
		exe,
	}
	baseExe := filepath.Base(exe)
	if !invokedAsAlias || (baseExe != "harpo-run" && baseExe != "isp-run") {
		reexecArgs = append(reexecArgs, subcmd)
	}
	reexecArgs = append(reexecArgs, args...)
	if err := syscall.Exec(systemdRun, reexecArgs, os.Environ()); err != nil {
		return fail(fmt.Sprintf("Failed to exec systemd-run: %v", err))
	}
	return 1
}

func verifyAndExec(paths config.Paths, finalArgv []string, extraEnv map[string]string) int {
	// 1. Paranoid check that we are actually inside the isolated cgroup.
	if !inIsolatedSlice() {
		return fail("Failed to enter " + config.SliceUnit + " — refusing to start.")
	}

	// 2. Live kernel kill-switch self-test: direct outbound TCP MUST be dropped.
	if c, err := net.DialTimeout("tcp4", "1.1.1.1:443", 250*time.Millisecond); err == nil {
		_ = c.Close()
		return fail("KILL-SWITCH BREACH — direct outbound network is NOT blocked! Check: sudo systemctl status " + config.FirewallUnit)
	}

	// 3. Require the dedicated local Harpo proxy (HTTP :10809, SOCKS5 :10808).
	if !probe.IsPortListening(config.HTTPPort) || !probe.IsPortListening(config.SocksPort) {
		return fail(fmt.Sprintf("%s (127.0.0.1:%d/%d) is not listening — check: systemctl --user status %s", config.ProxyUnit, config.SocksPort, config.HTTPPort, config.ProxyUnit))
	}

	// 4. Live end-to-end exit IP verification + IP-synced timezone detection.
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

	// Validate detected IANA timezone and cache it (0600) for fallback.
	resolvedTZ := ""
	if res.Timezone != "" && config.IsValidTimezone(res.Timezone) {
		resolvedTZ = res.Timezone
		if fi, err := os.Stat(paths.CfgDir); err == nil && fi.IsDir() {
			_ = config.WriteSecretText(paths.TimezonePath, resolvedTZ+"\n")
		}
	} else if cached := config.ReadTrimmed(paths.TimezonePath); cached != "" && config.IsValidTimezone(cached) {
		resolvedTZ = cached
	}

	// 5. Build sanitized proxy environment and exec target command.
	httpProxy := fmt.Sprintf("http://127.0.0.1:%d", config.HTTPPort)
	socksProxy := fmt.Sprintf("socks5h://127.0.0.1:%d", config.SocksPort)
	overrides := map[string]string{
		"HTTP_PROXY":  httpProxy,
		"HTTPS_PROXY": httpProxy,
		"http_proxy":  httpProxy,
		"https_proxy": httpProxy,
		"ALL_PROXY":   socksProxy,
		"all_proxy":   socksProxy,
		"NO_PROXY":    "localhost,127.0.0.1,::1",
		"no_proxy":    "localhost,127.0.0.1,::1",
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
	if err := syscall.Exec(finalArgv[0], finalArgv, env); err != nil {
		return fail(fmt.Sprintf("Failed to exec %s: %v", finalArgv[0], err))
	}
	return 0
}
