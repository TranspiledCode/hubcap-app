// model_issues.go
package main

import (
	"fmt"
	"io"
	"strings"
	"time"

	"hubcap/internal/github"

	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/huh"
	"github.com/charmbracelet/lipgloss"
)

// ── Issue action messages ─────────────────────────────────────────────────────

type issueActionDoneMsg struct {
	message    string
	number     int  // issue to re-fetch in detail view (0 = don't re-fetch)
	reloadList bool // refresh the full list (e.g. after creating an issue)
}

type issueActionErrMsg struct {
	err error
}

// clearIssueActionMsgMsg is sent by a timer to dismiss the toast notification.
type clearIssueActionMsgMsg struct{}

func clearIssueActionMsgCmd() tea.Cmd {
	return tea.Tick(3*time.Second, func(time.Time) tea.Msg {
		return clearIssueActionMsgMsg{}
	})
}

// silentIssueListRefreshMsg carries a fresh issue list fetched in the
// background without triggering the loading spinner.
type silentIssueListRefreshMsg struct {
	issues []github.Issue
	err    error
}

// silentFetchCmd re-fetches the issue list without changing any loading state.
func (m IssuesModel) silentFetchCmd() tea.Cmd {
	filters := m.filters
	return func() tea.Msg {
		issues, err := github.FetchIssues(filters)
		return silentIssueListRefreshMsg{issues: issues, err: err}
	}
}

// ── issueFormType ─────────────────────────────────────────────────────────────

type issueFormType int

const (
	issueFormNone   issueFormType = iota
	issueFormLabel                // "l" — add a label to the open issue
	issueFormBranch               // "d" — create a development branch
	issueFormNew                  // "n" — create a new issue
)

// issueFormVals is heap-allocated so huh's Value() pointers remain valid
// across BubbleTea value-receiver model copies.
type issueFormVals struct {
	formType      issueFormType
	labelVal      string
	branchVal     string
	branchDefault string
	newTitle      string
	newBody       string
}

// ── IssueListItem ─────────────────────────────────────────────────────────────

type issueListItem struct {
	issue github.Issue
}

func (i issueListItem) Title() string {
	return fmt.Sprintf("#%-5d %s", i.issue.Number, i.issue.Title)
}
func (i issueListItem) Description() string {
	return fmt.Sprintf("%s  %s", joinUsers(i.issue.Assignees), coloredLabelsCompact(i.issue.Labels, 60))
}
func (i issueListItem) FilterValue() string {
	return fmt.Sprintf("%d %s", i.issue.Number, i.issue.Title)
}

// ── issueDelegate ─────────────────────────────────────────────────────────────
// Compact single-line delegate: Height=1, Spacing=0.
// Layout per row:
//
//	[accent] ● #N   Title…(fill)…  label1 · label2

type issueDelegate struct{}

func (d issueDelegate) Height() int                              { return 1 }
func (d issueDelegate) Spacing() int                             { return 0 }
func (d issueDelegate) Update(_ tea.Msg, _ *list.Model) tea.Cmd { return nil }

