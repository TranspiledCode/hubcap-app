// button.go — reusable key-hint button for the footer bar.
//
// Three visual themes are supported (set via Config.UITheme):
//
//	minimal     — plain text:   a  assign
//	default     — bracket hint: [ a ]  assign
//	comfortable — 3-row box:    ╭───╮
//	                            │ a │  assign
//	                            ╰───╯
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
	case ThemeMinimal, ThemeDefault, ThemeComfortable:
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

// footerBg is the background colour for minimal/default themes.
// comfortableBg is a slightly darker gray for the comfortable 3-row footer strip,
// providing a subtle visual separation from the content area without stark contrast.
const (
	footerBg      = lipgloss.Color("235")
	comfortableBg = lipgloss.Color("234")
)

// themeBg returns the correct footer background for the given theme.
func themeBg(theme UITheme) lipgloss.Color {
	if theme == ThemeComfortable {
		return comfortableBg
	}
	return footerBg
}

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
//	minimal     → plain text:     a  assign
//	default     → bracket hint:   [ a ]  assign
//	comfortable → 3-row box:      ╭───╮
//	                              │ a │  assign
//	                              ╰───╯
func (b KeyButton) Render(theme UITheme) string {
	bg     := themeBg(theme)
	keySt  := lipgloss.NewStyle().Foreground(b.Color).Background(bg).Bold(true)
	descSt := lipgloss.NewStyle().Foreground(lipgloss.Color("244")).Background(bg)

	switch theme {

	// ── Minimal: plain bold key + muted description ───────────────────────────
	case ThemeMinimal:
		return keySt.Render(b.Key) + descSt.Render("  "+b.Desc)

	// ── Default: [ key ]  description ────────────────────────────────────────
	case ThemeDefault:
		bracketSt := lipgloss.NewStyle().Foreground(b.Color).Background(bg)
		return bracketSt.Render("[") +
			keySt.Render(" "+b.Key+" ") +
			bracketSt.Render("]") +
			descSt.Render("  "+b.Desc)

	// ── Comfortable: 3-row rounded border box ────────────────────────────────
	default: // ThemeComfortable (and any unknown value)
		border := lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(b.Color).
			BorderBackground(bg).
			Foreground(b.Color).
			Background(bg).
			Padding(0, 1).
			Bold(true).
			Render(b.Key)

		// Height(3) + AlignVertical(Center) makes the label occupy the same
		// 3 rows as the border box so the background fills the full height —
		// without this the 1-row description text leaves transparent rows
		// above and below it when joined alongside the taller button.
		label := lipgloss.NewStyle().
			Foreground(lipgloss.Color("244")).
			Background(bg).
			Height(3).
			AlignVertical(lipgloss.Center).
			Render(" " + b.Desc)

		return lipgloss.JoinHorizontal(lipgloss.Center, border, label)
	}
}

// singleRowBar builds a single-row padded footer string from already-rendered
// button strings. Used by minimal and default themes.
func singleRowBar(width int, theme UITheme, buttons ...KeyButton) string {
	bgSt := lipgloss.NewStyle().Background(themeBg(theme))
	edge := bgSt.Render("  ")
	gap  := bgSt.Render("   ")

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
}

// RenderFooterBar lays out buttons separated by gaps and pads to width.
// Minimal and Default produce a 1-row result; Comfortable produces 3 rows.
func RenderFooterBar(width int, theme UITheme, buttons ...KeyButton) string {
	bgSt := lipgloss.NewStyle().Background(themeBg(theme))

	// ── Comfortable: 3-row JoinHorizontal layout ──────────────────────────────
	if theme == ThemeComfortable {
		// Width + Height make each spacer a solid background rectangle that
		// matches the button height — a 1-row spacer would leave transparent
		// rows above and below when joined alongside the 3-row buttons.
		edge := bgSt.Width(2).Height(3).Render("")
		gap  := bgSt.Width(3).Height(3).Render("")
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
		// Separator line above the button strip.
		sep := lipgloss.NewStyle().
			Foreground(lipgloss.Color("238")).
			Background(themeBg(theme)).
			Render(strings.Repeat("─", width))

		return sep + "\n" + strings.Join(rows, "\n")
	}

	// ── Minimal / Default: single row ─────────────────────────────────────────
	return singleRowBar(width, theme, buttons...)
}

// CenterInFooterBar wraps a single-line string (toast, spinner) to match the
// height of RenderFooterBar for the same theme (1 row for minimal/default,
// 3 rows for comfortable).
func CenterInFooterBar(content string, width int, theme UITheme) string {
	bgSt := lipgloss.NewStyle().Background(themeBg(theme))
	w := lipgloss.Width(content)
	switch {
	case w < width:
		content += bgSt.Render(strings.Repeat(" ", width-w))
	case w > width:
		content = lipgloss.NewStyle().MaxWidth(width).Render(content)
	}
	if theme != ThemeComfortable {
		return content
	}
	// Comfortable: separator + 3-row block to match RenderFooterBar height.
	sep   := lipgloss.NewStyle().Foreground(lipgloss.Color("238")).Background(themeBg(theme)).Render(strings.Repeat("─", width))
	blank := bgSt.Width(width).Render("")
	return sep + "\n" + blank + "\n" + content + "\n" + blank
}
