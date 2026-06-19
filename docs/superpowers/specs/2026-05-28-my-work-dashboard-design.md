# hubcap — My Work Dashboard Design

**Date:** 2026-05-28
**Status:** Approved

## Overview

Add a "My Work" dashboard as the default landing screen for hubcap. The dashboard gives developers on small-to-medium teams an immediate answer to "what's on my plate and what can I pick up?" without leaving the terminal. The existing Issues and PRs tabs remain unchanged and fully functional.

## Goals

- Land on a useful, actionable view the moment hubcap opens
- Show review requests, your open PRs (with CI status), assigned issues, and available work in one screen
- Allow full keyboard navigation and action directly from the dashboard
- Keep the existing Issues/PRs tabs intact — the dashboard is additive, not a replacement

## Non-Goals

- Full PR review workflow (approve, comment, view diffs)
- Per-repo config (global config only for now)
- Multi-repo switching

---

## File Structure

The monolithic `main.go` is split into focused files, all in `package main`, plus a new `internal/github/` package for all `gh` CLI interaction.

```
hubcap/
├── main.go                   # AppState, entry point, main tab loop only
├── dashboard.go              # My Work tab: data model, render, navigation
├── issues.go                 # Issues tab (browseIssues, issueList, viewIssue, configureFilters)
├── prs.go                    # PRs tab (browsePRs, prList, viewPR, configurePRFilters)
├── terminal.go               # Raw mode, stty, termSize, clearScreen, prompt, menu, clipboard
├── render.go                 # renderHeader, shared display helpers (truncate, joinUsers, etc.)
├── config.go                 # Load/save ~/.config/hubcap/config.json
└── internal/
    └── github/
        ├── issues.go         # fetchIssues, fetchIssue, closeIssue, reopenIssue, assignIssueSelf, addIssueLabel
        ├── prs.go            # fetchPRs, fetchPR, closePR, reopenPR
        └── repo.go           # fetchRepo, runCommand, runCommandPassthrough
```

### Boundaries

- `internal/github/` owns all `gh` subprocess calls and JSON parsing. It has no knowledge of terminal state or UI.
- UI files (`dashboard.go`, `issues.go`, `prs.go`) import `internal/github/` for data and call into `terminal.go` / `render.go` for display.
- `main.go` only wires together `AppState` and the top-level tab loop.

---

## AppState Changes

Add `TabDashboard` as the first tab (value 0), shifting existing tabs:

```go
const (
    TabDashboard TabID = iota  // new — default landing
    TabIssues
    TabPRs
)
```

Add dashboard state to `AppState`:

```go
type AppState struct {
    ActiveTab        TabID
    IssueFilters     Filters
    PRFilters        PRFilters
    IssueSelected    int
    PRSelected       int
    DashboardCursor  int  // flat index across all visible dashboard rows
    Repo             string
}
```

---

## Dashboard Design

### Sections (in display order)

| Section | Source | Notes |
|---|---|---|
| Review Requests | `gh pr list --reviewer @me --state open` | PRs where you are a requested reviewer |
| My Open PRs | `gh pr list --author @me --state open` | Your own open PRs with CI status |
| Assigned to Me | `gh issue list --assignee @me --state open` | Issues assigned to you |
| Available to Grab | Configurable filter (see Config) | Issues matching the saved `available_filter` |

### Rendering

- Each section has a collapsible header showing name and item count: `▾ REVIEW REQUESTS (2)`
- Collapsed sections show: `▸ REVIEW REQUESTS (2)` — count stays visible
- Empty sections are hidden entirely
- Items show: state indicator, number, title (truncated), labels or CI status
- The cursor (`>`) moves through all rows (section headers + items) as a flat list

### Navigation

| Key | Action |
|-----|--------|
| `↑` / `↓` | Move cursor through all rows |
| `Enter` on section header | Toggle collapse/expand |
| `Enter` on item | Open existing issue or PR detail view |
| `←` | Collapse the section the cursor is in |
| `Tab` / `Shift+Tab` | Switch to next/previous tab |
| `1` / `2` / `3` | Jump directly to My Work / Issues / PRs tab (dashboard only — in list views these keys already jump to items by position) |
| `n` | New issue |
| `p` | New PR |
| `r` | Refresh all sections |
| `c` | Open hubcap config screen |
| `q` / `Esc` / `Ctrl+C` | Quit |

### Data Loading

All four sections fetch concurrently using goroutines. Each section renders independently — if one `gh` call fails, that section shows an inline error message; other sections still display normally. The user can press `r` to retry all sections.

### Opening Items

Opening an issue or PR from the dashboard reuses the existing `viewIssue` / `viewPR` functions unchanged. Pressing Back from a detail view returns to the dashboard with the cursor restored to its previous position.

---

## Tab Bar

The tab bar updates from two tabs to three:

```
[ ● My Work ]  [ Issues ]  [ Pull Requests ]
```

- My Work is always the startup default (`ActiveTab: TabDashboard`)
- `1` / `2` / `3` added as direct-jump shortcuts from the dashboard only (list views already use these keys for item jumping)
- Existing Tab/Shift+Tab cycling behaviour preserved, now cycles through three tabs

---

## Config System

### Location

`~/.config/hubcap/config.json` — global, not per-repo.

### Schema

```json
{
  "available_filter": {
    "state": "open",
    "assignee": "",
    "label": "",
    "milestone": "",
    "limit": 25
  }
}
```

### Behaviour

- Missing config file: use defaults (`state: open`, no assignee/label/milestone, `limit: 25`) — no error
- Malformed JSON: log a warning to stderr, use defaults — no crash
- Writes happen immediately on save; no restart required

### Config Screen

Accessible via `c` from the dashboard (and from the header in other tabs). Uses the same `menu()` + `prompt()` pattern as the existing filter screens:

```
Configure hubcap
  Change "Available to Grab" filter
    > state / assignee / label / milestone / limit
  Reset to defaults
  Back
```

---

## Error Handling

| Scenario | Behaviour |
|---|---|
| One dashboard section fails | Section shows `⚠ Could not load — r to retry` inline; other sections unaffected |
| All sections fail | Full dashboard error state with retry prompt |
| Config file unreadable | Use defaults silently |
| Config write fails | Print error, continue with in-memory state |
| `gh` not in PATH | Existing fatal check at startup, unchanged |

---

## Testing

- `internal/github/` functions are tested with table-driven tests using captured `gh` JSON output fixtures
- `config.go` load/save round-trip tested with temp files
- Pure utility functions (`deriveBranchName`, `truncate`, `truncateLines`, etc.) moved to `render.go` and covered in `main_test.go`
- Dashboard section merging / cursor logic tested as pure functions where possible

---

## Build & Migration

- No new external dependencies — stdlib only
- Existing `go.mod` unchanged
- Refactor is purely mechanical: move existing functions into their new files, adjust imports, verify `go build` and `go test ./...` pass before adding any new features
- The compiled `hubcap` binary in the repo root should be removed and added to `.gitignore`
