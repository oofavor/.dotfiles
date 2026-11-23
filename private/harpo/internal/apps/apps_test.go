package apps

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"harpo/internal/config"
)

func TestRegistryAndShimLifecycle(t *testing.T) {
	td := t.TempDir()
	p := config.Paths{
		Home:   td,
		CfgDir: filepath.Join(td, ".config", "harpo"),
	}

	entry := AppEntry{
		Name:            "demo-electron",
		Mode:            ModeElectron,
		Enabled:         true,
		DesktopEntry:    true,
		DesktopName:     "Demo Electron",
		IsolatedProfile: true,
		Env:             map[string]string{"DISABLE_TELEMETRY": "1"},
	}

	if err := Upsert(p, entry); err != nil {
		t.Fatalf("Upsert failed: %v", err)
	}
	if !HasShim(p, "demo-electron") {
		t.Fatal("expected shim symlink to exist")
	}
	if !HasDesktopOverride(p, "demo-electron") {
		t.Fatal("expected managed .desktop override to exist")
	}

	got, err := Find(p, "demo-electron")
	if err != nil || got == nil || got.Mode != ModeElectron || got.Env["DISABLE_TELEMETRY"] != "1" {
		t.Fatalf("unexpected Find result: %+v err=%v", got, err)
	}

	// BuildExecArgs injects anti-leak Chromium flags + isolated user-data-dir.
	argv, err := BuildExecArgs(p, *got, "/usr/bin/demo-electron", []string{"--foo"})
	if err != nil {
		t.Fatalf("BuildExecArgs: %v", err)
	}
	joined := strings.Join(argv, " ")
	for _, want := range []string{
		"--proxy-server=socks5://127.0.0.1:10808",
		"--host-resolver-rules=",
		"--force-webrtc-ip-handling-policy=disable_non_proxied_udp",
		"--disable-quic",
		"--user-data-dir=",
		"--foo",
	} {
		if !strings.Contains(joined, want) {
			t.Fatalf("missing %q in argv: %s", want, joined)
		}
	}

	// Disable should remove shim & desktop override while keeping entry in apps.json.
	entry.Enabled = false
	if err := Upsert(p, entry); err != nil {
		t.Fatalf("disable Upsert: %v", err)
	}
	if HasShim(p, "demo-electron") || HasDesktopOverride(p, "demo-electron") {
		t.Fatal("expected shim and desktop override to be removed when disabled")
	}

	// Remove cleans everything.
	if err := Remove(p, "demo-electron"); err != nil {
		t.Fatalf("Remove: %v", err)
	}
	if got, _ := Find(p, "demo-electron"); got != nil {
		t.Fatal("expected app to be removed")
	}
}

func TestResolveRealBinarySkipsShimsDir(t *testing.T) {
	td := t.TempDir()
	shims := filepath.Join(td, "shims")
	realDir := filepath.Join(td, "usr-bin")
	if err := os.MkdirAll(shims, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(realDir, 0o755); err != nil {
		t.Fatal(err)
	}

	fakeShim := filepath.Join(shims, "mytool")
	realBin := filepath.Join(realDir, "mytool")
	if err := os.WriteFile(fakeShim, []byte("#!/bin/sh\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(realBin, []byte("#!/bin/sh\n"), 0o755); err != nil {
		t.Fatal(err)
	}

	origPath := os.Getenv("PATH")
	t.Setenv("PATH", shims+string(os.PathListSeparator)+realDir+string(os.PathListSeparator)+origPath)

	got, err := ResolveRealBinary("mytool", shims)
	if err != nil {
		t.Fatalf("ResolveRealBinary failed: %v", err)
	}
	if got != realBin {
		t.Fatalf("got %q, want %q (shims dir should be skipped!)", got, realBin)
	}
}

func TestInstallShellPATHIdempotent(t *testing.T) {
	td := t.TempDir()
	p := config.Paths{Home: td, CfgDir: filepath.Join(td, ".config", "harpo")}
	if err := os.MkdirAll(filepath.Join(td, ".config", "fish"), 0o755); err != nil {
		t.Fatal(err)
	}
	bashrc := filepath.Join(td, ".bashrc")
	if err := os.WriteFile(bashrc, []byte("# existing bashrc\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	// Simulate a read-only Nix symlink for .zshrc — must be safely skipped.
	roTarget := filepath.Join(td, "nix-zshrc")
	_ = os.WriteFile(roTarget, []byte("# ro\n"), 0o444)
	_ = os.Symlink(roTarget, filepath.Join(td, ".zshrc"))

	updated, err := InstallShellPATH(p)
	if err != nil {
		t.Fatalf("InstallShellPATH: %v", err)
	}
	if len(updated) < 3 {
		t.Fatalf("expected env.d + fish + bashrc updated, got %v", updated)
	}

	// Verify files written.
	envFile := filepath.Join(td, ".config", "environment.d", "10-harpo.conf")
	fishFile := filepath.Join(td, ".config", "fish", "conf.d", "00-harpo.fish")
	for _, f := range []string{envFile, fishFile, bashrc} {
		b, err := os.ReadFile(f)
		if err != nil || !strings.Contains(string(b), "$HOME/.local/share/harpo/shims") {
			t.Fatalf("%s missing shims path: err=%v body=%q", f, err, string(b))
		}
	}

	// Running a second time must NOT duplicate the line in .bashrc.
	if _, err := InstallShellPATH(p); err != nil {
		t.Fatalf("second InstallShellPATH: %v", err)
	}
	b, _ := os.ReadFile(bashrc)
	if strings.Count(string(b), "harpo-shims") != 1 {
		t.Fatalf("expected 1 harpo-shims line in .bashrc, got:\n%s", string(b))
	}
}

func TestDefaultArgsAndCustomProfileDir(t *testing.T) {
	td := t.TempDir()
	p := config.Paths{Home: td, CfgDir: filepath.Join(td, ".config", "harpo")}
	a := AppEntry{
		Name:            "claude-web",
		Mode:            ModeChromium,
		Command:         "chromium",
		IsolatedProfile: true,
		ProfileDir:      "~/.local/share/claude-chromium",
		ExtraArgs:       []string{"--class=claude-web", "--lang=en-US"},
		DefaultArgs:     []string{"https://claude.ai"},
	}

	// Empty userArgs -> DefaultArgs included
	argv1, err := BuildExecArgs(p, a, "/usr/bin/chromium", nil)
	if err != nil {
		t.Fatal(err)
	}
	j1 := strings.Join(argv1, " ")
	wantDir := "--user-data-dir=" + filepath.Join(td, ".local", "share", "claude-chromium")
	if !strings.Contains(j1, wantDir) || !strings.HasSuffix(j1, "https://claude.ai") {
		t.Fatalf("unexpected argv1: %s", j1)
	}

	// Explicit userArgs -> replaces DefaultArgs
	argv2, err := BuildExecArgs(p, a, "/usr/bin/chromium", []string{"https://claude.ai/new"})
	if err != nil {
		t.Fatal(err)
	}
	j2 := strings.Join(argv2, " ")
	if strings.Contains(j2, " https://claude.ai ") || !strings.HasSuffix(j2, "https://claude.ai/new") {
		t.Fatalf("expected userArgs to override DefaultArgs, got: %s", j2)
	}
}
