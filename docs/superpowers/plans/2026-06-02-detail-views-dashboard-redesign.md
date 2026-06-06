# Detail Views & Dashboard Redesign — Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Add sticky metadata strips to issue/PR detail views, eliminate the double-footer, and trim the dashboard to three personal sections with improved row formatting.

**Architecture:** Six sequential tasks — header constants first (Tasks 1–2 establish shared foundations), then issue/PR detail views (Tasks 3–4), then footer suppression (Task 5), then dashboard cleanup (Task 6). Each task compiles and builds cleanly before the next begins.

**Tech Stack:** Go 1.24, BubbleTea, Lipgloss. Project at `/Users/joshua/Development/hubcap/hubcap-app/`. Build: `go build ./...`. Tests: `go test ./...`.

---

## File Map

| File | Tasks | Changes |
|------|-------|---------|
| `render.go` | 1, 2, 6 | Add height constants; add `renderIssueMetaStrip`, `renderPRMetaStrip`; update dashboard filter bar in `headerView` |
| `model_issues.go` | 1, 3 | Update `headerHeight()` call → constant; strip metadata from viewport content; wire meta strip; fix viewport sizing |
| `model_prs.go` | 1, 4 | Same as issues |
| `app.go` | 1, 5, 6 | Update `headerView` call → `detailActive`; suppress global footer when detail open; remove `Available` from Counts pass-through |
| `model_dashboard.go` | 6 | Remove `availableIssues`, drop 4th goroutine, new row rendering with PR/IS badges + CI checks |
| `dashboard_test.go` | 6 | Remove `secAvailable` cases; reflect 3-section model |

---

## Task 1 — Height constants and `headerView` `detailActive` parameter

**Files:**
- Modify: `render.go`
- Modify: `model_issues.go`
- Modify: `model_prs.go`
- Modify: `app.go`

The filter bar in `headerView` is 3 lines tall (blank + content + blank). It should be hidden when viewing a detail. We add named constants so the math is self-documenting.

- [ ] **Step 1.1 — Add height constants to `render.go`**

In `render.go`, find the line:
```go
// (currently no height constants — they live in model_issues.go as headerHeight())
```

Add these three constants right before `func headerView(...)` (around line 158):

```go
const (
	// headerHeightFull is the line count of headerView when the filter bar is shown.
	// 3 (title band) + 3 (tab band) + 3 (filter band) = 9
	headerHeightFull = 9

	// headerHeightDetail is the line count when the filter bar is suppressed (detail views).
	// 3 (title band) + 3 (tab band) = 6
	headerHeightDetail = 6

	// metaStripHeight is the fixed line count of the sticky metadata strip
	// rendered above the viewport in detail views.
	// title (1) + state row (1) + labels row (1) + separator (1) = 4
	metaStripHeight = 4
)
```

- [ ] **Step 1.2 — Add `detailActive bool` parameter to `headerView`**

Change the signature from:
```go
func headerView(activeTab TabID, repo string, issueFilters github.Filters, prFilters github.PRFilters, counts DashCounts, width int) string {
```
to:
```go
func headerView(activeTab TabID, repo string, issueFilters github.Filters, prFilters github.PRFilters, counts DashCounts, width int, detailActive bool) string {
```

- [ ] **Step 1.3 — Skip filter bar when `detailActive` is true**

In `headerView`, find the block that renders the filter bar (around line 254 — starts with `sep := filterSepStyle.Render(...)`):

```go
	// ── Line 3: filter/context bar ─────────────────────────────────────────
	sep := filterSepStyle.Render("  │  ")
	fmtFilter := func(key, val string) string { ... }

	blankFilter := filterBgStyle.Render(strings.Repeat(" ", width))
	indent := filterBgStyle.Render("  ")
	// ... filterContent switch ...
	b.WriteString(blankFilter + "\n")
	b.WriteString(filterContent + "\n")
	b.WriteString(blankFilter + "\n")

	return b.String()
```

Wrap the **entire filter section** (from `sep :=` through the final `b.WriteString(blankFilter)` lines) in a guard. `sep` and `fmtFilter` must be inside the block to avoid compiler errors when `detailActive` is true:

