// prs.go
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

func browsePRs(reader *bufio.Reader, state *AppState) string {
	for {
		clearScreen()
		renderHeader(state, false)
		fmt.Println("Fetching pull requests...")

		prs, err := github.FetchPRs(state.PRFilters)
		if err != nil {
			fmt.Println()
			fmt.Println(errorBox(fmt.Sprintf("Could not fetch pull requests:\n%v", err)))
			fmt.Println("  Tab — switch tabs  |  r — retry  |  q — quit")
			action := emptyTabAction(reader, state, TabDashboard)
			switch action {
			case "quit":
				return "quit"
			case "switch":
				return ""
			}
			continue
		}

		if len(prs) == 0 {
			fmt.Println("\nNo pull requests found with the current filters.")
			fmt.Println("  Tab — switch tabs  |  f — change filters  |  q — quit")
			action := emptyTabAction(reader, state, TabDashboard)
			switch action {
			case "quit":
				return "quit"
			case "switch":
				return ""
			case "filters":
				state.PRFilters = configurePRFilters(reader, state)
			}
			continue
		}

		number, action := prList(reader, state, prs)
		switch action {
		case "quit":
			return "quit"
		case "back":
			continue
		case "switch":
			return ""
		case "refresh":
			continue
		case "filters":
			state.PRFilters = configurePRFilters(reader, state)
		case "new":
			clearScreen()
			renderHeader(state, false)
			fmt.Println("Ctrl+C to cancel.")
			github.RunCommandPassthrough("gh", "pr", "create")
		case "open":
			viewPR(reader, state, number)
		}
	}
}

func prList(reader *bufio.Reader, state *AppState, prs []github.PullRequest) (int, string) {
	if len(prs) == 0 {
		return 0, "back"
	}

	selected := state.PRSelected
	if selected >= len(prs) {
		selected = 0
	}

	if err := enableRawMode(); err != nil {
		clearScreen()
		renderHeader(state, false)
		fmt.Printf("  %-6s %-58s %-12s %-9s %s\n", "#", "TITLE", "AUTHOR", "STATUS", "CHECKS")
		fmt.Printf("  %-6s %-58s %-12s %-9s %s\n", "-----", strings.Repeat("-", 58), "-----------", "--------", "------")
		for _, pr := range prs {
			status := pr.State
			if pr.IsDraft {
				status = "draft"
			}
			fmt.Printf("  %-6d %-58s %-12s %-9s %s\n",
				pr.Number, truncate(pr.Title, 58), pr.Author.Login, status, summarizeChecks(pr.StatusRollup))
		}
		fmt.Println()
		input := prompt(reader, "PR number, n new, f filters, r refresh, q quit: ")
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
		fmt.Printf("  %-8s %-58s %-12s %-9s %s\033[K\r\n", "  #", "TITLE", "AUTHOR", "STATUS", "CHECKS")
		fmt.Printf("  %-8s %-58s %-12s %-9s %s\033[K\r\n", "-----", strings.Repeat("-", 58), "-----------", "--------", "------")
		for index, pr := range prs {
			prefix := "  "
			if index == selected {
				prefix = styleCyan.Render(">") + " "
			}
			indicator := stateIndicator(pr.State, pr.IsDraft)
			status := pr.State
			if pr.IsDraft {
				status = "draft"
			}
			fmt.Printf("%s%s %-6d %-58s %-12s %-9s %s\033[K\r\n",
				prefix, indicator, pr.Number, truncate(pr.Title, 58),
				pr.Author.Login, status, summarizeChecks(pr.StatusRollup))
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
			state.PRSelected = selected
			state.ActiveTab = nextTab(state.ActiveTab)
			fmt.Print("\033[?25h\r\n")
			return 0, "switch"
		case "n", "N":
			state.PRSelected = selected
			fmt.Print("\033[?25h\r\n")
			return 0, "new"
		case "f", "F":
			state.PRSelected = selected
			fmt.Print("\033[?25h\r\n")
			return 0, "filters"
		case "\r", "\n":
			state.PRSelected = selected
			fmt.Print("\033[?25h\r\n")
			return prs[selected].Number, "open"
		case "q", "Q", "b", "B", "\x03", "\x1b":
			state.PRSelected = selected
			fmt.Print("\033[?25h\r\n")
			return 0, "quit"
		case "r", "R":
			state.PRSelected = selected
			fmt.Print("\033[?25h\r\n")
			return 0, "refresh"
		case "\x1b[A":
			selected--
			if selected < 0 {
				selected = len(prs) - 1
			}
			render()
		case "\x1b[B":
			selected++
			if selected >= len(prs) {
				selected = 0
			}
			render()
		default:
			if len(key) == 1 && key[0] >= '1' && key[0] <= '9' {
				index := int(key[0] - '1')
				if index >= 0 && index < len(prs) {
					selected = index
					render()
				}
			}
		}
	}
}

