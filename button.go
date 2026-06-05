// button.go — reusable key-hint button for the footer bar.
//
// Three visual themes are supported (set via Config.UITheme):
//
//	minimal     — single row:  │ enter │  open
//	default     — 3-row box:   ╭───────╮
//	                           │ enter │  open
//	                           ╰───────╯
//	comfortable — 3-row box with extra padding (wider terminals / accessibility)
//
// Usage:
//
//	btn := NewKeyButton("enter", "open", ColorAction)
//	row := btn.Render(theme)
//
//	bar := RenderFooterBar(width, theme, btn1, btn2, btn3)
package main

import (
	"strings"

	"github.com/charmbracelet/lipgloss"
)

// UITheme controls the visual density of the footer and forms.
type UITheme string

const (
	ThemeMinimal     UITheme = "minimal"
	ThemeDefault     UITheme = "default"
	ThemeComfortable UITheme = "comfortable"
)

// resolveTheme normalises an arbitrary string (e.g. from JSON) to a valid
// UITheme, falling back to ThemeDefault for unknown/empty values.
func resolveTheme(s string) UITheme {
	switch UITheme(s) {
	case ThemeMinimal, ThemeComfortable:
		return UITheme(s)
	default:
		return ThemeDefault
	}
}

// formWidth returns the appropriate huh form width for a given theme and the
// available inner model width.
func formWidth(modelWidth int, theme UITheme) int {
	switch theme {
	case ThemeComfortable:
		return modelWidth - 4
	case ThemeMinimal:
		return modelWidth - 10
	default:
		return modelWidth - 8
	}
}

// footerBg is the background colour shared by all footer elements.
const footerBg = lipgloss.Color("235")

// Semantic colours used throughout the footer.
var (
	ColorAction = lipgloss.Color("83")  // green — primary action keys
	ColorMeta   = lipgloss.Color("208") // amber — structural / meta keys
	ColorDanger = lipgloss.Color("196") // red   — destructive keys
)

// KeyButton is a key label paired with a short description.
type KeyButton struct {
	Key   string         // text shown inside the button  (e.g. "enter")
	Desc  string         // label to the right            (e.g. "open")
	Color lipgloss.Color // border + key foreground colour
}

// NewKeyButton is a convenience constructor.
func NewKeyButton(key, desc string, color lipgloss.Color) KeyButton {
	return KeyButton{Key: key, Desc: desc, Color: color}
}

// Render returns a styled string for the button at the requested theme density.
//
//	minimal      → single row: │ enter │  open
//	default      → 3-row rounded border, padding 1
//	comfortable  → 3-row rounded border, padding 2
func (b KeyButton) Render(theme UITheme) string {
	switch theme {

	// ── Minimal: single-row pipe borders ─────────────────────────────────────
	case ThemeMinimal:
		pipe := lipgloss.NewStyle().
			Foreground(b.Color).Background(footerBg).Render("│")
		text := lipgloss.NewStyle().
			Foreground(b.Color).Background(footerBg).Bold(true).
			Render(" " + b.Key + " ")
		desc := lipgloss.NewStyle().
			Foreground(lipgloss.Color("244")).Background(footerBg).
			Render(" " + b.Desc)
		return pipe + text + pipe + desc

	// ── Default / Comfortable: rounded border box ─────────────────────────────
	default:
		pad := 1
		if theme == ThemeComfortable {
			pad = 2
		}
		border := lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(b.Color).
			BorderBackground(footerBg).
			Foreground(b.Color).
			Background(footerBg).
			Padding(0, pad).
			Bold(true).
			Render(b.Key)

		label := lipgloss.NewStyle().
			Foreground(lipgloss.Color("244")).
			Background(footerBg).
			Render(" " + b.Desc)

		return lipgloss.JoinHorizontal(lipgloss.Center, border, label)
	}
}

// RenderFooterBar lays out buttons separated by gaps and pads to width.
// Minimal theme produces a 1-row result; Default/Comfortable produce 3 rows.
func RenderFooterBar(width int, theme UITheme, buttons ...KeyButton) string {
	bgSt := lipgloss.NewStyle().Background(footerBg)
	edge := bgSt.Render("  ")
	gap  := bgSt.Render("   ")

	switch theme {

	case ThemeMinimal:
		var sb strings.Builder
		sb.WriteString(edge)
		for i, btn := range buttons {
			if i > 0 {
				sb.WriteString(gap)
			}
			sb.WriteString(btn.Render(theme))
		}
		sb.WriteString(edge)
		line := sb.String()
		w := lipgloss.Width(line)
		switch {
		case w < width:
			line += bgSt.Render(strings.Repeat(" ", width-w))
		case w > width:
			line = lipgloss.NewStyle().MaxWidth(width).Render(line)
		}
		return line

	default: // ThemeDefault, ThemeComfortable
		parts := make([]string, 0, len(buttons)*2+2)
		parts = append(parts, edge)
		for i, btn := range buttons {
			if i > 0 {
				parts = append(parts, gap)
			}
			parts = append(parts, btn.Render(theme))
		}
		parts = append(parts, edge)

		joined := lipgloss.JoinHorizontal(lipgloss.Center, parts...)

		rows := strings.Split(joined, "\n")
		for i, row := range rows {
			w := lipgloss.Width(row)
			switch {
			case w < width:
				rows[i] = row + bgSt.Render(strings.Repeat(" ", width-w))
			case w > width:
				rows[i] = lipgloss.NewStyle().MaxWidth(width).Render(row)
			}
		}
		return strings.Join(rows, "\n")
	}
}

// CenterInFooterBar wraps a single-line string (toast, spinner) to match the
// height of RenderFooterBar for the same theme (1 row for minimal, 3 for others).
func CenterInFooterBar(content string, width int, theme UITheme) string {
	bgSt := lipgloss.NewStyle().Background(footerBg)
	w := lipgloss.Width(content)
	switch {
	case w < width:
		content += bgSt.Render(strings.Repeat(" ", width-w))
	case w > width:
		content = lipgloss.NewStyle().MaxWidth(width).Render(content)
	}
	if theme == ThemeMinimal {
		return content
	}
	// Default / Comfortable: wrap in 3-row block to match bordered button height.
	blank := bgSt.Width(width).Render("")
	return blank + "\n" + content + "\n" + blank
}
