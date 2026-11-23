package tui

import (
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/x/ansi"

	"harpo/internal/apps"
)

// Terminal-native ANSI-16 palette (inherits Ghostty / Stylix theme automatically; transparent background).
var (
	ansiDim     = lipgloss.Color("8")  // bright black / muted gray
	ansiRed     = lipgloss.Color("1")  // red
	ansiGreen   = lipgloss.Color("2")  // green
	ansiYellow  = lipgloss.Color("3")  // yellow / amber
	ansiBlue    = lipgloss.Color("4")  // blue / primary accent
	ansiMagenta = lipgloss.Color("5")  // magenta / secondary accent
	ansiCyan    = lipgloss.Color("6")  // cyan / highlight
	ansiWhite   = lipgloss.Color("15") // bright white

	styleBrand = lipgloss.NewStyle().
			Bold(true).
			Foreground(ansiBlue)

	styleTabActive = lipgloss.NewStyle().
			Bold(true).
			Underline(true).
			Foreground(ansiWhite)

	styleTabInactive = lipgloss.NewStyle().
				Foreground(ansiDim)

	styleSubActive = lipgloss.NewStyle().
			Bold(true).
			Underline(true).
			Foreground(ansiCyan)

	styleSubInactive = lipgloss.NewStyle().
				Foreground(ansiDim)

	styleHeading = lipgloss.NewStyle().
			Bold(true).
			Foreground(ansiCyan)

	styleLabel = lipgloss.NewStyle().
			Foreground(ansiDim)

	styleValue = lipgloss.NewStyle().
			Bold(true)

	styleDim = lipgloss.NewStyle().
			Foreground(ansiDim)

	styleGreen   = lipgloss.NewStyle().Foreground(ansiGreen).Bold(true)
	styleRed     = lipgloss.NewStyle().Foreground(ansiRed).Bold(true)
	styleAmber   = lipgloss.NewStyle().Foreground(ansiYellow).Bold(true)
	styleCyan    = lipgloss.NewStyle().Foreground(ansiCyan).Bold(true)
	styleMagenta = lipgloss.NewStyle().Foreground(ansiMagenta).Bold(true)
	styleBlue    = lipgloss.NewStyle().Foreground(ansiBlue).Bold(true)

	styleSelected = lipgloss.NewStyle().
			Bold(true).
			Foreground(ansiCyan)

	styleNormal = lipgloss.NewStyle()
)

// fitLine truncates any ANSI-styled string strictly to maxW display cells without breaking ANSI escapes.
func fitLine(s string, maxW int) string {
	if maxW <= 0 {
		return ""
	}
	if ansi.StringWidth(s) <= maxW {
		return s
	}
	if maxW == 1 {
		return ansi.Truncate(s, 1, "")
	}
	return ansi.Truncate(s, maxW, "…")
}

// padCell fits s strictly to width terminal cells (truncating or right-padding cleanly).
func padCell(s string, width int) string {
	if width <= 0 {
		return ""
	}
	s = fitLine(s, width)
	w := ansi.StringWidth(s)
	if w < width {
		return s + strings.Repeat(" ", width-w)
	}
	return s
}

// hairline draws a subtle horizontal divider across innerW cells.
func hairline(innerW int) string {
	if innerW < 4 {
		innerW = 4
	}
	return styleDim.Render(strings.Repeat("─", innerW))
}

// sectionHeader renders a clean borderless section heading with optional right-aligned subtle metadata.
func sectionHeader(title, rightMeta string, innerW int) string {
	left := styleHeading.Render(title)
	if rightMeta == "" {
		return fitLine(left, innerW)
	}
	right := styleDim.Render(rightMeta)
	lw := ansi.StringWidth(left)
	rw := ansi.StringWidth(right)
	if lw+rw+2 <= innerW {
		return left + strings.Repeat(" ", innerW-lw-rw) + right
	}
	return fitLine(left+"  "+right, innerW)
}

// renderModalBox renders a crisp, transparent-background thin rounded modal strictly within maxW x maxH.
func renderModalBox(lines []string, maxW, maxH int) string {
	w := min(84, max(44, maxW-6))
	innerW := w - 6 // 2 border + 4 horizontal padding
	innerMaxH := max(4, maxH-4)

	clamped := make([]string, 0, len(lines))
	for _, raw := range lines {
		for _, sub := range strings.Split(raw, "\n") {
			if len(clamped) >= innerMaxH {
				break
			}
			clamped = append(clamped, fitLine(sub, innerW))
		}
	}
	return lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(ansiBlue).
		Padding(1, 2).
		Width(w - 2).
		MaxWidth(w).
		MaxHeight(maxH).
		Render(strings.Join(clamped, "\n"))
}

func kvLine(label, val string, labelW int) string {
	return padCell(styleLabel.Render(label), labelW) + val
}

func dotState(state string) string {
	if state == "active" {
		return styleGreen.Render("● active")
	}
	if state == "unknown" || state == "" {
		return styleAmber.Render("○ " + orEmpty(state, "unknown"))
	}
	return styleRed.Render("● " + state)
}

func modeLabel(m apps.AppMode) string {
	switch m {
	case apps.ModeElectron:
		return styleMagenta.Render("electron")
	case apps.ModeChromium:
		return styleCyan.Render("chromium")
	case apps.ModeProxychains:
		return styleAmber.Render("proxychains")
	default:
		return styleDim.Render("cli")
	}
}

func checkMark(ok bool, okText, errText string) string {
	if ok {
		return styleGreen.Render("✓ ") + okText
	}
	return styleRed.Render("✗ ") + errText
}

func colorizeLogLine(l string) string {
	low := strings.ToLower(l)
	switch {
	case strings.Contains(low, "error") || strings.Contains(low, "failed") || strings.Contains(low, "panic"):
		return styleRed.Render(l)
	case strings.Contains(low, "warn"):
		return styleAmber.Render(l)
	default:
		return l
	}
}

func colorizeLogs(raw string, innerW int) string {
	if strings.TrimSpace(raw) == "" {
		return styleDim.Render("(no journal entries for harpo-proxy.service)")
	}
	lines := strings.Split(raw, "\n")
	out := make([]string, len(lines))
	for i, l := range lines {
		out[i] = fitLine(colorizeLogLine(l), innerW)
	}
	return strings.Join(out, "\n")
}
