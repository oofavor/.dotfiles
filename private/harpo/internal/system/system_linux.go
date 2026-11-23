//go:build !windows

package system

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"os/user"
	"path/filepath"
	"strings"
	"time"

	"harpo/internal/apps"
	"harpo/internal/config"
)

// ServiceActive returns systemd active state ("active", "inactive", "failed", "unknown").
func ServiceActive(unit string, isUser bool) string {
	systemctl := config.ResolveBin("HARPO_SYSTEMCTL", "", "systemctl")
	args := []string{}
	if isUser {
		args = append(args, "--user")
	}
	args = append(args, "is-active", unit)
	out, _ := exec.Command(systemctl, args...).Output()
	s := strings.TrimSpace(string(out))
	if s == "" {
		return "unknown"
	}
	return s
}

// RestartService restarts a systemd user or system service.
func RestartService(unit string, isUser bool) error {
	systemctl := config.ResolveBin("HARPO_SYSTEMCTL", "", "systemctl")
	args := []string{}
	if isUser {
		args = append(args, "--user")
	}
	args = append(args, "restart", unit)
	out, err := exec.Command(systemctl, args...).CombinedOutput()
	if err != nil {
		return fmt.Errorf("systemctl restart %s: %w (%s)", unit, err, strings.TrimSpace(string(out)))
	}
	return nil
}

// JournalLogs fetches recent journalctl lines for a systemd unit.
func JournalLogs(unit string, isUser bool, lines int) string {
	if lines <= 0 {
		lines = 50
	}
	journalctl := config.ResolveBin("HARPO_JOURNALCTL", "", "journalctl")
	args := []string{}
	if isUser {
		args = append(args, "--user")
	}
	args = append(args, "-u", unit, "-n", fmt.Sprintf("%d", lines), "--no-pager", "-o", "short-iso")
	out, err := exec.Command(journalctl, args...).CombinedOutput()
	if err != nil && len(out) == 0 {
		return fmt.Sprintf("Failed to read logs: %v", err)
	}
	return strings.TrimSpace(string(out))
}

// DetectFirewallBackend detects whether nftables or iptables is available on this distro.
func DetectFirewallBackend() string {
	hasNft := hasBin("nft")
	hasIpt := hasBin("iptables")
	hasIp6t := hasBin("ip6tables")
	if hasNft && hasIpt {
		if b, err := os.ReadFile("/proc/modules"); err == nil && strings.Contains(string(b), "xt_cgroup ") {
			return "iptables (xt_cgroup)"
		}
		return "nftables (cgroupv2)"
	}
	if hasNft {
		return "nftables (cgroupv2)"
	}
	if hasIpt && hasIp6t {
		return "iptables (xt_cgroup)"
	}
	if hasIpt {
		return "iptables"
	}
	return "none detected"
}

func hasBin(name string) bool {
	_, err := exec.LookPath(name)
	return err == nil
}

// RunProxyDaemon is a Windows-only helper; on Linux systemd manages harpo-proxy.service directly.
func RunProxyDaemon(ctx context.Context, p config.Paths) error {
	return errors.New("_proxy-daemon is only used on Windows (Linux uses systemd harpo-proxy.service)")
}

