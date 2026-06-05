// app.go
package main

import (
	"hubcap/internal/github"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/key"
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

// autoRefreshTickMsg is sent periodically to trigger auto-refresh of data
type autoRefreshTickMsg struct{}

// timerTickMsg is sent every second to update the timer display
type timerTickMsg struct{}

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
	activeTab TabID
	screen    Screen
	width     int
	height    int

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
	configVals      *ConfigVals

	// Configuration form state
	configForm *huh.Form // non-nil while config form is showing

	// confirmingQuit is true while the "Quit Hubcap? [y] / any key to cancel"
	// prompt is showing. Only y/Y proceeds to tea.Quit; anything else dismisses.
	confirmingQuit bool

	// showHelp is true while the ? help overlay is visible.
	showHelp bool

	// Auto-refresh state (global across all tabs)
	lastRefreshTime int64 // Unix timestamp of last refresh
	timerTick       int   // Counter to force view updates every second
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
		configVals:      &ConfigVals{},
	}
}

func (m AppModel) Init() tea.Cmd {
	cmds := []tea.Cmd{
		m.spinner.Tick,
		m.dashboard.spinner.Tick,
		m.dashboard.fetchCmd(),
	}
	// Start auto-refresh ticker if enabled
	if m.cfg.AutoRefreshEnabled {
		m.lastRefreshTime = time.Now().Unix()
		cmds = append(cmds, autoRefreshCmd(m.cfg.AutoRefreshInterval))
		cmds = append(cmds, timerCmd())
	}
	return tea.Batch(cmds...)
}

func inDetailMode(m AppModel) bool {
	return (m.activeTab == TabIssues && m.issues.showDetail) ||
		(m.activeTab == TabPRs && m.prs.showDetail)
}

