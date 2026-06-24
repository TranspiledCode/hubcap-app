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
	message       string
	number        int  // PR to re-fetch after action (0 = don't re-fetch)
	reloadList    bool // refresh the full list (e.g. after creating a PR)
	silentRefresh bool // true → background-refresh the list without loading state
}

type prActionErrMsg struct {
	err error
}

// prContentRenderedMsg is sent by renderPRContentCmd when the viewport body
// has been rendered off the Update loop.
type prContentRenderedMsg struct {
	content string
}

// renderPRContentCmd renders PR detail content in a goroutine so the heavy
// glamour call never blocks the BubbleTea Update loop.
func renderPRContentCmd(pr github.PullRequest, width int, pal Palette) tea.Cmd {
	return func() tea.Msg {
		return prContentRenderedMsg{content: renderPRDetailContent(pr, width, pal)}
	}
}

// clearPRActionMsgMsg is sent by a timer to dismiss the toast notification.
type clearPRActionMsgMsg struct{}

// reviewerDataFetchedMsg carries the collaborator list fetched before showing
// the reviewer select form.
type reviewerDataFetchedMsg struct {
	reviewers []string
	err       error
}

// fetchReviewerDataCmd fetches the project's collaborator list so the reviewer
// select form can be populated with real names.
func fetchReviewerDataCmd() tea.Cmd {
	return func() tea.Msg {
		reviewers, err := github.FetchAssignees()
		return reviewerDataFetchedMsg{reviewers: reviewers, err: err}
	}
}

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

// prefetchNextItemsCmd fires background detail fetches for the next 2 items
// after the current cursor position. Results are tagged with the current
// prefetchGen so stale results from earlier cursor positions are discarded.
func (m PRsModel) prefetchNextItemsCmd() tea.Cmd {
	items := m.list.Items()
	curIdx := m.list.Index()
	gen := m.prefetchGen

	var cmds []tea.Cmd
	for i := curIdx + 1; i <= curIdx+2 && i < len(items); i++ {
		item, ok := items[i].(prListItem)
		if !ok {
			continue
		}
		// Skip items already in the prefetch cache.
		if _, cached := m.prefetchedDetails[item.pr.Number]; cached {
			continue
		}
		num := item.pr.Number
		cmds = append(cmds, func() tea.Msg {
			pr, err := github.FetchPR(num)
			if err != nil {
				return nil // swallow silently — prefetch is best-effort
			}
			return prPrefetchedMsg{number: num, pr: pr, gen: gen}
		})
	}
	return tea.Batch(cmds...)
}

// ── prFormType ────────────────────────────────────────────────────────────────

type prFormType int

const (
	prFormNone   prFormType = iota
	prFormMerge             // "m" — choose merge strategy
	prFormNew               // "n" — create a new PR
	prFormReview            // "v" — request a reviewer
	prFormAssign            // "a" — assign to a specific user
)

// prFormVals is heap-allocated so huh's Value() pointers remain valid across
// BubbleTea value-receiver model copies.
type prFormVals struct {
	formType          prFormType
	mergeType         string
	newTitle          string
	newBody           string
	newBase           string
	newDraft          bool
	reviewerVals      []string // selected reviewer logins (multi-select)
	originalReviewers []string // reviewers when the form was opened (for diffing)
	assigneeVals      []string // selected assignee logins (multi-select)
	originalAssignees []string // assignees when the form was opened (for diffing)
}

// prAssigneeDataFetchedMsg carries the collaborator list for the PR assignee form.
type prAssigneeDataFetchedMsg struct {
	assignees []string
	err       error
}

func fetchPRAssigneeDataCmd() tea.Cmd {
	return func() tea.Msg {
		assignees, err := github.FetchAssignees()
		return prAssigneeDataFetchedMsg{assignees: assignees, err: err}
	}
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
	return fmt.Sprintf("%s  %s", p.pr.Author.Login, status)
}
func (p prListItem) FilterValue() string {
	return fmt.Sprintf("%d %s", p.pr.Number, p.pr.Title)
}

