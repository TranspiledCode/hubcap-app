// model_issues.go
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

// ── Issue action messages ─────────────────────────────────────────────────────

type issueActionDoneMsg struct {
	message       string
	number        int  // issue to re-fetch in detail view (0 = don't re-fetch)
	reloadList    bool // full list reload with loading indicator
	silentRefresh bool // quietly refresh list in background (no loading state)
}

type issueActionErrMsg struct {
	err error
}

// issueContentRenderedMsg is sent by renderIssueContentCmd when the viewport
// body has been rendered off the Update loop.
type issueContentRenderedMsg struct {
	content string
}

// renderIssueContentCmd renders issue detail content in a goroutine so the
// heavy glamour call never blocks the BubbleTea Update loop.
func renderIssueContentCmd(issue github.Issue, width int, pal Palette) tea.Cmd {
	return func() tea.Msg {
		return issueContentRenderedMsg{content: renderIssueDetailContent(issue, width, pal)}
	}
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

// prefetchNextItemsCmd fires background detail fetches for the next 2 items
// after the current cursor position. Results are tagged with the current
// prefetchGen so stale results from earlier cursor positions are discarded.
func (m IssuesModel) prefetchNextItemsCmd() tea.Cmd {
	items := m.list.Items()
	curIdx := m.list.Index()
	gen := m.prefetchGen

	var cmds []tea.Cmd
	for i := curIdx + 1; i <= curIdx+2 && i < len(items); i++ {
		item, ok := items[i].(issueListItem)
		if !ok {
			continue
		}
		// Skip items already in the prefetch cache.
		if _, cached := m.prefetchedDetails[item.issue.Number]; cached {
			continue
		}
		num := item.issue.Number
		cmds = append(cmds, func() tea.Msg {
			issue, err := github.FetchIssue(num)
			if err != nil {
				return nil // swallow silently — prefetch is best-effort
			}
			return issuePrefetchedMsg{number: num, issue: issue, gen: gen}
		})
	}
	return tea.Batch(cmds...)
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
	return fmt.Sprintf("%s  %s", joinUsers(i.issue.Assignees), joinLabels(i.issue.Labels))
}
func (i issueListItem) FilterValue() string {
	return fmt.Sprintf("%d %s", i.issue.Number, i.issue.Title)
}

// ── issueDelegate ─────────────────────────────────────────────────────────────
// Two-line delegate: Height=2, Spacing=0.
// Layout per row:
//
//	Line 1: [accent] ● #N   Title…(fill)
//	Line 2:         Assignee  Labels…

type issueDelegate struct{ pal Palette }

func (d issueDelegate) Height() int                             { return 3 }
func (d issueDelegate) Spacing() int                            { return 0 }
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
	selectedBg := d.pal.BgSelected
	if selected {
		base = lipgloss.NewStyle().Background(selectedBg)
	} else {
		base = lipgloss.NewStyle().Background(d.pal.BgBody)
	}

	// Left accent bar (2 chars wide either way so layout stays stable).
	var accent string
	if selected {
		accent = lipgloss.NewStyle().Foreground(d.pal.Accent).Background(selectedBg).Render("▌") +
			base.Render(" ")
	} else {
		accent = base.Render("  ")
	}

	// ⚑ colored by state: green = open, red = closed.
	dotColor := d.pal.StatusOpen
	if !strings.EqualFold(issue.State, "open") {
		dotColor = d.pal.StatusClosed
	}
	dot := base.Foreground(dotColor).Bold(true).Render("⚑")

	// Issue number — left-aligned in a 4-digit-wide field so columns stay
	// stable as numbers grow: " #20  " → " #200 " → " #2000".
	numStyle := base.Foreground(d.pal.Number)
	if selected {
		numStyle = numStyle.Bold(true)
	}
	numStr := numStyle.Render(fmt.Sprintf(" #%-4d", issue.Number))

	// Timestamp — rendered first so we know the width before sizing the title.
	var tsStr string
	if age := timeAgo(issue.CreatedAt); age != "" {
		tsStr = base.Foreground(d.pal.TextDim).Render(age)
	}
	tsW := lipgloss.Width(tsStr)

	// Title fills the space between the left prefix and the timestamp.
	// left prefix = accent(2) + dot(1) + numStr(6) + space(1) = 10
	// right suffix = gap(2) + tsW + trailing(1)
	titleStyle := base.Foreground(d.pal.Text)
	if selected {
		titleStyle = base.Foreground(d.pal.TextBold).Bold(true)
	}
	titleMaxW := width - 10 - 2 - tsW - 1
	if titleMaxW < 20 {
		titleMaxW = 20
	}
	titleStr := titleStyle.Render(truncate(issue.Title, titleMaxW))

	// Fill so the timestamp sits flush at the right edge.
	fillW := width - 10 - lipgloss.Width(titleStr) - tsW - 1
	if fillW < 1 {
		fillW = 1
	}
	fill := base.Render(strings.Repeat(" ", fillW))

	// Line 1: accent + dot + number + title + fill + timestamp
	line1 := accent + dot + numStr + base.Render(" ") + titleStr + fill + tsStr + base.Render(" ")

	// Line 2: assignee · labels … [fill] … type
	// indent = accent(2) + dot(1) + numStr(6) + space(1) = 10
	lineIndent := 10

	// Build the type badge first so we know its width before sizing labels.
	var typeStr string
	if issue.IssueType != "" {
		typeStr = base.Foreground(d.pal.Number).Italic(true).Render(issue.IssueType)
	} else {
		typeStr = base.Foreground(d.pal.TextFaint).Italic(true).Render("—")
	}
	typeW := lipgloss.Width(typeStr)

	// contentW is available for assignee + sep + labels, leaving room for
	// a minimum 2-char gap before the type and 1 trailing space.
	const typeGap = 2
	contentW := width - lineIndent - typeGap - typeW - 1
	if contentW < 20 {
		contentW = 20
	}

	const sepW = 5 // "  ·  "
	assigneeMax := (contentW - sepW) * 30 / 100
	if assigneeMax < 8 {
		assigneeMax = 8
	}
	labelMax := contentW - assigneeMax - sepW

	// Reuse accent on line 2 so the bar spans both rows; fill the rest of the indent.
	indent := accent + base.Render(strings.Repeat(" ", lineIndent-2))

	assigneeStyle := base.Foreground(d.pal.TextMuted).Italic(true)
	var assigneeText string
	if len(issue.Assignees) > 0 {
		assigneeText = "@" + joinUsers(issue.Assignees)
	} else {
		assigneeText = "unassigned"
	}
	assigneeStr := assigneeStyle.Render(truncate(assigneeText, assigneeMax))

	// Labels: show up to 3 badges, then a dim "+N" overflow count.
	const maxLabels = 3
	var labelPart string
	if len(issue.Labels) > 0 {
		shown := issue.Labels
		overflow := 0
		if len(issue.Labels) > maxLabels {
			shown = issue.Labels[:maxLabels]
			overflow = len(issue.Labels) - maxLabels
		}
		bgKey := string(d.pal.BgBody)
		if selected {
			bgKey = string(d.pal.BgSelected)
		}
		labelPart = issueRowLabels(shown, bgKey, labelMax, d.pal)
		if overflow > 0 {
			labelPart += base.Foreground(d.pal.TextDim).Render(fmt.Sprintf(" +%d", overflow))
		}
	}

	// Fill to push type to the right edge.
	line2LeftW := lineIndent + lipgloss.Width(assigneeStr)
	if labelPart != "" {
		line2LeftW += sepW + lipgloss.Width(labelPart)
	}
	line2FillW := width - line2LeftW - typeW - 1
	if line2FillW < typeGap {
		line2FillW = typeGap
	}
	line2Fill := base.Render(strings.Repeat(" ", line2FillW))

	dimSep := base.Foreground(d.pal.TextFaint).Render("  ·  ")
	line2 := indent + assigneeStr
	if labelPart != "" {
		line2 += dimSep + labelPart
	}
	line2 += line2Fill + typeStr + base.Render(" ")
	spacer := lipgloss.NewStyle().Background(d.pal.BgBody).Width(width).Render("")
	fmt.Fprintf(w, "%s\n%s\n%s", line1, line2, spacer)
}