```go
	if !detailActive {
		sep := filterSepStyle.Render("  │  ")
		fmtFilter := func(key, val string) string {
			active := val != "" && val != "any"
			v := filterValStyle
			if active {
				v = filterValOnStyle
			}
			return filterKeyStyle.Render(key+":") + " " + v.Render(val)
		}
		blankFilter := filterBgStyle.Render(strings.Repeat(" ", width))
		indent := filterBgStyle.Render("  ")

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
			countStyle := lipgloss.NewStyle().Background(filterBg).Foreground(lipgloss.Color("205")).Bold(true)
			countOrDash := func(n int) string {
				if n == 0 {
					return filterValStyle.Render("0")
				}
				return countStyle.Render(fmt.Sprintf("%d", n))
			}
			filterContent = indent +
				countOrDash(counts.ReviewRequests) + filterKeyStyle.Render(" review requests") + sep +
				countOrDash(counts.MyPRs) + filterKeyStyle.Render(" open PRs") + sep +
				countOrDash(counts.Assigned) + filterKeyStyle.Render(" assigned")
		}
		filterLineWidth := lipgloss.Width(filterContent)
		if filterLineWidth < width {
			filterContent += filterBgStyle.Render(strings.Repeat(" ", width-filterLineWidth))
		}
		b.WriteString(blankFilter + "\n")
		b.WriteString(filterContent + "\n")
		b.WriteString(blankFilter + "\n")
	}

	return b.String()
```

Note: the Dashboard case also removes the old `counts.Available` reference (will be done here to avoid a later compile break).

- [ ] **Step 1.4 — Update `headerHeight()` in `model_issues.go` to use the constant**

In `model_issues.go`, find:
```go
// headerHeight returns the number of lines used by headerView()
// 3 title bar + 3 tab bar + 3 filter bar = 9 lines
func headerHeight() int { return 9 }
```

Replace with:
```go
// headerHeight returns the number of lines used by headerView() with filter bar shown.
func headerHeight() int { return headerHeightFull }
```

- [ ] **Step 1.5 — Update the `headerView` call in `app.go`**

In `app.go`'s `View()` method, find:
```go
header := headerView(m.activeTab, m.repo, m.issues.filters, m.prs.filters, m.dashboard.Counts(), innerW)
```

Replace with:
```go
inDetail := (m.activeTab == TabIssues && m.issues.showDetail) ||
	(m.activeTab == TabPRs && m.prs.showDetail)
header := headerView(m.activeTab, m.repo, m.issues.filters, m.prs.filters, m.dashboard.Counts(), innerW, inDetail)
```

- [ ] **Step 1.6 — Build**

```bash
cd /Users/joshua/Development/hubcap/hubcap-app && go build ./... 2>&1
```

Expected: clean. The `indent` and `sep` variables were already declared above — if you see "declared but not used" or "redeclared", move the `var filterContent string` and the `sep` / `indent` assignments inside the `if !detailActive` block so they aren't in scope when not needed.

- [ ] **Step 1.7 — Commit**

```bash
cd /Users/joshua/Development/hubcap/hubcap-app && git add render.go model_issues.go app.go && git commit -m "refactor(header): add detailActive param, height constants, suppress filter bar in detail views"
```

---

## Task 2 — Add `renderIssueMetaStrip` and `renderPRMetaStrip` to `render.go`

**Files:**
- Modify: `render.go`

These are pure rendering functions. No existing code is changed.

- [ ] **Step 2.1 — Add `renderIssueMetaStrip`**

Add after the `headerView` function (around line 310 after `return b.String()`):

```go
// renderIssueMetaStrip renders the fixed 4-line metadata strip shown above the
// viewport in issue detail view. Always produces exactly metaStripHeight lines.
func renderIssueMetaStrip(issue github.Issue, width int) string {
	if width == 0 {
		width = 80
	}
	bg := lipgloss.Color("234")
	stripBg := lipgloss.NewStyle().Background(bg)
	titleSt := lipgloss.NewStyle().Background(bg).Foreground(lipgloss.Color("208")).Bold(true)
	mutedSt := lipgloss.NewStyle().Background(bg).Foreground(lipgloss.Color("244"))
	numSt := lipgloss.NewStyle().Background(bg).Foreground(lipgloss.Color("69"))
	authorSt := lipgloss.NewStyle().Background(bg).Foreground(lipgloss.Color("252"))
	sepSt := lipgloss.NewStyle().Background(bg).Foreground(lipgloss.Color("238"))
	sep := sepSt.Render("  ·  ")

	pad := func(s string) string {
		w := lipgloss.Width(s)
		if w < width {
			return s + stripBg.Render(strings.Repeat(" ", width-w))
		}
		return s
	}

	// Line 1: title
	titleLine := pad(titleSt.Render("  "+truncate(issue.Title, width-4)))

	// Line 2: state · number · author · assignee
	stateStr := stateIndicator(issue.State, false) + "  " + func() string {
		if strings.EqualFold(issue.State, "closed") {
			return lipgloss.NewStyle().Background(bg).Foreground(lipgloss.Color("196")).Bold(true).Render("CLOSED")
		}
		return lipgloss.NewStyle().Background(bg).Foreground(lipgloss.Color("83")).Bold(true).Render("OPEN")
	}()
	stateLine := "  " + stateStr + sep +
		numSt.Render(fmt.Sprintf("#%d", issue.Number)) + sep +
		mutedSt.Render("opened by ") + authorSt.Render(issue.Author.Login)
	if len(issue.Assignees) > 0 {
		stateLine += sep + mutedSt.Render("assigned to ") + authorSt.Render(joinUsers(issue.Assignees))
	}
	stateLine = pad(stripBg.Render(stateLine))

	// Line 3: labels (or blank padding line to keep height constant)
	var labelsLine string
	if len(issue.Labels) > 0 {
		labelsLine = pad(stripBg.Render("  " + coloredLabelsCompact(issue.Labels, width-4)))
	} else {
		labelsLine = pad(stripBg.Render(""))
	}

	// Line 4: separator
	sepLine := lipgloss.NewStyle().
		Foreground(lipgloss.Color("237")).
		Render(strings.Repeat("─", width))

	return titleLine + "\n" + stateLine + "\n" + labelsLine + "\n" + sepLine + "\n"
}
```

