package runner

import (
	"runtime"
	"strings"

	"harpo/internal/apps"
)

// RunOptions customizes how a command is executed inside the kill-switch sandbox.
type RunOptions struct {
	Args           []string
	Mode           apps.AppMode
	ExtraEnv       map[string]string
	AppName        string // If set, re-execs via `harpo launch <AppName>` so apps.json flags/env survive slice entry.
	InvokedAsAlias bool
	ForceSoft      bool // Windows: allow soft proxy-only mode without warning
	ForceStrict    bool // Windows: refuse to launch if Windows Firewall block rule is missing
}

// Run executes args inside the kill-switched sandbox with strict fail-closed verification.
func Run(args []string, invokedAsAlias bool) int {
	mode := apps.ModeCLI
	forceSoft := false
	forceStrict := false
	if !invokedAsAlias {
		for len(args) > 0 && strings.HasPrefix(args[0], "--") {
			switch args[0] {
			case "--electron":
				mode = apps.ModeElectron
				args = args[1:]
			case "--chromium":
				mode = apps.ModeChromium
				args = args[1:]
			case "--proxychains":
				mode = apps.ModeProxychains
				args = args[1:]
			case "--soft":
				forceSoft = true
				args = args[1:]
			case "--strict":
				forceStrict = true
				args = args[1:]
			case "--":
				args = args[1:]
				goto doneFlags
			default:
				goto doneFlags
			}
		}
	}
doneFlags:
	return RunWithOptions(RunOptions{
		Args:           args,
		Mode:           mode,
		InvokedAsAlias: invokedAsAlias,
		ForceSoft:      forceSoft,
		ForceStrict:    forceStrict,
	})
}

func mergeEnv(base []string, overrides map[string]string) []string {
	out := make([]string, 0, len(base)+len(overrides))
	seen := make(map[string]bool, len(overrides))

	if runtime.GOOS == "windows" {
		lowMap := make(map[string]string, len(overrides))
		keyCase := make(map[string]string, len(overrides))
		for k, v := range overrides {
			lk := strings.ToLower(k)
			lowMap[lk] = v
			keyCase[lk] = k
		}
		for _, kv := range base {
			k, _, _ := strings.Cut(kv, "=")
			lk := strings.ToLower(k)
			if v, ok := lowMap[lk]; ok {
				if !seen[lk] {
					out = append(out, keyCase[lk]+"="+v)
					seen[lk] = true
				}
			} else {
				out = append(out, kv)
			}
		}
		for lk, v := range lowMap {
			if !seen[lk] {
				out = append(out, keyCase[lk]+"="+v)
			}
		}
		return out
	}

	for _, kv := range base {
		k, _, _ := strings.Cut(kv, "=")
		if v, ok := overrides[k]; ok {
			out = append(out, k+"="+v)
			seen[k] = true
		} else {
			out = append(out, kv)
		}
	}
	for k, v := range overrides {
		if !seen[k] {
			out = append(out, k+"="+v)
		}
	}
	return out
}
