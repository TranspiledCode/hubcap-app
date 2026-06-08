# Theme-Driven Colors Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Eliminate every hardcoded `lipgloss.Color(...)` call outside of `theme.go` by adding semantic label color fields to `Palette` and threading palette through all rendering functions.

**Architecture:** All colour decisions live in the `Palette` struct (theme.go). Rendering functions that currently use package-level `styleXxx` vars or inline ANSI codes are updated to accept a `Palette` parameter instead. Dead legacy-CLI code (renderHeader, printIssueDetail, terminal menus/spinners) is deleted rather than ported, since it has no callers.

**Tech Stack:** Go, charmbracelet/lipgloss, charmbracelet/bubbletea

---

## File map

| File | Change |
|---|---|
| `theme.go` | Add 14 new Palette fields (7 label colours + 7 pill-text colours) to struct + all 4 palette definitions |
| `render.go` | Delete package-level style-var block + dead functions; update `stateIndicator`, `labelPillColors`, `labelPill`, `labelStyle`, `summarizeChecks`, `errorBox`; fix hardcoded pill colours in `renderPRMetaStrip` |
| `model_issues.go` | Simplify `Description()` to plain text; add `pal` param to `issueRowLabels`; replace 3× `styleGray` usages; update `errorBox` call |
| `model_prs.go` | Simplify `Description()` to plain text; pass `pal` to `issueRowLabels`; replace 3× `styleGray` usages; update `errorBox` call |
| `model_dashboard.go` | Pass `pal` to `summarizeChecks` (2 call sites); update `errorBox` call |
| `terminal.go` | Delete dead code: `withSpinner`, `startSpinner`, `spinnerModel`, `spinnerStyle`, `init()`, `menu()`, `numberedMenu()` |

---

## Task 1: Extend Palette with semantic label colours

**Files:**
- Modify: `theme.go`

### Background

`labelStyle()` (foreground text colour) and `labelPillColors()` (coloured chip bg + fg) both use hardcoded ANSI codes. Seven semantic categories exist: Danger, Warning, Success, Feature, Docs, Subtle, Default. Each needs two palette fields:
- `LabelXxx` — the category colour, used as **foreground** text in list rows and as **background** in pill chips
- `LabelXxxFg` — the text colour **on top of** the pill chip background (white or black depending on bg luminance)

This gives 14 new fields.

- [ ] **Add the 14 fields to the Palette struct**

Add this block after the `CheckPending` field (around line 55 of theme.go):

```go
// Semantic label category colours — used for text in list rows (LabelXxx)
// and as pill chip backgrounds in detail strips, with LabelXxxFg as pill text.
LabelDanger  lipgloss.Color // bug, critical, high-priority
LabelWarningC lipgloss.Color // medium-priority, question  (named C to avoid collision with pal.Danger)
LabelSuccess lipgloss.Color // low-priority, passing
LabelFeature lipgloss.Color // enhancement, feature
LabelDocs    lipgloss.Color // documentation
LabelSubtle  lipgloss.Color // effort, size (dim)
LabelDefault lipgloss.Color // uncategorised / fallback

LabelDangerFg  lipgloss.Color // text on top of LabelDanger pill
LabelWarningFg lipgloss.Color // text on top of LabelWarningC pill
LabelSuccessFg lipgloss.Color // text on top of LabelSuccess pill
LabelFeatureFg lipgloss.Color // text on top of LabelFeature pill
LabelDocsFg    lipgloss.Color // text on top of LabelDocs pill
LabelSubtleFg  lipgloss.Color // text on top of LabelSubtle pill
LabelDefaultFg lipgloss.Color // text on top of LabelDefault pill
```

> Note: `LabelWarningC` avoids a name collision — consider just `LabelWarn` if preferred. Pick one name and use it consistently throughout the plan.

**Actually use `LabelWarn` throughout** (shorter, no collision risk).

The final field names are:
`LabelDanger`, `LabelWarn`, `LabelSuccess`, `LabelFeature`, `LabelDocs`, `LabelSubtle`, `LabelDefault`,
`LabelDangerFg`, `LabelWarnFg`, `LabelSuccessFg`, `LabelFeatureFg`, `LabelDocsFg`, `LabelSubtleFg`, `LabelDefaultFg`

