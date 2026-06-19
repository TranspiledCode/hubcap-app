# hubcap — My Work Dashboard Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Split the monolithic `main.go` into focused files, move all `gh` CLI calls into `internal/github/`, add a config system, and build a fully-navigable "My Work" dashboard tab as the default landing screen.

**Architecture:** Phase 1 is a pure refactor — code moves, zero behaviour change, verified by `go build` and `go test ./...` after each task. Phase 2 adds `config.go` (load/save `~/.config/hubcap/config.json`). Phase 3 builds the dashboard (`dashboard.go`) with concurrent section fetching and flat keyboard-navigable rows. All new logic is TDD where testable.

**Tech Stack:** Go 1.21 stdlib only, `gh` CLI subprocess, `stty` raw terminal mode, `~/.config/hubcap/config.json`.

---

## File Map

| File | Responsibility |
|---|---|
| `main.go` | `TabID`, `AppState`, `main()` only |
| `dashboard.go` | My Work tab: data types, concurrent fetch, render, nav loop, config screen |
| `issues.go` | Issues tab: `browseIssues`, `issueList`, `viewIssue`, `configureFilters` |
| `prs.go` | PRs tab: `browsePRs`, `prList`, `viewPR`, `configurePRFilters` |
| `terminal.go` | Raw mode, `menu`, `prompt`, `clearScreen`, `termSize`, `clipboard`, `emptyTabAction` |
| `render.go` | `renderHeader`, display helpers (`truncate`, `joinUsers`, `summarizeChecks`, etc.) |
| `config.go` | `Config` struct, `loadConfig`, `saveConfig` |
| `internal/github/repo.go` | All types + `RunCommand`, `RunCommandPassthrough`, `FetchRepo` |
| `internal/github/issues.go` | `FetchIssues`, `FetchIssue`, `CloseIssue`, `ReopenIssue`, `AssignIssueSelf`, `AddIssueLabel` |
| `internal/github/prs.go` | `FetchPRs`, `FetchPR`, `FetchReviewRequests`, `ClosePR`, `ReopenPR`, `FilterNonDraftPRs` |

---

## Phase 1 — Refactor (zero behaviour change)

---

### Task 1: Create `internal/github/repo.go` — types + core infrastructure

**Files:**
- Create: `internal/github/repo.go`

- [ ] **Step 1: Create the file**

```go
// internal/github/repo.go
package github

import (
	"bytes"
	"encoding/json"
	"errors"
	"os"
	"os/exec"
	"strings"
)

type User struct {
	Login string `json:"login"`
}

type Label struct {
	Name string `json:"name"`
}

type CheckRun struct {
	Status     string `json:"status"`
	Conclusion string `json:"conclusion"`
}

type Issue struct {
	Number    int     `json:"number"`
	Title     string  `json:"title"`
	Body      string  `json:"body"`
	URL       string  `json:"url"`
	State     string  `json:"state"`
	Author    User    `json:"author"`
	Assignees []User  `json:"assignees"`
	Labels    []Label `json:"labels"`
	CreatedAt string  `json:"createdAt"`
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

type Filters struct {
	State     string
	Assignee  string
	Label     string
	Milestone string
	Limit     int
}

// PRFilters extends the original with Author and Search for dashboard queries.
type PRFilters struct {
	State        string
	Author       string
	Assignee     string
	Label        string
	Draft        string // "true" = draft only, "false" = non-draft only, "" = all
	ReviewStatus string // used by filter UI: maps to "review:<value>"
	Search       string // raw --search value; takes precedence over ReviewStatus
	Limit        int
}

func FetchRepo() string {
	type repoResponse struct {
		NameWithOwner string `json:"nameWithOwner"`
	}
	output, err := RunCommand("gh", "repo", "view", "--json", "nameWithOwner")
	if err != nil {
		return "—"
	}
	var repo repoResponse
	if err := json.Unmarshal(output, &repo); err != nil {
		return "—"
	}
	return repo.NameWithOwner
}

func RunCommand(name string, args ...string) ([]byte, error) {
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

func RunCommandPassthrough(name string, args ...string) error {
	cmd := exec.Command(name, args...)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}
```

- [ ] **Step 2: Verify it compiles**

```bash
go build ./internal/github/...
```

Expected: no output (success).

- [ ] **Step 3: Commit**

```bash
git add internal/github/repo.go
git commit -m "refactor: add internal/github package with core types and infrastructure"
```

---

### Task 2: Create `internal/github/issues.go`

**Files:**
- Create: `internal/github/issues.go`

- [ ] **Step 1: Create the file**

```go
// internal/github/issues.go
package github

import (
	"encoding/json"
	"strconv"
)

func FetchIssues(filters Filters) ([]Issue, error) {
	args := []string{
		"issue", "list",
		"--state", filters.State,
		"--limit", strconv.Itoa(filters.Limit),
		"--json", "number,title,assignees,labels,body,url,state,createdAt",
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
	output, err := RunCommand("gh", args...)
	if err != nil {
		return nil, err
	}
	return parseIssues(output)
}

func FetchIssue(number int) (Issue, error) {
	output, err := RunCommand(
		"gh", "issue", "view", strconv.Itoa(number),
		"--json", "number,title,body,state,author,assignees,labels,createdAt,url",
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

func CloseIssue(number int) error {
	return RunCommandPassthrough("gh", "issue", "close", strconv.Itoa(number))
}

func ReopenIssue(number int) error {
	return RunCommandPassthrough("gh", "issue", "reopen", strconv.Itoa(number))
}

func AssignIssueSelf(number int) error {
	return RunCommandPassthrough("gh", "issue", "edit", strconv.Itoa(number), "--add-assignee", "@me")
}

func AddIssueLabel(number int, label string) error {
	return RunCommandPassthrough("gh", "issue", "edit", strconv.Itoa(number), "--add-label", label)
}

func parseIssues(data []byte) ([]Issue, error) {
	var issues []Issue
	return issues, json.Unmarshal(data, &issues)
}
```

- [ ] **Step 2: Verify it compiles**

```bash
go build ./internal/github/...
```

Expected: no output (success).

- [ ] **Step 3: Commit**

```bash
git add internal/github/issues.go
git commit -m "refactor: add internal/github issues operations"
```

---

### Task 3: Create `internal/github/prs.go`

**Files:**
- Create: `internal/github/prs.go`

