//go:build windows

package apps

import (
	"encoding/csv"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"syscall"

	"harpo/internal/config"
)

const winShimMarker = "rem Managed by harpo"

func runHiddenPS(script string) (string, error) {
	cmd := exec.Command("powershell.exe", "-NoProfile", "-NonInteractive", "-WindowStyle", "Hidden", "-Command", script)
	cmd.SysProcAttr = &syscall.SysProcAttr{HideWindow: true, CreationFlags: 0x08000000}
	out, err := cmd.CombinedOutput()
	return strings.TrimSpace(string(out)), err
}

func psQuote(s string) string {
	return "'" + strings.ReplaceAll(s, "'", "''") + "'"
}

// ShimPath returns %LOCALAPPDATA%\harpo\shims\<name>.cmd.
func ShimPath(p config.Paths, name string) string {
	return filepath.Join(ShimsDir(p), name+".cmd")
}

// DesktopAppsDir returns the Start Menu Harpo folder (%APPDATA%\Microsoft\Windows\Start Menu\Programs\Harpo).
func DesktopAppsDir(p config.Paths) string {
	appData := os.Getenv("APPDATA")
	if appData == "" {
		appData = filepath.Join(p.Home, "AppData", "Roaming")
	}
	return filepath.Join(appData, "Microsoft", "Windows", "Start Menu", "Programs", "Harpo")
}

// DesktopPath returns the Start Menu .lnk path for name.
func DesktopPath(p config.Paths, name string) string {
	return filepath.Join(DesktopAppsDir(p), name+".lnk")
}

// EnsureShim creates/updates a zero-privilege .cmd shim in %LOCALAPPDATA%\harpo\shims\.
func EnsureShim(p config.Paths, name string) error {
	dir := ShimsDir(p)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	target := ResolveHarpoTarget(p)
	shimPath := ShimPath(p, name)
	if b, err := os.ReadFile(shimPath); err == nil {
		if !strings.Contains(string(b), winShimMarker) {
			return fmt.Errorf("refusing to overwrite unmanaged file at %s", shimPath)
		}
	}
	content := fmt.Sprintf("@echo off\r\n%s — update-proof proxy & kill-switch shim\r\n\"%s\" launch %s %%*\r\nexit /b %%ERRORLEVEL%%\r\n", winShimMarker, target, name)
	return os.WriteFile(shimPath, []byte(content), 0o755)
}

// RemoveShim removes the managed .cmd shim if present.
func RemoveShim(p config.Paths, name string) error {
	shimPath := ShimPath(p, name)
	b, err := os.ReadFile(shimPath)
	if err != nil {
		return nil
	}
	if strings.Contains(string(b), winShimMarker) {
		return os.Remove(shimPath)
	}
	return nil
}

// HasShim reports whether an active managed .cmd shim exists for name.
func HasShim(p config.Paths, name string) bool {
	b, err := os.ReadFile(ShimPath(p, name))
	return err == nil && strings.Contains(string(b), winShimMarker)
}

// IsShimsInPATH checks case-insensitively whether the shims folder is in PATH.
func IsShimsInPATH(p config.Paths) (inPath bool, position int) {
	shims := strings.ToLower(filepath.Clean(ShimsDir(p)))
	for i, dir := range filepath.SplitList(os.Getenv("PATH")) {
		if strings.ToLower(filepath.Clean(dir)) == shims {
			return true, i
		}
	}
	return false, -1
}

// InstallShellPATH prepends %LOCALAPPDATA%\harpo\shims to the Windows User Path (HKCU\Environment) idempotently.
func InstallShellPATH(p config.Paths) ([]string, error) {
	shimsDir := filepath.Clean(ShimsDir(p))
	if err := os.MkdirAll(shimsDir, 0o755); err != nil {
		return nil, err
	}
	ps := fmt.Sprintf(`
$target = %s
$cur = [Environment]::GetEnvironmentVariable('Path', 'User')
$parts = @()
if ($cur) { $parts = $cur -split ';' | Where-Object { $_ -and $_.Trim() -ne '' } }
$filtered = @($parts | Where-Object { $_.TrimEnd('\').ToLowerInvariant() -ne $target.TrimEnd('\').ToLowerInvariant() })
$newPath = (@($target) + $filtered) -join ';'
[Environment]::SetEnvironmentVariable('Path', $newPath, 'User')
`, psQuote(shimsDir))
	if out, err := runHiddenPS(ps); err != nil {
		return nil, fmt.Errorf("update Windows User PATH: %w (%s)", err, out)
	}
	return []string{`HKCU\Environment\Path (` + shimsDir + `)`}, nil
}

