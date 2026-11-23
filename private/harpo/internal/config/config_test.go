package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestWriteSecretTextAtomicPerms(t *testing.T) {
	td := t.TempDir()
	target := filepath.Join(td, "secret")
	if err := WriteSecretText(target, "top-secret\n"); err != nil {
		t.Fatalf("WriteSecretText: %v", err)
	}
	if got := ReadTrimmed(target); got != "top-secret" {
		t.Fatalf("got %q, want top-secret", got)
	}
	fi, err := os.Stat(target)
	if err != nil {
		t.Fatalf("Stat: %v", err)
	}
	if fi.Mode().Perm() != 0o600 {
		t.Fatalf("mode=%04o, want 0600", fi.Mode().Perm())
	}
	entries, _ := os.ReadDir(td)
	if len(entries) != 1 {
		t.Fatalf("expected 1 file in dir, got %d", len(entries))
	}
}

func TestIsValidTimezoneEmbedded(t *testing.T) {
	if !IsValidTimezone("America/New_York") || !IsValidTimezone("Europe/Amsterdam") {
		t.Fatal("expected valid IANA timezones to pass")
	}
	if IsValidTimezone("../etc/passwd") || IsValidTimezone("Not/A_Zone") {
		t.Fatal("expected invalid timezones to fail")
	}
}