// ── prDelegate ────────────────────────────────────────────────────────────────
// Two-row delegate matching the Issues list style: Height=2, Spacing=1.
//
// Line 1: [accent] ⤴ #N   Title…(fill)…  timestamp
// Line 2: [accent]         @author  ·  checks  ·  labels  (fill)  head → base

type prDelegate struct{ pal Palette }

func (d prDelegate) Height() int                             { return 3 }
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

	selectedBg := d.pal.BgSelected
	var base lipgloss.Style
	if selected {
		base = lipgloss.NewStyle().Background(selectedBg)
	} else {
		base = lipgloss.NewStyle().Background(d.pal.BgBody)
	}

	// Left accent bar — reused on both rows so the bar spans the full item height.
	var accent string
	if selected {
		accent = lipgloss.NewStyle().Foreground(d.pal.Accent).Background(selectedBg).Render("▌") +
			base.Render(" ")
	} else {
		accent = base.Render("  ")
	}

	// ⤴ colored by state: green = open, purple = merged, red = closed, amber = draft.
	var dotColor lipgloss.Color
	switch {
	case pr.IsDraft:
		dotColor = d.pal.StatusDraft
	case strings.EqualFold(pr.State, "merged"):
		dotColor = d.pal.StatusMerged
	case strings.EqualFold(pr.State, "closed"):
		dotColor = d.pal.StatusClosed
	default:
		dotColor = d.pal.StatusOpen
	}
	dot := base.Foreground(dotColor).Bold(true).Render("⤴")

	// PR number — left-aligned in a 4-digit-wide field for stable columns.
	numStyle := base.Foreground(d.pal.Number)
	if selected {
		numStyle = numStyle.Bold(true)
	}
	numStr := numStyle.Render(fmt.Sprintf(" #%-4d", pr.Number))

	// Timestamp — rendered first so we know the width before sizing the title.
	var tsStr string
	if age := timeAgo(pr.CreatedAt); age != "" {
		tsStr = base.Foreground(d.pal.TextDim).Render(age)
	}
	tsW := lipgloss.Width(tsStr)

	// Title fills the space between the left prefix and the timestamp.
	// left prefix = accent(2) + dot(1) + numStr(6) + space(1) = 10
	titleStyle := base.Foreground(d.pal.Text)
	if selected {
		titleStyle = base.Foreground(d.pal.TextBold).Bold(true)
	}
	titleMaxW := width - 10 - 2 - tsW - 1
	if titleMaxW < 20 {
		titleMaxW = 20
	}
	titleStr := titleStyle.Render(truncate(pr.Title, titleMaxW))

	fillW := width - 10 - lipgloss.Width(titleStr) - tsW - 1
	if fillW < 1 {
		fillW = 1
	}
	fill := base.Render(strings.Repeat(" ", fillW))

	// Line 1: accent + dot + number + title + fill + timestamp
	line1 := accent + dot + numStr + base.Render(" ") + titleStr + fill + tsStr + base.Render(" ")

	// ── Line 2 ──────────────────────────────────────────────────────────────
	// indent = accent(2) + dot(1) + numStr(6) + space(1) = 10
	const lineIndent = 10
	bgKey := string(d.pal.BgBody)
	if selected {
		bgKey = string(d.pal.BgSelected)
	}

	authorStyle := base.Foreground(d.pal.TextMuted).Italic(true)
	arrowStyle := base.Foreground(d.pal.Text)

	// Branch direction acts as the right-side badge (like typeStr in issues).
	var branchStr string
	if pr.HeadRefName != "" && pr.BaseRefName != "" {
		branchStr = authorStyle.Render(truncate(pr.HeadRefName, 16)) +
			arrowStyle.Render(" → ") +
			authorStyle.Render(pr.BaseRefName)
	}
	branchW := lipgloss.Width(branchStr)

	checksStr := prRowChecks(pr.StatusRollup, bgKey, d.pal)
	checksW := lipgloss.Width(checksStr)

	const (
		sepW      = 5 // "  ·  "
		branchGap = 2
	)
	dimSep := base.Foreground(d.pal.TextFaint).Render("  ·  ")

	// Author takes ~30% of the content area; labels fill the rest.
	contentW := width - lineIndent - branchGap - branchW - 1
	if contentW < 20 {
		contentW = 20
	}
	authorMax := contentW * 30 / 100
	if authorMax < 8 {
		authorMax = 8
	}
	authorStr := authorStyle.Render(truncate("@"+pr.Author.Login, authorMax))
	authorActualW := lipgloss.Width(authorStr)

	labelBudget := contentW - authorActualW - sepW
	if checksStr != "" {
		labelBudget -= checksW + sepW
	}
	if labelBudget < 0 {
		labelBudget = 0
	}
	const maxLabels = 3
	shownLabels := pr.Labels
	labelOverflow := 0
	if len(pr.Labels) > maxLabels {
		shownLabels = pr.Labels[:maxLabels]
		labelOverflow = len(pr.Labels) - maxLabels
	}
	labelStr := issueRowLabels(shownLabels, bgKey, labelBudget, d.pal)
	if labelOverflow > 0 {
		labelStr += base.Foreground(d.pal.TextDim).Render(fmt.Sprintf(" +%d", labelOverflow))
	}

	// Reuse accent on line 2 so the bar spans both rows.
	indent2 := accent + base.Render(strings.Repeat(" ", lineIndent-2))

	line2LeftW := lineIndent + authorActualW
	if checksStr != "" {
		line2LeftW += sepW + checksW
	}
	if labelStr != "" {
		line2LeftW += sepW + lipgloss.Width(labelStr)
	}
	line2FillW := width - line2LeftW - branchW - 1
	if line2FillW < branchGap {
		line2FillW = branchGap
	}
	line2Fill := base.Render(strings.Repeat(" ", line2FillW))

	line2 := indent2 + authorStr
	if checksStr != "" {
		line2 += dimSep + checksStr
	}
	if labelStr != "" {
		line2 += dimSep + labelStr
	}
	line2 += line2Fill + branchStr + base.Render(" ")

	spacer := lipgloss.NewStyle().Background(d.pal.BgBody).Width(width).Render("")
	fmt.Fprintf(w, "%s\n%s\n%s", line1, line2, spacer)
}

