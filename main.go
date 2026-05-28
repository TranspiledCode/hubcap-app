package main

import (
	"bufio"
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"runtime"
	"strconv"
	"strings"
	"text/tabwriter"
)

// ── ANSI Colors ───────────────────────────────────────────────────────────────

const (
	colorReset  = "\033[0m"
	colorGreen  = "\033[32m"
	colorYellow = "\033[33m"
	colorRed    = "\033[31m"
	colorPurple = "\033[35m"
	colorGray   = "\033[90m"
	colorInvert = "\033[7m"
)

type Issue struct {
	Number    int       `json:"number"`
	Title     string    `json:"title"`
	Body      string    `json:"body"`
	URL       string    `json:"url"`
	State     string    `json:"state"`
	Author    User      `json:"author"`
	Assignees []User    `json:"assignees"`
	Labels    []Label   `json:"labels"`
	CreatedAt string    `json:"createdAt"`
}

type User struct {
	Login string `json:"login"`
}

type Label struct {
	Name string `json:"name"`
}

type Filters struct {
	State     string
	Assignee  string
	Label     string
	Milestone string
	Limit     int
}

// ── New Types ─────────────────────────────────────────────────────────────────

type TabID int

const (
	TabIssues TabID = iota
	TabPRs
)

type PRFilters struct {
	State        string // "open", "closed", "merged", "all"
	Assignee     string
	Label        string
	Draft        string // "true" = draft only, "false" = non-draft only, "" = all
	ReviewStatus string // "", "none", "required", "approved", "changes-requested"
	Limit        int
}

type CheckRun struct {
	Status     string `json:"status"`
	Conclusion string `json:"conclusion"`
}

type PullRequest struct {
	Number         int        `json:"number"`
	Title          string     `json:"title"`
	Body           string     `json:"body"`
	State          string     `json:"state"`
	IsDraft        bool       `json:"isDraft"`
	Author         User       `json:"author"`
	Assignees      []User     `json:"assignees"`
	Labels         []Label    `json:"labels"`
	HeadRefName    string     `json:"headRefName"`
	ReviewDecision string     `json:"reviewDecision"`
	StatusRollup   []CheckRun `json:"statusCheckRollup"`
	URL            string     `json:"url"`
	CreatedAt      string     `json:"createdAt"`
}

type AppState struct {
	ActiveTab     TabID
	IssueFilters  Filters
	PRFilters     PRFilters
	IssueSelected int
	PRSelected    int
	Repo          string
}

func main() {
	if err := require("gh"); err != nil {
		fatal(err)
	}

	reader := bufio.NewReader(os.Stdin)

	state := &AppState{
		ActiveTab:    TabIssues,
		IssueFilters: Filters{State: "open", Limit: 50},
		PRFilters:    PRFilters{State: "open", Limit: 50},
	}
	state.Repo = fetchRepo()

	for {
		var action string
		if state.ActiveTab == TabIssues {
			action = browseIssues(reader, state)
		} else {
			action = browsePRs(reader, state)
		}
		if action == "quit" {
			fmt.Println("\nBye.")
			return
		}
	}
}

// ── Pull Requests Tab ────────────────────────────────────────────────────────

func browsePRs(reader *bufio.Reader, state *AppState) string {
	for {
		prs, err := fetchPRs(state.PRFilters)
		if err != nil {
			clearScreen()
			renderHeader(state, false)
			fmt.Println("Error fetching PRs:", err)
			pause(reader)
			return "quit"
		}
		number, action := prList(reader, state, prs)
		switch action {
		case "quit", "back":
			return "quit"
		case "switch":
			return ""
		case "refresh":
			continue
		case "filters":
			state.PRFilters = configurePRFilters(reader, state)
		case "new":
			clearScreen()
			renderHeader(state, false)
			runCommandPassthrough("gh", "pr", "create")
		case "open":
			viewPR(reader, state, number)
		}
	}
}

