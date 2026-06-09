// render.go
package main

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"

	"hubcap/internal/github"

	"github.com/charmbracelet/bubbles/viewport"
	"github.com/charmbracelet/glamour"
	glamourstyles "github.com/charmbracelet/glamour/styles"
	"github.com/charmbracelet/lipgloss"
)


const (
	// headerHeightFull is the line count of headerView when the filter bar is shown.
	// title(1) + ▄-spacer(1) + tabs(1) + ▄-spacer(1) + filter(1) + separator(1) + blank(1) = 7
	headerHeightFull = 7

	// headerHeightDetail is the line count when the filter bar is suppressed (detail views).
	// title(1) + ▄-spacer(1) + tabs(1) + ▀-spacer(1) = 4
	headerHeightDetail = 4

	// metaStripHeight is the fixed line count of the sticky metadata strip
	// rendered above the viewport in detail views.
	// spacer (1) + row1 (1) + half-line gap (1) + row2 (1) + separator (1) = 5
	metaStripHeight = 5

	// metaStripExpandedHeight is the line count when the meta strip is expanded.
	// spacer(1) + title(1) + thinGap(1) + assignee/type/author/created(1)
	// + halfGap(1) + labels(1) + halfGap(1) + separator(1) = 8
	metaStripExpandedHeight = 8

	// detailViewportOverhead is the number of non-viewport lines consumed in
	// the detail layout aside from the header and meta strip:
	//   border(2) + spacer_below_viewport(1) + min_footer(1) = 4
	// For Comfortable theme, add 3 more (separator + 3 button rows).
	detailViewportOverheadBase = 4
)

// detailViewportHeight returns the exact number of lines the viewport can use
// in a detail view, given the total terminal height, meta strip height, and
// theme. It accounts for: border(2), header, meta strip, spacer(1), footer.
func detailViewportHeight(termH, metaH int, theme UITheme) int {
	footerL := 1
	if theme == ThemeComfortable {
		footerL = 4
	}
	h := termH - detailViewportOverheadBase - headerHeightDetail - metaH - (footerL - 1)
	if h < 1 {
		h = 1
	}
	return h
}

