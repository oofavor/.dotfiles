package main

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"harpo/internal/apps"
	"harpo/internal/config"
	"harpo/internal/probe"
	"harpo/internal/runner"
	"harpo/internal/selftest"
	"harpo/internal/system"
	"harpo/internal/tui"
)

const (
	cReset  = "\033[0m"
	cBold   = "\033[1m"
	cDim    = "\033[2m"
	cRed    = "\033[31m"
	cGreen  = "\033[32m"
	cYellow = "\033[33m"
	cCyan   = "\033[36m"
)

func main() {
	os.Exit(dispatch(os.Args))
}

func dispatch(argv []string) int {
	base := filepath.Base(argv[0])
	if runtime.GOOS == "windows" {
		base = strings.TrimSuffix(strings.ToLower(base), ".exe")
	}

	// 1. Busybox-style `harpo-run` (and legacy `isp-run`) symlink/alias.
	if base == "harpo-run" || base == "isp-run" {
		return runner.Run(argv[1:], true)
	}

	// 2. Busybox-style update-proof app shim (`~/.local/share/harpo/shims/<app> -> harpo`).
	if base != "harpo" && base != ".harpo-wrapped" && !strings.HasPrefix(base, "harpo.") {
		return runner.LaunchApp(base, argv[1:])
	}

	if len(argv) < 2 {
		if err := tui.Run(tui.StartupOptions{}); err != nil {
			fmt.Fprintf(os.Stderr, "harpo: %v\n", err)
			return 1
		}
		return 0
	}

	switch argv[1] {
	case "-h", "--help", "help":
		printUsage()
		return 0

	case "_probe":
		return selftest.RunInternalProbe(argv[2:])

	case "_proxy-daemon":
		if err := system.RunProxyDaemon(context.Background(), config.DefaultPaths()); err != nil {
			fmt.Fprintf(os.Stderr, "harpo _proxy-daemon: %v\n", err)
			return 1
		}
		return 0

	case "run":
		return runner.Run(argv[2:], false)

	case "launch":
		if len(argv) < 3 {
			fmt.Fprintln(os.Stderr, "Usage: harpo launch <app-name> [args...]")
			return 2
		}
		return runner.LaunchApp(argv[2], argv[3:])

	case "wrap":
		return cmdWrap(argv[2:])

	case "unwrap":
		return cmdUnwrap(argv[2:])

	case "apps":
		if len(argv) >= 3 && argv[2] == "--tui" {
			if err := tui.Run(tui.StartupOptions{StartTab: 1}); err != nil {
				fmt.Fprintf(os.Stderr, "harpo: %v\n", err)
				return 1
			}
			return 0
		}
		return cmdListApps()

	case "shims":
		return cmdShims(argv[2:])

	case "status":
		return cmdStatus()

	case "probe":
		return cmdProbe(argv[2:])

	case "sub":
		subURL := ""
		if len(argv) >= 3 {
			subURL = argv[2]
		}
		if err := tui.Run(tui.StartupOptions{InitialSubURL: subURL, StartTab: 2}); err != nil {
			fmt.Fprintf(os.Stderr, "harpo: %v\n", err)
			return 1
		}
		return 0

	case "uri":
		uri := ""
		if len(argv) >= 3 {
			uri = argv[2]
		}
		if err := tui.Run(tui.StartupOptions{InitialURI: uri, StartTab: 2}); err != nil {
			fmt.Fprintf(os.Stderr, "harpo: %v\n", err)
			return 1
		}
		return 0

	case "test":
		return cmdTest(argv[2:])

	case "setup":
		return cmdSetup()

	case "setup-firewall":
		return cmdSetupFirewall(argv[2:])

	case "apply-firewall":
		return cmdApplyFirewall(argv[2:])

	default:
		fmt.Fprintf(os.Stderr, "harpo: unknown command %q\n\n", argv[1])
		printUsage()
		return 2
	}
}

