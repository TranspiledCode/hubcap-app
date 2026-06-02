// model_prs.go
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

// ── PR action messages ────────────────────────────────────────────────────────

type prActionDoneMsg struct {
	message string
	number  int // PR to re-fetch after action (0 = don't re-fetch)
}

type prActionErrMsg struct {
	err error
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

	// Action feedback
	actionMsg string
	actionErr error

	action string
}

func newPRsModel(filters github.PRFilters) PRsModel {
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

	return PRsModel{
		list:    l,
		spinner: s,
		loading: true,
		filters: filters,
	}
}

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

func (m PRsModel) Update(msg tea.Msg) (PRsModel, tea.Cmd) {
	var cmds []tea.Cmd

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
		m.detail = viewport.New(m.width-4, m.height-headerHeight()-4)
		m.detail.SetContent(content)
		m.showDetail = true

	case prActionDoneMsg:
		m.actionMsg = msg.message
		m.actionErr = nil
		if msg.number > 0 {
			m.loadingDetail = true
			return m, tea.Batch(fetchPRDetailCmd(msg.number), m.spinner.Tick)
		}
		return m, nil

	case prActionErrMsg:
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
				return m, fetchPRDetailCmd(m.detailPR.Number)
			case "o":
				num := m.detailPR.Number
				return m, tea.Exec(newFilterCmd(func() error {
					return github.RunCommandPassthrough("gh", "pr", "view",
						fmt.Sprintf("%d", num), "--web")
				}), func(err error) tea.Msg { return nil })
			case "u":
				if err := copyText(m.detailPR.URL); err != nil {
					m.actionErr = err
				} else {
					m.actionMsg = "URL copied to clipboard."
				}
				return m, nil
			case "x":
				// Close or Reopen PR
				pr := m.detailPR
				return m, func() tea.Msg {
					var err error
					var done string
					if pr.State == "closed" {
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
				}
			case "c":
				// Checkout branch
				pr := m.detailPR
				return m, tea.Exec(newFilterCmd(func() error {
					return github.RunCommandPassthrough("gh", "pr", "checkout",
						fmt.Sprintf("%d", pr.Number))
				}), func(err error) tea.Msg {
					if err != nil {
						return prActionErrMsg{err: err}
					}
					return nil
				})
			case "m":
				// Merge with type selection via huh
				pr := m.detailPR
				mergeType := "rebase" // default
				return m, tea.Exec(newFilterCmd(func() error {
					form := huh.NewForm(huh.NewGroup(
						huh.NewSelect[string]().
							Title(fmt.Sprintf("Merge PR #%d", pr.Number)).
							Options(
								huh.NewOption("Rebase and merge", "rebase"),
								huh.NewOption("Squash and merge", "squash"),
								huh.NewOption("Merge commit", "merge"),
							).
							Value(&mergeType),
					)).WithTheme(huh.ThemeCatppuccin())
					if err := form.Run(); err != nil {
						return nil // cancelled
					}
					return github.RunCommandPassthrough("gh", "pr", "merge",
						fmt.Sprintf("%d", pr.Number), "--"+mergeType)
				}), func(err error) tea.Msg {
					if err != nil {
						return prActionErrMsg{err: err}
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

		switch msg.String() {
		case "n":
			if !m.list.SettingFilter() {
				return m, tea.Exec(newFilterCmd(func() error {
					return runCreatePRForm()
				}), func(err error) tea.Msg {
					m.loading = true
					m.loaded = false
					return prsFetchedMsg{} // triggers refresh
				})
			}
		case "enter":
			if item, ok := m.list.SelectedItem().(prListItem); ok {
				m.loadingDetail = true
				cmds = append(cmds, fetchPRDetailCmd(item.pr.Number))
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

func (m PRsModel) View() string {
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
		b.WriteString(renderPRDetailView(m.detailPR, m.detail, m.actionMsg, m.actionErr))
		return b.String()
	}

	b.WriteString(lipgloss.NewStyle().Margin(0, 2).Render(m.list.View()))
	return b.String()
}

func renderPRDetailContent(pr github.PullRequest, width int) string {
	var b strings.Builder

	b.WriteString(styleTitle.Render(pr.Title) + "\n\n")
	b.WriteString(fmt.Sprintf("%s  #%d  %s  %s\n\n",
		stateIndicator(pr.State, pr.IsDraft),
		pr.Number,
		styleGray.Render("by"),
		pr.Author.Login,
	))

	if pr.HeadRefName != "" {
		b.WriteString(styleGray.Render("Branch: ") + pr.HeadRefName + "\n")
	}
	b.WriteString(styleGray.Render("Checks: ") + summarizeChecks(pr.StatusRollup) + "\n")
	if len(pr.Labels) > 0 {
		b.WriteString(styleGray.Render("Labels: ") + coloredLabelsCompact(pr.Labels, width-10) + "\n")
	}
	b.WriteString("\n")

	if pr.Body != "" {
		b.WriteString(pr.Body + "\n")
	} else {
		b.WriteString(styleGray.Render("No description.") + "\n")
	}

	return b.String()
}

func renderPRDetailView(pr github.PullRequest, vp viewport.Model, actionMsg string, actionErr error) string {
	var b strings.Builder
	b.WriteString(lipgloss.NewStyle().Margin(0, 2).Render(vp.View()))
	b.WriteString("\n")

	if actionErr != nil {
		b.WriteString(errorBox(actionErr.Error()))
	} else if actionMsg != "" {
		b.WriteString(successBox(actionMsg))
	}

	closeOrReopen := "close"
	if pr.State == "closed" {
		closeOrReopen = "reopen"
	}
	hints := hintBar(
		"c", "checkout",
		"m", "merge",
		"x", closeOrReopen,
		"o", "browser",
		"u", "copy URL",
		"r", "refresh",
		"b", "back",
	)
	b.WriteString(hints)
	return b.String()
}