// headerView returns the header as a string for use in bubbletea View() functions.
func headerView(activeTab TabID, repo string, issueFilters github.Filters, prFilters github.PRFilters, counts DashCounts, width int, detailActive bool, autoRefreshEnabled bool, autoRefreshInterval int, lastRefresh int64, currentTime int64, pal Palette) string {
	if width == 0 {
		width = 80
	}
	var b strings.Builder

	bg := pal.BgHeader
	tabBg := pal.BgTabs

	topBarStyle := lipgloss.NewStyle().Background(bg)
	versionStyle := lipgloss.NewStyle().Background(bg).Foreground(pal.TextDim)
	titleStyle := lipgloss.NewStyle().Background(bg).Foreground(pal.Title).Bold(true)
	helpStyle := lipgloss.NewStyle().Background(bg).Foreground(pal.TextMuted)
	refreshStyle := lipgloss.NewStyle().Background(bg).Foreground(pal.Accent).Bold(true)

	tabActiveStyle := lipgloss.NewStyle().
		Background(tabBg).
		Foreground(pal.Accent).
		Bold(true).
		Padding(0, 2)
	tabInactiveStyle := lipgloss.NewStyle().
		Background(tabBg).
		Foreground(pal.TextMuted).
		Padding(0, 2)
	tabFillStyle := lipgloss.NewStyle().Background(tabBg)

	// Filter bar — body background so light themes don't show terminal default.
	filterKeyStyle := lipgloss.NewStyle().Foreground(pal.TextMuted).Background(pal.BgBody)
	filterValStyle := lipgloss.NewStyle().Foreground(pal.Text).Background(pal.BgBody)
	filterValOnStyle := lipgloss.NewStyle().Foreground(pal.Accent).Background(pal.BgBody)
	filterSepStyle := lipgloss.NewStyle().Foreground(pal.TextFaint).Background(pal.BgBody)
	filterHintStyle := lipgloss.NewStyle().Foreground(pal.TextDim).Background(pal.BgBody)

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
	// ▄ transition: top half = title bg, bottom half = tab bg
	b.WriteString(lipgloss.NewStyle().Background(pal.BgHeader).Foreground(pal.BgTabs).Render(strings.Repeat("▄", width)) + "\n")

	// ── Line 2: tabs ──────────────────────────────────────────────────────
	tabActiveStyle = tabActiveStyle.Underline(true)

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
	// ▀ half-space: top half = tab bg, bottom half = body bg.
	// Provides a visual half-line of breathing room below the tab bar in all views.
	b.WriteString(lipgloss.NewStyle().Foreground(pal.BgTabs).Background(pal.BgBody).Render(strings.Repeat("▀", width)) + "\n")

	// ── Line 4+: filter/context bar (list views only) ──────────────────────
	if !detailActive {
		sep := filterSepStyle.Render("  │  ")
		indent := filterKeyStyle.Render("  ")
		fmtFilter := func(key, val string) string {
			active := val != "" && val != "any"
			v := filterValStyle
			if active {
				v = filterValOnStyle
			}
			return filterKeyStyle.Render(key+": ") + v.Render(val)
		}

		var filterContent string
		switch activeTab {
		case TabIssues:
			f := issueFilters
			filterContent = indent +
				fmtFilter("state", displayAny(f.State)) + sep +
				fmtFilter("assignee", displayAny(f.Assignee)) + sep +
				fmtFilter("label", displayAny(f.Label)) + sep +
				fmtFilter("limit", fmt.Sprintf("%d", f.Limit)) +
				filterHintStyle.Render("   [f] to change filters")
		case TabPRs:
			f := prFilters
			filterContent = indent +
				fmtFilter("state", displayAny(f.State)) + sep +
				fmtFilter("assignee", displayAny(f.Assignee)) + sep +
				fmtFilter("label", displayAny(f.Label)) + sep +
				fmtFilter("limit", fmt.Sprintf("%d", f.Limit)) +
				filterHintStyle.Render("   [f] to change filters")
		case TabDashboard:
			countStyle := lipgloss.NewStyle().Foreground(pal.Accent).Bold(true).Background(pal.BgBody)
			countOrDash := func(n int) string {
				if n == 0 {
					return filterValStyle.Render("0")
				}
				return countStyle.Render(fmt.Sprintf("%d", n))
			}
			filterContent = indent +
				countOrDash(counts.ReviewRequests) + filterKeyStyle.Render(" review requests") + sep +
				countOrDash(counts.MyPRs) + filterKeyStyle.Render(" my PRs") + sep +
				countOrDash(counts.Assigned) + filterKeyStyle.Render(" assigned issues") + sep +
				countOrDash(counts.AssignedPRs) + filterKeyStyle.Render(" assigned PRs")
		}
		b.WriteString(filterContent + "\n")
		// Separator between filter bar and content body, plus a blank line of breathing room.
		b.WriteString(lipgloss.NewStyle().Foreground(pal.TextFaint).Background(pal.BgBody).Render(strings.Repeat("─", width)) + "\n")
		b.WriteString(lipgloss.NewStyle().Background(pal.BgBody).Width(width).Render("") + "\n")
	}

	return b.String()
}

// metaSepLine renders the full-width separator line shared by all meta strips.
func metaSepLine(width int, pal Palette) string {
	return lipgloss.NewStyle().Foreground(pal.TextMuted).Background(pal.BgBody).Render(strings.Repeat("─", width))
}

