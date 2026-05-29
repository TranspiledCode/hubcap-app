// dashboard_test.go
package main

import (
	"testing"

	"hubcap/internal/github"
)

func TestBuildRows_AllSectionsPopulated(t *testing.T) {
	data := dashboardResult{
		reviewRequests:  []github.PullRequest{{Number: 1}},
		myPRs:           []github.PullRequest{{Number: 2}},
		assignedIssues:  []github.Issue{{Number: 3}},
		availableIssues: []github.Issue{{Number: 4}},
	}
	collapsed := [4]bool{}
	rows := buildRows(data, collapsed)

	// 4 section headers + 4 items = 8 rows total
	if len(rows) != 8 {
		t.Fatalf("expected 8 rows, got %d", len(rows))
	}
	if !rows[0].isHeader {
		t.Error("expected first row to be a section header")
	}
	if rows[1].isHeader {
		t.Error("expected second row to be an item")
	}
}

func TestBuildRows_CollapsedSection(t *testing.T) {
	data := dashboardResult{
		reviewRequests:  []github.PullRequest{{Number: 1}, {Number: 2}},
		myPRs:           []github.PullRequest{},
		assignedIssues:  []github.Issue{{Number: 3}},
		availableIssues: []github.Issue{},
	}
	// Collapse section 0 (review requests); sections 1,3 are empty so hidden
	collapsed := [4]bool{true, false, false, false}
	rows := buildRows(data, collapsed)

	// Section 0 header only (collapsed, 2 items hidden)
	// Section 1 hidden (empty)
	// Section 2 header + 1 item
	// Section 3 hidden (empty)
	// = 3 rows
	if len(rows) != 3 {
		t.Fatalf("expected 3 rows, got %d", len(rows))
	}
	if !rows[0].isHeader || rows[0].sectionID != 0 {
		t.Error("expected first row to be section 0 header")
	}
}

func TestBuildRows_EmptySectionsHidden(t *testing.T) {
	data := dashboardResult{} // all empty
	collapsed := [4]bool{}
	rows := buildRows(data, collapsed)
	if len(rows) != 0 {
		t.Errorf("expected 0 rows for empty data, got %d", len(rows))
	}
}
