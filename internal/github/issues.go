// internal/github/issues.go
package github

import (
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
)

// ── GraphQL response types ────────────────────────────────────────────────────
// The GitHub GraphQL API returns assignees/labels as connection nodes, so we
// need dedicated types for unmarshalling before converting to our flat Issue.

type gqlIssueNode struct {
	Number    int    `json:"number"`
	Title     string `json:"title"`
	Body      string `json:"body"`
	URL       string `json:"url"`
	State     string `json:"state"`
	CreatedAt string `json:"createdAt"`
	Author    struct {
		Login string `json:"login"`
	} `json:"author"`
	Assignees struct {
		Nodes []User `json:"nodes"`
	} `json:"assignees"`
	Labels struct {
		Nodes []Label `json:"nodes"`
	} `json:"labels"`
	IssueType *struct {
		Name string `json:"name"`
	} `json:"issueType"`
}

func (n gqlIssueNode) toIssue() Issue {
	i := Issue{
		Number:    n.Number,
		Title:     n.Title,
		Body:      n.Body,
		URL:       n.URL,
		State:     n.State,
		CreatedAt: n.CreatedAt,
		Author:    User{Login: n.Author.Login},
		Assignees: n.Assignees.Nodes,
		Labels:    n.Labels.Nodes,
	}
	if n.IssueType != nil {
		i.IssueType = n.IssueType.Name
	}
	return i
}

// ── gqlStateFilter maps our filter string to GraphQL IssueState list ─────────

func gqlStateFilter(state string) string {
	switch strings.ToLower(state) {
	case "closed":
		return "CLOSED"
	case "all":
		return "OPEN, CLOSED"
	default:
		return "OPEN"
	}
}

// ── Issue field fragment shared between list + single fetches ─────────────────

const issueFields = `
    number title body url state createdAt
    author { login }
    assignees(first:10) { nodes { login } }
    labels(first:20)    { nodes { name  } }
    issueType           { name           }`

// ── repoOwnerName resolves the current repo's owner and name ─────────────────
// The {owner}/{repo} gh-CLI placeholders only expand in REST URL paths, not in
// GraphQL -f field values, so we resolve them explicitly via FetchRepo().

func repoOwnerName() (owner, name string, err error) {
	nwo := FetchRepo() // "owner/repo" or "—" on error
	parts := strings.SplitN(nwo, "/", 2)
	if len(parts) != 2 || parts[0] == "" || parts[1] == "" {
		return "", "", fmt.Errorf("could not determine repository owner/name (got %q)", nwo)
	}
	return parts[0], parts[1], nil
}

// ── Public API ────────────────────────────────────────────────────────────────

// buildFetchIssuesArgs constructs the gh CLI args for FetchIssues.
// Extracted so the query-building logic can be unit-tested without invoking gh.
// label and milestone are passed as typed GraphQL variables ($label, $milestone)
// so user-supplied values are never interpolated into the query string.
func buildFetchIssuesArgs(owner, name string, filters Filters, assignee string) []string {
	states := gqlStateFilter(filters.State)

	fb := fmt.Sprintf("states:[%s]", states)
	if assignee != "" {
		fb += fmt.Sprintf(`, assignee:"%s"`, assignee)
	}
	if filters.Label != "" {
		fb += `, labels:[$label]`
	}
	if filters.Milestone != "" {
		fb += `, milestone:$milestone`
	}

	// Only declare $label / $milestone variables when they are referenced in
	// filterBy — GraphQL rejects declared-but-unused variables.
	varDecls := "$owner:String!, $name:String!, $limit:Int!"
	if filters.Label != "" {
		varDecls += ", $label:String"
	}
	if filters.Milestone != "" {
		varDecls += ", $milestone:String"
	}

	query := fmt.Sprintf(`
query(%s) {
  repository(owner:$owner, name:$name) {
    issues(first:$limit, filterBy:{%s}, orderBy:{field:CREATED_AT, direction:DESC}) {
      nodes {%s
      }
    }
  }
}`, varDecls, fb, issueFields)

	args := []string{"api", "graphql",
		"-f", "query=" + query,
		"-f", "owner=" + owner,
		"-f", "name=" + name,
		"-F", "limit=" + strconv.Itoa(filters.Limit),
	}
	if filters.Label != "" {
		args = append(args, "-f", "label="+filters.Label)
	}
	if filters.Milestone != "" {
		args = append(args, "-f", "milestone="+filters.Milestone)
	}
	return args
}

func FetchIssues(filters Filters) ([]Issue, error) {
	owner, name, err := repoOwnerName()
	if err != nil {
		return nil, err
	}

	// Resolve "@me" to the actual login — GraphQL filterBy does not accept it.
	assignee := filters.Assignee
	if assignee == "@me" {
		if login, err := GetCurrentUser(); err == nil && login != "" {
			assignee = login
		}
	}

	args := buildFetchIssuesArgs(owner, name, filters, assignee)
	out, err := RunCommand("gh", args...)
	if err != nil {
		return nil, err
	}

	var resp struct {
		Data struct {
			Repository struct {
				Issues struct {
					Nodes []gqlIssueNode `json:"nodes"`
				} `json:"issues"`
			} `json:"repository"`
		} `json:"data"`
	}
	if err := json.Unmarshal(out, &resp); err != nil {
		return nil, err
	}
	nodes := resp.Data.Repository.Issues.Nodes
	issues := make([]Issue, len(nodes))
	for i, n := range nodes {
		issues[i] = n.toIssue()
	}
	return issues, nil
}

func FetchIssue(number int) (Issue, error) {
	owner, name, err := repoOwnerName()
	if err != nil {
		return Issue{}, err
	}

	query := fmt.Sprintf(`
query($owner:String!, $name:String!) {
  repository(owner:$owner, name:$name) {
    issue(number:%d) {%s
    }
  }
}`, number, issueFields)

	out, err := RunCommand("gh", "api", "graphql",
		"-f", "query="+query,
		"-f", "owner="+owner,
		"-f", "name="+name,
	)
	if err != nil {
		return Issue{}, err
	}

	var resp struct {
		Data struct {
			Repository struct {
				Issue gqlIssueNode `json:"issue"`
			} `json:"repository"`
		} `json:"data"`
	}
	if err := json.Unmarshal(out, &resp); err != nil {
		return Issue{}, err
	}
	return resp.Data.Repository.Issue.toIssue(), nil
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

// AssignIssue assigns a specific user to an issue.
func AssignIssue(number int, login string) error {
	_, err := RunCommand("gh", "issue", "edit", strconv.Itoa(number), "--add-assignee", login)
	return err
}

// UnassignIssue removes a specific user from an issue's assignees.
func UnassignIssue(number int, login string) error {
	_, err := RunCommand("gh", "issue", "edit", strconv.Itoa(number), "--remove-assignee", login)
	return err
}

// ClearIssueAssignees removes all assignees from an issue.
func ClearIssueAssignees(number int, assignees []string) error {
	for _, login := range assignees {
		if _, err := RunCommand("gh", "issue", "edit", strconv.Itoa(number), "--remove-assignee", login); err != nil {
			return err
		}
	}
	return nil
}

// UpdateIssueAssignees adds and removes assignees in a single gh call.
func UpdateIssueAssignees(number int, add []string, remove []string) error {
	if len(add) == 0 && len(remove) == 0 {
		return nil
	}
	args := []string{"issue", "edit", strconv.Itoa(number)}
	for _, login := range add {
		args = append(args, "--add-assignee", login)
	}
	for _, login := range remove {
		args = append(args, "--remove-assignee", login)
	}
	_, err := RunCommand("gh", args...)
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
