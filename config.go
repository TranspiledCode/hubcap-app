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
	AvailableFilter github.Filters `json:"available_filter"`
}

func defaultConfig() Config {
	return Config{
		AvailableFilter: github.Filters{State: "open", Limit: 25},
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