```go
// Semantic label category colours
LabelDanger  lipgloss.Color
LabelWarn    lipgloss.Color
LabelSuccess lipgloss.Color
LabelFeature lipgloss.Color
LabelDocs    lipgloss.Color
LabelSubtle  lipgloss.Color
LabelDefault lipgloss.Color

LabelDangerFg  lipgloss.Color
LabelWarnFg    lipgloss.Color
LabelSuccessFg lipgloss.Color
LabelFeatureFg lipgloss.Color
LabelDocsFg    lipgloss.Color
LabelSubtleFg  lipgloss.Color
LabelDefaultFg lipgloss.Color
```

- [ ] **Add values to `paletteDefault`** (keep existing ANSI shades exactly as-is)

```go
LabelDanger:  lipgloss.Color("1"),
LabelWarn:    lipgloss.Color("3"),
LabelSuccess: lipgloss.Color("2"),
LabelFeature: lipgloss.Color("6"),
LabelDocs:    lipgloss.Color("5"),
LabelSubtle:  lipgloss.Color("8"),
LabelDefault: lipgloss.Color("208"),

LabelDangerFg:  lipgloss.Color("15"),
LabelWarnFg:    lipgloss.Color("0"),
LabelSuccessFg: lipgloss.Color("0"),
LabelFeatureFg: lipgloss.Color("0"),
LabelDocsFg:    lipgloss.Color("15"),
LabelSubtleFg:  lipgloss.Color("15"),
LabelDefaultFg: lipgloss.Color("0"),
```

- [ ] **Add values to `paletteCobalt2`**

```go
LabelDanger:  lipgloss.Color("#FF628C"),
LabelWarn:    lipgloss.Color("#FFC600"),
LabelSuccess: lipgloss.Color("#3AD900"),
LabelFeature: lipgloss.Color("#2AFFDF"),
LabelDocs:    lipgloss.Color("#C792EA"),
LabelSubtle:  lipgloss.Color("#546E7A"),
LabelDefault: lipgloss.Color("#FF9D00"),

LabelDangerFg:  lipgloss.Color("0"),
LabelWarnFg:    lipgloss.Color("0"),
LabelSuccessFg: lipgloss.Color("0"),
LabelFeatureFg: lipgloss.Color("0"),
LabelDocsFg:    lipgloss.Color("0"),
LabelSubtleFg:  lipgloss.Color("15"),
LabelDefaultFg: lipgloss.Color("0"),
```

- [ ] **Add values to `paletteTranspiled`**

```go
LabelDanger:  lipgloss.Color("#FF628C"),
LabelWarn:    lipgloss.Color("#FFC600"),
LabelSuccess: lipgloss.Color("#3AD900"),
LabelFeature: lipgloss.Color("#2AFFDF"),
LabelDocs:    lipgloss.Color("#C792EA"),
LabelSubtle:  lipgloss.Color("#546E7A"),
LabelDefault: lipgloss.Color("#FF9D00"),

LabelDangerFg:  lipgloss.Color("0"),
LabelWarnFg:    lipgloss.Color("0"),
LabelSuccessFg: lipgloss.Color("0"),
LabelFeatureFg: lipgloss.Color("0"),
LabelDocsFg:    lipgloss.Color("0"),
LabelSubtleFg:  lipgloss.Color("15"),
LabelDefaultFg: lipgloss.Color("0"),
```

- [ ] **Add values to `paletteImageScoop`**

```go
LabelDanger:  lipgloss.Color("#ec4899"),
LabelWarn:    lipgloss.Color("#f97316"),
LabelSuccess: lipgloss.Color("#a3e635"),
LabelFeature: lipgloss.Color("#06b6d4"),
LabelDocs:    lipgloss.Color("#a855f7"),
LabelSubtle:  lipgloss.Color("#6B7280"),
LabelDefault: lipgloss.Color("#667eea"),

LabelDangerFg:  lipgloss.Color("15"),
LabelWarnFg:    lipgloss.Color("0"),
LabelSuccessFg: lipgloss.Color("0"),
LabelFeatureFg: lipgloss.Color("0"),
LabelDocsFg:    lipgloss.Color("15"),
LabelSubtleFg:  lipgloss.Color("15"),
LabelDefaultFg: lipgloss.Color("15"),
```

- [ ] **Build to confirm struct compiles**

```bash
cd /Users/joshua/Development/hubcap/hubcap-app && go build ./...
```

Expected: clean build (Palette fields are additive — nothing else references them yet).

- [ ] **Commit**

```bash
git add theme.go
git commit -m "feat(theme): add 14 semantic label colour fields to Palette"
```

