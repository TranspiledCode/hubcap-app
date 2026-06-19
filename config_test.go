// config_test.go
package main

import (
	"os"
	"path/filepath"
	"testing"

	"hubcap/internal/github"
)

func TestLoadConfig_Defaults(t *testing.T) {
	cfg := loadConfigFrom("/nonexistent/path/config.json")
	if cfg.AvailableFilter.State != "open" {
		t.Errorf("expected default state open, got %s", cfg.AvailableFilter.State)
	}
	if cfg.AvailableFilter.Limit != 25 {
		t.Errorf("expected default limit 25, got %d", cfg.AvailableFilter.Limit)
	}
	if cfg.IssueFilters.Limit != 50 {
		t.Errorf("expected default IssueFilters.Limit 50, got %d", cfg.IssueFilters.Limit)
	}
	if cfg.PRFilters.Limit != 50 {
		t.Errorf("expected default PRFilters.Limit 50, got %d", cfg.PRFilters.Limit)
	}
}

func TestLoadConfig_InvalidJSON(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.json")
	if err := os.WriteFile(path, []byte("not json"), 0644); err != nil {
		t.Fatalf("setup: write file: %v", err)
	}

	cfg := loadConfigFrom(path)
	if cfg.AvailableFilter.State != "open" {
		t.Errorf("expected default state on bad JSON, got %s", cfg.AvailableFilter.State)
	}
}

func TestLoadConfig_MigratesZeroFilters(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.json")
	// Write a config that pre-dates filter persistence (no IssueFilters/PRFilters).
	legacy := `{"available_filter":{"State":"open","Limit":25},"ui_theme":"default"}`
	if err := os.WriteFile(path, []byte(legacy), 0644); err != nil {
		t.Fatalf("setup: %v", err)
	}
	cfg := loadConfigFrom(path)
	if cfg.IssueFilters.Limit != 50 {
		t.Errorf("expected migrated IssueFilters.Limit 50, got %d", cfg.IssueFilters.Limit)
	}
	if cfg.PRFilters.Limit != 50 {
		t.Errorf("expected migrated PRFilters.Limit 50, got %d", cfg.PRFilters.Limit)
	}
}

func TestSaveAndLoadConfig(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.json")

	cfg := Config{
		AvailableFilter: github.Filters{
			State: "open",
			Label: "ready",
			Limit: 10,
		},
		IssueFilters: github.Filters{State: "closed", Assignee: "@me", Limit: 30},
		PRFilters:    github.PRFilters{State: "open", Author: "bob", Limit: 20},
	}

	if err := saveConfigTo(cfg, path); err != nil {
		t.Fatalf("save failed: %v", err)
	}

	loaded := loadConfigFrom(path)
	if loaded.AvailableFilter.Label != "ready" {
		t.Errorf("expected label ready, got %s", loaded.AvailableFilter.Label)
	}
	if loaded.AvailableFilter.Limit != 10 {
		t.Errorf("expected limit 10, got %d", loaded.AvailableFilter.Limit)
	}
	if loaded.IssueFilters.State != "closed" {
		t.Errorf("expected IssueFilters.State closed, got %s", loaded.IssueFilters.State)
	}
	if loaded.IssueFilters.Assignee != "@me" {
		t.Errorf("expected IssueFilters.Assignee @me, got %s", loaded.IssueFilters.Assignee)
	}
	if loaded.PRFilters.Author != "bob" {
		t.Errorf("expected PRFilters.Author bob, got %s", loaded.PRFilters.Author)
	}
	if loaded.PRFilters.Limit != 20 {
		t.Errorf("expected PRFilters.Limit 20, got %d", loaded.PRFilters.Limit)
	}
}
