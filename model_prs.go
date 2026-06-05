// model_prs.go
package main

import (
	"fmt"
	"io"
	"strings"
	"time"

	"hubcap/internal/github"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/huh"
	"github.com/charmbracelet/lipgloss"
)

// ── PR action messages ────────────────────────────────────────────────────────

type prActionDoneMsg struct {
	message    string
	number     int  // PR to re-fetch after action (0 = don't re-fetch)
	reloadList bool // refresh the full list (e.g. after creating a PR)
}

type prActionErrMsg struct {
	err error
}

// clearPRActionMsgMsg is sent by a timer to dismiss the toast notification.
type clearPRActionMsgMsg struct{}

func clearPRActionMsgCmd() tea.Cmd {
	return tea.Tick(3*time.Second, func(time.Time) tea.Msg {
		return clearPRActionMsgMsg{}
	})
}

// silentPRListRefreshMsg carries a fresh PR list fetched in the background.
type silentPRListRefreshMsg struct {
	prs []github.PullRequest
	err error
}

// silentFetchCmd re-fetches the PR list without changing any loading state.
func (m PRsModel) silentFetchCmd() tea.Cmd {
	filters := m.filters
	return func() tea.Msg {
		prs, err := github.FetchPRs(filters)
		return silentPRListRefreshMsg{prs: prs, err: err}
	}
}

// ── prFormType ────────────────────────────────────────────────────────────────

type prFormType int

const (
	prFormNone  prFormType = iota
	prFormMerge            // "m" — choose merge strategy
	prFormNew              // "n" — create a new PR
)

// prFormVals is heap-allocated so huh's Value() pointers remain valid across
// BubbleTea value-receiver model copies.
type prFormVals struct {
	formType  prFormType
	mergeType string
	newTitle  string
	newBody   string
	newBase   string
	newDraft  bool
}

// ── PRListItem ────────────────────────────────────────────────────────────────

type prListItem struct {
	pr github.PullRequest
}

func (p prListItem) Title() string {
	return fmt.Sprintf("#%-5d %s", p.pr.Number, p.pr.Title)
}
func (p prListItem) Description() string {
	status := p.pr.State
	if p.pr.IsDraft {
		status = "draft"
	}
	return fmt.Sprintf("%s  %s  %s", p.pr.Author.Login, status, summarizeChecks(p.pr.StatusRollup))
}
func (p prListItem) FilterValue() string {
	return fmt.Sprintf("%d %s", p.pr.Number, p.pr.Title)
}

// ── prDelegate ────────────────────────────────────────────────────────────────
// Compact single-line delegate: Height=1, Spacing=0.
// Layout per row:
//
//	[accent] ● #N   Title…(fill)…  author  checks  label1 · label2

type prDelegate struct{}

func (d prDelegate) Height() int                             { return 1 }
func (d prDelegate) Spacing() int                            { return 0 }
func (d prDelegate) Update(_ tea.Msg, _ *list.Model) tea.Cmd { return nil }