// SetupUserUnits installs systemd user units for harpo-isolated.slice, harpo-isolated-anchor, and harpo-proxy.
func SetupUserUnits(p config.Paths) ([]string, error) {
	unitDir := filepath.Join(p.Home, ".config", "systemd", "user")
	if err := os.MkdirAll(unitDir, 0o755); err != nil {
		return nil, err
	}
	xrayBin := config.ResolveXrayBin()
	if abs, err := exec.LookPath(xrayBin); err == nil {
		xrayBin = abs
	}
	sleepBin := "/usr/bin/sleep"
	if s, err := exec.LookPath("sleep"); err == nil {
		sleepBin = s
	}

	units := map[string]string{
		config.SliceUnit: `[Unit]
Description=Network-isolated slice for Harpo sandboxed apps (loopback only)
`,
		config.AnchorUnit: fmt.Sprintf(`[Unit]
Description=Keep %s cgroup inode pinned for Harpo kill-switch

[Service]
Slice=%s
ExecStart=%s infinity
Restart=always
RestartSec=1

[Install]
WantedBy=default.target
`, config.SliceUnit, config.SliceUnit, sleepBin),
		config.ProxyUnit: fmt.Sprintf(`[Unit]
Description=Harpo Xray VLESS proxy (SOCKS5 :%d, HTTP :%d)
After=network-online.target
ConditionPathExists=%%h/.config/harpo/config.json

[Service]
ExecStart=%s run -c %%h/.config/harpo/config.json
Restart=always
RestartSec=3

[Install]
WantedBy=default.target
`, config.SocksPort, config.HTTPPort, xrayBin),
	}

	var written []string
	for _, name := range []string{config.SliceUnit, config.AnchorUnit, config.ProxyUnit} {
		content := units[name]
		dst := filepath.Join(unitDir, name)

		if fi, err := os.Lstat(dst); err == nil && fi.Mode()&os.ModeSymlink != 0 {
			written = append(written, dst+" (externally managed — kept)")
			continue
		}
		if _, err := os.Stat(filepath.Join("/etc/systemd/user", name)); err == nil {
			if _, localErr := os.Lstat(dst); os.IsNotExist(localErr) {
				written = append(written, filepath.Join("/etc/systemd/user", name)+" (system-provided — kept)")
				continue
			}
		}

		if err := os.WriteFile(dst, []byte(content), 0o644); err != nil {
			return written, fmt.Errorf("write %s: %w", dst, err)
		}
		written = append(written, dst)
	}

	systemctl := config.ResolveBin("HARPO_SYSTEMCTL", "", "systemctl")
	_ = exec.Command(systemctl, "--user", "daemon-reload").Run()
	_ = exec.Command(systemctl, "--user", "enable", "--now", config.AnchorUnit).Run()
	if _, err := os.Stat(p.CfgPath); err == nil {
		_ = exec.Command(systemctl, "--user", "enable", "--now", config.ProxyUnit).Run()
	} else {
		_ = exec.Command(systemctl, "--user", "enable", config.ProxyUnit).Run()
	}

	if pathFiles, err := apps.InstallShellPATH(p); err == nil {
		written = append(written, pathFiles...)
	}

	return written, nil
}

// SetupFirewallUnit installs the system-level harpo-killswitch.service (requires root).
func SetupFirewallUnit(ctx context.Context, targetUser string) (string, error) {
	if os.Geteuid() != 0 {
		return "", errors.New("setup-firewall must be run as root (e.g. sudo harpo setup-firewall --user $USER)")
	}
	if targetUser == "" {
		targetUser = os.Getenv("SUDO_USER")
	}
	if targetUser == "" {
		return "", errors.New("specify --user <username>")
	}
	u, err := user.Lookup(targetUser)
	if err != nil {
		return "", fmt.Errorf("lookup user %q: %w", targetUser, err)
	}
	exe, err := os.Executable()
	if err != nil {
		return "", err
	}
	if resolved, err := filepath.EvalSymlinks(exe); err == nil {
		exe = resolved
	}

	if loginctl, err := exec.LookPath("loginctl"); err == nil {
		_ = exec.CommandContext(ctx, loginctl, "enable-linger", u.Username).Run()
	}

	systemctl := config.ResolveBin("HARPO_SYSTEMCTL", "", "systemctl")
	unitPath := "/etc/systemd/system/" + config.FirewallUnit

	if fi, err := os.Lstat(unitPath); err == nil && fi.Mode()&os.ModeSymlink != 0 {
		if out, err := exec.CommandContext(ctx, systemctl, "restart", config.FirewallUnit).CombinedOutput(); err != nil {
			return "", fmt.Errorf("restart externally managed %s: %w (%s)", config.FirewallUnit, err, strings.TrimSpace(string(out)))
		}
		return unitPath + " is externally managed — kept untouched and restarted.", nil
	}

	unitContent := fmt.Sprintf(`[Unit]
Description=Install kernel cgroup v2 raw OUTPUT kill-switch for %s (%s)
After=network-pre.target systemd-user-sessions.service firewall.service nftables.service
Wants=systemd-user-sessions.service

[Service]
Type=oneshot
RemainAfterExit=yes
ExecStart=%s apply-firewall --user %s

[Install]
WantedBy=multi-user.target
`, config.SliceUnit, u.Username, exe, u.Username)

	if err := os.WriteFile(unitPath, []byte(unitContent), 0o644); err != nil {
		return "", fmt.Errorf("write %s: %w (if /etc/systemd/system is read-only/immutable, manage the unit declaratively or run `sudo harpo apply-firewall --user %s`)", unitPath, err, u.Username)
	}

	if out, err := exec.CommandContext(ctx, systemctl, "daemon-reload").CombinedOutput(); err != nil {
		return "", fmt.Errorf("systemctl daemon-reload: %w (%s)", err, strings.TrimSpace(string(out)))
	}
	if out, err := exec.CommandContext(ctx, systemctl, "enable", "--now", config.FirewallUnit).CombinedOutput(); err != nil {
		return "", fmt.Errorf("enable %s: %w (%s)", config.FirewallUnit, err, strings.TrimSpace(string(out)))
	}
	return "Installed and started " + unitPath, nil
}