- [ ] **Step 1: Create the file**

```go
// internal/github/prs.go
package github

import (
	"encoding/json"
	"strconv"
)

func FetchPRs(filters PRFilters) ([]PullRequest, error) {
	args := []string{
		"pr", "list",
		"--state", filters.State,
		"--limit", strconv.Itoa(filters.Limit),
		"--json", "number,title,author,assignees,labels,state,isDraft,headRefName,statusCheckRollup,url",
	}
	if filters.Author != "" {
		args = append(args, "--author", filters.Author)
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
	// Search takes precedence; fall back to ReviewStatus-based search.
	if filters.Search != "" {
		args = append(args, "--search", filters.Search)
	} else if filters.ReviewStatus != "" {
		args = append(args, "--search", "review:"+filters.ReviewStatus)
	}
	output, err := RunCommand("gh", args...)
	if err != nil {
		return nil, err
	}
	prs, err := parsePRs(output)
	if err != nil {
		return nil, err
	}
	if filters.Draft == "false" {
		prs = FilterNonDraftPRs(prs)
	}
	return prs, nil
}

// FetchReviewRequests returns open PRs where the authenticated user is a requested reviewer.
func FetchReviewRequests(limit int) ([]PullRequest, error) {
	return FetchPRs(PRFilters{
		State:  "open",
		Search: "review-requested:@me",
		Limit:  limit,
	})
}

func FetchPR(number int) (PullRequest, error) {
	output, err := RunCommand(
		"gh", "pr", "view", strconv.Itoa(number),
		"--json", "number,title,body,author,assignees,labels,state,isDraft,headRefName,reviewDecision,statusCheckRollup,url,createdAt",
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

func ClosePR(number int) error {
	return RunCommandPassthrough("gh", "pr", "close", strconv.Itoa(number))
}

func ReopenPR(number int) error {
	return RunCommandPassthrough("gh", "pr", "reopen", strconv.Itoa(number))
}

func FilterNonDraftPRs(prs []PullRequest) []PullRequest {
	out := make([]PullRequest, 0, len(prs))
	for _, pr := range prs {
		if !pr.IsDraft {
			out = append(out, pr)
		}
	}
	return out
}

func parsePRs(data []byte) ([]PullRequest, error) {
	var prs []PullRequest
	return prs, json.Unmarshal(data, &prs)
}
```

- [ ] **Step 2: Verify it compiles**

```bash
go build ./internal/github/...
```

Expected: no output (success).

- [ ] **Step 3: Commit**

```bash
git add internal/github/prs.go
git commit -m "refactor: add internal/github PR operations"
```

---

### Task 4: Write tests for `internal/github`

**Files:**
- Create: `internal/github/issues_test.go`
- Create: `internal/github/prs_test.go`

- [ ] **Step 1: Write `internal/github/issues_test.go`**

```go
// internal/github/issues_test.go
package github

import (
	"testing"
)

func TestParseIssues(t *testing.T) {
	data := []byte(`[
		{"number":42,"title":"Fix bug","state":"open","url":"https://github.com/o/r/issues/42",
		 "assignees":[{"login":"alice"}],"labels":[{"name":"bug"}],"createdAt":"2026-01-01T00:00:00Z"},
		{"number":43,"title":"Add feature","state":"closed","url":"https://github.com/o/r/issues/43",
		 "assignees":[],"labels":[],"createdAt":"2026-01-02T00:00:00Z"}
	]`)

	issues, err := parseIssues(data)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(issues) != 2 {
		t.Fatalf("expected 2 issues, got %d", len(issues))
	}
	if issues[0].Number != 42 {
		t.Errorf("expected issue 42, got %d", issues[0].Number)
	}
	if issues[0].Assignees[0].Login != "alice" {
		t.Errorf("expected assignee alice, got %s", issues[0].Assignees[0].Login)
	}
	if issues[0].Labels[0].Name != "bug" {
		t.Errorf("expected label bug, got %s", issues[0].Labels[0].Name)
	}
	if issues[1].State != "closed" {
		t.Errorf("expected closed state, got %s", issues[1].State)
	}
}

func TestParseIssues_Empty(t *testing.T) {
	issues, err := parseIssues([]byte(`[]`))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(issues) != 0 {
		t.Errorf("expected 0 issues, got %d", len(issues))
	}
}

func TestParseIssues_Invalid(t *testing.T) {
	_, err := parseIssues([]byte(`not json`))
	if err == nil {
		t.Error("expected error for invalid JSON, got nil")
	}
}
```

- [ ] **Step 2: Run the test to verify it passes**

```bash
go test ./internal/github/... -run TestParseIssues -v
```

Expected: all three `TestParseIssues` tests PASS.

- [ ] **Step 3: Write `internal/github/prs_test.go`**

```go
// internal/github/prs_test.go
package github

import (
	"testing"
)

func TestParsePRs(t *testing.T) {
	data := []byte(`[
		{"number":10,"title":"Add login","state":"open","isDraft":false,
		 "author":{"login":"bob"},"assignees":[],"labels":[{"name":"feature"}],
		 "headRefName":"10-add-login","statusCheckRollup":[{"status":"COMPLETED","conclusion":"SUCCESS"}],
		 "url":"https://github.com/o/r/pull/10","createdAt":"2026-01-01T00:00:00Z"},
		{"number":11,"title":"WIP: refactor","state":"open","isDraft":true,
		 "author":{"login":"bob"},"assignees":[],"labels":[],
		 "headRefName":"11-wip","statusCheckRollup":[],
		 "url":"https://github.com/o/r/pull/11","createdAt":"2026-01-02T00:00:00Z"}
	]`)

	prs, err := parsePRs(data)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(prs) != 2 {
		t.Fatalf("expected 2 PRs, got %d", len(prs))
	}
	if prs[0].Number != 10 {
		t.Errorf("expected PR 10, got %d", prs[0].Number)
	}
	if prs[1].IsDraft != true {
		t.Errorf("expected PR 11 to be draft")
	}
}

func TestFilterNonDraftPRs(t *testing.T) {
	prs := []PullRequest{
		{Number: 1, IsDraft: false},
		{Number: 2, IsDraft: true},
		{Number: 3, IsDraft: false},
	}
	result := FilterNonDraftPRs(prs)
	if len(result) != 2 {
		t.Fatalf("expected 2 non-draft PRs, got %d", len(result))
	}
	if result[0].Number != 1 || result[1].Number != 3 {
		t.Errorf("unexpected PR numbers: %v", result)
	}
}

func TestFilterNonDraftPRs_AllDraft(t *testing.T) {
	prs := []PullRequest{{Number: 1, IsDraft: true}, {Number: 2, IsDraft: true}}
	result := FilterNonDraftPRs(prs)
	if len(result) != 0 {
		t.Errorf("expected 0 results, got %d", len(result))
	}
}
```