// prRowChecks returns a compact colored check-status symbol for a list row.
func prRowChecks(checks []github.CheckRun, bgKey string, pal Palette) string {
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
			return base.Foreground(pal.CheckFail).Italic(true).Render("✗ failing")
		}
		if c.Status != "COMPLETED" {
			pending = true
		}
	}
	if pending {
		return base.Foreground(pal.CheckPending).Italic(true).Render("… pending")
	}
	return base.Foreground(pal.CheckPass).Italic(true).Render("✓ passing")
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

	// Inline form (merge strategy, new PR, reviewer). activeForm is nil when closed.
	// formVals is heap-allocated for stable pointers across model copies.
	activeForm            *huh.Form
	formVals              *prFormVals
	loadingReviewerForm   bool // true while fetching collaborators for reviewer select
	loadingPRAssigneeForm bool // true while fetching collaborators for assignee select

	// uiTheme mirrors Config.UITheme and controls form width + footer density.
	uiTheme UITheme

	// palette mirrors Config.ColorTheme for list item and detail colours.
	palette Palette

	// currentUser is the authenticated GitHub login, used to distinguish
	// Grab / Take / Drop when the user presses 'a' on the PR list.
	currentUser string

	// Detail prefetch: while the user navigates the list, the next 2 items
	// are fetched in the background and stored here. When the user presses
	// Enter, the cached detail is used immediately — no loading spinner.
	// prefetchGen is incremented on each cursor move; stale goroutine results
	// are discarded when their gen no longer matches.
	prefetchGen       int
	prefetchedDetails map[int]github.PullRequest
}