func (d issueDelegate) Render(w io.Writer, m list.Model, index int, item list.Item) {
	ii, ok := item.(issueListItem)
	if !ok {
		return
	}
	issue := ii.issue
	width := m.Width()
	selected := index == m.Index()

	// Base style — slightly lighter bg on the selected row.
	var base lipgloss.Style
	selectedBg := lipgloss.Color("235")
	if selected {
		base = lipgloss.NewStyle().Background(selectedBg)
	} else {
		base = lipgloss.NewStyle()
	}

	// Left accent bar (2 chars wide either way so layout stays stable).
	var accent string
	if selected {
		accent = lipgloss.NewStyle().Foreground(lipgloss.Color("86")).Background(selectedBg).Render("▌") +
			base.Render(" ")
	} else {
		accent = "  "
	}

	// ⚑ colored by state: green = open, red = closed.
	dotColor := lipgloss.Color("83")
	if !strings.EqualFold(issue.State, "open") {
		dotColor = lipgloss.Color("196")
	}
	dot := base.Foreground(dotColor).Bold(true).Render("⚑")

	// Issue number (right-padded to 5 chars for alignment: "#5   ", "#123 ").
	numStyle := base.Foreground(lipgloss.Color("69"))
	if selected {
		numStyle = numStyle.Bold(true)
	}
	numStr := numStyle.Render(fmt.Sprintf(" #%-4d", issue.Number))

	// Labels for the right column (max 44 chars, colored text + · separator).
	var bgKey string
	if selected {
		bgKey = "235"
	}
	labelStr := issueRowLabels(issue.Labels, bgKey, 44)
	labelW := lipgloss.Width(labelStr)

	// Calculate how much space the title can occupy.
	// fixed = accent(2) + dot(1) + numStr(6) + space(1) + gap-before-labels(2) + rightPad(1)
	fixed := 2 + 1 + lipgloss.Width(numStr) + 1 + 2 + 1
	totalMid := width - fixed - labelW
	if totalMid < 10 {
		totalMid = 10
	}

	titleStyle := base.Foreground(lipgloss.Color("252"))
	if selected {
		titleStyle = base.Foreground(lipgloss.Color("255")).Bold(true)
	}
	titleStr := titleStyle.Render(truncate(issue.Title, totalMid))
	titleActualW := lipgloss.Width(titleStr)

	// Fill gap between title and the label column.
	fill := base.Render(strings.Repeat(" ", totalMid-titleActualW))

	line := accent + dot + numStr + " " + titleStr + fill + "  " + labelStr + base.Render(" ")
	fmt.Fprint(w, line)
}

// issueRowLabels renders a short colored label string for a list row.
// bgKey is "" for no background or the color key (e.g. "235") when the row
// is selected — so each segment explicitly matches the row background.
func issueRowLabels(labels []github.Label, bgKey string, maxW int) string {
	if len(labels) == 0 {
		return ""
	}
	makeBase := func() lipgloss.Style {
		if bgKey != "" {
			return lipgloss.NewStyle().Background(lipgloss.Color(bgKey))
		}
		return lipgloss.NewStyle()
	}
	sep := makeBase().Foreground(lipgloss.Color("238")).Render(" · ")
	sepW := lipgloss.Width(sep)

	var parts []string
	used := 0
	for _, l := range labels {
		ls := labelStyle(l.Name)
		if bgKey != "" {
			ls = ls.Background(lipgloss.Color(bgKey))
		}
		rendered := ls.Render(l.Name)
		rw := lipgloss.Width(rendered)
		extra := 0
		if len(parts) > 0 {
			extra = sepW
		}
		if used+extra+rw > maxW {
			break
		}
		parts = append(parts, rendered)
		used += extra + rw
	}
	if len(parts) == 0 {
		return ""
	}
	return strings.Join(parts, sep)
}

// ── IssuesModel ───────────────────────────────────────────────────────────────

type IssuesModel struct {
	// List view
	list    list.Model
	spinner spinner.Model
	loading bool
	loaded  bool
	err     error
	filters github.Filters
	width   int
	height  int

	// Detail view
	showDetail    bool
	detail        viewport.Model
	detailIssue   github.Issue
	loadingDetail bool

	// Action feedback.
	// actionPending is set immediately on key press so the footer shows
	// a "working…" indicator before the goroutine returns.
	// actionMsg / actionErr are set when the goroutine completes.
	actionPending string
	actionMsg     string
	actionErr     error

	// Inline form (label, branch, new-issue). activeForm is nil when closed.
	// formVals is heap-allocated so its address is stable across model copies.
	activeForm *huh.Form
	formVals   *issueFormVals

	// Navigation signal back to AppModel
	action string
}

func newIssuesModel(filters github.Filters) IssuesModel {
	s := spinner.New()
	s.Spinner = spinner.Dot
	s.Style = lipgloss.NewStyle().Foreground(lipgloss.Color("86"))

	l := list.New([]list.Item{}, issueDelegate{}, 0, 0)
	l.SetShowTitle(false)
	l.SetShowStatusBar(false)
	l.SetShowHelp(false)
	l.SetFilteringEnabled(true)
	// Disable the list's built-in quit keybindings (q and esc) so they don't
	// call tea.Quit directly and bypass our confirmation prompt.
	// Must use DisableQuitKeybindings() — not SetEnabled(false) — because the
	// list re-enables Quit via SetEnabled(!disableQuitKeybindings) every time
	// items load or filter state changes.
	l.DisableQuitKeybindings()

	return IssuesModel{
		list:     l,
		spinner:  s,
		loading:  true,
		filters:  filters,
		formVals: &issueFormVals{},
	}
}

