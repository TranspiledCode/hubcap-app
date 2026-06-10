// splash.go — welcome/splash screen shown at startup.
package main

import (
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// splashDoneMsg is sent when the splash auto-dismiss timer expires.
type splashDoneMsg struct{}

// splashTimerCmd fires a splashDoneMsg after a short delay.
func splashTimerCmd() tea.Cmd {
	return tea.Tick(3*time.Second, func(time.Time) tea.Msg {
		return splashDoneMsg{}
	})
}

// splashLogo is a 5-line block-letter rendering of "HUBCAP".
// Each letter occupies 6 columns; letters are separated by 2 spaces (total 46 cols).
//
//	H  ██  ██   U  ██  ██   B  █████    C   ████    A    ██    P  ████
//	   ██  ██      ██  ██      ██  ██      ██          █  █       ██  ██
//	   ██████      ██  ██      █████       ██          ██████      ████
//	   ██  ██      ██  ██      ██  ██      ██          ██  ██      ██
//	   ██  ██       ████       █████        ████       ██  ██      ██
var splashLogo = [5]string{
	"██  ██  ██  ██  █████    ████    ██    ████  ",
	"██  ██  ██  ██  ██  ██  ██      █  █   ██  ██",
	"██████  ██  ██  █████   ██      ██████  ████  ",
	"██  ██  ██  ██  ██  ██  ██      ██  ██  ██    ",
	"██  ██   ████   █████    ████   ██  ██  ██    ",
}

const splashTagline = "GitHub issues & pull requests — right in your terminal."

// splashView renders the welcome screen centred inside the standard app border.
func splashView(m AppModel) string {
	if m.width == 0 || m.height == 0 {
		return ""
	}

	pal := m.palette
	innerW := m.width - 2
	innerH := m.height - 2

	bg := pal.BgBody

	bgSt := lipgloss.NewStyle().Background(bg)
	logoSt := lipgloss.NewStyle().Foreground(pal.Accent).Bold(true).Background(bg)
	tagSt := lipgloss.NewStyle().Foreground(pal.TextMuted).Background(bg)
	verSt := lipgloss.NewStyle().Foreground(pal.TextDim).Background(bg)
	hintSt := lipgloss.NewStyle().Foreground(pal.TextFaint).Background(bg)

	blankLine := bgSt.Width(innerW).Render("")

	// center returns s horizontally centred within w columns using bg-coloured padding.
	center := func(s string, w int) string {
		sw := lipgloss.Width(s)
		left := (w - sw) / 2
		if left < 0 {
			left = 0
		}
		right := w - left - sw
		if right < 0 {
			right = 0
		}
		return bgSt.Render(strings.Repeat(" ", left)) + s + bgSt.Render(strings.Repeat(" ", right))
	}

	// Content height: logo (5) + blank (1) + tagline (1) + version (1) + blank (1) + hint (1) = 10
	const contentH = 10
	topPad := (innerH - contentH) / 2
	if topPad < 1 {
		topPad = 1
	}

	var lines []string

	// Top padding
	for i := 0; i < topPad; i++ {
		lines = append(lines, blankLine)
	}

	// Logo
	for _, l := range splashLogo {
		lines = append(lines, center(logoSt.Render(l), innerW))
	}

	// Tagline + version
	lines = append(lines, blankLine)
	lines = append(lines, center(tagSt.Render(splashTagline), innerW))
	lines = append(lines, center(verSt.Render("v"+version), innerW))

	// Hint
	lines = append(lines, blankLine)
	lines = append(lines, center(hintSt.Render("press any key to continue"), innerW))

	// Bottom padding
	for len(lines) < innerH {
		lines = append(lines, blankLine)
	}
	if len(lines) > innerH {
		lines = lines[:innerH]
	}

	inner := strings.Join(lines, "\n")

	appBorder := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(pal.Title).
		Background(bg).
		Width(innerW)

	return appBorder.Render(inner)
}
