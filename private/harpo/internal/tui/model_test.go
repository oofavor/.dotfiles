package tui

import (
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/x/ansi"

	"harpo/internal/apps"
	"harpo/internal/probe"
	"harpo/internal/xray"
)

func assertFitsViewport(t *testing.T, label, view string, width, height int) {
	t.Helper()
	lines := strings.Split(view, "\n")
	if len(lines) > height {
		t.Fatalf("%s: vertical overflow: got %d lines, max %d", label, len(lines), height)
	}
	for i, line := range lines {
		w := ansi.StringWidth(line)
		if w > width {
			t.Fatalf("%s: horizontal overflow on line %d: got width %d > %d (line=%q)", label, i, w, width, ansi.Strip(line))
		}
	}
}

func TestZeroAutomaticNetworkProbingOnStartup(t *testing.T) {
	m := New(StartupOptions{})
	if m.exitProbing || m.exitProbed || m.domBatchActive {
		t.Fatal("expected zero automatic external probing on startup")
	}
}

func TestAllTabsStrictlyFitViewport(t *testing.T) {
	m := New(StartupOptions{})
	m.summary.Name = "🤖 Claude — Основной супер длинный профиль для теста переполнения"
	m.summary.Server = "198.51.100.10:2053"
	m.summary.SNI = "very.long.subdomain.example.org"
	m.summary.ExpectedIP = "203.0.113.42"
	m.summary.Distro = "NixOS 26.11 (Zokor)"
	m.exitProbed = true
	m.exitRes = probe.Result{
		IP:       "203.0.113.42",
		City:     "Frankfurt am Main",
		Country:  "DE",
		Org:      "AS64512 Example Telecom GmbH",
		Timezone: "Europe/Berlin",
	}
	m.ksStatus = probe.KillSwitchStatus{Blocked: true}
	m.logsText = "2026-11-23T16:00:00Z info proxy started on 127.0.0.1:10809\n2026-11-23T16:00:05Z warning very long log line " + strings.Repeat("abcdef ", 30)
	m.appList = []apps.AppEntry{
		{Name: "claude-desktop", Mode: apps.ModeElectron, Enabled: true, DesktopEntry: true, IsolatedProfile: true},
		{Name: "claude", Mode: apps.ModeCLI, Enabled: true, Env: map[string]string{"DISABLE_TELEMETRY": "1"}},
	}
	m.targetStates[0] = domainProbeState{
		done: true,
		res:  probe.DomainProbeResult{StatusCode: 404, Proto: "h2", RTTMS: 420, Verdict: "Reachable", Healthy: true},
	}

	sizes := [][2]int{{80, 24}, {100, 30}, {120, 36}, {160, 45}}
	for _, sz := range sizes {
		um, _ := m.Update(tea.WindowSizeMsg{Width: sz[0], Height: sz[1]})
		m = um.(Model)
		m.logsVP.SetContent(colorizeLogs(m.logsText, m.innerWidth()))
		m.dashLogsVP.SetContent(colorizeLogs(m.logsText, m.innerWidth()))

		for tab := tabDashboard; tab <= tabSystem; tab++ {
			m.tab = tab
			switch tab {
			case tabDashboard:
				for _, bm := range []dashBottomMode{dashBottomLogs, dashBottomProcs} {
					m.dashBottom = bm
					assertFitsViewport(t, "dashboard", m.View(), sz[0], sz[1])
				}
			case tabSystem:
				for st := sysTests; st <= sysLogs; st++ {
					m.sysTab = st
					assertFitsViewport(t, "system", m.View(), sz[0], sz[1])
				}
			default:
				assertFitsViewport(t, "tab", m.View(), sz[0], sz[1])
			}
		}
	}
}

func TestModalsStrictlyFitViewport(t *testing.T) {
	m := New(StartupOptions{})
	um, _ := m.Update(tea.WindowSizeMsg{Width: 80, Height: 24})
	m = um.(Model)

	m.modal = modalHelp
	assertFitsViewport(t, "modalHelp", m.View(), 80, 24)

	m.modal = modalInputAddTarget
	assertFitsViewport(t, "modalInputAddTarget", m.View(), 80, 24)

	um, _ = m.openAppDiscoverModal()
	m = um.(Model)
	assertFitsViewport(t, "modalAppDiscover", m.View(), 80, 24)

	um, _ = m.openAppEditModal(apps.AppEntry{Name: "claude-desktop", Mode: apps.ModeElectron, Enabled: true}, true)
	m = um.(Model)
	assertFitsViewport(t, "modalAppEdit", m.View(), 80, 24)

	p, err := xray.ParseVLESSURI("vless://11111111-2222-3333-4444-555555555555@example.com:443?type=tcp&security=tls&sni=example.com#Demo")
	if err != nil {
		t.Fatal(err)
	}
	_ = m.openActivateModal(*p)
	assertFitsViewport(t, "modalActivate", m.View(), 80, 24)
}