---

## Task 2: Delete dead code and package-level style vars from render.go

**Files:**
- Modify: `render.go`

### Background

The following are dead code (zero callers in any `.go` file):
- Package-level style var block (`styleReset`, `styleGreen`, `styleYellow`, `styleRed`, `stylePurple`, `styleGray`, `styleCyan`, `styleOrange`, `styleTitle`, all box styles, all status bar styles, all legacy compat vars)
- Functions: `renderHeader`, `printIssueDetail`, `printIssuesTable`, `printPRDetail`, `hintBar`, `hintSep`, `colorState`, `colorVal`, `coloredLabels`, `warningBox`, `successBox`, `infoBox`, `renderStatusBar`, `dominantLabelStyle`, `coloredLabelsCompact`, `truncateLines`

`labelPriority` has no colours and is still alive (called from `renderIssueMetaStrip`), so keep it.

- [ ] **Delete the entire package-level `var (...)` style block** (lines ~19–75 of render.go)

The block starts with `// Lipgloss styles` and ends just before `func renderHeader`. Remove it completely. This removes `styleReset`, `styleGreen`, `styleYellow`, `styleRed`, `stylePurple`, `styleGray`, `styleCyan`, `styleOrange`, `styleTitle`, `errorBoxStyle`, `warningBoxStyle`, `successBoxStyle`, `infoBoxStyle`, `statusBarStyle`, `statusBarAccent`, and the legacy compat block (`colorReset` etc.).

- [ ] **Delete dead functions** — remove each of the following function bodies from render.go:

  1. `renderHeader(state *AppState, rawMode bool)` — starts ~line 77
  2. `printIssueDetail(issue github.Issue, ...)` — starts ~line 733
  3. `printIssuesTable(issues []github.Issue)` — starts ~line 754
  4. `printPRDetail(pr github.PullRequest, ...)` — starts ~line 769
  5. `hintBar(pairs ...string) string` — starts ~line 869
  6. `hintSep(rawMode bool) string` — starts ~line 890
  7. `colorState(s string) string` — starts ~line 900
  8. `colorVal(s string) string` — starts ~line 911
  9. `coloredLabels(labels []github.Label) string` — starts ~line 1038
  10. `warningBox(msg string) string`
  11. `successBox(msg string) string`
  12. `infoBox(msg string) string`
  13. `renderStatusBar(state *AppState, stats string) string` — starts ~line 1081
  14. `dominantLabelStyle(labels []github.Label) lipgloss.Style` — starts ~line 1024
  15. `coloredLabelsCompact(labels []github.Label, maxWidth int) string` — starts ~line 1051
  16. `truncateLines(text string, ...) string` — starts ~line 1138

- [ ] **Remove the `"fmt"` import if it becomes unused** (check after deletions — `fmt` may still be needed by `renderIssueMetaStrip`)

- [ ] **Build to surface any remaining reference errors**

```bash
cd /Users/joshua/Development/hubcap/hubcap-app && go build ./... 2>&1
```

Expected: errors mentioning `styleGray`, `styleCyan`, etc. in model files + `stateIndicator` signature mismatches. These are fixed in subsequent tasks.

- [ ] **Commit**

```bash
git add render.go
git commit -m "refactor(render): delete dead legacy-CLI code and hardcoded style vars"
```

---

## Task 3: Update render.go live functions to use Palette

**Files:**
- Modify: `render.go`

### Background

These functions are called from live TUI code and use hardcoded colours:
- `stateIndicator` — used by `renderIssueMetaStrip` and `renderPRMetaStrip`
- `labelStyle` — used by `issueRowLabels` in `model_issues.go`
- `labelPillColors` — used by `labelPill`
- `labelPill` — used by `renderIssueMetaStrip` and `renderPRMetaStrip`
- `summarizeChecks` — used by `model_dashboard.go`
- `errorBox` — used by `model_issues.go`, `model_prs.go`, `model_dashboard.go`

Also, `renderPRMetaStrip` has inline hardcoded ANSI codes in the review/CI status pill calls (`"1"`, `"2"`, `"3"`, `"15"`, `"0"`).

- [ ] **Update `stateIndicator` signature and body**

Old signature: `func stateIndicator(state string, isDraft bool) string`

New signature: `func stateIndicator(state string, isDraft bool, pal Palette) string`

New body:
```go
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
```

