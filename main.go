// main.go
package main

import (
	"bufio"
	"fmt"
	"os"

	"hubcap/internal/github"
)

const version = "0.1.0"

type TabID int

const (
	TabDashboard TabID = iota
	TabIssues
	TabPRs
)

type AppState struct {
	ActiveTab         TabID
	IssueFilters      github.Filters
	PRFilters         github.PRFilters
	IssueSelected     int
	PRSelected        int
	DashboardCursor   int
	DashboardStatus   string
	Repo              string
}

// nextTab cycles TabDashboard → TabIssues → TabPRs → TabDashboard.
func nextTab(current TabID) TabID {
	return (current + 1) % 3
}

func main() {
	if err := require("gh"); err != nil {
		fatal(err)
	}

	reader := bufio.NewReader(os.Stdin)
	cfg := loadConfig()

	state := &AppState{
		ActiveTab:    TabDashboard,
		IssueFilters: github.Filters{State: "open", Limit: 50},
		PRFilters:    github.PRFilters{State: "open", Limit: 50},
	}
	state.Repo = github.FetchRepo()

	for {
		var action string
		switch state.ActiveTab {
		case TabDashboard:
			action = browseDashboard(reader, state, &cfg)
		case TabIssues:
			action = browseIssues(reader, state)
		case TabPRs:
			action = browsePRs(reader, state)
		}
		if action == "quit" {
			fmt.Println("\nBye.")
			return
		}
	}
}