- [ ] **Step 2.2 — Add `renderPRMetaStrip`**

Add immediately after `renderIssueMetaStrip`:

```go
// renderPRMetaStrip renders the fixed 4-line metadata strip shown above the
// viewport in PR detail view. Always produces exactly metaStripHeight lines.
func renderPRMetaStrip(pr github.PullRequest, width int) string {
	if width == 0 {
		width = 80
	}
	bg := lipgloss.Color("234")
	stripBg := lipgloss.NewStyle().Background(bg)
	titleSt := lipgloss.NewStyle().Background(bg).Foreground(lipgloss.Color("208")).Bold(true)
	mutedSt := lipgloss.NewStyle().Background(bg).Foreground(lipgloss.Color("244"))
	numSt := lipgloss.NewStyle().Background(bg).Foreground(lipgloss.Color("69"))
	authorSt := lipgloss.NewStyle().Background(bg).Foreground(lipgloss.Color("252"))
	sepSt := lipgloss.NewStyle().Background(bg).Foreground(lipgloss.Color("238"))
	branchSt := lipgloss.NewStyle().Background(bg).Foreground(lipgloss.Color("252"))
	sep := sepSt.Render("  ·  ")

	pad := func(s string) string {
		w := lipgloss.Width(s)
		if w < width {
			return s + stripBg.Render(strings.Repeat(" ", width-w))
		}
		return s
	}

	// Line 1: title
	titleLine := pad(titleSt.Render("  " + truncate(pr.Title, width-4)))

	// Line 2: state · number · author · branch
	stateStr := stateIndicator(pr.State, pr.IsDraft) + "  " + func() string {
		switch {
		case pr.IsDraft:
			return lipgloss.NewStyle().Background(bg).Foreground(lipgloss.Color("214")).Bold(true).Render("DRAFT")
		case strings.EqualFold(pr.State, "merged"):
			return lipgloss.NewStyle().Background(bg).Foreground(lipgloss.Color("141")).Bold(true).Render("MERGED")
		case strings.EqualFold(pr.State, "closed"):
			return lipgloss.NewStyle().Background(bg).Foreground(lipgloss.Color("196")).Bold(true).Render("CLOSED")
		default:
			return lipgloss.NewStyle().Background(bg).Foreground(lipgloss.Color("83")).Bold(true).Render("OPEN")
		}
	}()
	stateLine := "  " + stateStr + sep +
		numSt.Render(fmt.Sprintf("#%d", pr.Number)) + sep +
		mutedSt.Render("by ") + authorSt.Render(pr.Author.Login)
	if pr.HeadRefName != "" {
		stateLine += sep + mutedSt.Render("⎇ ") + branchSt.Render(truncate(pr.HeadRefName, 35))
	}
	stateLine = pad(stripBg.Render(stateLine))

	// Line 3: review decision · CI checks · labels (or blank if nothing)
	var reviewStr string
	switch pr.ReviewDecision {
	case "APPROVED":
		reviewStr = lipgloss.NewStyle().Background(bg).Foreground(lipgloss.Color("83")).Render("✓ APPROVED")
	case "CHANGES_REQUESTED":
		reviewStr = lipgloss.NewStyle().Background(bg).Foreground(lipgloss.Color("196")).Render("✗ CHANGES REQUESTED")
	case "REVIEW_REQUIRED":
		reviewStr = lipgloss.NewStyle().Background(bg).Foreground(lipgloss.Color("214")).Render("⟳ REVIEW REQUIRED")
	}

	checksStr := func() string {
		raw := summarizeChecks(pr.StatusRollup)
		switch {
		case strings.Contains(raw, "✓"):
			return lipgloss.NewStyle().Background(bg).Foreground(lipgloss.Color("83")).Render("✓ checks passing")
		case strings.Contains(raw, "✗"):
			return lipgloss.NewStyle().Background(bg).Foreground(lipgloss.Color("196")).Render("✗ checks failing")
		case strings.Contains(raw, "…"):
			return lipgloss.NewStyle().Background(bg).Foreground(lipgloss.Color("214")).Render("… checks pending")
		default:
			return ""
		}
	}()

	var row3Parts []string
	if reviewStr != "" {
		row3Parts = append(row3Parts, reviewStr)
	}
	if checksStr != "" {
		row3Parts = append(row3Parts, checksStr)
	}
	if len(pr.Labels) > 0 {
		row3Parts = append(row3Parts, coloredLabelsCompact(pr.Labels, 40))
	}
	var infoLine string
	if len(row3Parts) > 0 {
		infoLine = pad(stripBg.Render("  " + strings.Join(row3Parts, sep)))
	} else {
		infoLine = pad(stripBg.Render(""))
	}

	// Line 4: separator
	sepLine := lipgloss.NewStyle().
		Foreground(lipgloss.Color("237")).
		Render(strings.Repeat("─", width))

	return titleLine + "\n" + stateLine + "\n" + infoLine + "\n" + sepLine + "\n"
}
```

