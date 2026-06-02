// app.go
package main

import (
	"hubcap/internal/github"
	"strings"

	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// ── Message types ─────────────────────────────────────────────────────────────

// Navigation messages
type switchTabMsg struct{ tab TabID }
type quitMsg struct{}

// openItemMsg switches to the Issues or PRs tab and opens a specific item's detail.
type openItemMsg struct {
	isIssue bool
	number  int
}

// Data fetch results
type issuesFetchedMsg struct {
	issues []github.Issue
	err    error
}

type prsFetchedMsg struct {
	prs []github.PullRequest
	err error
}

type issueFetchedMsg struct {
	issue github.Issue
	err   error
}

type prFetchedMsg struct {
	pr  github.PullRequest
	err error
}

type dashboardFetchedMsg struct {
	data dashboardData
	err  error
}

type issueFiltersUpdatedMsg struct{ filters github.Filters }
type prFiltersUpdatedMsg struct{ filters github.PRFilters }

// dashboardData holds all sections of the dashboard
type dashboardData struct {
	reviewRequests []github.PullRequest
	myPRs          []github.PullRequest
	assignedIssues []github.Issue
	availableIssues []github.Issue
}

// ── Root model ────────────────────────────────────────────────────────────────

// Screen identifies which view is currently active
type Screen int

const (
	ScreenList   Screen = iota // issues or PRs list
	ScreenDetail               // issue or PR detail
)

// AppModel is the root bubbletea model
type AppModel struct {
	// Core state
	activeTab  TabID
	screen     Screen
	width      int
	height     int

	// Config & repo
	repo string
	cfg  Config

	// Sub-models (only one active at a time based on activeTab + screen)
	issues    IssuesModel
	prs       PRsModel
	dashboard DashboardModel

	// Shared spinner for top-level loading
	spinner spinner.Model
	loading bool
}

func newAppModel(repo string, cfg Config, issueFilters github.Filters, prFilters github.PRFilters) AppModel {
	s := spinner.New()
	s.Spinner = spinner.Dot
	s.Style = lipgloss.NewStyle().Foreground(lipgloss.Color("86"))

	return AppModel{
		activeTab: TabDashboard,
		screen:    ScreenList,
		repo:      repo,
		cfg:       cfg,
		spinner:   s,
		issues:    newIssuesModel(issueFilters),
		prs:       newPRsModel(prFilters),
		dashboard: newDashboardModel(cfg),
	}
}

func (m AppModel) Init() tea.Cmd {
	return tea.Batch(
		m.spinner.Tick,
		m.dashboard.spinner.Tick,
		m.dashboard.fetchCmd(),
	)
}

func (m AppModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		// Subtract border (2 cols wide, 2 rows tall)
		innerW := msg.Width - 2
		innerH := msg.Height - 2
		m.issues.width = innerW
		m.issues.height = innerH
		m.prs.width = innerW
		m.prs.height = innerH
		m.dashboard.width = innerW
		m.dashboard.height = innerH

	case issueFiltersUpdatedMsg:
		m.issues.filters = msg.filters
		m.issues.loading = true
		m.issues.loaded = false
		return m, tea.Batch(m.issues.fetchCmd(), m.issues.spinner.Tick)

	case prFiltersUpdatedMsg:
		m.prs.filters = msg.filters
		m.prs.loading = true
		m.prs.loaded = false
		return m, tea.Batch(m.prs.fetchCmd(), m.prs.spinner.Tick)

	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c":
			return m, tea.Quit
		case "tab":
			next := nextTab(m.activeTab)
			return m, func() tea.Msg { return switchTabMsg{tab: next} }
		case "shift+tab":
			prev := (m.activeTab + 2) % 3
			return m, func() tea.Msg { return switchTabMsg{tab: prev} }
		case "f":
			if m.activeTab == TabIssues && !m.issues.loading && !m.issues.showDetail {
				currentFilters := m.issues.filters
				state := &AppState{IssueFilters: currentFilters}
				return m, tea.Exec(newFilterCmd(func() error {
					state.IssueFilters = configureFilters(state)
					return nil
				}), func(err error) tea.Msg {
					return issueFiltersUpdatedMsg{filters: state.IssueFilters}
				})
			}
			if m.activeTab == TabPRs && !m.prs.loading && !m.prs.showDetail {
				currentFilters := m.prs.filters
				state := &AppState{PRFilters: currentFilters}
				return m, tea.Exec(newFilterCmd(func() error {
					state.PRFilters = configurePRFilters(state)
					return nil
				}), func(err error) tea.Msg {
					return prFiltersUpdatedMsg{filters: state.PRFilters}
				})
			}
		}
		_ = msg

	case switchTabMsg:
		m.activeTab = msg.tab
		switch msg.tab {
		case TabIssues:
			if !m.issues.loaded {
				m.issues.loading = true
				cmds = append(cmds, m.issues.fetchCmd(), m.issues.spinner.Tick)
			}
		case TabPRs:
			if !m.prs.loaded {
				m.prs.loading = true
				cmds = append(cmds, m.prs.fetchCmd(), m.prs.spinner.Tick)
			}
		case TabDashboard:
			if !m.dashboard.loaded {
				m.dashboard.loading = true
				cmds = append(cmds, m.dashboard.fetchCmd(), m.dashboard.spinner.Tick)
			}
		}

	case openItemMsg:
		if msg.isIssue {
			m.activeTab = TabIssues
			m.issues.loadingDetail = true
			if !m.issues.loaded {
				m.issues.loading = true
				cmds = append(cmds, m.issues.fetchCmd(), m.issues.spinner.Tick)
			}
			cmds = append(cmds, fetchIssueDetailCmd(msg.number), m.issues.spinner.Tick)
		} else {
			m.activeTab = TabPRs
			m.prs.loadingDetail = true
			if !m.prs.loaded {
				m.prs.loading = true
				cmds = append(cmds, m.prs.fetchCmd(), m.prs.spinner.Tick)
			}
			cmds = append(cmds, fetchPRDetailCmd(msg.number), m.prs.spinner.Tick)
		}

	case spinner.TickMsg:
		var cmd tea.Cmd
		m.spinner, cmd = m.spinner.Update(msg)
		cmds = append(cmds, cmd)
	}

	// Delegate to active sub-model
	switch m.activeTab {
	case TabIssues:
		updated, cmd := m.issues.Update(msg)
		m.issues = updated
		if cmd != nil {
			cmds = append(cmds, cmd)
		}
		// Handle navigation out of issues
		if m.issues.action == "switch" {
			m.issues.action = ""
			next := nextTab(m.activeTab)
			cmds = append(cmds, func() tea.Msg { return switchTabMsg{tab: next} })
		}
		if m.issues.action == "quit" {
			return m, tea.Quit
		}

	case TabPRs:
		updated, cmd := m.prs.Update(msg)
		m.prs = updated
		if cmd != nil {
			cmds = append(cmds, cmd)
		}
		if m.prs.action == "switch" {
			m.prs.action = ""
			next := nextTab(m.activeTab)
			cmds = append(cmds, func() tea.Msg { return switchTabMsg{tab: next} })
		}
		if m.prs.action == "quit" {
			return m, tea.Quit
		}

	case TabDashboard:
		updated, cmd := m.dashboard.Update(msg)
		m.dashboard = updated
		if cmd != nil {
			cmds = append(cmds, cmd)
		}
		if m.dashboard.action == "switch" {
			m.dashboard.action = ""
			next := nextTab(m.activeTab)
			cmds = append(cmds, func() tea.Msg { return switchTabMsg{tab: next} })
		}
		if m.dashboard.action == "quit" {
			return m, tea.Quit
		}
	}

	return m, tea.Batch(cmds...)
}

