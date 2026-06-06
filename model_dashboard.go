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
	sectionID int  // 0=reviewRequests, 1=myPRs, 2=assignedIssues (see sec* constants)
	itemIdx   int  // index within section (-1 for headers)
	isIssue   bool // true = Issue row, false = PullRequest row
}

const (
	secReviewRequests = 0
	secMyPRs          = 1
	secAssigned       = 2
)

var sectionNames = [3]string{
	"REVIEW REQUESTS",
	"MY OPEN PRs",
	"ASSIGNED TO ME",
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
}

func newDashboardModel(cfg Config) DashboardModel {
	s := spinner.New()
	s.Spinner = spinner.Dot
	s.Style = lipgloss.NewStyle().Foreground(lipgloss.Color("86"))

	return DashboardModel{
		spinner: s,
		loading: true,
		cfg:     cfg,
	}
}

func (m DashboardModel) fetchCmd() tea.Cmd {
	return func() tea.Msg {
		var data dashboardData
		var mu sync.Mutex
		var wg sync.WaitGroup
		var errs [3]error

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
				data.reviewRequests = result.([]github.PullRequest)
			case 1:
				data.myPRs = result.([]github.PullRequest)
			case 2:
				data.assignedIssues = result.([]github.Issue)
			}
		}

		wg.Add(3)
		go fetch(0, func() (interface{}, error) {
			return github.FetchPRs(github.PRFilters{Search: "review-requested:@me", State: "open", Limit: 20})
		})
		go fetch(1, func() (interface{}, error) {
			return github.FetchPRs(github.PRFilters{Author: "@me", State: "open", Limit: 20})
		})
		go fetch(2, func() (interface{}, error) {
			return github.FetchIssues(github.Filters{Assignee: "@me", State: "open", Limit: 20})
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
	sections := []struct {
		id    int
		count int
		issue bool
	}{
		{secReviewRequests, len(data.reviewRequests), false},
		{secMyPRs, len(data.myPRs), false},
		{secAssigned, len(data.assignedIssues), true},
	}
	for _, sec := range sections {
		if sec.count == 0 {
			continue
		}
		rows = append(rows, dashRow{isHeader: true, sectionID: sec.id, itemIdx: -1})
		for i := 0; i < sec.count; i++ {
			rows = append(rows, dashRow{isHeader: false, sectionID: sec.id, itemIdx: i, isIssue: sec.issue})
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
			case secReviewRequests:
				number = m.data.reviewRequests[row.itemIdx].Number
			case secMyPRs:
				number = m.data.myPRs[row.itemIdx].Number
			case secAssigned:
				number = m.data.assignedIssues[row.itemIdx].Number
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
	ReviewRequests int
	MyPRs          int
	Assigned       int
}

func (m DashboardModel) Counts() DashCounts {
	if !m.loaded {
		return DashCounts{}
	}
	return DashCounts{
		ReviewRequests: len(m.data.reviewRequests),
		MyPRs:          len(m.data.myPRs),
		Assigned:       len(m.data.assignedIssues),
	}
}

func (m DashboardModel) View() string {
	if m.loading {
		return fmt.Sprintf("\n  %s Loading dashboard...\n", m.spinner.View())
	}
	if m.err != nil {
		return errorBox(fmt.Sprintf("Dashboard error: %v\n\nPress r to retry.", m.err))
	}

	width := m.width - 4
	if width < 60 {
		width = 60
	}

	var b strings.Builder

	// ── Styles ────────────────────────────────────────────────────────────────
	selectedBg := lipgloss.Color("235")

	nameStyle := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("208"))
	countStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("141")).Bold(true)
	ruleStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("237"))
	mutedStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("244"))
	// stateIcon returns the type icon colored by state.
	// ⚑ = issue (flag), ⤴ = PR (upward arrow). Color = state.
	issueIcon := func(state string) string {
		color := lipgloss.Color("83") // green = open
		if strings.EqualFold(state, "closed") {
			color = lipgloss.Color("196") // red = closed
		}
		return lipgloss.NewStyle().Foreground(color).Bold(true).Render("⚑")
	}
	prIcon := func(state string, isDraft bool) string {
		var color lipgloss.Color
		switch {
		case isDraft:
			color = lipgloss.Color("214") // amber = draft
		case strings.EqualFold(state, "merged"):
			color = lipgloss.Color("141") // purple = merged
		case strings.EqualFold(state, "closed"):
			color = lipgloss.Color("196") // red = closed
		default:
			color = lipgloss.Color("83") // green = open
		}
		return lipgloss.NewStyle().Foreground(color).Bold(true).Render("⤴")
	}

	sectionIcons := [3]string{"⟳", "⎇", "●"}
	sectionCounts := [3]int{
		len(m.data.reviewRequests),
		len(m.data.myPRs),
		len(m.data.assignedIssues),
	}

	// ── renderSectionHeader ───────────────────────────────────────────────────
	// Renders: "  icon  NAME  ─────────────────────────────────  N"
	renderSectionHeader := func(sectionID int) string {
		icon := sectionIcons[sectionID]
		name := sectionNames[sectionID]
		count := sectionCounts[sectionID]

		left := "  " + icon + "  " + nameStyle.Render(name) + "  "
		right := "  " + countStyle.Render(fmt.Sprintf("%d", count))
		ruleW := width - lipgloss.Width(left) - lipgloss.Width(right)
		if ruleW < 1 {
			ruleW = 1
		}
		return left + ruleStyle.Render(strings.Repeat("─", ruleW)) + right
	}

	// ── renderRow ─────────────────────────────────────────────────────────────
	// Renders a two-row item matching the Issues list style.
	// Line 1: [accent] icon  #N  Title…(fill)…  timestamp
	// Line 2: [accent]           line2content
	renderRow := func(selected bool, icon, title, tsStr, line2content string, number int) string {
		var base lipgloss.Style
		if selected {
			base = lipgloss.NewStyle().Background(selectedBg)
		} else {
			base = lipgloss.NewStyle()
		}

		// Accent bar — reused on both rows so it spans the full item height.
		var accent string
		if selected {
			accent = lipgloss.NewStyle().Foreground(lipgloss.Color("86")).Background(selectedBg).Render("▌") +
				base.Render(" ")
		} else {
			accent = "  "
		}

		numStyle := base.Foreground(lipgloss.Color("69"))
		if selected {
			numStyle = numStyle.Bold(true)
		}
		numStr := numStyle.Render(fmt.Sprintf(" #%-4d", number))

		tsW := lipgloss.Width(tsStr)

		// left prefix = accent(2) + icon(1) + "  "(2) + numStr(6) + sp(1) = 12
		const prefixW = 12
		titleMaxW := width - prefixW - 2 - tsW - 1
		if titleMaxW < 10 {
			titleMaxW = 10
		}

		titleStyle := base.Foreground(lipgloss.Color("252"))
		if selected {
			titleStyle = base.Foreground(lipgloss.Color("255")).Bold(true)
		}
		titleStr := titleStyle.Render(truncate(cleanLine(title), titleMaxW))
		titleActualW := lipgloss.Width(titleStr)

		fillW := width - prefixW - titleActualW - tsW - 1
		if fillW < 1 {
			fillW = 1
		}
		fill := base.Render(strings.Repeat(" ", fillW))

		line1 := accent + icon + "  " + numStr + " " + titleStr + fill + tsStr + base.Render(" ")

		// Line 2: accent + indent matching prefixW + line2content.
		indent2 := accent + base.Render(strings.Repeat(" ", prefixW-2))
		line2 := indent2 + line2content + base.Render(" ")

		return line1 + "\n" + line2
	}

	// halfSpace is a blank line that blends with the body background, used as
	// a spacing row between section headers and items.
	halfSpace := strings.Repeat(" ", width)

	// ── Render rows ───────────────────────────────────────────────────────────
	firstSection := true
	for i, row := range m.rows {
		if row.isHeader {
			// Blank line + half-space before every section except the first.
			if !firstSection {
				b.WriteString("\n")
				b.WriteString(halfSpace + "\n")
			}
			firstSection = false
			b.WriteString(renderSectionHeader(row.sectionID) + "\n")
			// Half-space between header and first item.
			b.WriteString(halfSpace + "\n")
			continue
		}

		selected := i == m.cursor

		switch row.sectionID {
		case secReviewRequests:
			p := m.data.reviewRequests[row.itemIdx]
			ts := mutedStyle.Render(timeAgo(p.CreatedAt))
			line2 := mutedStyle.Render("@"+truncate(p.Author.Login, 14)) +
				mutedStyle.Render("  ·  ") + summarizeChecks(p.StatusRollup)
			b.WriteString(renderRow(selected, prIcon(p.State, p.IsDraft), p.Title, ts, line2, p.Number) + "\n")

		case secMyPRs:
			p := m.data.myPRs[row.itemIdx]
			ts := mutedStyle.Render(timeAgo(p.CreatedAt))
			var line2 string
			if p.HeadRefName != "" && p.BaseRefName != "" {
				line2 = mutedStyle.Render(truncate(p.HeadRefName, 18)) +
					lipgloss.NewStyle().Foreground(lipgloss.Color("252")).Render(" → ") +
					mutedStyle.Render(truncate(p.BaseRefName, 12))
			} else if p.IsDraft {
				line2 = mutedStyle.Render("draft")
			}
			if checks := summarizeChecks(p.StatusRollup); checks != "" {
				if line2 != "" {
					line2 += mutedStyle.Render("  ·  ") + checks
				} else {
					line2 = checks
				}
			}
			b.WriteString(renderRow(selected, prIcon(p.State, p.IsDraft), p.Title, ts, line2, p.Number) + "\n")

		case secAssigned:
			iss := m.data.assignedIssues[row.itemIdx]
			ts := mutedStyle.Render(timeAgo(iss.CreatedAt))
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
			line2 := mutedStyle.Render(truncate(assigneeText, 20))
			if labels := issueRowLabels(iss.Labels, "", labelMax); labels != "" {
				line2 += mutedStyle.Render("  ·  ") + labels
			}
			b.WriteString(renderRow(selected, issueIcon(iss.State), iss.Title, ts, line2, iss.Number) + "\n")
		}
	}

	return b.String()
}
