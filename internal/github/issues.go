// internal/github/issues.go
package github

import (
	"encoding/json"
	"strconv"
)

func FetchIssues(filters Filters) ([]Issue, error) {
	args := []string{
		"issue", "list",
		"--state", filters.State,
		"--limit", strconv.Itoa(filters.Limit),
		"--json", "number,title,assignees,labels,body,url,state,createdAt",
	}
	if filters.Assignee != "" {
		args = append(args, "--assignee", filters.Assignee)
	}
	if filters.Label != "" {
		args = append(args, "--label", filters.Label)
	}
	if filters.Milestone != "" {
		args = append(args, "--milestone", filters.Milestone)
	}
	output, err := RunCommand("gh", args...)
	if err != nil {
		return nil, err
	}
	return parseIssues(output)
}

func FetchIssue(number int) (Issue, error) {
	output, err := RunCommand(
		"gh", "issue", "view", strconv.Itoa(number),
		"--json", "number,title,body,state,author,assignees,labels,createdAt,url",
	)
	if err != nil {
		return Issue{}, err
	}
	var issue Issue
	if err := json.Unmarshal(output, &issue); err != nil {
		return Issue{}, err
	}
	return issue, nil
}

func CloseIssue(number int) error {
	_, err := RunCommand("gh", "issue", "close", strconv.Itoa(number))
	return err
}

func ReopenIssue(number int) error {
	_, err := RunCommand("gh", "issue", "reopen", strconv.Itoa(number))
	return err
}

func AssignIssueSelf(number int) error {
	_, err := RunCommand("gh", "issue", "edit", strconv.Itoa(number), "--add-assignee", "@me")
	return err
}

func UnassignIssueSelf(number int) error {
	_, err := RunCommand("gh", "issue", "edit", strconv.Itoa(number), "--remove-assignee", "@me")
	return err
}

func AddIssueLabel(number int, label string) error {
	_, err := RunCommand("gh", "issue", "edit", strconv.Itoa(number), "--add-label", label)
	return err
}

func DevelopBranch(issueNumber int, branchName string) error {
	_, err := RunCommand("gh", "issue", "develop", strconv.Itoa(issueNumber), "--checkout", "--name", branchName)
	return err
}

func parseIssues(data []byte) ([]Issue, error) {
	var issues []Issue
	return issues, json.Unmarshal(data, &issues)
}
