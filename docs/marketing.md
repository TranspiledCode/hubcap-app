# Hubcap Marketing

## What is Hubcap?

Hubcap is a keyboard-driven terminal UI for GitHub that keeps developers in the flow. Instead of context-switching to the browser, you can browse issues, review pull requests, check CI status, and take action — all without leaving your terminal.

---

## Value Proposition

**Stay in flow.** Developers live in the terminal. Hubcap brings GitHub there too — no browser tabs, no context switching, no breaking your rhythm.

**Ship faster.** Keyboard navigation, single-key shortcuts, and smart filtering make triaging issues and reviewing PRs feel instant.

**Works with what you have.** Built on the GitHub CLI (`gh`), Hubcap uses your existing auth and repos. No new accounts, no setup friction.

---

## Target Audience

- **Terminal-first developers** who prefer the command line over browser-based tools
- **Small-to-medium teams** where triaging issues and reviewing PRs is a daily workflow
- **GitHub users** who want a faster, more focused alternative to the web interface
- **Developers who value flow** and want to minimize context switching

---

## Key Features

### Issues Tab
- List, filter, view, create, close/reopen, assign, and label issues
- Develop branches directly from issues with automatic naming
- Filter by state, assignee, label, milestone, and result limit

### Pull Requests Tab
- List, filter, view, create, checkout, and merge PRs
- At-a-glance CI check status (pass/fail/pending) for each PR
- Merge options: merge commit, squash, or rebase
- Filter by state, assignee, label, draft status, review status

### Interactive Navigation
- Arrow-key selection through lists
- Tab-switching between tabs
- Single-key shortcuts for common actions
- Numbered jump shortcuts (1-9)

### Developer Experience
- Raw-mode terminal UI with graceful fallback to numbered prompts
- Copy URLs to clipboard (`pbcopy` on macOS, `wl-copy`/`xclip` on Linux)
- Open issues/PRs in browser when needed
- Works in terminals that don't support raw mode

---

## Positioning

**Against GitHub web interface:** Hubcap is faster, keyboard-driven, and keeps you in the terminal. It's for developers who want speed and focus over rich visual editing.

**Against other GitHub TUIs:** Hubcap is built on the GitHub CLI, so it leverages your existing setup and auth. It's lightweight, focused, and doesn't require additional configuration.

**Against `gh` CLI alone:** Hubcap adds an interactive layer on top of `gh` — you get the power of the CLI with the discoverability and ease of use of a visual interface.

---

## Use Cases

**Daily triage:** Start your day by reviewing assigned issues and PRs, checking CI status, and deciding what to work on — all from one screen.

**Review workflow:** Quickly scan review requests, see CI status at a glance, checkout branches, and merge — without leaving your terminal.

**Issue management:** Create issues, assign them to yourself, develop branches, and open PRs — all with keyboard shortcuts.

**Team coordination:** Filter by assignee or label to see what teammates are working on, grab available issues, and stay aligned.

---

## Messaging Framework

### Tagline
GitHub, in your terminal.

### Subtitle
A keyboard-driven TUI for managing GitHub issues and pull requests — browse, filter, merge, and act without leaving your terminal.

### Elevator Pitch
Hubcap is a fast, keyboard-first terminal interface for GitHub. Built on the GitHub CLI, it gives you an interactive way to browse issues, review PRs, check CI status, and take action — all without leaving your terminal. Arrow-key navigation, single-key shortcuts, and smart filtering make it feel like a native app for your GitHub workflow.

---

## Differentiators

1. **Built on `gh`** — leverages your existing GitHub CLI setup and authentication
2. **Keyboard-first** — designed for terminal power users who prefer shortcuts over clicks
3. **Focused** — does one thing well: issues and PRs in the terminal
4. **Graceful fallback** — works in terminals that don't support raw mode
5. **No configuration** — runs out of the box with your existing `gh` setup

---

## Future Roadmap

- **My Work Dashboard** — landing screen showing review requests, your open PRs, assigned issues, and available work
- **Configurable filters** — save and reuse common filter combinations
- **Multi-repo support** — switch between repositories without leaving hubcap
- **Enhanced PR review** — approve, comment, and view diffs from the terminal

---

## Call to Action

**Install Hubcap:**

```sh
go build -o hubcap .
mv hubcap /usr/local/bin/
```

**Run it from any GitHub repo:**

```sh
hubcap
```

**Requirements:** Go 1.21+, `gh` CLI authenticated and on your `PATH`

---

## Resources

- **GitHub:** [repository link]
- **Documentation:** [README link]
- **License:** [license link]