Update the two call sites in the same file:
- `renderIssueMetaStrip` line ~460: `stateIndicator(issue.State, false)` → `stateIndicator(issue.State, false, pal)`
- `renderPRMetaStrip` line ~641: `stateIndicator(pr.State, pr.IsDraft)` → `stateIndicator(pr.State, pr.IsDraft, pal)`

- [ ] **Update `labelStyle` signature and body**

Old: `func labelStyle(name string) lipgloss.Style`

New:
```go
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
```

- [ ] **Update `labelPillColors` signature and body**

Old: `func labelPillColors(name string) (bg, fg lipgloss.Color)`

New:
```go
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
```

- [ ] **Update `labelPill` signature**

Old: `func labelPill(stripBg lipgloss.Color, name string) string`

New:
```go
func labelPill(stripBg lipgloss.Color, name string, pal Palette) string {
	bg, fg := labelPillColors(name, pal)
	chip := lipgloss.NewStyle().Background(bg).Foreground(fg).Padding(0, 1).Render(name)
	gutter := lipgloss.NewStyle().Background(stripBg).Render(" ")
	return gutter + chip + gutter
}
```

Update every call site of `labelPill` in render.go — they all have `pal` in scope:
- `renderIssueMetaStrip` line ~500: `labelPill(bg, l.Name)` → `labelPill(bg, l.Name, pal)`
- `renderIssueMetaStrip` line ~529: `labelPill(bg, issue.Labels[i].Name)` → `labelPill(bg, issue.Labels[i].Name, pal)`
- `renderPRMetaStrip` line ~704: `labelPill(bg, l.Name)` → `labelPill(bg, l.Name, pal)`

- [ ] **Fix hardcoded ANSI codes in `renderPRMetaStrip` review/CI status pills**

In `renderPRMetaStrip`, the review decision and CI status pills use raw ANSI codes. Replace:

```go
// OLD:
switch pr.ReviewDecision {
case "APPROVED":
    rightChips = append(rightChips, prStatusPill(bg, "2", "0", "✓ approved"))
case "CHANGES_REQUESTED":
    rightChips = append(rightChips, prStatusPill(bg, "1", "15", "✗ changes"))
case "REVIEW_REQUIRED":
    rightChips = append(rightChips, prStatusPill(bg, "3", "0", "⟳ review"))
}

// CI block:
case failing:
    rightChips = append(rightChips, prStatusPill(bg, "1", "15", "✗ failing"))
case pending:
    rightChips = append(rightChips, prStatusPill(bg, "3", "0", "… pending"))
default:
    rightChips = append(rightChips, prStatusPill(bg, "2", "0", "✓ passing"))
```

```go
// NEW:
switch pr.ReviewDecision {
case "APPROVED":
    rightChips = append(rightChips, prStatusPill(bg, pal.LabelSuccess, pal.LabelSuccessFg, "✓ approved"))
case "CHANGES_REQUESTED":
    rightChips = append(rightChips, prStatusPill(bg, pal.LabelDanger, pal.LabelDangerFg, "✗ changes"))
case "REVIEW_REQUIRED":
    rightChips = append(rightChips, prStatusPill(bg, pal.LabelWarn, pal.LabelWarnFg, "⟳ review"))
}

// CI block:
case failing:
    rightChips = append(rightChips, prStatusPill(bg, pal.LabelDanger, pal.LabelDangerFg, "✗ failing"))
case pending:
    rightChips = append(rightChips, prStatusPill(bg, pal.LabelWarn, pal.LabelWarnFg, "… pending"))
default:
    rightChips = append(rightChips, prStatusPill(bg, pal.LabelSuccess, pal.LabelSuccessFg, "✓ passing"))
```

- [ ] **Update `summarizeChecks` signature and body**

Old: `func summarizeChecks(checks []github.CheckRun) string`

New:
```go
func summarizeChecks(checks []github.CheckRun, pal Palette) string {
	if len(checks) == 0 {
		return "—"
	}
	pending := false
	for _, c := range checks {
		if c.Conclusion == "FAILURE" || c.Conclusion == "ERROR" || c.Conclusion == "TIMED_OUT" {
			return lipgloss.NewStyle().Foreground(pal.CheckFail).Render("✗")
		}
		if c.Status != "COMPLETED" {
			pending = true
		}
	}
	if pending {
		return lipgloss.NewStyle().Foreground(pal.CheckPending).Render("…")
	}
	return lipgloss.NewStyle().Foreground(pal.CheckPass).Render("✓")
}
```

