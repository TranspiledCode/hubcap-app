// config_form.go
package main

import (
	"fmt"
	"strconv"

	"github.com/charmbracelet/huh"
)

// ConfigVals holds the mutable values bound to the configuration form.
// It must be heap-allocated (use &ConfigVals{}) so huh's Value() pointers
// remain stable across BubbleTea value-receiver model copies.
type ConfigVals struct {
	AutoRefreshEnabled  bool
	AutoRefreshInterval string
	UITheme             string
	ColorTheme          string
	ActionChoice        string
}

// InitConfigVals pre-populates vals from the current config settings.
func InitConfigVals(vals *ConfigVals, cfg Config) {
	vals.AutoRefreshEnabled = cfg.AutoRefreshEnabled
	vals.AutoRefreshInterval = fmt.Sprintf("%d", cfg.AutoRefreshInterval)
	vals.UITheme = cfg.UITheme
	if vals.UITheme == "" {
		vals.UITheme = "default"
	}
	vals.ColorTheme = cfg.ColorTheme
	if vals.ColorTheme == "" {
		vals.ColorTheme = "default"
	}
	vals.ActionChoice = "save"
}

// BuildConfigForm constructs a *huh.Form bound to vals. Call form.Init()
// to start it; route messages through form.Update(msg) inside your model.
func BuildConfigForm(vals *ConfigVals) *huh.Form {
	return huh.NewForm(huh.NewGroup(
		huh.NewConfirm().
			Title("Enable auto-refresh").
			Description("Automatically refresh data at the configured interval.").
			Value(&vals.AutoRefreshEnabled),
		huh.NewInput().
			Title("Refresh interval (seconds)").
			Placeholder("e.g. 60").
			Value(&vals.AutoRefreshInterval).
			DescriptionFunc(func() string {
				if vals.AutoRefreshEnabled {
					return "Required when auto-refresh is enabled."
				}
				return ""
			}, &vals.AutoRefreshEnabled),
		huh.NewSelect[string]().
			Title("UI theme").
			Description("Controls footer button size and form padding.").
			Options(
				huh.NewOption("Minimal  — single-row compact buttons", "minimal"),
				huh.NewOption("Default  — rounded bordered buttons", "default"),
				huh.NewOption("Comfortable — wider buttons & forms", "comfortable"),
			).
			Value(&vals.UITheme),
		huh.NewSelect[string]().
			Title("Colour theme").
			Description("Accent and status colours. Press t anywhere to cycle.").
			Options(
				huh.NewOption("Default    — amber & green", "default"),
				huh.NewOption("Transpiled — electric blue & violet", "transpiled"),
				huh.NewOption("Cobalt 2   — deep blue, mint & yellow", "cobalt2"),
				huh.NewOption("ImageScoop — periwinkle, purple & lime", "imagescoop"),
			).
			Value(&vals.ColorTheme),
		huh.NewSelect[string]().
			Title("Action").
			Options(
				huh.NewOption("Save settings", "save"),
				huh.NewOption("Reset to defaults", "reset"),
			).
			Value(&vals.ActionChoice),
	)).WithTheme(huh.ThemeCatppuccin())
}

// ResolveConfig reads the completed vals and returns an updated Config.
func ResolveConfig(vals *ConfigVals, current Config) Config {
	if vals.ActionChoice == "reset" {
		return defaultConfig()
	}
	cfg := current
	cfg.AutoRefreshEnabled = vals.AutoRefreshEnabled
	if vals.AutoRefreshInterval != "" {
		if interval, err := strconv.Atoi(vals.AutoRefreshInterval); err == nil && interval > 0 {
			cfg.AutoRefreshInterval = interval
		}
	}
	cfg.UITheme = vals.UITheme
	cfg.ColorTheme = vals.ColorTheme
	return cfg
}
