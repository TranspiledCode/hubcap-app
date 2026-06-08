// cache_test.go
package main

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"hubcap/internal/github"
)

func TestGetIssues_EmptyCache(t *testing.T) {
	c := AppCache{}
	issues, ok := c.GetIssues()
	if ok {
		t.Error("expected ok=false on empty cache, got true")
	}
	if issues != nil {
		t.Error("expected nil issues on empty cache")
	}
}

func TestSetAndGetIssues(t *testing.T) {
	c := AppCache{}
	want := []github.Issue{
		{Number: 1, Title: "First issue"},
		{Number: 2, Title: "Second issue"},
	}
	c.SetIssues(want)

	got, ok := c.GetIssues()
	if !ok {
		t.Fatal("expected ok=true after SetIssues")
	}
	if len(got) != len(want) {
		t.Fatalf("expected %d issues, got %d", len(want), len(got))
	}
	for i, issue := range got {
		if issue.Number != want[i].Number {
			t.Errorf("issue[%d]: expected number %d, got %d", i, want[i].Number, issue.Number)
		}
		if issue.Title != want[i].Title {
			t.Errorf("issue[%d]: expected title %q, got %q", i, want[i].Title, issue.Title)
		}
	}
}

func TestGetIssues_Expired(t *testing.T) {
	c := AppCache{}
	c.SetIssues([]github.Issue{{Number: 1}})
	// Back-date the timestamp past the TTL.
	c.Issues.Timestamp = time.Now().Add(-(listCacheTTL + time.Minute)).Unix()

	_, ok := c.GetIssues()
	if ok {
		t.Error("expected ok=false for expired cache entry, got true")
	}
}

func TestSetAndGetPRs(t *testing.T) {
	c := AppCache{}
	want := []github.PullRequest{
		{Number: 10, Title: "My PR"},
	}
	c.SetPRs(want)

	got, ok := c.GetPRs()
	if !ok {
		t.Fatal("expected ok=true after SetPRs")
	}
	if len(got) != 1 || got[0].Number != 10 {
		t.Errorf("unexpected PRs: %+v", got)
	}
}

func TestGetPRs_Expired(t *testing.T) {
	c := AppCache{}
	c.SetPRs([]github.PullRequest{{Number: 1}})
	c.PRs.Timestamp = time.Now().Add(-(listCacheTTL + time.Minute)).Unix()

	_, ok := c.GetPRs()
	if ok {
		t.Error("expected ok=false for expired PRs cache entry, got true")
	}
}

func TestSaveAndLoadCache(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "cache.json")

	c := AppCache{}
	c.SetIssues([]github.Issue{{Number: 42, Title: "Cached issue"}})
	c.SetPRs([]github.PullRequest{{Number: 7, Title: "Cached PR"}})

	if err := saveCacheTo(c, path); err != nil {
		t.Fatalf("saveCacheTo: %v", err)
	}

	loaded := loadCacheFrom(path)

	issues, ok := loaded.GetIssues()
	if !ok {
		t.Fatal("expected issues from loaded cache")
	}
	if len(issues) != 1 || issues[0].Number != 42 {
		t.Errorf("unexpected issues: %+v", issues)
	}

	prs, ok := loaded.GetPRs()
	if !ok {
		t.Fatal("expected PRs from loaded cache")
	}
	if len(prs) != 1 || prs[0].Number != 7 {
		t.Errorf("unexpected PRs: %+v", prs)
	}
}

func TestLoadCache_MissingFile(t *testing.T) {
	c := loadCacheFrom("/nonexistent/path/cache.json")
	_, ok := c.GetIssues()
	if ok {
		t.Error("expected ok=false when cache file is missing")
	}
}

func TestLoadCache_InvalidJSON(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "cache.json")
	if err := os.WriteFile(path, []byte("not json"), 0644); err != nil {
		t.Fatalf("setup: %v", err)
	}
	c := loadCacheFrom(path)
	_, ok := c.GetIssues()
	if ok {
		t.Error("expected ok=false on malformed cache file")
	}
}