func printUsage() {
	if config.IsWindows() {
		fmt.Printf(`%s%sharpo%s — Cross-Platform Proxy, App Sandbox & Windows Firewall Kill-Switch Manager

%sUsage:%s
  harpo                                  Launch minimal 5-tab TUI (zero background network traffic)
  harpo status                           Live health check (proxy, firewall, exit IP, ASN, TZ)
  harpo probe [--socks5] [url...]        On-demand domain reachability probe (or all targets.json)
  harpo run [--soft|--strict] [--electron|--chromium] <cmd> [args...]
                                         Run command with fail-closed exit-IP check & proxy env
  harpo wrap <name> [flags]              Wrap an app with an update-proof .cmd shim (+ optional Start Menu .lnk)
    --mode cli|electron|chromium
    --cmd <binary-or-path>               Underlying binary name/path if different from <name>
    --desktop                            Install Start Menu .lnk shortcut in Harpo folder
    --desktop-name <title>               Friendly name for shortcut
    --desktop-icon <path>                Icon path for shortcut
    --isolated-profile                   Pass separate --user-data-dir for Electron/Chromium
    --profile-dir <path>                 Custom --user-data-dir path (supports ~/)
    --default-arg <arg>                  Arg used only when no CLI args are passed (repeatable)
    --env K=V                            Extra env var (repeatable)
    --no-firewall                        Opt this app out of per-.exe Windows Firewall block (soft mode)
  harpo unwrap <name>                    Remove app shim & Start Menu shortcut
  harpo apps [--tui]                     List wrapped apps & resolved binaries
  harpo shims [init|sync]                Add shims dir to User PATH or re-sync all .cmd/.lnk shims
  harpo launch <name> [args...]          Launch a wrapped app by name from apps.json
  harpo sub [url]                        Open Exits workspace and fetch VLESS subscription URL
  harpo uri [vless://...]                Open Exits workspace and pre-flight probe a VLESS URI
  harpo test [--fast] [-v]               Run comprehensive 6-suite verification suite
  harpo setup                            Register hidden HarpoProxy logon Scheduled Task + User PATH (no Admin)
  harpo setup-firewall                   (Admin) Enable Windows Firewall & apply per-.exe outbound block rules
  harpo apply-firewall [--clear]         (Admin) Sync (or --clear) Windows Firewall kill-switch rules
`, cBold, cCyan, cReset, cBold, cReset)
		return
	}

	fmt.Printf(`%s%sharpo%s — Distro-agnostic Proxy, App Sandbox & Kernel Kill-Switch Manager

%sUsage:%s
  harpo                                  Launch minimal 5-tab TUI (zero background network traffic)
  harpo status                           Live health check (kill-switch, exit IP, ASN, TZ)
  harpo probe [--socks5] [url...]        On-demand domain reachability probe (or all targets.json)
  harpo run [--electron|--chromium|--proxychains] <cmd> [args...]
                                         Run command inside kill-switched cgroup + proxy (alias: harpo-run)
  harpo wrap <name> [flags]              Wrap an app with an update-proof PATH shim (+ optional .desktop)
    --mode cli|electron|chromium|proxychains
    --cmd <binary>                       Underlying binary name if different from <name>
    --desktop                            Install XDG override in ~/.local/share/applications/
    --desktop-name <title>               Friendly name for .desktop launcher
    --desktop-icon <icon>                Icon name for .desktop launcher
    --isolated-profile                   Pass separate --user-data-dir for Electron/Chromium
    --profile-dir <path>                 Custom --user-data-dir path (supports ~/)
    --default-arg <arg>                  Arg used only when no CLI args are passed (repeatable)
    --env K=V                            Extra env var (repeatable)
  harpo unwrap <name>                    Remove app shim & managed .desktop override
  harpo apps [--tui]                     List wrapped apps & resolved binaries
  harpo shims [init|sync]                Install shell PATH hooks or re-sync all shims & .desktop files
  harpo launch <name> [args...]          Launch a wrapped app by name from apps.json
  harpo sub [url]                        Open Exits workspace and fetch VLESS subscription URL
  harpo uri [vless://...]                Open Exits workspace and pre-flight probe a VLESS URI
  harpo test [--fast] [-v]               Run comprehensive 6-suite kill-switch & proxy test suite
  harpo setup                            Install distro-agnostic systemd user units
  sudo harpo setup-firewall --user U     Install & enable system kill-switch firewall service
  sudo harpo apply-firewall --user U     Apply kernel cgroup v2 kill-switch rules (nftables/iptables)
`, cBold, cCyan, cReset, cBold, cReset)
}