func (m IssuesModel) fetchCmd() tea.Cmd {
	return func() tea.Msg {
		issues, err := github.FetchIssues(m.filters)
		return issuesFetchedMsg{issues: issues, err: err}
	}
}

func fetchIssueDetailCmd(number int) tea.Cmd {
	return func() tea.Msg {
		issue, err := github.FetchIssue(number)
		return issueFetchedMsg{issue: issue, err: err}
	}
}

// ── handleFormComplete ────────────────────────────────────────────────────────

// handleFormComplete is called when m.activeForm reaches StateCompleted.
func (m IssuesModel) handleFormComplete() (IssuesModel, tea.Cmd) {
	ft := m.formVals.formType
	m.activeForm = nil
	m.formVals.formType = issueFormNone

	switch ft {
	case issueFormLabel:
		label := strings.TrimSpace(m.formVals.labelVal)
		if label == "" {
			return m, nil
		}
		m.actionPending = fmt.Sprintf("Adding label %q…", label)
		m.actionMsg = ""
		m.actionErr = nil
		issue := m.detailIssue
		return m, tea.Batch(m.spinner.Tick, func() tea.Msg {
			if err := github.AddIssueLabel(issue.Number, label); err != nil {
				return issueActionErrMsg{err: err}
			}
			return issueActionDoneMsg{
				message: fmt.Sprintf("Label %q added.", label),
				number:  issue.Number,
			}
		})

	case issueFormBranch:
		branchName := strings.TrimSpace(m.formVals.branchVal)
		if branchName == "" {
			branchName = m.formVals.branchDefault
		}
		m.actionPending = fmt.Sprintf("Creating branch %q…", branchName)
		m.actionMsg = ""
		m.actionErr = nil
		issue := m.detailIssue
		return m, tea.Batch(m.spinner.Tick, func() tea.Msg {
			if err := github.DevelopBranch(issue.Number, branchName); err != nil {
				return issueActionErrMsg{err: err}
			}
			return issueActionDoneMsg{
				message: fmt.Sprintf("Branch %q created and checked out.", branchName),
			}
		})

	case issueFormNew:
		title := strings.TrimSpace(m.formVals.newTitle)
		if title == "" {
			return m, nil
		}
		body := m.formVals.newBody
		filters := m.filters
		m.loading = true
		m.loaded = false
		return m, tea.Batch(
			m.spinner.Tick,
			func() tea.Msg {
				if err := github.CreateIssue(title, body, nil); err != nil {
					return issueActionErrMsg{err: err}
				}
				issues, err := github.FetchIssues(filters)
				return issuesFetchedMsg{issues: issues, err: err}
			},
		)
	}

	return m, nil
}

// ── Update ────────────────────────────────────────────────────────────────────

