//go:build !windows

package apps

import (
	"bufio"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strconv"
	"strings"

	"harpo/internal/config"
)

// ShimPath returns the path to the shim symlink for name.
func ShimPath(p config.Paths, name string) string {
	return filepath.Join(ShimsDir(p), name)
}

// DesktopAppsDir returns ~/.local/share/applications.
func DesktopAppsDir(p config.Paths) string {
	return filepath.Join(p.Home, ".local", "share", "applications")
}

// DesktopPath returns the path to the managed .desktop override for name.
func DesktopPath(p config.Paths, name string) string {
	return filepath.Join(DesktopAppsDir(p), name+".desktop")
}

// EnsureShim creates/updates ~/.local/share/harpo/shims/<name> -> <harpo>.
func EnsureShim(p config.Paths, name string) error {
	dir := ShimsDir(p)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	target := ResolveHarpoTarget(p)
	linkPath := ShimPath(p, name)
	if fi, err := os.Lstat(linkPath); err == nil {
		if fi.Mode()&os.ModeSymlink == 0 {
			return fmt.Errorf("refusing to overwrite non-symlink at %s", linkPath)
		}
		_ = os.Remove(linkPath)
	}
	return os.Symlink(target, linkPath)
}

// RemoveShim removes the shim symlink if present.
func RemoveShim(p config.Paths, name string) error {
	linkPath := ShimPath(p, name)
	fi, err := os.Lstat(linkPath)
	if err != nil {
		return nil
	}
	if fi.Mode()&os.ModeSymlink != 0 {
		return os.Remove(linkPath)
	}
	return nil
}

// HasShim reports whether an active shim symlink exists for name.
func HasShim(p config.Paths, name string) bool {
	linkPath := ShimPath(p, name)
	fi, err := os.Lstat(linkPath)
	return err == nil && fi.Mode()&os.ModeSymlink != 0
}

// IsShimsInPATH checks whether ~/.local/share/harpo/shims is present in PATH.
func IsShimsInPATH(p config.Paths) (inPath bool, position int) {
	shims := filepath.Clean(ShimsDir(p))
	for i, dir := range filepath.SplitList(os.Getenv("PATH")) {
		if filepath.Clean(dir) == shims {
			return true, i
		}
	}
	return false, -1
}

// InstallShellPATH configures PATH automatically and idempotently across systemd/desktop sessions,
// Fish, Bash, and Zsh so ~/.local/share/harpo/shims is prepended on any Linux distro.
// Skips read-only Nix store symlinks safely.
func InstallShellPATH(p config.Paths) ([]string, error) {
	shimsDir := ShimsDir(p)
	if err := os.MkdirAll(shimsDir, 0o755); err != nil {
		return nil, err
	}

	expr := shimsDir
	if rel, err := filepath.Rel(p.Home, shimsDir); err == nil && !strings.HasPrefix(rel, "..") {
		expr = "$HOME/" + filepath.ToSlash(rel)
	}

	var updated []string

	// 1. Systemd user environment.d (works for terminals AND Wayland/Niri/GNOME/KDE launchers).
	envDir := filepath.Join(p.Home, ".config", "environment.d")
	if err := os.MkdirAll(envDir, 0o755); err == nil {
		envFile := filepath.Join(envDir, "10-harpo.conf")
		content := fmt.Sprintf("# Managed by harpo — prepends update-proof app shims to PATH\nPATH=%s:$PATH\n", expr)
		if isWritableTarget(envFile) {
			if err := os.WriteFile(envFile, []byte(content), 0o644); err == nil {
				updated = append(updated, envFile)
			}
		}
	}

	// 2. Fish drop-in (~/.config/fish/conf.d/00-harpo.fish — zero edits to config.fish!).
	fishDir := filepath.Join(p.Home, ".config", "fish")
	_, hasFishBin := exec.LookPath("fish")
	if hasFishBin == nil || dirExists(fishDir) {
		confD := filepath.Join(fishDir, "conf.d")
		if err := os.MkdirAll(confD, 0o755); err == nil {
			fishFile := filepath.Join(confD, "00-harpo.fish")
			content := fmt.Sprintf("# Managed by harpo — prepends update-proof app shims to PATH\nfish_add_path -p \"%s\"\n", expr)
			if isWritableTarget(fishFile) {
				if err := os.WriteFile(fishFile, []byte(content), 0o644); err == nil {
					updated = append(updated, fishFile)
				}
			}
		}
	}

	// 3. Bash / Zsh rc files (append idempotent guarded line only if file exists & is writable).
	exportLine := fmt.Sprintf("export PATH=\"%s:$PATH\" # harpo-shims\n", expr)
	rcUpdated := false
	for _, rcName := range []string{".bashrc", ".zshrc"} {
		rcPath := filepath.Join(p.Home, rcName)
		if appendLineIfMissing(rcPath, "harpo-shims", expr, exportLine) {
			updated = append(updated, rcPath)
			rcUpdated = true
		}
	}
	if !rcUpdated {
		profPath := filepath.Join(p.Home, ".profile")
		if appendLineIfMissing(profPath, "harpo-shims", expr, exportLine) {
			updated = append(updated, profPath)
		}
	}

	return updated, nil
}