- [ ] **Update `errorBox` signature and body**

Old: `func errorBox(msg string) string`

New:
```go
func errorBox(msg string, pal Palette) string {
	return lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(pal.Danger).
		Padding(0, 1).
		Foreground(pal.Danger).
		Render("✗ " + msg)
}
```

- [ ] **Build — expect errors in model files (not yet updated)**

```bash
cd /Users/joshua/Development/hubcap/hubcap-app && go build ./... 2>&1
```

Expected: errors in `model_issues.go`, `model_prs.go`, `model_dashboard.go` about wrong number of args to `summarizeChecks`, `errorBox`, `labelStyle`, `labelPill`, `issueRowLabels`. These are fixed in Tasks 4–6.

- [ ] **Commit**

```bash
git add render.go
git commit -m "refactor(render): thread Palette through all colour-producing functions"
```

---

## Task 4: Fix model_issues.go

**Files:**
- Modify: `model_issues.go`

### Background

Five things to fix:
1. `issueListItem.Description()` calls `coloredLabelsCompact` (now deleted) — make plain text
2. `issueRowLabels` separator uses hardcoded `"238"` instead of `pal.TextFaint`
3. `issueRowLabels` calls `labelStyle(l.Name)` — needs `pal` added
4. `issueRowLabels` is called from the delegate without a `pal` arg — fix call site
5. Three `styleGray.Render(...)` calls — replace with palette

- [ ] **Simplify `issueListItem.Description()` to plain text**

Old (line ~134–136):
```go
func (i issueListItem) Description() string {
    return fmt.Sprintf("%s  %s", joinUsers(i.issue.Assignees), coloredLabelsCompact(i.issue.Labels, 60))
}
```

New:
```go
func (i issueListItem) Description() string {
    return fmt.Sprintf("%s  %s", joinUsers(i.issue.Assignees), joinLabels(i.issue.Labels))
}
```

(`joinLabels` is a plain-text label joiner already in render.go.)

- [ ] **Update `issueRowLabels` to accept `pal Palette`**

Old signature: `func issueRowLabels(labels []github.Label, bgKey string, maxW int) string`

New signature: `func issueRowLabels(labels []github.Label, bgKey string, maxW int, pal Palette) string`

Inside the function body, two changes:

1. The separator (line ~319) changes from:
```go
sep := makeBase().Foreground(lipgloss.Color("238")).Render(" · ")
```
to:
```go
sep := makeBase().Foreground(pal.TextFaint).Render(" · ")
```

2. The label style call (line ~325) changes from:
```go
ls := labelStyle(l.Name)
```
to:
```go
ls := labelStyle(l.Name, pal)
```

- [ ] **Update the call site of `issueRowLabels` in the delegate Render method**

In `issueDelegate.Render`, find the call to `issueRowLabels` and add `d.pal`:

Old:
```go
labelPart = issueRowLabels(shown, bgKey, labelMax)
```

New:
```go
labelPart = issueRowLabels(shown, bgKey, labelMax, d.pal)
```

- [ ] **Replace three `styleGray.Render(...)` calls**

These are in the `IssuesModel.Update()` and `issueDetailView()` methods where `m.palette` is in scope.

Find and replace each:

1. Line ~583 (inside `case issueFetchedMsg`):
   ```go
   // OLD:
   m.detail.SetContent(styleGray.Render("Rendering…") + "\n")
   // NEW:
   m.detail.SetContent(lipgloss.NewStyle().Foreground(m.palette.TextDim).Render("Rendering…") + "\n")
   ```

2. Line ~874 (inside another `issueFetchedMsg` or form-submit handler):
   ```go
   // OLD:
   m.detail.SetContent(styleGray.Render("Rendering…") + "\n")
   // NEW:
   m.detail.SetContent(lipgloss.NewStyle().Foreground(m.palette.TextDim).Render("Rendering…") + "\n")
   ```

3. Line ~958 (inside the detail view render, `renderIssueDetailView` or equivalent):
   This is in a function that already receives `pal Palette` — check the function signature. If `pal` is in scope:
   ```go
   // OLD:
   return styleGray.Render("No description.") + "\n"
   // NEW:
   return lipgloss.NewStyle().Foreground(pal.TextDim).Render("No description.") + "\n"
   ```
   If the function receives the model `m` instead, use `m.palette.TextDim`.

- [ ] **Update `errorBox` call**

