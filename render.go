// render.go
package main

import (
	"fmt"
	"os"
	"strings"
	"text/tabwriter"
	"time"

	"hubcap/internal/github"

	"github.com/charmbracelet/lipgloss"
)

// Lipgloss styles
var (
	// Base colors
	styleReset  = lipgloss.NewStyle()
	styleGreen  = lipgloss.NewStyle().Foreground(lipgloss.Color("2"))
	styleYellow = lipgloss.NewStyle().Foreground(lipgloss.Color("3"))
	styleRed    = lipgloss.NewStyle().Foreground(lipgloss.Color("1"))
	stylePurple = lipgloss.NewStyle().Foreground(lipgloss.Color("5"))
	styleGray   = lipgloss.NewStyle().Foreground(lipgloss.Color("8"))
	styleCyan   = lipgloss.NewStyle().Foreground(lipgloss.Color("6"))
	styleOrange = lipgloss.NewStyle().Foreground(lipgloss.Color("208"))
	styleTitle  = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("208"))
	// Box styles for messages
	errorBoxStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("1")).
			Padding(0, 1).
			Foreground(lipgloss.Color("1"))
	warningBoxStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("3")).
			Padding(0, 1).
			Foreground(lipgloss.Color("3"))
	successBoxStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("2")).
			Padding(0, 1).
			Foreground(lipgloss.Color("2"))
	infoBoxStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("6")).
			Padding(0, 1).
			Foreground(lipgloss.Color("6"))

	// Status bar style
	statusBarStyle = lipgloss.NewStyle().
			Background(lipgloss.Color("237")).
			Foreground(lipgloss.Color("252")).
			Padding(0, 1)
	statusBarAccent = lipgloss.NewStyle().
			Background(lipgloss.Color("237")).
			Foreground(lipgloss.Color("208")).
			Bold(true).
			Padding(0, 1)

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

const (
	// headerHeightFull is the line count of headerView when the filter bar is shown.
	// title(1) + ▄-spacer(1) + tabs(1) + ▄-spacer(1) + filter(1) + separator(1) + blank(1) = 7
	headerHeightFull = 7

	// headerHeightDetail is the line count when the filter bar is suppressed (detail views).
	// title(1) + ▄-spacer(1) + tabs(1) = 3
	headerHeightDetail = 3

	// metaStripHeight is the fixed line count of the sticky metadata strip
	// rendered above the viewport in detail views.
	// spacer (1) + row1 (1) + half-line gap (1) + row2 (1) + separator (1) = 5
	metaStripHeight = 5

	// metaStripExpandedHeight is the line count when the meta strip is expanded.
	// Adds: author+created row (1) + blank gap (1) = +2 → 7 total
	metaStripExpandedHeight = 7
)

