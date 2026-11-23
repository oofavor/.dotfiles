package apps

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"sort"
	"strings"

	"harpo/internal/config"
)

// AppMode determines how harpo wraps and injects proxy flags for an application.
type AppMode string

const (
	ModeCLI         AppMode = "cli"
	ModeElectron    AppMode = "electron"
	ModeChromium    AppMode = "chromium"
	ModeProxychains AppMode = "proxychains"
)

// AllModes lists all supported wrap modes in cycle order.
var AllModes = []AppMode{ModeCLI, ModeElectron, ModeChromium, ModeProxychains}

// ValidMode checks whether m is a supported AppMode.
func ValidMode(m AppMode) bool {
	for _, x := range AllModes {
		if x == m {
			return true
		}
	}
	return false
}

// AppEntry describes one harpo-managed application stored in apps.json.
type AppEntry struct {
	Name              string            `json:"name"`
	Mode              AppMode           `json:"mode"`
	Command           string            `json:"command,omitempty"`
	Enabled           bool              `json:"enabled"`
	DesktopEntry      bool              `json:"desktop_entry,omitempty"`
	DesktopName       string            `json:"desktop_name,omitempty"`
	DesktopIcon       string            `json:"desktop_icon,omitempty"`
	DesktopCategories string            `json:"desktop_categories,omitempty"`
	IsolatedProfile   bool              `json:"isolated_profile,omitempty"`
	ProfileDir        string            `json:"profile_dir,omitempty"`
	DefaultArgs       []string          `json:"default_args,omitempty"`
	ExtraArgs         []string          `json:"extra_args,omitempty"`
	Env               map[string]string `json:"env,omitempty"`
	NoFirewall        bool              `json:"no_firewall,omitempty"` // Windows: opt out of per-.exe firewall block (use soft mode)
}

// WantsFirewall reports whether this app should get an OS firewall kill-switch rule.
func (a AppEntry) WantsFirewall() bool {
	return a.Enabled && !a.NoFirewall
}