func (m IssuesModel) Update(msg tea.Msg) (IssuesModel, tea.Cmd) {
	var cmds []tea.Cmd

	// ── Embedded form takes priority ──────────────────────────────────────────
	// Route all messages to the active form exclusively so huh can manage its
	// own keyboard navigation. Return early to block all app shortcuts.
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
			m.formVals.formType = issueFormNone
			return m, nil
		}
		return m, cmd
	}

	// ── Normal update ─────────────────────────────────────────────────────────
	switch msg := msg.(type) {
	case issuesFetchedMsg:
		m.loading = false
		if msg.err != nil {
			m.err = msg.err
			m.loaded = true
			return m, nil
		}
		items := make([]list.Item, len(msg.issues))
		for i, issue := range msg.issues {
			items[i] = issueListItem{issue: issue}
		}
		m.list.SetItems(items)
		m.loaded = true
		m.list.SetSize(m.width-4, m.height-headerHeight()-2)

	case issueFetchedMsg:
		m.loadingDetail = false
		if msg.err != nil {
			m.err = msg.err
			return m, nil
		}
		m.detailIssue = msg.issue
		content := renderIssueDetailContent(msg.issue, m.width)
		m.detail = viewport.New(m.width-4, m.height-headerHeightDetail-metaStripHeight-2)
		m.detail.SetContent(content)
		m.showDetail = true

	case silentIssueListRefreshMsg:
		// Quietly update list items — no loading state change, no spinner.
		if msg.err == nil {
			items := make([]list.Item, len(msg.issues))
			for i, issue := range msg.issues {
				items[i] = issueListItem{issue: issue}
			}
			m.list.SetItems(items)
		}
		return m, nil

	case issueActionDoneMsg:
		m.actionPending = ""
		m.actionMsg = msg.message
		m.actionErr = nil
		if msg.number > 0 {
			m.loadingDetail = true
			// Refresh detail AND silently update the list so state changes
			// (close, reopen, assign, label) are reflected when returning.
			return m, tea.Batch(
				fetchIssueDetailCmd(msg.number),
				m.spinner.Tick,
				clearIssueActionMsgCmd(),
				m.silentFetchCmd(),
			)
		}
		if msg.reloadList {
			m.loading = true
			m.loaded = false
			return m, tea.Batch(m.fetchCmd(), m.spinner.Tick)
		}
		return m, clearIssueActionMsgCmd()

	case issueActionErrMsg:
		m.actionPending = ""
		m.actionErr = msg.err
		m.actionMsg = ""
		return m, clearIssueActionMsgCmd()

	case clearIssueActionMsgMsg:
		m.actionPending = ""
		m.actionMsg = ""
		m.actionErr = nil
		return m, nil

	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.list.SetSize(msg.Width-4, msg.Height-headerHeight()-2)
		if m.showDetail {
			m.detail.Width = msg.Width - 4
			m.detail.Height = msg.Height - headerHeightDetail - metaStripHeight - 2
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
			switch msg.String() {
			case "esc", "b", "backspace":
				m.showDetail = false
				m.actionMsg = ""
				m.actionErr = nil
				return m, nil
			case "q":
				m.action = "quit"
				return m, nil
			case "tab":
				m.showDetail = false
				m.action = "switch"
				return m, nil
			case "r":
				m.loadingDetail = true
				m.actionMsg = ""
				m.actionErr = nil
				return m, fetchIssueDetailCmd(m.detailIssue.Number)
			case "o":
				// Open in browser — instant, no pending indicator needed.
				url := m.detailIssue.URL
				return m, func() tea.Msg {
					github.OpenURL(url)
					return nil
				}
			case "u":
				m.actionPending = ""
				if err := copyText(m.detailIssue.URL); err != nil {
					m.actionErr = err
					m.actionMsg = ""
				} else {
					m.actionMsg = "URL copied to clipboard."
					m.actionErr = nil
				}
				return m, clearIssueActionMsgCmd()
			case "c":
				issue := m.detailIssue
				if strings.EqualFold(issue.State, "closed") {
					m.actionPending = "Reopening issue…"
				} else {
					m.actionPending = "Closing issue…"
				}
				m.actionMsg = ""
				m.actionErr = nil
				return m, tea.Batch(m.spinner.Tick, func() tea.Msg {
					var err error
					var done string
					if strings.EqualFold(issue.State, "closed") {
						err = github.ReopenIssue(issue.Number)
						done = "Issue reopened."
					} else {
						err = github.CloseIssue(issue.Number)
						done = "Issue closed."
					}
					if err != nil {
						return issueActionErrMsg{err: err}
					}
					return issueActionDoneMsg{message: done, number: issue.Number}
				})
			case "a":
				m.actionPending = "Assigning to @me…"
				m.actionMsg = ""
				m.actionErr = nil
				issue := m.detailIssue
				return m, tea.Batch(m.spinner.Tick, func() tea.Msg {
					if err := github.AssignIssueSelf(issue.Number); err != nil {
						return issueActionErrMsg{err: err}
					}
					return issueActionDoneMsg{message: "Assigned to @me.", number: issue.Number}
				})
			case "l":
				// Add label — embedded input form.
				m.formVals.labelVal = ""
				m.formVals.formType = issueFormLabel
				m.activeForm = huh.NewForm(huh.NewGroup(
					huh.NewInput().
						Title("Add label").
						Placeholder("label name").
						Value(&m.formVals.labelVal),
				)).WithTheme(huh.ThemeCatppuccin()).WithWidth(m.width - 8)
				return m, m.activeForm.Init()
			case "d":
				// Develop branch — embedded input form pre-filled with suggested name.
				defaultBranch := deriveBranchName(m.detailIssue.Number, m.detailIssue.Title)
				m.formVals.branchDefault = defaultBranch
				m.formVals.branchVal = defaultBranch
				m.formVals.formType = issueFormBranch
				m.activeForm = huh.NewForm(huh.NewGroup(
					huh.NewInput().
						Title("Branch name").
						Description(fmt.Sprintf("Default: %s", defaultBranch)).
						Value(&m.formVals.branchVal),
				)).WithTheme(huh.ThemeCatppuccin()).WithWidth(m.width - 8)
				return m, m.activeForm.Init()
			case "p":
				// Create PR from current branch using --fill (no user input needed).
				m.actionPending = "Creating PR…"
				m.actionMsg = ""
				m.actionErr = nil
				return m, tea.Batch(m.spinner.Tick, func() tea.Msg {
					if err := github.CreatePRFill(); err != nil {
						return issueActionErrMsg{err: err}
					}
					return issueActionDoneMsg{message: "PR created."}
				})
			}
			// Viewport scrolling
			var cmd tea.Cmd
			m.detail, cmd = m.detail.Update(msg)
			cmds = append(cmds, cmd)
			return m, tea.Batch(cmds...)
		}

		// List view keys
		switch msg.String() {
		case "n":
			if !m.list.SettingFilter() {
				// New issue — embedded form.
				m.formVals.newTitle = ""
				m.formVals.newBody = ""
				m.formVals.formType = issueFormNew
				m.activeForm = huh.NewForm(huh.NewGroup(
					huh.NewInput().
						Title("New Issue — Title").
						Placeholder("Short description").
						Value(&m.formVals.newTitle),
					huh.NewText().
						Title("Body").
						Placeholder("Describe the issue (optional)").
						CharLimit(4000).
						Value(&m.formVals.newBody),
				)).WithTheme(huh.ThemeCatppuccin()).WithWidth(m.width - 8)
				return m, m.activeForm.Init()
			}
		case "enter":
			if item, ok := m.list.SelectedItem().(issueListItem); ok {
				m.loadingDetail = true
				cmds = append(cmds, fetchIssueDetailCmd(item.issue.Number))
				cmds = append(cmds, m.spinner.Tick)
			}
		case "r":
			if !m.list.SettingFilter() {
				m.loading = true
				m.loaded = false
				cmds = append(cmds, m.fetchCmd())
				cmds = append(cmds, m.spinner.Tick)
			}
		case "q":
			if !m.list.SettingFilter() {
				m.action = "quit"
				return m, nil
			}
		case "tab":
			if !m.list.SettingFilter() {
				m.action = "switch"
				return m, nil
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

func (m IssuesModel) View() string {
	// When a form is active, render it — replacing list/detail content.
	if m.activeForm != nil {
		return m.activeForm.View()
	}

	var b strings.Builder

	if m.loading || m.loadingDetail {
		msg := "Fetching issues..."
		if m.loadingDetail {
			msg = fmt.Sprintf("Loading issue #%d...", func() int {
				if item, ok := m.list.SelectedItem().(issueListItem); ok {
					return item.issue.Number
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
		b.WriteString(renderIssueMetaStrip(m.detailIssue, m.width-4))
		b.WriteString(renderIssueDetailView(m.detailIssue, m.detail, m.actionMsg, m.actionErr))
		return b.String()
	}

	b.WriteString(lipgloss.NewStyle().Margin(0, 2).Render(m.list.View()))
	return b.String()
}

// headerHeight returns the number of lines used by headerView() with filter bar.
func headerHeight() int { return headerHeightFull }

// renderIssueDetailContent builds scrollable body-only content for the viewport.
func renderIssueDetailContent(issue github.Issue, _ int) string {
	var b strings.Builder
	if issue.Body != "" {
		b.WriteString(issue.Body + "\n")
	} else {
		b.WriteString(styleGray.Render("No description.") + "\n")
	}
	return b.String()
}

// renderIssueDetailView renders the scrollable viewport only.
// Action feedback (toast) is shown in the footer bar by AppModel so it never
// changes the body height.
func renderIssueDetailView(_ github.Issue, vp viewport.Model, _ string, _ error) string {
	return lipgloss.NewStyle().Margin(0, 2).Render(vp.View()) + "\n"
}