// headerView returns the header as a string for use in bubbletea View() functions.
func headerView(activeTab TabID, repo string, issueFilters github.Filters, prFilters github.PRFilters, counts DashCounts, width int, detailActive bool, autoRefreshEnabled bool, autoRefreshInterval int, lastRefresh int64, currentTime int64) string {
	if width == 0 {
		width = 80
	}
	var b strings.Builder

	bg := lipgloss.Color("235")
	tabBg := lipgloss.Color("236")

	topBarStyle := lipgloss.NewStyle().Background(bg)
	versionStyle := lipgloss.NewStyle().Background(bg).Foreground(lipgloss.Color("208"))
	titleStyle := lipgloss.NewStyle().Background(bg).Foreground(lipgloss.Color("208")).Bold(true)
	helpStyle := lipgloss.NewStyle().Background(bg).Foreground(lipgloss.Color("244"))
	refreshStyle := lipgloss.NewStyle().Background(bg).Foreground(lipgloss.Color("86")).Bold(true)

	tabActiveStyle := lipgloss.NewStyle().
		Background(tabBg).
		Foreground(lipgloss.Color("86")).
		Bold(true).
		Padding(0, 2)
	tabInactiveStyle := lipgloss.NewStyle().
		Background(tabBg).
		Foreground(lipgloss.Color("244")).
		Padding(0, 2)
	tabFillStyle := lipgloss.NewStyle().Background(tabBg)

	// Filter bar — transparent background, matches the body surface.
	filterKeyStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("244"))
	filterValStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("252"))
	filterValOnStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("86"))
	filterSepStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("238"))
	filterHintStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("241"))

	// ── Line 1: version · centered title · auto-refresh · [?] help ─────────
	verText := "v" + version
	helpText := "[?] help"
	title := "Hubcap — " + repo
	titleLen := lipgloss.Width(title)
	verLen := lipgloss.Width(verText)
	helpLen := lipgloss.Width(helpText)

	// Build auto-refresh indicator
	var refreshText string
	var refreshLen int
	if autoRefreshEnabled && lastRefresh > 0 {
		iconText := "↻" // anticlockwise gapped circle arrow
		elapsed := currentTime - lastRefresh
		remaining := int64(autoRefreshInterval) - elapsed
		if remaining < 0 {
			remaining = 0
		}
		iconText += " " + formatDurationShort(time.Duration(remaining)*time.Second)
		// Make the icon much larger with extra padding
		refreshText = refreshStyle.Render("   " + iconText + "   ")
		refreshLen = lipgloss.Width(refreshText)
	} else {
		refreshLen = 0
	}

	leftPad := (width/2 - titleLen/2) - verLen - 1
	if leftPad < 1 {
		leftPad = 1
	}
	rightPad := width - verLen - 1 - leftPad - titleLen - 1 - refreshLen - helpLen - 1
	if rightPad < 1 {
		rightPad = 1
	}

	line1 := versionStyle.Render(" "+verText) +
		topBarStyle.Render(strings.Repeat(" ", leftPad)) +
		titleStyle.Render(title) +
		topBarStyle.Render(strings.Repeat(" ", rightPad)) +
		refreshText +
		topBarStyle.Render(" ") +
		helpStyle.Render(helpText+" ")
	// pad to full width
	line1Width := lipgloss.Width(line1)
	if line1Width < width {
		line1 += topBarStyle.Render(strings.Repeat(" ", width-line1Width))
	}
	b.WriteString(line1 + "\n")
	// ▄ transition: top half = title bg (235), bottom half = tab bg (236)
	b.WriteString(lipgloss.NewStyle().Background(lipgloss.Color("235")).Foreground(lipgloss.Color("236")).Render(strings.Repeat("▄", width)) + "\n")

	// ── Line 2: tabs ──────────────────────────────────────────────────────
	type tabDef struct {
		label string
		id    TabID
	}
	tabs := []tabDef{
		{"1: Dashboard", TabDashboard},
		{"2: Issues", TabIssues},
		{"3: Pull Requests", TabPRs},
	}
	var tabRow strings.Builder
	tabsWidth := 0
	for _, t := range tabs {
		var rendered string
		if t.id == activeTab {
			rendered = tabActiveStyle.Render(t.label)
		} else {
			rendered = tabInactiveStyle.Render(t.label)
		}
		tabRow.WriteString(rendered)
		tabsWidth += lipgloss.Width(rendered)
	}
	fill := width - tabsWidth
	if fill < 0 {
		fill = 0
	}
	b.WriteString(tabRow.String() + tabFillStyle.Render(strings.Repeat(" ", fill)) + "\n")

	// ── Line 3: filter/context bar ─────────────────────────────────────────
	if !detailActive {
		// ▀ half-space: top half = tab bg (236), bottom half = transparent body.
		// Gives a visual half-line of breathing room between the nav and filter text.
		b.WriteString(lipgloss.NewStyle().Foreground(lipgloss.Color("236")).Render(strings.Repeat("▀", width)) + "\n")
		sep := filterSepStyle.Render("  │  ")
		fmtFilter := func(key, val string) string {
			active := val != "" && val != "any"
			v := filterValStyle
			if active {
				v = filterValOnStyle
			}
			return filterKeyStyle.Render(key+":") + " " + v.Render(val)
		}

		var filterContent string
		switch activeTab {
		case TabIssues:
			f := issueFilters
			filterContent = "  " +
				fmtFilter("state", displayAny(f.State)) + sep +
				fmtFilter("assignee", displayAny(f.Assignee)) + sep +
				fmtFilter("label", displayAny(f.Label)) + sep +
				fmtFilter("limit", fmt.Sprintf("%d", f.Limit)) +
				filterHintStyle.Render("   [f] to change filters")
		case TabPRs:
			f := prFilters
			filterContent = "  " +
				fmtFilter("state", displayAny(f.State)) + sep +
				fmtFilter("assignee", displayAny(f.Assignee)) + sep +
				fmtFilter("label", displayAny(f.Label)) + sep +
				fmtFilter("limit", fmt.Sprintf("%d", f.Limit)) +
				filterHintStyle.Render("   [f] to change filters")
		case TabDashboard:
			countStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("205")).Bold(true)
			countOrDash := func(n int) string {
				if n == 0 {
					return filterValStyle.Render("0")
				}
				return countStyle.Render(fmt.Sprintf("%d", n))
			}
			filterContent = "  " +
				countOrDash(counts.ReviewRequests) + filterKeyStyle.Render(" review requests") + sep +
				countOrDash(counts.MyPRs) + filterKeyStyle.Render(" open PRs") + sep +
				countOrDash(counts.Assigned) + filterKeyStyle.Render(" assigned")
		}
		b.WriteString(filterContent + "\n")
		// Separator between filter bar and content body, plus a blank line of breathing room.
		b.WriteString(lipgloss.NewStyle().Foreground(lipgloss.Color("237")).Render(strings.Repeat("─", width)) + "\n")
		b.WriteString("\n")
	}

	return b.String()
}

