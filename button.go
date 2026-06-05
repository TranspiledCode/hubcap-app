// button.go — reusable key-hint button for the footer bar.
//
// A KeyButton renders as a single-row label where the key text has all four
// edges of a box traced using text decorations + pipe characters:
//
//   │ enter │  open
//
// The overline decoration draws a line at the very top of the character cell,
// the underline at the very bottom, and │ pipes close the left and right sides.
// The result looks like a complete bordered box in a single terminal row.
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
	"github.com/muesli/termenv"
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
	Key   string         // text shown inside the button  (e.g. "enter")
	Desc  string         // label to the right            (e.g. "open")
	Color lipgloss.Color // border + key foreground colour
}

// NewKeyButton is a convenience constructor.
func NewKeyButton(key, desc string, color lipgloss.Color) KeyButton {
	return KeyButton{Key: key, Desc: desc, Color: color}
}

// Render returns a single-row string where all four sides of the button are
// visible:
//
//   │ enter │  open
//   ↑       ↑
//   overline+underline span the full width; │ close left and right.
func (b KeyButton) Render() string {
	p  := termenv.ColorProfile()
	fg := p.Color(string(b.Color))
	bg := p.Color(string(footerBg))

	// The full button — pipes + key text — gets overline, underline, bold, and
	// color applied as one unit so the top and bottom decoration lines run edge
	// to edge across the button (including the corner pipes).
	btn := termenv.String("│ "+b.Key+" │").
		Bold().
		Overline().
		Underline().
		Foreground(fg).
		Background(bg).
		String()

	desc := lipgloss.NewStyle().
		Foreground(lipgloss.Color("244")).
		Background(footerBg).
		Render(" " + b.Desc)

	return btn + desc
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