func (d prDelegate) Render(w io.Writer, m list.Model, index int, item list.Item) {
	pi, ok := item.(prListItem)
	if !ok {
		return
	}
	pr := pi.pr
	width := m.Width()
	selected := index == m.Index()

	selectedBg := lipgloss.Color("235")
	var base lipgloss.Style
	if selected {
		base = lipgloss.NewStyle().Background(selectedBg)
	} else {
		base = lipgloss.NewStyle()
	}

	// Left accent bar.
	var accent string
	if selected {
		accent = lipgloss.NewStyle().Foreground(lipgloss.Color("86")).Background(selectedBg).Render("▌") +
			base.Render(" ")
	} else {
		accent = "  "
	}

	// ⤴ colored by state: green = open, red = closed, purple = merged, amber = draft.
	var dotColor lipgloss.Color
	switch {
	case pr.IsDraft:
		dotColor = lipgloss.Color("214")
	case strings.EqualFold(pr.State, "merged"):
		dotColor = lipgloss.Color("141")
	case strings.EqualFold(pr.State, "closed"):
		dotColor = lipgloss.Color("196")
	default:
		dotColor = lipgloss.Color("83")
	}
	dot := base.Foreground(dotColor).Bold(true).Render("⤴")

	// PR number.
	numStyle := base.Foreground(lipgloss.Color("69"))
	if selected {
		numStyle = numStyle.Bold(true)
	}
	numStr := numStyle.Render(fmt.Sprintf(" #%-4d", pr.Number))

	// Right column: author · checks · labels (max 50 chars).
	var bgKey string
	if selected {
		bgKey = "235"
	}
	// Show "head → base" merge direction
	authorStyle := base.Foreground(lipgloss.Color("244"))
	arrowStyle := base.Foreground(lipgloss.Color("252"))
	var authorStr string
	if pr.BaseRefName != "" {
		authorStr = authorStyle.Render(truncate(pr.HeadRefName, 18)) +
			arrowStyle.Render(" → ") +
			authorStyle.Render(pr.BaseRefName)
	} else {
		authorStr = authorStyle.Render(truncate(pr.Author.Login, 14))
	}

	checksStr := prRowChecks(pr.StatusRollup, bgKey)

	// Right column budget: ~40% of list width; labels fill what author+checks don't use.
	rightBudget := width * 40 / 100
	authorW := lipgloss.Width(authorStr)
	checksW := lipgloss.Width(checksStr)
	labelBudget := rightBudget - authorW - checksW - 4 // 4 = two "  " separators
	if labelBudget < 5 {
		labelBudget = 5
	}
	labelStr := issueRowLabels(pr.Labels, bgKey, labelBudget)

	sep := base.Foreground(lipgloss.Color("238")).Render("  ")
	rightStr := authorStr
	if checksStr != "" {
		rightStr += sep + checksStr
	}
	if labelStr != "" {
		rightStr += sep + labelStr
	}
	rightW := lipgloss.Width(rightStr)

	// Title fills the middle.
	fixed := 2 + 1 + lipgloss.Width(numStr) + 1 + 2 + 1
	totalMid := width - fixed - rightW
	if totalMid < 10 {
		totalMid = 10
	}

	titleStyle := base.Foreground(lipgloss.Color("252"))
	if selected {
		titleStyle = base.Foreground(lipgloss.Color("255")).Bold(true)
	}
	titleStr := titleStyle.Render(truncate(pr.Title, totalMid))
	titleActualW := lipgloss.Width(titleStr)

	fill := base.Render(strings.Repeat(" ", totalMid-titleActualW))

	line := accent + dot + numStr + " " + titleStr + fill + "  " + rightStr + base.Render(" ")
	fmt.Fprint(w, line)
}

// prRowChecks returns a compact colored check-status symbol for a list row.
func prRowChecks(checks []github.CheckRun, bgKey string) string {
	if len(checks) == 0 {
		return ""
	}
	var base lipgloss.Style
	if bgKey != "" {
		base = lipgloss.NewStyle().Background(lipgloss.Color(bgKey))
	} else {
		base = lipgloss.NewStyle()
	}
	pending := false
	for _, c := range checks {
		if c.Conclusion == "FAILURE" || c.Conclusion == "ERROR" || c.Conclusion == "TIMED_OUT" {
			return base.Foreground(lipgloss.Color("196")).Render("✗ failing")
		}
		if c.Status != "COMPLETED" {
			pending = true
		}
	}
	if pending {
		return base.Foreground(lipgloss.Color("214")).Render("… pending")
	}
	return base.Foreground(lipgloss.Color("83")).Render("✓ passing")
}

// ── PRsModel ──────────────────────────────────────────────────────────────────

type PRsModel struct {
	list    list.Model
	spinner spinner.Model
	loading bool
	loaded  bool
	err     error
	filters github.PRFilters
	width   int
	height  int

	showDetail    bool
	detail        viewport.Model
	detailPR      github.PullRequest
	loadingDetail bool

	// Action feedback.
	// actionPending is set immediately on key press so the footer shows
	// a "working…" indicator before the goroutine returns.
	actionPending string
	actionMsg     string
	actionErr     error

	// Inline form (merge strategy, new PR). activeForm is nil when closed.
	// formVals is heap-allocated for stable pointers across model copies.
	activeForm *huh.Form
	formVals   *prFormVals

}