- [ ] **Step 4: Run all internal/github tests**

```bash
go test ./internal/github/... -v
```

Expected: all tests PASS.

- [ ] **Step 5: Commit**

```bash
git add internal/github/issues_test.go internal/github/prs_test.go
git commit -m "test: add internal/github parse and filter tests"
```

---

### Task 5: Create `terminal.go`

Move all terminal I/O primitives out of `main.go`.

**Files:**
- Create: `terminal.go`

- [ ] **Step 1: Create `terminal.go` with all terminal primitives**

```go
// terminal.go
package main

import (
	"bufio"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"runtime"
	"strconv"
	"strings"
)

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

func clearScreen() {
	fmt.Print("\033[H\033[2J")
}

func termSize() (rows, cols int) {
	cmd := exec.Command("stty", "size")
	cmd.Stdin = os.Stdin
	out, err := cmd.Output()
	if err != nil {
		return 40, 80
	}
	parts := strings.Fields(strings.TrimSpace(string(out)))
	if len(parts) != 2 {
		return 40, 80
	}
	r, err1 := strconv.Atoi(parts[0])
	c, err2 := strconv.Atoi(parts[1])
	if err1 != nil || err2 != nil || r <= 0 || c <= 0 {
		return 40, 80
	}
	return r, c
}

func bodyBudget(fixedLines, menuItems int) (visualRows, termCols int) {
	rows, cols := termSize()
	budget := rows - fixedLines - (menuItems + 2) - 2
	if budget < 3 {
		budget = 3
	}
	return budget, cols
}

func require(name string) error {
	_, err := exec.LookPath(name)
	if err != nil {
		return fmt.Errorf("%s is required but was not found in PATH", name)
	}
	return nil
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

// emptyTabAction reads a single keypress for empty-list and error screens.
// switchTarget is the tab to switch to on Tab/Shift-Tab.
// Returns "quit", "switch", "filters", or "" (retry/refresh).
func emptyTabAction(reader *bufio.Reader, state *AppState, switchTarget TabID) string {
	if err := enableRawMode(); err != nil {
		input := prompt(reader, "> ")
		switch strings.TrimSpace(strings.ToLower(input)) {
		case "q", "quit", "b":
			return "quit"
		case "f":
			return "filters"
		}
		return ""
	}
	defer disableRawMode()
	var buf [4]byte
	n, _ := os.Stdin.Read(buf[:])
	key := string(buf[:n])
	switch key {
	case "\t", "\x1b[Z":
		state.ActiveTab = switchTarget
		return "switch"
	case "f", "F":
		return "filters"
	case "q", "Q", "b", "B", "\x03", "\x1b":
		return "quit"
	}
	return ""
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
		for index, option := range options {
			prefix := "  "
			if index == selected {
				prefix = "> "
			}
			fmt.Printf("%s%s\033[K\r\n", prefix, option)
		}
		fmt.Print("\r\n")
		fmt.Print("↑/↓ navigate • enter submit • 1-9 jump • q quit/back\033[K")
		fmt.Printf("\033[%dF", len(options)+1)
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
			fmt.Printf("\033[%dB\r\n", len(options)+1)
			return options[selected]
		case "q", "Q", "\x03", "\x1b":
			fmt.Print("\033[?25h")
			fmt.Printf("\033[%dB\r\n", len(options)+1)
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
```

