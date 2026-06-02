// internal/github/repo.go
package github

import (
	"bytes"
	"encoding/json"
	"errors"
	"os"
	"os/exec"
	"os/signal"
	"strings"
)

type User struct {
	Login string `json:"login"`
}

type Label struct {
	Name string `json:"name"`
}

type CheckRun struct {
	Status     string `json:"status"`
	Conclusion string `json:"conclusion"`
}

type Issue struct {
	Number    int     `json:"number"`
	Title     string  `json:"title"`
	Body      string  `json:"body"`
	URL       string  `json:"url"`
	State     string  `json:"state"`
	Author    User    `json:"author"`
	Assignees []User  `json:"assignees"`
	Labels    []Label `json:"labels"`
	CreatedAt string  `json:"createdAt"`
}

type PullRequest struct {
	Number         int        `json:"number"`
	Title          string     `json:"title"`
	Body           string     `json:"body"`
	State          string     `json:"state"`
	IsDraft        bool       `json:"isDraft"`
	Author         User       `json:"author"`
	Assignees      []User     `json:"assignees"`
	Labels         []Label    `json:"labels"`
	HeadRefName    string     `json:"headRefName"`
	ReviewDecision string     `json:"reviewDecision"`
	StatusRollup   []CheckRun `json:"statusCheckRollup"`
	URL            string     `json:"url"`
	CreatedAt      string     `json:"createdAt"`
}

type Filters struct {
	State     string
	Assignee  string
	Label     string
	Milestone string
	Limit     int
}

// PRFilters extends the original with Author and Search for dashboard queries.
type PRFilters struct {
	State        string
	Author       string
	Assignee     string
	Label        string
	Draft        string // "true" = draft only, "false" = non-draft only, "" = all
	ReviewStatus string // used by filter UI: maps to "review:<value>"
	Search       string // raw --search value; takes precedence over ReviewStatus
	Limit        int
}

// FetchAssignees returns the list of users that can be assigned to issues/PRs
// in the current repo (collaborators with write access).
func FetchAssignees() ([]string, error) {
	output, err := RunCommand("gh", "api", "repos/{owner}/{repo}/assignees", "--paginate")
	if err != nil {
		return nil, err
	}
	var users []User
	if err := json.Unmarshal(output, &users); err != nil {
		return nil, err
	}
	logins := make([]string, 0, len(users))
	for _, u := range users {
		logins = append(logins, u.Login)
	}
	return logins, nil
}

// FetchLabels returns the list of label names defined on the current repo.
// Returns nil (no error) if labels cannot be fetched.
func FetchLabels() ([]string, error) {
	output, err := RunCommand("gh", "label", "list", "--limit", "200", "--json", "name")
	if err != nil {
		return nil, err
	}
	var labels []Label
	if err := json.Unmarshal(output, &labels); err != nil {
		return nil, err
	}
	names := make([]string, 0, len(labels))
	for _, l := range labels {
		names = append(names, l.Name)
	}
	return names, nil
}

func FetchRepo() string {
	type repoResponse struct {
		NameWithOwner string `json:"nameWithOwner"`
	}
	output, err := RunCommand("gh", "repo", "view", "--json", "nameWithOwner")
	if err != nil {
		return "—"
	}
	var repo repoResponse
	if err := json.Unmarshal(output, &repo); err != nil {
		return "—"
	}
	return repo.NameWithOwner
}

func RunCommand(name string, args ...string) ([]byte, error) {
	cmd := exec.Command(name, args...)

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		message := strings.TrimSpace(stderr.String())
		if message == "" {
			message = err.Error()
		}
		return nil, errors.New(message)
	}

	return stdout.Bytes(), nil
}

func CreateIssue(title, body string, labels []string) error {
	args := []string{"issue", "create", "--title", title, "--body", body}
	for _, l := range labels {
		args = append(args, "--label", l)
	}
	_, err := RunCommand("gh", args...)
	return err
}

func CreatePR(title, body, base string, draft bool) error {
	args := []string{"pr", "create", "--title", title, "--body", body, "--base", base}
	if draft {
		args = append(args, "--draft")
	}
	_, err := RunCommand("gh", args...)
	return err
}

func RunCommandPassthrough(name string, args ...string) error {
	cmd := exec.Command(name, args...)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	// Ignore SIGINT while the child runs so Ctrl+C cancels the child without
	// killing hubcap. Signal handling is restored when this function returns.
	signal.Ignore(os.Interrupt)
	defer signal.Reset(os.Interrupt)
	return cmd.Run()
}