func cmdWrap(args []string) int {
	if len(args) == 0 || args[0] == "-h" || args[0] == "--help" {
		fmt.Fprintln(os.Stderr, "Usage: harpo wrap <name> [--mode cli|electron|chromium|proxychains] [--cmd <bin>] [--desktop] [--isolated-profile] [--env K=V]")
		return 2
	}
	entry := apps.AppEntry{
		Name:    args[0],
		Mode:    apps.ModeCLI,
		Enabled: true,
		Env:     map[string]string{},
	}
	for i := 1; i < len(args); i++ {
		switch args[i] {
		case "--mode", "-m":
			if i+1 < len(args) {
				entry.Mode = apps.AppMode(strings.ToLower(args[i+1]))
				i++
			}
		case "--cmd", "-c":
			if i+1 < len(args) {
				entry.Command = args[i+1]
				i++
			}
		case "--desktop", "-d":
			entry.DesktopEntry = true
		case "--desktop-name":
			if i+1 < len(args) {
				entry.DesktopName = args[i+1]
				i++
			}
		case "--desktop-icon":
			if i+1 < len(args) {
				entry.DesktopIcon = args[i+1]
				i++
			}
		case "--isolated-profile":
			entry.IsolatedProfile = true
		case "--profile-dir":
			if i+1 < len(args) {
				entry.ProfileDir = args[i+1]
				entry.IsolatedProfile = true
				i++
			}
		case "--default-arg":
			if i+1 < len(args) {
				entry.DefaultArgs = append(entry.DefaultArgs, args[i+1])
				i++
			}
		case "--no-firewall":
			entry.NoFirewall = true
		case "--env", "-e":
			if i+1 < len(args) {
				if k, v, ok := strings.Cut(args[i+1], "="); ok && k != "" {
					entry.Env[k] = v
				}
				i++
			}
		case "--":
			entry.ExtraArgs = append(entry.ExtraArgs, args[i+1:]...)
			i = len(args)
		default:
			entry.ExtraArgs = append(entry.ExtraArgs, args[i])
		}
	}
	if len(entry.Env) == 0 {
		entry.Env = nil
	}
	if !apps.ValidMode(entry.Mode) {
		fmt.Fprintf(os.Stderr, "%sInvalid mode %q (use cli, electron, chromium, or proxychains)%s\n", cRed, entry.Mode, cReset)
		return 2
	}

	paths := config.DefaultPaths()
	if err := apps.Upsert(paths, entry); err != nil {
		fmt.Fprintf(os.Stderr, "%sFailed to wrap %s:%s %v\n", cRed, entry.Name, cReset, err)
		return 1
	}

	realBin, err := apps.ResolveRealBinary(entry.TargetCommand(), apps.ShimsDir(paths))
	fmt.Printf("%sWrapped '%s'%s (mode=%s)\n", cGreen, entry.Name, cReset, entry.Mode)
	fmt.Printf("  Shim:        %s\n", apps.ShimPath(paths, entry.Name))
	if err == nil {
		fmt.Printf("  Real binary: %s\n", realBin)
	} else {
		fmt.Printf("  Real binary: %snot currently in PATH (%s)%s\n", cYellow, entry.TargetCommand(), cReset)
	}
	if entry.DesktopEntry {
		fmt.Printf("  Launcher:    %s\n", apps.DesktopPath(paths, entry.Name))
	}
	if inPath, _ := apps.IsShimsInPATH(paths); !inPath {
		if config.IsWindows() {
			fmt.Printf("\n%sNote:%s Run `harpo shims init` (or `harpo setup`) to add %s to your User PATH.\n", cYellow, cReset, apps.ShimsDir(paths))
		} else {
			fmt.Printf("\n%sNote:%s Add shims to the front of PATH once in your shell profile:\n  export PATH=\"%s:$PATH\"\n", cYellow, cReset, apps.ShimsDir(paths))
		}
	}
	return 0
}

func cmdUnwrap(args []string) int {
	if len(args) == 0 {
		fmt.Fprintln(os.Stderr, "Usage: harpo unwrap <name>")
		return 2
	}
	paths := config.DefaultPaths()
	if err := apps.Remove(paths, args[0]); err != nil {
		fmt.Fprintf(os.Stderr, "%sError:%s %v\n", cRed, cReset, err)
		return 1
	}
	fmt.Printf("%sUnwrapped '%s'%s (removed shim & managed desktop entry).\n", cYellow, args[0], cReset)
	return 0
}