- [ ] **Step 2: Verify the file compiles (main.go still has duplicates — that's OK for now)**

```bash
go vet ./... 2>&1 | head -20
```

Expected: duplicate function errors from `main.go`. That's expected — we'll remove them in Task 9.

- [ ] **Step 3: Commit the file as-is**

```bash
git add terminal.go
git commit -m "refactor: extract terminal primitives into terminal.go"
```

---

### Task 6: Create `render.go`

**Files:**
- Create: `render.go`

- [ ] **Step 1: Create `render.go`**

```go
// render.go
package main

import (
	"fmt"
	"os"
	"strings"
	"text/tabwriter"

	"hubcap/internal/github"
)

const (
	colorReset  = "\033[0m"
	colorGreen  = "\033[32m"
	colorYellow = "\033[33m"
	colorRed    = "\033[31m"
	colorPurple = "\033[35m"
	colorGray   = "\033[90m"
	colorInvert = "\033[7m"
)

func renderHeader(state *AppState, rawMode bool) {
	nl := "\n"
	if rawMode {
		nl = "\r\n"
	}
	sep := strings.Repeat("=", 52)

	myWorkLabel := "  1: My Work  "
	issuesLabel := "  2: Issues  "
	prsLabel := "  3: Pull Requests  "
	switch state.ActiveTab {
	case TabDashboard:
		myWorkLabel = colorInvert + myWorkLabel + colorReset
	case TabIssues:
		issuesLabel = colorInvert + issuesLabel + colorReset
	case TabPRs:
		prsLabel = colorInvert + prsLabel + colorReset
	}

	fmt.Printf("GitHub TUI — %s%s", state.Repo, nl)
	fmt.Printf("%s%s", sep, nl)
	fmt.Printf("%s%s%s%s", myWorkLabel, issuesLabel, prsLabel, nl)
	fmt.Printf("%s%s", sep, nl)

	switch state.ActiveTab {
	case TabDashboard:
		fmt.Printf("My Work%s", nl)
	case TabIssues:
		f := state.IssueFilters
		fmt.Printf("State: %s | Assignee: %s | Label: %s | Limit: %d%s",
			f.State, displayAny(f.Assignee), displayAny(f.Label), f.Limit, nl)
	case TabPRs:
		f := state.PRFilters
		fmt.Printf("State: %s | Assignee: %s | Label: %s | Limit: %d%s",
			f.State, displayAny(f.Assignee), displayAny(f.Label), f.Limit, nl)
	}
	fmt.Printf("%s%s", sep, nl)
	fmt.Print(nl)
}

func printIssueDetail(issue github.Issue, maxBodyRows, termCols int) {
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
	fmt.Println(truncateLines(body, maxBodyRows, termCols))
	fmt.Println()
}

func printIssuesTable(issues []github.Issue) {
	writer := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintln(writer, "NUMBER\tTITLE\tASSIGNEE\tLABELS")
	fmt.Fprintln(writer, "------\t-----\t--------\t------")
	for _, issue := range issues {
		fmt.Fprintf(writer, "%d\t%s\t%s\t%s\n",
			issue.Number,
			truncate(cleanLine(issue.Title), 58),
			truncate(joinUsers(issue.Assignees), 22),
			truncate(joinLabels(issue.Labels), 34),
		)
	}
	writer.Flush()
}

func printPRDetail(pr github.PullRequest, maxBodyRows, termCols int) {
	sep := strings.Repeat("=", 80)
	fmt.Printf("#%d %s\n", pr.Number, pr.Title)
	fmt.Println(sep)

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

	assigneeStr := "—"
	if len(pr.Assignees) > 0 {
		logins := make([]string, len(pr.Assignees))
		for i, a := range pr.Assignees {
			logins[i] = a.Login
		}
		assigneeStr = strings.Join(logins, ", ")
	}
	fmt.Printf("%-12s %s\n", "Assignees:", assigneeStr)

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
	fmt.Printf("%-12s %s\n", "Checks:", summarizeChecks(pr.StatusRollup))

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
		fmt.Println(truncateLines(strings.TrimSpace(pr.Body), maxBodyRows, termCols))
	}
	fmt.Println()
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

func summarizeChecks(checks []github.CheckRun) string {
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

func joinUsers(users []github.User) string {
	if len(users) == 0 {
		return "Unassigned"
	}
	values := make([]string, 0, len(users))
	for _, user := range users {
		values = append(values, user.Login)
	}
	return strings.Join(values, ", ")
}

func joinLabels(labels []github.Label) string {
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

func truncateLines(text string, maxVisualRows, termCols int) string {
	if termCols <= 0 {
		termCols = 80
	}
	lines := strings.Split(text, "\n")
	usedRows := 0
	for i, line := range lines {
		runeLen := len([]rune(line))
		lineRows := 1
		if runeLen > termCols {
			lineRows = (runeLen + termCols - 1) / termCols
		}
		if usedRows+lineRows > maxVisualRows {
			out := strings.TrimRight(strings.Join(lines[:i], "\n"), " \t")
			return out + "\n" + colorGray + "… open in browser to read more" + colorReset
		}
		usedRows += lineRows
	}
	return text
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

func displayAny(value string) string {
	if strings.TrimSpace(value) == "" {
		return "any"
	}
	return value
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

func fatal(err error) {
	fmt.Fprintln(os.Stderr, err)
	os.Exit(1)
}
```

- [ ] **Step 2: Commit**

```bash
git add render.go
git commit -m "refactor: extract display utilities into render.go"
```

---

### Task 7: Create `issues.go`

**Files:**
- Create: `issues.go`

- [ ] **Step 1: Create `issues.go`**

```go
// issues.go
package main

import (
	"bufio"
	"fmt"
	"os"
	"strconv"
	"strings"

	"hubcap/internal/github"
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
			fmt.Printf("%s%s %-6d %-58s %-22s %-34s\r\n",
				prefix, indicator, issue.Number,
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
			filters = github.Filters{State: "open", Limit: 50}
		case "Back", "":
			return filters
		}
	}
}
```

- [ ] **Step 2: Commit**

```bash
git add issues.go
git commit -m "refactor: extract issues tab into issues.go"
```

---

### Task 8: Create `prs.go`

**Files:**
- Create: `prs.go`

- [ ] **Step 1: Create `prs.go`**

```go
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
			action := emptyTabAction(reader, state, TabIssues)
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
			action := emptyTabAction(reader, state, TabIssues)
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
		clearScreen()
		renderHeader(state, true)
		fmt.Print("\033[?25l")
		fmt.Printf("  %-8s %-58s %-12s %-9s %s\r\n", "  #", "TITLE", "AUTHOR", "STATUS", "CHECKS")
		fmt.Printf("  %-8s %-58s %-12s %-9s %s\r\n", "-----", strings.Repeat("-", 58), "-----------", "--------", "------")
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
			fmt.Printf("%s%s %-6d %-58s %-12s %-9s %s\r\n",
				prefix, indicator, pr.Number, truncate(pr.Title, 58),
				pr.Author.Login, status, summarizeChecks(pr.StatusRollup))
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
				github.RunCommandPassthrough("gh", "pr", "merge", strconv.Itoa(number), "--merge")
				return
			case "Squash and merge":
				clearScreen()
				renderHeader(state, false)
				github.RunCommandPassthrough("gh", "pr", "merge", strconv.Itoa(number), "--squash")
				return
			case "Rebase and merge":
				clearScreen()
				renderHeader(state, false)
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
```

- [ ] **Step 2: Commit**

```bash
git add prs.go
git commit -m "refactor: extract PRs tab into prs.go"
```

---

### Task 9: Slim `main.go` and verify the full build

Replace `main.go` with the thin version, delete the compiled binary, verify everything compiles and tests pass.

**Files:**
- Modify: `main.go` (full replacement)
- Delete: `hubcap` (compiled binary)

- [ ] **Step 1: Replace `main.go`**

```go
// main.go
package main

import (
	"bufio"
	"fmt"
	"os"

	"hubcap/internal/github"
)

type TabID int

const (
	TabDashboard TabID = iota
	TabIssues
	TabPRs
)

type AppState struct {
	ActiveTab       TabID
	IssueFilters    github.Filters
	PRFilters       github.PRFilters
	IssueSelected   int
	PRSelected      int
	DashboardCursor int
	Repo            string
}

// nextTab cycles TabDashboard → TabIssues → TabPRs → TabDashboard.
func nextTab(current TabID) TabID {
	return (current + 1) % 3
}

func main() {
	if err := require("gh"); err != nil {
		fatal(err)
	}

	reader := bufio.NewReader(os.Stdin)
	cfg := loadConfig()

	state := &AppState{
		ActiveTab:    TabDashboard,
		IssueFilters: github.Filters{State: "open", Limit: 50},
		PRFilters:    github.PRFilters{State: "open", Limit: 50},
	}
	state.Repo = github.FetchRepo()

	for {
		var action string
		switch state.ActiveTab {
		case TabDashboard:
			action = browseDashboard(reader, state, &cfg)
		case TabIssues:
			action = browseIssues(reader, state)
		case TabPRs:
			action = browsePRs(reader, state)
		}
		if action == "quit" {
			fmt.Println("\nBye.")
			return
		}
	}
}
```

- [ ] **Step 2: Remove the compiled binary and add it to `.gitignore`**

```bash
rm -f hubcap
echo "hubcap" >> .gitignore
echo "hubcap-*" >> .gitignore
```

- [ ] **Step 3: Build — expect errors about `browseDashboard` and `loadConfig` not defined yet**

```bash
go build ./... 2>&1
```

Expected: errors referencing `browseDashboard` and `loadConfig` — these will be added in Tasks 10 and 12. That is fine for now.

- [ ] **Step 4: Verify the existing test suite still passes (tests don't import main)**

```bash
go test ./internal/github/... -v
```

Expected: all tests PASS.

- [ ] **Step 5: Commit**

```bash
git add main.go .gitignore
git rm --cached hubcap 2>/dev/null || true
git commit -m "refactor: slim main.go to AppState + main loop only, remove compiled binary"
```

---

## Phase 2 — Config System

---

### Task 10: Create `config.go` with tests

**Files:**
- Create: `config.go`
- Create: `config_test.go`

- [ ] **Step 1: Write the failing test first**

```go
// config_test.go
package main

import (
	"os"
	"path/filepath"
	"testing"

	"hubcap/internal/github"
)

func TestLoadConfig_Defaults(t *testing.T) {
	cfg := loadConfigFrom("/nonexistent/path/config.json")
	if cfg.AvailableFilter.State != "open" {
		t.Errorf("expected default state open, got %s", cfg.AvailableFilter.State)
	}
	if cfg.AvailableFilter.Limit != 25 {
		t.Errorf("expected default limit 25, got %d", cfg.AvailableFilter.Limit)
	}
}

func TestLoadConfig_InvalidJSON(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.json")
	os.WriteFile(path, []byte("not json"), 0644)

	cfg := loadConfigFrom(path)
	if cfg.AvailableFilter.State != "open" {
		t.Errorf("expected default state on bad JSON, got %s", cfg.AvailableFilter.State)
	}
}

func TestSaveAndLoadConfig(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.json")

	cfg := Config{
		AvailableFilter: github.Filters{
			State: "open",
			Label: "ready",
			Limit: 10,
		},
	}

	if err := saveConfigTo(cfg, path); err != nil {
		t.Fatalf("save failed: %v", err)
	}

	loaded := loadConfigFrom(path)
	if loaded.AvailableFilter.Label != "ready" {
		t.Errorf("expected label ready, got %s", loaded.AvailableFilter.Label)
	}
	if loaded.AvailableFilter.Limit != 10 {
		t.Errorf("expected limit 10, got %d", loaded.AvailableFilter.Limit)
	}
}
```

- [ ] **Step 2: Run to see tests fail**

```bash
go test ./... -run TestLoadConfig -v 2>&1 | head -20
```

Expected: compile error — `loadConfigFrom`, `saveConfigTo`, `Config` not defined.

- [ ] **Step 3: Create `config.go`**

```go
// config.go
package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"hubcap/internal/github"
)

type Config struct {
	AvailableFilter github.Filters `json:"available_filter"`
}

func defaultConfig() Config {
	return Config{
		AvailableFilter: github.Filters{State: "open", Limit: 25},
	}
}

func configPath() string {
	dir, err := os.UserConfigDir()
	if err != nil {
		dir = os.Getenv("HOME")
	}
	return filepath.Join(dir, "hubcap", "config.json")
}

func loadConfig() Config {
	return loadConfigFrom(configPath())
}

func loadConfigFrom(path string) Config {
	data, err := os.ReadFile(path)
	if err != nil {
		return defaultConfig()
	}
	var cfg Config
	if err := json.Unmarshal(data, &cfg); err != nil {
		fmt.Fprintf(os.Stderr, "hubcap: warning: bad config file, using defaults (%v)\n", err)
		return defaultConfig()
	}
	return cfg
}

func saveConfig(cfg Config) error {
	return saveConfigTo(cfg, configPath())
}

func saveConfigTo(cfg Config, path string) error {
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return err
	}
	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0644)
}
```

- [ ] **Step 4: Run tests**

```bash
go test ./... -run TestLoadConfig -run TestSaveAndLoad -v
```

Expected: all three config tests PASS.

- [ ] **Step 5: Commit**

```bash
git add config.go config_test.go
git commit -m "feat: add config system with load/save and tests"
```

---

## Phase 3 — My Work Dashboard

---

### Task 11: Create `dashboard.go` — data types and concurrent fetch

**Files:**
- Create: `dashboard.go` (initial, data layer only)

- [ ] **Step 1: Write the failing test for `buildRows`**

Add to a new file `dashboard_test.go`:

```go
// dashboard_test.go
package main

import (
	"testing"

	"hubcap/internal/github"
)

func TestBuildRows_AllSectionsPopulated(t *testing.T) {
	data := dashboardResult{
		reviewRequests:  []github.PullRequest{{Number: 1}},
		myPRs:           []github.PullRequest{{Number: 2}},
		assignedIssues:  []github.Issue{{Number: 3}},
		availableIssues: []github.Issue{{Number: 4}},
	}
	collapsed := [4]bool{}
	rows := buildRows(data, collapsed)

	// 4 section headers + 4 items = 8 rows total
	if len(rows) != 8 {
		t.Fatalf("expected 8 rows, got %d", len(rows))
	}
	if !rows[0].isHeader {
		t.Error("expected first row to be a section header")
	}
	if rows[1].isHeader {
		t.Error("expected second row to be an item")
	}
}

func TestBuildRows_CollapsedSection(t *testing.T) {
	data := dashboardResult{
		reviewRequests:  []github.PullRequest{{Number: 1}, {Number: 2}},
		myPRs:           []github.PullRequest{},
		assignedIssues:  []github.Issue{{Number: 3}},
		availableIssues: []github.Issue{},
	}
	// Collapse section 0 (review requests); sections 1,3 are empty so hidden
	collapsed := [4]bool{true, false, false, false}
	rows := buildRows(data, collapsed)

	// Section 0 header only (collapsed, 2 items hidden)
	// Section 1 hidden (empty)
	// Section 2 header + 1 item
	// Section 3 hidden (empty)
	// = 3 rows
	if len(rows) != 3 {
		t.Fatalf("expected 3 rows, got %d", len(rows))
	}
	if !rows[0].isHeader || rows[0].sectionID != 0 {
		t.Error("expected first row to be section 0 header")
	}
}

func TestBuildRows_EmptySectionsHidden(t *testing.T) {
	data := dashboardResult{} // all empty
	collapsed := [4]bool{}
	rows := buildRows(data, collapsed)
	if len(rows) != 0 {
		t.Errorf("expected 0 rows for empty data, got %d", len(rows))
	}
}
```

- [ ] **Step 2: Run to verify tests fail**

```bash
go test ./... -run TestBuildRows -v 2>&1 | head -20
```

Expected: compile error — `dashboardResult`, `buildRows`, `dashRow` not defined.

- [ ] **Step 3: Create `dashboard.go` with data types and `buildRows`**

```go
// dashboard.go
package main

import (
	"bufio"
	"fmt"
	"os"
	"strconv"
	"strings"
	"sync"

	"hubcap/internal/github"
)

// ── Data types ────────────────────────────────────────────────────────────────

type dashboardResult struct {
	reviewRequests  []github.PullRequest
	myPRs           []github.PullRequest
	assignedIssues  []github.Issue
	availableIssues []github.Issue
	errs            [4]error
}

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

// sectionLen returns the item count for a given section.
func sectionLen(data dashboardResult, sec int) int {
	switch sec {
	case secReviewRequests:
		return len(data.reviewRequests)
	case secMyPRs:
		return len(data.myPRs)
	case secAssigned:
		return len(data.assignedIssues)
	case secAvailable:
		return len(data.availableIssues)
	}
	return 0
}

// buildRows constructs the flat navigable row list from dashboard data.
// Empty sections are omitted entirely. Collapsed sections show only their header.
func buildRows(data dashboardResult, collapsed [4]bool) []dashRow {
	var rows []dashRow
	for sec := 0; sec < 4; sec++ {
		count := sectionLen(data, sec)
		if count == 0 && data.errs[sec] == nil {
			continue // hidden when empty and no error
		}
		rows = append(rows, dashRow{isHeader: true, sectionID: sec, itemIdx: -1})
		if collapsed[sec] {
			continue
		}
		isIssue := sec == secAssigned || sec == secAvailable
		for i := 0; i < count; i++ {
			rows = append(rows, dashRow{isHeader: false, sectionID: sec, itemIdx: i, isIssue: isIssue})
		}
	}
	return rows
}

// ── Concurrent data fetch ─────────────────────────────────────────────────────

func fetchDashboard(cfg Config) dashboardResult {
	var wg sync.WaitGroup
	var result dashboardResult

	wg.Add(4)

	go func() {
		defer wg.Done()
		result.reviewRequests, result.errs[secReviewRequests] = github.FetchReviewRequests(25)
	}()
	go func() {
		defer wg.Done()
		result.myPRs, result.errs[secMyPRs] = github.FetchPRs(github.PRFilters{
			Author: "@me", State: "open", Limit: 25,
		})
	}()
	go func() {
		defer wg.Done()
		result.assignedIssues, result.errs[secAssigned] = github.FetchIssues(github.Filters{
			Assignee: "@me", State: "open", Limit: 25,
		})
	}()
	go func() {
		defer wg.Done()
		result.availableIssues, result.errs[secAvailable] = github.FetchIssues(cfg.AvailableFilter)
	}()

	wg.Wait()
	return result
}
```

- [ ] **Step 4: Run the `buildRows` tests**

```bash
go test ./... -run TestBuildRows -v
```

Expected: all three `TestBuildRows` tests PASS.

- [ ] **Step 5: Commit**

```bash
git add dashboard.go dashboard_test.go
git commit -m "feat: add dashboard data model, buildRows, and concurrent fetchDashboard"
```

---

### Task 12: Add dashboard rendering to `dashboard.go`

**Files:**
- Modify: `dashboard.go` (append rendering functions)

- [ ] **Step 1: Append rendering functions to `dashboard.go`**

```go
// ── Rendering ─────────────────────────────────────────────────────────────────

func renderDashboard(state *AppState, data dashboardResult, rows []dashRow, cursor int, collapsed [4]bool, rawMode bool) {
	nl := "\n"
	cr := ""
	if rawMode {
		nl = "\r\n"
		cr = "\r"
	}

	clearScreen()
	renderHeader(state, rawMode)

	if len(rows) == 0 {
		fmt.Printf("No items to show. Press r to refresh.%s", nl)
		return
	}

	for i, row := range rows {
		sel := ""
		if i == cursor {
			sel = "> "
		} else {
			sel = "  "
		}

		if row.isHeader {
			count := sectionLen(data, row.sectionID)
			errMark := ""
			if data.errs[row.sectionID] != nil {
				errMark = " ⚠ could not load"
				count = 0
			}
			arrow := "▾"
			if collapsed[row.sectionID] {
				arrow = "▸"
			}
			fmt.Printf("%s%s %s (%d)%s%s%s",
				sel, arrow, sectionNames[row.sectionID], count, errMark, cr, nl)
			continue
		}

		// item row
		switch {
		case row.sectionID == secReviewRequests || row.sectionID == secMyPRs:
			pr := data.reviewRequests
			if row.sectionID == secMyPRs {
				pr = data.myPRs
			}
			p := pr[row.itemIdx]
			indicator := stateIndicator(p.State, p.IsDraft)
			checks := summarizeChecks(p.StatusRollup)
			fmt.Printf("  %s%s #%-5d %-52s %s%s%s",
				sel, indicator, p.Number, truncate(p.Title, 52), checks, cr, nl)
		case row.isIssue:
			var issue github.Issue
			if row.sectionID == secAssigned {
				issue = data.assignedIssues[row.itemIdx]
			} else {
				issue = data.availableIssues[row.itemIdx]
			}
			indicator := stateIndicator(issue.State, false)
			labels := truncate(joinLabels(issue.Labels), 24)
			fmt.Printf("  %s%s #%-5d %-48s %s%s%s",
				sel, indicator, issue.Number, truncate(cleanLine(issue.Title), 48), labels, cr, nl)
		}
	}

	fmt.Print(nl)
	if rawMode {
		fmt.Print("↑/↓ navigate • enter open • ← collapse • tab switch • n issue • p PR • r refresh • c config • q quit\r\n")
	} else {
		fmt.Print("number open • n new issue • p new PR • r refresh • c config • q quit\n")
	}
}
```

- [ ] **Step 2: Build to verify no compile errors**

```bash
go build ./... 2>&1
```

Expected: no output (success) — `browseDashboard` is still missing but the render function compiles.

Actually, `go build` will still fail because `browseDashboard` is referenced in `main.go` but not yet defined. That's expected — we add it in the next task.

- [ ] **Step 3: Commit**

```bash
git add dashboard.go
git commit -m "feat: add dashboard rendering"
```

---

### Task 13: Add `browseDashboard` navigation loop to `dashboard.go`

**Files:**
- Modify: `dashboard.go` (append `browseDashboard`)

- [ ] **Step 1: Append `browseDashboard` to `dashboard.go`**

```go
// ── Navigation loop ───────────────────────────────────────────────────────────

func browseDashboard(reader *bufio.Reader, state *AppState, cfg *Config) string {
	var data dashboardResult
	var rows []dashRow
	collapsed := [4]bool{}
	needsRefresh := true

	for {
		if needsRefresh {
			clearScreen()
			renderHeader(state, false)
			fmt.Println("Loading...")
			data = fetchDashboard(*cfg)
			rows = buildRows(data, collapsed)
			needsRefresh = false
		}

		cursor := state.DashboardCursor
		if cursor >= len(rows) {
			cursor = 0
		}

		if err := enableRawMode(); err != nil {
			// Fallback: non-raw mode
			renderDashboard(state, data, rows, cursor, collapsed, false)
			input := prompt(reader, "Number to open, r refresh, c config, q quit: ")
			input = strings.TrimSpace(strings.ToLower(input))
			switch input {
			case "q", "quit", "b":
				return "quit"
			case "r":
				needsRefresh = true
				continue
			case "c":
				*cfg = configureHubcap(reader, state, *cfg)
				data = fetchDashboard(*cfg)
				rows = buildRows(data, collapsed)
				continue
			}
			num, err := strconv.Atoi(input)
			if err == nil {
				openDashboardItem(reader, state, data, rows, num)
			}
			continue
		}

		renderDashboard(state, data, rows, cursor, collapsed, true)

		buf := make([]byte, 3)
		n, err := os.Stdin.Read(buf)
		disableRawMode()
		if err != nil || n == 0 {
			return "quit"
		}

		key := string(buf[:n])
		switch key {
		case "q", "Q", "\x03", "\x1b":
			state.DashboardCursor = cursor
			return "quit"

		case "\t", "\x1b[Z":
			state.DashboardCursor = cursor
			state.ActiveTab = nextTab(state.ActiveTab)
			return ""

		case "1":
			state.ActiveTab = TabDashboard
			return ""
		case "2":
			state.ActiveTab = TabIssues
			return ""
		case "3":
			state.ActiveTab = TabPRs
			return ""

		case "r", "R":
			needsRefresh = true

		case "n", "N":
			disableRawMode()
			clearScreen()
			renderHeader(state, false)
			github.RunCommandPassthrough("gh", "issue", "create")
			needsRefresh = true

		case "p", "P":
			disableRawMode()
			clearScreen()
			renderHeader(state, false)
			github.RunCommandPassthrough("gh", "pr", "create")
			needsRefresh = true

		case "c", "C":
			disableRawMode()
			*cfg = configureHubcap(reader, state, *cfg)
			needsRefresh = true

		case "\x1b[A": // up
			cursor--
			if cursor < 0 {
				cursor = len(rows) - 1
			}
			state.DashboardCursor = cursor

		case "\x1b[B": // down
			cursor++
			if cursor >= len(rows) {
				cursor = 0
			}
			state.DashboardCursor = cursor

		case "\x1b[D": // left — collapse current section
			if len(rows) > 0 {
				sec := rows[cursor].sectionID
				collapsed[sec] = true
				rows = buildRows(data, collapsed)
				// keep cursor on the section header
				for i, r := range rows {
					if r.isHeader && r.sectionID == sec {
						cursor = i
						break
					}
				}
				state.DashboardCursor = cursor
			}

		case "\r", "\n":
			if len(rows) == 0 {
				continue
			}
			row := rows[cursor]
			if row.isHeader {
				// toggle collapse
				collapsed[row.sectionID] = !collapsed[row.sectionID]
				rows = buildRows(data, collapsed)
				// keep cursor on header
				for i, r := range rows {
					if r.isHeader && r.sectionID == row.sectionID {
						cursor = i
						break
					}
				}
				state.DashboardCursor = cursor
			} else {
				// open item
				state.DashboardCursor = cursor
				disableRawMode()
				openDashboardItemByRow(reader, state, data, row)
				needsRefresh = true
			}
		}
	}
}

// openDashboardItemByRow opens the detail view for the item described by row.
func openDashboardItemByRow(reader *bufio.Reader, state *AppState, data dashboardResult, row dashRow) {
	switch {
	case row.sectionID == secReviewRequests:
		viewPR(reader, state, data.reviewRequests[row.itemIdx].Number)
	case row.sectionID == secMyPRs:
		viewPR(reader, state, data.myPRs[row.itemIdx].Number)
	case row.sectionID == secAssigned:
		viewIssue(reader, state, data.assignedIssues[row.itemIdx].Number)
	case row.sectionID == secAvailable:
		viewIssue(reader, state, data.availableIssues[row.itemIdx].Number)
	}
}

// openDashboardItem finds an item by number across all sections and opens it.
func openDashboardItem(reader *bufio.Reader, state *AppState, data dashboardResult, rows []dashRow, number int) {
	for _, row := range rows {
		if row.isHeader {
			continue
		}
		switch row.sectionID {
		case secReviewRequests:
			if data.reviewRequests[row.itemIdx].Number == number {
				viewPR(reader, state, number)
				return
			}
		case secMyPRs:
			if data.myPRs[row.itemIdx].Number == number {
				viewPR(reader, state, number)
				return
			}
		case secAssigned:
			if data.assignedIssues[row.itemIdx].Number == number {
				viewIssue(reader, state, number)
				return
			}
		case secAvailable:
			if data.availableIssues[row.itemIdx].Number == number {
				viewIssue(reader, state, number)
				return
			}
		}
	}
	fmt.Printf("Item #%d not found in current view.\n", number)
	pause(reader)
}
```

- [ ] **Step 2: Build to verify everything compiles**

```bash
go build ./...
```

Expected: no output (success). `browseDashboard` is now defined, so `main.go` should compile cleanly.

- [ ] **Step 3: Commit**

```bash
git add dashboard.go
git commit -m "feat: add browseDashboard navigation loop"
```

---

### Task 14: Add `configureHubcap` to `dashboard.go`

**Files:**
- Modify: `dashboard.go` (append config screen function)

- [ ] **Step 1: Append `configureHubcap` to `dashboard.go`**

```go
// ── Config screen ─────────────────────────────────────────────────────────────

func configureHubcap(reader *bufio.Reader, state *AppState, cfg Config) Config {
	for {
		clearScreen()
		renderHeader(state, false)
		fmt.Println("Configure hubcap")
		fmt.Println()

		choice := menu(reader, []string{
			"Change \"Available to Grab\" filter",
			"Reset to defaults",
			"Back",
		})

		switch choice {
		case "Change \"Available to Grab\" filter":
			cfg.AvailableFilter = configureAvailableFilter(reader, state, cfg.AvailableFilter)
			if err := saveConfig(cfg); err != nil {
				fmt.Println("Could not save config:", err)
				pause(reader)
			}
		case "Reset to defaults":
			cfg = defaultConfig()
			if err := saveConfig(cfg); err != nil {
				fmt.Println("Could not save config:", err)
				pause(reader)
			} else {
				fmt.Println("Reset to defaults.")
				pause(reader)
			}
		case "Back", "":
			return cfg
		}
	}
}

func configureAvailableFilter(reader *bufio.Reader, state *AppState, filters github.Filters) github.Filters {
	for {
		clearScreen()
		renderHeader(state, false)
		fmt.Println("Available to Grab filter")
		fmt.Println()

		choice := menu(reader, []string{
			"Change state",
			"Change assignee",
			"Change label",
			"Change milestone",
			"Change limit",
			"Clear filter",
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
			filters.Assignee = strings.TrimSpace(prompt(reader, "Assignee, or blank for any: "))
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
		case "Clear filter":
			filters = github.Filters{State: "open", Limit: 25}
		case "Back", "":
			return filters
		}
	}
}
```

- [ ] **Step 2: Build and test**

```bash
go build ./... && go test ./...
```

Expected: build succeeds, all tests PASS.

- [ ] **Step 3: Commit**

```bash
git add dashboard.go
git commit -m "feat: add configureHubcap config screen"
```

---

### Task 15: Fix `emptyTabAction` target in `prs.go`

With three tabs cycling Dashboard(0)→Issues(1)→PRs(2)→Dashboard(0), Tab from the PRs empty/error screen must go to Dashboard, not Issues. The `browsePRs` written in Task 8 passes `TabIssues` — fix it to `TabDashboard`.

**Files:**
- Modify: `prs.go`

- [ ] **Step 1: Fix `emptyTabAction` target in `browsePRs`**

In `prs.go`, find the two `emptyTabAction` calls inside `browsePRs` and change both from `TabIssues` to `TabDashboard`:

```go
// Error screen
action := emptyTabAction(reader, state, TabDashboard)

// Empty screen
action := emptyTabAction(reader, state, TabDashboard)
```

(`browseIssues` already passes `TabPRs`, which is correct for Issues→PRs cycling.)

- [ ] **Step 2: Final full build and test**

```bash
go build ./... && go test ./... -v
```

Expected: build succeeds, all tests PASS.

- [ ] **Step 3: Commit**

```bash
git commit -m "feat: complete My Work dashboard — refactor + config + navigation"
```

---

## Phase 4 — Verification

---

### Task 16: Add `.gitignore` and clean up repo

**Files:**
- Modify: `.gitignore`

- [ ] **Step 1: Ensure `.gitignore` is complete**

```bash
cat .gitignore
```

Should contain at minimum:
```
hubcap
hubcap-*
.superpowers/
```

Add any missing lines:
```bash
grep -q "^\.superpowers" .gitignore || echo ".superpowers/" >> .gitignore
```

- [ ] **Step 2: Commit if changed**

```bash
git add .gitignore && git diff --cached --quiet || git commit -m "chore: update .gitignore"
```

---

### Task 17: End-to-end verification

- [ ] **Step 1: Run full test suite**

```bash
go test ./... -v
```

Expected: all tests in `internal/github/`, root package pass.

- [ ] **Step 2: Run `go vet`**

```bash
go vet ./...
```

Expected: no output (no issues).

- [ ] **Step 3: Build the binary**

```bash
go build -o hubcap .
```

Expected: `hubcap` binary created.

- [ ] **Step 4: Smoke test — verify it starts and shows the dashboard**

```bash
./hubcap
```

Expected: dashboard loads, shows "My Work" tab highlighted, four sections attempt to load. Press `q` to quit.

- [ ] **Step 5: Verify tab switching**

Launch `./hubcap`, press Tab → should switch to Issues tab. Press Tab again → PRs tab. Press Tab again → back to My Work dashboard.

- [ ] **Step 6: Verify config screen**

Launch `./hubcap`, press `c` from the dashboard → "Configure hubcap" menu appears. Change the "Available to Grab" filter label to something, press Back. Press `r` to refresh. Press `q`.

Verify config was saved:
```bash
cat ~/.config/hubcap/config.json
```

- [ ] **Step 7: Remove built binary**

```bash
rm hubcap
```

- [ ] **Step 8: Final commit if any loose files**

```bash
git status
```

If clean, done. If any untracked/modified files remain, add and commit them.

---

## Summary

| Phase | Tasks | What it delivers |
|---|---|---|
| Refactor | 1–9 | Identical behaviour, clean file structure |
| Config | 10 | `~/.config/hubcap/config.json` with load/save |
| Dashboard | 11–15 | My Work tab with concurrent fetch, collapsible sections, navigation |
| Cleanup | 16–17 | `.gitignore`, verified binary, smoke tested |
