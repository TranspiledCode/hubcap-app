// internal/github/prs.go
package github

import (
	"encoding/json"
	"strconv"
)

func FetchPRs(filters PRFilters) ([]PullRequest, error) {
	args := []string{
		"pr", "list",
		"--state", filters.State,
		"--limit", strconv.Itoa(filters.Limit),
		"--json", "number,title,author,assignees,labels,state,isDraft,headRefName,baseRefName,statusCheckRollup,url",
	}
	if filters.Author != "" {
		args = append(args, "--author", filters.Author)
	}
	if filters.Assignee != "" {
		args = append(args, "--assignee", filters.Assignee)
	}
	if filters.Label != "" {
		args = append(args, "--label", filters.Label)
	}
	if filters.Draft == "true" {
		args = append(args, "--draft")
	}
	// Search takes precedence; fall back to ReviewStatus-based search.
	if filters.Search != "" {
		args = append(args, "--search", filters.Search)
	} else if filters.ReviewStatus != "" {
		args = append(args, "--search", "review:"+filters.ReviewStatus)
	}
	output, err := RunCommand("gh", args...)
	if err != nil {
		return nil, err
	}
	prs, err := parsePRs(output)
	if err != nil {
		return nil, err
	}
	if filters.Draft == "false" {
		prs = FilterNonDraftPRs(prs)
	}
	return prs, nil
}

// FetchReviewRequests returns open PRs where the authenticated user is a requested reviewer.
func FetchReviewRequests(limit int) ([]PullRequest, error) {
	return FetchPRs(PRFilters{
		State:  "open",
		Search: "review-requested:@me",
		Limit:  limit,
	})
}

func FetchPR(number int) (PullRequest, error) {
	output, err := RunCommand(
		"gh", "pr", "view", strconv.Itoa(number),
		"--json", "number,title,body,author,assignees,labels,state,isDraft,headRefName,baseRefName,reviewDecision,statusCheckRollup,url,createdAt",
	)
	if err != nil {
		return PullRequest{}, err
	}
	var pr PullRequest
	if err := json.Unmarshal(output, &pr); err != nil {
		return PullRequest{}, err
	}
	return pr, nil
}

func ClosePR(number int) error {
	_, err := RunCommand("gh", "pr", "close", strconv.Itoa(number))
	return err
}

func ReopenPR(number int) error {
	_, err := RunCommand("gh", "pr", "reopen", strconv.Itoa(number))
	return err
}

func CheckoutPR(number int) error {
	_, err := RunCommand("gh", "pr", "checkout", strconv.Itoa(number))
	return err
}

func MergePR(number int, strategy string) error {
	_, err := RunCommand("gh", "pr", "merge", strconv.Itoa(number), "--"+strategy)
	return err
}

func CreatePRFill() error {
	_, err := RunCommand("gh", "pr", "create", "--fill")
	return err
}

func RequestReview(number int, reviewer string) error {
	_, err := RunCommand("gh", "pr", "edit", strconv.Itoa(number), "--add-reviewer", reviewer)
	return err
}

func FilterNonDraftPRs(prs []PullRequest) []PullRequest {
	out := make([]PullRequest, 0, len(prs))
	for _, pr := range prs {
		if !pr.IsDraft {
			out = append(out, pr)
		}
	}
	return out
}

func parsePRs(data []byte) ([]PullRequest, error) {
	var prs []PullRequest
	return prs, json.Unmarshal(data, &prs)
}
