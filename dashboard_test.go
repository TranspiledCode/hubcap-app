// dashboard_test.go
package main

import (
	"testing"

	"hubcap/internal/github"
)

func TestBuildDashRows_AllSectionsPopulated(t *testing.T) {
	data := dashboardData{
		myPRs:          []github.PullRequest{{Number: 1}},
		assignedIssues: []github.Issue{{Number: 2}},
		reviewRequests: []github.PullRequest{{Number: 3}},
	}
	rows := buildDashRows(data)

	// 3 sections × (1 header + 1 item) = 6 rows
	if len(rows) != 6 {
		t.Fatalf("expected 6 rows, got %d", len(rows))
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
		myPRs:          []github.PullRequest{}, // empty — hidden
		assignedIssues: []github.Issue{{Number: 2}},
		reviewRequests: []github.PullRequest{{Number: 3}},
	}
	rows := buildDashRows(data)

	// secMyPRs: hidden (empty)
	// secAssigned: header + 1 item = 2
	// secReviewRequests: header + 1 item = 2
	// total = 4
	if len(rows) != 4 {
		t.Fatalf("expected 4 rows, got %d", len(rows))
	}
	if rows[0].sectionID != secAssigned {
		t.Errorf("expected first header to be secAssigned, got %d", rows[0].sectionID)
	}
	if rows[2].sectionID != secReviewRequests {
		t.Errorf("expected third row to be secReviewRequests header, got %d", rows[2].sectionID)
	}
}

func TestBuildDashRows_AllEmpty(t *testing.T) {
	data := dashboardData{}
	rows := buildDashRows(data)
	if len(rows) != 0 {
		t.Errorf("expected 0 rows for all-empty data, got %d", len(rows))
	}
}

func TestBuildDashRows_SectionOrder(t *testing.T) {
	data := dashboardData{
		myPRs:          []github.PullRequest{{Number: 10}, {Number: 11}},
		assignedIssues: []github.Issue{{Number: 20}},
		reviewRequests: []github.PullRequest{{Number: 30}},
	}
	rows := buildDashRows(data)

	// secMyPRs:          1 header + 2 items = 3
	// secAssigned:       1 header + 1 item  = 2
	// secReviewRequests: 1 header + 1 item  = 2
	// total = 7
	if len(rows) != 7 {
		t.Fatalf("expected 7 rows, got %d", len(rows))
	}
	if rows[0].sectionID != secMyPRs || !rows[0].isHeader {
		t.Errorf("row 0 should be secMyPRs header")
	}
	if rows[1].isHeader || rows[1].sectionID != secMyPRs {
		t.Errorf("row 1 should be secMyPRs item")
	}
	if rows[3].sectionID != secAssigned || !rows[3].isHeader {
		t.Errorf("row 3 should be secAssigned header")
	}
	if rows[5].sectionID != secReviewRequests || !rows[5].isHeader {
		t.Errorf("row 5 should be secReviewRequests header")
	}
}

func TestBuildDashRows_AssignedMixed(t *testing.T) {
	data := dashboardData{
		assignedIssues: []github.Issue{{Number: 10}, {Number: 11}},
		assignedPRs:    []github.PullRequest{{Number: 20}},
	}
	rows := buildDashRows(data)

	// secAssigned: 1 header + 2 issue rows + 1 PR row = 4
	if len(rows) != 4 {
		t.Fatalf("expected 4 rows, got %d", len(rows))
	}
	if rows[0].sectionID != secAssigned || !rows[0].isHeader {
		t.Errorf("row 0 should be secAssigned header")
	}
	if !rows[1].isIssue {
		t.Errorf("row 1 should be an issue row")
	}
	if !rows[2].isIssue {
		t.Errorf("row 2 should be an issue row")
	}
	if rows[3].isIssue {
		t.Errorf("row 3 should be a PR row")
	}
	if rows[3].itemIdx != 0 {
		t.Errorf("row 3 PR itemIdx should be 0, got %d", rows[3].itemIdx)
	}
}