func viewPR(reader *bufio.Reader, state *AppState, number int) {
	for {
		clearScreen()
		renderHeader(state, false)
		pr, err := github.FetchPR(number)
		if err != nil {
			fmt.Println(errorBox(fmt.Sprintf("Could not fetch PR:\n%v", err)))
			pause(reader)
			return
		}
		const prFixedLines = 21
		const prMenuItems = 7
		budget, cols := bodyBudget(prFixedLines, prMenuItems)
		printPRDetail(pr, budget, cols)

		closeLabel := "Close PR"
		if pr.State == "closed" {
			closeLabel = "Reopen PR"
		}

		choice := menu(reader, []string{
			"Checkout branch",
			"Merge",
			closeLabel,
			"Open in browser",
			"Copy URL",
			"Refresh",
			"Back",
		})

		switch choice {
		case "Checkout branch":
			clearScreen()
			renderHeader(state, false)
			fmt.Println("Ctrl+C to cancel.")
			github.RunCommandPassthrough("gh", "pr", "checkout", strconv.Itoa(number))
			return
		case "Merge":
			mergeChoice := menu(reader, []string{
				"Merge commit",
				"Squash and merge",
				"Rebase and merge",
				"Cancel",
			})
			switch mergeChoice {
			case "Merge commit":
				if !confirmAction(fmt.Sprintf("Merge PR #%d (merge commit)?", number),
					"This will create a merge commit on the base branch.", "Merge") {
					continue
				}
				clearScreen()
				renderHeader(state, false)
				fmt.Println("Ctrl+C to cancel.")
				github.RunCommandPassthrough("gh", "pr", "merge", strconv.Itoa(number), "--merge")
				return
			case "Squash and merge":
				if !confirmAction(fmt.Sprintf("Squash and merge PR #%d?", number),
					"This will squash all commits into a single commit on the base branch.", "Squash & merge") {
					continue
				}
				clearScreen()
				renderHeader(state, false)
				fmt.Println("Ctrl+C to cancel.")
				github.RunCommandPassthrough("gh", "pr", "merge", strconv.Itoa(number), "--squash")
				return
			case "Rebase and merge":
				if !confirmAction(fmt.Sprintf("Rebase and merge PR #%d?", number),
					"This will rebase commits onto the base branch.", "Rebase & merge") {
					continue
				}
				clearScreen()
				renderHeader(state, false)
				fmt.Println("Ctrl+C to cancel.")
				github.RunCommandPassthrough("gh", "pr", "merge", strconv.Itoa(number), "--rebase")
				return
			case "Cancel", "":
				continue
			}
		case "Close PR":
			if !confirmAction(
				fmt.Sprintf("Close PR #%d?", number),
				"This will close the pull request without merging.",
				"Close",
			) {
				continue
			}
			if err := github.ClosePR(number); err != nil {
				fmt.Println(errorBox(err.Error()))
			} else {
				fmt.Println(successBox("PR closed."))
			}
			pause(reader)
			continue
		case "Reopen PR":
			if err := github.ReopenPR(number); err != nil {
				fmt.Println(errorBox(err.Error()))
			} else {
				fmt.Println(successBox("PR reopened."))
			}
			pause(reader)
			continue
		case "Open in browser":
			if err := github.RunCommandPassthrough("gh", "pr", "view", strconv.Itoa(number), "--web"); err != nil {
				fmt.Println(err)
				pause(reader)
			}
			continue
		case "Copy URL":
			if err := copyText(pr.URL); err != nil {
				fmt.Println(warningBox(fmt.Sprintf("Could not copy URL. Here it is:\n%s", pr.URL)))
			} else {
				fmt.Println("Copied PR URL.")
			}
			pause(reader)
			continue
		case "Refresh":
			continue
		case "Back", "":
			return
		}
	}
}

