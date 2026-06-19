// main.go
package main

import (
	_ "embed"
	"fmt"
	"os"
	"strings"

	"hubcap/internal/github"

	tea "github.com/charmbracelet/bubbletea"
)

//go:embed VERSION
var versionRaw string

// version is the trimmed contents of the VERSION file, embedded at build time.
var version = strings.TrimSpace(versionRaw)

type TabID int

const (
	TabDashboard TabID = iota
	TabIssues
	TabPRs
)

// AppState is kept for compatibility with legacy code still being migrated
type AppState struct {
	ActiveTab       TabID
	IssueFilters    github.Filters
	PRFilters       github.PRFilters
	IssueSelected   int
	PRSelected      int
	DashboardCursor int
	DashboardStatus string
	Repo            string
}

// nextTab cycles TabDashboard → TabIssues → TabPRs → TabDashboard.
func nextTab(current TabID) TabID {
	return (current + 1) % 3
}

func main() {
	if err := require("gh"); err != nil {
		fatal(err)
	}

	cfg := loadConfig()
	cache := loadCache()
	repo := github.FetchRepo()

	model := newAppModel(
		repo,
		cfg,
		cfg.IssueFilters,
		cfg.PRFilters,
		cache,
	)

	p := tea.NewProgram(model, tea.WithAltScreen())
	if _, err := p.Run(); err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(1)
	}
	fmt.Println("\nBye.")
}