// renderIssueMetaStrip renders the fixed 5-line metadata strip shown above the
// viewport in issue detail view.
// Collapsed (metaStripHeight = 5 lines):
//
//	Line 1 — spacer
//	Line 2 — #number  title (truncated)  ···  ● STATE
//	Line 3 — half-line gap
//	Line 4 — Assignee: …  ·  Type: …  (left)   label pills capped at 3 (right)
//	Line 5 — separator
//
// Expanded (metaStripExpandedHeight = 7 lines), adds before separator:
//
//	Line 5 — Author: …  ·  Created: …  (left)   remaining pills (right)
//	Line 6 — blank gap
//	Line 7 — separator
func renderIssueMetaStrip(issue github.Issue, width int, expanded bool) string {
	if width == 0 {
		width = 80
	}
	bg := lipgloss.Color("") // transparent — matches the body background
	s := lipgloss.NewStyle()
	titleSt := s.Foreground(lipgloss.Color("208")).Bold(true)
	numSt := s.Foreground(lipgloss.Color("69")).Bold(true)
	authorSt := s.Foreground(lipgloss.Color("252"))
	mutedSt := s.Foreground(lipgloss.Color("244"))

	// fill returns n background-colored spaces.
	fill := func(n int) string {
		if n < 0 {
			n = 0
		}
		return s.Render(strings.Repeat(" ", n))
	}

	// ── Line 1: spacer ───────────────────────────────────────────────────────
	spacer := fill(width)

	// ── Line 2: #number  title  ···  ● STATE ────────────────────────────────
	numStr := s.Render("  ") + numSt.Render(fmt.Sprintf("#%d", issue.Number)) + s.Render("  ")

	stateLabel := func() string {
		if strings.EqualFold(issue.State, "closed") {
			return s.Foreground(lipgloss.Color("196")).Bold(true).Render("CLOSED")
		}
		return s.Foreground(lipgloss.Color("83")).Bold(true).Render("OPEN")
	}()
	stateDot := s.Render(stateIndicator(issue.State, false))
	stateStr := stateDot + s.Render("  ") + stateLabel + s.Render("  ")

	numW := lipgloss.Width(numStr)
	stateW := lipgloss.Width(stateStr)
	maxTitleW := width - numW - stateW
	if maxTitleW < 5 {
		maxTitleW = 5
	}
	titleStr := titleSt.Render(truncate(issue.Title, maxTitleW))
	titleW := lipgloss.Width(titleStr)

	row1 := numStr + titleStr + fill(width-numW-titleW-stateW) + stateStr

	// ── Line 3: blank gap ────────────────────────────────────────────────────
	thinGap := fill(width)

	// ── Line 4: Assignee: name  ·  Type: name (left)  ···  label pills (right)
	var assigneeStr string
	if len(issue.Assignees) > 0 {
		assigneeStr = s.Render("  ") + mutedSt.Render("Assignee: ") + authorSt.Render(joinUsers(issue.Assignees))
	} else {
		assigneeStr = s.Render("  ") + mutedSt.Render("Assignee: ") + mutedSt.Render("unassigned")
	}

	dimDot := s.Foreground(lipgloss.Color("238")).Render("  ·  ")
	typeVal := issue.IssueType
	if typeVal == "" {
		typeVal = "—"
	}
	typeStr := dimDot + mutedSt.Render("Type: ") + authorSt.Render(typeVal)

	// Collapsed: show only the single highest-priority label + dim "+N more".
	// Expanded: show every label with no cap.
	var pillsStr string
	if len(issue.Labels) > 0 {
		var shown []github.Label
		overflow := 0
		if expanded {
			shown = issue.Labels
		} else {
			// Pick the label with the lowest labelPriority value (= most important).
			best := 0
			for i := 1; i < len(issue.Labels); i++ {
				if labelPriority(issue.Labels[i].Name) < labelPriority(issue.Labels[best].Name) {
					best = i
				}
			}
			shown = issue.Labels[best : best+1]
			overflow = len(issue.Labels) - 1
		}
		pills := make([]string, len(shown))
		for i, l := range shown {
			pills[i] = labelPill(bg, l.Name)
		}
		pillsStr = strings.Join(pills, "")
		if overflow > 0 {
			pillsStr += s.Foreground(lipgloss.Color("240")).Render(fmt.Sprintf(" +%d", overflow))
		}
		pillsStr += s.Render("  ")
	}

	leftW := lipgloss.Width(assigneeStr) + lipgloss.Width(typeStr)
	pillsW := lipgloss.Width(pillsStr)
	row2 := assigneeStr + typeStr + fill(width-leftW-pillsW) + pillsStr

	// ── Separator (last line either way) ─────────────────────────────────────
	sepLine := lipgloss.NewStyle().
		Foreground(lipgloss.Color("237")).
		Render(strings.Repeat("─", width))

	if !expanded {
		return spacer + "\n" + row1 + "\n" + thinGap + "\n" + row2 + "\n" + sepLine + "\n"
	}

	// ── Expanded lines 5–6 ────────────────────────────────────────────────────
	authorVal := "—"
	if issue.Author.Login != "" {
		authorVal = "@" + issue.Author.Login
	}
	createdVal := timeAgo(issue.CreatedAt)
	if createdVal == "" {
		createdVal = "—"
	}
	authorStr := s.Render("  ") + mutedSt.Render("Author: ") + authorSt.Render(authorVal)
	createdStr := dimDot + mutedSt.Render("Created: ") + authorSt.Render(createdVal)
	row3 := authorStr + createdStr + fill(width-lipgloss.Width(authorStr)-lipgloss.Width(createdStr))

	expandGap := fill(width)

	return spacer + "\n" + row1 + "\n" + thinGap + "\n" + row2 + "\n" + row3 + "\n" + expandGap + "\n" + sepLine + "\n"
}

