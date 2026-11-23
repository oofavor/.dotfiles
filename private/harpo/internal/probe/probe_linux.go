//go:build !windows

package probe

import (
	"bufio"
	"context"
	"encoding/binary"
	"net"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"time"

	"harpo/internal/config"
)

// CheckKillSwitch verifies that direct outbound TCP to 1.1.1.1:443 is dropped inside harpo-isolated.slice.
func CheckKillSwitch(ctx context.Context) KillSwitchStatus {
	exe, err := os.Executable()
	if err != nil {
		return KillSwitchStatus{Err: err, Detail: "unable to locate harpo binary"}
	}
	systemdRun := config.ResolveBin("HARPO_SYSTEMD_RUN", "", "systemd-run")
	cCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	cmd := exec.CommandContext(
		cCtx,
		systemdRun,
		"--user",
		"--scope",
		"--quiet",
		"--collect",
		"--slice="+config.SliceUnit,
		"--",
		exe,
		"_probe",
		"tcp4",
		"1.1.1.1",
		"443",
		"400",
	)
	out, err := cmd.CombinedOutput()
	outStr := strings.TrimSpace(string(out))
	if err == nil && strings.Contains(outStr, "LEAKED") {
		return KillSwitchStatus{Blocked: false, Detail: "BREACH — direct 1.1.1.1:443 connected!"}
	}
	if strings.Contains(outStr, "BLOCKED:") {
		return KillSwitchStatus{Blocked: true, Detail: strings.TrimPrefix(outStr, "BLOCKED:")}
	}
	// If systemd-run itself failed (e.g. slice doesn't exist), report error clearly.
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok && exitErr.ExitCode() == 42 {
			return KillSwitchStatus{Blocked: true, Detail: "direct outbound dropped"}
		}
		return KillSwitchStatus{Blocked: false, Detail: outStr, Err: err}
	}
	return KillSwitchStatus{Blocked: true, Detail: "direct outbound dropped"}
}

// HasAppFirewallRule reports true on Linux where cgroup v2 covers all processes inside harpo-isolated.slice.
func HasAppFirewallRule(ctx context.Context, exePath string) bool {
	return CheckKillSwitch(ctx).Blocked
}

// ParseProcNetTCP reads listening TCP sockets from /proc/net/tcp and /proc/net/tcp6 natively (no `ss` needed).
func ParseProcNetTCP() ([]ListeningSocket, error) {
	var sockets []ListeningSocket
	parseFile := func(path string, ipv6 bool) error {
		f, err := os.Open(path)
		if err != nil {
			if os.IsNotExist(err) {
				return nil
			}
			return err
		}
		defer f.Close()
		sc := bufio.NewScanner(f)
		first := true
		for sc.Scan() {
			if first {
				first = false
				continue
			}
			fields := strings.Fields(sc.Text())
			if len(fields) < 4 {
				continue
			}
			// State 0A == TCP_LISTEN
			if fields[3] != "0A" {
				continue
			}
			parts := strings.Split(fields[1], ":")
			if len(parts) != 2 {
				continue
			}
			port64, err := strconv.ParseUint(parts[1], 16, 16)
			if err != nil {
				continue
			}
			addr := decodeProcAddr(parts[0], ipv6)
			sockets = append(sockets, ListeningSocket{Addr: addr, Port: int(port64), IPv6: ipv6})
		}
		return sc.Err()
	}
	if err := parseFile("/proc/net/tcp", false); err != nil {
		return nil, err
	}
	if err := parseFile("/proc/net/tcp6", true); err != nil {
		return nil, err
	}
	return sockets, nil
}

func decodeProcAddr(hexStr string, ipv6 bool) string {
	if !ipv6 && len(hexStr) == 8 {
		v, err := strconv.ParseUint(hexStr, 16, 32)
		if err != nil {
			return hexStr
		}
		var b [4]byte
		binary.LittleEndian.PutUint32(b[:], uint32(v))
		return net.IP(b[:]).String()
	}
	if ipv6 && len(hexStr) == 32 {
		var ip [16]byte
		for i := 0; i < 4; i++ {
			v, err := strconv.ParseUint(hexStr[i*8:(i+1)*8], 16, 32)
			if err != nil {
				return hexStr
			}
			binary.LittleEndian.PutUint32(ip[i*4:(i+1)*4], uint32(v))
		}
		return net.IP(ip[:]).String()
	}
	return hexStr
}