// ApplyFirewall installs kernel raw OUTPUT kill-switch rules for harpo-isolated.slice.
func ApplyFirewall(ctx context.Context, targetUser string, forceBackend string) (string, error) {
	if os.Geteuid() != 0 {
		return "", errors.New("apply-firewall must be run as root")
	}
	if targetUser == "" {
		targetUser = os.Getenv("SUDO_USER")
	}
	if targetUser == "" {
		return "", errors.New("specify --user <username>")
	}
	u, err := user.Lookup(targetUser)
	if err != nil {
		return "", fmt.Errorf("lookup user %q: %w", targetUser, err)
	}
	uid := u.Uid
	relPath := fmt.Sprintf("user.slice/user-%s.slice/user@%s.service/harpo.slice/%s", uid, uid, config.SliceUnit)
	cgDir := filepath.Join("/sys/fs/cgroup", relPath)

	systemctl := config.ResolveBin("HARPO_SYSTEMCTL", "", "systemctl")
	_ = exec.CommandContext(ctx, systemctl, "start", fmt.Sprintf("user@%s.service", uid)).Run()
	_ = exec.CommandContext(ctx, systemctl, fmt.Sprintf("--machine=%s@.host", u.Username), "--user", "daemon-reload").Run()
	_ = exec.CommandContext(ctx, systemctl, fmt.Sprintf("--machine=%s@.host", u.Username), "--user", "start", config.AnchorUnit).Run()

	for i := 0; i < 30; i++ {
		if fi, err := os.Stat(cgDir); err == nil && fi.IsDir() {
			break
		}
		time.Sleep(500 * time.Millisecond)
	}
	if fi, err := os.Stat(cgDir); err != nil || !fi.IsDir() {
		return "", fmt.Errorf("cgroup directory %s does not exist (run `harpo setup` as %s first)", cgDir, u.Username)
	}

	backend := strings.ToLower(strings.TrimSpace(forceBackend))
	if backend == "" || backend == "auto" {
		if hasBin("nft") {
			backend = "nftables"
		} else if hasBin("iptables") {
			backend = "iptables"
		} else {
			return "", errors.New("neither nft nor iptables found in PATH")
		}
	}

	switch backend {
	case "nft", "nftables":
		if err := applyNftablesKillSwitch(ctx, relPath); err != nil {
			if forceBackend == "" && hasBin("iptables") {
				if iptErr := applyIptablesKillSwitch(ctx, relPath); iptErr == nil {
					return "iptables (fallback)", nil
				}
			}
			return "", err
		}
		return "nftables", nil
	case "iptables":
		if err := applyIptablesKillSwitch(ctx, relPath); err != nil {
			return "", err
		}
		return "iptables", nil
	default:
		return "", fmt.Errorf("unknown firewall backend %q (use auto, nftables, or iptables)", backend)
	}
}