Line ~930:
```go
// OLD:
b.WriteString(errorBox(fmt.Sprintf("Error: %v\n\nPress r to retry.", m.err)))
// NEW:
b.WriteString(errorBox(fmt.Sprintf("Error: %v\n\nPress r to retry.", m.err), m.palette))
```

- [ ] **Build**

```bash
cd /Users/joshua/Development/hubcap/hubcap-app && go build ./... 2>&1
```

Expected: errors only in model_prs.go and model_dashboard.go now.

- [ ] **Commit**

```bash
git add model_issues.go
git commit -m "refactor(issues): replace hardcoded colours with palette in list and detail"
```

---

## Task 5: Fix model_prs.go

**Files:**
- Modify: `model_prs.go`

### Background

Mirror of Task 4 for PRs. Four things:
1. `prListItem.Description()` calls `summarizeChecks` (now requires pal) — simplify to plain text
2. `issueRowLabels` call needs `pal` added
3. Three `styleGray.Render(...)` calls to replace
4. `errorBox` call needs `pal`

- [ ] **Simplify `prListItem.Description()` to plain text**

Old (line ~150–156):
```go
func (p prListItem) Description() string {
    status := p.pr.State
    if p.pr.IsDraft {
        status = "draft"
    }
    return fmt.Sprintf("%s  %s  %s", p.pr.Author.Login, status, summarizeChecks(p.pr.StatusRollup))
}
```

New:
```go
func (p prListItem) Description() string {
    status := p.pr.State
    if p.pr.IsDraft {
        status = "draft"
    }
    return fmt.Sprintf("%s  %s", p.pr.Author.Login, status)
}
```

- [ ] **Update the `issueRowLabels` call in `prDelegate.Render`**

Find the call and add `d.pal`:

Old:
```go
labelStr := issueRowLabels(shownLabels, bgKey, labelBudget)
```

New:
```go
labelStr := issueRowLabels(shownLabels, bgKey, labelBudget, d.pal)
```

- [ ] **Replace three `styleGray.Render(...)` calls**

1. Line ~587 (inside `case prFetchedMsg`):
   ```go
   // OLD:
   m.detail.SetContent(styleGray.Render("Rendering…") + "\n")
   // NEW:
   m.detail.SetContent(lipgloss.NewStyle().Foreground(m.palette.TextDim).Render("Rendering…") + "\n")
   ```

2. Line ~911 (inside a form-submit or refresh handler):
   ```go
   // OLD:
   m.detail.SetContent(styleGray.Render("Rendering…") + "\n")
   // NEW:
   m.detail.SetContent(lipgloss.NewStyle().Foreground(m.palette.TextDim).Render("Rendering…") + "\n")
   ```

3. Line ~984 (inside `renderPRDetailView` which receives `pal Palette`):
   ```go
   // OLD:
   return styleGray.Render("No description.") + "\n"
   // NEW:
   return lipgloss.NewStyle().Foreground(pal.TextDim).Render("No description.") + "\n"
   ```

- [ ] **Update `errorBox` call**

Line ~967:
```go
// OLD:
b.WriteString(errorBox(fmt.Sprintf("Error: %v\n\nPress r to retry.", m.err)))
// NEW:
b.WriteString(errorBox(fmt.Sprintf("Error: %v\n\nPress r to retry.", m.err), m.palette))
```

- [ ] **Build**

```bash
cd /Users/joshua/Development/hubcap/hubcap-app && go build ./... 2>&1
```

Expected: errors only in model_dashboard.go now.

- [ ] **Commit**

```bash
git add model_prs.go
git commit -m "refactor(prs): replace hardcoded colours with palette in list and detail"
```

---

## Task 6: Fix model_dashboard.go

**Files:**
- Modify: `model_dashboard.go`

### Background

Two `summarizeChecks` call sites and one `errorBox` call. All are inside a render function where `pal` is available as a local variable (`pal := m.palette` at line ~273).

- [ ] **Update both `summarizeChecks` calls**

Line ~423:
```go
// OLD:
line2Left := base.Foreground(pal.TextMuted).Render("@"+truncate(p.Author.Login, 14)) +
    dimSep + summarizeChecks(p.StatusRollup)
// NEW:
line2Left := base.Foreground(pal.TextMuted).Render("@"+truncate(p.Author.Login, 14)) +
    dimSep + summarizeChecks(p.StatusRollup, pal)
```