func dirExists(path string) bool {
	fi, err := os.Stat(path)
	return err == nil && fi.IsDir()
}

func isWritableTarget(path string) bool {
	fi, err := os.Lstat(path)
	if err != nil {
		return os.IsNotExist(err)
	}
	if fi.Mode()&os.ModeSymlink != 0 {
		return false
	}
	return fi.Mode().Perm()&0o200 != 0
}

func appendLineIfMissing(path, marker, shimsExpr, line string) bool {
	fi, err := os.Lstat(path)
	if err != nil || fi.IsDir() || fi.Mode()&os.ModeSymlink != 0 || fi.Mode().Perm()&0o200 == 0 {
		return false
	}
	b, err := os.ReadFile(path)
	if err != nil {
		return false
	}
	s := string(b)
	if strings.Contains(s, marker) || strings.Contains(s, shimsExpr) {
		return false
	}
	prefix := ""
	if len(s) > 0 && !strings.HasSuffix(s, "\n") {
		prefix = "\n"
	}
	f, err := os.OpenFile(path, os.O_APPEND|os.O_WRONLY, 0o644)
	if err != nil {
		return false
	}
	defer f.Close()
	_, err = f.WriteString(prefix + line)
	return err == nil
}

// ResolveRealBinary locates cmdName on PATH while strictly skipping shimsDir and any symlink resolving to harpo itself.
func ResolveRealBinary(cmdName, shimsDir string) (string, error) {
	cmdName = strings.TrimSpace(cmdName)
	if cmdName == "" {
		return "", errors.New("empty command name")
	}
	if strings.ContainsRune(cmdName, os.PathSeparator) {
		fi, err := os.Stat(cmdName)
		if err != nil || fi.IsDir() || fi.Mode()&0o111 == 0 {
			return "", fmt.Errorf("executable not found: %s", cmdName)
		}
		return cmdName, nil
	}

	cleanShims := ""
	if shimsDir != "" {
		cleanShims = filepath.Clean(shimsDir)
		if eval, err := filepath.EvalSymlinks(cleanShims); err == nil {
			cleanShims = eval
		}
	}
	selfExe, _ := os.Executable()
	selfReal, _ := filepath.EvalSymlinks(selfExe)

	for _, dir := range filepath.SplitList(os.Getenv("PATH")) {
		if dir == "" {
			continue
		}
		cleanDir := filepath.Clean(dir)
		evalDir := cleanDir
		if e, err := filepath.EvalSymlinks(cleanDir); err == nil {
			evalDir = e
		}
		if cleanShims != "" && (cleanDir == cleanShims || evalDir == cleanShims) {
			continue
		}
		cand := filepath.Join(cleanDir, cmdName)
		fi, err := os.Stat(cand)
		if err != nil || fi.IsDir() || fi.Mode()&0o111 == 0 {
			continue
		}
		if cmdName != "harpo" && selfReal != "" {
			if candReal, err := filepath.EvalSymlinks(cand); err == nil && candReal == selfReal {
				continue
			}
		}
		return cand, nil
	}
	return "", fmt.Errorf("command %q not found in PATH (outside harpo shims)", cmdName)
}