// hasActiveForm returns true when any sub-model or the app-level filter form
// has an active huh form, or when the list's inline filter input is open.
// Used to suppress global key shortcuts that would conflict with form navigation
// (e.g. tab to switch tabs vs. tab to next field, q to quit vs. q while typing).
func (m AppModel) hasActiveForm() bool {
	return m.filterForm != nil ||
		m.filterLoading ||
		m.configForm != nil ||
		m.issues.activeForm != nil ||
		m.prs.activeForm != nil ||
		(m.activeTab == TabIssues && m.issues.IsFiltering()) ||
		(m.activeTab == TabPRs && m.prs.IsFiltering())
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

// autoRefreshCmd returns a command that sends autoRefreshTickMsg at the configured interval
func autoRefreshCmd(interval int) tea.Cmd {
	if interval <= 0 {
		return nil
	}
	return tea.Tick(time.Duration(interval)*time.Second, func(t time.Time) tea.Msg {
		return autoRefreshTickMsg{}
	})
}

// timerCmd returns a command that sends timerTickMsg every second to update the display
func timerCmd() tea.Cmd {
	return tea.Tick(time.Second, func(t time.Time) tea.Msg {
		return timerTickMsg{}
	})
}

func (m AppModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd

	// Snapshot detail state before processing — used to detect header changes.
	wasInDetail := inDetailMode(m)

	// ── Help overlay — dismiss on any key ────────────────────────────────────
	if m.showHelp {
		if _, ok := msg.(tea.KeyMsg); ok {
			m.showHelp = false
			return m, nil
		}
		if wm, ok := msg.(tea.WindowSizeMsg); ok {
			m.width = wm.Width
			m.height = wm.Height
		}
		return m, nil
	}

	// ── Quit confirmation takes highest priority ──────────────────────────────
	// While the prompt is showing, only y/Y quits; anything else dismisses it.
	// Non-key messages (resize, spinner ticks) still flow through normally.
	if m.confirmingQuit {
		if km, ok := msg.(tea.KeyMsg); ok {
			switch km.String() {
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

	// ── Config form takes priority ────────────────────────────────────────────
	// Route all messages to the active config form exclusively.
	if m.configForm != nil {
		form, cmd := m.configForm.Update(msg)
		if f, ok := form.(*huh.Form); ok {
			m.configForm = f
		}
		switch m.configForm.State {
		case huh.StateCompleted:
			m.configForm = nil
			newCfg := ResolveConfig(m.configVals, m.cfg)
			m.cfg = newCfg
			// Save the config
			if err := saveConfig(newCfg); err != nil {
				// Handle error - for now just continue
			}
			// Update dashboard model with new config
			m.dashboard.cfg = newCfg
			// Start or restart auto-refresh ticker based on new config
			if newCfg.AutoRefreshEnabled {
				m.lastRefreshTime = time.Now().Unix()
				return m, tea.Batch(autoRefreshCmd(newCfg.AutoRefreshInterval), timerCmd())
			}
			// If disabled, reset the refresh time
			m.lastRefreshTime = 0
			return m, nil
		case huh.StateAborted:
			m.configForm = nil
			return m, nil
		}
		return m, cmd
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

	case autoRefreshTickMsg:
		// Update global refresh time
		m.lastRefreshTime = time.Now().Unix()
		// Auto-refresh the current tab's data
		switch m.activeTab {
		case TabIssues:
			if !m.issues.loading && !m.issues.showDetail {
				m.issues.loading = true
				m.issues.loaded = false
				cmds = append(cmds, m.issues.fetchCmd(), m.issues.spinner.Tick)
			}
		case TabPRs:
			if !m.prs.loading && !m.prs.showDetail {
				m.prs.loading = true
				m.prs.loaded = false
				cmds = append(cmds, m.prs.fetchCmd(), m.prs.spinner.Tick)
			}
		case TabDashboard:
			if !m.dashboard.loading {
				m.dashboard.loading = true
				m.dashboard.loaded = false
				cmds = append(cmds, m.dashboard.fetchCmd(), m.dashboard.spinner.Tick)
			}
		}
		// Schedule the next tick
		if m.cfg.AutoRefreshEnabled {
			cmds = append(cmds, autoRefreshCmd(m.cfg.AutoRefreshInterval))
		}
		return m, tea.Batch(cmds...)

	case timerTickMsg:
		// Increment counter to force view update
		m.timerTick++
		// Only continue timer if auto-refresh is enabled
		if m.cfg.AutoRefreshEnabled {
			cmds = append(cmds, timerCmd())
		}
		return m, tea.Batch(cmds...)

	case tea.KeyMsg:
		switch {
		case key.Matches(msg, keys.ForceQuit):
			return m, tea.Quit
		case key.Matches(msg, keys.Help) && !m.hasActiveForm():
			m.showHelp = !m.showHelp
			return m, nil
		case key.Matches(msg, keys.Quit) && !m.hasActiveForm():
			m.confirmingQuit = true
			return m, nil
		case key.Matches(msg, keys.Tab) && !m.hasActiveForm():
			// Don't switch tabs when a sub-model form is open — let the form
			// use tab for field navigation.
			next := nextTab(m.activeTab)
			return m, func() tea.Msg { return switchTabMsg{tab: next} }
		case key.Matches(msg, keys.ShiftTab) && !m.hasActiveForm():
			prev := (m.activeTab + 2) % 3
			return m, func() tea.Msg { return switchTabMsg{tab: TabID(prev)} }
		case key.Matches(msg, keys.Config) && !m.hasActiveForm():
			// Open configuration form
			InitConfigVals(m.configVals, m.cfg)
			m.configForm = BuildConfigForm(m.configVals).WithWidth(m.width - 8)
			return m, m.configForm.Init()
		case key.Matches(msg, keys.Filters) && !m.hasActiveForm():
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

	case TabPRs:
		updated, cmd := m.prs.Update(msg)
		m.prs = updated
		if cmd != nil {
			cmds = append(cmds, cmd)
		}

	case TabDashboard:
		updated, cmd := m.dashboard.Update(msg)
		m.dashboard = updated
		if cmd != nil {
			cmds = append(cmds, cmd)
		}
	}

	// When inDetail transitions (list↔detail), the header height changes by 3 lines.
	// Force a full clear+repaint to prevent BubbleTea's diff renderer from misplacing content.
	if wasInDetail != inDetailMode(m) {
		cmds = append(cmds, tea.ClearScreen)
	}

	return m, tea.Batch(cmds...)
}

// ── Footer button helpers ─────────────────────────────────────────────────────
//
// Every footer variant produces a 3-row block (matching the height of a
// bordered button) so the layout never shifts when switching between hint,
// toast, and confirmation modes.
//
// Color palette:
//   green  (83)  — primary action keys
//   amber (208)  — structural / meta keys  (tab, ?, ,)
//   red   (196)  — destructive keys (quit, close)

const btnBg = lipgloss.Color("235")

// keyBtn renders key text inside a rounded lipgloss border.
// The result is always 3 terminal rows: top border, content row, bottom border.
func keyBtn(text string, fg lipgloss.Color) string {
	return lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(fg).
		BorderBackground(btnBg).
		Foreground(fg).
		Background(btnBg).
		Padding(0, 1).
		Bold(true).
		Render(text)
}

// keyHint pairs a 3-row button with a description label, centering the label
// on the middle row via JoinHorizontal.
func keyHint(keyText, desc string, fg lipgloss.Color) string {
	btn := keyBtn(keyText, fg)
	lbl := lipgloss.NewStyle().
		Foreground(lipgloss.Color("244")).
		Background(btnBg).
		Render(" " + desc)
	return lipgloss.JoinHorizontal(lipgloss.Center, btn, lbl)
}

// buildButtonFooter joins hint strings horizontally with equal gaps, pads every
// line of the result to width, and returns a 3-row footer string.
func buildButtonFooter(hints []string, width int) string {
	bgSt := lipgloss.NewStyle().Background(btnBg)
	edge := bgSt.Render("  ")
	gap := bgSt.Render("   ")

	parts := make([]string, 0, len(hints)*2+2)
	parts = append(parts, edge)
	for i, h := range hints {
		if i > 0 {
			parts = append(parts, gap)
		}
		parts = append(parts, h)
	}
	parts = append(parts, edge)

	joined := lipgloss.JoinHorizontal(lipgloss.Center, parts...)

	// Pad every line to the full footer width.
	lines := strings.Split(joined, "\n")
	for i, l := range lines {
		w := lipgloss.Width(l)
		if w < width {
			lines[i] = l + bgSt.Render(strings.Repeat(" ", width-w))
		} else if w > width {
			lines[i] = lipgloss.NewStyle().MaxWidth(width).Render(l)
		}
	}
	return strings.Join(lines, "\n")
}

// centerInFooter wraps a single-line string in a 3-row block so toasts and
// confirmations occupy the same height as the button footer.
func centerInFooter(content string, width int) string {
	bgSt := lipgloss.NewStyle().Background(btnBg)
	blank := bgSt.Width(width).Render("")
	w := lipgloss.Width(content)
	if w < width {
		content += bgSt.Render(strings.Repeat(" ", width-w))
	} else if w > width {
		content = lipgloss.NewStyle().MaxWidth(width).Render(content)
	}
	return blank + "\n" + content + "\n" + blank
}

// quitConfirmFooter renders the quit confirmation prompt.
// y/Y confirms; any other key cancels.
func quitConfirmFooter(width int) string {
	hints := []string{
		keyHint("y", "confirm quit", lipgloss.Color("196")),
		keyHint("any key", "cancel", lipgloss.Color("208")),
	}
	return buildButtonFooter(hints, width)
}

// footerPendingToast renders a 3-row "working…" indicator with a spinner.
func footerPendingToast(spinnerView string, msg string, width int) string {
	bgSt := lipgloss.NewStyle().Background(btnBg)
	spinSt := lipgloss.NewStyle().Background(btnBg).Foreground(lipgloss.Color("83"))
	txtSt := lipgloss.NewStyle().Background(btnBg).Foreground(lipgloss.Color("244"))
	line := bgSt.Render("  ") + spinSt.Render(spinnerView) + " " + txtSt.Render(msg)
	return centerInFooter(line, width)
}

// footerToast renders a 3-row success / error toast notification.
func footerToast(msg string, isErr bool, width int) string {
	fg := lipgloss.Color("83") // green
	icon := "✓"
	if isErr {
		fg = lipgloss.Color("196") // red
		icon = "✗"
	}
	bgSt := lipgloss.NewStyle().Background(btnBg)
	txtSt := lipgloss.NewStyle().Background(btnBg).Foreground(fg).Bold(true)
	line := bgSt.Render("  ") + txtSt.Render(icon+" "+msg)
	return centerInFooter(line, width)
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

	green := lipgloss.Color("83")
	amber := lipgloss.Color("208")
	red := lipgloss.Color("196")

	var hints []string
	add := func(b key.Binding, desc string, fg lipgloss.Color) {
		hints = append(hints, keyHint(b.Help().Key, desc, fg))
	}

	switch {
	case m.activeTab == TabIssues && m.issues.showDetail:
		closeLabel := "close"
		if strings.EqualFold(m.issues.detailIssue.State, "closed") {
			closeLabel = "reopen"
		}
		assignLabel := "assign"
		if len(m.issues.detailIssue.Assignees) > 0 {
			assignLabel = "unassign"
		}
		add(keys.IssueDevelop, "develop", green)
		add(keys.IssuePR, "PR", green)
		add(keys.IssueClose, closeLabel, red)
		add(keys.IssueAssign, assignLabel, green)
		add(keys.IssueLabel, "label", green)
		add(keys.Browser, "browser", green)
		add(keys.CopyURL, "copy URL", green)
		add(keys.Refresh, "refresh", green)
		add(keys.Back, "back", amber)
	case m.activeTab == TabPRs && m.prs.showDetail:
		closeLabel := "close"
		if strings.EqualFold(m.prs.detailPR.State, "closed") {
			closeLabel = "reopen"
		}
		add(keys.PRCheckout, "checkout", green)
		add(keys.PRMerge, "merge", green)
		add(keys.PRClose, closeLabel, red)
		add(keys.Browser, "browser", green)
		add(keys.CopyURL, "copy URL", green)
		add(keys.Refresh, "refresh", green)
		add(keys.Back, "back", amber)
	}

	return buildButtonFooter(hints, width)
}

func footerView(activeTab TabID, width int) string {
	green := lipgloss.Color("83")
	amber := lipgloss.Color("208")
	red := lipgloss.Color("196")

	add := func(hints *[]string, b key.Binding, desc string, fg lipgloss.Color) {
		*hints = append(*hints, keyHint(b.Help().Key, desc, fg))
	}

	var hints []string
	add(&hints, keys.Up, "navigate", green)
	add(&hints, keys.Open, "open", green)
	add(&hints, keys.Tab, "switch view", amber)
	switch activeTab {
	case TabIssues:
		add(&hints, keys.New, "new issue", green)
		add(&hints, keys.Filters, "filters", amber)
	case TabPRs:
		add(&hints, keys.New, "new PR", green)
		add(&hints, keys.Filters, "filters", amber)
	}
	add(&hints, keys.Config, "config", amber)
	add(&hints, keys.Refresh, "refresh", green)
	add(&hints, keys.Help, "help", amber)
	add(&hints, keys.Quit, "quit", red)

	return buildButtonFooter(hints, width)
}

// helpOverlayView renders a context-sensitive shortcut reference.
// It uses the same border style as other overlays and derives key names
// directly from the central keys registry so descriptions never drift.
func helpOverlayView(m AppModel, innerW int) string {
	titleStyle := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("208"))
	sectionStyle := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("86"))
	keyStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("208")).Bold(true)
	descStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("244"))
	dimStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("238"))

	row := func(b key.Binding) string {
		h := b.Help()
		return keyStyle.Render("["+h.Key+"]") + " " + descStyle.Render(h.Desc)
	}

	col := func(left, right string) string {
		pad := innerW/2 - 4
		if pad < 20 {
			pad = 20
		}
		leftW := lipgloss.Width(left)
		spaces := pad - leftW
		if spaces < 1 {
			spaces = 1
		}
		return left + strings.Repeat(" ", spaces) + right
	}

	var b strings.Builder
	b.WriteString("\n  " + titleStyle.Render("KEYBOARD SHORTCUTS") + "\n")
	b.WriteString("  " + dimStyle.Render(strings.Repeat("─", innerW-6)) + "\n\n")

	// ── Navigation (always shown) ─────────────────────────────────────────
	b.WriteString("  " + sectionStyle.Render("Navigation") + "\n")
	b.WriteString("  " + col(row(keys.Up), row(keys.Down)) + "\n")
	b.WriteString("  " + col(row(keys.Top), row(keys.Bottom)) + "\n")
	b.WriteString("  " + col(row(keys.Open), row(keys.Refresh)) + "\n")
	b.WriteString("  " + col(row(keys.Tab), row(keys.ShiftTab)) + "\n")
	b.WriteString("  " + col(row(keys.Config), row(keys.Quit)) + "\n")

	inDetail := inDetailMode(m)

	if !inDetail {
		// List-level extras
		b.WriteString("\n  " + sectionStyle.Render("List") + "\n")
		b.WriteString("  " + col(row(keys.New), row(keys.Filters)) + "\n")
	}

	if m.activeTab == TabIssues && inDetail {
		b.WriteString("\n  " + sectionStyle.Render("Issue detail") + "\n")
		b.WriteString("  " + col(row(keys.IssueDevelop), row(keys.IssuePR)) + "\n")
		b.WriteString("  " + col(row(keys.IssueClose), row(keys.IssueAssign)) + "\n")
		b.WriteString("  " + col(row(keys.IssueLabel), row(keys.Browser)) + "\n")
		b.WriteString("  " + col(row(keys.CopyURL), row(keys.Back)) + "\n")
	}

	if m.activeTab == TabPRs && inDetail {
		b.WriteString("\n  " + sectionStyle.Render("PR detail") + "\n")
		b.WriteString("  " + col(row(keys.PRCheckout), row(keys.PRMerge)) + "\n")
		b.WriteString("  " + col(row(keys.PRClose), row(keys.Browser)) + "\n")
		b.WriteString("  " + col(row(keys.CopyURL), row(keys.Back)) + "\n")
	}

	b.WriteString("\n  " + dimStyle.Render("Press any key to close") + "\n")

	appBorder := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("208")).
		Width(innerW)
	return appBorder.Render(b.String())
}

func (m AppModel) View() string {
	if m.width == 0 || m.height == 0 {
		return ""
	}

	innerW := m.width - 2
	innerH := m.height - 2

	// ── Help overlay ──────────────────────────────────────────────────────────
	if m.showHelp {
		return helpOverlayView(m, innerW)
	}

	// ── Config form overlay ───────────────────────────────────────────────────
	// Show the configuration form when active.
	if m.configForm != nil {
		body := m.configForm.View()

		appBorder := lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("208")).
			Width(innerW)
		return appBorder.Render(body)
	}

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
	header := headerView(m.activeTab, m.repo, m.issues.filters, m.prs.filters, m.dashboard.Counts(), innerW, inDetail, m.cfg.AutoRefreshEnabled, m.cfg.AutoRefreshInterval, m.lastRefreshTime, time.Now().Unix())

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