// issueSepLine renders the separator for the issue meta strip.
// It embeds a centred [e] expand / [e] collapse shortcut hint so users can
// discover the expand feature without it taking up a footer button slot.
func issueSepLine(width int, expanded bool, pal Palette) string {
	dashSt := lipgloss.NewStyle().Foreground(pal.TextMuted).Background(pal.BgBody)
	bracketSt := lipgloss.NewStyle().Foreground(pal.TextMuted).Background(pal.BgBody)
	keySt := lipgloss.NewStyle().Foreground(pal.Meta).Bold(true).Background(pal.BgBody)
	labelSt := lipgloss.NewStyle().Foreground(pal.Text).Background(pal.BgBody)

	action := "expand"
	if expanded {
		action = "collapse"
	}
	hint := bracketSt.Render("[") + keySt.Render("e") + bracketSt.Render("]") +
		labelSt.Render(" "+action+" ")
	hintW := lipgloss.Width(hint)
	left := (width - hintW) / 2
	right := width - hintW - left
	if left < 0 {
		left = 0
	}
	if right < 0 {
		right = 0
	}
	return dashSt.Render(strings.Repeat("─", left)) + hint + dashSt.Render(strings.Repeat("─", right))
}

// viewportWithScrollHint overlays a light-grey "↓ N%" badge at the
// bottom-right corner of the viewport when there is more content to scroll.
//
// The viewport pads every line to vp.Width with spaces, so we can't simply
// append the badge (that would push the line past vp.Width and cause the
// terminal to wrap it onto a new row). Instead we reconstruct the last line
// as (vp.Width - badgeW) spaces + badge, keeping the total exactly vp.Width.
func viewportWithScrollHint(vp viewport.Model, pal Palette) string {
	view := vp.View()
	if vp.AtBottom() {
		return view
	}
	pct := int(vp.ScrollPercent() * 100)
	badge := lipgloss.NewStyle().
		Background(pal.TextDim).
		Foreground(pal.TextBold).
		Padding(0, 1).
		Render(fmt.Sprintf("↓ %d%%", pct))
	badgeW := lipgloss.Width(badge)

	lines := strings.Split(view, "\n")
	idx := len(lines) - 1
	if idx < 0 {
		return view
	}
	spaceW := vp.Width - badgeW
	if spaceW < 0 {
		spaceW = 0
	}
	lines[idx] = lipgloss.NewStyle().Background(pal.BgBody).Render(strings.Repeat(" ", spaceW)) + badge
	return strings.Join(lines, "\n")
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
func renderIssueMetaStrip(issue github.Issue, width int, expanded bool, pal Palette) string {
	if width == 0 {
		width = 80
	}
	bg := pal.BgBody
	s := lipgloss.NewStyle().Background(bg)
	titleSt := s.Foreground(pal.Title).Bold(true)
	numSt := s.Foreground(pal.Number).Bold(true)
	authorSt := s.Foreground(pal.Text)
	mutedSt := s.Foreground(pal.TextMuted)

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
			return s.Foreground(pal.StatusClosed).Bold(true).Render("CLOSED")
		}
		return s.Foreground(pal.StatusOpen).Bold(true).Render("OPEN")
	}()
	stateDot := s.Render(stateIndicator(issue.State, false, pal))
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

	dimDot := s.Foreground(pal.TextFaint).Render("  ·  ")
	typeVal := issue.IssueType
	if typeVal == "" {
		typeVal = "—"
	}
	typeStr := dimDot + mutedSt.Render("Type: ") + authorSt.Render(typeVal)

	// Collapsed: single highest-priority pill right-aligned on row 2.
	// Expanded: pills move to their own dedicated row 6 (row 2 stays clean).
	buildPills := func(labels []github.Label) string {
		if len(labels) == 0 {
			return ""
		}
		pills := make([]string, len(labels))
		for i, l := range labels {
			pills[i] = labelPill(bg, l.Name, pal)
		}
		return strings.Join(pills, "") + s.Render("  ")
	}

	var collapsedPill string // right side of row2 in collapsed mode
	if !expanded && len(issue.Labels) > 0 {
		// Pick up to 2 highest-priority labels by sorting indices.
		const maxCollapsed = 2
		idxs := make([]int, len(issue.Labels))
		for i := range idxs {
			idxs[i] = i
		}
		// Partial insertion sort — only need the top maxCollapsed.
		for i := 0; i < len(idxs) && i < maxCollapsed; i++ {
			best := i
			for j := i + 1; j < len(idxs); j++ {
				if labelPriority(issue.Labels[idxs[j]].Name) < labelPriority(issue.Labels[idxs[best]].Name) {
					best = j
				}
			}
			idxs[i], idxs[best] = idxs[best], idxs[i]
		}
		shown := idxs
		if len(shown) > maxCollapsed {
			shown = shown[:maxCollapsed]
		}
		overflow := len(issue.Labels) - len(shown)
		for _, i := range shown {
			collapsedPill += labelPill(bg, issue.Labels[i].Name, pal)
		}
		if overflow > 0 {
			collapsedPill += s.Foreground(pal.Text).Render(fmt.Sprintf(" +%d", overflow))
		}
		collapsedPill += s.Render("  ")
	}

	leftW := lipgloss.Width(assigneeStr) + lipgloss.Width(typeStr)
	pillW := lipgloss.Width(collapsedPill)
	row2 := assigneeStr + typeStr + fill(width-leftW-pillW) + collapsedPill

	// ── Separator — always shows [e] expand / [e] collapse hint centred ─────
	sepLine := issueSepLine(width, expanded, pal)

	if !expanded {
		return spacer + "\n" + row1 + "\n" + thinGap + "\n" + row2 + "\n" + sepLine + "\n"
	}

	// ── Expanded lines 5–8 ────────────────────────────────────────────────────
	halfGap := fill(width)

	// Row 4 (expanded): append Author · Created onto the Assignee/Type row
	authorVal := "—"
	if issue.Author.Login != "" {
		authorVal = "@" + issue.Author.Login
	}
	createdVal := timeAgo(issue.CreatedAt)
	if createdVal == "" {
		createdVal = "—"
	}
	authorStr := mutedSt.Render("Author: ") + authorSt.Render(authorVal)
	createdStr := dimDot + mutedSt.Render("Created: ") + authorSt.Render(createdVal)
	rightStr := authorStr + createdStr + s.Render("  ")
	expandedLeftW := lipgloss.Width(assigneeStr) + lipgloss.Width(typeStr)
	expandedRightW := lipgloss.Width(rightStr) + lipgloss.Width(collapsedPill)
	row2 = assigneeStr + typeStr + fill(width-expandedLeftW-expandedRightW) + rightStr + collapsedPill

	// Row 6: label pills with single leading space
	allPills := buildPills(issue.Labels)
	allPillsW := lipgloss.Width(allPills)
	row4 := " " + allPills + fill(width-1-allPillsW)

	// spacer + title + thinGap + row2 + halfGap + row4 + halfGap + sep = 8 lines
	return spacer + "\n" + row1 + "\n" + thinGap + "\n" + row2 + "\n" + halfGap + "\n" + row4 + "\n" + halfGap + "\n" + sepLine + "\n"
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
func renderPRMetaStrip(pr github.PullRequest, width int, pal Palette) string {
	if width == 0 {
		width = 80
	}
	bg := pal.BgBody
	s := lipgloss.NewStyle().Background(bg)
	titleSt := s.Foreground(pal.Title).Bold(true)
	numSt := s.Foreground(pal.Number).Bold(true)
	mutedSt := s.Foreground(pal.TextMuted)
	authorSt := s.Foreground(pal.Text)
	branchSt := s.Foreground(pal.Accent)
	arrowSt := s.Foreground(pal.Text)
	sepSt := s.Foreground(pal.TextFaint)
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
			return s.Foreground(pal.StatusDraft).Bold(true).Render("DRAFT")
		case strings.EqualFold(pr.State, "merged"):
			return s.Foreground(pal.StatusMerged).Bold(true).Render("MERGED")
		case strings.EqualFold(pr.State, "closed"):
			return s.Foreground(pal.StatusClosed).Bold(true).Render("CLOSED")
		default:
			return s.Foreground(pal.StatusOpen).Bold(true).Render("OPEN")
		}
	}()
	stateDot := s.Render(stateIndicator(pr.State, pr.IsDraft, pal))
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
		rightChips = append(rightChips, prStatusPill(bg, pal.LabelSuccess, pal.LabelSuccessFg, "✓ approved"))
	case "CHANGES_REQUESTED":
		rightChips = append(rightChips, prStatusPill(bg, pal.LabelDanger, pal.LabelDangerFg, "✗ changes"))
	case "REVIEW_REQUIRED":
		rightChips = append(rightChips, prStatusPill(bg, pal.LabelWarn, pal.LabelWarnFg, "⟳ review"))
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
			rightChips = append(rightChips, prStatusPill(bg, pal.LabelDanger, pal.LabelDangerFg, "✗ failing"))
		case pending:
			rightChips = append(rightChips, prStatusPill(bg, pal.LabelWarn, pal.LabelWarnFg, "… pending"))
		default:
			rightChips = append(rightChips, prStatusPill(bg, pal.LabelSuccess, pal.LabelSuccessFg, "✓ passing"))
		}
	}

	for _, l := range pr.Labels {
		rightChips = append(rightChips, labelPill(bg, l.Name, pal))
	}

	var rightStr string
	if len(rightChips) > 0 {
		rightStr = strings.Join(rightChips, "") + s.Render("  ")
	}

	leftW := lipgloss.Width(leftStr)
	rightW := lipgloss.Width(rightStr)
	row2 := leftStr + fill(width-leftW-rightW) + rightStr

	// ── Line 5: separator ────────────────────────────────────────────────────
	sepLine := metaSepLine(width, pal)

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


