// model_issues.go
package main

import (
	"fmt"
	"strings"

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
	message string
	number  int // issue to re-fetch after action
}

type issueActionErrMsg struct {
	err error
}

// ── IssueListItem ─────────────────────────────────────────────────────────────

type issueListItem struct {
	issue github.Issue
}

func (i issueListItem) Title() string       { return fmt.Sprintf("#%-5d %s", i.issue.Number, i.issue.Title) }
func (i issueListItem) Description() string { return fmt.Sprintf("%s  %s", joinUsers(i.issue.Assignees), coloredLabelsCompact(i.issue.Labels, 60)) }
func (i issueListItem) FilterValue() string { return fmt.Sprintf("%d %s", i.issue.Number, i.issue.Title) }

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

	// Action feedback
	actionMsg string // success/info message to show briefly
	actionErr error  // error from last action

	// Navigation signal back to AppModel
	action string
}

func newIssuesModel(filters github.Filters) IssuesModel {
	s := spinner.New()
	s.Spinner = spinner.Dot
	s.Style = lipgloss.NewStyle().Foreground(lipgloss.Color("86"))

	delegate := list.NewDefaultDelegate()
	delegate.Styles.SelectedTitle = lipgloss.NewStyle().
		Foreground(lipgloss.Color("86")).
		Bold(true).
		Border(lipgloss.NormalBorder(), false, false, false, true).
		BorderForeground(lipgloss.Color("86")).
		Padding(0, 0, 0, 1)
	delegate.Styles.SelectedDesc = lipgloss.NewStyle().
		Foreground(lipgloss.Color("245")).
		Border(lipgloss.NormalBorder(), false, false, false, true).
		BorderForeground(lipgloss.Color("86")).
		Padding(0, 0, 0, 1)

	l := list.New([]list.Item{}, delegate, 0, 0)
	l.SetShowTitle(false)
	l.SetShowStatusBar(true)
	l.SetShowHelp(false)
	l.SetFilteringEnabled(true)
	l.Styles.StatusBar = lipgloss.NewStyle().
		Foreground(lipgloss.Color("244")).
		Padding(1, 2, 1, 4)
	l.Styles.StatusBarFilterCount = lipgloss.NewStyle().
		Foreground(lipgloss.Color("86")).
		Padding(1, 2, 1, 4)

	return IssuesModel{
		list:    l,
		spinner: s,
		loading: true,
		filters: filters,
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

func (m IssuesModel) Update(msg tea.Msg) (IssuesModel, tea.Cmd) {
	var cmds []tea.Cmd

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
		m.detail = viewport.New(m.width-4, m.height-headerHeight()-4)
		m.detail.SetContent(content)
		m.showDetail = true

	case issueActionDoneMsg:
		m.actionMsg = msg.message
		m.actionErr = nil
		if msg.number > 0 {
			m.loadingDetail = true
			return m, tea.Batch(fetchIssueDetailCmd(msg.number), m.spinner.Tick)
		}
		return m, nil

	case issueActionErrMsg:
		m.actionErr = msg.err
		m.actionMsg = ""
		return m, nil

	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.list.SetSize(msg.Width-4, msg.Height-headerHeight()-2)
		if m.showDetail {
			m.detail.Width = msg.Width - 4
			m.detail.Height = msg.Height - headerHeight() - 4
		}

	case spinner.TickMsg:
		var cmd tea.Cmd
		m.spinner, cmd = m.spinner.Update(msg)
		cmds = append(cmds, cmd)

	case tea.KeyMsg:
		// Don't handle keys while loading
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
				// Refresh detail
				m.loadingDetail = true
				m.actionMsg = ""
				m.actionErr = nil
				return m, fetchIssueDetailCmd(m.detailIssue.Number)
			case "o":
				// Open in browser
				num := m.detailIssue.Number
				return m, tea.Exec(newFilterCmd(func() error {
					return github.RunCommandPassthrough("gh", "issue", "view",
						fmt.Sprintf("%d", num), "--web")
				}), func(err error) tea.Msg { return nil })
			case "u":
				// Copy URL
				if err := copyText(m.detailIssue.URL); err != nil {
					m.actionErr = err
				} else {
					m.actionMsg = "URL copied to clipboard."
				}
				return m, nil
			case "c":
				// Close or Reopen
				issue := m.detailIssue
				return m, func() tea.Msg {
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
				}
			case "a":
				// Assign to @me
				issue := m.detailIssue
				return m, func() tea.Msg {
					if err := github.AssignIssueSelf(issue.Number); err != nil {
						return issueActionErrMsg{err: err}
					}
					return issueActionDoneMsg{message: "Assigned to @me.", number: issue.Number}
				}
			case "l":
				// Add label via huh input
				issue := m.detailIssue
				label := ""
				return m, tea.Exec(newFilterCmd(func() error {
					form := huh.NewForm(huh.NewGroup(
						huh.NewInput().
							Title("Add label").
							Placeholder("label name").
							Value(&label),
					)).WithTheme(huh.ThemeCatppuccin())
					if err := form.Run(); err != nil {
						return nil // cancelled
					}
					label = strings.TrimSpace(label)
					if label == "" {
						return nil
					}
					return github.AddIssueLabel(issue.Number, label)
				}), func(err error) tea.Msg {
					if err != nil {
						return issueActionErrMsg{err: err}
					}
					if label == "" {
						return nil
					}
					return issueActionDoneMsg{message: fmt.Sprintf("Label %q added.", label), number: issue.Number}
				})
			case "d":
				// Develop branch
				issue := m.detailIssue
				defaultBranch := deriveBranchName(issue.Number, issue.Title)
				branchName := defaultBranch
				return m, tea.Exec(newFilterCmd(func() error {
					form := huh.NewForm(huh.NewGroup(
						huh.NewInput().
							Title("Branch name").
							Description(fmt.Sprintf("Default: %s", defaultBranch)).
							Placeholder(defaultBranch).
							Value(&branchName),
					)).WithTheme(huh.ThemeCatppuccin())
					if err := form.Run(); err != nil {
						return nil // cancelled
					}
					if strings.TrimSpace(branchName) == "" {
						branchName = defaultBranch
					}
					return github.RunCommandPassthrough("gh", "issue", "develop",
						fmt.Sprintf("%d", issue.Number), "--checkout", "--name", branchName)
				}), func(err error) tea.Msg {
					if err != nil {
						return issueActionErrMsg{err: err}
					}
					// After developing branch, return to list
					return nil
				})
			case "p":
				// Create PR from current branch
				return m, tea.Exec(newFilterCmd(func() error {
					return github.RunCommandPassthrough("gh", "pr", "create", "--fill")
				}), func(err error) tea.Msg {
					if err != nil {
						return issueActionErrMsg{err: err}
					}
					return nil
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
				return m, tea.Exec(newFilterCmd(func() error {
					return runCreateIssueForm()
				}), func(err error) tea.Msg {
					m.loading = true
					m.loaded = false
					return issuesFetchedMsg{} // triggers refresh
				})
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

func (m IssuesModel) View() string {
	var b strings.Builder

	// Loading state
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

	// Error state
	if m.err != nil {
		b.WriteString(errorBox(fmt.Sprintf("Error: %v\n\nPress r to retry.", m.err)))
		return b.String()
	}

	// Detail view
	if m.showDetail {
		b.WriteString(renderIssueDetailView(m.detailIssue, m.detail, m.actionMsg, m.actionErr))
		return b.String()
	}

	// List view
	b.WriteString(lipgloss.NewStyle().Margin(0, 2).Render(m.list.View()))
	return b.String()
}

// headerHeight returns the number of lines used by headerView()
// 3 title bar + 3 tab bar + 3 filter bar = 9 lines
func headerHeight() int { return 9 }

// renderIssueDetailContent builds the full text content for the viewport
func renderIssueDetailContent(issue github.Issue, width int) string {
	var b strings.Builder

	b.WriteString(styleTitle.Render(issue.Title) + "\n\n")
	b.WriteString(fmt.Sprintf("%s  #%d  %s  %s\n",
		stateIndicator(issue.State, false),
		issue.Number,
		styleGray.Render("opened by"),
		issue.Author.Login,
	))

	if len(issue.Assignees) > 0 {
		b.WriteString(styleGray.Render("Assignees: ") + joinUsers(issue.Assignees) + "\n")
	}
	if len(issue.Labels) > 0 {
		b.WriteString(styleGray.Render("Labels:    ") + coloredLabelsCompact(issue.Labels, width-14) + "\n")
	}
	b.WriteString("\n")

	if issue.Body != "" {
		b.WriteString(issue.Body + "\n")
	} else {
		b.WriteString(styleGray.Render("No description.") + "\n")
	}

	return b.String()
}

// renderIssueDetailView renders the detail screen with viewport, hint bar, and action feedback.
func renderIssueDetailView(issue github.Issue, vp viewport.Model, actionMsg string, actionErr error) string {
	var b strings.Builder
	b.WriteString(lipgloss.NewStyle().Margin(0, 2).Render(vp.View()))
	b.WriteString("\n")

	// Show action feedback
	if actionErr != nil {
		b.WriteString(errorBox(actionErr.Error()))
	} else if actionMsg != "" {
		b.WriteString(successBox(actionMsg))
	}

	// Hint bar
	closeOrReopen := "close"
	if strings.EqualFold(issue.State, "closed") {
		closeOrReopen = "reopen"
	}
	hints := hintBar(
		"d", "develop",
		"p", "PR",
		"c", closeOrReopen,
		"a", "assign",
		"l", "label",
		"o", "browser",
		"u", "copy URL",
		"r", "refresh",
		"b", "back",
	)
	b.WriteString(hints)
	return b.String()
}
