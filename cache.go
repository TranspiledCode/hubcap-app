// cache.go
package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"time"

	"hubcap/internal/github"
)

// listCacheTTL is how long list data (issues, PRs) stays fresh in the cache.
const listCacheTTL = 5 * time.Minute

// cacheEntry holds marshalled JSON data and the Unix timestamp of when it was
// stored. A zero Timestamp means the entry is empty.
type cacheEntry struct {
	Data      json.RawMessage `json:"data"`
	Timestamp int64           `json:"timestamp"`
}

// AppCache is the in-memory representation of ~/.config/hubcap/cache.json.
// It stores the most recently fetched issue and PR list data so the app can
// display something instantly at startup while a fresh fetch runs in the
// background (stale-while-revalidate).
type AppCache struct {
	Issues cacheEntry `json:"issues"`
	PRs    cacheEntry `json:"prs"`
}

func cachePath() string {
	dir, err := os.UserConfigDir()
	if err != nil {
		dir = os.Getenv("HOME")
	}
	return filepath.Join(dir, "hubcap", "cache.json")
}

func loadCache() AppCache {
	return loadCacheFrom(cachePath())
}

func loadCacheFrom(path string) AppCache {
	data, err := os.ReadFile(path)
	if err != nil {
		return AppCache{}
	}
	var c AppCache
	if err := json.Unmarshal(data, &c); err != nil {
		return AppCache{}
	}
	return c
}

func saveCache(c AppCache) error {
	return saveCacheTo(c, cachePath())
}

func saveCacheTo(c AppCache, path string) error {
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return err
	}
	data, err := json.MarshalIndent(c, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0644)
}

// GetIssues returns the cached issue list if the entry is younger than
// listCacheTTL. Returns nil, false when the cache is empty or expired.
func (c *AppCache) GetIssues() ([]github.Issue, bool) {
	if c.Issues.Data == nil || c.Issues.Timestamp == 0 {
		return nil, false
	}
	if time.Since(time.Unix(c.Issues.Timestamp, 0)) > listCacheTTL {
		return nil, false
	}
	var issues []github.Issue
	if err := json.Unmarshal(c.Issues.Data, &issues); err != nil {
		return nil, false
	}
	return issues, true
}

// SetIssues stores the issue list with the current timestamp.
func (c *AppCache) SetIssues(issues []github.Issue) {
	data, err := json.Marshal(issues)
	if err != nil {
		return
	}
	c.Issues = cacheEntry{Data: data, Timestamp: time.Now().Unix()}
}

// GetPRs returns the cached PR list if the entry is younger than listCacheTTL.
// Returns nil, false when the cache is empty or expired.
func (c *AppCache) GetPRs() ([]github.PullRequest, bool) {
	if c.PRs.Data == nil || c.PRs.Timestamp == 0 {
		return nil, false
	}
	if time.Since(time.Unix(c.PRs.Timestamp, 0)) > listCacheTTL {
		return nil, false
	}
	var prs []github.PullRequest
	if err := json.Unmarshal(c.PRs.Data, &prs); err != nil {
		return nil, false
	}
	return prs, true
}

// SetPRs stores the PR list with the current timestamp.
func (c *AppCache) SetPRs(prs []github.PullRequest) {
	data, err := json.Marshal(prs)
	if err != nil {
		return
	}
	c.PRs = cacheEntry{Data: data, Timestamp: time.Now().Unix()}
}
