// model_dashboard.go
package main

import (
	"fmt"
	"strings"
	"sync"

	"hubcap/internal/github"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// ── Shared dashboard types ────────────────────────────────────────────────────

// dashRow is one entry in the flat navigable list rendered on screen.
type dashRow struct {
	isHeader  bool
	sectionID int  // 0=myPRs, 1=assigned, 2=reviewRequests (see sec* constants)
	itemIdx   int  // index within section slice (-1 for headers)
	isIssue   bool // true = Issue row, false = PullRequest row
}

const (
	secMyPRs          = 0 // PRs I opened
	secAssigned       = 1 // issues + PRs assigned to me (mixed)
	secReviewRequests = 2 // PRs where I'm a requested reviewer
)

var sectionNames = [3]string{
	"MY OPEN PRs",
	"ASSIGNED TO ME",
	"PRs TO REVIEW",
}

// ── DashboardModel ────────────────────────────────────────────────────────────

type DashboardModel struct {
	spinner spinner.Model
	loading bool
	loaded  bool
	err     error
	data    dashboardData
	cfg     Config
	cursor  int
	rows    []dashRow
	width   int
	height  int

	// palette mirrors Config.ColorTheme for dashboard item colours.
	palette Palette
}

func newDashboardModel(cfg Config, pal Palette) DashboardModel {
	s := spinner.New()
	s.Spinner = spinner.Dot
	s.Style = lipgloss.NewStyle().Foreground(pal.Accent)

	return DashboardModel{
		spinner: s,
		loading: true,
		cfg:     cfg,
		palette: pal,
	}
}

func (m DashboardModel) fetchCmd() tea.Cmd {
	return func() tea.Msg {
		var data dashboardData
		var mu sync.Mutex
		var wg sync.WaitGroup
		var errs [4]error

		fetch := func(i int, fn func() (interface{}, error)) {
			defer wg.Done()
			result, err := fn()
			mu.Lock()
			defer mu.Unlock()
			errs[i] = err
			if err != nil {
				return
			}
			switch i {
			case 0:
				data.myPRs = result.([]github.PullRequest)
			case 1:
				data.assignedIssues = result.([]github.Issue)
			case 2:
				data.assignedPRs = result.([]github.PullRequest)
			case 3:
				data.reviewRequests = result.([]github.PullRequest)
			}
		}

		wg.Add(4)
		go fetch(0, func() (interface{}, error) {
			return github.FetchPRs(github.PRFilters{Author: "@me", State: "open", Limit: 20})
		})
		go fetch(1, func() (interface{}, error) {
			return github.FetchIssues(github.Filters{Assignee: "@me", State: "open", Limit: 20})
		})
		go fetch(2, func() (interface{}, error) {
			return github.FetchPRs(github.PRFilters{Assignee: "@me", State: "open", Limit: 20})
		})
		go fetch(3, func() (interface{}, error) {
			return github.FetchPRs(github.PRFilters{Search: "review-requested:@me", State: "open", Limit: 20})
		})
		wg.Wait()

		for _, err := range errs {
			if err != nil {
				return dashboardFetchedMsg{data: data, err: err}
			}
		}
		return dashboardFetchedMsg{data: data, err: nil}
	}
}

func buildDashRows(data dashboardData) []dashRow {
	var rows []dashRow

	// MY OPEN PRs
	if len(data.myPRs) > 0 {
		rows = append(rows, dashRow{isHeader: true, sectionID: secMyPRs, itemIdx: -1})
		for i := range data.myPRs {
			rows = append(rows, dashRow{sectionID: secMyPRs, itemIdx: i, isIssue: false})
		}
	}

	// ASSIGNED TO ME — issues first, then PRs, both under secAssigned.
	// isIssue distinguishes the two when looking up items.
	if len(data.assignedIssues)+len(data.assignedPRs) > 0 {
		rows = append(rows, dashRow{isHeader: true, sectionID: secAssigned, itemIdx: -1})
		for i := range data.assignedIssues {
			rows = append(rows, dashRow{sectionID: secAssigned, itemIdx: i, isIssue: true})
		}
		for i := range data.assignedPRs {
			rows = append(rows, dashRow{sectionID: secAssigned, itemIdx: i, isIssue: false})
		}
	}

	// PRs TO REVIEW
	if len(data.reviewRequests) > 0 {
		rows = append(rows, dashRow{isHeader: true, sectionID: secReviewRequests, itemIdx: -1})
		for i := range data.reviewRequests {
			rows = append(rows, dashRow{sectionID: secReviewRequests, itemIdx: i, isIssue: false})
		}
	}

	return rows
}

func (m DashboardModel) Update(msg tea.Msg) (DashboardModel, tea.Cmd) {
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case dashboardFetchedMsg:
		m.loading = false
		m.loaded = true
		if msg.err != nil {
			m.err = msg.err
			return m, nil
		}
		m.data = msg.data
		m.rows = buildDashRows(m.data)

	case tea.WindowSizeMsg:
		m.width = msg.Width - 2
		m.height = msg.Height - 2

	case spinner.TickMsg:
		var cmd tea.Cmd
		m.spinner, cmd = m.spinner.Update(msg)
		cmds = append(cmds, cmd)

	case tea.KeyMsg:
		if m.loading {
			break
		}
		switch {
		case key.Matches(msg, keys.Refresh):
			m.loading = true
			m.loaded = false
			cmds = append(cmds, m.fetchCmd())
			cmds = append(cmds, m.spinner.Tick)
		case key.Matches(msg, keys.Up):
			m.moveCursor(-1)
		case key.Matches(msg, keys.Down):
			m.moveCursor(1)
		case key.Matches(msg, keys.Top):
			m.moveCursorTop()
		case key.Matches(msg, keys.Bottom):
			m.moveCursorBottom()
		case key.Matches(msg, keys.Open) || msg.String() == " ":
			if len(m.rows) == 0 || m.cursor >= len(m.rows) {
				break
			}
			row := m.rows[m.cursor]
			if row.isHeader {
				break
			}
			isIssue := row.isIssue
			var number int
			switch row.sectionID {
			case secMyPRs:
				number = m.data.myPRs[row.itemIdx].Number
			case secAssigned:
				if row.isIssue {
					number = m.data.assignedIssues[row.itemIdx].Number
				} else {
					number = m.data.assignedPRs[row.itemIdx].Number
				}
			case secReviewRequests:
				number = m.data.reviewRequests[row.itemIdx].Number
			}
			return m, func() tea.Msg { return openItemMsg{isIssue: isIssue, number: number} }
		}
	}

	return m, tea.Batch(cmds...)
}