func newPRsModel(filters github.PRFilters) PRsModel {
	s := spinner.New()
	s.Spinner = spinner.Dot
	s.Style = lipgloss.NewStyle().Foreground(lipgloss.Color("86"))

	l := list.New([]list.Item{}, prDelegate{}, 0, 0)
	l.SetShowTitle(false)
	l.SetShowStatusBar(false)
	l.SetShowHelp(false)
	l.SetFilteringEnabled(true)
	// Disable the list's built-in quit keybindings — must use this method, not
	// SetEnabled(false), which gets overridden on every item load.
	l.DisableQuitKeybindings()

	// Align the list's navigation key map with our central registry.
	l.KeyMap.CursorUp = keys.Up
	l.KeyMap.CursorDown = keys.Down
	l.KeyMap.GoToStart = keys.Top
	l.KeyMap.GoToEnd = keys.Bottom

	return PRsModel{
		list:     l,
		spinner:  s,
		loading:  true,
		filters:  filters,
		formVals: &prFormVals{mergeType: "rebase", newBase: "main"},
	}
}

// IsFiltering reports whether the list's inline filter is currently active.
func (m PRsModel) IsFiltering() bool { return m.list.SettingFilter() }

func (m PRsModel) fetchCmd() tea.Cmd {
	return func() tea.Msg {
		prs, err := github.FetchPRs(m.filters)
		return prsFetchedMsg{prs: prs, err: err}
	}
}

func fetchPRDetailCmd(number int) tea.Cmd {
	return func() tea.Msg {
		pr, err := github.FetchPR(number)
		return prFetchedMsg{pr: pr, err: err}
	}
}

// ── handleFormComplete ────────────────────────────────────────────────────────

func (m PRsModel) handleFormComplete() (PRsModel, tea.Cmd) {
	ft := m.formVals.formType
	m.activeForm = nil
	m.formVals.formType = prFormNone

	switch ft {
	case prFormMerge:
		strategy := m.formVals.mergeType
		pr := m.detailPR
		m.actionPending = fmt.Sprintf("Merging PR #%d (%s)…", pr.Number, strategy)
		m.actionMsg = ""
		m.actionErr = nil
		return m, tea.Batch(m.spinner.Tick, func() tea.Msg {
			if err := github.MergePR(pr.Number, strategy); err != nil {
				return prActionErrMsg{err: err}
			}
			return prActionDoneMsg{
				message: fmt.Sprintf("PR #%d merged (%s).", pr.Number, strategy),
				number:  pr.Number,
			}
		})

	case prFormNew:
		title := strings.TrimSpace(m.formVals.newTitle)
		if title == "" {
			return m, nil
		}
		body := m.formVals.newBody
		base := m.formVals.newBase
		draft := m.formVals.newDraft
		filters := m.filters
		m.loading = true
		m.loaded = false
		return m, tea.Batch(
			m.spinner.Tick,
			func() tea.Msg {
				if err := github.CreatePR(title, body, base, draft); err != nil {
					return prActionErrMsg{err: err}
				}
				prs, err := github.FetchPRs(filters)
				return prsFetchedMsg{prs: prs, err: err}
			},
		)
	}

	return m, nil
}

// ── Update ────────────────────────────────────────────────────────────────────

