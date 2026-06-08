// app.go
package main

import (
	"hubcap/internal/github"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/viewport"
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

// issuePrefetchedMsg carries a prefetched issue detail fetched in the
// background while the user navigates the list. The gen field is compared
// against IssuesModel.prefetchGen to discard stale results.
type issuePrefetchedMsg struct {
	number int
	issue  github.Issue
	gen    int
}

// prPrefetchedMsg is the PR equivalent of issuePrefetchedMsg.
type prPrefetchedMsg struct {
	number int
	pr     github.PullRequest
	gen    int
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
	helpVP   viewport.Model

	// Auto-refresh state (global across all tabs)
	lastRefreshTime int64 // Unix timestamp of last refresh
	timerTick       int   // Counter to force view updates every second

	// currentUser is the GitHub login of the authenticated user, fetched once
	// at startup. Used to distinguish Grab / Take / Drop on the issues list.
	currentUser string
}

func newAppModel(repo string, cfg Config, issueFilters github.Filters, prFilters github.PRFilters, cache AppCache) AppModel {
	s := spinner.New()
	s.Spinner = spinner.Dot
	s.Style = lipgloss.NewStyle().Foreground(lipgloss.Color("86"))

	theme := resolveTheme(cfg.UITheme)
	issues := newIssuesModel(issueFilters)
	issues.uiTheme = theme

	// Seed issues list from cache for instant startup display.
	if cachedIssues, ok := cache.GetIssues(); ok {
		items := make([]list.Item, len(cachedIssues))
		for i, issue := range cachedIssues {
			items[i] = issueListItem{issue: issue}
		}
		issues.list.SetItems(items)
		issues.loaded = true
		issues.loading = false
	}

	prs := newPRsModel(prFilters)
	prs.uiTheme = theme

	// Seed PRs list from cache for instant startup display.
	if cachedPRs, ok := cache.GetPRs(); ok {
		items := make([]list.Item, len(cachedPRs))
		for i, pr := range cachedPRs {
			items[i] = prListItem{pr: pr}
		}
		prs.list.SetItems(items)
		prs.loaded = true
		prs.loading = false
	}

	helpVP := viewport.New(80, 20) // resized on first WindowSizeMsg / help open
	helpVP.Style = lipgloss.NewStyle()

	return AppModel{
		activeTab:       TabDashboard,
		screen:          ScreenList,
		repo:            repo,
		cfg:             cfg,
		spinner:         s,
		issues:          issues,
		prs:             prs,
		dashboard:       newDashboardModel(cfg),
		issueFilterVals: &IssueFilterVals{},
		prFilterVals:    &PRFilterVals{},
		configVals:      &ConfigVals{},
		helpVP:          helpVP,
	}
}

type currentUserFetchedMsg struct {
	login string
	err   error
}

func fetchCurrentUserCmd() tea.Cmd {
	return func() tea.Msg {
		login, err := github.GetCurrentUser()
		return currentUserFetchedMsg{login: login, err: err}
	}
}

func (m AppModel) Init() tea.Cmd {
	cmds := []tea.Cmd{
		m.spinner.Tick,
		m.dashboard.spinner.Tick,
		m.dashboard.fetchCmd(),
		fetchCurrentUserCmd(),
	}

	// Parallel startup prefetch: fire Issues and PRs fetches immediately so
	// switching tabs is instant. If the sub-model was seeded from cache
	// (loaded=true, loading=false) this acts as a stale-while-revalidate
	// background refresh; if not seeded, the spinner is shown.
	cmds = append(cmds, m.issues.fetchCmd(), m.prs.fetchCmd())
	if !m.issues.loaded {
		cmds = append(cmds, m.issues.spinner.Tick)
	}
	if !m.prs.loaded {
		cmds = append(cmds, m.prs.spinner.Tick)
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

// saveIssuesToCacheCmd returns a fire-and-forget Cmd that persists the issue
// list to disk without blocking the UI.
func saveIssuesToCacheCmd(issues []github.Issue) tea.Cmd {
	return func() tea.Msg {
		c := loadCache()
		c.SetIssues(issues)
		_ = saveCache(c)
		return nil
	}
}

// savePRsToCacheCmd returns a fire-and-forget Cmd that persists the PR list
// to disk without blocking the UI.
func savePRsToCacheCmd(prs []github.PullRequest) tea.Cmd {
	return func() tea.Msg {
		c := loadCache()
		c.SetPRs(prs)
		_ = saveCache(c)
		return nil
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

	// ── Help overlay — scroll or dismiss ─────────────────────────────────────
	if m.showHelp {
		if wm, ok := msg.(tea.WindowSizeMsg); ok {
			m.width = wm.Width
			m.height = wm.Height
			m.helpVP.Width = wm.Width - 4
			m.helpVP.Height = wm.Height - 5
		}
		if kmsg, ok := msg.(tea.KeyMsg); ok {
			switch kmsg.String() {
			case "up", "k", "down", "j", "pgup", "pgdown", "ctrl+u", "ctrl+d":
				var cmd tea.Cmd
				m.helpVP, cmd = m.helpVP.Update(kmsg)
				return m, cmd
			default:
				m.showHelp = false
				return m, nil
			}
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
			// Propagate relevant settings to sub-models.
			m.dashboard.cfg = newCfg
			theme := resolveTheme(newCfg.UITheme)
			m.issues.uiTheme = theme
			m.prs.uiTheme = theme
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
	case currentUserFetchedMsg:
		if msg.err == nil {
			m.currentUser = msg.login
			m.issues.currentUser = msg.login
		}
		return m, nil

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
		theme := resolveTheme(m.cfg.UITheme)
		fw := formWidth(m.width-2, theme)
		if msg.forPRs {
			InitPRFilterVals(m.prFilterVals, m.prs.filters, msg.assignees)
			m.filterForm = BuildPRFilterForm(m.prFilterVals, msg.assignees, msg.labels).
				WithWidth(fw)
		} else {
			InitIssueFilterVals(m.issueFilterVals, m.issues.filters, msg.assignees)
			m.filterForm = BuildIssueFilterForm(m.issueFilterVals, msg.assignees, msg.labels).
				WithWidth(fw)
		}
		return m, m.filterForm.Init()

	case issueFiltersUpdatedMsg:
		m.issues.filters = msg.filters
		m.issues.loading = true
		m.issues.loaded = false
		m.cfg.IssueFilters = msg.filters
		_ = saveConfig(m.cfg)
		return m, tea.Batch(m.issues.fetchCmd(), m.issues.spinner.Tick)

	case prFiltersUpdatedMsg:
		m.prs.filters = msg.filters
		m.prs.loading = true
		m.prs.loaded = false
		m.cfg.PRFilters = msg.filters
		_ = saveConfig(m.cfg)
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
			if m.showHelp {
				m.helpVP.Width = m.width - 4
				m.helpVP.Height = m.height - 5
				m.helpVP.SetContent(buildHelpContent(m.width - 4))
				m.helpVP.GotoTop()
			}
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
			m.configForm = BuildConfigForm(m.configVals).WithWidth(formWidth(m.width-2, resolveTheme(m.cfg.UITheme)))
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
			// Skip fetch if already loaded or a background fetch is in flight
			// (parallel startup prefetch fires fetchCmd in Init).
			if !m.issues.loaded && !m.issues.loading {
				m.issues.loading = true
				cmds = append(cmds, m.issues.fetchCmd(), m.issues.spinner.Tick)
			}
		case TabPRs:
			// Same guard as TabIssues above.
			if !m.prs.loaded && !m.prs.loading {
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

	// ── Always-route messages ─────────────────────────────────────────────
	// These messages must reach their target sub-model regardless of which tab
	// is currently active. This supports:
	//   • Parallel startup prefetch (issuesFetched / prsFetched fired in Init)
	//   • Detail prefetching (prefetch goroutines complete after tab switch)
	// We also save fresh list data to the disk cache here.

	case issuesFetchedMsg:
		updated, cmd := m.issues.Update(msg)
		m.issues = updated
		if cmd != nil {
			cmds = append(cmds, cmd)
		}
		if msg.err == nil {
			cmds = append(cmds, saveIssuesToCacheCmd(msg.issues))
		}
		if wasInDetail != inDetailMode(m) {
			cmds = append(cmds, tea.ClearScreen)
		}
		return m, tea.Batch(cmds...)

	case prsFetchedMsg:
		updated, cmd := m.prs.Update(msg)
		m.prs = updated
		if cmd != nil {
			cmds = append(cmds, cmd)
		}
		if msg.err == nil {
			cmds = append(cmds, savePRsToCacheCmd(msg.prs))
		}
		if wasInDetail != inDetailMode(m) {
			cmds = append(cmds, tea.ClearScreen)
		}
		return m, tea.Batch(cmds...)

	case issuePrefetchedMsg:
		updated, cmd := m.issues.Update(msg)
		m.issues = updated
		if cmd != nil {
			cmds = append(cmds, cmd)
		}
		return m, tea.Batch(cmds...)

	case prPrefetchedMsg:
		updated, cmd := m.prs.Update(msg)
		m.prs = updated
		if cmd != nil {
			cmds = append(cmds, cmd)
		}
		return m, tea.Batch(cmds...)
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

// quitConfirmFooter renders the quit confirmation prompt.
// y/Y confirms; any other key cancels.
func quitConfirmFooter(width int, theme UITheme) string {
	return RenderFooterBar(width, theme,
		NewKeyButton("y", "confirm quit", ColorDanger),
		NewKeyButton("any key", "cancel", ColorMeta),
	)
}

// footerPendingToast renders a "working…" indicator in the footer.
func footerPendingToast(spinnerView string, msg string, width int, theme UITheme) string {
	bgSt := lipgloss.NewStyle().Background(footerBg)
	spinSt := lipgloss.NewStyle().Background(footerBg).Foreground(ColorAction)
	txtSt := lipgloss.NewStyle().Background(footerBg).Foreground(lipgloss.Color("244"))
	line := bgSt.Render("  ") + spinSt.Render(spinnerView) + " " + txtSt.Render(msg)
	return CenterInFooterBar(line, width, theme)
}

// footerToast renders a success / error toast notification.
func footerToast(msg string, isErr bool, width int, theme UITheme) string {
	fg := ColorAction // green
	icon := "✓"
	if isErr {
		fg = ColorDanger // red
		icon = "✗"
	}
	bgSt := lipgloss.NewStyle().Background(footerBg)
	txtSt := lipgloss.NewStyle().Background(footerBg).Foreground(fg).Bold(true)
	line := bgSt.Render("  ") + txtSt.Render(icon+" "+msg)
	return CenterInFooterBar(line, width, theme)
}

// detailActionFooter renders the context-specific footer shown in detail views.
// When there is an active toast (action message / error), it replaces the key
// hints for the duration of the auto-dismiss timer — keeping the layout height
// exactly constant.
func detailActionFooter(m AppModel, width int) string {
	theme := resolveTheme(m.cfg.UITheme)

	// Toast / pending indicator — shown in place of key hints.
	// Priority: error > success > pending (working…)
	if m.activeTab == TabIssues && m.issues.showDetail {
		if m.issues.actionErr != nil {
			return footerToast(m.issues.actionErr.Error(), true, width, theme)
		}
		if m.issues.actionMsg != "" {
			return footerToast(m.issues.actionMsg, false, width, theme)
		}
		if m.issues.actionPending != "" {
			return footerPendingToast(m.issues.spinner.View(), m.issues.actionPending, width, theme)
		}
	}
	if m.activeTab == TabPRs && m.prs.showDetail {
		if m.prs.actionErr != nil {
			return footerToast(m.prs.actionErr.Error(), true, width, theme)
		}
		if m.prs.actionMsg != "" {
			return footerToast(m.prs.actionMsg, false, width, theme)
		}
		if m.prs.actionPending != "" {
			return footerPendingToast(m.prs.spinner.View(), m.prs.actionPending, width, theme)
		}
	}

	kb := func(b key.Binding, desc string, c lipgloss.Color) KeyButton {
		return NewKeyButton(b.Help().Key, desc, c)
	}

	var btns []KeyButton
	switch {
	case m.activeTab == TabIssues && m.issues.showDetail:
		if strings.EqualFold(m.issues.detailIssue.State, "closed") {
			// Closed issue: only allow back, browser, help, and reopen.
			btns = []KeyButton{
				kb(keys.Back, "back", ColorMeta),
				kb(keys.Browser, "browser", ColorMeta),
				kb(keys.Help, "more actions", ColorMeta),
				kb(keys.IssueClose, "reopen", ColorDanger),
			}
		} else {
			assignLabel := "assign"
			if isMeAssigned(m.issues.detailIssue.Assignees, m.currentUser) {
				assignLabel = "unassign"
			}
			btns = []KeyButton{
				kb(keys.Back, "back", ColorMeta),
				kb(keys.IssueClose, "close", ColorDanger),
				kb(keys.IssueAssign, assignLabel, ColorAction),
				kb(keys.IssueDevelop, "develop", ColorAction),
				kb(keys.Browser, "browser", ColorMeta),
				kb(keys.Help, "more actions", ColorMeta),
			}
		}
	case m.activeTab == TabPRs && m.prs.showDetail:
		pr := m.prs.detailPR
		switch {
		case strings.EqualFold(pr.State, "merged"):
			// Merged PR: no further actions — can't reopen, checkout, or merge again.
			btns = []KeyButton{
				kb(keys.Back, "back", ColorMeta),
				kb(keys.Browser, "browser", ColorMeta),
				kb(keys.Help, "more actions", ColorMeta),
			}
		case strings.EqualFold(pr.State, "closed"):
			// Closed (not merged) PR: only reopen is meaningful.
			btns = []KeyButton{
				kb(keys.Back, "back", ColorMeta),
				kb(keys.Browser, "browser", ColorMeta),
				kb(keys.Help, "more actions", ColorMeta),
				kb(keys.PRClose, "reopen", ColorDanger),
			}
		default:
			// Open or draft PR: full action set.
			btns = []KeyButton{
				kb(keys.Back, "back", ColorMeta),
				kb(keys.PRClose, "close", ColorDanger),
				kb(keys.PRCheckout, "checkout", ColorAction),
				kb(keys.PRMerge, "merge", ColorAction),
				kb(keys.PRReview, "reviewer", ColorAction),
				kb(keys.Browser, "browser", ColorMeta),
				kb(keys.Help, "more actions", ColorMeta),
			}
		}
	}

	return RenderFooterBar(width, theme, btns...)
}

// isMeAssigned reports whether login appears in the assignees list.
// Returns false if login is empty (current user not yet fetched).
func isMeAssigned(assignees []github.User, login string) bool {
	if login == "" {
		return false
	}
	for _, a := range assignees {
		if strings.EqualFold(a.Login, login) {
			return true
		}
	}
	return false
}

func footerView(m AppModel, width int, theme UITheme) string {
	kb := func(b key.Binding, desc string, c lipgloss.Color) KeyButton {
		return NewKeyButton(b.Help().Key, desc, c)
	}

	// List-level action feedback (e.g. assign from list) — show toast in place
	// of the normal buttons so the user knows the action is in flight or done.
	if m.activeTab == TabIssues && !m.issues.showDetail {
		if m.issues.actionErr != nil {
			return footerToast(m.issues.actionErr.Error(), true, width, theme)
		}
		if m.issues.actionMsg != "" {
			return footerToast(m.issues.actionMsg, false, width, theme)
		}
		if m.issues.actionPending != "" {
			return footerPendingToast(m.issues.spinner.View(), m.issues.actionPending, width, theme)
		}
	}

	switch m.activeTab {
	case TabDashboard:
		return RenderFooterBar(width, theme,
			kb(keys.Up, "navigate", ColorAction),
			kb(keys.Open, "open", ColorAction),
			kb(keys.Tab, "switch view", ColorMeta),
			kb(keys.Help, "shortcuts", ColorMeta),
		)
	case TabIssues:
		// Grab / Take / Drop label depends on the selected issue's assignees.
		assignLabel, assignColor := "grab", ColorAction
		if item, ok := m.issues.list.SelectedItem().(issueListItem); ok {
			switch {
			case isMeAssigned(item.issue.Assignees, m.currentUser):
				assignLabel, assignColor = "drop", ColorDanger
			case len(item.issue.Assignees) > 0:
				assignLabel, assignColor = "take", ColorMeta
			}
		}
		return RenderFooterBar(width, theme,
			kb(keys.Open, "open", ColorAction),
			kb(keys.Browser, "browser", ColorMeta),
			kb(keys.New, "new issue", ColorAction),
			NewKeyButton(keys.IssueAssign.Help().Key, assignLabel, assignColor),
			kb(keys.Help, "shortcuts", ColorMeta),
		)
	case TabPRs:
		return RenderFooterBar(width, theme,
			kb(keys.Open, "open", ColorAction),
			kb(keys.Browser, "browser", ColorMeta),
			kb(keys.New, "new PR", ColorAction),
			kb(keys.PRCheckout, "checkout", ColorAction),
			kb(keys.Help, "shortcuts", ColorMeta),
		)
	default:
		return RenderFooterBar(width, theme, kb(keys.Help, "shortcuts", ColorMeta))
	}
}

// buildHelpContent returns the full scrollable help text showing every
// keyboard shortcut in the application, organised by section.
// contentW is the available visual width for line-length purposes.
func buildHelpContent(contentW int) string {
	titleStyle := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("208"))
	sectionStyle := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("86"))
	keyStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("208")).Bold(true)
	descStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("244"))
	dimStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("238"))

	// keyColW reserves enough space for the widest badge "[shift+tab]" (11
	// visible chars) plus a comfortable gap before the description.
	const keyColW = 16

	// line renders one shortcut row with a fixed-width key column.
	// Pass desc="" to fall back to the binding's own help description.
	line := func(b key.Binding, desc string) string {
		if desc == "" {
			desc = b.Help().Desc
		}
		badge := keyStyle.Render("[" + b.Help().Key + "]")
		gap := keyColW - lipgloss.Width(badge)
		if gap < 2 {
			gap = 2
		}
		return "    " + badge + strings.Repeat(" ", gap) + descStyle.Render(desc)
	}

	sec := func(title string) string {
		return "\n  " + sectionStyle.Render(title) + "\n"
	}

	hr := dimStyle.Render(strings.Repeat("─", contentW-4))

	var buf strings.Builder

	buf.WriteString("\n  " + titleStyle.Render("KEYBOARD SHORTCUTS") + "\n")
	buf.WriteString("  " + hr + "\n")

	// ── General ───────────────────────────────────────────────────────────
	buf.WriteString(sec("General"))
	buf.WriteString(line(keys.Tab, "switch to next tab") + "\n")
	buf.WriteString(line(keys.ShiftTab, "switch to previous tab") + "\n")
	buf.WriteString("\n")
	buf.WriteString(line(keys.Config, "open settings") + "\n")
	buf.WriteString(line(keys.Refresh, "refresh current view") + "\n")
	buf.WriteString("\n")
	buf.WriteString(line(keys.Help, "open / close this screen") + "\n")
	buf.WriteString(line(keys.Quit, "quit") + "\n")

	// ── Browse (list view) ────────────────────────────────────────────────
	buf.WriteString(sec("Browse"))
	buf.WriteString(line(keys.Up, "navigate up / down") + "\n")
	buf.WriteString(line(keys.Top, "jump to top") + "\n")
	buf.WriteString(line(keys.Bottom, "jump to bottom") + "\n")
	buf.WriteString("\n")
	buf.WriteString(line(keys.Open, "open item") + "\n")
	buf.WriteString(line(keys.Browser, "open in browser") + "\n")
	buf.WriteString(line(keys.Filters, "filter list") + "\n")

	// ── Issues list ───────────────────────────────────────────────────────
	buf.WriteString(sec("Issues — List"))
	buf.WriteString(line(keys.New, "create new issue") + "\n")
	buf.WriteString(line(keys.IssueAssign, "grab (unassigned) / take (from someone) / drop (yourself)") + "\n")

	// ── Issue detail ──────────────────────────────────────────────────────
	buf.WriteString(sec("Issues — Detail"))
	buf.WriteString(line(keys.IssueDevelop, "create a branch for this issue") + "\n")
	buf.WriteString(line(keys.IssuePR, "create a pull request") + "\n")
	buf.WriteString(line(keys.IssueClose, "close / reopen issue") + "\n")
	buf.WriteString(line(keys.IssueAssign, "assign / unassign @me") + "\n")
	buf.WriteString(line(keys.IssueLabel, "add label") + "\n")
	buf.WriteString("\n")
	buf.WriteString(line(keys.Browser, "open in browser") + "\n")
	buf.WriteString(line(keys.CopyURL, "copy URL to clipboard") + "\n")
	buf.WriteString("\n")
	buf.WriteString(line(keys.Refresh, "refresh issue") + "\n")
	buf.WriteString(line(keys.Back, "back to list") + "\n")

	// ── Pull requests list ────────────────────────────────────────────────
	buf.WriteString(sec("Pull Requests — List"))
	buf.WriteString(line(keys.New, "create new pull request") + "\n")
	buf.WriteString(line(keys.Browser, "open selected PR in browser") + "\n")

	// ── Pull request detail ───────────────────────────────────────────────
	buf.WriteString(sec("Pull Requests — Detail"))
	buf.WriteString(line(keys.PRCheckout, "checkout branch locally") + "\n")
	buf.WriteString(line(keys.PRMerge, "merge pull request") + "\n")
	buf.WriteString(line(keys.PRClose, "close / reopen pull request") + "\n")
	buf.WriteString(line(keys.PRReview, "request a reviewer") + "\n")
	buf.WriteString("\n")
	buf.WriteString(line(keys.Browser, "open in browser") + "\n")
	buf.WriteString(line(keys.CopyURL, "copy URL to clipboard") + "\n")
	buf.WriteString("\n")
	buf.WriteString(line(keys.Refresh, "refresh PR") + "\n")
	buf.WriteString(line(keys.Back, "back to list") + "\n")

	buf.WriteString("\n")
	return buf.String()
}

// helpOverlayView renders the full keyboard-shortcut reference inside a
// scrollable viewport. ↑/↓ scroll; any other key dismisses.
func helpOverlayView(m AppModel, innerW int) string {
	dimStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("238"))

	hint := "  " + dimStyle.Render("↑ / ↓  scroll    ·    any other key to close")

	appBorder := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("208")).
		Width(innerW).
		Height(m.height - 2)
	return appBorder.Render(m.helpVP.View() + "\n" + hint)
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

	theme := resolveTheme(m.cfg.UITheme)
	var footer string
	switch {
	case m.confirmingQuit:
		footer = quitConfirmFooter(innerW, theme)
	case inDetail:
		footer = detailActionFooter(m, innerW)
	default:
		footer = footerView(m, innerW, theme)
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
