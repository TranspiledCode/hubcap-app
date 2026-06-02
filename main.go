// main.go
package main

import (
	"fmt"
	"os"

	"hubcap/internal/github"

	tea "github.com/charmbracelet/bubbletea"
)

const version = "0.1.0"

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
	repo := github.FetchRepo()

	model := newAppModel(
		repo,
		cfg,
		github.Filters{State: "open", Limit: 50},
		github.PRFilters{State: "open", Limit: 50},
	)

	p := tea.NewProgram(model, tea.WithAltScreen())
	if _, err := p.Run(); err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(1)
	}
	fmt.Println("\nBye.")
}
