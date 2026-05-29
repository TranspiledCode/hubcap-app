// internal/github/issues_test.go
package github

import (
	"testing"
)

func TestParseIssues(t *testing.T) {
	data := []byte(`[
		{"number":42,"title":"Fix bug","state":"open","url":"https://github.com/o/r/issues/42",
		 "assignees":[{"login":"alice"}],"labels":[{"name":"bug"}],"createdAt":"2026-01-01T00:00:00Z"},
		{"number":43,"title":"Add feature","state":"closed","url":"https://github.com/o/r/issues/43",
		 "assignees":[],"labels":[],"createdAt":"2026-01-02T00:00:00Z"}
	]`)

	issues, err := parseIssues(data)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(issues) != 2 {
		t.Fatalf("expected 2 issues, got %d", len(issues))
	}
	if issues[0].Number != 42 {
		t.Errorf("expected issue 42, got %d", issues[0].Number)
	}
	if issues[0].Assignees[0].Login != "alice" {
		t.Errorf("expected assignee alice, got %s", issues[0].Assignees[0].Login)
	}
	if issues[0].Labels[0].Name != "bug" {
		t.Errorf("expected label bug, got %s", issues[0].Labels[0].Name)
	}
	if issues[1].State != "closed" {
		t.Errorf("expected closed state, got %s", issues[1].State)
	}
}

func TestParseIssues_Empty(t *testing.T) {
	issues, err := parseIssues([]byte(`[]`))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(issues) != 0 {
		t.Errorf("expected 0 issues, got %d", len(issues))
	}
}

func TestParseIssues_Invalid(t *testing.T) {
	_, err := parseIssues([]byte(`not json`))
	if err == nil {
		t.Error("expected error for invalid JSON, got nil")
	}
}