- [ ] **Step 2.3 — Build**

```bash
cd /Users/joshua/Development/hubcap/hubcap-app && go build ./... 2>&1
```

Expected: clean.

- [ ] **Step 2.4 — Commit**

```bash
cd /Users/joshua/Development/hubcap/hubcap-app && git add render.go && git commit -m "feat(render): add renderIssueMetaStrip and renderPRMetaStrip"
```

---

## Task 3 — Wire meta strip into `model_issues.go`

**Files:**
- Modify: `model_issues.go`

Three changes: (a) strip metadata from `renderIssueDetailContent` — body only, (b) fix viewport height to account for strip, (c) prepend strip in `View()`.

- [ ] **Step 3.1 — Strip metadata from `renderIssueDetailContent`**

In `model_issues.go`, replace `renderIssueDetailContent` entirely:

```go
// renderIssueDetailContent builds scrollable body-only content for the viewport.
// Title and metadata are handled by renderIssueMetaStrip above the viewport.
func renderIssueDetailContent(issue github.Issue, _ int) string {
	var b strings.Builder
	if issue.Body != "" {
		b.WriteString(issue.Body + "\n")
	} else {
		b.WriteString(styleGray.Render("No description.") + "\n")
	}
	return b.String()
}
```

- [ ] **Step 3.2 — Fix viewport height in `issueFetchedMsg` handler**

In `model_issues.go`, find the `issueFetchedMsg` case:
```go
	case issueFetchedMsg:
		m.loadingDetail = false
		if msg.err != nil {
			m.err = msg.err
			return m, nil
		}
		m.detailIssue = msg.issue
		content := renderIssueDetailContent(msg.issue, m.width)
		m.detail = viewport.New(m.width-4, m.height-headerHeight()-4)
		m.detail.SetContent(content)
		m.showDetail = true
```

Change the viewport construction line:
```go
		m.detail = viewport.New(m.width-4, m.height-headerHeightDetail-metaStripHeight-4)
```

- [ ] **Step 3.3 — Fix viewport resize in `tea.WindowSizeMsg` handler**

In `model_issues.go`, find the `tea.WindowSizeMsg` case:
```go
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.list.SetSize(msg.Width-4, msg.Height-headerHeight()-2)
		if m.showDetail {
			m.detail.Width = msg.Width - 4
			m.detail.Height = msg.Height - headerHeight() - 4
		}
```

Change the detail resize lines:
```go
		if m.showDetail {
			m.detail.Width = msg.Width - 4
			m.detail.Height = msg.Height - headerHeightDetail - metaStripHeight - 4
		}
```

- [ ] **Step 3.4 — Prepend the meta strip in `View()` when showing detail**

In `model_issues.go`, find the detail view render block:
```go
	// Detail view
	if m.showDetail {
		b.WriteString(renderIssueDetailView(m.detailIssue, m.detail, m.actionMsg, m.actionErr))
		return b.String()
	}
```

Replace with:
```go
	// Detail view: meta strip (fixed) above scrollable viewport
	if m.showDetail {
		b.WriteString(renderIssueMetaStrip(m.detailIssue, m.width-4))
		b.WriteString(renderIssueDetailView(m.detailIssue, m.detail, m.actionMsg, m.actionErr))
		return b.String()
	}
```

- [ ] **Step 3.5 — Build**

