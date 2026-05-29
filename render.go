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
	colorSelect = "\033[36m"           // cyan text
	colorOrange = "\033[38;5;208m"    // normal orange text
	colorTitle  = "\033[1;38;5;208m"  // bold orange text
	colorSelBg  = "\033[48;5;23m"     // dark teal selection background
)

func renderHeader(state *AppState, rawMode bool) {
	nl := "\n"
	if rawMode {
		nl = "\033[K\r\n"
	}
	_, cols := termSize()
	sep := strings.Repeat("─", cols)

	plainMyWork := "  1: Dashboard  "
	plainIssues := "  2: Issues  "
	plainPRs := "  3: Pull Requests  "
	myWorkLabel := plainMyWork
	issuesLabel := plainIssues
	prsLabel := plainPRs
	switch state.ActiveTab {
	case TabDashboard:
		myWorkLabel = colorSelect + myWorkLabel + colorReset
	case TabIssues:
		issuesLabel = colorSelect + issuesLabel + colorReset
	case TabPRs:
		prsLabel = colorSelect + prsLabel + colorReset
	}
	tabWidth := len(plainMyWork) + len(plainIssues) + len(plainPRs)
	tabPad := ""
	if cols > tabWidth {
		tabPad = strings.Repeat(" ", cols-tabWidth)
	}

	// Line 1: version (left) + help hint (right)
	verText := "  v" + version
	helpText := "[?] help  "
	verLen := len([]rune(verText))
	helpLen := len([]rune(helpText))
	gap := cols - verLen - helpLen
	if gap < 0 {
		gap = 0
	}
	fmt.Printf("%s%s%s%s%s%s", colorTitle, verText, strings.Repeat(" ", gap), helpText, colorReset, nl)

	// Line 2: centered title
	title := "Hubcap — " + state.Repo
	tLen := len([]rune(title))
	lPad, rPad := 0, 0
	if cols > tLen {
		lPad = (cols - tLen) / 2
		rPad = cols - tLen - lPad
	}
	fmt.Printf("%s%s%s%s%s%s", colorTitle, strings.Repeat(" ", lPad), title, strings.Repeat(" ", rPad), colorReset, nl)

	// Line 3: blank
	fmt.Printf("%s%s%s%s", colorTitle, strings.Repeat(" ", cols), colorReset, nl)
	fmt.Printf("%s%s", sep, nl)
	fmt.Printf("%s%s%s%s%s", myWorkLabel, issuesLabel, prsLabel, tabPad, nl)
	fmt.Printf("%s%s", sep, nl)

	dim := colorGray
	rst := colorReset
	pipe := colorGray + " | " + colorReset
	switch state.ActiveTab {
	case TabDashboard:
		if state.DashboardStatus != "" {
			fmt.Printf("%s%s", state.DashboardStatus, nl)
		} else {
			fmt.Printf("%sLoading...%s", colorGray, colorReset+nl)
		}
	case TabIssues:
		f := state.IssueFilters
		fmt.Printf(dim+"State:"+rst+" %s"+pipe+dim+"Assignee:"+rst+" %s"+pipe+dim+"Label:"+rst+" %s"+pipe+dim+"Limit:"+rst+" "+colorGray+"%d"+rst+"%s",
			colorState(f.State), colorVal(displayAny(f.Assignee)), colorVal(displayAny(f.Label)), f.Limit, nl)
	case TabPRs:
		f := state.PRFilters
		fmt.Printf(dim+"State:"+rst+" %s"+pipe+dim+"Assignee:"+rst+" %s"+pipe+dim+"Label:"+rst+" %s"+pipe+dim+"Limit:"+rst+" "+colorGray+"%d"+rst+"%s",
			colorState(f.State), colorVal(displayAny(f.Assignee)), colorVal(displayAny(f.Label)), f.Limit, nl)
	}
	fmt.Printf("%s%s", sep, nl)
	fmt.Print(nl)
}

func printIssueDetail(issue github.Issue, maxBodyRows, termCols int) {
	sep := strings.Repeat("─", termCols)
	fmt.Printf("#%d %s\n", issue.Number, issue.Title)
	fmt.Println(sep)
	fmt.Printf("State:     %s\n", issue.State)
	fmt.Printf("Author:    %s\n", issue.Author.Login)
	fmt.Printf("Created:   %s\n", strings.TrimSuffix(strings.Split(issue.CreatedAt, "T")[0], "Z"))
	fmt.Printf("Assignees: %s\n", joinUsers(issue.Assignees))
	fmt.Printf("Labels:    %s\n", coloredLabels(issue.Labels))
	fmt.Printf("URL:       %s\n", issue.URL)
	fmt.Println(sep)
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
	sep := strings.Repeat("─", termCols)
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

	fmt.Printf("%-12s %s\n", "Labels:", coloredLabels(pr.Labels))
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

// hintBar formats alternating key/description pairs into a styled hint line,
// auto-truncating so it never exceeds the terminal width.
func hintBar(pairs ...string) string {
	_, cols := termSize()
	const sep = "  ·  "
	used := 1 // leading space
	var parts []string
	for i := 0; i+1 < len(pairs); i += 2 {
		k, d := pairs[i], pairs[i+1]
		w := 3 + len([]rune(k)) + 1 + len([]rune(d)) // "[k] d"
		if len(parts) > 0 {
			w += len(sep)
		}
		if cols > 0 && used+w > cols {
			break
		}
		used += w
		kFmt := colorGray + "[" + colorReset + colorSelect + k + colorReset + colorGray + "]" + colorReset
		parts = append(parts, kFmt+" "+colorGray+d+colorReset)
	}
	return " " + strings.Join(parts, colorGray+sep+colorReset)
}

// hintSep returns a full-width dim horizontal rule for use above hint bars.
func hintSep(rawMode bool) string {
	_, cols := termSize()
	line := colorGray + strings.Repeat("─", cols) + colorReset
	if rawMode {
		return line + "\033[K\r\n"
	}
	return line + "\n"
}

func colorState(s string) string {
	switch s {
	case "open":
		return colorGreen + s + colorReset
	case "closed":
		return colorRed + s + colorReset
	default:
		return s
	}
}

func colorVal(s string) string {
	if s == "any" {
		return colorGray + s + colorReset
	}
	return colorYellow + s + colorReset
}

func coloredLabels(labels []github.Label) string {
	if len(labels) == 0 {
		return colorGray + "—" + colorReset
	}
	parts := make([]string, len(labels))
	for i, l := range labels {
		parts[i] = colorYellow + l.Name + colorReset
	}
	return strings.Join(parts, colorGray+", "+colorReset)
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