func cmdListApps() int {
	paths := config.DefaultPaths()
	list, err := apps.Load(paths)
	if err != nil {
		fmt.Fprintf(os.Stderr, "%sError:%s %v\n", cRed, cReset, err)
		return 1
	}
	inPath, _ := apps.IsShimsInPATH(paths)
	fmt.Printf("\n%s=== Harpo Wrapped Apps (%s) ===%s\n", cBold, apps.RegistryPath(paths), cReset)
	if !inPath {
		fmt.Printf("  %sWarning:%s %s is not in PATH yet.\n", cYellow, cReset, apps.ShimsDir(paths))
	}
	if len(list) == 0 {
		fmt.Printf("  %s(none — run `harpo wrap <app>` or open `harpo` -> Apps)%s\n", cDim, cReset)
		return 0
	}
	shims := apps.ShimsDir(paths)
	for _, a := range list {
		state := cGreen + "●" + cReset
		if !a.Enabled {
			state = cDim + "○ paused" + cReset
		}
		realBin, err := apps.ResolveRealBinary(a.TargetCommand(), shims)
		if err != nil {
			realBin = cRed + "(not in PATH)" + cReset
		}
		desk := ""
		if apps.HasDesktopOverride(paths, a.Name) {
			desk = " [.desktop]"
		}
		fmt.Printf("  %s %-18s %-11s -> %s%s\n", state, a.Name, "["+string(a.Mode)+"]", realBin, desk)
	}
	return 0
}

func cmdShims(args []string) int {
	paths := config.DefaultPaths()
	if len(args) > 0 && args[0] == "init" {
		updated, err := apps.InstallShellPATH(paths)
		if err != nil {
			fmt.Fprintf(os.Stderr, "%sFailed to configure PATH:%s %v\n", cRed, cReset, err)
			return 1
		}
		fmt.Printf("%sConfigured automatic PATH hooks for %s:%s\n", cGreen, apps.ShimsDir(paths), cReset)
		for _, f := range updated {
			fmt.Printf("  - %s\n", f)
		}
		fmt.Printf("\nOpen a new shell (or reload your shell config) for PATH changes to apply.\n")
		return 0
	}
	if len(args) > 0 && args[0] == "sync" {
		if err := apps.SyncAll(paths); err != nil {
			fmt.Fprintf(os.Stderr, "%sSync failed:%s %v\n", cRed, cReset, err)
			return 1
		}
		fmt.Printf("%sSynchronized all shims in %s%s\n", cGreen, apps.ShimsDir(paths), cReset)
		return 0
	}
	fmt.Printf("Shims directory: %s\n", apps.ShimsDir(paths))
	if inPath, pos := apps.IsShimsInPATH(paths); inPath {
		fmt.Printf("PATH status:     %sin PATH (position %d)%s\n", cGreen, pos, cReset)
	} else {
		fmt.Printf("PATH status:     %sNOT in PATH%s\n", cYellow, cReset)
		fmt.Printf("  Run `harpo shims init` (or `harpo setup`) to install PATH hooks automatically.\n")
	}
	return 0
}

func cmdStatus() int {
	paths := config.DefaultPaths()
	s := system.LoadSummary(paths)
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	fmt.Printf("\n%s=== Harpo Proxy & Kill-Switch Health ===%s\n", cBold, cReset)
	fmt.Printf("  OS / FW:         %s · %s\n", s.Distro, s.FirewallKind)
	fmt.Printf("  Active profile:  %s%s%s (%s · %s/%s · SNI %s)\n", cCyan, s.Name, cReset, s.Server, s.Network, s.Security, s.SNI)
	fmt.Printf("  harpo-proxy:     %s\n", colorState(s.ProxyState))
	if config.IsWindows() {
		fmt.Printf("  logon task:      %s\n", colorState(s.AnchorState))
		fmt.Printf("  win firewall:    %s\n", colorState(s.FirewallState))
	} else {
		fmt.Printf("  cgroup anchor:   %s\n", colorState(s.AnchorState))
		fmt.Printf("  kill-switch svc: %s\n", colorState(s.FirewallState))
	}
	fmt.Printf("  expected-ip:     %s\n", s.ExpectedIP)
	fmt.Printf("  cached TZ:       %s\n", s.CachedTZ)

	ks := probe.CheckKillSwitch(ctx)
	if ks.Blocked {
		fmt.Printf("  kill-switch:     %sBLOCKED (%s)%s\n", cGreen, ks.Detail, cReset)
	} else if config.IsWindows() {
		fmt.Printf("  kill-switch:     %sSOFT MODE — %s%s\n", cYellow, ks.Detail, cReset)
	} else {
		fmt.Printf("  kill-switch:     %sBREACH — %s%s\n", cRed, ks.Detail, cReset)
	}

	procs := apps.ListIsolatedProcesses()
	nonAnchor := 0
	for _, p := range procs {
		if !p.IsAnchor {
			nonAnchor++
		}
	}
	fmt.Printf("  sandboxed PIDs:  %d active app process(es)\n", nonAnchor)

	fmt.Printf("%sProbing live exit via 127.0.0.1:%d...%s\n", cDim, config.HTTPPort, cReset)
	res := probe.ProbeHTTPPort(ctx, config.HTTPPort, false)
	if res.IP == "" {
		fmt.Printf("  live exit:       %sFAILED (cannot reach internet via 127.0.0.1:%d)%s\n", cRed, config.HTTPPort, cReset)
		return 1
	}
	match := res.IP == s.ExpectedIP
	col, status := cYellow, "MISMATCH"
	if match {
		col, status = cGreen, "MATCH"
	}
	fmt.Printf("  live exit IP:    %s%s (%s)%s\n", col, res.IP, status, cReset)
	fmt.Printf("  location/ASN:    %s · %s\n", res.Location(), res.Org)
	fmt.Printf("  live timezone:   %s\n", res.Timezone)
	return 0
}