func stateIndicator(state string, isDraft bool, pal Palette) string {
	switch {
	case isDraft:
		return lipgloss.NewStyle().Foreground(pal.StatusDraft).Render("◐")
	case strings.EqualFold(state, "merged"):
		return lipgloss.NewStyle().Foreground(pal.StatusMerged).Render("✓")
	case strings.EqualFold(state, "closed"):
		return lipgloss.NewStyle().Foreground(pal.StatusClosed).Render("✗")
	case strings.EqualFold(state, "open"):
		return lipgloss.NewStyle().Foreground(pal.StatusOpen).Render("●")
	default:
		return lipgloss.NewStyle().Foreground(pal.TextMuted).Render("○")
	}
}

func summarizeChecks(checks []github.CheckRun, bg lipgloss.Color, pal Palette) string {
	st := func(fg lipgloss.Color) lipgloss.Style {
		return lipgloss.NewStyle().Foreground(fg).Background(bg)
	}
	if len(checks) == 0 {
		return st(pal.TextFaint).Render("—")
	}
	pending := false
	for _, c := range checks {
		if c.Conclusion == "FAILURE" || c.Conclusion == "ERROR" || c.Conclusion == "TIMED_OUT" {
			return st(pal.CheckFail).Render("✗")
		}
		if c.Status != "COMPLETED" {
			pending = true
		}
	}
	if pending {
		return st(pal.CheckPending).Render("…")
	}
	return st(pal.CheckPass).Render("✓")
}


