// internal/github/repo.go
package github

import (
	"bytes"
	"encoding/json"
	"errors"
	"os"
	"os/exec"
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

func RunCommandPassthrough(name string, args ...string) error {
	cmd := exec.Command(name, args...)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}
