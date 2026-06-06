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

func FetchIssues(filters Filters) ([]Issue, error) {
	owner, name, err := repoOwnerName()
	if err != nil {
		return nil, err
	}

	states := gqlStateFilter(filters.State)

	// Build inline filterBy — values come from our own form so no injection risk.
	fb := fmt.Sprintf("states:[%s]", states)
	if filters.Assignee != "" {
		fb += fmt.Sprintf(`, assignee:"%s"`, filters.Assignee)
	}
	if filters.Label != "" {
		fb += fmt.Sprintf(`, labels:["%s"]`, filters.Label)
	}
	if filters.Milestone != "" {
		fb += fmt.Sprintf(`, milestone:"%s"`, filters.Milestone)
	}

	query := fmt.Sprintf(`
query($owner:String!, $name:String!, $limit:Int!) {
  repository(owner:$owner, name:$name) {
    issues(first:$limit, filterBy:{%s}, orderBy:{field:CREATED_AT, direction:DESC}) {
      nodes {%s
      }
    }
  }
}`, fb, issueFields)

	out, err := RunCommand("gh", "api", "graphql",
		"-f", "query="+query,
		"-f", "owner="+owner,
		"-f", "name="+name,
		"-F", "limit="+strconv.Itoa(filters.Limit),
	)
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

func AddIssueLabel(number int, label string) error {
	_, err := RunCommand("gh", "issue", "edit", strconv.Itoa(number), "--add-label", label)
	return err
}

func DevelopBranch(issueNumber int, branchName string) error {
	_, err := RunCommand("gh", "issue", "develop", strconv.Itoa(issueNumber), "--checkout", "--name", branchName)
	return err
}