Line ~438:
```go
// OLD:
if checks := summarizeChecks(p.StatusRollup); checks != "" {
// NEW:
if checks := summarizeChecks(p.StatusRollup, pal); checks != "" {
```

- [ ] **Update `errorBox` call**

Line ~262:
```go
// OLD:
return errorBox(fmt.Sprintf("Dashboard error: %v\n\nPress r to retry.", m.err))
// NEW:
return errorBox(fmt.Sprintf("Dashboard error: %v\n\nPress r to retry.", m.err), m.palette)
```

- [ ] **Build — expect clean**

```bash
cd /Users/joshua/Development/hubcap/hubcap-app && go build ./... 2>&1
```

Expected: zero errors.

- [ ] **Commit**

```bash
git add model_dashboard.go
git commit -m "refactor(dashboard): thread palette through summarizeChecks and errorBox"
```

---

## Task 7: Delete dead code from terminal.go

**Files:**
- Modify: `terminal.go`

### Background

`terminal.go` contains standalone CLI functions (`withSpinner`, `startSpinner`, `menu`, `numberedMenu`) plus a package-level `spinnerModel`/`spinnerStyle` with hardcoded colour `"6"` and `"86"`. None of these are called from any `.go` file. The `init()` function initialises the dead spinner. Remove them all.

- [ ] **Delete the following from terminal.go:**

  1. `var spinnerModel = spinner.New()` (package-level)
  2. `var spinnerStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("6"))` (package-level)
  3. `func init() { ... }` block that initialises the spinner
  4. `func withSpinner(message string, fn func() error) error { ... }`
  5. `func startSpinner(message string) chan struct{} { ... }`
  6. `func menu(reader *bufio.Reader, options []string) string { ... }`
  7. `func numberedMenu(reader *bufio.Reader, options []string) string { ... }`

- [ ] **Remove now-unused imports from terminal.go**

After the deletions, check which imports are still needed. Likely candidates for removal:
- `"github.com/charmbracelet/bubbles/spinner"` — used only by `spinnerModel`
- `"github.com/charmbracelet/lipgloss"` — used only by `spinnerStyle`

Remove any import that the compiler flags as unused.

- [ ] **Build — expect clean**

```bash
cd /Users/joshua/Development/hubcap/hubcap-app && go build ./... 2>&1
```

Expected: zero errors.

- [ ] **Verify no hardcoded colours remain outside theme.go**

```bash
grep -rn 'lipgloss\.Color("' /Users/joshua/Development/hubcap/hubcap-app --include="*.go" | grep -v "theme.go" | grep -v "_test.go"
```

Expected: only `lipgloss.Color("")` lines (transparent bg, which is not a colour value) — no numeric or hex strings.

- [ ] **Commit**

```bash
git add terminal.go
git commit -m "refactor(terminal): remove dead CLI spinner and menu code with hardcoded colours"
```

---

## Self-review

### Spec coverage

| Requirement | Task |
|---|---|
| No hardcoded `lipgloss.Color("...")` outside theme.go | Tasks 1–7 |
| All label foreground colours from palette | Task 3 (`labelStyle`) |
| All label pill bg/fg from palette | Task 3 (`labelPillColors`, `labelPill`) |
| PR detail review/CI status pills from palette | Task 3 (`renderPRMetaStrip`) |
| `stateIndicator` colours from palette | Task 3 |
| `summarizeChecks` colours from palette | Tasks 3, 6 |
| `errorBox` colour from palette | Tasks 3, 4, 5, 6 |
| `issueRowLabels` separator from palette | Task 4 |
| "Rendering…" / "No description." from palette | Tasks 4, 5 |
| Dead terminal spinner/menu removed | Task 7 |

### Placeholder scan

No TBDs or "implement later" items found.

### Type consistency

- `labelStyle(name string, pal Palette)` — defined Task 3, called Task 4 ✓
- `labelPill(stripBg, name, pal)` — defined Task 3, called Task 3 (renderIssueMetaStrip / renderPRMetaStrip) ✓
- `issueRowLabels(labels, bgKey, maxW, pal)` — updated Task 4, called Tasks 4 and 5 ✓
- `summarizeChecks(checks, pal)` — defined Task 3, called Tasks 5 and 6 ✓
- `errorBox(msg, pal)` — defined Task 3, called Tasks 4, 5, and 6 ✓
- `stateIndicator(state, isDraft, pal)` — defined and called Task 3 ✓