```bash
cd /Users/joshua/Development/hubcap/hubcap-app && go build ./... 2>&1
```

Expected: clean.

- [ ] **Step 3.6 — Commit**

```bash
cd /Users/joshua/Development/hubcap/hubcap-app && git add model_issues.go && git commit -m "feat(issues): sticky metadata strip above viewport, body-only scroll content"
```

---

## Task 4 — Wire meta strip into `model_prs.go`

**Files:**
- Modify: `model_prs.go`

Mirror of Task 3 for PRs.

- [ ] **Step 4.1 — Strip metadata from `renderPRDetailContent`**

In `model_prs.go`, replace `renderPRDetailContent` entirely:

```go
// renderPRDetailContent builds scrollable body-only content for the viewport.
// Title and metadata are handled by renderPRMetaStrip above the viewport.
func renderPRDetailContent(pr github.PullRequest, _ int) string {
	var b strings.Builder
	if pr.Body != "" {
		b.WriteString(pr.Body + "\n")
	} else {
		b.WriteString(styleGray.Render("No description.") + "\n")
	}
	return b.String()
}
```

- [ ] **Step 4.2 — Fix viewport height in `prFetchedMsg` handler**

In `model_prs.go`, find the `prFetchedMsg` case:
```go
	case prFetchedMsg:
		m.loadingDetail = false
		if msg.err != nil {
			m.err = msg.err
			return m, nil
		}
		m.detailPR = msg.pr
		content := renderPRDetailContent(msg.pr, m.width)
		m.detail = viewport.New(m.width-4, m.height-headerHeight()-4)
		m.detail.SetContent(content)
		m.showDetail = true
```

Change the viewport construction line:
```go
		m.detail = viewport.New(m.width-4, m.height-headerHeightDetail-metaStripHeight-4)
```

- [ ] **Step 4.3 — Fix viewport resize in `tea.WindowSizeMsg` handler**

In `model_prs.go`, find:
```go
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.list.SetSize(msg.Width-4, msg.Height-headerHeight()-2)
		if m.showDetail {
			m.detail.Width = msg.Width - 4
			m.detail.Height = msg.Height - headerHeight() - 4
		}
```

Change the detail resize lines:
```go
		if m.showDetail {
			m.detail.Width = msg.Width - 4
			m.detail.Height = msg.Height - headerHeightDetail - metaStripHeight - 4
		}
```

- [ ] **Step 4.4 — Prepend the meta strip in `View()`**

In `model_prs.go`, find:
```go
	if m.showDetail {
		b.WriteString(renderPRDetailView(m.detailPR, m.detail, m.actionMsg, m.actionErr))
		return b.String()
	}
```

Replace with:
```go
	if m.showDetail {
		b.WriteString(renderPRMetaStrip(m.detailPR, m.width-4))
		b.WriteString(renderPRDetailView(m.detailPR, m.detail, m.actionMsg, m.actionErr))
		return b.String()
	}
```

- [ ] **Step 4.5 — Build**

```bash
cd /Users/joshua/Development/hubcap/hubcap-app && go build ./... 2>&1
```

Expected: clean.

- [ ] **Step 4.6 — Commit**

```bash
cd /Users/joshua/Development/hubcap/hubcap-app && git add model_prs.go && git commit -m "feat(prs): sticky metadata strip above viewport, body-only scroll content"
```

---

## Task 5 — Suppress global footer when detail is active (`app.go`)

**Files:**
- Modify: `app.go`

When `inDetail` is true, the detail action bar rendered inside the model's `View()` output is the only footer. Skip `footerView` entirely.

- [ ] **Step 5.1 — Suppress footer and adjust fill calculation**

In `app.go`, find the `View()` method footer section:

```go
	// Build the footer hint bar
	footer := footerView(m.activeTab, innerW)

	// Count used lines: header + body lines + footer
	headerLines := strings.Count(header, "\n")
	bodyLines := strings.Count(body, "\n")
	footerLines := strings.Count(footer, "\n") + 1
	usedLines := headerLines + bodyLines + footerLines

	// Fill remaining space so footer sticks to the bottom
	remaining := innerH - usedLines
	if remaining < 0 {
		remaining = 0
	}
	fill := strings.Repeat("\n", remaining)

	inner := header + body + fill + footer
```

Replace with:
```go
	// Build the footer hint bar — suppressed when a detail view is active
	// (the detail view renders its own context-specific action bar)
	var footer string
	if !inDetail {
		footer = footerView(m.activeTab, innerW)
	}

	// Count used lines: header + body + footer
	headerLines := strings.Count(header, "\n")
	bodyLines := strings.Count(body, "\n")
	footerLines := 0
	if footer != "" {
		footerLines = strings.Count(footer, "\n") + 1
	}
	usedLines := headerLines + bodyLines + footerLines

	// Fill remaining space so footer sticks to the bottom
	remaining := innerH - usedLines
	if remaining < 0 {
		remaining = 0
	}
	fill := strings.Repeat("\n", remaining)

	inner := header + body + fill + footer
```