// labelPillColors returns background and foreground terminal colors for a
// label pill based on its name category.
func labelPillColors(name string, pal Palette) (bg, fg lipgloss.Color) {
	low := strings.ToLower(name)
	switch {
	case strings.Contains(low, "priority:high"),
		strings.Contains(low, "priority:critical"),
		strings.Contains(low, "type:bug"),
		low == "bug", low == "critical", low == "blocker":
		return pal.LabelDanger, pal.LabelDangerFg
	case strings.Contains(low, "priority:medium"),
		strings.Contains(low, "type:question"),
		low == "question":
		return pal.LabelWarn, pal.LabelWarnFg
	case strings.Contains(low, "priority:low"):
		return pal.LabelSuccess, pal.LabelSuccessFg
	case strings.Contains(low, "type:enhancement"),
		strings.Contains(low, "type:feature"),
		low == "enhancement", low == "feature":
		return pal.LabelFeature, pal.LabelFeatureFg
	case strings.Contains(low, "type:docs"),
		strings.Contains(low, "documentation"),
		low == "docs":
		return pal.LabelDocs, pal.LabelDocsFg
	case strings.HasPrefix(low, "effort:"),
		strings.HasPrefix(low, "size:"):
		return pal.LabelSubtle, pal.LabelSubtleFg
	default:
		return pal.LabelDefault, pal.LabelDefaultFg
	}
}

