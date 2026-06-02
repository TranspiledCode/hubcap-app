// dashboard_test.go
package main

import (
	"testing"

	"hubcap/internal/github"
)

func TestBuildDashRows_AllSectionsPopulated(t *testing.T) {
	data := dashboardData{
		reviewRequests: []github.PullRequest{{Number: 1}},
		myPRs:          []github.PullRequest{{Number: 2}},
		assignedIssues: []github.Issue{{Number: 3}},
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
		reviewRequests: []github.PullRequest{{Number: 1}},
		myPRs:          []github.PullRequest{}, // empty — hidden
		assignedIssues: []github.Issue{{Number: 3}},
	}
	rows := buildDashRows(data)

	// section 0: header + 1 item = 2
	// section 1: hidden (empty)
	// section 2: header + 1 item = 2
	// total = 4
	if len(rows) != 4 {
		t.Fatalf("expected 4 rows, got %d", len(rows))
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

func TestBuildDashRows_SectionOrder(t *testing.T) {
	data := dashboardData{
		reviewRequests: []github.PullRequest{{Number: 10}, {Number: 11}},
		myPRs:          []github.PullRequest{{Number: 20}},
		assignedIssues: []github.Issue{{Number: 30}},
	}
	rows := buildDashRows(data)

	// section 0: 1 header + 2 items = 3
	// section 1: 1 header + 1 item  = 2
	// section 2: 1 header + 1 item  = 2
	// total = 7
	if len(rows) != 7 {
		t.Fatalf("expected 7 rows, got %d", len(rows))
	}
	if rows[0].sectionID != secReviewRequests || !rows[0].isHeader {
		t.Errorf("row 0 should be secReviewRequests header")
	}
	if rows[1].isHeader || rows[1].sectionID != secReviewRequests {
		t.Errorf("row 1 should be secReviewRequests item")
	}
	if rows[3].sectionID != secMyPRs || !rows[3].isHeader {
		t.Errorf("row 3 should be secMyPRs header")
	}
	if rows[5].sectionID != secAssigned || !rows[5].isHeader {
		t.Errorf("row 5 should be secAssigned header")
	}
}
