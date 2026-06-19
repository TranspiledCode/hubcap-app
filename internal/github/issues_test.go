// internal/github/issues_test.go
package github

import (
	"encoding/json"
	"strings"
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

// ── buildFetchIssuesArgs injection tests ─────────────────────────────────────

// queryArg extracts the value of the "-f query=..." argument from args.
func queryArg(args []string) string {
	for i, a := range args {
		if a == "-f" && i+1 < len(args) && len(args[i+1]) > 6 && args[i+1][:6] == "query=" {
			return args[i+1][6:]
		}
	}
	return ""
}

// hasFlag reports whether args contains "-f key=value".
func hasFlag(args []string, key, value string) bool {
	target := key + "=" + value
	for i, a := range args {
		if a == "-f" && i+1 < len(args) && args[i+1] == target {
			return true
		}
	}
	return false
}

func TestBuildFetchIssuesArgs_LabelNotInterpolated(t *testing.T) {
	label := `bug"injection`
	filters := Filters{Label: label, Limit: 30}
	args := buildFetchIssuesArgs("owner", "repo", filters, "")

	query := queryArg(args)
	if strings.Contains(query, label) {
		t.Errorf("query string must not contain raw label value %q", label)
	}
	if !strings.Contains(query, "$label") {
		t.Error("query string must reference $label variable")
	}
	if !hasFlag(args, "label", label) {
		t.Errorf("args must contain -f label=%q", label)
	}
}

func TestBuildFetchIssuesArgs_MilestoneNotInterpolated(t *testing.T) {
	milestone := `v1.0"injection`
	filters := Filters{Milestone: milestone, Limit: 30}
	args := buildFetchIssuesArgs("owner", "repo", filters, "")

	query := queryArg(args)
	if strings.Contains(query, milestone) {
		t.Errorf("query string must not contain raw milestone value %q", milestone)
	}
	if !strings.Contains(query, "$milestone") {
		t.Error("query string must reference $milestone variable")
	}
	if !hasFlag(args, "milestone", milestone) {
		t.Errorf("args must contain -f milestone=%q", milestone)
	}
}

func TestBuildFetchIssuesArgs_NoLabelOrMilestone(t *testing.T) {
	filters := Filters{Limit: 30}
	args := buildFetchIssuesArgs("owner", "repo", filters, "")

	query := queryArg(args)
	if strings.Contains(query, "labels:[$label]") {
		t.Error("query must not include labels filterBy entry when Label is empty")
	}
	if strings.Contains(query, "milestone:$milestone") {
		t.Error("query must not include milestone filterBy entry when Milestone is empty")
	}
	// Variable declarations must be absent too — GraphQL rejects unused vars.
	if strings.Contains(query, "$label:String") {
		t.Error("query must not declare $label variable when Label is empty")
	}
	if strings.Contains(query, "$milestone:String") {
		t.Error("query must not declare $milestone variable when Milestone is empty")
	}
	if hasFlag(args, "label", "") {
		t.Error("args must not include -f label= when Label is empty")
	}
	if hasFlag(args, "milestone", "") {
		t.Error("args must not include -f milestone= when Milestone is empty")
	}
}
