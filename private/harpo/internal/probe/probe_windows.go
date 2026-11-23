//go:build windows

package probe

import (
	"context"
	"fmt"
	"net"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
	"time"

	"harpo/internal/config"
)

func runHiddenPowerShell(ctx context.Context, script string) (string, error) {
	cCtx, cancel := context.WithTimeout(ctx, 6*time.Second)
	defer cancel()
	cmd := exec.CommandContext(cCtx, "powershell.exe", "-NoProfile", "-NonInteractive", "-WindowStyle", "Hidden", "-Command", script)
	cmd.SysProcAttr = &syscall.SysProcAttr{HideWindow: true, CreationFlags: 0x08000000}
	out, err := cmd.CombinedOutput()
	return strings.TrimSpace(string(out)), err
}

// CheckKillSwitch verifies Windows Defender Firewall is enabled and Harpo outbound block rules are active.
func CheckKillSwitch(ctx context.Context) KillSwitchStatus {
	// Single fast PowerShell call: checks firewall profiles + counts active Harpo outbound block rules.
	ps := fmt.Sprintf(`
$svc = Get-Service -Name mpssvc -ErrorAction SilentlyContinue
if (-not $svc -or $svc.Status -ne 'Running') { Write-Output 'SVC_OFF'; exit 0 }
$profs = @(Get-NetFirewallProfile -ErrorAction SilentlyContinue | Where-Object { $_.Enabled -eq $true })
if ($profs.Count -eq 0) { Write-Output 'FW_OFF'; exit 0 }
$rules = @(Get-NetFirewallRule -DisplayName '%s*' -ErrorAction SilentlyContinue | Where-Object { $_.Enabled -eq 'True' -and $_.Direction -eq 'Outbound' -and $_.Action -eq 'Block' })
Write-Output ("OK:" + $rules.Count + ":" + $profs.Count)
`, config.WinFirewallPrefix)

	out, err := runHiddenPowerShell(ctx, ps)
	if err != nil {
		return KillSwitchStatus{Blocked: false, Detail: "unable to query Windows Firewall", Err: err}
	}
	for _, line := range strings.Split(out, "\n") {
		line = strings.TrimSpace(line)
		switch {
		case line == "SVC_OFF":
			return KillSwitchStatus{Blocked: false, Detail: "Windows Defender Firewall service (mpssvc) is stopped"}
		case line == "FW_OFF":
			return KillSwitchStatus{Blocked: false, Detail: "Windows Defender Firewall is disabled on all profiles"}
		case strings.HasPrefix(line, "OK:"):
			parts := strings.Split(line, ":")
			ruleCount := 0
			if len(parts) >= 2 {
				ruleCount, _ = strconv.Atoi(parts[1])
			}
			if ruleCount <= 0 {
				return KillSwitchStatus{
					Blocked: false,
					Detail:  "Windows Firewall active, 0 app rules (run Admin: harpo apply-firewall)",
				}
			}
			return KillSwitchStatus{
				Blocked: true,
				Detail:  fmt.Sprintf("Windows Firewall active (%d outbound block rule(s))", ruleCount),
			}
		}
	}
	return KillSwitchStatus{Blocked: false, Detail: "unexpected firewall query response: " + out}
}

// HasAppFirewallRule checks whether Windows Defender Firewall has an active Harpo outbound block rule for exePath.
func HasAppFirewallRule(ctx context.Context, exePath string) bool {
	if strings.TrimSpace(exePath) == "" {
		return false
	}
	cleanTarget := strings.ToLower(filepath.Clean(exePath))
	if eval, err := filepath.EvalSymlinks(exePath); err == nil {
		cleanTarget = strings.ToLower(filepath.Clean(eval))
	}

	ps := fmt.Sprintf(`
$svc = Get-Service -Name mpssvc -ErrorAction SilentlyContinue
if (-not $svc -or $svc.Status -ne 'Running') { exit 1 }
$profs = @(Get-NetFirewallProfile -ErrorAction SilentlyContinue | Where-Object { $_.Enabled -eq $true })
if ($profs.Count -eq 0) { exit 1 }
$rules = Get-NetFirewallRule -DisplayName '%s*' -ErrorAction SilentlyContinue | Where-Object { $_.Enabled -eq 'True' -and $_.Direction -eq 'Outbound' -and $_.Action -eq 'Block' }
foreach ($r in $rules) {
  $af = Get-NetFirewallApplicationFilter -AssociatedNetFirewallRule $r -ErrorAction SilentlyContinue
  if ($af -and $af.Program) { Write-Output ([Environment]::ExpandEnvironmentVariables($af.Program)) }
}
`, config.WinFirewallPrefix)

	out, err := runHiddenPowerShell(ctx, ps)
	if err != nil {
		return false
	}
	for _, line := range strings.Split(out, "\n") {
		prog := strings.TrimSpace(line)
		if prog == "" {
			continue
		}
		if strings.ToLower(filepath.Clean(prog)) == cleanTarget {
			return true
		}
	}
	return false
}

// ParseProcNetTCP parses listening TCP sockets on Windows via `netstat -ano -p TCP` & `-p TCPv6`.
func ParseProcNetTCP() ([]ListeningSocket, error) {
	var sockets []ListeningSocket
	for _, spec := range []struct {
		proto string
		ipv6  bool
	}{
		{"TCP", false},
		{"TCPv6", true},
	} {
		cmd := exec.Command("netstat", "-ano", "-p", spec.proto)
		cmd.SysProcAttr = &syscall.SysProcAttr{HideWindow: true, CreationFlags: 0x08000000}
		out, err := cmd.Output()
		if err != nil {
			continue
		}
		for _, line := range strings.Split(string(out), "\n") {
			fields := strings.Fields(line)
			if len(fields) < 4 {
				continue
			}
			// State column can be localized on some Windows installs, but LISTENING is standard;
			// also match foreign address 0.0.0.0:0 / [::]:0 for listening sockets.
			state := strings.ToUpper(fields[3])
			foreign := fields[2]
			if state != "LISTENING" && foreign != "0.0.0.0:0" && foreign != "[::]:0" {
				continue
			}
			host, portStr, err := net.SplitHostPort(fields[1])
			if err != nil {
				continue
			}
			port, err := strconv.Atoi(portStr)
			if err != nil {
				continue
			}
			sockets = append(sockets, ListeningSocket{Addr: host, Port: port, IPv6: spec.ipv6})
		}
	}
	return sockets, nil
}
