// internal/github/prs_test.go
package github

import (
	"testing"
)

func TestParsePRs(t *testing.T) {
	data := []byte(`[
		{"number":10,"title":"Add login","state":"open","isDraft":false,
		 "author":{"login":"bob"},"assignees":[],"labels":[{"name":"feature"}],
		 "headRefName":"10-add-login","statusCheckRollup":[{"status":"COMPLETED","conclusion":"SUCCESS"}],
		 "url":"https://github.com/o/r/pull/10","createdAt":"2026-01-01T00:00:00Z"},
		{"number":11,"title":"WIP: refactor","state":"open","isDraft":true,
		 "author":{"login":"bob"},"assignees":[],"labels":[],
		 "headRefName":"11-wip","statusCheckRollup":[],
		 "url":"https://github.com/o/r/pull/11","createdAt":"2026-01-02T00:00:00Z"}
	]`)

	prs, err := parsePRs(data)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(prs) != 2 {
		t.Fatalf("expected 2 PRs, got %d", len(prs))
	}
	if prs[0].Number != 10 {
		t.Errorf("expected PR 10, got %d", prs[0].Number)
	}
	if prs[1].IsDraft != true {
		t.Errorf("expected PR 11 to be draft")
	}
}

func TestFilterNonDraftPRs(t *testing.T) {
	prs := []PullRequest{
		{Number: 1, IsDraft: false},
		{Number: 2, IsDraft: true},
		{Number: 3, IsDraft: false},
	}
	result := FilterNonDraftPRs(prs)
	if len(result) != 2 {
		t.Fatalf("expected 2 non-draft PRs, got %d", len(result))
	}
	if result[0].Number != 1 || result[1].Number != 3 {
		t.Errorf("unexpected PR numbers: %v", result)
	}
}

func TestFilterNonDraftPRs_AllDraft(t *testing.T) {
	prs := []PullRequest{{Number: 1, IsDraft: true}, {Number: 2, IsDraft: true}}
	result := FilterNonDraftPRs(prs)
	if len(result) != 0 {
		t.Errorf("expected 0 results, got %d", len(result))
	}
}