// ResolveRealBinary locates cmdName on Windows PATH (skipping shimsDir and harpo.exe)
// plus standard Windows Program Files / LocalAppData install folders for browsers & Electron apps.
func ResolveRealBinary(cmdName, shimsDir string) (string, error) {
	cmdName = strings.TrimSpace(cmdName)
	if cmdName == "" {
		return "", errors.New("empty command name")
	}
	// Direct path supplied.
	if strings.ContainsAny(cmdName, `/\`) || filepath.VolumeName(cmdName) != "" {
		if found := statWinExe(cmdName); found != "" {
			return found, nil
		}
		return "", fmt.Errorf("executable not found: %s", cmdName)
	}

	cleanShims := ""
	if shimsDir != "" {
		cleanShims = strings.ToLower(filepath.Clean(shimsDir))
	}
	selfExe, _ := os.Executable()
	selfClean := strings.ToLower(filepath.Clean(selfExe))

	exts := winPathExts(cmdName)
	for _, dir := range filepath.SplitList(os.Getenv("PATH")) {
		if strings.TrimSpace(dir) == "" {
			continue
		}
		cleanDir := filepath.Clean(dir)
		if cleanShims != "" && strings.ToLower(cleanDir) == cleanShims {
			continue
		}
		for _, ext := range exts {
			cand := filepath.Join(cleanDir, cmdName+ext)
			fi, err := os.Stat(cand)
			if err != nil || fi.IsDir() {
				continue
			}
			if !strings.EqualFold(strings.TrimSuffix(cmdName, ".exe"), "harpo") && selfClean != "" {
				if strings.ToLower(filepath.Clean(cand)) == selfClean {
					continue
				}
			}
			return cand, nil
		}
	}

	// Fallback: search common Windows GUI/CLI install paths (Chrome, Brave, Edge, Claude, VS Code, Cursor, etc.).
	if found := lookupKnownWindowsApp(cmdName); found != "" {
		return found, nil
	}
	return "", fmt.Errorf("command %q not found in PATH or standard Windows install folders", cmdName)
}

func winPathExts(cmdName string) []string {
	if filepath.Ext(cmdName) != "" {
		return []string{""}
	}
	raw := os.Getenv("PATHEXT")
	if raw == "" {
		raw = ".COM;.EXE;.BAT;.CMD"
	}
	var out []string
	for _, e := range strings.Split(raw, ";") {
		e = strings.TrimSpace(e)
		if e != "" {
			out = append(out, e)
		}
	}
	return out
}

func statWinExe(path string) string {
	for _, ext := range winPathExts(path) {
		cand := path + ext
		if fi, err := os.Stat(cand); err == nil && !fi.IsDir() {
			return cand
		}
	}
	return ""
}

func lookupKnownWindowsApp(cmdName string) string {
	base := strings.ToLower(strings.TrimSuffix(cmdName, ".exe"))
	local := os.Getenv("LOCALAPPDATA")
	appData := os.Getenv("APPDATA")
	pf := os.Getenv("ProgramFiles")
	pf86 := os.Getenv("ProgramFiles(x86)")
	home, _ := os.UserHomeDir()

	var cands []string
	switch base {
	case "chrome", "chromium":
		cands = []string{
			filepath.Join(pf, "Google", "Chrome", "Application", "chrome.exe"),
			filepath.Join(pf86, "Google", "Chrome", "Application", "chrome.exe"),
			filepath.Join(local, "Google", "Chrome", "Application", "chrome.exe"),
			filepath.Join(local, "Chromium", "Application", "chrome.exe"),
		}
	case "brave", "brave-browser":
		cands = []string{
			filepath.Join(pf, "BraveSoftware", "Brave-Browser", "Application", "brave.exe"),
			filepath.Join(pf86, "BraveSoftware", "Brave-Browser", "Application", "brave.exe"),
			filepath.Join(local, "BraveSoftware", "Brave-Browser", "Application", "brave.exe"),
		}
	case "edge", "msedge":
		cands = []string{
			filepath.Join(pf86, "Microsoft", "Edge", "Application", "msedge.exe"),
			filepath.Join(pf, "Microsoft", "Edge", "Application", "msedge.exe"),
		}
	case "claude":
		cands = []string{
			filepath.Join(home, ".local", "bin", "claude.exe"),
			filepath.Join(local, "Programs", "claude-code", "claude.exe"),
			filepath.Join(appData, "npm", "claude.cmd"),
		}
	case "claude-desktop":
		cands = []string{
			filepath.Join(local, "AnthropicClaude", "claude.exe"),
		}
	case "code":
		cands = []string{
			filepath.Join(local, "Programs", "Microsoft VS Code", "Code.exe"),
			filepath.Join(pf, "Microsoft VS Code", "Code.exe"),
		}
	case "cursor":
		cands = []string{
			filepath.Join(local, "Programs", "cursor", "Cursor.exe"),
		}
	case "obsidian":
		cands = []string{
			filepath.Join(local, "Obsidian", "Obsidian.exe"),
			filepath.Join(local, "Programs", "obsidian", "Obsidian.exe"),
		}
	case "telegram":
		cands = []string{
			filepath.Join(appData, "Telegram Desktop", "Telegram.exe"),
		}
	}
	for _, c := range cands {
		if c != "" {
			if fi, err := os.Stat(c); err == nil && !fi.IsDir() {
				return c
			}
		}
	}
	return ""
}

// DiscoverSystemApps discovers installed Windows browsers & Start Menu desktop applications.
func DiscoverSystemApps(p config.Paths) []DiscoveredApp {
	seen := map[string]bool{}
	var out []DiscoveredApp

	add := func(name, bin, src string) {
		if bin == "" || name == "" {
			return
		}
		key := strings.ToLower(bin)
		if seen[key] {
			return
		}
		base := strings.ToLower(filepath.Base(bin))
		if base == "harpo.exe" || strings.Contains(base, "unins") || strings.Contains(base, "setup") {
			return
		}
		seen[key] = true
		id := strings.ToLower(strings.TrimSuffix(filepath.Base(bin), filepath.Ext(bin)))
		out = append(out, DiscoveredApp{
			DesktopID:     id,
			Name:          name,
			Binary:        bin,
			Icon:          bin,
			SuggestedMode: guessMode(bin, name),
			SourcePath:    src,
		})
	}

	// 1. Fast check for well-known browsers & AI apps.
	for _, known := range []struct {
		name, key string
	}{
		{"Google Chrome", "chrome"},
		{"Brave Browser", "brave"},
		{"Microsoft Edge", "msedge"},
		{"Claude CLI", "claude"},
		{"Claude Desktop", "claude-desktop"},
		{"Visual Studio Code", "code"},
		{"Cursor", "cursor"},
		{"Obsidian", "obsidian"},
		{"Telegram Desktop", "telegram"},
	} {
		if bin := lookupKnownWindowsApp(known.key); bin != "" {
			add(known.name, bin, bin)
		}
	}

	// 2. Scan Start Menu shortcuts (.lnk) quickly via WScript.Shell.
	ps := `
$sh = New-Object -ComObject WScript.Shell
$dirs = @(
  "$env:APPDATA\Microsoft\Windows\Start Menu\Programs",
  "$env:ProgramData\Microsoft\Windows\Start Menu\Programs"
)
foreach ($d in $dirs) {
  if (-not (Test-Path $d)) { continue }
  Get-ChildItem -Path $d -Filter *.lnk -Recurse -ErrorAction SilentlyContinue | Select-Object -First 80 | ForEach-Object {
    if ($_.FullName -like '*\Harpo\*') { return }
    try {
      $sc = $sh.CreateShortcut($_.FullName)
      $tp = $sc.TargetPath
      if ($tp -and $tp.EndsWith('.exe', [StringComparison]::OrdinalIgnoreCase) -and (Test-Path $tp)) {
        Write-Output ($_.BaseName + '|' + $tp + '|' + $_.FullName)
      }
    } catch {}
  }
}
`
	if raw, err := runHiddenPS(ps); err == nil && raw != "" {
		for _, line := range strings.Split(raw, "\n") {
			parts := strings.SplitN(strings.TrimSpace(line), "|", 3)
			if len(parts) == 3 {
				add(parts[0], parts[1], parts[2])
			}
		}
	}

	sort.Slice(out, func(i, j int) bool {
		return strings.ToLower(out[i].Name) < strings.ToLower(out[j].Name)
	})
	return out
}

// InstallDesktopOverride creates a Start Menu .lnk shortcut in %APPDATA%\Microsoft\Windows\Start Menu\Programs\Harpo\.
func InstallDesktopOverride(p config.Paths, a AppEntry) error {
	dir := DesktopAppsDir(p)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	harpoBin := ResolveHarpoTarget(p)
	lnkPath := DesktopPath(p, a.Name)

	iconPath := a.DesktopIcon
	if iconPath == "" {
		if realBin, err := ResolveRealBinary(a.TargetCommand(), ShimsDir(p)); err == nil {
			iconPath = realBin
		} else {
			iconPath = harpoBin
		}
	}

	// WindowStyle 7 (minimized) for GUI apps so the transient harpo launcher window stays out of sight.
	winStyle := 1
	if a.Mode != ModeCLI {
		winStyle = 7
	}

	ps := fmt.Sprintf(`
$sh = New-Object -ComObject WScript.Shell
$sc = $sh.CreateShortcut(%s)
$sc.TargetPath = %s
$sc.Arguments = %s
$sc.Description = %s
$sc.IconLocation = %s
$sc.WindowStyle = %d
$sc.Save()
`,
		psQuote(lnkPath),
		psQuote(harpoBin),
		psQuote("launch "+a.Name),
		psQuote("Harpo-Managed: Isolated via Harpo Proxy & Kill-Switch"),
		psQuote(iconPath+",0"),
		winStyle,
	)
	if out, err := runHiddenPS(ps); err != nil {
		return fmt.Errorf("create Start Menu shortcut %s: %w (%s)", lnkPath, err, out)
	}
	return nil
}

// RemoveDesktopOverride removes the Harpo Start Menu .lnk shortcut if present.
func RemoveDesktopOverride(p config.Paths, name string) error {
	dst := DesktopPath(p, name)
	if _, err := os.Stat(dst); err != nil {
		return nil
	}
	return os.Remove(dst)
}

// HasDesktopOverride reports whether a Harpo Start Menu .lnk shortcut exists for name.
func HasDesktopOverride(p config.Paths, name string) bool {
	_, err := os.Stat(DesktopPath(p, name))
	return err == nil
}

// ListIsolatedProcesses lists running wrapped app processes and the managed xray.exe proxy process on Windows.
func ListIsolatedProcesses() []IsolatedProc {
	paths := config.DefaultPaths()
	list, _ := Load(paths)
	want := map[string]string{}
	shimsDir := ShimsDir(paths)
	for _, a := range list {
		if !a.Enabled {
			continue
		}
		target := strings.ToLower(filepath.Base(a.TargetCommand()))
		if !strings.HasSuffix(target, ".exe") {
			target += ".exe"
		}
		want[target] = a.Name
		if realBin, err := ResolveRealBinary(a.TargetCommand(), shimsDir); err == nil {
			want[strings.ToLower(filepath.Base(realBin))] = a.Name
		}
	}

	cmd := exec.Command("tasklist", "/FO", "CSV", "/NH")
	cmd.SysProcAttr = &syscall.SysProcAttr{HideWindow: true, CreationFlags: 0x08000000}
	out, err := cmd.Output()
	if err != nil {
		return nil
	}
	r := csv.NewReader(strings.NewReader(string(out)))
	rows, err := r.ReadAll()
	if err != nil {
		return nil
	}

	var procs []IsolatedProc
	for _, row := range rows {
		if len(row) < 2 {
			continue
		}
		img := strings.TrimSpace(row[0])
		pid, err := strconv.Atoi(strings.TrimSpace(row[1]))
		if err != nil {
			continue
		}
		low := strings.ToLower(img)
		if low == "xray.exe" {
			procs = append(procs, IsolatedProc{
				PID:      pid,
				Comm:     img,
				Cmdline:  "harpo proxy daemon",
				Scope:    config.WinProxyTask,
				IsAnchor: true,
			})
			continue
		}
		if appName, ok := want[low]; ok {
			procs = append(procs, IsolatedProc{
				PID:      pid,
				Comm:     img,
				Cmdline:  "wrapped app: " + appName,
				Scope:    appName,
				IsAnchor: false,
			})
		}
	}
	sort.Slice(procs, func(i, j int) bool {
		if procs[i].IsAnchor != procs[j].IsAnchor {
			return !procs[i].IsAnchor
		}
		return procs[i].PID < procs[j].PID
	})
	return procs
}
