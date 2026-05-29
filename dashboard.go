// dashboard.go
package main

import (
	"fmt"
	"sync"

	"hubcap/internal/github"
)

// ── Data types ────────────────────────────────────────────────────────────────

type dashboardResult struct {
	reviewRequests  []github.PullRequest
	myPRs           []github.PullRequest
	assignedIssues  []github.Issue
	availableIssues []github.Issue
	errs            [4]error
}

// dashRow is one entry in the flat navigable list rendered on screen.
type dashRow struct {
	isHeader  bool
	sectionID int  // 0=reviewRequests, 1=myPRs, 2=assignedIssues, 3=availableIssues
	itemIdx   int  // index within section (-1 for headers)
	isIssue   bool // true = Issue row, false = PullRequest row
}

const (
	secReviewRequests = 0
	secMyPRs          = 1
	secAssigned       = 2
	secAvailable      = 3
)

var sectionNames = [4]string{
	"REVIEW REQUESTS",
	"MY OPEN PRs",
	"ASSIGNED TO ME",
	"AVAILABLE TO GRAB",
}

// sectionLen returns the item count for a given section.
func sectionLen(data dashboardResult, sec int) int {
	switch sec {
	case secReviewRequests:
		return len(data.reviewRequests)
	case secMyPRs:
		return len(data.myPRs)
	case secAssigned:
		return len(data.assignedIssues)
	case secAvailable:
		return len(data.availableIssues)
	}
	return 0
}

// buildRows constructs the flat navigable row list from dashboard data.
// Empty sections are omitted entirely. Collapsed sections show only their header.
func buildRows(data dashboardResult, collapsed [4]bool) []dashRow {
	var rows []dashRow
	for sec := 0; sec < 4; sec++ {
		count := sectionLen(data, sec)
		if count == 0 && data.errs[sec] == nil {
			continue // hidden when empty and no error
		}
		rows = append(rows, dashRow{isHeader: true, sectionID: sec, itemIdx: -1})
		if collapsed[sec] {
			continue
		}
		isIssue := sec == secAssigned || sec == secAvailable
		for i := 0; i < count; i++ {
			rows = append(rows, dashRow{isHeader: false, sectionID: sec, itemIdx: i, isIssue: isIssue})
		}
	}
	return rows
}

// ── Concurrent data fetch ─────────────────────────────────────────────────────

func fetchDashboard(cfg Config) dashboardResult {
	var wg sync.WaitGroup
	var result dashboardResult

	wg.Add(4)

	go func() {
		defer wg.Done()
		result.reviewRequests, result.errs[secReviewRequests] = github.FetchReviewRequests(25)
	}()
	go func() {
		defer wg.Done()
		result.myPRs, result.errs[secMyPRs] = github.FetchPRs(github.PRFilters{
			Author: "@me", State: "open", Limit: 25,
		})
	}()
	go func() {
		defer wg.Done()
		result.assignedIssues, result.errs[secAssigned] = github.FetchIssues(github.Filters{
			Assignee: "@me", State: "open", Limit: 25,
		})
	}()
	go func() {
		defer wg.Done()
		result.availableIssues, result.errs[secAvailable] = github.FetchIssues(cfg.AvailableFilter)
	}()

	wg.Wait()
	return result
}

// ── Stub — replaced in Task 12 ────────────────────────────────────────────────

// browseDashboard is the interactive TUI loop for the My Work tab.
// This stub is replaced with a full implementation in Task 12.
func browseDashboard(reader interface{}, state *AppState, cfg *Config) string {
	_ = reader
	_ = state
	_ = cfg
	return "quit"
}

// ensure fmt is used — will be used by render functions in later tasks.
var _ = fmt.Sprintf