// renderPRMetaStrip renders the fixed 5-line metadata strip shown above the
// viewport in PR detail view. Always produces exactly metaStripHeight lines.
//
//	Line 1 — spacer
//	Line 2 — #number  title (truncated)  ···  ● STATE
//	Line 3 — half-line gap (▁ thin rule)
//	Line 4 — ⎇ branch · author (left)  ···  review · checks · pills (right)
//	Line 5 — separator
//
// prStatusPill renders a review or CI status as a colored background chip,
// matching the visual style of labelPill.
func prStatusPill(stripBg lipgloss.Color, bg lipgloss.Color, fg lipgloss.Color, text string) string {
	chip := lipgloss.NewStyle().Background(bg).Foreground(fg).Padding(0, 1).Render(text)
	gutter := lipgloss.NewStyle().Background(stripBg).Render(" ")
	return gutter + chip + gutter
}

// renderPRMetaStrip renders the fixed 5-line metadata strip shown above the
// viewport in PR detail view. Always produces exactly metaStripHeight lines.
//
//	Line 1 — spacer
//	Line 2 — #number  title (truncated)  ···  ● STATE
//	Line 3 — blank gap
//	Line 4 — ⎇ head → base · by author (left)  ···  status + label pills (right)
//	Line 5 — separator
func renderPRMetaStrip(pr github.PullRequest, width int) string {
	if width == 0 {
		width = 80
	}
	bg := lipgloss.Color("") // transparent — matches the body background
	s := lipgloss.NewStyle()
	titleSt := s.Foreground(lipgloss.Color("208")).Bold(true)
	numSt := s.Foreground(lipgloss.Color("69")).Bold(true)
	mutedSt := s.Foreground(lipgloss.Color("244"))
	authorSt := s.Foreground(lipgloss.Color("252"))
	branchSt := s.Foreground(lipgloss.Color("86"))
	arrowSt := s.Foreground(lipgloss.Color("252")) // bright arrow
	sepSt := s.Foreground(lipgloss.Color("238"))
	dot := sepSt.Render("  ·  ")

	fill := func(n int) string {
		if n < 0 {
			n = 0
		}
		return s.Render(strings.Repeat(" ", n))
	}

	// ── Line 1: spacer ───────────────────────────────────────────────────────
	spacer := fill(width)

	// ── Line 2: #number  title  ···  ● STATE ────────────────────────────────
	numStr := s.Render("  ") + numSt.Render(fmt.Sprintf("#%d", pr.Number)) + s.Render("  ")

	stateLabel := func() string {
		switch {
		case pr.IsDraft:
			return s.Foreground(lipgloss.Color("214")).Bold(true).Render("DRAFT")
		case strings.EqualFold(pr.State, "merged"):
			return s.Foreground(lipgloss.Color("141")).Bold(true).Render("MERGED")
		case strings.EqualFold(pr.State, "closed"):
			return s.Foreground(lipgloss.Color("196")).Bold(true).Render("CLOSED")
		default:
			return s.Foreground(lipgloss.Color("83")).Bold(true).Render("OPEN")
		}
	}()
	stateDot := s.Render(stateIndicator(pr.State, pr.IsDraft))
	stateStr := stateDot + s.Render("  ") + stateLabel + s.Render("  ")

	numW := lipgloss.Width(numStr)
	stateW := lipgloss.Width(stateStr)
	maxTitleW := width - numW - stateW
	if maxTitleW < 5 {
		maxTitleW = 5
	}
	titleStr := titleSt.Render(truncate(pr.Title, maxTitleW))
	titleW := lipgloss.Width(titleStr)

	row1 := numStr + titleStr + fill(width-numW-titleW-stateW) + stateStr

	// ── Line 3: blank gap ────────────────────────────────────────────────────
	blank := fill(width)

	// ── Line 4: ⎇ head → base · by author (left)  status+label pills (right) ─
	var leftParts []string
	if pr.HeadRefName != "" {
		branchStr := branchSt.Render(truncate(pr.HeadRefName, 28))
		if pr.BaseRefName != "" {
			branchStr += arrowSt.Render(" → ") + branchSt.Render(pr.BaseRefName)
		}
		leftParts = append(leftParts, mutedSt.Render("⎇ ")+branchStr)
	}
	if pr.Author.Login != "" {
		leftParts = append(leftParts, mutedSt.Render("by ")+authorSt.Render(pr.Author.Login))
	}
	leftStr := s.Render("  ") + strings.Join(leftParts, dot)

	// Right side: review + CI status as pills, then label pills.
	var rightChips []string

	switch pr.ReviewDecision {
	case "APPROVED":
		rightChips = append(rightChips, prStatusPill(bg, "2", "0", "✓ approved"))
	case "CHANGES_REQUESTED":
		rightChips = append(rightChips, prStatusPill(bg, "1", "15", "✗ changes"))
	case "REVIEW_REQUIRED":
		rightChips = append(rightChips, prStatusPill(bg, "3", "0", "⟳ review"))
	}

	if len(pr.StatusRollup) > 0 {
		failing, pending := false, false
		for _, c := range pr.StatusRollup {
			if c.Conclusion == "FAILURE" || c.Conclusion == "ERROR" || c.Conclusion == "TIMED_OUT" {
				failing = true
			} else if c.Status != "COMPLETED" {
				pending = true
			}
		}
		switch {
		case failing:
			rightChips = append(rightChips, prStatusPill(bg, "1", "15", "✗ failing"))
		case pending:
			rightChips = append(rightChips, prStatusPill(bg, "3", "0", "… pending"))
		default:
			rightChips = append(rightChips, prStatusPill(bg, "2", "0", "✓ passing"))
		}
	}

	for _, l := range pr.Labels {
		rightChips = append(rightChips, labelPill(bg, l.Name))
	}

	var rightStr string
	if len(rightChips) > 0 {
		rightStr = strings.Join(rightChips, "") + s.Render("  ")
	}

	leftW := lipgloss.Width(leftStr)
	rightW := lipgloss.Width(rightStr)
	row2 := leftStr + fill(width-leftW-rightW) + rightStr

	// ── Line 5: separator ────────────────────────────────────────────────────
	sepLine := lipgloss.NewStyle().
		Foreground(lipgloss.Color("237")).
		Render(strings.Repeat("─", width))

	return spacer + "\n" + row1 + "\n" + blank + "\n" + row2 + "\n" + sepLine + "\n"
}