- [ ] **Step 5.2 — Build and test**

```bash
cd /Users/joshua/Development/hubcap/hubcap-app && go build ./... 2>&1 && go test ./... 2>&1
```

Expected: clean build, all tests pass.

- [ ] **Step 5.3 — Commit**

```bash
cd /Users/joshua/Development/hubcap/hubcap-app && git add app.go && git commit -m "fix(app): suppress global footer when issue/PR detail is open"
```

---

## Task 6 — Dashboard: trim to 3 sections, new row format

**Files:**
- Modify: `model_dashboard.go`
- Modify: `app.go`
- Modify: `dashboard_test.go`

Remove `availableIssues`, update the fetch, update row rendering with `PR`/`IS` badges + CI checks + author.

- [ ] **Step 6.1 — Remove `availableIssues` from `dashboardData` in `app.go`**

In `app.go`, find:
```go
// dashboardData holds all sections of the dashboard
type dashboardData struct {
	reviewRequests []github.PullRequest
	myPRs          []github.PullRequest
	assignedIssues []github.Issue
	availableIssues []github.Issue
}
```

Replace with:
```go
// dashboardData holds the three personal sections of the My Work dashboard.
type dashboardData struct {
	reviewRequests []github.PullRequest
	myPRs          []github.PullRequest
	assignedIssues []github.Issue
}
```

- [ ] **Step 6.2 — Remove `Available` from `DashCounts` in `model_dashboard.go`**

In `model_dashboard.go`, find:
```go
type DashCounts struct {
	ReviewRequests int
	MyPRs          int
	Assigned       int
	Available      int
}
```

Replace with:
```go
type DashCounts struct {
	ReviewRequests int
	MyPRs          int
	Assigned       int
}
```

- [ ] **Step 6.3 — Update `Counts()` in `model_dashboard.go`**

Find:
```go
func (m DashboardModel) Counts() DashCounts {
	if !m.loaded {
		return DashCounts{}
	}
	return DashCounts{
		ReviewRequests: len(m.data.reviewRequests),
		MyPRs:          len(m.data.myPRs),
		Assigned:       len(m.data.assignedIssues),
		Available:      len(m.data.availableIssues),
	}
}
```

Replace with:
```go
func (m DashboardModel) Counts() DashCounts {
	if !m.loaded {
		return DashCounts{}
	}
	return DashCounts{
		ReviewRequests: len(m.data.reviewRequests),
		MyPRs:          len(m.data.myPRs),
		Assigned:       len(m.data.assignedIssues),
	}
}
```

- [ ] **Step 6.4 — Drop 4th goroutine from `fetchCmd` in `model_dashboard.go`**

Find the `fetchCmd` function. Replace the entire function body:

```go
func (m DashboardModel) fetchCmd() tea.Cmd {
	return func() tea.Msg {
		var data dashboardData
		var mu sync.Mutex
		var wg sync.WaitGroup
		var errs [3]error

		fetch := func(i int, fn func() (interface{}, error)) {
			defer wg.Done()
			result, err := fn()
			mu.Lock()
			defer mu.Unlock()
			errs[i] = err
			if err != nil {
				return
			}
			switch i {
			case 0:
				data.reviewRequests = result.([]github.PullRequest)
			case 1:
				data.myPRs = result.([]github.PullRequest)
			case 2:
				data.assignedIssues = result.([]github.Issue)
			}
		}

		wg.Add(3)
		go fetch(0, func() (interface{}, error) {
			return github.FetchPRs(github.PRFilters{Search: "review-requested:@me", State: "open", Limit: 20})
		})
		go fetch(1, func() (interface{}, error) {
			return github.FetchPRs(github.PRFilters{Author: "@me", State: "open", Limit: 20})
		})
		go fetch(2, func() (interface{}, error) {
			return github.FetchIssues(github.Filters{Assignee: "@me", State: "open", Limit: 20})
		})
		wg.Wait()

		for _, err := range errs {
			if err != nil {
				return dashboardFetchedMsg{data: data, err: err}
			}
		}
		return dashboardFetchedMsg{data: data, err: nil}
	}
}
```

- [ ] **Step 6.5 — Update `buildDashRows` in `model_dashboard.go`**

Replace the existing `buildDashRows`:

```go
func buildDashRows(data dashboardData) []dashRow {
	var rows []dashRow
	sections := []struct {
		id    int
		count int
		issue bool
	}{
		{secReviewRequests, len(data.reviewRequests), false},
		{secMyPRs, len(data.myPRs), false},
		{secAssigned, len(data.assignedIssues), true},
	}
	for _, sec := range sections {
		if sec.count == 0 {
			continue
		}
		rows = append(rows, dashRow{isHeader: true, sectionID: sec.id, itemIdx: -1})
		for i := 0; i < sec.count; i++ {
			rows = append(rows, dashRow{isHeader: false, sectionID: sec.id, itemIdx: i, isIssue: sec.issue})
		}
	}
	return rows
}
```

- [ ] **Step 6.6 — Rewrite `DashboardModel.View()` with new row format**

Replace the entire `View()` function body (keeping the loading/error guards):

```go
func (m DashboardModel) View() string {
	if m.loading {
		return fmt.Sprintf("\n  %s Loading dashboard...\n", m.spinner.View())
	}
	if m.err != nil {
		return errorBox(fmt.Sprintf("Dashboard error: %v\n\nPress r to retry.", m.err))
	}

	var b strings.Builder

	sectionIcons := [4]string{"⟳", "⎇", "◉", "○"}
	sectionHeaderStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("208"))
	sectionCountStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("141"))
	sectionDivStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("237"))
	selectedStyle := lipgloss.NewStyle().
		Background(lipgloss.Color("23")).
		Foreground(lipgloss.Color("86"))
	prBadgeStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("141")).
		Bold(true)
	isBadgeStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("83")).
		Bold(true)
	numStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("69"))
	mutedStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("244"))

	sectionCounts := [4]int{
		len(m.data.reviewRequests),
		len(m.data.myPRs),
		len(m.data.assignedIssues),
		0,
	}

	lastSectionID := -1

	for i, row := range m.rows {
		if row.isHeader {
			// Section divider (not before the very first section)
			if lastSectionID >= 0 {
				b.WriteString(sectionDivStyle.Render(strings.Repeat("─", 60)) + "\n")
			}
			lastSectionID = row.sectionID
			icon := sectionIcons[row.sectionID]
			name := sectionNames[row.sectionID]
			count := sectionCounts[row.sectionID]
			b.WriteString(sectionHeaderStyle.Render(fmt.Sprintf("  %s %s ", icon, name)) +
				sectionCountStyle.Render(fmt.Sprintf("(%d)", count)) + "\n")
			continue
		}

		selected := i == m.cursor
		prefix := "    "
		if selected {
			prefix = "  ▶ "
		}

		var line string
		switch row.sectionID {
		case secReviewRequests:
			p := m.data.reviewRequests[row.itemIdx]
			checksCol := summarizeChecks(p.StatusRollup)
			authorCol := mutedStyle.Render("by " + truncate(p.Author.Login, 12))
			line = fmt.Sprintf("%s%s %-6d %-50s  %s  %s",
				prefix,
				prBadgeStyle.Render("PR"),
				p.Number,
				truncate(cleanLine(p.Title), 50),
				authorCol,
				checksCol,
			)
		case secMyPRs:
			p := m.data.myPRs[row.itemIdx]
			checksCol := summarizeChecks(p.StatusRollup)
			statusCol := func() string {
				if p.IsDraft {
					return mutedStyle.Render("draft")
				}
				return ""
			}()
			line = fmt.Sprintf("%s%s %-6d %-52s  %s  %s",
				prefix,
				prBadgeStyle.Render("PR"),
				p.Number,
				truncate(cleanLine(p.Title), 52),
				statusCol,
				checksCol,
			)
		case secAssigned:
			iss := m.data.assignedIssues[row.itemIdx]
			labelCol := coloredLabelsCompact(iss.Labels, 25)
			line = fmt.Sprintf("%s%s %-6d %-52s  %s",
				prefix,
				isBadgeStyle.Render("IS"),
				iss.Number,
				truncate(cleanLine(iss.Title), 52),
				labelCol,
			)
		}

		if selected {
			b.WriteString(selectedStyle.Render(line) + "\n")
		} else {
			b.WriteString(line + "\n")
		}
	}

	return b.String()
}
```

- [ ] **Step 6.7 — Build**

```bash
cd /Users/joshua/Development/hubcap/hubcap-app && go build ./... 2>&1
```

Expected: clean. If you see references to `counts.Available` in `render.go`, that was already removed in Task 1 Step 1.3. If any `secAvailable` references remain in `model_dashboard.go`'s old `View()` code that you replaced, they are now gone.

- [ ] **Step 6.8 — Update `dashboard_test.go`**

Replace the entire file:

```go
// dashboard_test.go
package main

import (
	"testing"

	"hubcap/internal/github"
)

func TestBuildDashRows_AllSectionsPopulated(t *testing.T) {
	data := dashboardData{
		reviewRequests: []github.PullRequest{{Number: 1}},
		myPRs:          []github.PullRequest{{Number: 2}},
		assignedIssues: []github.Issue{{Number: 3}},
	}
	rows := buildDashRows(data)

	// 3 sections × (1 header + 1 item) = 6 rows
	if len(rows) != 6 {
		t.Fatalf("expected 6 rows, got %d", len(rows))
	}
	if !rows[0].isHeader {
		t.Error("expected first row to be a section header")
	}
	if rows[1].isHeader {
		t.Error("expected second row to be an item")
	}
}

func TestBuildDashRows_EmptySectionsHidden(t *testing.T) {
	data := dashboardData{
		reviewRequests: []github.PullRequest{{Number: 1}},
		myPRs:          []github.PullRequest{}, // empty — hidden
		assignedIssues: []github.Issue{{Number: 3}},
	}
	rows := buildDashRows(data)

	// section 0: header + 1 item = 2
	// section 1: hidden (empty)
	// section 2: header + 1 item = 2
	// total = 4
	if len(rows) != 4 {
		t.Fatalf("expected 4 rows, got %d", len(rows))
	}
	if rows[0].sectionID != secReviewRequests {
		t.Errorf("expected first header to be secReviewRequests, got %d", rows[0].sectionID)
	}
	if rows[2].sectionID != secAssigned {
		t.Errorf("expected third row header to be secAssigned, got %d", rows[2].sectionID)
	}
}

func TestBuildDashRows_AllEmpty(t *testing.T) {
	data := dashboardData{}
	rows := buildDashRows(data)
	if len(rows) != 0 {
		t.Errorf("expected 0 rows for all-empty data, got %d", len(rows))
	}
}

func TestBuildDashRows_SectionOrder(t *testing.T) {
	data := dashboardData{
		reviewRequests: []github.PullRequest{{Number: 10}, {Number: 11}},
		myPRs:          []github.PullRequest{{Number: 20}},
		assignedIssues: []github.Issue{{Number: 30}},
	}
	rows := buildDashRows(data)

	// section 0: 1 header + 2 items = 3
	// section 1: 1 header + 1 item  = 2
	// section 2: 1 header + 1 item  = 2
	// total = 7
	if len(rows) != 7 {
		t.Fatalf("expected 7 rows, got %d", len(rows))
	}
	if rows[0].sectionID != secReviewRequests || !rows[0].isHeader {
		t.Errorf("row 0 should be secReviewRequests header")
	}
	if rows[1].isHeader || rows[1].sectionID != secReviewRequests {
		t.Errorf("row 1 should be secReviewRequests item")
	}
	if rows[3].sectionID != secMyPRs || !rows[3].isHeader {
		t.Errorf("row 3 should be secMyPRs header")
	}
	if rows[5].sectionID != secAssigned || !rows[5].isHeader {
		t.Errorf("row 5 should be secAssigned header")
	}
}
```

- [ ] **Step 6.9 — Run tests**

```bash
cd /Users/joshua/Development/hubcap/hubcap-app && go test ./... 2>&1
```

Expected: `ok hubcap` and `ok hubcap/internal/github`.

- [ ] **Step 6.10 — Commit**

```bash
cd /Users/joshua/Development/hubcap/hubcap-app && git add model_dashboard.go app.go dashboard_test.go && git commit -m "feat(dashboard): trim to 3 personal sections, new PR/IS row format with CI checks"
```

---

## Final verification

- [ ] **Full build and test**

```bash
cd /Users/joshua/Development/hubcap/hubcap-app && go build ./... && go test ./... -count=1
```

Expected: clean build, all tests pass.

- [ ] **Manual smoke test checklist**

Build and run:
```bash
cd /Users/joshua/Development/hubcap/hubcap-app && go build -o hubcap . && ./hubcap
```

1. Dashboard loads with ⟳ / ⎇ / ◉ section headers — no "Available to Grab" section
2. PR rows show `PR` badge (purple) + CI check symbol at right; author shown for review requests
3. Issue rows show `IS` badge (green) + label at right
4. Press `↑↓` to navigate, `enter` on a PR row → switches to PRs tab, opens detail immediately
5. In PR detail: orange title visible at top; state/branch row below; review + checks row; body scrolls below separator
6. In issue detail: orange title visible at top; state/author row; labels row; body scrolls below separator
7. Press `b` or `esc` → returns to list
8. No double footer in any detail view — only the context action bar at the bottom
9. Filter bar (`state: open | assignee: any…`) is NOT shown when a detail is open
10. Switching back to Issues/PRs list → filter bar reappears
11. Dashboard: `r` refreshes all three sections