func (m *DashboardModel) moveCursor(delta int) {
	items := len(m.rows)
	if items == 0 {
		return
	}
	// Skip header rows
	next := m.cursor + delta
	for next >= 0 && next < items && m.rows[next].isHeader {
		next += delta
	}
	if next >= 0 && next < items {
		m.cursor = next
	}
}

func (m *DashboardModel) moveCursorTop() {
	for i, row := range m.rows {
		if !row.isHeader {
			m.cursor = i
			return
		}
	}
}

func (m *DashboardModel) moveCursorBottom() {
	for i := len(m.rows) - 1; i >= 0; i-- {
		if !m.rows[i].isHeader {
			m.cursor = i
			return
		}
	}
}

// DashCounts holds per-section counts for display in the header filter bar.
type DashCounts struct {
	MyPRs          int
	Assigned       int // issues + PRs assigned to me
	ReviewRequests int
}

func (m DashboardModel) Counts() DashCounts {
	if !m.loaded {
		return DashCounts{}
	}
	return DashCounts{
		MyPRs:          len(m.data.myPRs),
		Assigned:       len(m.data.assignedIssues) + len(m.data.assignedPRs),
		ReviewRequests: len(m.data.reviewRequests),
	}
}

func (m DashboardModel) View() string {
	if m.loading {
		line := fmt.Sprintf("\n  %s Loading dashboard...\n", m.spinner.View())
		if bg := string(m.palette.BgBody); bg != "" {
			line = injectDocBg(line, bg)
		}
		return line
	}
	if m.err != nil {
		return errorBox(fmt.Sprintf("Dashboard error: %v\n\nPress r to retry.", m.err), m.palette)
	}

	width := m.width - 4
	if width < 60 {
		width = 60
	}

	var b strings.Builder

	// ── Styles ────────────────────────────────────────────────────────────────
	pal := m.palette
	selectedBg := pal.BgSelected

	bgSt := lipgloss.NewStyle().Background(pal.BgBody)
	nameStyle := lipgloss.NewStyle().Bold(true).Foreground(pal.Meta).Background(pal.BgBody)
	countStyle := lipgloss.NewStyle().Foreground(pal.StatusMerged).Bold(true).Background(pal.BgBody)
	ruleStyle := lipgloss.NewStyle().Foreground(pal.TextFaint).Background(pal.BgBody)
	// stateIcon returns the type icon colored by state.
	// ⚑ = issue (flag), ⤴ = PR (upward arrow). Color = state.
	// These are kept as named color lookups; actual rendering now happens
	// inside the render loop with the row's base style (so selectedBg applies).
	prIconColor := func(state string, isDraft bool) lipgloss.Color {
		switch {
		case isDraft:
			return pal.StatusDraft
		case strings.EqualFold(state, "merged"):
			return pal.StatusMerged
		case strings.EqualFold(state, "closed"):
			return pal.StatusClosed
		default:
			return pal.StatusOpen
		}
	}

	sectionIcons := [3]string{"⎇", "●", "⟳"}
	sectionCounts := [3]int{
		len(m.data.myPRs),
		len(m.data.assignedIssues) + len(m.data.assignedPRs),
		len(m.data.reviewRequests),
	}

	// ── renderSectionHeader ───────────────────────────────────────────────────
	// Renders: "  icon  NAME  ─────────────────────────────────  N"
	renderSectionHeader := func(sectionID int) string {
		icon := sectionIcons[sectionID]
		name := sectionNames[sectionID]
		count := sectionCounts[sectionID]

		left := bgSt.Render(icon+"  ") + nameStyle.Render(name) + bgSt.Render("  ")
		right := bgSt.Render("  ") + countStyle.Render(fmt.Sprintf("%d", count))
		ruleW := width - lipgloss.Width(left) - lipgloss.Width(right)
		if ruleW < 1 {
			ruleW = 1
		}
		return left + ruleStyle.Render(strings.Repeat("─", ruleW)) + right
	}

	// ── renderRow ─────────────────────────────────────────────────────────────
	// Renders a two-row item. base must be pre-computed by the caller (with or
	// without Background(selectedBg)) so every piece uses the same background.
	// Line 1: [accent] icon  #N  Title…(fill)…  timestamp
	// Line 2: [accent]           line2Left  (fill)  line2Right
	renderRow := func(base lipgloss.Style, selected bool, icon, title, tsStr, line2Left, line2Right string, number int) string {
		// Accent bar — reused on both rows so it spans the full item height.
		var accent string
		if selected {
			accent = lipgloss.NewStyle().Foreground(pal.Accent).Background(selectedBg).Render("▌") +
				base.Render(" ")
		} else {
			accent = base.Render("  ")
		}

		numStyle := base.Foreground(pal.Number)
		if selected {
			numStyle = numStyle.Bold(true)
		}
		numStr := numStyle.Render(fmt.Sprintf(" #%-4d", number))

		tsW := lipgloss.Width(tsStr)

		// left prefix = accent(2) + icon(1) + numStr(6) + sp(1) = 10 — matches Issues delegate
		const prefixW = 10
		titleMaxW := width - prefixW - 2 - tsW - 1
		if titleMaxW < 10 {
			titleMaxW = 10
		}

		titleStyle := base.Foreground(pal.Text)
		if selected {
			titleStyle = base.Foreground(pal.TextBold).Bold(true)
		}
		titleStr := titleStyle.Render(truncate(cleanLine(title), titleMaxW))
		titleActualW := lipgloss.Width(titleStr)

		fillW := width - prefixW - titleActualW - tsW - 1
		if fillW < 1 {
			fillW = 1
		}
		fill := base.Render(strings.Repeat(" ", fillW))

		line1 := accent + icon + numStr + base.Render(" ") + titleStr + fill + tsStr + base.Render(" ")

		// Line 2: accent + indent + line2Left (fill) line2Right
		indent2 := accent + base.Render(strings.Repeat(" ", prefixW-2))
		line2RightW := lipgloss.Width(line2Right)
		line2LeftW := prefixW + lipgloss.Width(line2Left)
		const typeGap = 2
		line2FillW := width - line2LeftW - line2RightW - 1
		if line2FillW < typeGap {
			line2FillW = typeGap
		}
		line2Fill := base.Render(strings.Repeat(" ", line2FillW))
		line2 := indent2 + line2Left + line2Fill + line2Right + base.Render(" ")

		return line1 + "\n" + line2
	}

	// halfSpace is a blank line that blends with the body background, used as
	// a spacing row between section headers and items.
	halfSpace := bgSt.Width(width).Render("")

	// ── Render rows ───────────────────────────────────────────────────────────
	firstSection := true
	for i, row := range m.rows {
		if row.isHeader {
			// Blank line + half-space before every section except the first.
			if !firstSection {
				b.WriteString(bgSt.Width(width).Render("") + "\n")
				b.WriteString(halfSpace + "\n")
			}
			firstSection = false
			b.WriteString(renderSectionHeader(row.sectionID) + "\n")
			// Half-space between header and first item.
			b.WriteString(halfSpace + "\n")
			continue
		}

		selected := i == m.cursor

		// Blank line between items — matches Issues delegate Spacing()=1.
		if i > 0 && !m.rows[i-1].isHeader {
			b.WriteString(halfSpace + "\n")
		}

		// base carries Background(selectedBg) on the selected row so every
		// rendered segment — icon, timestamp, assignee, separator, type badge —
		// shares the same background and there are no dark gaps.
		var base lipgloss.Style
		var rowBg lipgloss.Color
		if selected {
			rowBg = selectedBg
			base = lipgloss.NewStyle().Background(selectedBg)
		} else {
			rowBg = pal.BgBody
			base = lipgloss.NewStyle().Background(pal.BgBody)
		}
		dimSep := base.Foreground(pal.TextFaint).Render("  ·  ")

		// renderPRRow renders a PR item (used for secMyPRs, secAssigned PR rows, secReviewRequests).
		renderPRRow := func(p github.PullRequest) {
			icon := base.Foreground(prIconColor(p.State, p.IsDraft)).Bold(true).Render("⤴")
			ts := base.Foreground(pal.TextDim).Render(timeAgo(p.CreatedAt))
			var line2Left string
			if p.HeadRefName != "" && p.BaseRefName != "" {
				line2Left = base.Foreground(pal.TextMuted).Render(truncate(p.HeadRefName, 18)) +
					base.Foreground(pal.Text).Render(" → ") +
					base.Foreground(pal.TextMuted).Render(truncate(p.BaseRefName, 12))
			} else if p.IsDraft {
				line2Left = base.Foreground(pal.TextMuted).Render("draft")
			}
			if checks := summarizeChecks(p.StatusRollup, rowBg, pal); checks != "" {
				if line2Left != "" {
					line2Left += dimSep + checks
				} else {
					line2Left = checks
				}
			}
			b.WriteString(renderRow(base, selected, icon, p.Title, ts, line2Left, "", p.Number) + "\n")
		}

		switch row.sectionID {
		case secMyPRs:
			renderPRRow(m.data.myPRs[row.itemIdx])

		case secAssigned:
			if row.isIssue {
				iss := m.data.assignedIssues[row.itemIdx]
				issIconColor := pal.StatusOpen
				if strings.EqualFold(iss.State, "closed") {
					issIconColor = pal.StatusClosed
				}
				icon := base.Foreground(issIconColor).Bold(true).Render("⚑")
				ts := base.Foreground(pal.TextDim).Render(timeAgo(iss.CreatedAt))
				labelMax := width * 40 / 100
				if labelMax < 15 {
					labelMax = 15
				}
				var assigneeText string
				if len(iss.Assignees) > 0 {
					assigneeText = "@" + joinUsers(iss.Assignees)
				} else {
					assigneeText = "unassigned"
				}
				line2Left := base.Foreground(pal.TextMuted).Render(truncate(assigneeText, 20))
				const dashMaxLabels = 3
				shownLabels := iss.Labels
				labelOverflow := 0
				if len(iss.Labels) > dashMaxLabels {
					shownLabels = iss.Labels[:dashMaxLabels]
					labelOverflow = len(iss.Labels) - dashMaxLabels
				}
				bgKey := string(pal.BgBody)
				if selected {
					bgKey = string(pal.BgSelected)
				}
				if labels := issueRowLabels(shownLabels, bgKey, labelMax, pal); labels != "" {
					line2Left += dimSep + labels
					if labelOverflow > 0 {
						line2Left += base.Foreground(pal.TextDim).Render(fmt.Sprintf(" +%d", labelOverflow))
					}
				}
				var line2Right string
				if iss.IssueType != "" {
					line2Right = base.Foreground(pal.Number).Render(iss.IssueType)
				} else {
					line2Right = base.Foreground(pal.TextFaint).Render("—")
				}
				b.WriteString(renderRow(base, selected, icon, iss.Title, ts, line2Left, line2Right, iss.Number) + "\n")
			} else {
				renderPRRow(m.data.assignedPRs[row.itemIdx])
			}

		case secReviewRequests:
			p := m.data.reviewRequests[row.itemIdx]
			icon := base.Foreground(prIconColor(p.State, p.IsDraft)).Bold(true).Render("⤴")
			ts := base.Foreground(pal.TextDim).Render(timeAgo(p.CreatedAt))
			line2Left := base.Foreground(pal.TextMuted).Render("@"+truncate(p.Author.Login, 14)) +
				dimSep + summarizeChecks(p.StatusRollup, rowBg, pal)
			b.WriteString(renderRow(base, selected, icon, p.Title, ts, line2Left, "", p.Number) + "\n")
		}
	}

	return lipgloss.NewStyle().Padding(0, 2).Background(pal.BgBody).Render(strings.TrimRight(b.String(), "\n"))
}