func newPRsModel(filters github.PRFilters, pal Palette) PRsModel {
	s := spinner.New()
	s.Spinner = spinner.Dot
	s.Style = lipgloss.NewStyle().Foreground(pal.Accent)

	l := list.New([]list.Item{}, prDelegate{pal: pal}, 0, 0)
	l.SetShowTitle(false)
	l.SetShowStatusBar(false)
	l.SetShowHelp(false)
	l.SetFilteringEnabled(true)
	l.Styles.NoItems = l.Styles.NoItems.Background(pal.BgBody).Foreground(pal.TextDim)
	l.Styles.FilterPrompt = l.Styles.FilterPrompt.Background(pal.BgBody).Foreground(pal.TextMuted)
	l.Styles.FilterCursor = l.Styles.FilterCursor.Background(pal.BgBody).Foreground(pal.Accent)
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
		palette:  pal,
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

	case prFormReview:
		pr := m.detailPR

		// Diff: compute reviewers to add and remove.
		newSet := make(map[string]bool, len(m.formVals.reviewerVals))
		for _, l := range m.formVals.reviewerVals {
			newSet[l] = true
		}
		oldSet := make(map[string]bool, len(m.formVals.originalReviewers))
		for _, l := range m.formVals.originalReviewers {
			oldSet[l] = true
		}
		var add, remove []string
		for l := range newSet {
			if !oldSet[l] {
				add = append(add, l)
			}
		}
		for l := range oldSet {
			if !newSet[l] {
				remove = append(remove, l)
			}
		}
		if len(add) == 0 && len(remove) == 0 {
			return m, nil
		}

		var done string
		switch {
		case len(add) > 0 && len(remove) > 0:
			done = fmt.Sprintf("Reviewers updated (+%d / -%d).", len(add), len(remove))
		case len(add) > 0:
			done = fmt.Sprintf("Review requested from @%s.", strings.Join(add, ", @"))
		default:
			done = fmt.Sprintf("Removed reviewer @%s.", strings.Join(remove, ", @"))
		}

		m.actionPending = "Updating reviewers…"
		m.actionMsg = ""
		m.actionErr = nil
		num := pr.Number
		return m, tea.Batch(m.spinner.Tick, func() tea.Msg {
			if err := github.UpdatePRReviewers(num, add, remove); err != nil {
				return prActionErrMsg{err: err}
			}
			return prActionDoneMsg{message: done, number: num}
		})

	case prFormAssign:
		pr := m.detailPR
		if pr.Number == 0 {
			if item, ok := m.list.SelectedItem().(prListItem); ok {
				pr = item.pr
			}
		}
		if pr.Number == 0 {
			return m, nil
		}

		newSet := make(map[string]bool, len(m.formVals.assigneeVals))
		for _, l := range m.formVals.assigneeVals {
			newSet[l] = true
		}
		oldSet := make(map[string]bool, len(m.formVals.originalAssignees))
		for _, l := range m.formVals.originalAssignees {
			oldSet[l] = true
		}
		var add, remove []string
		for l := range newSet {
			if !oldSet[l] {
				add = append(add, l)
			}
		}
		for l := range oldSet {
			if !newSet[l] {
				remove = append(remove, l)
			}
		}
		if len(add) == 0 && len(remove) == 0 {
			return m, nil
		}

		var done string
		switch {
		case len(add) > 0 && len(remove) > 0:
			done = fmt.Sprintf("Assignees updated (+%d / -%d).", len(add), len(remove))
		case len(add) > 0:
			done = fmt.Sprintf("Assigned @%s.", strings.Join(add, ", @"))
		default:
			done = fmt.Sprintf("Unassigned @%s.", strings.Join(remove, ", @"))
		}

		m.actionPending = "Updating assignees…"
		m.actionMsg = ""
		m.actionErr = nil
		num := pr.Number
		inDetail := m.showDetail
		return m, tea.Batch(m.spinner.Tick, func() tea.Msg {
			if err := github.UpdatePRAssignees(num, add, remove); err != nil {
				return prActionErrMsg{err: err}
			}
			return prActionDoneMsg{message: done, number: num, silentRefresh: !inDetail}
		})
	}

	return m, nil
}

// ── Update ────────────────────────────────────────────────────────────────────