// labelPill renders a label as a colored background chip.
// stripBg is the background color of the containing row, used to color the
// gap between pills so the strip stays uniformly dark.
func labelPill(stripBg lipgloss.Color, name string, pal Palette) string {
	bg, fg := labelPillColors(name, pal)
	chip := lipgloss.NewStyle().Background(bg).Foreground(fg).Padding(0, 1).Render(name)
	gutter := lipgloss.NewStyle().Background(stripBg).Render(" ")
	return gutter + chip + gutter
}

// labelStyle returns a lipgloss style for a label based on its name.
// Labels are categorized by common prefixes (priority:, type:, effort:) or
// well-known keywords like "bug", "enhancement", "feature".
func labelStyle(name string, pal Palette) lipgloss.Style {
	low := strings.ToLower(name)
	s := lipgloss.NewStyle()
	switch {
	case strings.Contains(low, "priority:high"),
		strings.Contains(low, "priority:critical"),
		strings.Contains(low, "type:bug"),
		low == "bug", low == "critical", low == "blocker":
		return s.Foreground(pal.LabelDanger)
	case strings.Contains(low, "priority:medium"),
		strings.Contains(low, "type:question"),
		low == "question":
		return s.Foreground(pal.LabelWarn)
	case strings.Contains(low, "priority:low"):
		return s.Foreground(pal.LabelSuccess)
	case strings.Contains(low, "type:enhancement"),
		strings.Contains(low, "type:feature"),
		low == "enhancement", low == "feature":
		return s.Foreground(pal.LabelFeature)
	case strings.Contains(low, "type:docs"),
		strings.Contains(low, "documentation"),
		low == "docs":
		return s.Foreground(pal.LabelDocs)
	case strings.HasPrefix(low, "effort:"),
		strings.HasPrefix(low, "size:"):
		return s.Foreground(pal.LabelSubtle)
	default:
		return s.Foreground(pal.LabelDefault)
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


// errorBox wraps a message in a red bordered box with an error icon.
func errorBox(msg string, pal Palette) string {
	return lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(pal.Danger).
		Padding(0, 1).
		Foreground(pal.Danger).
		Render("✗ " + msg)
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

// rendererCache caches glamour TermRenderers keyed by (width, docBg).
var rendererCache sync.Map // map[rendererKey]*glamour.TermRenderer

type rendererKey struct {
	width int
	bg    string
}

// renderMarkdown renders a Markdown string to ANSI-styled terminal output.
// docBg is the hex background color of the containing area (e.g. "#F2ECD8" for
// parchment). When non-empty, glamour is configured with a matching document
// background so that ANSI reset codes emitted inside the rendered content reset
// to docBg instead of the terminal default. Pass "" for dark/auto-detected themes.
// Falls back to the raw string if glamour fails so the body is never blank.
func renderMarkdown(body string, width int, docBg string) string {
	if body == "" {
		return ""
	}
	key := rendererKey{width, docBg}
	var r *glamour.TermRenderer
	if cached, ok := rendererCache.Load(key); ok {
		r = cached.(*glamour.TermRenderer)
	} else {
		var opt glamour.TermRendererOption
		if docBg != "" {
			// Light theme: copy the light style, set the document background to
			// match the surrounding area so ANSI resets don't bleed through, and
			// zero the document margin (glamour renders it with the *parent* block
			// style = terminal default, causing dark left bars). The outer
			// Padding(0,2).Background(BgBody) in the view functions provides
			// equivalent indentation with the correct background.
			cfg := glamourstyles.LightStyleConfig
			cfg.Document.StylePrimitive.BackgroundColor = strPtr(docBg)
			cfg.Document.Margin = nil
			// Deep-copy the Chroma block and set its background to docBg so that
			// code blocks render consistently on the parchment background after
			// injectDocBg replaces every \x1b[0m with \x1b[0m+<parchment bg>.
			// Without this, chroma tokens (which only carry fg codes) would
			// inherit the injected parchment bg instead of the dark code-block bg.
			if cfg.CodeBlock.Chroma != nil {
				chromaCopy := *cfg.CodeBlock.Chroma
				chromaCopy.Background.BackgroundColor = strPtr(docBg)
				cfg.CodeBlock.Chroma = &chromaCopy
			}
			opt = glamour.WithStyles(cfg)
		} else {
			// Dark theme: always use the standard dark style, never auto-detect.
			// Auto-detection (termenv.HasDarkBackground) can return false on some
			// terminals, causing glamour to pick the light style and making the
			// viewport body appear white on dark-palette themes.
			opt = glamour.WithStandardStyle("dark")
		}
		var err error
		r, err = glamour.NewTermRenderer(opt, glamour.WithWordWrap(width))
		if err != nil {
			return body
		}
		rendererCache.Store(key, r)
	}
	out, err := r.Render(body)
	if err != nil {
		return body
	}
	if docBg != "" {
		// Trim blank lines that glamour emits from Document BlockPrefix/BlockSuffix
		// ("\n" each). Those bare newlines render on the terminal default (black)
		// regardless of per-line bg injection because they carry no ANSI codes.
		// The outer Padding(0,2).Background(BgBody) wrapper provides spacing.
		out = strings.TrimLeft(out, "\n")
		out = strings.TrimRight(out, "\n")
		out = injectDocBg(out, docBg)
	}
	return out
}

// injectDocBg ensures every line of s renders on docBg by:
//  1. Replacing every ANSI reset with reset+bgCode so that plain-space padding
//     emitted by lipgloss (e.g. table cell margins) inherits parchment after
//     any reset rather than the terminal default.
//  2. Prepending bgCode to EVERY line so that viewport's line-split model
//     (SetContent splits on \n; View rejoins and pads independently) doesn't
//     require ANSI state to carry across newlines.
func injectDocBg(s, docBg string) string {
	r, g, b := hexToRGB(docBg)
	bgCode := fmt.Sprintf("\x1b[48;2;%d;%d;%dm", r, g, b)
	// Step 1: inject after every reset so trailing padding spaces get parchment bg.
	s = strings.ReplaceAll(s, "\x1b[0m", "\x1b[0m"+bgCode)
	s = strings.ReplaceAll(s, "\x1b[m", "\x1b[m"+bgCode)
	// Step 2: prefix every line so each line is self-contained.
	lines := strings.Split(s, "\n")
	for i, line := range lines {
		lines[i] = bgCode + line
	}
	return strings.Join(lines, "\n")
}

// hexToRGB converts a CSS hex color string (e.g. "#F2ECD8" or "F2ECD8") to
// R, G, B integer components in the range [0, 255].
func hexToRGB(hex string) (r, g, b int) {
	hex = strings.TrimPrefix(hex, "#")
	val, err := strconv.ParseInt(hex, 16, 32)
	if err != nil {
		return 0, 0, 0
	}
	return int((val >> 16) & 0xFF), int((val >> 8) & 0xFF), int(val & 0xFF)
}

func strPtr(s string) *string { return &s }

func fatal(err error) {
	fmt.Fprintln(os.Stderr, err)
	os.Exit(1)
}
