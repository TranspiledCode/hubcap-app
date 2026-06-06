# AI Workflow Instructions

> **Purpose:** This document defines the exact development workflow and process for AI-assisted development AND serves as a comprehensive reference guide for human developers. Follow these instructions precisely to maintain consistency across AI assistants, human developers, and projects.

---

## Table of Contents

### 👥 Getting Started (For Everyone)

- [Quick Start for Humans](#-quick-start-for-humans-5-minute-overview)
- [Quick Start for AI Assistants](#-quick-start-for-ai-assistants)
- [Visual Workflow Diagram](#visual-workflow-diagram)

### 📋 Workflow Phases (Detailed)

- [Phase 1: Planning & Discussion](#phase-1-planning--discussion)
- [Phase 2: Issue & Branch Creation](#phase-2-issue--branch-creation)
- [Phase 3: Implementation](#phase-3-implementation)
- [Phase 4: User Testing & Verification](#phase-4-user-testing--verification)
- [Phase 5: Commit & Push](#phase-5-commit--push)
- [Phase 6: Release Process](#phase-6-release-process)
- [Phase 7: Hotfix Process](#phase-7-hotfix-process-production-emergencies)

### ⚙️ Quality & Standards

- [Core Principles](#core-principles)
- [Code Quality Standards](#code-quality-standards)
- [Testing Philosophy](#2-testing-philosophy)

### 📖 Command References

- [Git Commands Reference](#git-commands-reference)
- [GitHub CLI Commands Reference](#github-cli-commands-reference)
- [Go Commands Reference](#go-commands-reference)
- [Quick Copy-Paste Commands](#quick-copy-paste-commands)

### 🔀 Decision Trees

- [Should I Use Hotfix or Normal Flow?](#decision-tree-hotfix-vs-normal-flow)
- [What Commit Type Should I Use?](#decision-tree-commit-types)
- [When to Write Tests?](#decision-tree-when-to-write-tests)
- [Which Label to Apply?](#decision-tree-label-selection)

### 🛠️ Troubleshooting Guide

- [Common Problems & Solutions](#troubleshooting-guide)
- [My Commit Was Rejected](#my-commit-was-rejected)
- [Tests Are Failing](#tests-are-failing)
- [Merge Conflicts](#merge-conflicts)
- [Wrong Branch](#wrong-branch)
- [Emergency Procedures](#emergency-procedures)

### 🚀 Project Setup

- [For New Team Members](#-for-new-team-members)
- [First-Time Setup Checklist](#first-time-setup-checklist)
- [Required Tools Installation](#required-tools-installation)
- [First Feature Walkthrough](#first-feature-walkthrough)
- [Common Gotchas for Newcomers](#common-gotchas-for-newcomers)

### 📚 Reference Tables

- [Labels Reference Table](#labels-reference-table)
- [Commit Types Comparison](#commit-types-comparison-table)
- [Version Bump Matrix](#version-bump-matrix)

### 🤖 AI-Specific Sections

- [AI Assistant: Non-Negotiable Rules](#-ai-assistant-non-negotiable-rules)
- [How to Ensure AI Follow This Document](#-how-to-ensure-ai-assistants-follow-this-document)
- [AI Assistant Response Templates](#-ai-assistant-response-templates)

### 📝 Additional Resources

- [Communication Style](#communication-style)
- [Branch Cleanup Checklist](#branch-cleanup-checklist)
- [Project-Specific Notes](#project-specific-notes)
- [Success Metrics](#success-metrics)
- [Quick Reference Card](#quick-reference-card)

---

## 👥 Quick Start for Humans (5-Minute Overview)

**Welcome! This guide helps you (and AI assistants) follow our development workflow.**

### Visual Workflow Diagram

```
┌──────────────────────────────────────────────────────────────────────────┐
│                         DEVELOPMENT WORKFLOW                             │
└──────────────────────────────────────────────────────────────────────────┘

 1. PLANNING          2. ISSUE/BRANCH       3. IMPLEMENT
┌─────────────┐      ┌─────────────┐      ┌─────────────┐
│ Discuss     │      │ gh issue    │      │ Code        │
│ requirements│─────▶│ create      │─────▶│ changes     │
│ Present plan│      │ gh issue    │      │ Test as you │
│ 🛑 GET      │      │ develop     │      │ go          │
│ APPROVAL    │      └─────────────┘      └─────────────┘
└─────────────┘                                  │
                                                 ▼

 6. RELEASE           5. COMMIT/PUSH        4. TESTING
┌─────────────┐      ┌─────────────┐      ┌─────────────┐
│ git tag vX.Y.Z│      │ git commit  │      │ go fmt ./... │
│ Push tags   │◀─────│ git push    │◀─────│ go vet ./... │
│ PR to main  │      │ gh pr create│      │ Browser test│
│ Deploy      │      └─────────────┘      │ 🛑 GET      │
└─────────────┘                           │ APPROVAL    │
                                          └─────────────┘

                    🛑 = CHECKPOINT - WAIT FOR APPROVAL
```

### The 6 Phases

1. **Planning** - Discuss what to build, get approval (🛑 STOP here)
2. **Issue/Branch** - Create GitHub issue and branch
3. **Implementation** - Write code, test incrementally
4. **Testing** - Format, lint, test, get approval (🛑 STOP here)
5. **Commit/Push** - Commit with conventional format, create PR
6. **Release** - Version bump, changelog, deploy to production

### Most Commonly Used Commands

```bash
# Start a new feature
gh issue create --title "Add login form" --body "Description"
gh issue develop 123 --checkout

# During development (run frequently!)
go fmt ./...       # Format code
go vet ./...       # Run static analysis
go test ./...      # Run tests

# Ready to commit? (only after testing & approval!)
git add .
git commit -m "feat(auth): add login form

- Created login form component
- Added validation
- Integrated with API

Closes #123"

git push origin 123-add-login-form
gh pr create --fill

# Ready to release? (on dev branch)
# Create version tag manually
git tag -a v1.0.0 -m "Release v1.0.0"
git push --follow-tags origin dev  # Push release commit + tags
gh pr create --base main --head dev --fill
```

### Quick Tips

- ⚠️ **Always format BEFORE vetting**: `go fmt ./... && go vet ./...`
- ⚠️ **Never commit without approval**: Always get someone to test first
- ⚠️ **Use conventional commits**: `feat:`, `fix:`, `docs:`, etc.
- ⚠️ **One feature = One branch**: Created from GitHub issues
- ⚠️ **Test incrementally**: Don't wait until the end

### Need More Detail?

- **Detailed workflow**: See [Development Workflow](#development-workflow)
- **Commit format**: See [Commit Types](#commit-types-comparison-table)
- **Troubleshooting**: See [Troubleshooting Guide](#troubleshooting-guide)
- **Decision guides**: See [Decision Trees](#decision-trees)

---

## 🤖 Quick Start for AI Assistants

### ⚠️ CRITICAL: Instructions for AI Assistants

**Before ANY development work:**

1. 📖 Read this document
2. 📋 Present a plan → 🛑 WAIT for approval
3. 🔨 Implement step-by-step
4. ✅ `go fmt ./... && go vet ./...` → Fix all errors
5. 🧪 Ask developer to test → 🛑 WAIT for approval
6. 💾 Only then commit

**🚫 NEVER:**

- Skip planning or commit without testing
- Ignore the 🛑 checkpoints
- Commit with linting errors

**If you are an AI assistant, you MUST:**

1. **Read this entire document** before starting any development work
2. **Follow every step in order** - no skipping, no shortcuts
3. **Stop and ask** if anything is unclear or ambiguous
4. **Use the checklists** provided and confirm completion of each step
5. **Wait for user approval** at designated checkpoints (marked with 🛑)
6. **Reference this document** when the user asks you to follow the workflow

**How to use this document:**

- When starting a new task, read the relevant workflow phase
- Before committing code, review the "Testing Before Committing" checklist
- When unsure, search this document for guidance
- If this document conflicts with user instructions, ask for clarification

**The user may remind you to follow these instructions by saying:**

- "Follow the workflow"
- "Check the AI instructions"
- "Follow the process"

When you hear these phrases, **immediately re-read the relevant sections** of this document.

---

## Core Principles

### 1. 👥 **Always Follow the Process**

- Never skip steps
- Always create issues before branches
- Always test before committing
- Always use conventional commits
- Always follow the PR workflow

### 2. 👥 **Step-by-Step Development**

- Break down tasks into clear steps
- Present a plan before starting
- Ask clarifying questions upfront
- Execute one step at a time
- Test after each significant change

### 3. 👥 **Quality First**

- Run `go fmt ./...` BEFORE `go vet ./...`
- Fix all linting errors before committing
- Test in the browser before shipping
- No shortcuts, no "we'll fix it later"

---

## 🤖 AI Assistant: Non-Negotiable Rules

**These rules MUST be followed. No exceptions.**

### RULE 1: Never Code Without Approval

```
IF user requests a feature
THEN present a plan
THEN wait for approval (🛑 CHECKPOINT)
THEN create issue and branch
THEN implement

DO NOT start coding immediately
DO NOT skip the planning phase
```

### RULE 2: Never Commit Without Testing

```
IF implementation is complete
THEN run go fmt ./...
THEN run go vet ./...
THEN fix all errors
THEN ask developer to test (🛑 CHECKPOINT)
THEN wait for explicit approval
THEN commit

DO NOT commit without user testing
DO NOT commit with linting errors
DO NOT skip the format step
```

### RULE 3: Always Use Conventional Commits

```
IF committing code
THEN use format: type(scope): description
THEN include body with details
THEN reference issue number

DO NOT use generic commit messages
DO NOT commit without proper format
```

### RULE 4: Test After Every Change

```
IF you make a code change
THEN run go vet on that file
THEN check for errors immediately
THEN fix before moving on

DO NOT batch multiple changes before testing
DO NOT ignore linting warnings
```

### RULE 5: Follow the Order

```
The workflow phases MUST be followed in order:
1. Planning & Discussion (🛑 wait for approval)
2. Issue & Branch Creation
3. Implementation
4. User Testing & Verification (🛑 wait for approval)
5. Commit & Push
6. Release (when applicable)

DO NOT skip phases
DO NOT reorder phases
DO NOT combine phases without permission
```

---

## Development Workflow

### Phase 1: Planning & Discussion

**👥 When the developer presents a feature request:**

1. **Understand the requirement fully**
   - Ask clarifying questions
   - Confirm edge cases
   - Discuss user experience implications

2. **Present a plan**

   ```
   Here's my plan:
   1. [Step 1]
   2. [Step 2]
   3. [Step 3]

   Does this approach work for you?
   ```

3. **🛑 CHECKPOINT: Wait for approval** before proceeding
   - Do NOT create issues or branches yet
   - Do NOT start coding
   - WAIT for explicit approval from the developer

---

### Phase 2: Issue & Branch Creation

**👥 Always use GitHub CLI:**

```bash
# 1. Create issue (no labels initially)
gh issue create --title "Feature title" --body "Description with acceptance criteria"

# 2. Create branch from issue — ALWAYS use --name with a short name (≤ 45 chars)
gh issue develop <issue-number> --checkout --name "<issue-number>-short-summary"

# 3. Verify you're on the correct branch
git branch --show-current
```

**Branch naming rules:**

- Keep feature branch names concise and descriptive
- Format: `<issue-number>-<short-kebab-summary>`
- Use concise verbs/nouns (e.g., `193-doc-merge-strategy`, `205-fix-cart-totals`)
- Avoid filler words, articles, or unnecessary context

**Issue Template:**

```markdown
**Problem:**
[Describe the problem or need]

**Solution:**
[Describe the proposed solution]

**Acceptance Criteria:**

- [ ] Criterion 1
- [ ] Criterion 2
- [ ] Criterion 3
```

**Issue Labels:**

When creating multiple issues (e.g., from a project review), create and assign labels systematically:

```bash
# 1. Create priority labels
gh label create "priority:critical" --color "d73a4a" --description "Critical priority - must fix immediately"
gh label create "priority:high" --color "ff6b35" --description "High priority - fix soon"
gh label create "priority:medium" --color "fbca04" --description "Medium priority - fix when possible"
gh label create "priority:low" --color "0e8a16" --description "Low priority - nice to have"

# 2. Create type labels
gh label create "type:security" --color "d73a4a" --description "Security related"
gh label create "type:bug" --color "d73a4a" --description "Bug fix"
gh label create "type:feature" --color "0075ca" --description "New feature"
gh label create "type:enhancement" --color "a2eeef" --description "Enhancement to existing feature"
gh label create "type:docs" --color "0075ca" --description "Documentation"
gh label create "type:refactor" --color "5319e7" --description "Code refactoring"
gh label create "type:performance" --color "0e8a16" --description "Performance improvement"
gh label create "type:accessibility" --color "f9d0c4" --description "Accessibility improvement"

# 3. Create effort labels (Fibonacci scale)
gh label create "effort:1" --color "c2e0c6" --description "Very small effort (< 1 hour)"
gh label create "effort:2" --color "bfdadc" --description "Small effort (1-3 hours)"
gh label create "effort:3" --color "fef2c0" --description "Medium effort (3-8 hours)"
gh label create "effort:5" --color "fad8c7" --description "Large effort (1-2 days)"
gh label create "effort:8" --color "f9c5d1" --description "Very large effort (2+ days)"

# 4. Assign labels to issues
gh issue edit <issue-number> --add-label "priority:critical,type:security,effort:2"
```

**Label Guidelines:**

- **Priority**: Assign ONE priority label per issue (critical, high, medium, low)
- **Type**: Assign one or more type labels as appropriate
- **Effort**: Assign ONE effort label using Fibonacci scale (1, 2, 3, 5, 8)
  - effort:1 = Quick wins (< 1 hour)
  - effort:2 = Small tasks (1-3 hours)
  - effort:3 = Medium tasks (3-8 hours)
  - effort:5 = Large tasks (1-2 days)
  - effort:8 = Very large tasks (2+ days)

**When creating labels:**

- Use these exact colors for consistency across projects
- Create all labels at once before assigning to issues
- Batch assign labels using `&&` to chain commands efficiently

---

### Phase 3: Implementation

**👥 Step-by-step execution:**

1. **Read relevant files first**
   - Use `read_file` to understand context
   - Check existing patterns
   - Identify dependencies

2. **Make changes incrementally**
   - One logical change at a time
   - Explain what you're doing
   - Show the code changes

3. **Write tests (REQUIRED for new features)**
   - **Unit Tests**: Test individual functions, components, and utilities
   - **E2E Tests**: Test complete user workflows and critical paths
   - **Integration Tests**: Test how different parts work together
   - Write tests for complex business logic
   - Write tests for utility functions
   - Write tests for critical user flows
   - Write tests for bug fixes (regression tests)
   - Run `go test ./...` to verify all tests pass

4. **Test immediately after implementation**
   - Run linter: `go vet ./...`
   - Run tests: `go test ./...` (if tests exist)
   - Test CLI manually if applicable
   - Fix errors before moving on

5. **Format and lint before committing**
   ```bash
   # ALWAYS in this order:
   go fmt ./...
   go vet ./...
   go test ./...     # Run all tests
   ```

---

### Phase 4: User Testing & Verification

**👥 CRITICAL: Always verify with the developer before committing**

Before staging any changes:

1. **Ask the developer to test**
   - "Can you please test this in the browser?"
   - "Does everything work as expected?"
   - Wait for confirmation

2. **Verify all functionality**
   - New features work correctly
   - Existing features still work (no regressions)
   - UI looks good and is responsive
   - No console errors
   - No visual bugs

3. **🛑 CHECKPOINT: Only proceed after approval**
   - WAIT for developer to confirm it works
   - Fix ALL issues found during testing
   - Do NOT commit until you receive explicit approval
   - If there are issues, fix them and ask for testing again

**⚠️ NEVER COMMIT WITHOUT USER TESTING AND APPROVAL ⚠️**

**🤖 AI Assistant: Before moving to Phase 5, you MUST:**

- [ ] Have asked the developer to test
- [ ] Have received explicit approval
- [ ] Have fixed any issues that were found
- [ ] Have run `go fmt ./... && go vet ./...` successfully
- [ ] Have written unit tests for new features
- [ ] Have run `go test ./...` successfully

---

### Phase 5: Commit & Push

**👥 Conventional Commit Format:**

```bash
<type>(<scope>): <subject>

<body>

<footer>
```

**Types:**

- `feat`: New feature (bumps MINOR version)
- `fix`: Bug fix (bumps PATCH version)
- `feat!` or `fix!`: Breaking change (bumps MAJOR version)
- `docs`: Documentation only
- `chore`: Maintenance tasks
- `refactor`: Code refactoring
- `perf`: Performance improvements
- `test`: Adding/updating tests

**Example commits:**

```bash
git commit -m "feat(checkout): prevent access with empty cart

- Add empty cart check on mount
- Redirect to homepage if empty
- Return null to prevent flash

Closes #173"
```

**Push and create PR:**

```bash
git push origin <branch-name>
gh pr create --fill
```

**PR Merge Strategy:**

- Always use GitHub's **Rebase and merge** option for feature branches
- Never squash merge (we rely on individual conventional commits for releases)
- Avoid merge commits on dev/main to keep history linear

---

### Phase 6: Release Process

> **⚠️ CRITICAL: You MUST create a git tag before creating the PR to main!**
>
> This step:
>
> - Creates a version tag (e.g., v1.0.0)
> - Follows semantic versioning based on changes
> - Tags should follow conventional commit analysis
>
> **Without this step, the release will not be properly versioned!**

**👥 Release Checklist - Follow in Order:**

```bash
# ✅ Step 1: Switch to dev and pull latest
git checkout dev
git pull origin dev

# ✅ Step 2: Determine version based on commits
# Review recent commits to decide version bump:
# - feat: commits → MINOR bump (1.0.0 → 1.1.0)
# - fix: commits → PATCH bump (1.0.0 → 1.0.1)
# - Breaking changes → MAJOR bump (1.0.0 → 2.0.0)
git log --oneline origin/main..origin/dev

# ✅ Step 3: Update VERSION file (REQUIRED - this is what the binary reads!)
echo "1.0.0" > VERSION   # replace with actual new version
git add VERSION
git commit -m "chore: bump version to v1.0.0"
git push origin dev

# ✅ Step 4: Create version tag (REQUIRED - DO NOT SKIP!)
git tag -a v1.0.0 -m "Release v1.0.0"
# Examples:
# git tag -a v1.0.1 -m "Release v1.0.1"  # PATCH
# git tag -a v1.1.0 -m "Release v1.1.0"  # MINOR
# git tag -a v2.0.0 -m "Release v2.0.0"  # MAJOR

# ✅ Step 5: Push tags
git push origin v1.0.0

# ✅ Step 6: Create PR to main for production deployment
gh pr create --base main --head dev --fill

# ✅ Step 7: Wait for PR approval and merge
gh pr view <PR-number> --json state,mergedAt

# ✅ Step 8: IMMEDIATELY after merge - Sync dev with main
# ⚠️  REQUIRED — skipping this causes conflicts on the next release
git checkout dev
git pull origin dev
git merge origin/main
git push origin dev

# ✅ Step 8: Verify branches are synced (output should be empty)
git log --oneline origin/main..origin/dev
git log --oneline origin/dev..origin/main

# ✅ Step 9: Clean up merged feature branches
git branch -D <feature-branch-name>           # Delete local
git push origin --delete <feature-branch-name> # Delete remote

# ✅ Step 10: Update local main
git checkout main
git pull origin main

# ✅ Step 11: Return to dev for next work
git checkout dev
```

**Version Bumping:**

- `feat:` commits → MINOR bump (2.0.0 → 2.1.0)
- `fix:` commits → PATCH bump (2.0.0 → 2.0.1)
- `feat!:` or `fix!:` → MAJOR bump (2.0.0 → 3.0.0)
- `docs:`, `chore:`, etc. → PATCH bump

> **⚠️ CRITICAL: After every merge to main, you MUST sync dev back immediately!**
>
> ```bash
> git checkout dev && git pull origin dev
> git merge origin/main
> git push origin dev
> ```
>
> **Skipping this causes conflicts on the next release.**

**Critical: Always sync dev with main after release!**

- This prevents merge conflicts on future releases
- Keeps branch histories aligned
- Use `merge` not `reset --hard` to preserve history

---

### Phase 7: Hotfix Process (Production Emergencies)

**🧑‍💻 For critical bugs in production that can't wait for normal release cycle:**

```bash
# 1. Create hotfix branch from main
git checkout main
git pull origin main
git checkout -b hotfix/<issue-number>-description

# 2. Fix the bug
[make changes]
go fmt ./...
go vet ./...
[test thoroughly]

# 3. Commit with fix type
git commit -m "fix(scope): critical bug description

Fixes critical production issue.

Closes #<issue-number>"

# 4. Create PR to main
git push origin hotfix/<issue-number>-description
gh pr create --base main --head hotfix/<issue-number>-description --fill

# 5. After merge to main, backport to dev
git checkout dev
git pull origin dev
git merge origin/main
git push origin dev

# 6. Clean up hotfix branch
git branch -D hotfix/<issue-number>-description
git push origin --delete hotfix/<issue-number>-description

# 7. Verify sync (should be empty)
git log --oneline origin/main..origin/dev
git log --oneline origin/dev..origin/main

# 8. Update local main
git checkout main
git pull origin main
git checkout dev
```

**Hotfix Guidelines:**

- ⚠️ Only for critical production bugs
- 🚨 Must be tested thoroughly before merge
- 🔄 Always backport to dev after deploying
- 📝 Still create GitHub issue first
- 🏷️ Use `fix:` commit type (bumps PATCH)

**When to use hotfix:**

- Production is broken
- Security vulnerability
- Data loss risk
- Critical user-facing bug

**When NOT to use hotfix:**

- Minor bugs (use normal flow)
- New features (use normal flow)
- Non-critical improvements

---

## Branch Cleanup Checklist

### 🧑‍💻 After Feature PR Merge to Dev

**When a feature PR is merged into dev (normal development):**

✅ **Basic cleanup:**

```bash
# 1. Switch to dev and pull latest
git checkout dev
git pull origin dev

# 2. Delete the merged feature branch
git branch -D <feature-branch-name>              # Delete local
git push origin --delete <feature-branch-name>    # Delete remote
```

**That's it!** No need for complex sync operations since dev is already up to date.

---

### 👥 After Release or Hotfix Merge to Main

**When code is released to production (merged to main):**

✅ **Sync dev with main:**

```bash
git checkout dev
git pull origin dev
git merge origin/main
git push origin dev
```

✅ **Verify sync (both should be empty):**

```bash
git log --oneline origin/main..origin/dev
git log --oneline origin/dev..origin/main
```

✅ **Delete merged feature branches:**

```bash
# Local
git branch -D <branch-name>

# Remote
git push origin --delete <branch-name>

# Or use fetch --prune to clean up stale remote refs
git fetch origin --prune
```

✅ **Update local main:**

```bash
git checkout main
git pull origin main
git checkout dev
```

✅ **Verify clean state:**

```bash
git branch --all
# Should only see: dev, main, origin/dev, origin/main
```

**Why this matters:**

- Prevents merge conflicts on future releases
- Keeps repository clean and organized
- Ensures dev and main histories stay aligned
- Makes it easy to see what's in progress vs deployed

---

## Code Quality Standards

### 1. 👥 **Linting & Formatting**

**CRITICAL ORDER:**

```bash
# 1. Format FIRST
go fmt ./...

# 2. Lint SECOND
go vet ./...
```

**Never commit with linting errors.**

- **Respond immediately to IDE lint feedback**: resolve errors/warnings before making additional edits

### 2. 👥 **Testing Philosophy**

**CRITICAL: Multi-layered testing is mandatory**

**Always test before committing:**

1. **Write automated tests where appropriate**
2. **Run all tests** (`go test ./...`)
3. **Lint and format** (`go fmt ./... && go vet ./...`)
4. **Manual CLI testing** (test all functionality)
5. **Ask developer to verify** (wait for approval)
6. **Only commit after approval**

**Write automated tests for:**

- ✅ Complex business logic
- ✅ Utility functions
- ✅ Critical user flows
- ✅ Bug fixes (regression tests)
- ✅ API integrations
- ✅ Data transformations
- ✅ **ALL NEW FEATURES** - Unit tests required

**Skip automated tests for:**

- ⏭️ Simple functions with no logic
- ⏭️ Trivial changes
- ⏭️ Proof-of-concept code

**Testing commands:**

```bash
# Run all tests
go test ./...

# Run tests with coverage
go test -cover ./...

# Run tests with verbose output
go test -v ./...

# Run specific test
go test -run TestFunctionName ./...
```

**Test types required for new features:**

- **Unit Tests**: Test individual functions and utilities in isolation
- **Integration Tests**: Test how different parts of the system work together

**Test location:**

- Unit tests go in `*_test.go` files next to source files
- Mirror the source structure
- Use `_test.go` suffix for test files
- Use `TestFunctionName` naming convention

**Manual testing checklist:**

- [ ] Feature works as expected
- [ ] No errors in CLI output
- [ ] Edge cases handled
- [ ] Existing features still work (no regressions)

**User testing (Developer):**

- Always ask: "Can you please test this in the CLI?"
- Wait for confirmation before committing
- Fix any issues found
- Never skip this step

### 3. 👥 **Code Style**

**General Guidelines:**

- Follow the existing code style and patterns in the codebase
- Use `gofmt` for consistent formatting
- Follow existing patterns for imports and file organization
- Use godoc comments for documentation

**Code Organization:**

- Keep functions small and focused
- Use interfaces for struct types
- Follow Go naming conventions (PascalCase for exported, camelCase for unexported)
- Use meaningful variable names
- Avoid package-level state when possible

**File naming:**

- Use lowercase with underscores for file names (e.g., `config.go`, `issues.go`)
- Test files should end with `_test.go` (e.g., `config_test.go`)
- Keep related files in the same package

### 4. 👥 **Import Order**

**Follow Go's standard import order:**

```go
// 1. Standard library
import (
    "fmt"
    "os"
)

// 2. Third-party libraries
import (
    "github.com/user/package"
)

// 3. Internal packages
import (
    "internal/github"
)
```

### 5. 👥 **Error Handling**

**Error handling:**

- Always check and handle errors
- Use `fmt.Errorf` for wrapping errors with context
- Return errors from functions rather than panicking
- Use descriptive error messages

```go
if err != nil {
    return fmt.Errorf("failed to process: %w", err)
}
```

**Development logging:**

```go
// Use log package for debugging
log.Printf("[Component] %s", message)
```

---

## Command References

### Git Commands Reference

| Command                                  | Description                            | When to Use                        |
| ---------------------------------------- | -------------------------------------- | ---------------------------------- |
| `git checkout <branch>`                  | Switch to an existing branch           | When moving between branches       |
| `git checkout -b <branch>`               | Create and switch to new branch        | When creating hotfix branches      |
| `git branch --show-current`              | Show current branch name               | To verify you're on correct branch |
| `git branch -D <branch>`                 | Delete local branch (force)            | After branch merged and cleaned up |
| `git status`                             | Show working tree status               | Check what's staged/unstaged       |
| `git add <files>`                        | Stage files for commit                 | Before committing changes          |
| `git commit -m "message"`                | Commit staged changes                  | After testing and approval         |
| `git push origin <branch>`               | Push branch to remote                  | After committing                   |
| `git push --follow-tags origin dev`      | Push commits and tags                  | During release process             |
| `git push origin --delete <branch>`      | Delete remote branch                   | Clean up merged feature branches   |
| `git pull origin <branch>`               | Pull latest from remote                | Before starting work or merging    |
| `git merge origin/main`                  | Merge main into current branch         | Sync dev with main after release   |
| `git log --oneline <branch1>..<branch2>` | Show commits in branch2 not in branch1 | Verify branch sync                 |
| `git fetch origin --prune`               | Remove stale remote tracking refs      | Clean up deleted remote branches   |

### GitHub CLI Commands Reference

| Command                                                     | Description                        | When to Use                      |
| ----------------------------------------------------------- | ---------------------------------- | -------------------------------- |
| `gh issue create --title "X" --body "Y"`                    | Create new issue                   | Start of every feature/fix       |
| `gh issue develop <number> --checkout`                      | Create branch from issue           | After creating issue             |
| `gh issue list`                                             | List all issues                    | Check open issues                |
| `gh issue edit <number> --add-label "X,Y"`                  | Add labels to issue                | Categorize and prioritize issues |
| `gh label create "name" --color "hex" --description "text"` | Create new label                   | Set up label system              |
| `gh pr create --fill`                                       | Create PR with auto-filled details | After pushing feature branch     |
| `gh pr create --base main --head dev --fill`                | Create release PR                  | Release to production            |
| `gh pr list`                                                | List all PRs                       | Check PR status                  |
| `gh pr view <number>`                                       | View PR details                    | Check PR status/reviews          |
| `gh pr view <number> --json state,mergedAt`                 | Get PR state JSON                  | Verify PR merged                 |

### Go Commands Reference

| Command                                 | Description              | When to Use                    |
| --------------------------------------- | ------------------------ | ------------------------------ |
| `go build .`                            | Build binary             | Before distributing            |
| `go fmt ./...`                          | Format code              | **FIRST** before vetting       |
| `go vet ./...`                          | Run static analysis      | **SECOND** after formatting    |
| `go test ./...`                         | Run all tests            | Verify code works              |
| `go test -v ./...`                      | Run tests with verbose   | During test development        |
| `go test -cover ./...`                  | Generate coverage report | Check test coverage            |
| `go test -run TestFunctionName ./...`   | Run specific test        | Debug specific test            |
| `go run .`                              | Run application          | During development             |
| `git tag -a v1.0.0 -m "Release v1.0.0"` | Create version tag       | **REQUIRED** before PR to main |

### Quick Copy-Paste Commands

**Start a new feature:**

```bash
gh issue create --title "Feature name" --body "Description with acceptance criteria"
gh issue develop <issue-number> --checkout
git branch --show-current  # Verify branch
```

**Development cycle:**

```bash
# Make changes, then:
go fmt ./... && go vet ./... && go test ./...
# Test in CLI
# Get approval
```

**Commit and push:**

```bash
git add .
git commit -m "feat(scope): description

- Detail 1
- Detail 2
- Detail 3

Closes #<issue-number>"

git push origin <branch-name>
gh pr create --fill
```

**Release to production:**

```bash
git checkout dev && git pull origin dev
# Review commits to determine version
git log --oneline origin/main..origin/dev
# Create version tag
git tag -a v1.0.0 -m "Release v1.0.0"
git push origin v1.0.0
gh pr create --base main --head dev --fill
# After merge:
git checkout dev && git pull origin dev && git merge origin/main && git push origin dev
git checkout main && git pull origin main && git checkout dev
```

**Create all labels at once:**

```bash
# Priority labels
gh label create "priority:critical" --color "d73a4a" --description "Critical priority - must fix immediately" && \
gh label create "priority:high" --color "ff6b35" --description "High priority - fix soon" && \
gh label create "priority:medium" --color "fbca04" --description "Medium priority - fix when possible" && \
gh label create "priority:low" --color "0e8a16" --description "Low priority - nice to have" && \

# Type labels
gh label create "type:security" --color "d73a4a" --description "Security related" && \
gh label create "type:bug" --color "d73a4a" --description "Bug fix" && \
gh label create "type:feature" --color "0075ca" --description "New feature" && \
gh label create "type:enhancement" --color "a2eeef" --description "Enhancement to existing feature" && \
gh label create "type:docs" --color "0075ca" --description "Documentation" && \
gh label create "type:refactor" --color "5319e7" --description "Code refactoring" && \
gh label create "type:performance" --color "0e8a16" --description "Performance improvement" && \
gh label create "type:accessibility" --color "f9d0c4" --description "Accessibility improvement" && \

# Effort labels
gh label create "effort:1" --color "c2e0c6" --description "Very small effort (< 1 hour)" && \
gh label create "effort:2" --color "bfdadc" --description "Small effort (1-3 hours)" && \
gh label create "effort:3" --color "fef2c0" --description "Medium effort (3-8 hours)" && \
gh label create "effort:5" --color "fad8c7" --description "Large effort (1-2 days)" && \
gh label create "effort:8" --color "f9c5d1" --description "Very large effort (2+ days)"
```

---

## Decision Trees

### Decision Tree: Hotfix vs Normal Flow

```
Is production currently broken or at immediate risk?
│
├─ YES → Is it critical? (security, data loss, broken core feature)
│   │
│   ├─ YES → Use HOTFIX flow
│   │         1. Branch from main
│   │         2. Fix immediately
│   │         3. PR to main
│   │         4. Backport to dev
│   │
│   └─ NO → Use NORMAL flow
│             (It can wait for next release)
│
└─ NO → Use NORMAL flow
          1. Branch from issue
          2. Follow full workflow
          3. Release when ready
```

**Examples:**

- **HOTFIX**: Payment processing broken, user data exposed, critical security flaw
- **NORMAL**: Minor UI bug, feature enhancement, documentation update, non-critical fixes

### Decision Tree: Commit Types

```
What are you changing?
│
├─ Adding NEW functionality?
│   ├─ Breaking existing APIs/behavior? → feat!(scope): description
│   └─ Not breaking anything? → feat(scope): description
│
├─ Fixing a BUG?
│   ├─ Breaking change? → fix!(scope): description
│   └─ Regular fix? → fix(scope): description
│
├─ Changing code structure without changing behavior?
│   └─ refactor(scope): description
│
├─ Improving PERFORMANCE?
│   └─ perf(scope): description
│
├─ Updating DOCUMENTATION?
│   └─ docs(scope): description
│
├─ Adding/updating TESTS?
│   └─ test(scope): description
│
└─ Maintenance tasks (dependencies, config, build)?
    └─ chore(scope): description
```

**Quick Rules:**

- New feature = `feat:`
- Bug fix = `fix:`
- Breaking change = add `!` (e.g., `feat!:` or `fix!:`)
- Everything else = specific type (`docs:`, `chore:`, `refactor:`, `perf:`, `test:`)

### Decision Tree: When to Write Tests

```
What are you working on?
│
├─ Complex business logic or calculations?
│   └─ YES → Write tests ✅
│
├─ Utility/helper functions?
│   └─ YES → Write tests ✅
│
├─ Critical user flows (auth, payments, checkout)?
│   └─ YES → Write tests ✅
│
├─ Bug fix?
│   └─ YES → Write regression test ✅
│
├─ API integration or data transformation?
│   └─ YES → Write tests ✅
│
├─ Simple function with no logic?
│   └─ NO → Skip tests, manual CLI test ⏭️
│
├─ Trivial changes?
│   └─ NO → Skip tests for now ⏭️
│
└─ Proof-of-concept or experimental code?
    └─ NO → Skip tests for now ⏭️
```

**Always Required:**

- Run `go vet ./...` and `go fmt ./...`
- Manual CLI testing
- Get developer approval

### Decision Tree: Label Selection

```
PRIORITY (pick ONE):
├─ Will cause data loss or security breach? → priority:critical
├─ Blocking users from core functionality? → priority:high
├─ Annoying but has workaround? → priority:medium
└─ Nice to have improvement? → priority:low

TYPE (pick one or more):
├─ Security issue? → type:security
├─ Something broken? → type:bug
├─ Brand new feature? → type:feature
├─ Improving existing feature? → type:enhancement
├─ Documentation? → type:docs
├─ Restructuring code? → type:refactor
├─ Speed improvement? → type:performance
└─ Accessibility? → type:accessibility

EFFORT (pick ONE - Fibonacci scale):
├─ Less than 1 hour? → effort:1
├─ 1-3 hours? → effort:2
├─ 3-8 hours? → effort:3
├─ 1-2 days? → effort:5
└─ 2+ days? → effort:8
```

**Examples:**

- Security vulnerability: `priority:critical, type:security, effort:2`
- New login form: `priority:high, type:feature, effort:5`
- Fix typo in docs: `priority:low, type:docs, effort:1`
- Refactor for performance: `priority:medium, type:refactor, type:performance, effort:3`

---

## Troubleshooting Guide

### My Commit Was Rejected

**Error: Commit message doesn't follow conventional format**

```bash
# Check your commit message format
# It should be: type(scope): description
# Examples:
feat(auth): add login form
fix(cart): calculate totals correctly
docs(readme): update installation steps
```

**Fix:**

```bash
# Amend your last commit message
git commit --amend -m "feat(scope): proper description

- Detail 1
- Detail 2

Closes #123"
```

**Error: Linting errors present**

```bash
# Run format and lint
go fmt ./...
go vet ./...

# Fix all errors, then try committing again
```

**Error: Pre-commit hook failed**

```bash
# Check what failed (usually linting or formatting)
# Fix the issues
go fmt ./... && go vet ./...

# Stage the fixes
git add .

# Commit again
git commit -m "your message"
```

### Tests Are Failing

**Step 1: Read the error output**

```bash
go test ./...

# Read the failure messages carefully
# Identify which test is failing and why
```

**Step 2: Common test failures**

| Error                  | Likely Cause                 | Solution                      |
| ---------------------- | ---------------------------- | ----------------------------- |
| `undefined: X`         | Missing import or wrong path | Check import paths            |
| `syntax error`         | Syntax error                 | Check for typos               |
| `expected X but got Y` | Logic error                  | Review your implementation    |
| `panic: runtime error` | Nil pointer or bounds check  | Add nil checks, validate data |

**Step 3: Debug the test**

```bash
# Run specific test
go test -run TestFunctionName ./...

# Run tests with verbose output
go test -v ./...
```

**Step 4: Fix and verify**

```bash
# Make fixes
# Run tests again
go test ./...

# Only proceed when all tests pass
```

### Merge Conflicts

**When rebasing or merging:**

```bash
# Step 1: See which files have conflicts
git status

# Step 2: Open conflicted files
# Look for conflict markers:
# <<<<<<< HEAD
# Your changes
# =======
# Their changes
# >>>>>>> branch-name

# Step 3: Manually resolve
# - Decide which changes to keep
# - Remove conflict markers
# - Ensure code still works

# Step 4: Stage resolved files
git add <resolved-files>

# Step 5: Continue merge/rebase
git merge --continue
# or
git rebase --continue

# Step 6: Verify everything works
go fmt ./... && go vet ./... && go test ./...
```

**Prevention tips:**

- Sync dev with main regularly
- Keep feature branches short-lived
- Communicate with team about overlapping work

### Wrong Branch

**Started work on wrong branch?**

```bash
# Step 1: Stash your changes
git stash

# Step 2: Switch to correct branch
git checkout <correct-branch>

# Step 3: Apply stashed changes
git stash pop

# Step 4: Continue working
```

**Committed to wrong branch?**

```bash
# Step 1: Create correct branch from current position
git branch <correct-branch-name>

# Step 2: Switch to correct branch
git checkout <correct-branch-name>

# Step 3: Go back to wrong branch
git checkout <wrong-branch>

# Step 4: Reset wrong branch (remove commits)
git reset --hard HEAD~1  # Remove last commit
# Or: git reset --hard origin/<wrong-branch>  # Reset to remote

# Step 5: Switch back to correct branch
git checkout <correct-branch-name>

# Step 6: Continue working
```

### Common Linting Errors

**Unused variables:**

```javascript
// Error: 'x' is assigned a value but never used
const x = getValue();

// Solution 1: Use it
const x = getValue();
console.log(x);

// Solution 2: Prefix with underscore (intentionally unused)
const _x = getValue();

// Solution 3: Remove it
// Just delete the line
```

**Unused imports:**

```javascript
// Error: 'useState' is defined but never used
import { useState, useEffect } from "react";

// Solution: Remove unused import
import { useEffect } from "react";
```

**Case block declarations:**

```javascript
// Error: Unexpected lexical declaration in case block
switch (action) {
  case "ADD":
    const newValue = 1; // ❌ Error
    break;
}

// Solution: Wrap in braces
switch (action) {
  case "ADD": {
    const newValue = 1; // ✅ Works
    break;
  }
}
```

**Console statements:**

```javascript
// Error: Unexpected console statement
console.log("debug");

// Solution: Use eslint-disable comment
// eslint-disable-next-line no-console
console.log("debug");

// Or remove it before committing
```

---

## 🧑‍💻 For New Team Members

### Welcome to the Team!

This section helps you get started with our development workflow. Follow these steps to set up your environment and complete your first feature.

### First-Time Setup Checklist

**Required Tools:**

- [ ] **Go** (1.21 or higher)

  ```bash
  go version  # Check if installed
  ```

  - Download: https://go.dev/dl/

- [ ] **Git**

  ```bash
  git --version  # Check if installed
  ```

  - Download: https://git-scm.com/

- [ ] **GitHub CLI**

  ```bash
  gh --version  # Check if installed
  ```

  - Install: https://cli.github.com/
  - Authenticate: `gh auth login`

- [ ] **Code Editor** (VS Code recommended)
  - Download: https://code.visualstudio.com/
  - Install extensions:
    - Go
    - GitLens

**Project Setup:**

- [ ] Clone the repository

  ```bash
  git clone <repository-url>
  cd <project-name>
  ```

- [ ] Install dependencies

  ```bash
  go mod download
  ```

- [ ] Set up environment variables

  ```bash
  cp .env.example .env
  # Edit .env with your values
  ```

- [ ] Verify setup works

  ```bash
  go build .    # Build binary
  go test ./... # Run tests
  go vet ./...  # Run linter
  ```

- [ ] Configure Git
  ```bash
  git config user.name "Your Name"
  git config user.email "your.email@example.com"
  ```

### Required Tools Installation

**macOS:**

```bash
# Install Homebrew (if not installed)
/bin/bash -c "$(curl -fsSL https://raw.githubusercontent.com/Homebrew/install/HEAD/install.sh)"

# Install Go
brew install go

# Install GitHub CLI
brew install gh

# Authenticate with GitHub
gh auth login
```

**Windows:**

```bash
# Install via Chocolatey (package manager)
choco install golang
choco install gh

# Or download installers from:
# Go: https://go.dev/dl/
# GitHub CLI: https://cli.github.com/
```

**Linux (Ubuntu/Debian):**

```bash
# Install Go
sudo apt update
sudo apt install golang-go

# Install GitHub CLI
curl -fsSL https://cli.github.com/packages/githubcli-archive-keyring.gpg | sudo dd of=/usr/share/keyrings/githubcli-archive-keyring.gpg
echo "deb [arch=$(dpkg --print-architecture) signed-by=/usr/share/keyrings/githubcli-archive-keyring.gpg] https://cli.github.com/packages stable main" | sudo tee /etc/apt/sources.list.d/github-cli.list > /dev/null
sudo apt update
sudo apt install gh

# Authenticate with GitHub
gh auth login
```

### First Feature Walkthrough

**Let's create a simple feature together! We'll add a "Hello World" function.**

**Step 1: Create an issue**

```bash
gh issue create \
  --title "Add Hello World function" \
  --body "Create a simple Hello World function for testing workflow

**Acceptance Criteria:**
- [ ] Function returns 'Hello World'
- [ ] Function is exported properly
- [ ] No errors when running"

# Note the issue number (e.g., #42)
```

**Step 2: Create a branch**

```bash
gh issue develop 42 --checkout
git branch --show-current  # Should show something like "42-add-hello-world-function"
```

**Step 3: Create the function**

Create file `hello.go`:

```go
package main

import "fmt"

func HelloWorld() string {
    return "Hello World"
}

func main() {
    fmt.Println(HelloWorld())
}
```

**Step 4: Test your changes**

```bash
# Format and lint
go fmt ./...
go vet ./...

# If tests exist, run them
go test ./...

# Build and run
go build .
./hello
```

**Step 5: Commit your changes**

```bash
git add hello.go

git commit -m "feat(hello): add Hello World function

- Created HelloWorld function
- Added main function to test output

Closes #42"

git push origin 42-add-hello-world-function
gh pr create --fill
```

**Step 6: Wait for review**

- Your PR will be reviewed
- Make any requested changes
- Once approved, it will be merged

**Congratulations! You've completed your first feature!** 🎉

### Common Gotchas for Newcomers

**1. Forgetting to format before linting**

```bash
# ❌ Wrong order
go vet ./...
go fmt ./...

# ✅ Correct order
go fmt ./...  # FIRST
go vet ./...  # SECOND
```

**2. Committing without testing**

```bash
# ❌ Don't do this
git add .
git commit -m "changes"
git push

# ✅ Always test first
go fmt ./... && go vet ./... && go test ./...
# Test in CLI
# Get approval
# Then commit
```

**3. Wrong commit message format**

```bash
# ❌ Wrong
git commit -m "fixed the bug"
git commit -m "Added feature"

# ✅ Correct
git commit -m "fix(scope): fixed the bug"
git commit -m "feat(scope): add feature"
```

**4. Working on main or dev directly**

```bash
# ❌ Don't do this
git checkout main
# Make changes directly

# ✅ Always use feature branches
gh issue create --title "Feature"
gh issue develop <number> --checkout
```

**5. Skipping issue creation**

```bash
# ❌ Don't do this
git checkout -b my-feature
# Work on feature

# ✅ Always create issue first
gh issue create --title "Feature"
gh issue develop <number> --checkout
```

**6. Not syncing before starting work**

```bash
# ❌ Don't do this
# Start work on old code

# ✅ Always pull latest
git checkout dev
git pull origin dev
# Then create branch
```

**7. Committing unrelated changes**

```bash
# ❌ Don't do this
git add .  # Adds everything including unrelated files

# ✅ Stage only related files
git add src/components/Feature.jsx
git add src/utils/helper.js
```

**8. Ignoring linting errors**

```bash
# ❌ Don't do this
go vet ./...
# See errors
# Commit anyway

# ✅ Fix all errors first
go vet ./...
# Fix errors
go vet ./...  # Run again until clean
# Then commit
```

**Quick Tips:**

- 💡 Read error messages carefully - they usually tell you what's wrong
- 💡 When in doubt, ask! It's better to ask than to make mistakes
- 💡 Follow the visual workflow diagram - it shows you exactly what to do
- 💡 Use this document as a reference - bookmark it!
- 💡 Watch how others do it - review merged PRs for examples

---

## Reference Tables

### Labels Reference Table

| Label                                            | Color        | Description                              | When to Use                                          | Example                      |
| ------------------------------------------------ | ------------ | ---------------------------------------- | ---------------------------------------------------- | ---------------------------- |
| **Priority Labels (Choose ONE)**                 |
| `priority:critical`                              | 🔴 `#d73a4a` | Critical priority - must fix immediately | Production broken, security vulnerability, data loss | Payment processing down      |
| `priority:high`                                  | 🟠 `#ff6b35` | High priority - fix soon                 | Blocking core functionality, major bugs              | Login fails for all users    |
| `priority:medium`                                | 🟡 `#fbca04` | Medium priority - fix when possible      | Annoying but has workaround                          | Slow page load               |
| `priority:low`                                   | 🟢 `#0e8a16` | Low priority - nice to have              | Minor improvements, nice-to-haves                    | Update button color          |
| **Type Labels (Choose one or more)**             |
| `type:security`                                  | 🔴 `#d73a4a` | Security related                         | Security vulnerabilities, auth issues                | SQL injection vulnerability  |
| `type:bug`                                       | 🔴 `#d73a4a` | Bug fix                                  | Something is broken                                  | Cart total calculates wrong  |
| `type:feature`                                   | 🔵 `#0075ca` | New feature                              | Brand new functionality                              | Add wish list feature        |
| `type:enhancement`                               | 🔵 `#a2eeef` | Enhancement to existing feature          | Improving what exists                                | Add filters to search        |
| `type:docs`                                      | 🔵 `#0075ca` | Documentation                            | README, comments, guides                             | Update installation docs     |
| `type:refactor`                                  | 🟣 `#5319e7` | Code refactoring                         | Restructure without changing behavior                | Extract utility functions    |
| `type:performance`                               | 🟢 `#0e8a16` | Performance improvement                  | Speed, optimization                                  | Reduce bundle size           |
| `type:accessibility`                             | 🟤 `#f9d0c4` | Accessibility improvement                | A11y compliance                                      | Add ARIA labels              |
| **Effort Labels (Choose ONE - Fibonacci scale)** |
| `effort:1`                                       | 🟢 `#c2e0c6` | Very small effort (< 1 hour)             | Quick fixes, typos                                   | Fix typo in button text      |
| `effort:2`                                       | 🔵 `#bfdadc` | Small effort (1-3 hours)                 | Small features, simple bugs                          | Add validation to form field |
| `effort:3`                                       | 🟡 `#fef2c0` | Medium effort (3-8 hours)                | Medium features, complex bugs                        | Create new dashboard page    |
| `effort:5`                                       | 🟠 `#fad8c7` | Large effort (1-2 days)                  | Large features, major refactors                      | Implement authentication     |
| `effort:8`                                       | 🔴 `#f9c5d1` | Very large effort (2+ days)              | Very large features, architecture changes            | Rebuild entire checkout flow |

**Usage Example:**

```bash
# Critical security bug, small effort
gh issue edit 45 --add-label "priority:critical,type:security,effort:2"

# New feature, medium priority, large effort
gh issue edit 46 --add-label "priority:medium,type:feature,effort:5"

# Performance refactor, low priority, medium effort
gh issue edit 47 --add-label "priority:low,type:refactor,type:performance,effort:3"
```

### Commit Types Comparison Table

| Type        | When to Use                          | Version Bump          | Example                                       | Breaking Change? |
| ----------- | ------------------------------------ | --------------------- | --------------------------------------------- | ---------------- |
| `feat:`     | New feature added                    | MINOR (2.0.0 → 2.1.0) | `feat(auth): add password reset`              | No               |
| `feat!:`    | New feature with breaking change     | MAJOR (2.0.0 → 3.0.0) | `feat!(api)!: change user endpoint structure` | Yes              |
| `fix:`      | Bug fix                              | PATCH (2.0.0 → 2.0.1) | `fix(cart): correct total calculation`        | No               |
| `fix!:`     | Bug fix with breaking change         | MAJOR (2.0.0 → 3.0.0) | `fix!(auth)!: change token format`            | Yes              |
| `docs:`     | Documentation only                   | PATCH (2.0.0 → 2.0.1) | `docs(readme): update installation steps`     | No               |
| `chore:`    | Maintenance, dependencies            | PATCH (2.0.0 → 2.0.1) | `chore(deps): update react to v18`            | No               |
| `refactor:` | Code restructure, no behavior change | PATCH (2.0.0 → 2.0.1) | `refactor(utils): extract validation logic`   | No               |
| `perf:`     | Performance improvement              | PATCH (2.0.0 → 2.0.1) | `perf(images): lazy load product images`      | No               |
| `test:`     | Adding or updating tests             | PATCH (2.0.0 → 2.0.1) | `test(checkout): add cart validation tests`   | No               |

**Breaking Change Indicator:**

- Add `!` after type: `feat!:` or `fix!:`
- Always bumps MAJOR version
- Document what breaks in commit body
- Include migration guide if needed

**Commit Body Guidelines:**

```bash
type(scope): short description (50 chars max)
# Blank line
- Bullet point detail 1
- Bullet point detail 2
- Bullet point detail 3
# Blank line
Closes #issue-number
```

### Version Bump Matrix

| Commit Type | No Breaking Change        | With Breaking Change (`!`)  |
| ----------- | ------------------------- | --------------------------- |
| `feat:`     | 2.0.0 → **2.1.0** (MINOR) | 2.0.0 → **3.0.0** (MAJOR)   |
| `fix:`      | 2.0.0 → **2.0.1** (PATCH) | 2.0.0 → **3.0.0** (MAJOR)   |
| `docs:`     | 2.0.0 → **2.0.1** (PATCH) | N/A (docs can't break)      |
| `chore:`    | 2.0.0 → **2.0.1** (PATCH) | N/A (rarely breaking)       |
| `refactor:` | 2.0.0 → **2.0.1** (PATCH) | 2.0.0 → **3.0.0** (MAJOR)   |
| `perf:`     | 2.0.0 → **2.0.1** (PATCH) | 2.0.0 → **3.0.0** (MAJOR)   |
| `test:`     | 2.0.0 → **2.0.1** (PATCH) | N/A (tests can't break API) |

**Understanding Semantic Versioning:**

```
MAJOR.MINOR.PATCH
  │     │     │
  │     │     └─── Bug fixes, no new features (fix:, chore:, docs:)
  │     └─────────── New features, backwards compatible (feat:)
  └───────────────── Breaking changes (feat!:, fix!:)

Examples:
2.0.0 → 2.0.1  = Bug fix (fix:)
2.0.1 → 2.1.0  = New feature (feat:)
2.1.0 → 3.0.0  = Breaking change (feat!: or fix!:)
```

**Multiple Commits:**

- If dev branch has both `feat:` and `fix:`, the highest wins (MINOR)
- If dev branch has any `feat!:` or `fix!:`, MAJOR wins
- Review commits with `git log --oneline origin/main..origin/dev` to determine version

**Manual Override:**

```bash
# Review commits to determine version
git log --oneline origin/main..origin/dev
# Create appropriate tag
git tag -a v1.0.0 -m "Release v1.0.0"  # Adjust version as needed
```

---

## Communication Style

### 1. 👥 **Be Direct & Concise**

- No fluff or unnecessary pleasantries
- Get straight to the point
- Use bullet points and clear headings

### 2. 👥 **Show, Don't Just Tell**

- Show code changes
- Provide command examples
- Include expected outcomes

### 3. 👥 **Acknowledge Mistakes Quickly**

- If something breaks, acknowledge it immediately
- Explain what went wrong
- Fix it right away

### 4. 👥 **Use Emojis Sparingly**

- ✅ for success
- ❌ for errors
- 🎯 for goals achieved
- 🚀 for deployments
- Don't overuse

### 5. 👥 **Progress Updates**

```markdown
## What We Did:

1. ✅ Created issue #123
2. ✅ Implemented feature
3. ✅ Tested and fixed bugs
4. ✅ Created PR #124

**Next steps:** Review and merge PR
```

---

## Common Patterns

### 1. 👥 **Testing Before Committing**

**CRITICAL: Multi-layered testing approach**

```bash
# 1. Write tests where appropriate
go test ./...

# 2. Vet the specific file
go vet ./path/to/file.go

# 3. Format and vet everything
go fmt ./...
go vet ./...

# 4. Run all tests
go test ./...

# 5. Manual CLI testing
# - Run the application
# - Test all functionality
# - Check for errors
# - Test edge cases

# 6. Ask developer to verify
# "Can you please test this? Does everything work as expected?"

# 7. Only after approval, commit
git add <files>
git commit -m "type(scope): message"
```

**Never skip user testing. Always wait for developer's approval before committing.**

### 2. 👥 **Fixing Linting Errors**

**Unused variables:**

```javascript
// Prefix with underscore
const _unusedVar = something;

// Or remove if truly unused
```

**Unused imports:**

```javascript
// Just remove them
```

**Case blocks:**

```javascript
// Wrap in braces
case 'action': {
  const variable = value;
  break;
}
```

### 3. 👥 **When Things Break**

1. **Read the error message carefully**
2. **Check the browser console**
3. **Verify variable names match context exports**
4. **Test the fix immediately**
5. **Don't commit until it works**

---

## Project-Specific Notes

> **🧑‍💻 Template: Customize this section for your project**

This section should be customized for each project. Include:

### 1. **State Management**

- Document your state management approach (Redux, Context API, Zustand, etc.)
- List key stores/contexts and their purpose
- Note any important patterns or gotchas

### 2. **Backend/Database**

- Database technology (PostgreSQL, MongoDB, Firebase, etc.)
- API architecture (REST, GraphQL, tRPC, etc.)
- Environment configuration
- Key environment variables

### 3. **Third-Party Integrations**

- Payment processors
- Authentication providers
- Analytics services
- Error tracking (Sentry, etc.)
- Other APIs and services

### 4. **Serverless Functions / API Routes**

- Location of API routes/functions
- Key endpoints and their purpose
- Authentication requirements
- Rate limiting considerations

### 5. **Styling Approach**

- CSS framework or methodology (Tailwind, CSS Modules, CSS-in-JS, etc.)
- Theme configuration
- Responsive design approach
- Icon library

### 6. **Testing**

- Testing framework (Jest, Vitest, Playwright, etc.)
- Test location and naming conventions
- Mocking strategies
- Coverage requirements

### 7. **Build & Deployment**

- Build tool (Vite, Webpack, Next.js, etc.)
- Deployment platform (Vercel, Netlify, AWS, etc.)
- Branch/environment mapping
- CI/CD pipeline
- Environment variables management

### 8. **Import Paths**

- Absolute vs relative imports
- Path aliases configured in tsconfig/jsconfig
- Import organization standards

---

## Workflow Checklist

**👥 For every feature:**

- [ ] Understand the requirement
- [ ] Present a plan
- [ ] Get approval
- [ ] Create GitHub issue
- [ ] Create branch from issue
- [ ] Implement step-by-step
- [ ] Write tests where appropriate (`go test ./...`)
- [ ] Test after each change
- [ ] Format code (`go fmt ./...`)
- [ ] Lint code (`go vet ./...`)
- [ ] Run tests (`go test ./...`)
- [ ] Fix all errors
- [ ] Test in CLI thoroughly
- [ ] **Ask developer to verify and test**
- [ ] **Wait for developer's approval**
- [ ] Fix any issues found during testing
- [ ] Commit with conventional commit message (only after approval)
- [ ] Push branch
- [ ] Create PR
- [ ] Wait for merge
- [ ] Create release tag (on dev)
- [ ] Push with tags
- [ ] Create PR to main

**Never skip steps. Never commit without user testing and approval. This process works flawlessly.**

---

## Emergency Procedures

### If Linting Fails

1. Read the error messages
2. Fix one error at a time
3. Run `go vet ./...` after each fix
4. Don't commit until clean

### If Build Fails

1. Check the error in terminal
2. Verify imports are correct
3. Check for typos in variable names
4. Run `go build .` to test

### If Tests Fail

1. Read the test output
2. Fix the code, not the test
3. Ensure test expectations are correct
4. Run tests again

---

## Success Metrics

**👥 You're doing it right when:**

- ✅ No linting errors
- ✅ No errors in CLI output
- ✅ Features work as expected
- ✅ Conventional commits used
- ✅ PRs created for all changes
- ✅ Releases are clean and documented
- ✅ Developer approves for deployment

---

## Final Notes

**👥 This workflow has been proven to work flawlessly. Follow it exactly.**

- Don't suggest shortcuts
- Don't skip testing
- Don't commit broken code
- Don't merge without review
- Don't deploy without testing

**When in doubt:**

1. Ask the developer
2. Check this document
3. Look at recent PRs for examples
4. Follow the established patterns

---

## Quick Reference Card

```bash
# Start new feature
gh issue create --title "Feature" --body "Description"
gh issue develop <number> --checkout

# Development cycle
[make changes]
go fmt ./...
go vet ./...
[test in CLI]

# Ship it
git add <files>
git commit -m "feat(scope): description"
git push origin <branch>
gh pr create --fill

# Release (after PRs merged to dev)
git checkout dev && git pull
# Review commits to determine version
git log --oneline origin/main..origin/dev
git tag -a v1.0.0 -m "Release v1.0.0"  # ⚠️ REQUIRED - Creates version tag
git push origin v1.0.0                   # Push tag
gh pr create --base main --head dev --fill
```

**Remember: This process works. Follow it exactly. No shortcuts.**

---

## 🤖 How to Ensure AI Assistants Follow This Document

### For Developers: Setting Up Your AI Assistant

**At the start of each session, use this prompt:**

```
Please read the AI_WORKFLOW_INSTRUCTIONS.MD file in this repository.
This document contains the complete development workflow you must follow.
Confirm that you've read it and will follow each phase in order.
```

**When the AI skips steps, use these reminders:**

```
Stop. Follow the workflow document. What phase are we in?
```

```
Have you asked me to test this yet? Check Phase 4 of the workflow.
```

```
Don't commit yet. Review the testing checklist first.
```

### For AI Assistants: Self-Check Questions

**Before creating issues/branches:**

- Have I presented a plan and received approval?
- Have I asked clarifying questions?

**Before committing:**

- Have I run `go fmt ./... && go vet ./...`?
- Have I asked the developer to test?
- Have I received explicit approval?
- Are there any linting errors?

**Before creating a PR:**

- Is the branch pushed?
- Have all tests passed?
- Is the commit message following conventional format?

### Adding to Claude Projects (Recommended)

If using Claude.ai Projects, add this to your project instructions:

```
DEVELOPMENT WORKFLOW:
- Always read and follow AI_WORKFLOW_INSTRUCTIONS.MD in the repository
- Never skip testing phases or commit without user approval
- Use conventional commits and follow the branching strategy
- Stop at checkpoints marked with 🛑 and wait for approval
```

### Integration with GitHub

**Add to `.github/CONTRIBUTING.md`:**

```markdown
## AI-Assisted Development

This project uses AI assistants for development. All AI assistants must follow
the workflow defined in `AI_WORKFLOW_INSTRUCTIONS.MD`. Key requirements:

- Present plans before implementation
- Wait for approval at checkpoints
- Never commit without user testing
- Follow conventional commit format
```

### Validation Script (Optional)

Create a git hook to validate commits:

```bash
# .git/hooks/commit-msg
#!/bin/bash
# Validate conventional commit format
commit_msg=$(cat "$1")
if ! echo "$commit_msg" | grep -qE "^(feat|fix|docs|chore|refactor|perf|test|feat!|fix!)(\(.+\))?: .+"; then
    echo "Error: Commit message must follow conventional commit format"
    echo "See AI_WORKFLOW_INSTRUCTIONS.MD for details"
    exit 1
fi
```

---

## 🤖 AI Assistant Response Templates

**Use these templates to communicate at key checkpoints:**

### After Receiving a Feature Request

```
I understand you want to [summarize request].

Let me make sure I understand correctly:
- [Key requirement 1]
- [Key requirement 2]
- [Edge case or clarification question]

Here's my plan:
1. [Step 1]
2. [Step 2]
3. [Step 3]

Does this approach work for you?

🛑 Waiting for your approval before proceeding.
```

### After Implementation, Before Committing

```
I've completed the implementation. Here's what I did:
- [Change 1]
- [Change 2]
- [Change 3]

I've run `go fmt ./... && go vet ./...` - no errors.

🛑 Can you please test this in the CLI?
- Test [specific functionality]
- Check [specific edge case]
- Verify [specific behavior]

Does everything work as expected?
```

### When Receiving Test Feedback with Issues

```
Thank you for testing. I'll fix these issues:
- [Issue 1] - [how I'll fix it]
- [Issue 2] - [how I'll fix it]

[Make fixes]

I've fixed the issues and re-run `go fmt ./... && go vet ./...`.

🛑 Can you please test again to confirm everything works?
```

### When Ready to Commit (After Approval)

```
Great! I have your approval. I'll now:
1. Stage the changes
2. Create a commit with message: "feat(scope): description"
3. Push the branch
4. Create a PR

[Proceed with commit]
```

### If Tempted to Skip a Step

```
⚠️ I need to stop here and follow the workflow.

According to AI_WORKFLOW_INSTRUCTIONS.MD, I should:
- [What the workflow says to do]
- [Checkpoint that needs approval]

Let me do that correctly.
```

---

**Document Version:** 5.0.0
**Last Updated:** 2026-01-15
**Maintained By:** [Developer Name]

**Changelog:**

- v5.0.0: Major reorganization for human developers - Added Quick Start for Humans, Visual Workflow Diagram, Command References, Decision Trees, Troubleshooting Guide, For New Team Members section, Reference Tables (Labels, Commit Types, Version Bump Matrix), section markers (👥/🧑‍💻/🤖), improved navigation and scannability
- v4.0.0: Added comprehensive issue label system (priority, type, effort), genericized project-specific section as template
- v3.1.0: Added project-specific details for individual projects
- v3.0.0: Added AI assistant instructions, checkpoints, response templates, and enforcement mechanisms
- v2.0.0: Generalized for all projects, changed from "project lead" to "developer"
- v1.0.0: Initial version

---

**🎉 You're all set! Use this document as your daily reference for development workflow.**

Quick links to most useful sections:

- **Starting work?** See [Quick Start](#-quick-start-for-humans-5-minute-overview)
- **Stuck?** See [Troubleshooting](#troubleshooting-guide)
- **Making decisions?** See [Decision Trees](#decision-trees)
- **Need a command?** See [Command References](#command-references)
- **New to team?** See [For New Team Members](#-for-new-team-members)
