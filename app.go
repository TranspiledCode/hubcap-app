// app.go
package main

import (
	"hubcap/internal/github"
	"strings"

	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/huh"
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

// filterDataFetchedMsg carries the result of fetching assignees + labels
// needed to build a filter form without leaving the TUI.
type filterDataFetchedMsg struct {
	forPRs    bool
	assignees []string
	labels    []string
}

// dashboardData holds the three personal sections of the My Work dashboard.
type dashboardData struct {
	reviewRequests []github.PullRequest
	myPRs          []github.PullRequest
	assignedIssues []github.Issue
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

	// ── Filter form state ─────────────────────────────────────────────────────
	// When the user presses "f", we fetch assignees + labels async, then show
	// an embedded huh form. No terminal suspension required.

	filterLoading bool      // true while fetching assignees + labels
	filterForPRs  bool      // which tab's filters are being configured
	filterForm    *huh.Form // non-nil while the form is showing
	filterLabels  []string  // stored after fetch for use in ResolveFilters

	// Heap-allocated value containers — their addresses are stable across
	// BubbleTea model copies so huh's Value() pointers remain valid.
	issueFilterVals *IssueFilterVals
	prFilterVals    *PRFilterVals

	// confirmingQuit is true while the "Quit Hubcap? [y] / any key to cancel"
	// prompt is showing. Only y/Y proceeds to tea.Quit; anything else dismisses.
	confirmingQuit bool
}

func newAppModel(repo string, cfg Config, issueFilters github.Filters, prFilters github.PRFilters) AppModel {
	s := spinner.New()
	s.Spinner = spinner.Dot
	s.Style = lipgloss.NewStyle().Foreground(lipgloss.Color("86"))

	return AppModel{
		activeTab:       TabDashboard,
		screen:          ScreenList,
		repo:            repo,
		cfg:             cfg,
		spinner:         s,
		issues:          newIssuesModel(issueFilters),
		prs:             newPRsModel(prFilters),
		dashboard:       newDashboardModel(cfg),
		issueFilterVals: &IssueFilterVals{},
		prFilterVals:    &PRFilterVals{},
	}
}

func (m AppModel) Init() tea.Cmd {
	return tea.Batch(
		m.spinner.Tick,
		m.dashboard.spinner.Tick,
		m.dashboard.fetchCmd(),
	)
}

func inDetailMode(m AppModel) bool {
	return (m.activeTab == TabIssues && m.issues.showDetail) ||
		(m.activeTab == TabPRs && m.prs.showDetail)
}

// hasActiveForm returns true when any sub-model or the app-level filter form
// has an active huh form. Used to suppress global key shortcuts that would
// conflict with form navigation (tab to switch tabs vs. tab to next field).
func (m AppModel) hasActiveForm() bool {
	return m.filterForm != nil ||
		m.filterLoading ||
		m.issues.activeForm != nil ||
		m.prs.activeForm != nil
}

// fetchFilterDataCmd fetches assignees and labels in the background so the
// filter form can be shown inline without suspending the TUI.
func fetchFilterDataCmd(forPRs bool) tea.Cmd {
	return func() tea.Msg {
		// Errors are non-fatal; the form falls back to plain text inputs.
		assignees, _ := github.FetchAssignees()
		labels, _ := github.FetchLabels()
		return filterDataFetchedMsg{
			forPRs:    forPRs,
			assignees: assignees,
			labels:    labels,
		}
	}
}

func (m AppModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd

	// Snapshot detail state before processing — used to detect header changes.
	wasInDetail := inDetailMode(m)

	// ── Quit confirmation takes highest priority ──────────────────────────────
	// While the prompt is showing, only y/Y quits; anything else dismisses it.
	// Non-key messages (resize, spinner ticks) still flow through normally.
	if m.confirmingQuit {
		if key, ok := msg.(tea.KeyMsg); ok {
			switch key.String() {
			case "y", "Y":
				return m, tea.Quit
			default:
				m.confirmingQuit = false
			}
			return m, nil
		}
		// Allow window resize through so the prompt stays correctly sized.
		if wm, ok := msg.(tea.WindowSizeMsg); ok {
			m.width = wm.Width
			m.height = wm.Height
		}
		return m, nil
	}

	// ── Filter form takes priority ────────────────────────────────────────────
	// Route all messages to the active filter form exclusively.
	if m.filterForm != nil {
		form, cmd := m.filterForm.Update(msg)
		if f, ok := form.(*huh.Form); ok {
			m.filterForm = f
		}
		switch m.filterForm.State {
		case huh.StateCompleted:
			m.filterForm = nil
			if m.filterForPRs {
				newFilters := ResolvePRFilters(m.prFilterVals, m.prs.filters, m.filterLabels)
				return m, func() tea.Msg { return prFiltersUpdatedMsg{filters: newFilters} }
			}
			newFilters := ResolveIssueFilters(m.issueFilterVals, m.issues.filters, m.filterLabels)
			return m, func() tea.Msg { return issueFiltersUpdatedMsg{filters: newFilters} }
		case huh.StateAborted:
			m.filterForm = nil
			return m, nil
		}
		return m, cmd
	}

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

	case filterDataFetchedMsg:
		m.filterLoading = false
		m.filterLabels = msg.labels
		if msg.forPRs {
			InitPRFilterVals(m.prFilterVals, m.prs.filters, msg.assignees)
			m.filterForm = BuildPRFilterForm(m.prFilterVals, msg.assignees, msg.labels).
				WithWidth(m.width - 8)
		} else {
			InitIssueFilterVals(m.issueFilterVals, m.issues.filters, msg.assignees)
			m.filterForm = BuildIssueFilterForm(m.issueFilterVals, msg.assignees, msg.labels).
				WithWidth(m.width - 8)
		}
		return m, m.filterForm.Init()

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
			// Don't switch tabs when a sub-model form is open — let the form
			// use tab for field navigation.
			if !m.hasActiveForm() {
				next := nextTab(m.activeTab)
				return m, func() tea.Msg { return switchTabMsg{tab: next} }
			}
		case "shift+tab":
			if !m.hasActiveForm() {
				prev := (m.activeTab + 2) % 3
				return m, func() tea.Msg { return switchTabMsg{tab: prev} }
			}
		case "f":
			if m.hasActiveForm() {
				break
			}
			if m.activeTab == TabIssues && !m.issues.loading && !m.issues.showDetail {
				m.filterLoading = true
				m.filterForPRs = false
				return m, tea.Batch(fetchFilterDataCmd(false), m.spinner.Tick)
			}
			if m.activeTab == TabPRs && !m.prs.loading && !m.prs.showDetail {
				m.filterLoading = true
				m.filterForPRs = true
				return m, tea.Batch(fetchFilterDataCmd(true), m.spinner.Tick)
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
		// Only keep the app-level spinner alive while fetching filter data.
		if m.filterLoading {
			var cmd tea.Cmd
			m.spinner, cmd = m.spinner.Update(msg)
			cmds = append(cmds, cmd)
		}
	}

	// Delegate to active sub-model
	switch m.activeTab {
	case TabIssues:
		updated, cmd := m.issues.Update(msg)
		m.issues = updated
		if cmd != nil {
			cmds = append(cmds, cmd)
		}
		if m.issues.action == "switch" {
			m.issues.action = ""
			next := nextTab(m.activeTab)
			cmds = append(cmds, func() tea.Msg { return switchTabMsg{tab: next} })
		}
		if m.issues.action == "quit" {
			m.issues.action = ""
			m.confirmingQuit = true
			return m, nil
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
			m.prs.action = ""
			m.confirmingQuit = true
			return m, nil
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
			m.dashboard.action = ""
			m.confirmingQuit = true
			return m, nil
		}
	}

	// When inDetail transitions (list↔detail), the header height changes by 3 lines.
	// Force a full clear+repaint to prevent BubbleTea's diff renderer from misplacing content.
	if wasInDetail != inDetailMode(m) {
		cmds = append(cmds, tea.ClearScreen)
	}

	return m, tea.Batch(cmds...)
}

// quitConfirmFooter renders the quit confirmation prompt in place of the
// normal footer. y/Y confirms, any other key cancels.
func quitConfirmFooter(width int) string {
	footerBg := lipgloss.NewStyle().Background(lipgloss.Color("235"))
	promptSt := lipgloss.NewStyle().Background(lipgloss.Color("235")).Foreground(lipgloss.Color("252"))
	keySt := lipgloss.NewStyle().Background(lipgloss.Color("235")).Foreground(lipgloss.Color("208")).Bold(true)
	yesSt := lipgloss.NewStyle().Background(lipgloss.Color("235")).Foreground(lipgloss.Color("196")).Bold(true)
	cancelSt := lipgloss.NewStyle().Background(lipgloss.Color("235")).Foreground(lipgloss.Color("244"))

	line := footerBg.Render("  ") +
		promptSt.Render("Quit Hubcap?  ") +
		yesSt.Render("[y]") + promptSt.Render(" quit  ") +
		keySt.Render("[any key]") + cancelSt.Render(" cancel")

	lineW := lipgloss.Width(line)
	if lineW < width {
		line += footerBg.Render(strings.Repeat(" ", width-lineW))
	}
	return line
}

// footerPendingToast renders a single-line "in progress" indicator in the
// footer bar using the model's spinner for animation.
func footerPendingToast(spinnerView string, msg string, width int) string {
	bg := lipgloss.NewStyle().Background(lipgloss.Color("235"))
	spin := lipgloss.NewStyle().
		Background(lipgloss.Color("235")).
		Foreground(lipgloss.Color("86")) // green spinner
	txt := lipgloss.NewStyle().
		Background(lipgloss.Color("235")).
		Foreground(lipgloss.Color("244")) // muted text while working
	line := bg.Render("  ") + spin.Render(spinnerView) + " " + txt.Render(msg)
	w := lipgloss.Width(line)
	if w < width {
		line += bg.Render(strings.Repeat(" ", width-w))
	}
	return line
}

// footerToast renders a single-line toast notification styled like the footer
// bar. isErr controls red vs green colouring. The line is padded to width so
// it fills the full footer row without changing the overall layout height.
func footerToast(msg string, isErr bool, width int) string {
	footerBg := lipgloss.Color("235")
	fg := lipgloss.Color("86") // green
	icon := "✓"
	if isErr {
		fg = lipgloss.Color("196") // red
		icon = "✗"
	}
	bg := lipgloss.NewStyle().Background(footerBg)
	txt := lipgloss.NewStyle().Background(footerBg).Foreground(fg).Bold(true)
	line := bg.Render("  ") + txt.Render(icon+" "+msg)
	w := lipgloss.Width(line)
	if w < width {
		line += bg.Render(strings.Repeat(" ", width-w))
	}
	return line
}

// detailActionFooter renders the context-specific footer shown in detail views.
// When there is an active toast (action message / error), it replaces the key
// hints for the duration of the auto-dismiss timer — keeping the layout height
// exactly constant.
func detailActionFooter(m AppModel, width int) string {
	// Toast / pending indicator — all shown in place of key hints.
	// Priority: error > success > pending (working…)
	if m.activeTab == TabIssues && m.issues.showDetail {
		if m.issues.actionErr != nil {
			return footerToast(m.issues.actionErr.Error(), true, width)
		}
		if m.issues.actionMsg != "" {
			return footerToast(m.issues.actionMsg, false, width)
		}
		if m.issues.actionPending != "" {
			return footerPendingToast(m.issues.spinner.View(), m.issues.actionPending, width)
		}
	}
	if m.activeTab == TabPRs && m.prs.showDetail {
		if m.prs.actionErr != nil {
			return footerToast(m.prs.actionErr.Error(), true, width)
		}
		if m.prs.actionMsg != "" {
			return footerToast(m.prs.actionMsg, false, width)
		}
		if m.prs.actionPending != "" {
			return footerPendingToast(m.prs.spinner.View(), m.prs.actionPending, width)
		}
	}

	footerBg := lipgloss.NewStyle().Background(lipgloss.Color("235"))
	keyStyle := lipgloss.NewStyle().Background(lipgloss.Color("235")).Foreground(lipgloss.Color("208")).Bold(true)
	descStyle := lipgloss.NewStyle().Background(lipgloss.Color("235")).Foreground(lipgloss.Color("244"))
	sepStyle := lipgloss.NewStyle().Background(lipgloss.Color("235")).Foreground(lipgloss.Color("238"))
	sep := sepStyle.Render(" · ")

	var hints []string
	addHint := func(key, desc string) {
		hints = append(hints, keyStyle.Render("["+key+"]")+" "+descStyle.Render(desc))
	}

	switch {
	case m.activeTab == TabIssues && m.issues.showDetail:
		closeLabel := "close"
		if strings.EqualFold(m.issues.detailIssue.State, "closed") {
			closeLabel = "reopen"
		}
		addHint("d", "develop")
		addHint("p", "PR")
		addHint("c", closeLabel)
		addHint("a", "assign")
		addHint("l", "label")
		addHint("o", "browser")
		addHint("u", "copy URL")
		addHint("r", "refresh")
		addHint("b", "back")
	case m.activeTab == TabPRs && m.prs.showDetail:
		closeLabel := "close"
		if strings.EqualFold(m.prs.detailPR.State, "closed") {
			closeLabel = "reopen"
		}
		addHint("c", "checkout")
		addHint("m", "merge")
		addHint("x", closeLabel)
		addHint("o", "browser")
		addHint("u", "copy URL")
		addHint("r", "refresh")
		addHint("b", "back")
	}

	line := footerBg.Render("  ") + strings.Join(hints, sep)
	lineW := lipgloss.Width(line)
	if lineW < width {
		line += footerBg.Render(strings.Repeat(" ", width-lineW))
	} else if lineW > width {
		line = lipgloss.NewStyle().MaxWidth(width).Render(line)
	}
	return line
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

	// ── Filter loading / form overlay ─────────────────────────────────────────
	// Show spinner while fetching filter data, then the embedded form.
	// Both replace the normal body content — no terminal suspension.
	if m.filterLoading || m.filterForm != nil {
		var body string
		if m.filterLoading {
			body = "\n  " + m.spinner.View() + " Loading filter options…\n"
		} else {
			body = m.filterForm.View()
		}

		appBorder := lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("208")).
			Width(innerW)
		return appBorder.Render(body)
	}

	inDetail := inDetailMode(m)
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

	var footer string
	switch {
	case m.confirmingQuit:
		footer = quitConfirmFooter(innerW)
	case inDetail:
		footer = detailActionFooter(m, innerW)
	default:
		footer = footerView(m.activeTab, innerW)
	}

	headerLines := strings.Count(header, "\n")
	bodyLines := strings.Count(body, "\n")
	footerLines := strings.Count(footer, "\n") + 1
	usedLines := headerLines + bodyLines + footerLines

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