// EffectiveProfileDir returns the resolved --user-data-dir path for an isolated profile.
func (a AppEntry) EffectiveProfileDir(p config.Paths) string {
	dir := strings.TrimSpace(a.ProfileDir)
	if dir == "" {
		return filepath.Join(ProfilesBaseDir(p), a.Name)
	}
	if strings.HasPrefix(dir, "~/") || strings.HasPrefix(dir, `~\`) {
		return filepath.Join(p.Home, dir[2:])
	}
	return dir
}

// TargetCommand returns the underlying binary/command name to look up in PATH.
func (a AppEntry) TargetCommand() string {
	if strings.TrimSpace(a.Command) != "" {
		return strings.TrimSpace(a.Command)
	}
	return strings.TrimSpace(a.Name)
}

// RegistryPath returns the path to apps.json inside the config directory.
func RegistryPath(p config.Paths) string {
	return filepath.Join(p.CfgDir, "apps.json")
}

func effectiveDataDir(p config.Paths) string {
	if strings.TrimSpace(p.DataDir) != "" {
		return p.DataDir
	}
	if runtime.GOOS == "windows" {
		if local := os.Getenv("LOCALAPPDATA"); local != "" && p.Home == "" {
			return filepath.Join(local, "harpo")
		}
		return filepath.Join(p.Home, "AppData", "Local", "harpo")
	}
	return filepath.Join(p.Home, ".local", "share", "harpo")
}

// ShimsDir returns the shims directory (or HARPO_SHIMS_DIR).
func ShimsDir(p config.Paths) string {
	if v := strings.TrimSpace(os.Getenv("HARPO_SHIMS_DIR")); v != "" {
		return v
	}
	return filepath.Join(effectiveDataDir(p), "shims")
}

// ProfilesBaseDir returns the base directory for optional isolated browser/Electron profiles.
func ProfilesBaseDir(p config.Paths) string {
	return filepath.Join(effectiveDataDir(p), "profiles")
}

// Load reads apps.json, returning an empty slice if it does not exist yet.
func Load(p config.Paths) ([]AppEntry, error) {
	b, err := os.ReadFile(RegistryPath(p))
	if err != nil {
		if os.IsNotExist(err) {
			return []AppEntry{}, nil
		}
		return nil, err
	}
	var list []AppEntry
	if err := json.Unmarshal(b, &list); err != nil {
		return nil, err
	}
	for i := range list {
		if !ValidMode(list[i].Mode) {
			list[i].Mode = ModeCLI
		}
	}
	return list, nil
}

// Save writes apps.json atomically with mode 0600.
func Save(p config.Paths, list []AppEntry) error {
	if list == nil {
		list = []AppEntry{}
	}
	sort.SliceStable(list, func(i, j int) bool {
		return strings.ToLower(list[i].Name) < strings.ToLower(list[j].Name)
	})
	b, err := json.MarshalIndent(list, "", "  ")
	if err != nil {
		return err
	}
	return config.WriteSecretText(RegistryPath(p), string(b)+"\n")
}

// Find looks up an app entry by name (case-insensitive fallback on Windows).
func Find(p config.Paths, name string) (*AppEntry, error) {
	list, err := Load(p)
	if err != nil {
		return nil, err
	}
	for i := range list {
		if list[i].Name == name || (runtime.GOOS == "windows" && strings.EqualFold(list[i].Name, name)) {
			cp := list[i]
			return &cp, nil
		}
	}
	return nil, nil
}

// Upsert inserts or updates an AppEntry and synchronizes its shim and desktop/Start Menu override.
func Upsert(p config.Paths, entry AppEntry) error {
	entry.Name = strings.TrimSpace(entry.Name)
	if entry.Name == "" || strings.ContainsAny(entry.Name, `/\ :`+"\t\n") {
		return fmt.Errorf("invalid app name %q", entry.Name)
	}
	low := strings.ToLower(entry.Name)
	if low == "harpo" || low == "harpo-run" || low == "isp-run" || low == "harpo.exe" {
		return fmt.Errorf("cannot wrap reserved name %q", entry.Name)
	}
	if !ValidMode(entry.Mode) {
		entry.Mode = ModeCLI
	}

	list, err := Load(p)
	if err != nil {
		return err
	}
	replaced := false
	for i := range list {
		if list[i].Name == entry.Name || (runtime.GOOS == "windows" && strings.EqualFold(list[i].Name, entry.Name)) {
			list[i] = entry
			replaced = true
			break
		}
	}
	if !replaced {
		list = append(list, entry)
	}
	if err := Save(p, list); err != nil {
		return err
	}
	return SyncOne(p, entry)
}

// Remove deletes an app from apps.json and cleans up its shim and managed launcher shortcut.
func Remove(p config.Paths, name string) error {
	list, err := Load(p)
	if err != nil {
		return err
	}
	out := make([]AppEntry, 0, len(list))
	for _, a := range list {
		if a.Name != name && !(runtime.GOOS == "windows" && strings.EqualFold(a.Name, name)) {
			out = append(out, a)
		}
	}
	if err := Save(p, out); err != nil {
		return err
	}
	_ = RemoveShim(p, name)
	_ = RemoveDesktopOverride(p, name)
	return nil
}

// ResolveHarpoTarget resolves the stable path to harpo suitable for shims and launcher shortcuts.
func ResolveHarpoTarget(p config.Paths) string {
	shims := ShimsDir(p)
	if found, err := ResolveRealBinary("harpo", shims); err == nil && found != "" {
		return found
	}
	if exe, err := os.Executable(); err == nil {
		return exe
	}
	if runtime.GOOS == "windows" {
		return "harpo.exe"
	}
	return "harpo"
}

// SyncAll ensures every entry in apps.json has up-to-date shims and desktop/Start Menu overrides.
func SyncAll(p config.Paths) error {
	list, err := Load(p)
	if err != nil {
		return err
	}
	for _, a := range list {
		if err := SyncOne(p, a); err != nil {
			return err
		}
	}
	return nil
}

// SyncOne synchronizes the shim and optional desktop/Start Menu override for a single AppEntry.
func SyncOne(p config.Paths, a AppEntry) error {
	if !a.Enabled {
		_ = RemoveShim(p, a.Name)
		_ = RemoveDesktopOverride(p, a.Name)
		return nil
	}
	if err := EnsureShim(p, a.Name); err != nil {
		return err
	}
	if a.DesktopEntry {
		if err := InstallDesktopOverride(p, a); err != nil {
			return err
		}
	} else {
		_ = RemoveDesktopOverride(p, a.Name)
	}
	return nil
}

// BuildExecArgs constructs the final argv for launching realBin according to mode, flags, and user args.
func BuildExecArgs(p config.Paths, a AppEntry, realBin string, userArgs []string) ([]string, error) {
	effectiveArgs := userArgs
	if len(effectiveArgs) == 0 && len(a.DefaultArgs) > 0 {
		effectiveArgs = a.DefaultArgs
	}

	switch a.Mode {
	case ModeElectron, ModeChromium:
		flags := []string{
			fmt.Sprintf("--proxy-server=socks5://127.0.0.1:%d", config.SocksPort),
			"--host-resolver-rules=MAP * ~NOTFOUND , EXCLUDE 127.0.0.1",
			"--force-webrtc-ip-handling-policy=disable_non_proxied_udp",
			"--disable-quic",
		}
		if a.IsolatedProfile || strings.TrimSpace(a.ProfileDir) != "" {
			flags = append(flags, "--user-data-dir="+a.EffectiveProfileDir(p))
		}
		out := []string{realBin}
		out = append(out, flags...)
		out = append(out, a.ExtraArgs...)
		out = append(out, effectiveArgs...)
		return out, nil

	case ModeProxychains:
		if runtime.GOOS == "windows" {
			return nil, errors.New("proxychains mode is Linux-only (use cli, electron, or chromium mode on Windows)")
		}
		pcBin := ""
		for _, cand := range []string{"proxychains4", "proxychains"} {
			if found, err := exec.LookPath(cand); err == nil {
				pcBin = found
				break
			}
		}
		if pcBin == "" {
			return nil, errors.New("proxychains4 / proxychains not found in PATH")
		}
		confPath := filepath.Join(p.CfgDir, "proxychains.conf")
		confBody := fmt.Sprintf("strict_chain\nproxy_dns\nquiet_mode\ntcp_read_time_out 15000\ntcp_connect_time_out 8000\n\n[ProxyList]\nsocks5 127.0.0.1 %d\n", config.SocksPort)
		if err := config.WriteSecretText(confPath, confBody); err != nil {
			return nil, fmt.Errorf("write proxychains.conf: %w", err)
		}
		out := []string{pcBin, "-q", "-f", confPath, realBin}
		out = append(out, a.ExtraArgs...)
		out = append(out, effectiveArgs...)
		return out, nil

	default: // ModeCLI
		out := []string{realBin}
		out = append(out, a.ExtraArgs...)
		out = append(out, effectiveArgs...)
		return out, nil
	}
}

// DiscoveredApp represents an installed desktop application discovered on the system.
type DiscoveredApp struct {
	DesktopID     string
	Name          string
	Binary        string
	Icon          string
	SuggestedMode AppMode
	SourcePath    string
}

// IsolatedProc describes a live isolated/managed process.
type IsolatedProc struct {
	PID      int
	Comm     string
	Cmdline  string
	Scope    string
	IsAnchor bool
}

func guessMode(bin, name string) AppMode {
	low := strings.ToLower(bin + " " + name)
	for _, kw := range []string{"chromium", "chrome", "brave", "vivaldi", "edge", "msedge", "opera"} {
		if strings.Contains(low, kw) {
			return ModeChromium
		}
	}
	for _, kw := range []string{"claude", "electron", "discord", "slack", "cursor", "code", "obsidian", "notion", "telegram", "spotify", "element", "signal", "vesktop"} {
		if strings.Contains(low, kw) {
			return ModeElectron
		}
	}
	return ModeCLI
}