func configurePRFilters(reader *bufio.Reader, state *AppState) github.PRFilters {
	filters := state.PRFilters

	// Initialize with current values
	stateChoice := filters.State
	availableAssignees, _ := github.FetchAssignees()
	assigneeChoice := assigneeToChoice(filters.Assignee, availableAssignees)
	assigneeCustom := ""
	if assigneeChoice == "custom" {
		assigneeCustom = filters.Assignee
	}
	labelInput := filters.Label
	selectedLabels := splitCSV(filters.Label)
	draftChoice := filters.Draft
	reviewStatusInput := filters.ReviewStatus
	limitInput := fmt.Sprintf("%d", filters.Limit)
	actionChoice := "save"

	availableLabels, _ := github.FetchLabels()

	groupFields := []huh.Field{
		huh.NewSelect[string]().
			Title("State").
			Options(
				huh.NewOption("open", "open"),
				huh.NewOption("closed", "closed"),
				huh.NewOption("merged", "merged"),
				huh.NewOption("all", "all"),
			).
			Value(&stateChoice),
		huh.NewSelect[string]().
			Title("Assignee").
			Options(assigneeOptions(availableAssignees)...).
			Value(&assigneeChoice),
		huh.NewInput().
			Title("Custom assignee").
			Placeholder("GitHub username").
			Value(&assigneeCustom).
			DescriptionFunc(func() string {
				if assigneeChoice == "custom" {
					return "Required when Custom is selected."
				}
				return ""
			}, &assigneeChoice),
	}

	if len(availableLabels) > 0 {
		labelOptions := make([]huh.Option[string], 0, len(availableLabels))
		for _, name := range availableLabels {
			labelOptions = append(labelOptions, huh.NewOption(name, name))
		}
		height := len(labelOptions)
		if height > 8 {
			height = 8 // Limit visible rows for long lists
		}
		groupFields = append(groupFields,
			huh.NewMultiSelect[string]().
				Title("Labels").
				Description("Space to toggle. Matches PRs with ALL selected labels.").
				Options(labelOptions...).
				Height(height).
				Value(&selectedLabels),
		)
	} else {
		groupFields = append(groupFields,
			huh.NewInput().
				Title("Label").
				Placeholder("Label name (comma-separated) or blank for any").
				Value(&labelInput),
		)
	}

	groupFields = append(groupFields,
		huh.NewSelect[string]().
			Title("Draft").
			Options(
				huh.NewOption("all", ""),
				huh.NewOption("draft only", "true"),
				huh.NewOption("non-draft only", "false"),
			).
			Value(&draftChoice),
		huh.NewInput().
			Title("Review status").
			Placeholder("approved, changes-requested, etc.").
			Value(&reviewStatusInput),
		huh.NewInput().
			Title("Limit").
			Placeholder(fmt.Sprintf("%d", filters.Limit)).
			Value(&limitInput),
		huh.NewSelect[string]().
			Title("Action").
			Options(
				huh.NewOption("Save filters", "save"),
				huh.NewOption("Reset to defaults", "reset"),
			).
			Value(&actionChoice),
	)

	form := huh.NewForm(huh.NewGroup(groupFields...)).WithTheme(huh.ThemeCatppuccin())

	if err := form.Run(); err != nil {
		return filters // Return original on error/cancel
	}

	if actionChoice == "reset" {
		return github.PRFilters{State: "open", Limit: 50}
	}

	if stateChoice != "" {
		filters.State = stateChoice
	}
	filters.Assignee = resolveAssignee(assigneeChoice, assigneeCustom)
	if len(availableLabels) > 0 {
		filters.Label = strings.Join(selectedLabels, ",")
	} else if labelInput != "" {
		filters.Label = strings.TrimSpace(labelInput)
	}
	if draftChoice != "" {
		filters.Draft = draftChoice
	}
	if reviewStatusInput != "" {
		filters.ReviewStatus = strings.TrimSpace(reviewStatusInput)
	}
	if limitInput != "" {
		limit, err := strconv.Atoi(limitInput)
		if err == nil && limit > 0 {
			filters.Limit = limit
		}
	}

	return filters
}
