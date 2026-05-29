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
}

func TestLoadConfig_InvalidJSON(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.json")
	os.WriteFile(path, []byte("not json"), 0644)

	cfg := loadConfigFrom(path)
	if cfg.AvailableFilter.State != "open" {
		t.Errorf("expected default state on bad JSON, got %s", cfg.AvailableFilter.State)
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
}