func (m PRsModel) Update(msg tea.Msg) (PRsModel, tea.Cmd) {
	var cmds []tea.Cmd

	// ── Embedded form takes priority ──────────────────────────────────────────
	if m.activeForm != nil {
		// Esc / b / backspace cancels the form without submitting.
		if km, ok := msg.(tea.KeyMsg); ok && key.Matches(km, keys.Back) {
			m.activeForm = nil
			m.formVals.formType = prFormNone
			return m, nil
		}
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
	case prPrefetchedMsg:
		// Discard results from stale cursor positions.
		if msg.gen == m.prefetchGen {
			if m.prefetchedDetails == nil {
				m.prefetchedDetails = make(map[int]github.PullRequest)
			}
			m.prefetchedDetails[msg.number] = msg.pr
		}
		return m, nil

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
		// New list data invalidates prefetched details.
		m.prefetchedDetails = make(map[int]github.PullRequest)
		m.prefetchGen++

	case prFetchedMsg:
		m.loadingDetail = false
		if msg.err != nil {
			m.err = msg.err
			return m, nil
		}
		m.detailPR = msg.pr
		m.detail = viewport.New(m.width-4, detailViewportHeight(m.height, prMetaStripHeight, m.uiTheme))
		m.detail.Style = lipgloss.NewStyle().Background(m.palette.BgBody)
		m.detail.SetContent(lipgloss.NewStyle().Foreground(m.palette.TextDim).Background(m.palette.BgBody).Render("Rendering…") + "\n")
		m.showDetail = true
		return m, renderPRContentCmd(msg.pr, m.width, m.palette)

	case prContentRenderedMsg:
		m.detail.SetContent(msg.content)
		return m, nil

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
		if msg.silentRefresh {
			return m, tea.Batch(clearPRActionMsgCmd(), m.silentFetchCmd())
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

	case reviewerDataFetchedMsg:
		m.loadingReviewerForm = false

		// Pre-populate with the PR's current requested reviewers.
		var currentReviewers []string
		for _, u := range m.detailPR.RequestedReviewers {
			currentReviewers = append(currentReviewers, u.Login)
		}
		m.formVals.originalReviewers = currentReviewers
		m.formVals.reviewerVals = append([]string(nil), currentReviewers...)
		m.formVals.formType = prFormReview

		if msg.err != nil || len(msg.reviewers) == 0 {
			// Fallback: plain text input if fetch failed or repo has no collaborators.
			m.formVals.reviewerVals = nil
			m.activeForm = huh.NewForm(huh.NewGroup(
				huh.NewInput().
					Title("Request reviewer").
					Placeholder("GitHub username").
					Value(&m.formVals.newTitle), // reuse newTitle as scratch space
			)).WithTheme(buildHuhTheme(m.palette)).WithShowHelp(false).WithWidth(formWidth(m.width, m.uiTheme))
			return m, m.activeForm.Init()
		}

		// Build multi-select options. Pre-selection is driven by the initial
		// value of reviewerVals — do NOT use Selected(true) or huh will scroll
		// the cursor to the last checked item instead of starting at the top.
		opts := make([]huh.Option[string], len(msg.reviewers))
		for i, r := range msg.reviewers {
			opts[i] = huh.NewOption(r, r)
		}
		m.activeForm = huh.NewForm(huh.NewGroup(
			huh.NewMultiSelect[string]().
				Title("Reviewers").
				Description("Space to toggle, enter to confirm, esc to cancel").
				Options(opts...).
				Value(&m.formVals.reviewerVals),
		)).WithTheme(buildHuhTheme(m.palette)).WithShowHelp(false).WithWidth(formWidth(m.width, m.uiTheme))
		return m, m.activeForm.Init()

	case prAssigneeDataFetchedMsg:
		m.loadingPRAssigneeForm = false

		// Pre-populate with the PR's current assignees.
		var currentAssignees []string
		if m.showDetail {
			for _, u := range m.detailPR.Assignees {
				currentAssignees = append(currentAssignees, u.Login)
			}
		} else if item, ok := m.list.SelectedItem().(prListItem); ok {
			for _, u := range item.pr.Assignees {
				currentAssignees = append(currentAssignees, u.Login)
			}
		}
		m.formVals.originalAssignees = currentAssignees
		m.formVals.assigneeVals = append([]string(nil), currentAssignees...)
		m.formVals.formType = prFormAssign

		if msg.err != nil || len(msg.assignees) == 0 {
			m.formVals.assigneeVals = nil
			m.activeForm = huh.NewForm(huh.NewGroup(
				huh.NewInput().
					Title("Assign to").
					Placeholder("GitHub username").
					Value(&m.formVals.newBase), // reuse newBase as scratch space
			)).WithTheme(buildHuhTheme(m.palette)).WithShowHelp(false).WithWidth(formWidth(m.width, m.uiTheme))
			return m, m.activeForm.Init()
		}

		// Build multi-select options — no Selected(true) so cursor starts at top.
		opts := make([]huh.Option[string], len(msg.assignees))
		for i, a := range msg.assignees {
			opts[i] = huh.NewOption(a, a)
		}
		m.activeForm = huh.NewForm(huh.NewGroup(
			huh.NewMultiSelect[string]().
				Title("Assignees").
				Description("Space to toggle, enter to confirm, esc to cancel").
				Options(opts...).
				Value(&m.formVals.assigneeVals),
		)).WithTheme(buildHuhTheme(m.palette)).WithShowHelp(false).WithWidth(formWidth(m.width, m.uiTheme))
		return m, m.activeForm.Init()

	case tea.WindowSizeMsg:
		m.width = msg.Width - 2
		m.height = msg.Height - 2
		m.list.SetSize(m.width-4, m.height-headerHeight()-2)
		if m.showDetail {
			m.detail.Width = m.width - 4
			m.detail.Height = detailViewportHeight(m.height, prMetaStripHeight, m.uiTheme)
		}

	case spinner.TickMsg:
		// Keep spinning while loading or while a background action is in flight.
		if m.loading || m.loadingDetail || m.actionPending != "" || m.loadingReviewerForm || m.loadingPRAssigneeForm {
			var cmd tea.Cmd
			m.spinner, cmd = m.spinner.Update(msg)
			cmds = append(cmds, cmd)
		}

	case tea.KeyMsg:
		if m.loading || m.loadingDetail || m.loadingReviewerForm || m.loadingPRAssigneeForm {
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
				// Close or Reopen PR — merged PRs cannot be closed or reopened.
				pr := m.detailPR
				if strings.EqualFold(pr.State, "merged") {
					return m, nil
				}
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
				// Checkout branch — not allowed on merged or closed PRs.
				if strings.EqualFold(m.detailPR.State, "merged") || strings.EqualFold(m.detailPR.State, "closed") {
					return m, nil
				}
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
			case key.Matches(msg, keys.PRReview):
				// Request reviewer — only allowed on open PRs.
				if strings.EqualFold(m.detailPR.State, "merged") || strings.EqualFold(m.detailPR.State, "closed") {
					return m, nil
				}
				m.loadingReviewerForm = true
				return m, tea.Batch(fetchReviewerDataCmd(), m.spinner.Tick)
			case key.Matches(msg, keys.IssueAssign):
				if strings.EqualFold(m.detailPR.State, "merged") || strings.EqualFold(m.detailPR.State, "closed") {
					return m, nil
				}
				m.loadingPRAssigneeForm = true
				m.actionMsg = ""
				m.actionErr = nil
				return m, tea.Batch(m.spinner.Tick, fetchPRAssigneeDataCmd())
			case key.Matches(msg, keys.PRMerge):
				// Merge — not allowed on already merged or closed PRs.
				if strings.EqualFold(m.detailPR.State, "merged") || strings.EqualFold(m.detailPR.State, "closed") {
					return m, nil
				}
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
				)).WithTheme(buildHuhTheme(m.palette)).WithShowHelp(false).WithWidth(formWidth(m.width, m.uiTheme))
				return m, m.activeForm.Init()
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
				if item, ok := m.list.SelectedItem().(prListItem); ok {
					url := item.pr.URL
					return m, func() tea.Msg { github.OpenURL(url); return nil }
				}
			case key.Matches(msg, keys.IssueAssign):
				if item, ok := m.list.SelectedItem().(prListItem); ok {
					if strings.EqualFold(item.pr.State, "merged") || strings.EqualFold(item.pr.State, "closed") {
						return m, nil
					}
					m.loadingPRAssigneeForm = true
					m.actionMsg = ""
					m.actionErr = nil
					return m, tea.Batch(m.spinner.Tick, fetchPRAssigneeDataCmd())
				}
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
				)).WithTheme(buildHuhTheme(m.palette)).WithShowHelp(false).WithWidth(formWidth(m.width, m.uiTheme))
				return m, m.activeForm.Init()
			case key.Matches(msg, keys.Refresh):
				m.loading = true
				m.loaded = false
				m.err = nil
				cmds = append(cmds, m.fetchCmd())
				cmds = append(cmds, m.spinner.Tick)
			}
		}
		if key.Matches(msg, keys.Open) {
			if item, ok := m.list.SelectedItem().(prListItem); ok {
				// Use prefetched detail immediately if available — no spinner needed.
				if prefetched, ok := m.prefetchedDetails[item.pr.Number]; ok {
					m.detailPR = prefetched
					m.detail = viewport.New(m.width-4, detailViewportHeight(m.height, prMetaStripHeight, m.uiTheme))
					m.detail.Style = lipgloss.NewStyle().Background(m.palette.BgBody)
					m.detail.SetContent(lipgloss.NewStyle().Foreground(m.palette.TextDim).Background(m.palette.BgBody).Render("Rendering…") + "\n")
					m.showDetail = true
					delete(m.prefetchedDetails, item.pr.Number)
					cmds = append(cmds, renderPRContentCmd(prefetched, m.width, m.palette))
				} else {
					m.loadingDetail = true
					cmds = append(cmds, fetchPRDetailCmd(item.pr.Number))
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
				m.prefetchedDetails = make(map[int]github.PullRequest)
			}
			m.prefetchGen++
			cmds = append(cmds, m.prefetchNextItemsCmd())
		}
	}

	return m, tea.Batch(cmds...)
}

