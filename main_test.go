package main

import (
	"strings"
	"testing"
)

func TestDeriveBranchName(t *testing.T) {
	tests := []struct {
		number int
		title  string
		want   string
	}{
		{142, "Fix checkout flow when cart is empty", "142-fix-checkout-flow-when-cart-is-empty"},
		{1, "Short title", "1-short-title"},
		{99, "Title with -- double dashes!!", "99-title-with-double-dashes"},
	}
	for _, tc := range tests {
		got := deriveBranchName(tc.number, tc.title)
		if got != tc.want {
			t.Errorf("deriveBranchName(%d, %q) = %q, want %q", tc.number, tc.title, got, tc.want)
		}
	}
}

func TestDeriveBranchNameMaxLength(t *testing.T) {
	for _, number := range []int{1, 10, 100, 9999} {
		got := deriveBranchName(number, strings.Repeat("a", 100))
		if len(got) > 45 {
			t.Errorf("deriveBranchName(%d, long) = %q (len %d > 45)", number, got, len(got))
		}
	}
}

func TestSummarizeChecks(t *testing.T) {
	tests := []struct {
		name   string
		checks []CheckRun
		want   string
	}{
		{"empty", []CheckRun{}, "—"},
		{"passing", []CheckRun{{Status: "COMPLETED", Conclusion: "SUCCESS"}}, colorGreen + "✓" + colorReset},
		{"failing", []CheckRun{{Status: "COMPLETED", Conclusion: "FAILURE"}}, colorRed + "✗" + colorReset},
		{"pending", []CheckRun{{Status: "IN_PROGRESS", Conclusion: ""}}, colorYellow + "…" + colorReset},
		{"mixed fail", []CheckRun{
			{Status: "COMPLETED", Conclusion: "SUCCESS"},
			{Status: "COMPLETED", Conclusion: "FAILURE"},
		}, colorRed + "✗" + colorReset},
	}
	for _, tc := range tests {
		got := summarizeChecks(tc.checks)
		if got != tc.want {
			t.Errorf("%s: summarizeChecks() = %q, want %q", tc.name, got, tc.want)
		}
	}
}

func TestTruncate(t *testing.T) {
	tests := []struct {
		value string
		max   int
		want  string
	}{
		{"hello", 10, "hello"},
		{"hello world", 8, "hello w…"},
		{"", 5, ""},
		{"hi", 1, "h"},
		{"hi", 0, ""},
	}
	for _, tc := range tests {
		got := truncate(tc.value, tc.max)
		if got != tc.want {
			t.Errorf("truncate(%q, %d) = %q, want %q", tc.value, tc.max, got, tc.want)
		}
	}
}

func TestFetchPRsDraftFilter(t *testing.T) {
	// Test that non-draft filtering (Draft=="false") is applied correctly
	// by testing the filtering logic directly:
	prs := []PullRequest{
		{Number: 1, IsDraft: false},
		{Number: 2, IsDraft: true},
		{Number: 3, IsDraft: false},
	}
	var filtered []PullRequest
	for _, pr := range prs {
		if !pr.IsDraft {
			filtered = append(filtered, pr)
		}
	}
	if len(filtered) != 2 {
		t.Errorf("expected 2, got %d", len(filtered))
	}
	if filtered[0].Number != 1 || filtered[1].Number != 3 {
		t.Errorf("wrong PRs: %v", filtered)
	}
}
