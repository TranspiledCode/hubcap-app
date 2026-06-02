// model_dashboard.go
package main

import (
	"fmt"
	"strings"
	"sync"

	"hubcap/internal/github"

	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// ── Shared dashboard types ────────────────────────────────────────────────────

// dashRow is one entry in the flat navigable list rendered on screen.
type dashRow struct {
	isHeader  bool
	sectionID int  // 0=reviewRequests, 1=myPRs, 2=assignedIssues, 3=availableIssues
	itemIdx   int  // index within section (-1 for headers)
	isIssue   bool // true = Issue row, false = PullRequest row
}

const (
	secReviewRequests = 0
	secMyPRs          = 1
	secAssigned       = 2
	secAvailable      = 3
)

var sectionNames = [4]string{
	"REVIEW REQUESTS",
	"MY OPEN PRs",
	"ASSIGNED TO ME",
	"AVAILABLE TO GRAB",
}

// ── DashboardModel ────────────────────────────────────────────────────────────

type DashboardModel struct {
	spinner spinner.Model
	loading bool
	loaded   bool
	err      error
	data     dashboardData
	cfg      Config
	cursor   int
	rows     []dashRow
	width    int
	height   int
	action   string
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
				data.reviewRequests = result.([]github.PullRequest)
			case 1:
				data.myPRs = result.([]github.PullRequest)
			case 2:
				data.assignedIssues = result.([]github.Issue)
			case 3:
				data.availableIssues = result.([]github.Issue)
			}
		}

		wg.Add(4)
		go fetch(0, func() (interface{}, error) {
			return github.FetchPRs(github.PRFilters{Search: "review-requested:@me", State: "open", Limit: 20})
		})
		go fetch(1, func() (interface{}, error) {
			return github.FetchPRs(github.PRFilters{Author: "@me", State: "open", Limit: 20})
		})
		go fetch(2, func() (interface{}, error) {
			return github.FetchIssues(github.Filters{Assignee: "@me", State: "open", Limit: 20})
		})
		cfg := m.cfg // capture for goroutine
		go fetch(3, func() (interface{}, error) {
			f := cfg.AvailableFilter
			if f.State == "" {
				f.State = "open"
			}
			if f.Limit == 0 {
				f.Limit = 20
			}
			return github.FetchIssues(f)
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
		{secAvailable, len(data.availableIssues), true},
	}
	for _, sec := range sections {
		if sec.count == 0 {
			continue // hide empty sections entirely
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
		m.width = msg.Width
		m.height = msg.Height

	case spinner.TickMsg:
		var cmd tea.Cmd
		m.spinner, cmd = m.spinner.Update(msg)
		cmds = append(cmds, cmd)

	case tea.KeyMsg:
		if m.loading {
			break
		}
		switch msg.String() {
		case "q":
			m.action = "quit"
			return m, nil
		case "tab":
			m.action = "switch"
			return m, nil
		case "r":
			m.loading = true
			m.loaded = false
			cmds = append(cmds, m.fetchCmd())
			cmds = append(cmds, m.spinner.Tick)
		case "up", "k":
			m.moveCursor(-1)
		case "down", "j":
			m.moveCursor(1)
		case "enter", " ":
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
			case secAvailable:
				number = m.data.availableIssues[row.itemIdx].Number
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

// DashCounts holds per-section counts for display in the header filter bar
type DashCounts struct {
	ReviewRequests int
	MyPRs          int
	Assigned       int
	Available      int
}

func (m DashboardModel) Counts() DashCounts {
	if !m.loaded {
		return DashCounts{}
	}
	return DashCounts{
		ReviewRequests: len(m.data.reviewRequests),
		MyPRs:          len(m.data.myPRs),
		Assigned:       len(m.data.assignedIssues),
		Available:      len(m.data.availableIssues),
	}
}

func (m DashboardModel) View() string {
	if m.loading {
		return fmt.Sprintf("\n  %s Loading dashboard...\n", m.spinner.View())
	}
	if m.err != nil {
		return errorBox(fmt.Sprintf("Dashboard error: %v\n\nPress r to retry.", m.err))
	}

	var b strings.Builder

	headerStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("208")).
		MarginTop(1)
	selectedStyle := lipgloss.NewStyle().
		Background(lipgloss.Color("23")).
		Foreground(lipgloss.Color("86"))

	sectionCounts := map[int]int{
		secReviewRequests: len(m.data.reviewRequests),
		secMyPRs:         len(m.data.myPRs),
		secAssigned:      len(m.data.assignedIssues),
		secAvailable:     len(m.data.availableIssues),
	}
	countStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("205")).Bold(true)

	for i, row := range m.rows {
		if row.isHeader {
			count := sectionCounts[row.sectionID]
			label := headerStyle.Render("  "+sectionNames[row.sectionID]) +
				countStyle.Render(fmt.Sprintf(" (%d)", count))
			b.WriteString(label + "\n")
			continue
		}
		selected := i == m.cursor
		prefix := "    "
		if selected {
			prefix = "  > "
		}

		var line string
		switch row.sectionID {
		case secReviewRequests, secMyPRs:
			pr := m.data.reviewRequests
			if row.sectionID == secMyPRs {
				pr = m.data.myPRs
			}
			p := pr[row.itemIdx]
			line = fmt.Sprintf("%s%s %-6d %-55s %s",
				prefix, stateIndicator(p.State, p.IsDraft), p.Number,
				truncate(p.Title, 55), coloredLabelsCompact(p.Labels, 30))
		case secAssigned:
			iss := m.data.assignedIssues[row.itemIdx]
			line = fmt.Sprintf("%s%s %-6d %-55s %s",
				prefix, stateIndicator(iss.State, false), iss.Number,
				truncate(iss.Title, 55), coloredLabelsCompact(iss.Labels, 30))
		case secAvailable:
			iss := m.data.availableIssues[row.itemIdx]
			line = fmt.Sprintf("%s%s %-6d %-55s %s",
				prefix, stateIndicator(iss.State, false), iss.Number,
				truncate(iss.Title, 55), coloredLabelsCompact(iss.Labels, 30))
		}

		if selected {
			b.WriteString(selectedStyle.Render(line) + "\n")
		} else {
			b.WriteString(line + "\n")
		}
	}

	return b.String()
}
