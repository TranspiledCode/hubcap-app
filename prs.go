// prs.go
package main

import (
	"bufio"
	"fmt"
	"os"
	"strconv"
	"strings"

	"hubcap/internal/github"
)

func browsePRs(reader *bufio.Reader, state *AppState) string {
	for {
		clearScreen()
		renderHeader(state, false)
		fmt.Println("Fetching pull requests...")

		prs, err := github.FetchPRs(state.PRFilters)
		if err != nil {
			fmt.Println("\nCould not fetch pull requests:")
			fmt.Println(err)
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
				prefix = colorSelect + ">" + colorReset + " "
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
			fmt.Println("Error fetching PR:", err)
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
				clearScreen()
				renderHeader(state, false)
				fmt.Println("Ctrl+C to cancel.")
				github.RunCommandPassthrough("gh", "pr", "merge", strconv.Itoa(number), "--merge")
				return
			case "Squash and merge":
				clearScreen()
				renderHeader(state, false)
				fmt.Println("Ctrl+C to cancel.")
				github.RunCommandPassthrough("gh", "pr", "merge", strconv.Itoa(number), "--squash")
				return
			case "Rebase and merge":
				clearScreen()
				renderHeader(state, false)
				fmt.Println("Ctrl+C to cancel.")
				github.RunCommandPassthrough("gh", "pr", "merge", strconv.Itoa(number), "--rebase")
				return
			case "Cancel", "":
				continue
			}
		case "Close PR":
			if err := github.ClosePR(number); err != nil {
				fmt.Println(err)
			} else {
				fmt.Println("PR closed.")
			}
			pause(reader)
			continue
		case "Reopen PR":
			if err := github.ReopenPR(number); err != nil {
				fmt.Println(err)
			} else {
				fmt.Println("PR reopened.")
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
				fmt.Println("Could not copy URL. Here it is:")
				fmt.Println(pr.URL)
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
	for {
		clearScreen()
		renderHeader(state, false)

		choice := menu(reader, []string{
			"Change state",
			"Change assignee",
			"Change label",
			"Change draft",
			"Change review status",
			"Change limit",
			"Clear filters",
			"Back",
		})

		switch choice {
		case "Change state":
			s := menu(reader, []string{"open", "closed", "merged", "all", "Back"})
			if s != "" && s != "Back" {
				filters.State = s
			}
		case "Change assignee":
			clearScreen()
			renderHeader(state, false)
			filters.Assignee = strings.TrimSpace(prompt(reader, "Assignee, @me, or blank for any: "))
		case "Change label":
			clearScreen()
			renderHeader(state, false)
			filters.Label = strings.TrimSpace(prompt(reader, "Label name, or blank for any: "))
		case "Change draft":
			d := menu(reader, []string{"all", "draft only", "non-draft only", "Back"})
			switch d {
			case "all":
				filters.Draft = ""
			case "draft only":
				filters.Draft = "true"
			case "non-draft only":
				filters.Draft = "false"
			}
		case "Change review status":
			clearScreen()
			renderHeader(state, false)
			filters.ReviewStatus = strings.TrimSpace(prompt(reader, "Review status (approved, changes-requested, etc.), or blank for any: "))
		case "Change limit":
			clearScreen()
			renderHeader(state, false)
			value := strings.TrimSpace(prompt(reader, fmt.Sprintf("Limit [%d]: ", filters.Limit)))
			if value == "" {
				continue
			}
			limit, err := strconv.Atoi(value)
			if err != nil || limit <= 0 {
				fmt.Println("Limit must be a positive number.")
				pause(reader)
				continue
			}
			filters.Limit = limit
		case "Clear filters":
			filters = github.PRFilters{State: "open", Limit: 50}
		case "Back", "":
			return filters
		}
	}
}
