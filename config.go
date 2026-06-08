// config.go
package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"hubcap/internal/github"
)

type Config struct {
	AvailableFilter     github.Filters   `json:"available_filter"`
	AutoRefreshEnabled  bool             `json:"auto_refresh_enabled"`
	AutoRefreshInterval int              `json:"auto_refresh_interval"` // in seconds
	UITheme             string           `json:"ui_theme"`              // "minimal" | "default" | "comfortable"
	IssueFilters        github.Filters   `json:"issue_filters"`
	PRFilters           github.PRFilters `json:"pr_filters"`
}

func defaultConfig() Config {
	return Config{
		AvailableFilter:     github.Filters{State: "open", Limit: 25},
		AutoRefreshEnabled:  false,
		AutoRefreshInterval: 60,
		UITheme:             "default",
		IssueFilters:        github.Filters{State: "open", Limit: 50},
		PRFilters:           github.PRFilters{State: "open", Limit: 50},
	}
}

func configPath() string {
	dir, err := os.UserConfigDir()
	if err != nil {
		dir = os.Getenv("HOME")
	}
	return filepath.Join(dir, "hubcap", "config.json")
}

func loadConfig() Config {
	return loadConfigFrom(configPath())
}

func loadConfigFrom(path string) Config {
	data, err := os.ReadFile(path)
	if err != nil {
		return defaultConfig()
	}
	var cfg Config
	if err := json.Unmarshal(data, &cfg); err != nil {
		fmt.Fprintf(os.Stderr, "hubcap: warning: bad config file, using defaults (%v)\n", err)
		return defaultConfig()
	}
	// Migrate existing configs that pre-date filter persistence: if filters
	// are zero-valued (Limit == 0), fill in sensible defaults.
	def := defaultConfig()
	if cfg.IssueFilters.Limit == 0 {
		cfg.IssueFilters = def.IssueFilters
	}
	if cfg.PRFilters.Limit == 0 {
		cfg.PRFilters = def.PRFilters
	}
	return cfg
}

func saveConfig(cfg Config) error {
	return saveConfigTo(cfg, configPath())
}

func saveConfigTo(cfg Config, path string) error {
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return err
	}
	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0644)
}
