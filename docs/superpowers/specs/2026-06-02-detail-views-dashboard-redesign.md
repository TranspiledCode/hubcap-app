# Detail Views & Dashboard Redesign

**Date:** 2026-06-02  
**Status:** Approved  

---

## Problem

The current UI has three compounding issues:

1. **Double footer** — detail views render their own hint bar inside the body, and the global footer always renders below it. Two footers, one irrelevant (global shows `n new issue`, `f filters` while reading a detail).
2. **Metadata scrolls away** — issue/PR title, state, assignees, and labels live inside the scrollable viewport. On any issue with a long body, the metadata disappears.
3. **Dashboard shows non-personal work** — the "Available to Grab" section was unrelated to the user's own work and cluttered the "My Work" intent of the dashboard.

---

## Solution Overview

**Sticky metadata strip** above the viewport in all detail views. **Context-aware single footer** — global footer suppressed when a detail is open; detail actions take its place. **Dashboard trimmed** to three personal sections only.

---

## Issue Detail View

### Metadata Strip

Rendered above the viewport, fixed (does not scroll). Three visual rows:

```
┌─────────────────────────────────────────────────────────────────┐
│ Epic: Optional local AI via Ollama for issue/PR drafting        │  ← title (orange, bold)
│ ● OPEN · #5 · opened by joshuacrass · assigned to joshuacrass  │  ← state row
│ [type:enhancement] [priority:medium] [effort:5]                 │  ← labels row (hidden if none)
└─────────────────────────────────────────────────────────────────┘
```

- **State dot**: colored circle (green=open, red=closed)
- **STATE label**: `OPEN` (green) / `CLOSED` (red) — bold
- Assignees row only rendered when `len(issue.Assignees) > 0`
- Labels row only rendered when `len(issue.Labels) > 0`

### Body Viewport

Pure scrollable body content — no title or metadata inside the viewport content. `renderIssueDetailContent()` strips the title/metadata block it currently writes.

Viewport height is recalculated accounting for strip height (title line + up to 2 meta rows ≈ 3–4 lines depending on label presence).

### Action Footer

Replaces the global footer entirely when detail is active:

```
[d] develop · [p] PR · [c] close · [a] assign · [l] label · [o] browser · [u] copy URL · [b] back
```

- `[c]` label changes to `reopen` when `issue.State == "closed"`

### Filter Bar

Hidden when in detail view (irrelevant context). The `headerView()` function skips rendering the filter bar row when `IssuesModel.showDetail == true`.

---

## PR Detail View

Same sticky strip pattern as issues, with PR-specific fields.

### Metadata Strip

```
┌────────────────────────────────────────────────────────────────┐
│ feat(auth): add OAuth2 authentication flow                     │  ← title (orange, bold)
│ ◐ DRAFT · #12 · by alice · ⎇ 12-add-oauth2-auth-flow          │  ← state/branch row
│ ⟳ REVIEW REQUIRED · ✗ checks failing · [type:feature]         │  ← review/checks/labels row
└────────────────────────────────────────────────────────────────┘
```

**State indicators:**
- `● OPEN` (green)
- `◐ DRAFT` (yellow)  
- `✓ MERGED` (purple)
- `✗ CLOSED` (red)

**Review decision colors:**
- `APPROVED` → green
- `REVIEW REQUIRED` → yellow
- `CHANGES REQUESTED` → red
- Empty → hidden

**CI checks** (from `summarizeChecks()`):
- `✓ checks passing` (green)
- `✗ checks failing` (red)
- `… checks pending` (yellow)
- `—` when no checks

Branch row hidden when `pr.HeadRefName == ""`.  
Labels row hidden when no labels and no review decision.

### Action Footer

```
[c] checkout · [m] merge · [x] close · [o] browser · [u] copy URL · [r] refresh · [b] back
```

- `[x]` label changes to `reopen` when `pr.State == "closed"`

---

## Dashboard — My Work

Three sections only. **Available to Grab removed** from this view; users who want unassigned issues use the Issues tab with filters.

### Sections

| Icon | Section | Data source |
|------|---------|-------------|
| `⟳` | REVIEW REQUESTS | `FetchPRs` with `Search: "review-requested:@me"` |
| `⎇` | MY OPEN PRs | `FetchPRs` with `Author: "@me", State: "open"` |
| `◉` | ASSIGNED TO ME | `FetchIssues` with `Assignee: "@me", State: "open"` |

Empty sections are hidden entirely (current behavior, no change).

### Row Format