// ── View ──────────────────────────────────────────────────────────────────────

func (m PRsModel) View() string {
	// When a form is active, render it — replacing list/detail content.
	if m.activeForm != nil {
		body := m.activeForm.View()
		if bg := string(m.palette.BgBody); bg != "" {
			body = injectDocBg(body, bg)
		}
		return body
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
		line := fmt.Sprintf("\n  %s %s\n", m.spinner.View(), msg)
		if bg := string(m.palette.BgBody); bg != "" {
			line = injectDocBg(line, bg)
		}
		return line
	}

	if m.err != nil {
		b.WriteString(errorBox(fmt.Sprintf("Error: %v\n\nPress r to retry.", m.err), m.palette))
		return b.String()
	}

	if m.showDetail {
		b.WriteString(renderPRMetaStrip(m.detailPR, m.width-4, m.palette))
		b.WriteString(renderPRDetailView(m.detailPR, m.detail, m.actionMsg, m.actionErr, m.palette))
		return b.String()
	}

	b.WriteString(lipgloss.NewStyle().Padding(0, 2).Background(m.palette.BgBody).Render(m.list.View()))
	return b.String()
}

// renderPRDetailContent builds scrollable body-only content for the viewport.
func renderPRDetailContent(pr github.PullRequest, width int, pal Palette) string {
	if pr.Body == "" {
		return lipgloss.NewStyle().Foreground(pal.TextDim).Background(pal.BgBody).Render("No description.") + "\n"
	}
	return renderMarkdown(pr.Body, width-4, string(pal.BgBody))
}

// renderPRDetailView renders the scrollable viewport only.
// Action feedback (toast) is shown in the footer bar by AppModel.
func renderPRDetailView(_ github.PullRequest, vp viewport.Model, _ string, _ error, pal Palette) string {
	view := lipgloss.NewStyle().Padding(0, 2).Background(pal.BgBody).Render(viewportWithScrollHint(vp, pal))
	spacer := lipgloss.NewStyle().Background(pal.BgBody).Width(vp.Width + 4).Render("")
	return view + "\n" + spacer + "\n"
}