func cmdProbe(args []string) int {
	useSocks5 := false
	var rawTargets []string
	for _, a := range args {
		switch a {
		case "--socks5", "-s":
			useSocks5 = true
		case "-h", "--help":
			fmt.Println("Usage: harpo probe [--socks5] [domain-or-url ...]")
			return 0
		default:
			rawTargets = append(rawTargets, a)
		}
	}

	paths := config.DefaultPaths()
	var targets []probe.DomainTarget
	if len(rawTargets) > 0 {
		for _, raw := range rawTargets {
			u := probe.NormalizeTargetURL(raw)
			targets = append(targets, probe.DomainTarget{Name: raw, URL: u})
		}
	} else {
		targets = probe.LoadTargets(paths)
	}

	portLabel := fmt.Sprintf("HTTP :%d", config.HTTPPort)
	if useSocks5 {
		portLabel = fmt.Sprintf("SOCKS5h :%d", config.SocksPort)
	}
	fmt.Printf("\n%sProbing %d target(s) via %s...%s\n", cBold, len(targets), portLabel, cReset)

	results := make([]probe.DomainProbeResult, len(targets))
	probe.ProbeDomainsConcurrent(context.Background(), targets, useSocks5, func(ev probe.DomainBatchEvent) {
		results[ev.Index] = ev.Result
	})

	failed := 0
	for i, t := range targets {
		r := results[i]
		col := cGreen
		if !r.Healthy {
			failed++
			if r.StatusCode > 0 {
				col = cYellow
			} else {
				col = cRed
			}
		}
		code := ""
		if r.StatusCode > 0 {
			code = fmt.Sprintf("%d ", r.StatusCode)
		}
		proto := r.Proto
		if proto == "" {
			proto = "-"
		}
		fmt.Printf("  %-18s  %s%-18s%s  %-4s  %4dms  %s%s%s\n",
			t.Name, col, code+r.Verdict, cReset, proto, r.RTTMS, cDim, t.URL, cReset)
	}
	if failed > 0 {
		return 1
	}
	return 0
}

func colorState(st string) string {
	if st == "active" {
		return cGreen + st + cReset
	}
	return cRed + st + cReset
}