**PR rows** (review requests, my PRs):
```
  PR  #14  feat(ui): redesign settings panel with new layout   by carol   ✓
```
- `PR` badge in purple (`#a371f7`)
- Number in blue
- Title truncated to fill available width
- Author right-aligned (for review requests); `draft` for draft PRs in "my PRs"
- CI check symbol at far right (`✓`/`✗`/`…`/`—`)

**Issue rows** (assigned):
```
  IS  #5   Epic: Optional local AI via Ollama for issue/PR...  [priority:medium]
```
- `IS` badge in green (`#3fb950`)
- Number in blue
- Title truncated
- Single most-significant label at right (priority label preferred; falls back to first label)

### Section Headers

```
⟳ REVIEW REQUESTS (2)
```
- Icon + name in orange bold
- Count in parentheses, muted purple
- Top border `#21262d` between sections

### Footer

No change from current: `[↑↓] move · [enter] open · [tab] switch · [r] refresh · [q] quit`

---

## Technical Implementation

### Double footer fix

In `AppModel.View()`, detect when a detail is active and suppress the global footer:

```go
inDetail := (m.activeTab == TabIssues && m.issues.showDetail) ||
            (m.activeTab == TabPRs && m.prs.showDetail)

if !inDetail {
    footer = footerView(m.activeTab, innerW)
}
```

When `inDetail` is true, the detail action bar inside the model's `View()` output serves as the only footer.

### New render functions

Add to `render.go`:

- `renderIssueMetaStrip(issue github.Issue, width int) string` — returns the 2–3 line lipgloss strip
- `renderPRMetaStrip(pr github.PullRequest, width int) string` — returns the 2–3 line lipgloss strip

Both functions are called from `IssuesModel.View()` / `PRsModel.View()` when `showDetail == true`, rendered before the viewport.

### Viewport height

Current: `viewport.New(m.width-4, m.height-headerHeight()-4)`  
New: `viewport.New(m.width-4, m.height-headerHeight()-metaStripHeight-4)`

`metaStripHeight` is a constant = `4` (title + state row + labels row + 1 separator line). If labels are absent, it's `3`; for simplicity use a fixed value of `4` (one extra blank line is acceptable).

### renderIssueDetailContent cleanup

Remove the title, state, author, assignees, and labels from `renderIssueDetailContent()`. It should render only the body text going forward, since the strip handles all metadata.

### renderPRDetailContent cleanup

Same — strip title, branch, state, review, checks, labels out of viewport content. Body only.

### Dashboard data model

Remove `availableIssues` from `dashboardData`:

```go
type dashboardData struct {
    reviewRequests []github.PullRequest
    myPRs          []github.PullRequest
    assignedIssues []github.Issue
    // availableIssues removed
}
```

`fetchCmd()` drops to 3 concurrent goroutines. `buildDashRows()` drops the `secAvailable` case.

`Config.AvailableFilter` field is retained in the struct (ignored for now; available for future use).

### Dashboard row rendering

Update `DashboardModel.View()` to render the new row format:
- PR rows: `PR` badge, number, title (flex), author, CI check
- Issue rows: `IS` badge, number, title (flex), dominant label

Use `coloredLabelsCompact(issue.Labels, 25)` from `render.go` — this already uses `dominantLabelStyle` internally and returns a rendered string.

### headerView filter bar suppression

In `render.go`'s `headerView()`, the filter bar row for the Issues tab (state/assignee/label/limit) should be hidden when in detail view. Pass `showDetail bool` as a parameter, or expose it through a `headerHeight()` recalculation. Simplest approach: add a `detailActive bool` parameter to `headerView()`.

---

## Files Changed

| File | Change |
|------|--------|
| `render.go` | Add `renderIssueMetaStrip`, `renderPRMetaStrip`; update `headerView` signature for `detailActive` |
| `model_issues.go` | Use strip in View; shrink viewport height; strip metadata from `renderIssueDetailContent`; update detail footer |
| `model_prs.go` | Use strip in View; shrink viewport height; strip metadata from `renderPRDetailContent`; update detail footer |
| `model_dashboard.go` | Remove `secAvailable`; update row rendering; new section header icons/format |
| `app.go` | Suppress global footer when detail active; update `headerView` call |
| `dashboard_test.go` | Remove `secAvailable` cases; add tests for new 3-section row format |

---

## Non-Goals

- No change to the list views (Issues or PRs list screens)
- No change to filter forms or create forms  
- `Config.AvailableFilter` kept for future use but not surfaced in this change
- No new keyboard shortcuts introduced
