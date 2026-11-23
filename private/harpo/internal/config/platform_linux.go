//go:build !windows

package config

import (
	"bufio"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

func resolveHome() string {
	home, _ := os.UserHomeDir()
	if envHome := os.Getenv("HOME"); envHome != "" {
		home = envHome
	}
	return home
}

func defaultCfgDir(home string) string {
	if v := os.Getenv("HARPO_CONFIG_DIR"); v != "" {
		return v
	}
	return filepath.Join(home, ".config", "harpo")
}

func defaultDataDir(home string) string {
	return filepath.Join(home, ".local", "share", "harpo")
}

func defaultThroneDB(home string) string {
	if v := os.Getenv("HARPO_THRONE_DB"); v != "" {
		return v
	}
	return filepath.Join(home, ".config", "Throne", "config", "throne.db")
}

// ResolveXrayBin locates the xray binary in a distro-agnostic manner.
func ResolveXrayBin() string {
	if v := strings.TrimSpace(os.Getenv("HARPO_XRAY")); v != "" {
		return v
	}
	if p, err := exec.LookPath("xray"); err == nil {
		return p
	}
	// Fallback: inspect user systemd unit ExecStart.
	systemctl := ResolveBin("HARPO_SYSTEMCTL", "", "systemctl")
	if out, err := exec.Command(systemctl, "--user", "show", "-p", "ExecStart", "--value", ProxyUnit).Output(); err == nil {
		for _, tok := range strings.Fields(string(out)) {
			if strings.HasPrefix(tok, "path=") {
				cand := strings.TrimSuffix(strings.TrimPrefix(tok, "path="), ";")
				if fi, err := os.Stat(cand); err == nil && !fi.IsDir() {
					return cand
				}
			}
		}
	}
	home, _ := os.UserHomeDir()
	for _, cand := range []string{
		"/usr/local/bin/xray",
		"/usr/bin/xray",
		filepath.Join(home, ".local", "bin", "xray"),
	} {
		if fi, err := os.Stat(cand); err == nil && !fi.IsDir() {
			return cand
		}
	}
	return "xray"
}

// DetectDistro reads /etc/os-release to display a clean human-readable distro name.
func DetectDistro() string {
	f, err := os.Open("/etc/os-release")
	if err != nil {
		return "Linux"
	}
	defer f.Close()
	var pretty, name string
	sc := bufio.NewScanner(f)
	for sc.Scan() {
		line := strings.TrimSpace(sc.Text())
		if strings.HasPrefix(line, "PRETTY_NAME=") {
			pretty = strings.Trim(strings.TrimPrefix(line, "PRETTY_NAME="), `"`)
		} else if strings.HasPrefix(line, "NAME=") {
			name = strings.Trim(strings.TrimPrefix(line, "NAME="), `"`)
		}
	}
	if pretty != "" {
		return pretty
	}
	if name != "" {
		return name
	}
	return "Linux"
}
