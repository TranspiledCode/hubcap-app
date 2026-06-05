// button.go — reusable key-hint button for the footer bar.
//
// A KeyButton is a small bordered label that looks like a physical key:
//
//   ╭───────╮
//   │ enter │  open
//   ╰───────╯
//
// Usage:
//
//   btn := NewKeyButton("enter", "open", lipgloss.Color("83"))
//   row := btn.Render()        // 3-row string ready for JoinHorizontal
//
// Build a complete footer bar:
//
//   bar := RenderFooterBar(width, btn1, btn2, btn3)
package main

import (
	"strings"

	"github.com/charmbracelet/lipgloss"
)

// footerBg is the background colour shared by all footer elements.
const footerBg = lipgloss.Color("235")

// FooterColor groups the three semantic colours used in the footer.
var (
	ColorAction  = lipgloss.Color("83")  // green  — primary action keys
	ColorMeta    = lipgloss.Color("208") // amber  — structural / meta keys
	ColorDanger  = lipgloss.Color("196") // red    — destructive keys
)

// KeyButton is a key label paired with a short description.
type KeyButton struct {
	Key   string           // text displayed inside the border  (e.g. "enter")
	Desc  string           // label shown to the right          (e.g. "open")
	Color lipgloss.Color   // border + key foreground colour
}

// NewKeyButton is a convenience constructor.
func NewKeyButton(key, desc string, color lipgloss.Color) KeyButton {
	return KeyButton{Key: key, Desc: desc, Color: color}
}

// Render returns a 3-row string:
//
//   ╭───────╮
//   │ enter │  open
//   ╰───────╯
//
// The description is vertically centred alongside the border via
// lipgloss.JoinHorizontal(Center, …).
func (b KeyButton) Render() string {
	border := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(b.Color).
		BorderBackground(footerBg).
		Foreground(b.Color).
		Background(footerBg).
		Padding(0, 1).
		Bold(true).
		Render(b.Key)

	label := lipgloss.NewStyle().
		Foreground(lipgloss.Color("244")).
		Background(footerBg).
		Render(" " + b.Desc)

	return lipgloss.JoinHorizontal(lipgloss.Center, border, label)
}

// RenderFooterBar lays out a slice of KeyButtons horizontally, separated by
// small gaps, and pads every row to width so the footer fills the terminal.
// The returned string is always 3 rows tall.
func RenderFooterBar(width int, buttons ...KeyButton) string {
	bgSt := lipgloss.NewStyle().Background(footerBg)
	edge := bgSt.Render("  ")
	gap  := bgSt.Render("   ")

	parts := make([]string, 0, len(buttons)*2+2)
	parts = append(parts, edge)
	for i, btn := range buttons {
		if i > 0 {
			parts = append(parts, gap)
		}
		parts = append(parts, btn.Render())
	}
	parts = append(parts, edge)

	joined := lipgloss.JoinHorizontal(lipgloss.Center, parts...)

	// Ensure every row fills the full width.
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

// CenterInFooterBar wraps a single-line string (e.g. a toast message) in a
// 3-row block so it occupies the same height as a normal button footer bar.
func CenterInFooterBar(content string, width int) string {
	bgSt  := lipgloss.NewStyle().Background(footerBg)
	blank := bgSt.Width(width).Render("")
	w := lipgloss.Width(content)
	switch {
	case w < width:
		content += bgSt.Render(strings.Repeat(" ", width-w))
	case w > width:
		content = lipgloss.NewStyle().MaxWidth(width).Render(content)
	}
	return blank + "\n" + content + "\n" + blank
}