func (m PRsModel) Update(msg tea.Msg) (PRsModel, tea.Cmd) {
	var cmds []tea.Cmd

	// ── Embedded form takes priority ──────────────────────────────────────────
	if m.activeForm != nil {
		form, cmd := m.activeForm.Update(msg)
		if f, ok := form.(*huh.Form); ok {
			m.activeForm = f
		}
		switch m.activeForm.State {
		case huh.StateCompleted:
			return m.handleFormComplete()
		case huh.StateAborted:
			m.activeForm = nil
			m.formVals.formType = prFormNone
			return m, nil
		}
		return m, cmd
	}

	// ── Normal update ─────────────────────────────────────────────────────────
	switch msg := msg.(type) {
	case prsFetchedMsg:
		m.loading = false
		if msg.err != nil {
			m.err = msg.err
			m.loaded = true
			return m, nil
		}
		items := make([]list.Item, len(msg.prs))
		for i, pr := range msg.prs {
			items[i] = prListItem{pr: pr}
		}
		m.list.SetItems(items)
		m.loaded = true
		m.list.SetSize(m.width-4, m.height-headerHeight()-2)

	case prFetchedMsg:
		m.loadingDetail = false
		if msg.err != nil {
			m.err = msg.err
			return m, nil
		}
		m.detailPR = msg.pr
		content := renderPRDetailContent(msg.pr, m.width)
		m.detail = viewport.New(m.width-4, m.height-headerHeightDetail-metaStripHeight-2)
		m.detail.SetContent(content)
		m.showDetail = true

	case silentPRListRefreshMsg:
		if msg.err == nil {
			items := make([]list.Item, len(msg.prs))
			for i, pr := range msg.prs {
				items[i] = prListItem{pr: pr}
			}
			m.list.SetItems(items)
		}
		return m, nil

	case prActionDoneMsg:
		m.actionPending = ""
		m.actionMsg = msg.message
		m.actionErr = nil
		if msg.number > 0 {
			m.loadingDetail = true
			return m, tea.Batch(
				fetchPRDetailCmd(msg.number),
				m.spinner.Tick,
				clearPRActionMsgCmd(),
				m.silentFetchCmd(),
			)
		}
		if msg.reloadList {
			m.loading = true
			m.loaded = false
			return m, tea.Batch(m.fetchCmd(), m.spinner.Tick)
		}
		return m, clearPRActionMsgCmd()

	case prActionErrMsg:
		m.actionPending = ""
		m.actionErr = msg.err
		m.actionMsg = ""
		return m, clearPRActionMsgCmd()

	case clearPRActionMsgMsg:
		m.actionPending = ""
		m.actionMsg = ""
		m.actionErr = nil
		return m, nil

	case tea.WindowSizeMsg:
		m.width = msg.Width - 2
		m.height = msg.Height - 2
		m.list.SetSize(m.width-4, m.height-headerHeight()-2)
		if m.showDetail {
			m.detail.Width = m.width - 4
			m.detail.Height = m.height - headerHeightDetail - metaStripHeight - 2
		}

	case spinner.TickMsg:
		// Keep spinning while loading or while a background action is in flight.
		if m.loading || m.loadingDetail || m.actionPending != "" {
			var cmd tea.Cmd
			m.spinner, cmd = m.spinner.Update(msg)
			cmds = append(cmds, cmd)
		}

	case tea.KeyMsg:
		if m.loading || m.loadingDetail {
			break
		}

		if m.showDetail {
			switch {
			case key.Matches(msg, keys.Back):
				m.showDetail = false
				m.actionMsg = ""
				m.actionErr = nil
				return m, nil
			case key.Matches(msg, keys.Refresh):
				m.loadingDetail = true
				m.actionMsg = ""
				m.actionErr = nil
				return m, fetchPRDetailCmd(m.detailPR.Number)
			case key.Matches(msg, keys.Browser):
				// Open in browser — instant, no pending indicator needed.
				url := m.detailPR.URL
				return m, func() tea.Msg {
					github.OpenURL(url)
					return nil
				}
			case key.Matches(msg, keys.CopyURL):
				m.actionPending = ""
				if err := copyText(m.detailPR.URL); err != nil {
					m.actionErr = err
					m.actionMsg = ""
				} else {
					m.actionMsg = "URL copied to clipboard."
					m.actionErr = nil
				}
				return m, clearPRActionMsgCmd()
			case key.Matches(msg, keys.PRClose):
				// Close or Reopen PR
				pr := m.detailPR
				isClosed := strings.EqualFold(pr.State, "closed")
				if isClosed {
					m.actionPending = "Reopening PR…"
				} else {
					m.actionPending = "Closing PR…"
				}
				m.actionMsg = ""
				m.actionErr = nil
				return m, tea.Batch(m.spinner.Tick, func() tea.Msg {
					var err error
					var done string
					if isClosed {
						err = github.ReopenPR(pr.Number)
						done = "PR reopened."
					} else {
						err = github.ClosePR(pr.Number)
						done = "PR closed."
					}
					if err != nil {
						return prActionErrMsg{err: err}
					}
					return prActionDoneMsg{message: done, number: pr.Number}
				})
			case key.Matches(msg, keys.PRCheckout):
				// Checkout branch.
				pr := m.detailPR
				m.actionPending = fmt.Sprintf("Checking out %q…", pr.HeadRefName)
				m.actionMsg = ""
				m.actionErr = nil
				return m, tea.Batch(m.spinner.Tick, func() tea.Msg {
					if err := github.CheckoutPR(pr.Number); err != nil {
						return prActionErrMsg{err: err}
					}
					return prActionDoneMsg{
						message: fmt.Sprintf("Checked out branch %q.", pr.HeadRefName),
					}
				})
			case key.Matches(msg, keys.PRMerge):
				// Merge — embedded select form for strategy choice.
				m.formVals.mergeType = "rebase"
				m.formVals.formType = prFormMerge
				prNumber := m.detailPR.Number
				m.activeForm = huh.NewForm(huh.NewGroup(
					huh.NewSelect[string]().
						Title(fmt.Sprintf("Merge PR #%d", prNumber)).
						Options(
							huh.NewOption("Rebase and merge", "rebase"),
							huh.NewOption("Squash and merge", "squash"),
							huh.NewOption("Merge commit", "merge"),
						).
						Value(&m.formVals.mergeType),
				)).WithTheme(huh.ThemeCatppuccin()).WithWidth(m.width - 8)
				return m, m.activeForm.Init()
			}
			// Viewport scrolling
			var cmd tea.Cmd
			m.detail, cmd = m.detail.Update(msg)
			cmds = append(cmds, cmd)
			return m, tea.Batch(cmds...)
		}

		// List view keys — only fire outside of the list's filter input.
		if !m.list.SettingFilter() {
			switch {
			case key.Matches(msg, keys.New):
				m.formVals.newTitle = ""
				m.formVals.newBody = ""
				m.formVals.newBase = "main"
				m.formVals.newDraft = false
				m.formVals.formType = prFormNew
				m.activeForm = huh.NewForm(huh.NewGroup(
					huh.NewInput().
						Title("New PR — Title").
						Placeholder("Short description").
						Value(&m.formVals.newTitle),
					huh.NewText().
						Title("Body").
						Placeholder("Describe the changes (optional)").
						CharLimit(4000).
						Value(&m.formVals.newBody),
					huh.NewInput().
						Title("Base branch").
						Placeholder("main").
						Value(&m.formVals.newBase),
					huh.NewConfirm().
						Title("Draft PR?").
						Value(&m.formVals.newDraft),
				)).WithTheme(huh.ThemeCatppuccin()).WithWidth(m.width - 8)
				return m, m.activeForm.Init()
			case key.Matches(msg, keys.Refresh):
				m.loading = true
				m.loaded = false
				cmds = append(cmds, m.fetchCmd())
				cmds = append(cmds, m.spinner.Tick)
			}
		}
		if key.Matches(msg, keys.Open) {
			if item, ok := m.list.SelectedItem().(prListItem); ok {
				m.loadingDetail = true
				cmds = append(cmds, fetchPRDetailCmd(item.pr.Number))
				cmds = append(cmds, m.spinner.Tick)
			}
		}
	}

	if !m.loading && !m.loadingDetail && !m.showDetail {
		var cmd tea.Cmd
		m.list, cmd = m.list.Update(msg)
		cmds = append(cmds, cmd)
	}

	return m, tea.Batch(cmds...)
}

