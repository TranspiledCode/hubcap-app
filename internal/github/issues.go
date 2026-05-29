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
	return RunCommandPassthrough("gh", "issue", "close", strconv.Itoa(number))
}

func ReopenIssue(number int) error {
	return RunCommandPassthrough("gh", "issue", "reopen", strconv.Itoa(number))
}

func AssignIssueSelf(number int) error {
	return RunCommandPassthrough("gh", "issue", "edit", strconv.Itoa(number), "--add-assignee", "@me")
}

func AddIssueLabel(number int, label string) error {
	return RunCommandPassthrough("gh", "issue", "edit", strconv.Itoa(number), "--add-label", label)
}

func parseIssues(data []byte) ([]Issue, error) {
	var issues []Issue
	return issues, json.Unmarshal(data, &issues)
}
