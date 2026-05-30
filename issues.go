// issues.go
package main

import (
	"bufio"
	"fmt"
	"os"
	"strconv"
	"strings"

	"hubcap/internal/github"

	"github.com/charmbracelet/huh"
)

func browseIssues(reader *bufio.Reader, state *AppState) string {
	for {
		clearScreen()
		renderHeader(state, false)
		fmt.Println("Fetching issues...")

		issues, err := github.FetchIssues(state.IssueFilters)
		if err != nil {
			fmt.Println("\nCould not fetch issues:")
			fmt.Println(err)
			fmt.Println("  Tab — switch tabs  |  r — retry  |  q — quit")
			action := emptyTabAction(reader, state, TabPRs)
			switch action {
			case "quit":
				return "quit"
			case "switch":
				return ""
			}
			continue
		}

		if len(issues) == 0 {
			fmt.Println("\nNo issues found with the current filters.")
			fmt.Println("  Tab — switch tabs  |  f — change filters  |  q — quit")
			action := emptyTabAction(reader, state, TabPRs)
			switch action {
			case "quit":
				return "quit"
			case "switch":
				return ""
			case "filters":
				state.IssueFilters = configureFilters(reader, state)
			}
			continue
		}

		number, action := issueList(reader, state, issues)
		switch action {
		case "quit":
			return "quit"
		case "switch":
			return ""
		case "refresh":
			continue
		case "filters":
			state.IssueFilters = configureFilters(reader, state)
		case "new":
			clearScreen()
			renderHeader(state, false)
			fmt.Println("Ctrl+C to cancel.")
			github.RunCommandPassthrough("gh", "issue", "create")
		case "open":
			viewIssue(reader, state, number)
		}
	}
}

func issueList(reader *bufio.Reader, state *AppState, issues []github.Issue) (int, string) {
	if len(issues) == 0 {
		return 0, "back"
	}

	selected := state.IssueSelected
	if selected >= len(issues) {
		selected = 0
	}

	if err := enableRawMode(); err != nil {
		clearScreen()
		renderHeader(state, false)
		printIssuesTable(issues)

		fmt.Println()
		input := prompt(reader, "Issue number, n new, f filters, r refresh, q quit: ")
		input = strings.TrimSpace(strings.ToLower(input))

		switch input {
		case "", "q", "quit", "b", "back":
			return 0, "quit"
		case "r", "refresh":
			return 0, "refresh"
		case "n", "new":
			return 0, "new"
		case "f", "filters":
			return 0, "filters"
		}

		number, err := strconv.Atoi(input)
		if err != nil {
			return 0, "quit"
		}
		return number, "open"
	}
	defer disableRawMode()
	defer fmt.Print("\033[?25h")

	render := func() {
		fmt.Print("\033[H")
		renderHeader(state, true)
		fmt.Print("\033[?25l")
		fmt.Printf("  %-8s %-58s %-22s %-34s\033[K\r\n", "  #", "TITLE", "ASSIGNEE", "LABELS")
		fmt.Printf("  %-8s %-58s %-22s %-34s\033[K\r\n", "---", "-----", "--------", "------")
		for index, issue := range issues {
			prefix := "  "
			if index == selected {
				prefix = styleCyan.Render(">") + " "
			}
			indicator := stateIndicator(issue.State, false)
			fmt.Printf("%s%s %-6d %-58s %-22s %-34s\033[K\r\n",
				prefix, indicator, issue.Number,
				truncate(cleanLine(issue.Title), 58),
				truncate(joinUsers(issue.Assignees), 22),
				truncate(joinLabels(issue.Labels), 34),
			)
		}
		fmt.Print(hintSep(true))
		fmt.Print(hintBar("↑↓", "move", "enter", "open", "tab", "switch", "n", "new", "f", "filters", "r", "refresh", "q", "quit") + "\033[K\r\n")
		fmt.Print("\033[J")
	}

	render()
	buffer := make([]byte, 3)

	for {
		n, err := os.Stdin.Read(buffer)
		if err != nil || n == 0 {
			return 0, "quit"
		}
		key := string(buffer[:n])
		switch key {
		case "\t", "\x1b[Z":
			state.IssueSelected = selected
			state.ActiveTab = nextTab(state.ActiveTab)
			fmt.Print("\033[?25h\r\n")
			return 0, "switch"
		case "n", "N":
			state.IssueSelected = selected
			fmt.Print("\033[?25h\r\n")
			return 0, "new"
		case "f", "F":
			state.IssueSelected = selected
			fmt.Print("\033[?25h\r\n")
			return 0, "filters"
		case "\r", "\n":
			state.IssueSelected = selected
			fmt.Print("\033[?25h\r\n")
			return issues[selected].Number, "open"
		case "q", "Q", "b", "B", "\x03", "\x1b":
			state.IssueSelected = selected
			fmt.Print("\033[?25h\r\n")
			return 0, "quit"
		case "r", "R":
			state.IssueSelected = selected
			fmt.Print("\033[?25h\r\n")
			return 0, "refresh"
		case "\x1b[A":
			selected--
			if selected < 0 {
				selected = len(issues) - 1
			}
			render()
		case "\x1b[B":
			selected++
			if selected >= len(issues) {
				selected = 0
			}
			render()
		default:
			if len(key) == 1 && key[0] >= '1' && key[0] <= '9' {
				index := int(key[0] - '1')
				if index >= 0 && index < len(issues) {
					selected = index
					render()
				}
			}
		}
	}
}

