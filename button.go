// button.go — reusable key-hint button for the footer bar.
//
// A KeyButton renders as a compact single-row label:
//
//   │ enter │  open
//
// Usage:
//
//	btn := NewKeyButton("enter", "open", ColorAction)
//	row := btn.Render()              // single-row string
//
// Build a complete footer bar:
//
//	bar := RenderFooterBar(width, btn1, btn2, btn3)
package main

import (
	"strings"

	"github.com/charmbracelet/lipgloss"
)

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
	Key   string         // text shown inside the bars  (e.g. "enter")
	Desc  string         // label to the right           (e.g. "open")
	Color lipgloss.Color // bar + key foreground colour
}

// NewKeyButton is a convenience constructor.
func NewKeyButton(key, desc string, color lipgloss.Color) KeyButton {
	return KeyButton{Key: key, Desc: desc, Color: color}
}

// Render returns a single-row string:  │ enter │  open
func (b KeyButton) Render() string {
	pipe := lipgloss.NewStyle().
		Foreground(b.Color).
		Background(footerBg).
		Render("│")

	text := lipgloss.NewStyle().
		Foreground(b.Color).
		Background(footerBg).
		Bold(true).
		Render(" " + b.Key + " ")

	desc := lipgloss.NewStyle().
		Foreground(lipgloss.Color("244")).
		Background(footerBg).
		Render(" " + b.Desc)

	return pipe + text + pipe + desc
}

// RenderFooterBar lays out buttons in a single row separated by gaps and pads
// the line to width so the footer fills the full terminal width.
func RenderFooterBar(width int, buttons ...KeyButton) string {
	bgSt := lipgloss.NewStyle().Background(footerBg)
	edge := bgSt.Render("  ")
	gap  := bgSt.Render("   ")

	var sb strings.Builder
	sb.WriteString(edge)
	for i, btn := range buttons {
		if i > 0 {
			sb.WriteString(gap)
		}
		sb.WriteString(btn.Render())
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

// CenterInFooterBar pads a single-line content string (toast, spinner) to
// width so it fills the footer row consistently with RenderFooterBar.
func CenterInFooterBar(content string, width int) string {
	bgSt := lipgloss.NewStyle().Background(footerBg)
	w := lipgloss.Width(content)
	switch {
	case w < width:
		content += bgSt.Render(strings.Repeat(" ", width-w))
	case w > width:
		content = lipgloss.NewStyle().MaxWidth(width).Render(content)
	}
	return content
}