// formatDurationShort formats a time.Duration in a very compact way for the header
func formatDurationShort(d time.Duration) string {
	if d < time.Minute {
		return fmt.Sprintf("%ds", int(d.Seconds()))
	}
	if d < time.Hour {
		return fmt.Sprintf("%dm", int(d.Minutes()))
	}
	return fmt.Sprintf("%dh", int(d.Hours()))
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

// labelPillColors returns background and foreground terminal colors for a
// label pill based on its name category.
func labelPillColors(name string) (bg, fg lipgloss.Color) {
	low := strings.ToLower(name)
	switch {
	case strings.Contains(low, "priority:high"),
		strings.Contains(low, "priority:critical"),
		strings.Contains(low, "type:bug"),
		low == "bug", low == "critical", low == "blocker":
		return "1", "15" // red bg, white text
	case strings.Contains(low, "priority:medium"),
		strings.Contains(low, "type:question"),
		low == "question":
		return "3", "0" // yellow bg, black text
	case strings.Contains(low, "priority:low"):
		return "2", "0" // green bg, black text
	case strings.Contains(low, "type:enhancement"),
		strings.Contains(low, "type:feature"),
		low == "enhancement", low == "feature":
		return "6", "0" // cyan bg, black text
	case strings.Contains(low, "type:docs"),
		strings.Contains(low, "documentation"),
		low == "docs":
		return "5", "15" // purple bg, white text
	case strings.HasPrefix(low, "effort:"),
		strings.HasPrefix(low, "size:"):
		return "8", "15" // dark gray bg, white text
	default:
		return "208", "0" // orange bg, black text
	}
}

// labelPill renders a label as a colored background chip.
// stripBg is the background color of the containing row, used to color the
// gap between pills so the strip stays uniformly dark.
func labelPill(stripBg lipgloss.Color, name string) string {
	bg, fg := labelPillColors(name)
	chip := lipgloss.NewStyle().
		Background(bg).
		Foreground(fg).
		Padding(0, 1).
		Render(name)
	// Wrap with a single-space gutter in strip color on each side.
	gutter := lipgloss.NewStyle().Background(stripBg).Render(" ")
	return gutter + chip + gutter
}

// labelStyle returns a lipgloss style for a label based on its name.
// Labels are categorized by common prefixes (priority:, type:, effort:) or
// well-known keywords like "bug", "enhancement", "feature".
func labelStyle(name string) lipgloss.Style {
	low := strings.ToLower(name)
	switch {
	case strings.Contains(low, "priority:high"),
		strings.Contains(low, "priority:critical"),
		strings.Contains(low, "type:bug"),
		low == "bug",
		low == "critical",
		low == "blocker":
		return styleRed
	case strings.Contains(low, "priority:medium"),
		strings.Contains(low, "type:question"),
		low == "question":
		return styleYellow
	case strings.Contains(low, "priority:low"):
		return styleGreen
	case strings.Contains(low, "type:enhancement"),
		strings.Contains(low, "type:feature"),
		low == "enhancement",
		low == "feature":
		return styleCyan
	case strings.Contains(low, "type:docs"),
		strings.Contains(low, "documentation"),
		low == "docs":
		return stylePurple
	case strings.HasPrefix(low, "effort:"),
		strings.HasPrefix(low, "size:"):
		return styleGray
	default:
		return styleOrange
	}
}

// labelPriority returns a sort priority (lower = higher priority) for a label,
// used to pick the dominant color when multiple labels are present.
func labelPriority(name string) int {
	low := strings.ToLower(name)
	switch {
	case strings.Contains(low, "priority:critical"), strings.Contains(low, "blocker"):
		return 0
	case strings.Contains(low, "priority:high"), strings.Contains(low, "type:bug"), low == "bug":
		return 1
	case strings.Contains(low, "priority:medium"):
		return 2
	case strings.Contains(low, "type:enhancement"), strings.Contains(low, "type:feature"):
		return 3
	case strings.Contains(low, "priority:low"):
		return 4
	default:
		return 5
	}
}

// dominantLabelStyle returns the lipgloss style of the highest-priority label
// in the list, used for compact list views where a single color must represent
// the row.
func dominantLabelStyle(labels []github.Label) lipgloss.Style {
	if len(labels) == 0 {
		return styleGray
	}
	bestIdx, bestPrio := 0, 999
	for i, l := range labels {
		if p := labelPriority(l.Name); p < bestPrio {
			bestPrio = p
			bestIdx = i
		}
	}
	return labelStyle(labels[bestIdx].Name)
}

func coloredLabels(labels []github.Label) string {
	if len(labels) == 0 {
		return styleGray.Render("—")
	}
	parts := make([]string, len(labels))
	for i, l := range labels {
		parts[i] = labelStyle(l.Name).Render(l.Name)
	}
	return strings.Join(parts, styleGray.Render(", "))
}

// coloredLabelsCompact joins labels into a single colored string suitable for
// list views. The whole string is colored using the dominant label color.
func coloredLabelsCompact(labels []github.Label, maxWidth int) string {
	if len(labels) == 0 {
		return styleGray.Render(truncate("—", maxWidth))
	}
	plain := joinLabels(labels)
	return dominantLabelStyle(labels).Render(truncate(plain, maxWidth))
}

// errorBox wraps a message in a red bordered box with an error icon.
func errorBox(msg string) string {
	return errorBoxStyle.Render("✗ " + msg)
}

// warningBox wraps a message in a yellow bordered box with a warning icon.
func warningBox(msg string) string {
	return warningBoxStyle.Render("⚠ " + msg)
}

// successBox wraps a message in a green bordered box with a check icon.
func successBox(msg string) string {
	return successBoxStyle.Render("✓ " + msg)
}

// infoBox wraps a message in a cyan bordered box with an info icon.
func infoBox(msg string) string {
	return infoBoxStyle.Render("ℹ " + msg)
}

// renderStatusBar prints a single-line status bar at the bottom showing repo,
// active tab, and an optional stats string.
func renderStatusBar(state *AppState, stats string) string {
	tabName := "Dashboard"
	switch state.ActiveTab {
	case TabIssues:
		tabName = "Issues"
	case TabPRs:
		tabName = "Pull Requests"
	}
	repo := state.Repo
	if repo == "" {
		repo = "(no repo)"
	}
	left := statusBarAccent.Render(tabName)
	mid := statusBarStyle.Render(repo)
	right := ""
	if stats != "" {
		right = statusBarStyle.Render(stats)
	}

	_, cols := termSize()
	content := left + mid + right
	plain := tabName + "  " + repo + "  " + stats
	pad := cols - len([]rune(plain)) - 4
	if pad < 0 {
		pad = 0
	}
	return content + statusBarStyle.Render(strings.Repeat(" ", pad))
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

// timeAgo returns a short human-readable relative time string for an ISO 8601
// timestamp (e.g. "2h ago", "3d ago", "2mo ago"). Returns "" on parse error.
func timeAgo(ts string) string {
	if ts == "" {
		return ""
	}
	t, err := time.Parse(time.RFC3339, ts)
	if err != nil {
		// Try without timezone suffix (some GitHub responses omit the Z)
		t, err = time.Parse("2006-01-02T15:04:05", strings.TrimSuffix(ts, "Z"))
		if err != nil {
			return ""
		}
	}
	d := time.Since(t)
	switch {
	case d < time.Minute:
		return "just now"
	case d < time.Hour:
		return fmt.Sprintf("%dm ago", int(d.Minutes()))
	case d < 24*time.Hour:
		return fmt.Sprintf("%dh ago", int(d.Hours()))
	case d < 7*24*time.Hour:
		return fmt.Sprintf("%dd ago", int(d.Hours()/24))
	case d < 30*24*time.Hour:
		return fmt.Sprintf("%dw ago", int(d.Hours()/(24*7)))
	case d < 365*24*time.Hour:
		return fmt.Sprintf("%dmo ago", int(d.Hours()/(24*30)))
	default:
		return fmt.Sprintf("%dy ago", int(d.Hours()/(24*365)))
	}
}

func fatal(err error) {
	fmt.Fprintln(os.Stderr, err)
	os.Exit(1)
}
