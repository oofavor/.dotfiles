package config

import (
	"encoding/json"
	"fmt"
	"net/netip"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"time"
	_ "time/tzdata" // Embed IANA tzdata so timezone validation works on any OS/distro.
)

const (
	SocksPort = 10808
	HTTPPort  = 10809

	ProxyUnit     = "harpo-proxy.service"
	SliceUnit     = "harpo-isolated.slice"
	AnchorUnit    = "harpo-isolated-anchor.service"
	FirewallUnit  = "harpo-killswitch.service"
	OutboundTag   = "harpo-exit"
	IptablesChain = "HARPO_KILLSWITCH"
	NftTable      = "harpo_killswitch"

	// Windows-specific identifiers
	WinProxyTask      = "HarpoProxy"
	WinFirewallPrefix = "Harpo Kill-Switch - "
)

// IsWindows reports whether harpo is running on Windows.
func IsWindows() bool {
	return runtime.GOOS == "windows"
}

// SavedSub represents a saved subscription URL entry.
type SavedSub struct {
	Name string `json:"name"`
	URL  string `json:"url"`
}

// Paths holds all resolved filesystem paths used by harpo.
type Paths struct {
	Home          string
	CfgDir        string
	DataDir       string
	LogsDir       string
	ProxyLog      string
	ProxyPID      string
	CfgPath       string
	BackupPath    string
	ExpectedIP    string
	TimezonePath  string
	SubsPath      string
	ActiveProfile string
	ThroneDB      string
}

// DefaultPaths resolves runtime paths honoring HARPO_CONFIG_DIR / HOME / APPDATA / LOCALAPPDATA.
func DefaultPaths() Paths {
	home := resolveHome()
	cfgDir := defaultCfgDir(home)
	dataDir := defaultDataDir(home)
	logsDir := filepath.Join(dataDir, "logs")
	return Paths{
		Home:          home,
		CfgDir:        cfgDir,
		DataDir:       dataDir,
		LogsDir:       logsDir,
		ProxyLog:      filepath.Join(logsDir, "xray.log"),
		ProxyPID:      filepath.Join(dataDir, "xray.pid"),
		CfgPath:       filepath.Join(cfgDir, "config.json"),
		BackupPath:    filepath.Join(cfgDir, "config.json.bak"),
		ExpectedIP:    filepath.Join(cfgDir, "expected-ip"),
		TimezonePath:  filepath.Join(cfgDir, "timezone"),
		SubsPath:      filepath.Join(cfgDir, "subscriptions.json"),
		ActiveProfile: filepath.Join(cfgDir, "active-profile"),
		ThroneDB:      defaultThroneDB(home),
	}
}

// EnsureCfgDir ensures the config directory exists with mode 0700 (user-only on Unix).
func EnsureCfgDir(dir string) error {
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return err
	}
	if runtime.GOOS != "windows" {
		return os.Chmod(dir, 0o700)
	}
	return nil
}

// WriteSecretText writes text atomically with mode 0600 inside dir.
func WriteSecretText(path, text string) error {
	dir := filepath.Dir(path)
	if err := EnsureCfgDir(dir); err != nil {
		return err
	}
	tmp, err := os.CreateTemp(dir, "."+filepath.Base(path)+".*")
	if err != nil {
		return err
	}
	tmpName := tmp.Name()
	cleanup := func() {
		_ = tmp.Close()
		_ = os.Remove(tmpName)
	}
	if runtime.GOOS != "windows" {
		if err := tmp.Chmod(0o600); err != nil {
			cleanup()
			return err
		}
	}
	if _, err := tmp.WriteString(text); err != nil {
		cleanup()
		return err
	}
	if err := tmp.Sync(); err != nil {
		cleanup()
		return err
	}
	if err := tmp.Close(); err != nil {
		_ = os.Remove(tmpName)
		return err
	}
	if err := os.Rename(tmpName, path); err != nil {
		_ = os.Remove(tmpName)
		return err
	}
	return nil
}

// ReadTrimmed reads a UTF-8 text file and trims surrounding whitespace.
func ReadTrimmed(path string) string {
	b, err := os.ReadFile(path)
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(b))
}

// IsIPLiteral reports whether host is a valid IPv4 or IPv6 literal.
func IsIPLiteral(host string) bool {
	_, err := netip.ParseAddr(strings.TrimSpace(host))
	return err == nil
}

// IsValidTimezone validates an IANA timezone name across any OS/distro (using embedded tzdata + system zoneinfo).
func IsValidTimezone(tz string) bool {
	tz = strings.TrimSpace(tz)
	if tz == "" || strings.Contains(tz, "..") || strings.HasPrefix(tz, "/") {
		return false
	}
	if _, err := time.LoadLocation(tz); err == nil {
		return true
	}
	for _, base := range []string{
		os.Getenv("HARPO_TZDATA"),
		os.Getenv("TZDIR"),
		"/usr/share/zoneinfo",
		"/etc/zoneinfo",
	} {
		if base == "" {
			continue
		}
		if fi, err := os.Stat(filepath.Join(base, tz)); err == nil && !fi.IsDir() {
			return true
		}
	}
	return false
}

// ResolveBin resolves an executable from optional env overrides or PATH.
func ResolveBin(env1, env2, fallback string) string {
	for _, k := range []string{env1, env2} {
		if k != "" {
			if v := strings.TrimSpace(os.Getenv(k)); v != "" {
				return v
			}
		}
	}
	if p, err := exec.LookPath(fallback); err == nil {
		return p
	}
	return fallback
}

// LoadSavedSubs loads saved subscriptions from subscriptions.json.
func LoadSavedSubs(path string) []SavedSub {
	b, err := os.ReadFile(path)
	if err != nil {
		return nil
	}
	var subs []SavedSub
	if err := json.Unmarshal(b, &subs); err != nil {
		return nil
	}
	out := make([]SavedSub, 0, len(subs))
	for _, s := range subs {
		if strings.TrimSpace(s.URL) != "" {
			if s.Name == "" {
				s.Name = "sub"
			}
			out = append(out, s)
		}
	}
	return out
}

// SaveSubs writes saved subscriptions atomically with mode 0600.
func SaveSubs(path string, subs []SavedSub) error {
	if subs == nil {
		subs = []SavedSub{}
	}
	b, err := json.MarshalIndent(subs, "", "  ")
	if err != nil {
		return err
	}
	return WriteSecretText(path, string(b)+"\n")
}

// CopyFileSecret copies src to dst with 0600 permissions atomically.
func CopyFileSecret(src, dst string) error {
	b, err := os.ReadFile(src)
	if err != nil {
		return fmt.Errorf("read %s: %w", src, err)
	}
	return WriteSecretText(dst, string(b))
}