func cmdTest(args []string) int {
	fast := false
	verbose := false
	for _, a := range args {
		switch a {
		case "--fast":
			fast = true
		case "-v", "--verbose":
			verbose = true
		case "-h", "--help":
			fmt.Println("Usage: harpo test [--fast] [-v]")
			return 0
		}
	}

	sum := selftest.RunAll(context.Background(), selftest.Options{
		Fast:    fast,
		Verbose: verbose,
		OnEvent: func(ev selftest.Event) {
			if ev.SuiteTitle != "" {
				fmt.Printf("\n%s%s==> %s%s\n", cBold, cCyan, ev.SuiteTitle, cReset)
			}
			if ev.Result != nil {
				r := ev.Result
				switch r.Status {
				case selftest.StatusPass:
					suffix := fmt.Sprintf("%s(%dms)%s", cDim, r.DurationMS, cReset)
					if r.Detail != "" {
						suffix = fmt.Sprintf("%s(%s, %dms)%s", cDim, r.Detail, r.DurationMS, cReset)
					}
					fmt.Printf("  %sPASS%s  %s %s\n", cGreen, cReset, r.Name, suffix)
				case selftest.StatusSkip:
					fmt.Printf("  %sSKIP%s  %s %s(%s)%s\n", cYellow, cReset, r.Name, cDim, r.Detail, cReset)
				case selftest.StatusFail:
					fmt.Printf("  %sFAIL%s  %s %s(%dms)%s\n", cRed, cReset, r.Name, cDim, r.DurationMS, cReset)
					fmt.Printf("        %s%s%s\n", cRed, r.Detail, cReset)
				}
			}
		},
	})

	fmt.Printf("\n%s%s%s\n", cBold, strings.Repeat("=", 64), cReset)
	col := cGreen
	if sum.Failed > 0 {
		col = cRed
	}
	failCol := cDim
	if sum.Failed > 0 {
		failCol = cRed
	}
	skipCol := cDim
	if sum.Skipped > 0 {
		skipCol = cYellow
	}
	fmt.Printf("%sResults:%s %s%d/%d passed%s, %s%d failed%s, %s%d skipped%s\n",
		cBold, cReset, col, sum.Passed, sum.Total, cReset,
		failCol, sum.Failed, cReset, skipCol, sum.Skipped, cReset)
	if sum.Failed > 0 {
		fmt.Printf("\n%s%sFailed tests:%s\n", cRed, cBold, cReset)
		for _, r := range sum.Results {
			if r.Status == selftest.StatusFail {
				fmt.Printf("  - [%s] %s: %s\n", r.Suite, r.Name, r.Detail)
			}
		}
		return 1
	}
	return 0
}

func cmdSetup() int {
	paths := config.DefaultPaths()
	files, err := system.SetupUserUnits(paths)
	if err != nil {
		fmt.Fprintf(os.Stderr, "%sSetup failed:%s %v\n", cRed, cReset, err)
		return 1
	}
	if config.IsWindows() {
		fmt.Printf("%sConfigured Harpo Windows proxy task & User PATH:%s\n", cGreen, cReset)
	} else {
		fmt.Printf("%sInstalled user systemd units:%s\n", cGreen, cReset)
	}
	for _, f := range files {
		fmt.Printf("  - %s\n", f)
	}
	if system.ServiceActive(config.FirewallUnit, false) == "active" {
		fmt.Printf("\n%sKill-switch firewall is already active.%s\n", cGreen, cReset)
		return 0
	}
	if config.IsWindows() {
		fmt.Printf("\nOptional (run in an Administrator terminal) to enable per-.exe Windows Firewall kill-switch:\n  %sharpo setup-firewall%s\n", cCyan, cReset)
		return 0
	}
	uname := os.Getenv("USER")
	if uname == "" {
		uname = "$USER"
	}
	fmt.Printf("\nNext step (once as root) to install the kernel kill-switch firewall:\n  %ssudo harpo setup-firewall --user %s%s\n", cCyan, uname, cReset)
	return 0
}

func parseUserAndBackendFlags(args []string) (string, string) {
	var userArg, backendArg string
	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--user", "-u":
			if i+1 < len(args) {
				userArg = args[i+1]
				i++
			}
		case "--backend", "-b":
			if i+1 < len(args) {
				backendArg = args[i+1]
				i++
			}
		case "--clear":
			backendArg = "clear"
		}
	}
	return userArg, backendArg
}

func cmdSetupFirewall(args []string) int {
	userArg, _ := parseUserAndBackendFlags(args)
	msg, err := system.SetupFirewallUnit(context.Background(), userArg)
	if err != nil {
		fmt.Fprintf(os.Stderr, "%sError:%s %v\n", cRed, cReset, err)
		return 1
	}
	fmt.Printf("%s%s%s\n", cGreen, msg, cReset)
	return 0
}

func cmdApplyFirewall(args []string) int {
	userArg, backendArg := parseUserAndBackendFlags(args)
	used, err := system.ApplyFirewall(context.Background(), userArg, backendArg)
	if err != nil {
		fmt.Fprintf(os.Stderr, "%sError:%s %v\n", cRed, cReset, err)
		return 1
	}
	fmt.Printf("%sApplied kernel kill-switch rules via %s.%s\n", cGreen, used, cReset)
	return 0
}