// ── View ──────────────────────────────────────────────────────────────────────

func (m PRsModel) View() string {
	// When a form is active, render it — replacing list/detail content.
	if m.activeForm != nil {
		return m.activeForm.View()
	}

	var b strings.Builder

	if m.loading || m.loadingDetail {
		msg := "Fetching pull requests..."
		if m.loadingDetail {
			msg = fmt.Sprintf("Loading PR #%d...", func() int {
				if item, ok := m.list.SelectedItem().(prListItem); ok {
					return item.pr.Number
				}
				return 0
			}())
		}
		b.WriteString(fmt.Sprintf("\n  %s %s\n", m.spinner.View(), msg))
		return b.String()
	}

	if m.err != nil {
		b.WriteString(errorBox(fmt.Sprintf("Error: %v\n\nPress r to retry.", m.err)))
		return b.String()
	}

	if m.showDetail {
		b.WriteString(renderPRMetaStrip(m.detailPR, m.width-4))
		b.WriteString(renderPRDetailView(m.detailPR, m.detail, m.actionMsg, m.actionErr))
		return b.String()
	}

	b.WriteString(lipgloss.NewStyle().Margin(0, 2).Render(m.list.View()))
	return b.String()
}

// renderPRDetailContent builds scrollable body-only content for the viewport.
func renderPRDetailContent(pr github.PullRequest, _ int) string {
	var b strings.Builder
	if pr.Body != "" {
		b.WriteString(pr.Body + "\n")
	} else {
		b.WriteString(styleGray.Render("No description.") + "\n")
	}
	return b.String()
}

// renderPRDetailView renders the scrollable viewport only.
// Action feedback (toast) is shown in the footer bar by AppModel.
func renderPRDetailView(_ github.PullRequest, vp viewport.Model, _ string, _ error) string {
	return lipgloss.NewStyle().Margin(0, 2).Render(vp.View()) + "\n"
}