func viewIssue(reader *bufio.Reader, state *AppState, number int) {
	for {
		clearScreen()
		renderHeader(state, false)
		issue, err := github.FetchIssue(number)
		if err != nil {
			fmt.Println("Could not fetch issue:")
			fmt.Println(err)
			pause(reader)
			return
		}

		const issueFixedLines = 18
		const issueMenuItems = 9
		budget, cols := bodyBudget(issueFixedLines, issueMenuItems)
		printIssueDetail(issue, budget, cols)

		closeLabel := "Close issue"
		if strings.EqualFold(issue.State, "closed") {
			closeLabel = "Reopen issue"
		}

		choice := menu(reader, []string{
			"Develop branch",
			"Create PR",
			closeLabel,
			"Assign to @me",
			"Add label",
			"Open in browser",
			"Copy URL",
			"Refresh",
			"Back",
		})

		switch choice {
		case "Develop branch":
			clearScreen()
			defaultName := deriveBranchName(issue.Number, issue.Title)
			for {
				name, ok := promptBranchName(reader, defaultName)
				if !ok {
					pause(reader)
					continue
				}
				fmt.Println("Ctrl+C to cancel.")
				if err := github.RunCommandPassthrough("gh", "issue", "develop",
					strconv.Itoa(number), "--checkout", "--name", name); err != nil {
					fmt.Println(err)
					pause(reader)
					break
				}
				return
			}
		case "Create PR":
			clearScreen()
			renderHeader(state, false)
			fmt.Println("Ctrl+C to cancel.")
			if err := github.RunCommandPassthrough("gh", "pr", "create", "--fill"); err != nil {
				fmt.Println(err)
				pause(reader)
			}
			return
		case "Close issue":
			if err := github.CloseIssue(number); err != nil {
				fmt.Println(err)
			} else {
				fmt.Println("Issue closed.")
			}
			pause(reader)
			continue
		case "Reopen issue":
			if err := github.ReopenIssue(number); err != nil {
				fmt.Println(err)
			} else {
				fmt.Println("Issue reopened.")
			}
			pause(reader)
			continue
		case "Assign to @me":
			if err := github.AssignIssueSelf(number); err != nil {
				fmt.Println(err)
			} else {
				fmt.Println("Assigned to @me.")
			}
			pause(reader)
			continue
		case "Add label":
			clearScreen()
			label := strings.TrimSpace(prompt(reader, "Label name: "))
			if label != "" {
				if err := github.AddIssueLabel(number, label); err != nil {
					fmt.Println(err)
				} else {
					fmt.Printf("Label %q added.\n", label)
				}
				pause(reader)
			}
			continue
		case "Open in browser":
			if err := github.RunCommandPassthrough("gh", "issue", "view", strconv.Itoa(number), "--web"); err != nil {
				fmt.Println(err)
				pause(reader)
			}
		case "Copy URL":
			if err := copyText(issue.URL); err != nil {
				fmt.Println("Could not copy URL. Here it is:")
				fmt.Println(issue.URL)
			} else {
				fmt.Println("Copied issue URL.")
			}
			pause(reader)
		case "Refresh":
			continue
		case "Back", "":
			return
		}
	}
}

func configureFilters(reader *bufio.Reader, state *AppState) github.Filters {
	filters := state.IssueFilters

	// Initialize with current values
	stateChoice := filters.State
	assigneeInput := filters.Assignee
	labelInput := filters.Label
	milestoneInput := filters.Milestone
	limitInput := fmt.Sprintf("%d", filters.Limit)
	var clearFilters bool

	form := huh.NewForm(
		huh.NewGroup(
			huh.NewSelect[string]().
				Title("State").
				Options(
					huh.NewOption("open", "open"),
					huh.NewOption("closed", "closed"),
					huh.NewOption("all", "all"),
				).
				Value(&stateChoice),
			huh.NewInput().
				Title("Assignee").
				Placeholder("@me or blank for any").
				Value(&assigneeInput),
			huh.NewInput().
				Title("Label").
				Placeholder("Label name or blank for any").
				Value(&labelInput),
			huh.NewInput().
				Title("Milestone").
				Placeholder("Milestone title or blank for any").
				Value(&milestoneInput),
			huh.NewInput().
				Title("Limit").
				Placeholder(fmt.Sprintf("%d", filters.Limit)).
				Value(&limitInput),
			huh.NewConfirm().
				Title("Clear all filters").
				Value(&clearFilters),
		),
	).WithTheme(huh.ThemeCatppuccin())

	if err := form.Run(); err != nil {
		return filters // Return original on error/cancel
	}

	if clearFilters {
		return github.Filters{State: "open", Limit: 50}
	}

	if stateChoice != "" {
		filters.State = stateChoice
	}
	if assigneeInput != "" {
		filters.Assignee = strings.TrimSpace(assigneeInput)
	}
	if labelInput != "" {
		filters.Label = strings.TrimSpace(labelInput)
	}
	if milestoneInput != "" {
		filters.Milestone = strings.TrimSpace(milestoneInput)
	}
	if limitInput != "" {
		limit, err := strconv.Atoi(limitInput)
		if err == nil && limit > 0 {
			filters.Limit = limit
		}
	}

	return filters
}