// DiscoverSystemApps scans standard XDG application directories across any Linux distro.
func DiscoverSystemApps(p config.Paths) []DiscoveredApp {
	dirs := []string{
		"/usr/share/applications",
		"/usr/local/share/applications",
		"/run/current-system/sw/share/applications",
		filepath.Join("/etc/profiles/per-user", os.Getenv("USER"), "share", "applications"),
		filepath.Join(p.Home, ".nix-profile", "share", "applications"),
		DesktopAppsDir(p),
	}
	seen := map[string]bool{}
	var out []DiscoveredApp

	for _, dir := range dirs {
		entries, err := os.ReadDir(dir)
		if err != nil {
			continue
		}
		for _, e := range entries {
			if e.IsDir() || !strings.HasSuffix(e.Name(), ".desktop") {
				continue
			}
			full := filepath.Join(dir, e.Name())
			d, ok := parseDesktopFile(full, e.Name())
			if !ok || d.Binary == "" || seen[d.Binary] {
				continue
			}
			seen[d.Binary] = true
			out = append(out, d)
		}
	}
	sort.Slice(out, func(i, j int) bool {
		return strings.ToLower(out[i].Name) < strings.ToLower(out[j].Name)
	})
	return out
}

func parseDesktopFile(path, filename string) (DiscoveredApp, bool) {
	f, err := os.Open(path)
	if err != nil {
		return DiscoveredApp{}, false
	}
	defer f.Close()

	var d DiscoveredApp
	d.DesktopID = strings.TrimSuffix(filename, ".desktop")
	d.SourcePath = path
	inDesktopEntry := false
	noDisplay := false
	harpoManaged := false

	sc := bufio.NewScanner(f)
	for sc.Scan() {
		line := strings.TrimSpace(sc.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		if strings.HasPrefix(line, "[") {
			inDesktopEntry = (line == "[Desktop Entry]")
			continue
		}
		if !inDesktopEntry {
			continue
		}
		k, v, ok := strings.Cut(line, "=")
		if !ok {
			continue
		}
		switch k {
		case "Name":
			if d.Name == "" {
				d.Name = v
			}
		case "Icon":
			d.Icon = v
		case "NoDisplay":
			if strings.EqualFold(v, "true") {
				noDisplay = true
			}
		case "X-Harpo-Managed":
			if strings.EqualFold(v, "true") {
				harpoManaged = true
			}
		case "Exec":
			d.Binary = extractExecBinary(v)
		}
	}
	if noDisplay || harpoManaged || d.Name == "" || d.Binary == "" {
		return DiscoveredApp{}, false
	}
	d.SuggestedMode = guessMode(d.Binary, d.Name)
	return d, true
}

func extractExecBinary(execLine string) string {
	fields := strings.Fields(execLine)
	for _, f := range fields {
		f = strings.Trim(f, `"'`)
		if f == "" || f == "env" || strings.Contains(f, "=") {
			continue
		}
		base := filepath.Base(f)
		if base == "harpo" || base == "harpo-run" || base == "isp-run" {
			return ""
		}
		return base
	}
	return ""
}

// InstallDesktopOverride writes ~/.local/share/applications/<name>.desktop pointing to `harpo launch <name> %U`.
func InstallDesktopOverride(p config.Paths, a AppEntry) error {
	dir := DesktopAppsDir(p)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	harpoBin := ResolveHarpoTarget(p)
	dispName := a.DesktopName
	if dispName == "" {
		dispName = a.Name
	}
	icon := a.DesktopIcon
	if icon == "" {
		icon = a.Name
	}
	term := "false"
	if a.Mode == ModeCLI {
		term = "true"
	}

	cats := strings.TrimSpace(a.DesktopCategories)
	if cats == "" {
		if a.Mode == ModeChromium {
			cats = "Network;WebBrowser;"
		} else {
			cats = "Network;Utility;"
		}
	}

	content := fmt.Sprintf(`[Desktop Entry]
Type=Application
Name=%s
Comment=Isolated via Harpo Proxy & Kernel Kill-Switch
Exec=%s launch %s %%U
Icon=%s
Terminal=%s
Categories=%s
StartupNotify=true
X-Harpo-Managed=true
`, dispName, harpoBin, a.Name, icon, term, cats)

	dst := DesktopPath(p, a.Name)
	return os.WriteFile(dst, []byte(content), 0o644)
}

// RemoveDesktopOverride removes ~/.local/share/applications/<name>.desktop ONLY if managed by harpo.
func RemoveDesktopOverride(p config.Paths, name string) error {
	dst := DesktopPath(p, name)
	b, err := os.ReadFile(dst)
	if err != nil {
		return nil
	}
	if strings.Contains(string(b), "X-Harpo-Managed=true") {
		return os.Remove(dst)
	}
	return nil
}

// HasDesktopOverride reports whether a harpo-managed desktop entry exists for name.
func HasDesktopOverride(p config.Paths, name string) bool {
	dst := DesktopPath(p, name)
	b, err := os.ReadFile(dst)
	return err == nil && strings.Contains(string(b), "X-Harpo-Managed=true")
}

// ListIsolatedProcesses inspects /sys/fs/cgroup/.../harpo-isolated.slice directly.
func ListIsolatedProcesses() []IsolatedProc {
	uid := os.Getuid()
	cgRoot := fmt.Sprintf("/sys/fs/cgroup/user.slice/user-%d.slice/user@%d.service/harpo.slice/%s", uid, uid, config.SliceUnit)
	if fi, err := os.Stat(cgRoot); err != nil || !fi.IsDir() {
		return nil
	}

	seen := map[int]bool{}
	var procs []IsolatedProc
	_ = filepath.WalkDir(cgRoot, func(path string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() || d.Name() != "cgroup.procs" {
			return nil
		}
		scope := filepath.Base(filepath.Dir(path))
		isAnchor := scope == config.AnchorUnit
		b, err := os.ReadFile(path)
		if err != nil {
			return nil
		}
		for _, tok := range strings.Fields(string(b)) {
			pid, err := strconv.Atoi(tok)
			if err != nil || seen[pid] {
				continue
			}
			seen[pid] = true
			comm := strings.TrimSpace(readFileQuick(fmt.Sprintf("/proc/%d/comm", pid)))
			rawCmd := readFileQuick(fmt.Sprintf("/proc/%d/cmdline", pid))
			cmdline := strings.TrimSpace(strings.ReplaceAll(rawCmd, "\x00", " "))
			if comm == "" {
				continue
			}
			procs = append(procs, IsolatedProc{
				PID:      pid,
				Comm:     comm,
				Cmdline:  cmdline,
				Scope:    scope,
				IsAnchor: isAnchor,
			})
		}
		return nil
	})

	sort.Slice(procs, func(i, j int) bool {
		if procs[i].IsAnchor != procs[j].IsAnchor {
			return !procs[i].IsAnchor
		}
		return procs[i].PID < procs[j].PID
	})
	return procs
}

func readFileQuick(path string) string {
	b, err := os.ReadFile(path)
	if err != nil {
		return ""
	}
	return string(b)
}
