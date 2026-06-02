// dashboard_test.go
package main

import (
	"testing"

	"hubcap/internal/github"
)

func TestBuildDashRows_AllSectionsPopulated(t *testing.T) {
	data := dashboardData{
		reviewRequests:  []github.PullRequest{{Number: 1}},
		myPRs:           []github.PullRequest{{Number: 2}},
		assignedIssues:  []github.Issue{{Number: 3}},
		availableIssues: []github.Issue{{Number: 4}},
	}
	rows := buildDashRows(data)

	// 4 sections × (1 header + 1 item) = 8 rows
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

func TestBuildDashRows_EmptySectionsHidden(t *testing.T) {
	data := dashboardData{
		reviewRequests:  []github.PullRequest{{Number: 1}},
		myPRs:           []github.PullRequest{},   // empty — should be hidden
		assignedIssues:  []github.Issue{{Number: 3}},
		availableIssues: []github.Issue{},           // empty — should be hidden
	}
	rows := buildDashRows(data)

	// section 0: header + 1 item = 2
	// section 1: hidden (empty)
	// section 2: header + 1 item = 2
	// section 3: hidden (empty)
	// total = 4
	if len(rows) != 4 {
		t.Fatalf("expected 4 rows (2 non-empty sections × 2), got %d", len(rows))
	}
	if rows[0].sectionID != secReviewRequests {
		t.Errorf("expected first header to be secReviewRequests, got %d", rows[0].sectionID)
	}
	if rows[2].sectionID != secAssigned {
		t.Errorf("expected third row header to be secAssigned, got %d", rows[2].sectionID)
	}
}

func TestBuildDashRows_AllEmpty(t *testing.T) {
	data := dashboardData{}
	rows := buildDashRows(data)
	if len(rows) != 0 {
		t.Errorf("expected 0 rows for all-empty data, got %d", len(rows))
	}
}