// issueRowLabels renders a short colored label string for a list row.
// bgKey is "" for no background or the color key (e.g. "235") when the row
// is selected — so each segment explicitly matches the row background.
func issueRowLabels(labels []github.Label, bgKey string, maxW int, pal Palette) string {
	if len(labels) == 0 {
		return ""
	}
	makeBase := func() lipgloss.Style {
		if bgKey != "" {
			return lipgloss.NewStyle().Background(lipgloss.Color(bgKey))
		}
		return lipgloss.NewStyle()
	}
	sep := makeBase().Foreground(pal.TextFaint).Render(" · ")
	sepW := lipgloss.Width(sep)

	var parts []string
	used := 0
	for _, l := range labels {
		ls := labelStyle(l.Name, pal)
		if bgKey != "" {
			ls = ls.Background(lipgloss.Color(bgKey))
		}
		rendered := ls.Italic(true).Render(l.Name)
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
	metaExpanded  bool // true = show expanded metadata strip

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

	// uiTheme mirrors Config.UITheme and controls form width + footer density.
	uiTheme UITheme

	// palette mirrors Config.ColorTheme for list item and detail colours.
	palette Palette

	// currentUser is the authenticated GitHub login, used to distinguish
	// Grab / Take / Drop when the user presses 'a' on the issue list.
	currentUser string

	// Detail prefetch: while the user navigates the list, the next 2 items
	// are fetched in the background and stored here. When the user presses
	// Enter, the cached detail is used immediately — no loading spinner.
	// prefetchGen is incremented on each cursor move; stale goroutine results
	// are discarded when their gen no longer matches.
	prefetchGen       int
	prefetchedDetails map[int]github.Issue
}

func newIssuesModel(filters github.Filters, pal Palette) IssuesModel {
	s := spinner.New()
	s.Spinner = spinner.Dot
	s.Style = lipgloss.NewStyle().Foreground(pal.Accent)

	l := list.New([]list.Item{}, issueDelegate{pal: pal}, 0, 0)
	l.SetShowTitle(false)
	l.SetShowStatusBar(false)
	l.SetShowHelp(false)
	l.SetFilteringEnabled(true)
	l.Styles.NoItems = l.Styles.NoItems.Background(pal.BgBody).Foreground(pal.TextDim)
	l.Styles.FilterPrompt = l.Styles.FilterPrompt.Background(pal.BgBody).Foreground(pal.TextMuted)
	l.Styles.FilterCursor = l.Styles.FilterCursor.Background(pal.BgBody).Foreground(pal.Accent)
	// Disable the list's built-in quit keybindings (q and esc) so they don't
	// call tea.Quit directly and bypass our confirmation prompt.
	// Must use DisableQuitKeybindings() — not SetEnabled(false) — because the
	// list re-enables Quit via SetEnabled(!disableQuitKeybindings) every time
	// items load or filter state changes.
	l.DisableQuitKeybindings()

	// Align the list's navigation key map with our central registry so that
	// j/k/g/G work consistently and the source of truth is always keys.go.
	l.KeyMap.CursorUp = keys.Up
	l.KeyMap.CursorDown = keys.Down
	l.KeyMap.GoToStart = keys.Top
	l.KeyMap.GoToEnd = keys.Bottom

	return IssuesModel{
		list:     l,
		spinner:  s,
		loading:  true,
		filters:  filters,
		palette:  pal,
		formVals: &issueFormVals{},
	}
}

// IsFiltering reports whether the list's inline filter is currently active.
// AppModel uses this to suppress global shortcuts (e.g. tab/q) while typing.
func (m IssuesModel) IsFiltering() bool { return m.list.SettingFilter() }

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
	case issuePrefetchedMsg:
		// Discard results from stale cursor positions.
		if msg.gen == m.prefetchGen {
			if m.prefetchedDetails == nil {
				m.prefetchedDetails = make(map[int]github.Issue)
			}
			m.prefetchedDetails[msg.number] = msg.issue
		}
		return m, nil

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
		// New list data invalidates prefetched details.
		m.prefetchedDetails = make(map[int]github.Issue)
		m.prefetchGen++

	case issueFetchedMsg:
		m.loadingDetail = false
		if msg.err != nil {
			m.err = msg.err
			return m, nil
		}
		m.detailIssue = msg.issue
		m.detail = viewport.New(m.width-4, m.height-headerHeightDetail-m.currentMetaHeight()-2)
		m.detail.Style = lipgloss.NewStyle().Background(m.palette.BgBody)
		m.detail.SetContent(lipgloss.NewStyle().Foreground(m.palette.TextDim).Background(m.palette.BgBody).Render("Rendering…") + "\n")
		m.showDetail = true
		return m, renderIssueContentCmd(msg.issue, m.width, m.palette)

	case issueContentRenderedMsg:
		m.detail.SetContent(msg.content)
		return m, nil

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
		if msg.silentRefresh {
			return m, tea.Batch(clearIssueActionMsgCmd(), m.silentFetchCmd())
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
		// Subtract the app border (1 char each side) so m.width/m.height
		// reflect the usable content area, matching AppModel's innerW/innerH.
		m.width = msg.Width - 2
		m.height = msg.Height - 2
		m.list.SetSize(m.width-4, m.height-headerHeight()-2)
		if m.showDetail {
			m.detail.Width = m.width - 4
			m.detail.Height = m.height - headerHeightDetail - m.currentMetaHeight() - 2
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
				m.metaExpanded = false
				m.actionMsg = ""
				m.actionErr = nil
				return m, nil
			case key.Matches(msg, keys.IssueExpandMeta):
				m.metaExpanded = !m.metaExpanded
				m.detail.Height = m.height - headerHeightDetail - m.currentMetaHeight() - 2
				return m, nil
			case key.Matches(msg, keys.Refresh):
				m.loadingDetail = true
				m.actionMsg = ""
				m.actionErr = nil
				return m, fetchIssueDetailCmd(m.detailIssue.Number)
			case key.Matches(msg, keys.Browser):
				// Open in browser — instant, no pending indicator needed.
				url := m.detailIssue.URL
				return m, func() tea.Msg {
					github.OpenURL(url)
					return nil
				}
			case key.Matches(msg, keys.CopyURL):
				m.actionPending = ""
				if err := copyText(m.detailIssue.URL); err != nil {
					m.actionErr = err
					m.actionMsg = ""
				} else {
					m.actionMsg = "URL copied to clipboard."
					m.actionErr = nil
				}
				return m, clearIssueActionMsgCmd()
			case key.Matches(msg, keys.IssueClose):
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
			case key.Matches(msg, keys.IssueAssign):
				issue := m.detailIssue
				// Assign/unassign is not allowed on a closed issue.
				if strings.EqualFold(issue.State, "closed") {
					return m, nil
				}
				if len(issue.Assignees) > 0 {
					m.actionPending = "Unassigning from @me…"
					m.actionMsg = ""
					m.actionErr = nil
					return m, tea.Batch(m.spinner.Tick, func() tea.Msg {
						if err := github.UnassignIssueSelf(issue.Number); err != nil {
							return issueActionErrMsg{err: err}
						}
						return issueActionDoneMsg{message: "Unassigned from @me.", number: issue.Number}
					})
				}
				m.actionPending = "Assigning to @me…"
				m.actionMsg = ""
				m.actionErr = nil
				return m, tea.Batch(m.spinner.Tick, func() tea.Msg {
					if err := github.AssignIssueSelf(issue.Number); err != nil {
						return issueActionErrMsg{err: err}
					}
					return issueActionDoneMsg{message: "Assigned to @me.", number: issue.Number}
				})
			case key.Matches(msg, keys.IssueLabel):
				// Add label — embedded input form.
				m.formVals.labelVal = ""
				m.formVals.formType = issueFormLabel
				m.activeForm = huh.NewForm(huh.NewGroup(
					huh.NewInput().
						Title("Add label").
						Placeholder("label name").
						Value(&m.formVals.labelVal),
				)).WithTheme(huh.ThemeCatppuccin()).WithWidth(formWidth(m.width, m.uiTheme))
				return m, m.activeForm.Init()
			case key.Matches(msg, keys.IssueDevelop):
				// Developing a branch is not allowed on a closed issue.
				if strings.EqualFold(m.detailIssue.State, "closed") {
					return m, nil
				}
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
				)).WithTheme(huh.ThemeCatppuccin()).WithWidth(formWidth(m.width, m.uiTheme))
				return m, m.activeForm.Init()
			case key.Matches(msg, keys.IssuePR):
				// Creating a PR is not allowed on a closed issue.
				if strings.EqualFold(m.detailIssue.State, "closed") {
					return m, nil
				}
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
			case key.Matches(msg, keys.Top):
				m.detail.GotoTop()
				return m, nil
			case key.Matches(msg, keys.Bottom):
				m.detail.GotoBottom()
				return m, nil
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
			case key.Matches(msg, keys.Browser):
				if item, ok := m.list.SelectedItem().(issueListItem); ok {
					url := item.issue.URL
					return m, func() tea.Msg { github.OpenURL(url); return nil }
				}
			case key.Matches(msg, keys.IssueAssign):
				if item, ok := m.list.SelectedItem().(issueListItem); ok {
					issue := item.issue
					if isMeAssigned(issue.Assignees, m.currentUser) {
						// Drop — remove @me, leave any other assignees.
						m.actionPending = "Dropping…"
						m.actionMsg = ""
						m.actionErr = nil
						return m, tea.Batch(m.spinner.Tick, func() tea.Msg {
							if err := github.UnassignIssueSelf(issue.Number); err != nil {
								return issueActionErrMsg{err: err}
							}
							return issueActionDoneMsg{message: "Dropped.", silentRefresh: true}
						})
					}
					// Grab (unassigned) or Take (assigned to someone else).
					verb := "Grabbing…"
					done := "Grabbed."
					if len(issue.Assignees) > 0 {
						verb = "Taking…"
						done = "Taken."
					}
					m.actionPending = verb
					m.actionMsg = ""
					m.actionErr = nil
					doneMsg := done
					return m, tea.Batch(m.spinner.Tick, func() tea.Msg {
						if err := github.AssignIssueSelf(issue.Number); err != nil {
							return issueActionErrMsg{err: err}
						}
						return issueActionDoneMsg{message: doneMsg, silentRefresh: true}
					})
				}
			case key.Matches(msg, keys.New):
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
				)).WithTheme(huh.ThemeCatppuccin()).WithWidth(formWidth(m.width, m.uiTheme))
				return m, m.activeForm.Init()
			case key.Matches(msg, keys.Refresh):
				m.loading = true
				m.loaded = false
				cmds = append(cmds, m.fetchCmd())
				cmds = append(cmds, m.spinner.Tick)
			}
		}
		if key.Matches(msg, keys.Open) {
			if item, ok := m.list.SelectedItem().(issueListItem); ok {
				// Use prefetched detail immediately if available — no spinner needed.
				if prefetched, ok := m.prefetchedDetails[item.issue.Number]; ok {
					m.detailIssue = prefetched
					m.detail = viewport.New(m.width-4, m.height-headerHeightDetail-m.currentMetaHeight()-2)
					m.detail.Style = lipgloss.NewStyle().Background(m.palette.BgBody)
					m.detail.SetContent(lipgloss.NewStyle().Foreground(m.palette.TextDim).Background(m.palette.BgBody).Render("Rendering…") + "\n")
					m.showDetail = true
					delete(m.prefetchedDetails, item.issue.Number)
					cmds = append(cmds, renderIssueContentCmd(prefetched, m.width, m.palette))
				} else {
					m.loadingDetail = true
					cmds = append(cmds, fetchIssueDetailCmd(item.issue.Number))
					cmds = append(cmds, m.spinner.Tick)
				}
			}
		}
	}

	if !m.loading && !m.loadingDetail && !m.showDetail {
		prevIdx := m.list.Index()
		var cmd tea.Cmd
		m.list, cmd = m.list.Update(msg)
		cmds = append(cmds, cmd)
		// When cursor moves, kick off background prefetch for the next 2 items.
		if m.list.Index() != prevIdx {
			if m.prefetchedDetails == nil {
				m.prefetchedDetails = make(map[int]github.Issue)
			}
			m.prefetchGen++
			cmds = append(cmds, m.prefetchNextItemsCmd())
		}
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
		b.WriteString(errorBox(fmt.Sprintf("Error: %v\n\nPress r to retry.", m.err), m.palette))
		return b.String()
	}

	if m.showDetail {
		b.WriteString(renderIssueMetaStrip(m.detailIssue, m.width-4, m.metaExpanded, m.palette))
		b.WriteString(renderIssueDetailView(m.detailIssue, m.detail, m.actionMsg, m.actionErr, m.palette))
		return b.String()
	}

	b.WriteString(lipgloss.NewStyle().Padding(0, 2).Background(m.palette.BgBody).Render(m.list.View()))
	return b.String()
}

// headerHeight returns the number of lines used by headerView() with filter bar.
func headerHeight() int { return headerHeightFull }

// currentMetaHeight returns the meta strip height based on the expansion state.
func (m IssuesModel) currentMetaHeight() int {
	if m.metaExpanded {
		return metaStripExpandedHeight
	}
	return metaStripHeight
}

// renderIssueDetailContent builds scrollable body-only content for the viewport.
func renderIssueDetailContent(issue github.Issue, width int, pal Palette) string {
	if issue.Body == "" {
		return lipgloss.NewStyle().Foreground(pal.TextDim).Background(pal.BgBody).Render("No description.") + "\n"
	}
	glamourStyle := "auto"
	if pal.BgBody != "" {
		glamourStyle = "light"
	}
	return renderMarkdown(issue.Body, width-4, glamourStyle)
}

// renderIssueDetailView renders the scrollable viewport only.
// Action feedback (toast) is shown in the footer bar by AppModel so it never
// changes the body height.
func renderIssueDetailView(_ github.Issue, vp viewport.Model, _ string, _ error, pal Palette) string {
	return lipgloss.NewStyle().Padding(0, 2).Background(pal.BgBody).Render(viewportWithScrollHint(vp, pal)) + "\n"
}
