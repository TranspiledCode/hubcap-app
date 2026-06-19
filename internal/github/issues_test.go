// internal/github/issues_test.go
package github

import (
	"encoding/json"
	"testing"
)

// parseGQLIssues is a test helper that unmarshals a GraphQL response body into
// a slice of Issue, mirroring what FetchIssues does internally.
func parseGQLIssues(data []byte) ([]Issue, error) {
	var resp struct {
		Data struct {
			Repository struct {
				Issues struct {
					Nodes []gqlIssueNode `json:"nodes"`
				} `json:"issues"`
			} `json:"repository"`
		} `json:"data"`
	}
	if err := json.Unmarshal(data, &resp); err != nil {
		return nil, err
	}
	nodes := resp.Data.Repository.Issues.Nodes
	issues := make([]Issue, len(nodes))
	for i, n := range nodes {
		issues[i] = n.toIssue()
	}
	return issues, nil
}

func TestParseIssues(t *testing.T) {
	data := []byte(`{
		"data": { "repository": { "issues": { "nodes": [
			{"number":42,"title":"Fix bug","state":"open","url":"https://github.com/o/r/issues/42",
			 "createdAt":"2026-01-01T00:00:00Z",
			 "author":{"login":"bob"},
			 "assignees":{"nodes":[{"login":"alice"}]},
			 "labels":{"nodes":[{"name":"bug"}]},
			 "issueType":{"name":"Bug"}},
			{"number":43,"title":"Add feature","state":"closed","url":"https://github.com/o/r/issues/43",
			 "createdAt":"2026-01-02T00:00:00Z",
			 "author":{"login":"bob"},
			 "assignees":{"nodes":[]},
			 "labels":{"nodes":[]},
			 "issueType":null}
		]}}}
	}`)

	issues, err := parseGQLIssues(data)
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
	if issues[0].IssueType != "Bug" {
		t.Errorf("expected IssueType Bug, got %q", issues[0].IssueType)
	}
	if issues[1].State != "closed" {
		t.Errorf("expected closed state, got %s", issues[1].State)
	}
	if issues[1].IssueType != "" {
		t.Errorf("expected empty IssueType for null, got %q", issues[1].IssueType)
	}
}

func TestParseIssues_Empty(t *testing.T) {
	data := []byte(`{"data":{"repository":{"issues":{"nodes":[]}}}}`)
	issues, err := parseGQLIssues(data)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(issues) != 0 {
		t.Errorf("expected 0 issues, got %d", len(issues))
	}
}

func TestParseIssues_Invalid(t *testing.T) {
	_, err := parseGQLIssues([]byte(`not json`))
	if err == nil {
		t.Error("expected error for invalid JSON, got nil")
	}
}