func footerView(activeTab TabID, width int) string {
	footerBg := lipgloss.NewStyle().Background(lipgloss.Color("235"))
	keyStyle := lipgloss.NewStyle().Background(lipgloss.Color("235")).Foreground(lipgloss.Color("208")).Bold(true)
	descStyle := lipgloss.NewStyle().Background(lipgloss.Color("235")).Foreground(lipgloss.Color("244"))
	sepStyle := lipgloss.NewStyle().Background(lipgloss.Color("235")).Foreground(lipgloss.Color("238"))
	sep := sepStyle.Render(" · ")

	var hints []string
	addHint := func(key, desc string) {
		hints = append(hints, keyStyle.Render("["+key+"]")+" "+descStyle.Render(desc))
	}

	addHint("↑↓", "move")
	addHint("enter", "open")
	addHint("tab", "switch")
	switch activeTab {
	case TabIssues:
		addHint("n", "new issue")
		addHint("f", "filters")
	case TabPRs:
		addHint("n", "new PR")
		addHint("f", "filters")
	}
	addHint("r", "refresh")
	addHint("q", "quit")

	line := footerBg.Render("  ") + strings.Join(hints, sep)
	lineW := lipgloss.Width(line)
	if lineW < width {
		line += footerBg.Render(strings.Repeat(" ", width-lineW))
	} else if lineW > width {
		line = lipgloss.NewStyle().MaxWidth(width).Render(line)
	}

	return line
}

func (m AppModel) View() string {
	if m.width == 0 || m.height == 0 {
		return ""
	}

	innerW := m.width - 2
	innerH := m.height - 2

	inDetail := (m.activeTab == TabIssues && m.issues.showDetail) ||
		(m.activeTab == TabPRs && m.prs.showDetail)
	header := headerView(m.activeTab, m.repo, m.issues.filters, m.prs.filters, m.dashboard.Counts(), innerW, inDetail)

	var body string
	switch m.activeTab {
	case TabIssues:
		body = m.issues.View()
	case TabPRs:
		body = m.prs.View()
	case TabDashboard:
		body = m.dashboard.View()
	}

	// Build the footer hint bar
	footer := footerView(m.activeTab, innerW)

	// Count used lines: header + body lines + footer
	headerLines := strings.Count(header, "\n")
	bodyLines := strings.Count(body, "\n")
	footerLines := strings.Count(footer, "\n") + 1
	usedLines := headerLines + bodyLines + footerLines

	// Fill remaining space so footer sticks to the bottom
	remaining := innerH - usedLines
	if remaining < 0 {
		remaining = 0
	}
	fill := strings.Repeat("\n", remaining)

	inner := header + body + fill + footer

	appBorder := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("208")).
		Width(innerW)

	return appBorder.Render(inner)
}