func prList(reader *bufio.Reader, state *AppState, prs []PullRequest) (int, string) {
	if len(prs) == 0 {
		return 0, "back"
	}

	selected := state.PRSelected
	if selected >= len(prs) {
		selected = 0
	}

	if err := enableRawMode(); err != nil {
		// Fallback: non-raw mode
		clearScreen()
		renderHeader(state, false)

		fmt.Printf("  %-6s %-58s %-12s %-9s %s\n", "#", "TITLE", "AUTHOR", "STATUS", "CHECKS")
		fmt.Printf("  %-6s %-58s %-12s %-9s %s\n", "-----", "-----------------------------------------------------------", "-----------", "--------", "------")
		for _, pr := range prs {
			status := pr.State
			if pr.IsDraft {
				status = "draft"
			}
			fmt.Printf("  %-6d %-58s %-12s %-9s %s\n",
				pr.Number,
				truncate(pr.Title, 58),
				pr.Author.Login,
				status,
				summarizeChecks(pr.StatusRollup),
			)
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
		clearScreen()
		renderHeader(state, true)
		fmt.Print("\033[?25l")

		fmt.Printf("  %-8s %-58s %-12s %-9s %s\r\n", "  #", "TITLE", "AUTHOR", "STATUS", "CHECKS")
		fmt.Printf("  %-8s %-58s %-12s %-9s %s\r\n", "-----", "-----------------------------------------------------------", "-----------", "--------", "------")

		for index, pr := range prs {
			prefix := "  "
			if index == selected {
				prefix = "> "
			}
			indicator := stateIndicator(pr.State, pr.IsDraft)
			status := pr.State
			if pr.IsDraft {
				status = "draft"
			}
			fmt.Printf(
				"%s%s %-6d %-58s %-12s %-9s %s\r\n",
				prefix,
				indicator,
				pr.Number,
				truncate(pr.Title, 58),
				pr.Author.Login,
				status,
				summarizeChecks(pr.StatusRollup),
			)
		}

		fmt.Print("\r\n")
		fmt.Print("↑/↓ navigate • enter open • tab switch tab • n new • f filters • r refresh • q quit\r\n")
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
		case "\t", "\x1b[Z": // Tab / Shift+Tab — switch active tab
			state.PRSelected = selected
			if state.ActiveTab == TabPRs {
				state.ActiveTab = TabIssues
			} else {
				state.ActiveTab = TabPRs
			}
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
			fmt.Print("\033[?25h")
			fmt.Print("\r\n")
			return prs[selected].Number, "open"
		case "q", "Q", "b", "B", "\x03", "\x1b":
			state.PRSelected = selected
			fmt.Print("\033[?25h")
			fmt.Print("\r\n")
			return 0, "quit"
		case "r", "R":
			state.PRSelected = selected
			fmt.Print("\033[?25h")
			fmt.Print("\r\n")
			return 0, "refresh"
		case "\x1b[A": // up arrow
			selected--
			if selected < 0 {
				selected = len(prs) - 1
			}
			render()
		case "\x1b[B": // down arrow
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

func printPRDetail(pr PullRequest) {
	sep := strings.Repeat("=", 80)
	fmt.Printf("#%d %s\n", pr.Number, pr.Title)
	fmt.Println(sep)

	// State with color
	stateColor := colorGreen
	switch {
	case pr.IsDraft:
		stateColor = colorYellow
	case pr.State == "merged":
		stateColor = colorPurple
	case pr.State == "closed":
		stateColor = colorRed
	}
	stateStr := pr.State
	if pr.IsDraft {
		stateStr = "draft"
	}

	fmt.Printf("%-12s %s%s%s\n", "State:", stateColor, stateStr, colorReset)
	fmt.Printf("%-12s %s\n", "Author:", pr.Author.Login)
	fmt.Printf("%-12s %s\n", "Branch:", pr.HeadRefName)

	createdDate := pr.CreatedAt
	if len(pr.CreatedAt) >= 10 {
		createdDate = pr.CreatedAt[:10]
	}
	fmt.Printf("%-12s %s\n", "Created:", createdDate)

	// Assignees
	assigneeStr := "—"
	if len(pr.Assignees) > 0 {
		logins := make([]string, len(pr.Assignees))
		for i, a := range pr.Assignees {
			logins[i] = a.Login
		}
		assigneeStr = strings.Join(logins, ", ")
	}
	fmt.Printf("%-12s %s\n", "Assignees:", assigneeStr)

	// Review decision with color
	reviewColor := colorYellow
	switch pr.ReviewDecision {
	case "APPROVED":
		reviewColor = colorGreen
	case "CHANGES_REQUESTED":
		reviewColor = colorRed
	}
	reviewStr := pr.ReviewDecision
	if reviewStr == "" {
		reviewStr = "—"
	}
	fmt.Printf("%-12s %s%s%s\n", "Review:", reviewColor, reviewStr, colorReset)

	// Checks
	checksStr := summarizeChecks(pr.StatusRollup)
	fmt.Printf("%-12s %s\n", "Checks:", checksStr)

	// Labels
	labelStr := "—"
	if len(pr.Labels) > 0 {
		names := make([]string, len(pr.Labels))
		for i, l := range pr.Labels {
			names[i] = l.Name
		}
		labelStr = strings.Join(names, ", ")
	}
	fmt.Printf("%-12s %s\n", "Labels:", labelStr)
	fmt.Printf("%-12s %s\n", "URL:", pr.URL)
	fmt.Println(sep)

	if pr.Body != "" {
		fmt.Println()
		fmt.Println(pr.Body)
	}
	fmt.Println()
}

func viewPR(reader *bufio.Reader, state *AppState, number int) {
	for {
		pr, err := fetchPR(number)
		if err != nil {
			clearScreen()
			renderHeader(state, false)
			fmt.Println("Error fetching PR:", err)
			pause(reader)
			return
		}
		clearScreen()
		renderHeader(state, false)
		printPRDetail(pr)

		// Build menu
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
			runCommandPassthrough("gh", "pr", "checkout", strconv.Itoa(number))
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
				runCommandPassthrough("gh", "pr", "merge", strconv.Itoa(number), "--merge")
				return
			case "Squash and merge":
				clearScreen()
				renderHeader(state, false)
				runCommandPassthrough("gh", "pr", "merge", strconv.Itoa(number), "--squash")
				return
			case "Rebase and merge":
				clearScreen()
				renderHeader(state, false)
				runCommandPassthrough("gh", "pr", "merge", strconv.Itoa(number), "--rebase")
				return
			case "Cancel", "":
				continue // re-render detail
			}
		case "Close PR":
			if err := closePR(number); err != nil {
				fmt.Println(err)
			} else {
				fmt.Println("PR closed.")
			}
			pause(reader)
			continue
		case "Reopen PR":
			if err := reopenPR(number); err != nil {
				fmt.Println(err)
			} else {
				fmt.Println("PR reopened.")
			}
			pause(reader)
			continue
		case "Open in browser":
			if err := runCommandPassthrough("gh", "pr", "view", strconv.Itoa(number), "--web"); err != nil {
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

func configurePRFilters(reader *bufio.Reader, state *AppState) PRFilters {
	// Task 10 will implement this
	return state.PRFilters
}

func browseIssues(reader *bufio.Reader, state *AppState) string {
	for {
		clearScreen()
		renderHeader(state, false)
		fmt.Println("Fetching issues...")

		issues, err := fetchIssues(state.IssueFilters)
		if err != nil {
			fmt.Println("\nCould not fetch issues:")
			fmt.Println(err)
			pause(reader)
			return ""
		}

		if len(issues) == 0 {
			fmt.Println("\nNo issues found with the current filters.")
			pause(reader)
			return ""
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
			runCommandPassthrough("gh", "issue", "create")
		case "open":
			viewIssue(reader, state, number)
		}
	}
}

func viewIssue(reader *bufio.Reader, state *AppState, number int) {
	for {
		clearScreen()

		issue, err := fetchIssue(number)
		if err != nil {
			fmt.Println("Could not fetch issue:")
			fmt.Println(err)
			pause(reader)
			return
		}

		printIssueDetail(issue)

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
				if err := runCommandPassthrough("gh", "issue", "develop",
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
			if err := runCommandPassthrough("gh", "pr", "create", "--fill"); err != nil {
				fmt.Println(err)
				pause(reader)
			}
			return
		case "Close issue":
			if err := closeIssue(number); err != nil {
				fmt.Println(err)
			} else {
				fmt.Println("Issue closed.")
			}
			pause(reader)
			continue
		case "Reopen issue":
			if err := reopenIssue(number); err != nil {
				fmt.Println(err)
			} else {
				fmt.Println("Issue reopened.")
			}
			pause(reader)
			continue
		case "Assign to @me":
			if err := assignIssueSelf(number); err != nil {
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
				if err := addIssueLabel(number, label); err != nil {
					fmt.Println(err)
				} else {
					fmt.Printf("Label %q added.\n", label)
				}
				pause(reader)
			}
			continue
		case "Open in browser":
			if err := runCommandPassthrough("gh", "issue", "view", strconv.Itoa(number), "--web"); err != nil {
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

func configureFilters(reader *bufio.Reader, state *AppState) Filters {
	filters := state.IssueFilters
	for {
		clearScreen()
		renderHeader(state, false)

		choice := menu(reader, []string{
			"Change state",
			"Change assignee",
			"Change label",
			"Change milestone",
			"Change limit",
			"Clear filters",
			"Back",
		})

		switch choice {
		case "Change state":
			s := menu(reader, []string{"open", "closed", "all", "Back"})
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
		case "Change milestone":
			clearScreen()
			renderHeader(state, false)
			filters.Milestone = strings.TrimSpace(prompt(reader, "Milestone title, or blank for any: "))
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
			filters = Filters{State: "open", Limit: 50}
		case "Back", "":
			return filters
		}
	}
}

func fetchIssues(filters Filters) ([]Issue, error) {
	args := []string{
		"issue", "list",
		"--state", filters.State,
		"--limit", strconv.Itoa(filters.Limit),
		"--json", "number,title,assignees,labels,body,url,state",
	}
	if filters.Assignee != "" {
		args = append(args, "--assignee", filters.Assignee)
	}
	if filters.Label != "" {
		args = append(args, "--label", filters.Label)
	}
	if filters.Milestone != "" {
		args = append(args, "--milestone", filters.Milestone)
	}
	output, err := runCommand("gh", args...)
	if err != nil {
		return nil, err
	}
	var issues []Issue
	if err := json.Unmarshal(output, &issues); err != nil {
		return nil, err
	}
	return issues, nil
}

func fetchIssue(number int) (Issue, error) {
	output, err := runCommand(
		"gh",
		"issue",
		"view",
		strconv.Itoa(number),
		"--json",
		"number,title,body,state,author,assignees,labels,createdAt,url",
	)
	if err != nil {
		return Issue{}, err
	}

	var issue Issue
	if err := json.Unmarshal(output, &issue); err != nil {
		return Issue{}, err
	}

	return issue, nil
}

func fetchRepo() string {
	type repoResponse struct {
		NameWithOwner string `json:"nameWithOwner"`
	}
	output, err := runCommand("gh", "repo", "view", "--json", "nameWithOwner")
	if err != nil {
		return "—"
	}
	var repo repoResponse
	if err := json.Unmarshal(output, &repo); err != nil {
		return "—"
	}
	return repo.NameWithOwner
}

func developIssue(number int) error {
	fmt.Printf("Creating development branch for issue #%d...\n", number)
	return runCommandPassthrough("gh", "issue", "develop", strconv.Itoa(number), "--checkout")
}

func closeIssue(number int) error {
	return runCommandPassthrough("gh", "issue", "close", strconv.Itoa(number))
}

func reopenIssue(number int) error {
	return runCommandPassthrough("gh", "issue", "reopen", strconv.Itoa(number))
}

func assignIssueSelf(number int) error {
	return runCommandPassthrough("gh", "issue", "edit", strconv.Itoa(number), "--add-assignee", "@me")
}

func addIssueLabel(number int, label string) error {
	return runCommandPassthrough("gh", "issue", "edit", strconv.Itoa(number), "--add-label", label)
}

func fetchPRs(filters PRFilters) ([]PullRequest, error) {
	args := []string{
		"pr", "list",
		"--state", filters.State,
		"--limit", strconv.Itoa(filters.Limit),
		"--json", "number,title,author,assignees,labels,state,isDraft,headRefName,statusCheckRollup,url",
	}
	if filters.Assignee != "" {
		args = append(args, "--assignee", filters.Assignee)
	}
	if filters.Label != "" {
		args = append(args, "--label", filters.Label)
	}
	if filters.Draft == "true" {
		args = append(args, "--draft")
	}
	if filters.ReviewStatus != "" {
		args = append(args, "--search", "review:"+filters.ReviewStatus)
	}
	output, err := runCommand("gh", args...)
	if err != nil {
		return nil, err
	}
	var prs []PullRequest
	if err := json.Unmarshal(output, &prs); err != nil {
		return nil, err
	}
	if filters.Draft == "false" {
		prs = filterNonDraftPRs(prs)
	}
	return prs, nil
}

func fetchPR(number int) (PullRequest, error) {
	output, err := runCommand(
		"gh",
		"pr",
		"view",
		strconv.Itoa(number),
		"--json",
		"number,title,body,author,assignees,labels,state,isDraft,headRefName,reviewDecision,statusCheckRollup,url,createdAt",
	)
	if err != nil {
		return PullRequest{}, err
	}
	var pr PullRequest
	if err := json.Unmarshal(output, &pr); err != nil {
		return PullRequest{}, err
	}
	return pr, nil
}

func closePR(number int) error {
	return runCommandPassthrough("gh", "pr", "close", strconv.Itoa(number))
}

func reopenPR(number int) error {
	return runCommandPassthrough("gh", "pr", "reopen", strconv.Itoa(number))
}

func printIssuesTable(issues []Issue) {
	writer := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintln(writer, "NUMBER\tTITLE\tASSIGNEE\tLABELS")
	fmt.Fprintln(writer, "------\t-----\t--------\t------")

	for _, issue := range issues {
		fmt.Fprintf(
			writer,
			"%d\t%s\t%s\t%s\n",
			issue.Number,
			truncate(cleanLine(issue.Title), 58),
			truncate(joinUsers(issue.Assignees), 22),
			truncate(joinLabels(issue.Labels), 34),
		)
	}

	writer.Flush()
}

func issueList(reader *bufio.Reader, state *AppState, issues []Issue) (int, string) {
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
		clearScreen()
		renderHeader(state, true)
		fmt.Print("\033[?25l")

		fmt.Printf("  %-8s %-58s %-22s %-34s\r\n", "  #", "TITLE", "ASSIGNEE", "LABELS")
		fmt.Printf("  %-8s %-58s %-22s %-34s\r\n", "---", "-----", "--------", "------")

		for index, issue := range issues {
			prefix := "  "
			if index == selected {
				prefix = "> "
			}
			indicator := stateIndicator(issue.State, false)
			fmt.Printf(
				"%s%s %-6d %-58s %-22s %-34s\r\n",
				prefix,
				indicator,
				issue.Number,
				truncate(cleanLine(issue.Title), 58),
				truncate(joinUsers(issue.Assignees), 22),
				truncate(joinLabels(issue.Labels), 34),
			)
		}

		fmt.Print("\r\n")
		fmt.Print("↑/↓ navigate • enter open • tab switch tab • n new • f filters • r refresh • q quit\r\n")
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
		case "\t", "\x1b[Z": // Tab / Shift+Tab — switch active tab
			state.IssueSelected = selected
			if state.ActiveTab == TabIssues {
				state.ActiveTab = TabPRs
			} else {
				state.ActiveTab = TabIssues
			}
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
			fmt.Print("\033[?25h")
			fmt.Print("\r\n")
			return issues[selected].Number, "open"
		case "q", "Q", "b", "B", "\x03", "\x1b":
			state.IssueSelected = selected
			fmt.Print("\033[?25h")
			fmt.Print("\r\n")
			return 0, "quit"
		case "r", "R":
			state.IssueSelected = selected
			fmt.Print("\033[?25h")
			fmt.Print("\r\n")
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

func printIssueDetail(issue Issue) {
	fmt.Printf("#%d %s\n", issue.Number, issue.Title)
	fmt.Println(strings.Repeat("=", 80))
	fmt.Printf("State:     %s\n", issue.State)
	fmt.Printf("Author:    %s\n", issue.Author.Login)
	fmt.Printf("Created:   %s\n", strings.TrimSuffix(strings.Split(issue.CreatedAt, "T")[0], "Z"))
	fmt.Printf("Assignees: %s\n", joinUsers(issue.Assignees))
	fmt.Printf("Labels:    %s\n", joinLabels(issue.Labels))
	fmt.Printf("URL:       %s\n", issue.URL)
	fmt.Println(strings.Repeat("=", 80))
	fmt.Println()

	body := strings.TrimSpace(issue.Body)
	if body == "" {
		body = "No description provided."
	}

	fmt.Println(body)
	fmt.Println()
}

func menu(reader *bufio.Reader, options []string) string {
	if len(options) == 0 {
		return ""
	}

	selected := 0

	if err := enableRawMode(); err != nil {
		return numberedMenu(reader, options)
	}
	defer disableRawMode()

	renderMenu := func() {
		fmt.Print("\033[?25l")
		fmt.Print("\033[s")

		for index, option := range options {
			prefix := "  "
			if index == selected {
				prefix = "> "
			}

			fmt.Printf("%s%s\033[K\r\n", prefix, option)
		}

		fmt.Print("\r\n")
		fmt.Print("↑/↓ navigate • enter submit • 1-9 jump • q quit/back\033[K")
		fmt.Print("\033[u")
	}

	renderMenu()
	defer fmt.Print("\033[?25h")

	buffer := make([]byte, 3)

	for {
		n, err := os.Stdin.Read(buffer)
		if err != nil || n == 0 {
			return ""
		}

		key := string(buffer[:n])

		switch key {
		case "\r", "\n":
			fmt.Print("\033[?25h")
			fmt.Print("\r\n")
			return options[selected]
		case "q", "Q", "\x03", "\x1b":
			fmt.Print("\033[?25h")
			fmt.Print("\r\n")
			return ""
		case "\x1b[A":
			selected--
			if selected < 0 {
				selected = len(options) - 1
			}
			renderMenu()
		case "\x1b[B":
			selected++
			if selected >= len(options) {
				selected = 0
			}
			renderMenu()
		default:
			if len(key) == 1 && key[0] >= '1' && key[0] <= '9' {
				index := int(key[0] - '1')
				if index >= 0 && index < len(options) {
					selected = index
					renderMenu()
				}
			}
		}
	}
}

func numberedMenu(reader *bufio.Reader, options []string) string {
	for index, option := range options {
		fmt.Printf("%d) %s\n", index+1, option)
	}

	for {
		input := strings.TrimSpace(prompt(reader, "Choose: "))
		if input == "" {
			return ""
		}

		number, err := strconv.Atoi(input)
		if err != nil || number < 1 || number > len(options) {
			fmt.Println("Enter a number from the menu.")
			continue
		}

		return options[number-1]
	}
}

func enableRawMode() error {
	cmd := exec.Command("stty", "raw", "-echo")
	cmd.Stdin = os.Stdin
	return cmd.Run()
}

func disableRawMode() {
	cmd := exec.Command("stty", "-raw", "echo")
	cmd.Stdin = os.Stdin
	_ = cmd.Run()
}

func prompt(reader *bufio.Reader, label string) string {
	fmt.Print(label)
	value, err := reader.ReadString('\n')
	if err != nil {
		return ""
	}

	return strings.TrimRight(value, "\r\n")
}

func pause(reader *bufio.Reader) {
	fmt.Println()
	fmt.Print("Press Enter to continue...")
	_, _ = reader.ReadString('\n')
}

func promptBranchName(reader *bufio.Reader, defaultName string) (string, bool) {
	fmt.Printf("Branch name [%s]: ", defaultName)
	value, err := reader.ReadString('\n')
	if err != nil {
		return "", false
	}
	value = strings.TrimRight(value, "\r\n")
	if strings.TrimSpace(value) == "" {
		value = defaultName
	}
	if len(value) > 45 {
		fmt.Printf("Name is %d chars (max 45). Try again.\n", len(value))
		return "", false
	}
	return value, true
}

func renderHeader(state *AppState, rawMode bool) {
	nl := "\n"
	if rawMode {
		nl = "\r\n"
	}
	sep := strings.Repeat("=", 52)

	issuesLabel := "  1: Issues  "
	prsLabel := "  2: Pull Requests  "
	if state.ActiveTab == TabIssues {
		issuesLabel = colorInvert + issuesLabel + colorReset
	} else {
		prsLabel = colorInvert + prsLabel + colorReset
	}

	fmt.Printf("GitHub TUI — %s%s", state.Repo, nl)
	fmt.Printf("%s%s", sep, nl)
	fmt.Printf("%s%s%s", issuesLabel, prsLabel, nl)
	fmt.Printf("%s%s", sep, nl)

	if state.ActiveTab == TabIssues {
		f := state.IssueFilters
		fmt.Printf("State: %s | Assignee: %s | Label: %s | Limit: %d%s",
			f.State, displayAny(f.Assignee), displayAny(f.Label), f.Limit, nl)
	} else {
		f := state.PRFilters
		fmt.Printf("State: %s | Assignee: %s | Label: %s | Limit: %d%s",
			f.State, displayAny(f.Assignee), displayAny(f.Label), f.Limit, nl)
	}
	fmt.Printf("%s%s", sep, nl)
	fmt.Print(nl)
}

func require(name string) error {
	_, err := exec.LookPath(name)
	if err != nil {
		return fmt.Errorf("%s is required but was not found in PATH", name)
	}

	return nil
}

func runCommand(name string, args ...string) ([]byte, error) {
	cmd := exec.Command(name, args...)
	cmd.Stdin = os.Stdin

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		message := strings.TrimSpace(stderr.String())
		if message == "" {
			message = err.Error()
		}
		return nil, errors.New(message)
	}

	return stdout.Bytes(), nil
}

func runCommandPassthrough(name string, args ...string) error {
	cmd := exec.Command(name, args...)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

func copyText(text string) error {
	switch runtime.GOOS {
	case "darwin":
		cmd := exec.Command("pbcopy")
		cmd.Stdin = strings.NewReader(text)
		return cmd.Run()
	case "linux":
		if _, err := exec.LookPath("wl-copy"); err == nil {
			cmd := exec.Command("wl-copy")
			cmd.Stdin = strings.NewReader(text)
			return cmd.Run()
		}

		if _, err := exec.LookPath("xclip"); err == nil {
			cmd := exec.Command("xclip", "-selection", "clipboard")
			cmd.Stdin = strings.NewReader(text)
			return cmd.Run()
		}
	}

	return errors.New("no clipboard command available")
}

func clearScreen() {
	fmt.Print("\033[H\033[2J")
}

func displayAny(value string) string {
	if strings.TrimSpace(value) == "" {
		return "any"
	}

	return value
}

func joinUsers(users []User) string {
	if len(users) == 0 {
		return "Unassigned"
	}

	values := make([]string, 0, len(users))
	for _, user := range users {
		values = append(values, user.Login)
	}

	return strings.Join(values, ", ")
}

func joinLabels(labels []Label) string {
	if len(labels) == 0 {
		return "-"
	}

	values := make([]string, 0, len(labels))
	for _, label := range labels {
		values = append(values, label.Name)
	}

	return strings.Join(values, ", ")
}

func cleanLine(value string) string {
	value = strings.ReplaceAll(value, "\n", " ")
	value = strings.ReplaceAll(value, "\t", " ")
	return strings.Join(strings.Fields(value), " ")
}

func truncate(value string, max int) string {
	if max <= 0 {
		return ""
	}

	runes := []rune(value)
	if len(runes) <= max {
		return value
	}

	if max <= 1 {
		return string(runes[:max])
	}

	return string(runes[:max-1]) + "…"
}

func fatal(err error) {
	fmt.Fprintln(os.Stderr, err)
	os.Exit(1)
}

// ── String/Display Utilities ──────────────────────────────────────────────────

func filterNonDraftPRs(prs []PullRequest) []PullRequest {
	var out []PullRequest
	for _, pr := range prs {
		if !pr.IsDraft {
			out = append(out, pr)
		}
	}
	return out
}

func deriveBranchName(number int, title string) string {
	title = strings.ToLower(title)
	var sb strings.Builder
	for _, r := range title {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') {
			sb.WriteRune(r)
		} else {
			sb.WriteRune('-')
		}
	}
	slug := sb.String()
	for strings.Contains(slug, "--") {
		slug = strings.ReplaceAll(slug, "--", "-")
	}
	slug = strings.Trim(slug, "-")
	prefix := fmt.Sprintf("%d-", number)
	if slug == "" {
		return strings.TrimRight(prefix, "-")
	}
	full := prefix + slug
	if len(full) > 45 {
		full = full[:45]
		full = strings.TrimRight(full, "-")
	}
	return full
}

func summarizeChecks(checks []CheckRun) string {
	if len(checks) == 0 {
		return "—"
	}
	pending := false
	for _, c := range checks {
		if c.Conclusion == "FAILURE" || c.Conclusion == "ERROR" || c.Conclusion == "TIMED_OUT" {
			return colorRed + "✗" + colorReset
		}
		if c.Status != "COMPLETED" {
			pending = true
		}
	}
	if pending {
		return colorYellow + "…" + colorReset
	}
	return colorGreen + "✓" + colorReset
}

func stateIndicator(state string, isDraft bool) string {
	switch {
	case isDraft:
		return colorYellow + "◐" + colorReset
	case strings.EqualFold(state, "merged"):
		return colorPurple + "✓" + colorReset
	case strings.EqualFold(state, "closed"):
		return colorRed + "✗" + colorReset
	case strings.EqualFold(state, "open"):
		return colorGreen + "●" + colorReset
	default:
		return colorGray + "○" + colorReset
	}
}
