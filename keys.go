// keys.go
package main

import "github.com/charmbracelet/bubbles/key"

// keyMap centralises every key binding used across the application.
// Handlers use key.Matches(msg, keys.Foo) instead of bare string comparisons,
// so renaming a binding only requires changing it here.
type keyMap struct {
	// ── Global / navigation ────────────────────────────────────────────────
	Tab       key.Binding
	ShiftTab  key.Binding
	Quit      key.Binding
	ForceQuit key.Binding
	Config    key.Binding
	Filters   key.Binding
	Help      key.Binding

	// ── List navigation ────────────────────────────────────────────────────
	Up      key.Binding
	Down    key.Binding
	Top     key.Binding
	Bottom  key.Binding
	Open    key.Binding
	New     key.Binding
	Refresh key.Binding

	// ── Detail (shared) ────────────────────────────────────────────────────
	Back    key.Binding
	Browser key.Binding
	CopyURL key.Binding

	// ── Issue detail ───────────────────────────────────────────────────────
	IssueClose   key.Binding
	IssueAssign  key.Binding
	IssueLabel   key.Binding
	IssueDevelop key.Binding
	IssuePR      key.Binding

	// ── PR detail ──────────────────────────────────────────────────────────
	PRCheckout key.Binding
	PRMerge    key.Binding
	PRClose    key.Binding
}

// keys is the single global key-binding registry used throughout the app.
var keys = keyMap{
	// ── Global ────────────────────────────────────────────────────────────
	Tab:       key.NewBinding(key.WithKeys("tab"), key.WithHelp("tab", "next tab")),
	ShiftTab:  key.NewBinding(key.WithKeys("shift+tab"), key.WithHelp("shift+tab", "prev tab")),
	Quit:      key.NewBinding(key.WithKeys("q"), key.WithHelp("q", "quit")),
	ForceQuit: key.NewBinding(key.WithKeys("ctrl+c")),
	Config:    key.NewBinding(key.WithKeys(","), key.WithHelp(",", "config")),
	Filters:   key.NewBinding(key.WithKeys("f"), key.WithHelp("f", "filters")),
	Help:      key.NewBinding(key.WithKeys("?"), key.WithHelp("?", "help")),

	// ── List navigation ───────────────────────────────────────────────────
	Up:      key.NewBinding(key.WithKeys("up", "k"), key.WithHelp("↑/k", "up")),
	Down:    key.NewBinding(key.WithKeys("down", "j"), key.WithHelp("↓/j", "down")),
	Top:     key.NewBinding(key.WithKeys("g", "home"), key.WithHelp("g", "top")),
	Bottom:  key.NewBinding(key.WithKeys("G", "end"), key.WithHelp("G", "bottom")),
	Open:    key.NewBinding(key.WithKeys("enter"), key.WithHelp("enter", "open")),
	New:     key.NewBinding(key.WithKeys("n"), key.WithHelp("n", "new")),
	Refresh: key.NewBinding(key.WithKeys("r"), key.WithHelp("r", "refresh")),

	// ── Detail (shared) ───────────────────────────────────────────────────
	Back:    key.NewBinding(key.WithKeys("esc", "b", "backspace"), key.WithHelp("b", "back")),
	Browser: key.NewBinding(key.WithKeys("o"), key.WithHelp("o", "browser")),
	CopyURL: key.NewBinding(key.WithKeys("u"), key.WithHelp("u", "copy URL")),

	// ── Issue detail ──────────────────────────────────────────────────────
	IssueClose:   key.NewBinding(key.WithKeys("x"), key.WithHelp("x", "close/reopen")),
	IssueAssign:  key.NewBinding(key.WithKeys("a"), key.WithHelp("a", "assign/unassign")),
	IssueLabel:   key.NewBinding(key.WithKeys("l"), key.WithHelp("l", "add label")),
	IssueDevelop: key.NewBinding(key.WithKeys("d"), key.WithHelp("d", "develop")),
	IssuePR:      key.NewBinding(key.WithKeys("p"), key.WithHelp("p", "create PR")),

	// ── PR detail ─────────────────────────────────────────────────────────
	PRCheckout: key.NewBinding(key.WithKeys("c"), key.WithHelp("c", "checkout")),
	PRMerge:    key.NewBinding(key.WithKeys("m"), key.WithHelp("m", "merge")),
	PRClose:    key.NewBinding(key.WithKeys("x"), key.WithHelp("x", "close/reopen")),
}
