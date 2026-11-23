package system

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"time"

	"harpo/internal/config"
	"harpo/internal/xray"
)

// Summary represents the current live/static state of harpo's proxy and firewall.
type Summary struct {
	Name          string
	Server        string
	Security      string
	Network       string
	SNI           string
	Flow          string
	ExpectedIP    string
	CachedTZ      string
	ProxyState    string
	AnchorState   string
	FirewallState string
	Distro        string
	FirewallKind  string
	XrayBin       string
}

// LoadSummary gathers current config and OS service/daemon states (strictly local, zero external traffic).
func LoadSummary(p config.Paths) Summary {
	s := Summary{
		Name:          orDefault(config.ReadTrimmed(p.ActiveProfile), "(unnamed)"),
		Server:        "none",
		Security:      "-",
		Network:       "-",
		SNI:           "-",
		Flow:          "-",
		ExpectedIP:    orDefault(config.ReadTrimmed(p.ExpectedIP), "(unset)"),
		CachedTZ:      orDefault(config.ReadTrimmed(p.TimezonePath), "(auto)"),
		ProxyState:    ServiceActive(config.ProxyUnit, true),
		AnchorState:   ServiceActive(config.AnchorUnit, true),
		FirewallState: ServiceActive(config.FirewallUnit, false),
		Distro:        config.DetectDistro(),
		FirewallKind:  DetectFirewallBackend(),
		XrayBin:       config.ResolveXrayBin(),
	}

	if b, err := os.ReadFile(p.CfgPath); err == nil {
		var cfg map[string]any
		if json.Unmarshal(b, &cfg) == nil {
			if obs, ok := cfg["outbounds"].([]any); ok && len(obs) > 0 {
				if ob, ok := obs[0].(map[string]any); ok {
					prof := xray.ProfileFromOutbound(s.Name, "", ob)
					if prof.Host != "" {
						s.Server = fmt.Sprintf("%s:%d", prof.Host, prof.Port)
					}
					s.Security = prof.Security
					s.Network = prof.Network
					s.SNI = prof.SNI
					s.Flow = prof.Flow
				}
			}
		}
	}
	return s
}

func orDefault(v, def string) string {
	if strings.TrimSpace(v) == "" {
		return def
	}
	return v
}

// ActivateOptions controls profile activation behavior.
type ActivateOptions struct {
	PinIPv4          string
	UpdateExpectedIP string
	Timezone         string
}

// ActivateProfile atomically installs a profile, restarts the Harpo proxy, and rolls back on failure.
func ActivateProfile(p config.Paths, prof xray.Profile, opts ActivateOptions) error {
	if err := config.EnsureCfgDir(p.CfgDir); err != nil {
		return err
	}
	outbound := prof.Outbound
	if opts.PinIPv4 != "" {
		outbound = xray.SetOutboundHost(outbound, opts.PinIPv4)
	}

	hadBackup := false
	if _, err := os.Stat(p.CfgPath); err == nil {
		if err := config.CopyFileSecret(p.CfgPath, p.BackupPath); err != nil {
			return fmt.Errorf("backup config.json: %w", err)
		}
		hadBackup = true
	}

	newCfg := xray.BuildConfig(outbound, config.SocksPort, config.HTTPPort)
	cfgBytes, err := json.MarshalIndent(newCfg, "", "  ")
	if err != nil {
		return err
	}
	if err := config.WriteSecretText(p.CfgPath, string(cfgBytes)+"\n"); err != nil {
		return fmt.Errorf("write config.json: %w", err)
	}
	if err := config.WriteSecretText(p.ActiveProfile, prof.Name+"\n"); err != nil {
		return fmt.Errorf("write active-profile: %w", err)
	}

	_ = RestartService(config.ProxyUnit, true)
	time.Sleep(800 * time.Millisecond)

	if state := ServiceActive(config.ProxyUnit, true); state != "active" {
		// Automatic rollback
		if hadBackup {
			_ = config.CopyFileSecret(p.BackupPath, p.CfgPath)
			_ = RestartService(config.ProxyUnit, true)
		}
		return fmt.Errorf("harpo proxy failed to start (state=%s) — rolled back previous config", state)
	}

	if opts.UpdateExpectedIP != "" && config.IsIPLiteral(opts.UpdateExpectedIP) {
		_ = config.WriteSecretText(p.ExpectedIP, opts.UpdateExpectedIP+"\n")
	}
	if opts.Timezone != "" && config.IsValidTimezone(opts.Timezone) {
		_ = config.WriteSecretText(p.TimezonePath, opts.Timezone+"\n")
	}
	return nil
}
