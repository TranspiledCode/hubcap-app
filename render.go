// render.go
package main

import (
	"fmt"
	"os"
	"strings"
	"text/tabwriter"

	"hubcap/internal/github"

	"github.com/charmbracelet/lipgloss"
)

// Lipgloss styles
var (
	// Base colors
	styleReset   = lipgloss.NewStyle()
	styleGreen   = lipgloss.NewStyle().Foreground(lipgloss.Color("2"))
	styleYellow  = lipgloss.NewStyle().Foreground(lipgloss.Color("3"))
	styleRed     = lipgloss.NewStyle().Foreground(lipgloss.Color("1"))
	stylePurple  = lipgloss.NewStyle().Foreground(lipgloss.Color("5"))
	styleGray    = lipgloss.NewStyle().Foreground(lipgloss.Color("8"))
	styleCyan    = lipgloss.NewStyle().Foreground(lipgloss.Color("6"))
	styleOrange  = lipgloss.NewStyle().Foreground(lipgloss.Color("208"))
	styleTitle   = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("208"))
	styleSelBg   = lipgloss.NewStyle().Background(lipgloss.Color("23"))

	// Legacy compatibility constants (for gradual migration)
	colorReset  = ""
	colorGreen  = ""
	colorYellow = ""
	colorRed    = ""
	colorPurple = ""
	colorGray   = ""
	colorSelect = ""
	colorOrange = ""
	colorTitle  = ""
	colorSelBg  = ""
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
		myWorkLabel = styleCyan.Render(myWorkLabel)
	case TabIssues:
		issuesLabel = styleCyan.Render(issuesLabel)
	case TabPRs:
		prsLabel = styleCyan.Render(prsLabel)
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
	fmt.Printf("%s%s", styleTitle.Render(verText+strings.Repeat(" ", gap)+helpText), nl)

	// Line 2: centered title
	title := "Hubcap — " + state.Repo
	tLen := len([]rune(title))
	lPad, rPad := 0, 0
	if cols > tLen {
		lPad = (cols - tLen) / 2
		rPad = cols - tLen - lPad
	}
	fmt.Printf("%s%s", styleTitle.Render(strings.Repeat(" ", lPad)+title+strings.Repeat(" ", rPad)), nl)

	// Line 3: blank
	fmt.Printf("%s%s", styleTitle.Render(strings.Repeat(" ", cols)), nl)
	fmt.Printf("%s%s", sep, nl)
	fmt.Printf("%s%s%s%s%s", myWorkLabel, issuesLabel, prsLabel, tabPad, nl)
	fmt.Printf("%s%s", sep, nl)

	dim := styleGray
	switch state.ActiveTab {
	case TabDashboard:
		if state.DashboardStatus != "" {
			fmt.Printf("%s%s", state.DashboardStatus, nl)
		} else {
			fmt.Printf("%s%s", styleGray.Render("Loading..."), nl)
		}
	case TabIssues:
		f := state.IssueFilters
		fmt.Printf("%s %s %s %s %s %s %s %s %s",
			dim.Render("State:"), colorState(f.State),
			dim.Render("Assignee:"), colorVal(displayAny(f.Assignee)),
			dim.Render("Label:"), colorVal(displayAny(f.Label)),
			dim.Render("Limit:"), styleGray.Render(fmt.Sprintf("%d", f.Limit)), nl)
	case TabPRs:
		f := state.PRFilters
		fmt.Printf("%s %s %s %s %s %s %s %s %s",
			dim.Render("State:"), colorState(f.State),
			dim.Render("Assignee:"), colorVal(displayAny(f.Assignee)),
			dim.Render("Label:"), colorVal(displayAny(f.Label)),
			dim.Render("Limit:"), styleGray.Render(fmt.Sprintf("%d", f.Limit)), nl)
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

	stateColor := styleGreen
	switch {
	case pr.IsDraft:
		stateColor = styleYellow
	case pr.State == "merged":
		stateColor = stylePurple
	case pr.State == "closed":
		stateColor = styleRed
	}
	stateStr := pr.State
	if pr.IsDraft {
		stateStr = "draft"
	}

	fmt.Printf("%-12s %s\n", "State:", stateColor.Render(stateStr))
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

	reviewColor := styleYellow
	switch pr.ReviewDecision {
	case "APPROVED":
		reviewColor = styleGreen
	case "CHANGES_REQUESTED":
		reviewColor = styleRed
	}
	reviewStr := pr.ReviewDecision
	if reviewStr == "" {
		reviewStr = "—"
	}
	fmt.Printf("%-12s %s\n", "Review:", reviewColor.Render(reviewStr))
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
		return styleYellow.Render("◐")
	case strings.EqualFold(state, "merged"):
		return stylePurple.Render("✓")
	case strings.EqualFold(state, "closed"):
		return styleRed.Render("✗")
	case strings.EqualFold(state, "open"):
		return styleGreen.Render("●")
	default:
		return styleGray.Render("○")
	}
}

func summarizeChecks(checks []github.CheckRun) string {
	if len(checks) == 0 {
		return "—"
	}
	pending := false
	for _, c := range checks {
		if c.Conclusion == "FAILURE" || c.Conclusion == "ERROR" || c.Conclusion == "TIMED_OUT" {
			return styleRed.Render("✗")
		}
		if c.Status != "COMPLETED" {
			pending = true
		}
	}
	if pending {
		return styleYellow.Render("…")
	}
	return styleGreen.Render("✓")
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
		kFmt := styleGray.Render("[" + styleCyan.Render(k) + "]")
		parts = append(parts, kFmt+" "+styleGray.Render(d))
	}
	return " " + strings.Join(parts, styleGray.Render(sep))
}

// hintSep returns a full-width dim horizontal rule for use above hint bars.
func hintSep(rawMode bool) string {
	_, cols := termSize()
	line := styleGray.Render(strings.Repeat("─", cols))
	if rawMode {
		return line + "\033[K\r\n"
	}
	return line + "\n"
}

func colorState(s string) string {
	switch s {
	case "open":
		return styleGreen.Render(s)
	case "closed":
		return styleRed.Render(s)
	default:
		return s
	}
}

func colorVal(s string) string {
	if s == "any" {
		return styleGray.Render(s)
	}
	return styleYellow.Render(s)
}

func coloredLabels(labels []github.Label) string {
	if len(labels) == 0 {
		return styleGray.Render("—")
	}
	parts := make([]string, len(labels))
	for i, l := range labels {
		parts[i] = styleYellow.Render(l.Name)
	}
	return strings.Join(parts, styleGray.Render(", "))
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
			return out + "\n" + styleGray.Render("… open in browser to read more")
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
