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
			if i == 0 {
				return text
			}
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