func applyNftablesKillSwitch(ctx context.Context, relPath string) error {
	level := len(strings.Split(relPath, "/"))
	nftScript := fmt.Sprintf(`table inet harpo_killswitch
delete table inet harpo_killswitch
table inet harpo_killswitch {
	chain output {
		type filter hook output priority raw; policy accept;
		socket cgroupv2 level %d "%s" ip daddr 127.0.0.1 return
		socket cgroupv2 level %d "%s" ip6 daddr ::1 return
		socket cgroupv2 level %d "%s" drop
	}
}
`, level, relPath, level, relPath, level, relPath)

	cmd := exec.CommandContext(ctx, "nft", "-f", "-")
	cmd.Stdin = strings.NewReader(nftScript)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("nft apply failed: %w (%s)", err, strings.TrimSpace(string(out)))
	}
	return nil
}

func applyIptablesKillSwitch(ctx context.Context, relPath string) error {
	if modprobe, err := exec.LookPath("modprobe"); err == nil {
		_ = exec.CommandContext(ctx, modprobe, "xt_cgroup").Run()
	}
	chain := config.IptablesChain
	for _, spec := range []struct {
		tool    string
		loopDst string
	}{
		{"iptables", "127.0.0.1/32"},
		{"ip6tables", "::1/128"},
	} {
		if !hasBin(spec.tool) {
			continue
		}
		if err := exec.CommandContext(ctx, spec.tool, "-w", "5", "-t", "raw", "-N", chain).Run(); err != nil {
			if out, err := exec.CommandContext(ctx, spec.tool, "-w", "5", "-t", "raw", "-F", chain).CombinedOutput(); err != nil {
				return fmt.Errorf("%s flush chain: %w (%s)", spec.tool, err, strings.TrimSpace(string(out)))
			}
		}
		if out, err := exec.CommandContext(ctx, spec.tool, "-w", "5", "-t", "raw", "-A", chain, "-d", spec.loopDst, "-j", "RETURN").CombinedOutput(); err != nil {
			return fmt.Errorf("%s allow loopback: %w (%s)", spec.tool, err, strings.TrimSpace(string(out)))
		}
		if out, err := exec.CommandContext(ctx, spec.tool, "-w", "5", "-t", "raw", "-A", chain, "-j", "DROP").CombinedOutput(); err != nil {
			return fmt.Errorf("%s drop rule: %w (%s)", spec.tool, err, strings.TrimSpace(string(out)))
		}

		if listOut, err := exec.CommandContext(ctx, spec.tool, "-w", "5", "-t", "raw", "-S", "OUTPUT").Output(); err == nil {
			for _, line := range strings.Split(string(listOut), "\n") {
				line = strings.TrimSpace(line)
				isTarget := strings.HasSuffix(line, " -j "+chain) || strings.HasSuffix(line, " -j ISP_KILLSWITCH")
				if isTarget && strings.HasPrefix(line, "-A OUTPUT ") {
					rawFields := strings.Fields(strings.TrimPrefix(line, "-A OUTPUT "))
					delArgs := make([]string, 0, len(rawFields)+6)
					delArgs = append(delArgs, "-w", "5", "-t", "raw", "-D", "OUTPUT")
					for _, f := range rawFields {
						delArgs = append(delArgs, strings.Trim(f, `"`))
					}
					_ = exec.CommandContext(ctx, spec.tool, delArgs...).Run()
				}
			}
		}
		_ = exec.CommandContext(ctx, spec.tool, "-w", "5", "-t", "raw", "-F", "ISP_KILLSWITCH").Run()
		_ = exec.CommandContext(ctx, spec.tool, "-w", "5", "-t", "raw", "-X", "ISP_KILLSWITCH").Run()

		if out, err := exec.CommandContext(ctx, spec.tool, "-w", "5", "-t", "raw", "-I", "OUTPUT", "1", "-m", "cgroup", "--path", relPath, "-j", chain).CombinedOutput(); err != nil {
			return fmt.Errorf("%s insert cgroup jump: %w (%s)", spec.tool, err, strings.TrimSpace(string(out)))
		}
	}
	return nil
}
