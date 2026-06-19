# hubcap

<img src="https://pub-98b467145a3b4d5aab71817835431ccc.r2.dev/images/hubcap-logo.png" alt="hubcap logo" width="200"/>

A terminal UI for browsing GitHub issues and pull requests, built on top of the [GitHub CLI (`gh`)](https://cli.github.com/).

## Features

- **Issues tab** — list, filter, view, create, close/reopen, assign, label, and develop branches from issues
- **Pull Requests tab** — list, filter, view, create, checkout, merge (merge commit / squash / rebase), close/reopen PRs
- **Interactive navigation** — arrow-key selection, tab-switching between Issues and PRs, single-key shortcuts
- **CI check rollup** — at-a-glance pass/fail/pending status for each PR
- **Filterable lists** — filter by state, assignee, label, milestone (issues), draft status, review status, and result limit
- **Copy URL** — copy issue or PR URL to clipboard (`pbcopy` on macOS, `wl-copy`/`xclip` on Linux)
- **Raw-mode fallback** — works in terminals that don't support raw mode via a numbered prompt fallback

## Requirements

- [`gh`](https://cli.github.com/) authenticated and on your `PATH`
- Run from inside a directory that is part of a GitHub repository (or any subdirectory thereof)

## Installation

### Homebrew (recommended)

```sh
brew tap TranspiledCode/tap
brew install hubcap
```

### Build from source

```sh
go build -o hubcap .
mv hubcap /usr/local/bin/
```

## Usage

Run `hubcap` from within any Git repository that has a GitHub remote:

```sh
hubcap
```

### Keyboard shortcuts

| Key                    | Action                             |
| ---------------------- | ---------------------------------- |
| `↑` / `↓`              | Navigate list                      |
| `Enter`                | Open selected item                 |
| `Tab` / `Shift+Tab`    | Switch between Issues and PRs tabs |
| `n`                    | New issue / new PR                 |
| `f`                    | Change filters                     |
| `r`                    | Refresh list                       |
| `q` / `Esc` / `Ctrl+C` | Quit / back                        |
| `1`–`9`                | Jump to item by position           |

### Issue actions

From an issue detail view:

- **Develop branch** — create and check out a branch linked to the issue (`gh issue develop`)
- **Create PR** — open a PR creation prompt pre-filled from the current branch
- **Close / Reopen issue**
- **Assign to @me**
- **Add label**
- **Open in browser**
- **Copy URL**

### PR actions

From a PR detail view:

- **Checkout branch** — check out the PR's head branch locally
- **Merge** — merge commit, squash, or rebase
- **Close / Reopen PR**
- **Open in browser**
- **Copy URL**

## Running tests

```sh
go test ./...
```

## How it works

`hubcap` shells out to `gh` for all GitHub API calls, parsing the JSON output directly. Terminal raw mode is enabled via `stty` for interactive navigation and gracefully falls back to a numbered-prompt interface when raw mode is unavailable.
