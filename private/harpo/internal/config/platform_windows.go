//go:build windows

package config

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"syscall"
)

func resolveHome() string {
	if v := os.Getenv("USERPROFILE"); v != "" {
		return v
	}
	if v := os.Getenv("HOME"); v != "" {
		return v
	}
	home, _ := os.UserHomeDir()
	return home
}

func defaultCfgDir(home string) string {
	if v := os.Getenv("HARPO_CONFIG_DIR"); v != "" {
		return v
	}
	if appData := os.Getenv("APPDATA"); appData != "" {
		return filepath.Join(appData, "harpo")
	}
	return filepath.Join(home, "AppData", "Roaming", "harpo")
}

func defaultDataDir(home string) string {
	if localAppData := os.Getenv("LOCALAPPDATA"); localAppData != "" {
		return filepath.Join(localAppData, "harpo")
	}
	return filepath.Join(home, "AppData", "Local", "harpo")
}

func defaultThroneDB(home string) string {
	if v := os.Getenv("HARPO_THRONE_DB"); v != "" {
		return v
	}
	candidates := []string{}
	if appData := os.Getenv("APPDATA"); appData != "" {
		candidates = append(candidates, filepath.Join(appData, "Throne", "config", "throne.db"))
	}
	if localAppData := os.Getenv("LOCALAPPDATA"); localAppData != "" {
		candidates = append(candidates, filepath.Join(localAppData, "Throne", "config", "throne.db"))
	}
	if exe, err := os.Executable(); err == nil {
		candidates = append(candidates, filepath.Join(filepath.Dir(exe), "config", "throne.db"))
	}
	candidates = append(candidates,
		filepath.Join(home, "Throne", "config", "throne.db"),
		filepath.Join(home, ".config", "Throne", "config", "throne.db"),
	)
	for _, c := range candidates {
		if fi, err := os.Stat(c); err == nil && !fi.IsDir() {
			return c
		}
	}
	if appData := os.Getenv("APPDATA"); appData != "" {
		return filepath.Join(appData, "Throne", "config", "throne.db")
	}
	return filepath.Join(home, "AppData", "Roaming", "Throne", "config", "throne.db")
}

// ResolveXrayBin locates xray.exe on Windows (env override, adjacent to harpo.exe, PATH, or common install folders).
func ResolveXrayBin() string {
	if v := strings.TrimSpace(os.Getenv("HARPO_XRAY")); v != "" {
		return v
	}
	// 1. Sibling xray.exe next to harpo.exe (portable Windows bundle).
	if exe, err := os.Executable(); err == nil {
		sib := filepath.Join(filepath.Dir(exe), "xray.exe")
		if fi, err := os.Stat(sib); err == nil && !fi.IsDir() {
			return sib
		}
	}
	// 2. PATH lookup.
	for _, name := range []string{"xray.exe", "xray"} {
		if p, err := exec.LookPath(name); err == nil {
			return p
		}
	}
	// 3. Common Windows install paths.
	home := resolveHome()
	localAppData := os.Getenv("LOCALAPPDATA")
	appData := os.Getenv("APPDATA")
	progFiles := os.Getenv("ProgramFiles")
	progData := os.Getenv("ProgramData")
	candidates := []string{
		filepath.Join(localAppData, "harpo", "bin", "xray.exe"),
		filepath.Join(localAppData, "harpo", "xray.exe"),
		filepath.Join(localAppData, "Programs", "xray", "xray.exe"),
		filepath.Join(home, "scoop", "shims", "xray.exe"),
		filepath.Join(progData, "chocolatey", "bin", "xray.exe"),
		filepath.Join(progFiles, "xray", "xray.exe"),
		filepath.Join(appData, "Throne", "xray.exe"),
	}
	for _, c := range candidates {
		if c == "" {
			continue
		}
		if fi, err := os.Stat(c); err == nil && !fi.IsDir() {
			return c
		}
	}
	return "xray.exe"
}

// DetectDistro returns a human-readable Windows OS version string.
func DetectDistro() string {
	cmd := exec.Command("reg", "query", `HKLM\SOFTWARE\Microsoft\Windows NT\CurrentVersion`, "/v", "ProductName")
	cmd.SysProcAttr = &syscall.SysProcAttr{HideWindow: true, CreationFlags: 0x08000000}
	prod := ""
	if out, err := cmd.Output(); err == nil {
		for _, line := range strings.Split(string(out), "\n") {
			if idx := strings.Index(line, "REG_SZ"); idx >= 0 {
				prod = strings.TrimSpace(line[idx+len("REG_SZ"):])
				break
			}
		}
	}
	cmdB := exec.Command("reg", "query", `HKLM\SOFTWARE\Microsoft\Windows NT\CurrentVersion`, "/v", "CurrentBuildNumber")
	cmdB.SysProcAttr = &syscall.SysProcAttr{HideWindow: true, CreationFlags: 0x08000000}
	build := ""
	if out, err := cmdB.Output(); err == nil {
		for _, line := range strings.Split(string(out), "\n") {
			if idx := strings.Index(line, "REG_SZ"); idx >= 0 {
				build = strings.TrimSpace(line[idx+len("REG_SZ"):])
				break
			}
		}
	}
	// Windows 11 still reports "Windows 10 ..." in ProductName registry key; build >= 22000 is Windows 11.
	if strings.Contains(prod, "Windows 10") && build >= "22000" && len(build) >= 5 {
		prod = strings.Replace(prod, "Windows 10", "Windows 11", 1)
	}
	if prod != "" && build != "" {
		return prod + " (Build " + build + ")"
	}
	if prod != "" {
		return prod
	}
	return "Windows"
}
