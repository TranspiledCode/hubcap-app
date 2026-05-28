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
	filters := Filters{
		State: "open",
		Limit: 50,
	}

	if err := require("gh"); err != nil {
		fatal(err)
	}

	reader := bufio.NewReader(os.Stdin)

	for {
		clearScreen()
		printHeader(filters)

		choice := menu(reader, []string{
			"Browse issues",
			"Filters",
			"Refresh",
			"Quit",
		})

		switch choice {
		case "Browse issues", "Refresh":
			browseIssues(reader, filters)
		case "Filters":
			filters = configureFilters(reader, filters)
		case "Quit", "":
			fmt.Println("Bye.")
			return
		}
	}
}

func browseIssues(reader *bufio.Reader, filters Filters) {
	for {
		clearScreen()
		printHeader(filters)
		fmt.Println("Fetching issues...")

		issues, err := fetchIssues(filters)
		if err != nil {
			fmt.Println()
			fmt.Println("Could not fetch issues:")
			fmt.Println(err)
			pause(reader)
			return
		}

		if len(issues) == 0 {
			fmt.Println()
			fmt.Println("No issues found with the current filters.")
			pause(reader)
			return
		}

		number, action := issueList(reader, filters, issues)
		switch action {
		case "back":
			return
		case "refresh":
			continue
		case "open":
			viewIssue(reader, number)
		}
	}
}

func viewIssue(reader *bufio.Reader, number int) {
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

		choice := menu(reader, []string{
			"Develop branch",
			"Open in browser",
			"Copy URL",
			"Refresh issue",
			"Back to list",
			"Quit",
		})

		switch choice {
		case "Develop branch":
			if err := developIssue(number); err != nil {
				fmt.Println(err)
				pause(reader)
				continue
			}
			return
		case "Open in browser":
			if err := runCommandPassthrough("gh", "issue", "view", strconv.Itoa(number), "--web"); err != nil {
				fmt.Println(err)
				pause(reader)
			}
		case "Copy URL":
			if err := copyText(issue.URL); err != nil {
				fmt.Println("Could not copy URL automatically. Here it is:")
				fmt.Println(issue.URL)
				fmt.Println(err)
			} else {
				fmt.Println("Copied issue URL.")
			}
			pause(reader)
		case "Refresh issue":
			continue
		case "Back to list", "":
			return
		case "Quit":
			os.Exit(0)
		}
	}
}

func configureFilters(reader *bufio.Reader, filters Filters) Filters {
	for {
		clearScreen()
		printHeader(filters)

		choice := menu(reader, []string{
			"Change state",
			"Change assignee",
			"Change label",
			"Change limit",
			"Clear filters",
			"Back",
		})

		switch choice {
		case "Change state":
			state := menu(reader, []string{"open", "closed", "all", "Back"})
			if state != "" && state != "Back" {
				filters.State = state
			}
		case "Change assignee":
			clearScreen()
			printHeader(filters)
			value := prompt(reader, "Assignee, @me, or blank for any: ")
			filters.Assignee = strings.TrimSpace(value)
		case "Change label":
			clearScreen()
			printHeader(filters)
			value := prompt(reader, "Label name, or blank for any: ")
			filters.Label = strings.TrimSpace(value)
		case "Change limit":
			clearScreen()
			printHeader(filters)
			value := prompt(reader, fmt.Sprintf("Limit [%d]: ", filters.Limit))
			value = strings.TrimSpace(value)
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
		"issue",
		"list",
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

func printHeader(filters Filters) {
	fmt.Println("GitHub Issues")
	fmt.Println(strings.Repeat("=", 48))
	fmt.Printf("State:    %s\n", filters.State)
	fmt.Printf("Assignee: %s\n", displayAny(filters.Assignee))
	fmt.Printf("Label:    %s\n", displayAny(filters.Label))
	fmt.Printf("Limit:    %d\n", filters.Limit)
	fmt.Println(strings.Repeat("=", 48))
	fmt.Println()
}

func printHeaderRaw(filters Filters) {
	fmt.Print("GitHub Issues\r\n")
	fmt.Printf("%s\r\n", strings.Repeat("=", 48))
	fmt.Printf("State:    %s\r\n", filters.State)
	fmt.Printf("Assignee: %s\r\n", displayAny(filters.Assignee))
	fmt.Printf("Label:    %s\r\n", displayAny(filters.Label))
	fmt.Printf("Limit:    %d\r\n", filters.Limit)
	fmt.Printf("%s\r\n", strings.Repeat("=", 48))
	fmt.Print("\r\n")
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

func issueList(reader *bufio.Reader, filters Filters, issues []Issue) (int, string) {
	if len(issues) == 0 {
		return 0, "back"
	}

	selected := 0

	if err := enableRawMode(); err != nil {
		clearScreen()
		printHeader(filters)
		printIssuesTable(issues)

		fmt.Println()
		input := prompt(reader, "Issue number to open, r to refresh, b to go back: ")
		input = strings.TrimSpace(strings.ToLower(input))

		switch input {
		case "", "b", "back", "q", "quit":
			return 0, "back"
		case "r", "refresh":
			return 0, "refresh"
		}

		number, err := strconv.Atoi(input)
		if err != nil {
			return 0, "back"
		}

		return number, "open"
	}
	defer disableRawMode()
	defer fmt.Print("\033[?25h")

	render := func() {
		clearScreen()
		printHeaderRaw(filters)
		fmt.Print("\033[?25l")

		fmt.Printf("  %-7s %-58s %-22s %-34s\r\n", "NUMBER", "TITLE", "ASSIGNEE", "LABELS")
		fmt.Printf("  %-7s %-58s %-22s %-34s\r\n", "------", "-----", "--------", "------")

		for index, issue := range issues {
			prefix := "  "
			if index == selected {
				prefix = "> "
			}

			fmt.Printf(
				"%s%-7d %-58s %-22s %-34s\r\n",
				prefix,
				issue.Number,
				truncate(cleanLine(issue.Title), 58),
				truncate(joinUsers(issue.Assignees), 22),
				truncate(joinLabels(issue.Labels), 34),
			)
		}

		fmt.Print("\r\n")
		fmt.Print("↑/↓ navigate • enter open • r refresh • q back • 1-9 jump\r\n")
	}

	render()

	buffer := make([]byte, 3)

	for {
		n, err := os.Stdin.Read(buffer)
		if err != nil || n == 0 {
			return 0, "back"
		}

		key := string(buffer[:n])

		switch key {
		case "\r", "\n":
			fmt.Print("\033[?25h")
			fmt.Print("\r\n")
			return issues[selected].Number, "open"
		case "q", "Q", "b", "B", "\x03", "\x1b":
			fmt.Print("\033[?25h")
			fmt.Print("\r\n")
			return 0, "back"
		case "r", "R":
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
