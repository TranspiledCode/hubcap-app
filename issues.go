// issues.go
package main

import (
	"fmt"
	"strconv"
	"strings"

	"hubcap/internal/github"

	"github.com/charmbracelet/huh"
)

func configureFilters(state *AppState) github.Filters {
	filters := state.IssueFilters

	// Initialize with current values
	stateChoice := filters.State

	var availableAssignees []string
	var availableLabels []string

	// Fetch available options with spinners
	withSpinner("Fetching assignees...", func() error {
		var err error
		availableAssignees, err = github.FetchAssignees()
		return err
	})

	withSpinner("Fetching labels...", func() error {
		var err error
		availableLabels, err = github.FetchLabels()
		return err
	})

	assigneeChoice := assigneeToChoice(filters.Assignee, availableAssignees)
	assigneeCustom := ""
	if assigneeChoice == "custom" {
		assigneeCustom = filters.Assignee
	}
	labelInput := filters.Label
	selectedLabels := splitCSV(filters.Label)
	limitInput := fmt.Sprintf("%d", filters.Limit)
	actionChoice := "save"

	groupFields := []huh.Field{
		huh.NewSelect[string]().
			Title("State").
			Options(
				huh.NewOption("open", "open"),
				huh.NewOption("closed", "closed"),
				huh.NewOption("all", "all"),
			).
			Value(&stateChoice),
		huh.NewSelect[string]().
			Title("Assignee").
			Options(assigneeOptions(availableAssignees)...).
			Value(&assigneeChoice),
		huh.NewInput().
			Title("Custom assignee").
			Placeholder("GitHub username").
			Value(&assigneeCustom).
			DescriptionFunc(func() string {
				if assigneeChoice == "custom" {
					return "Required when Custom is selected."
				}
				return ""
			}, &assigneeChoice),
	}

	if len(availableLabels) > 0 {
		labelOptions := make([]huh.Option[string], 0, len(availableLabels))
		for _, name := range availableLabels {
			labelOptions = append(labelOptions, huh.NewOption(name, name))
		}
		height := len(labelOptions)
		if height > 8 {
			height = 8 // Limit visible rows for long lists
		}
		groupFields = append(groupFields,
			huh.NewMultiSelect[string]().
				Title("Labels").
				Description("Space to toggle. Matches issues with ALL selected labels.").
				Options(labelOptions...).
				Height(height).
				Value(&selectedLabels),
		)
	} else {
		groupFields = append(groupFields,
			huh.NewInput().
				Title("Label").
				Placeholder("Label name (comma-separated) or blank for any").
				Value(&labelInput),
		)
	}

	groupFields = append(groupFields,
		huh.NewInput().
			Title("Limit").
			Placeholder(fmt.Sprintf("%d", filters.Limit)).
			Value(&limitInput),
		huh.NewSelect[string]().
			Title("Action").
			Options(
				huh.NewOption("Save filters", "save"),
				huh.NewOption("Reset to defaults", "reset"),
			).
			Value(&actionChoice),
	)

	form := huh.NewForm(huh.NewGroup(groupFields...)).WithTheme(huh.ThemeCatppuccin())

	if err := form.Run(); err != nil {
		return filters // Return original on error/cancel
	}

	if actionChoice == "reset" {
		return github.Filters{State: "open", Limit: 50}
	}

	if stateChoice != "" {
		filters.State = stateChoice
	}
	filters.Assignee = resolveAssignee(assigneeChoice, assigneeCustom)
	if len(availableLabels) > 0 {
		filters.Label = strings.Join(selectedLabels, ",")
	} else if labelInput != "" {
		filters.Label = strings.TrimSpace(labelInput)
	}
	if limitInput != "" {
		limit, err := strconv.Atoi(limitInput)
		if err == nil && limit > 0 {
			filters.Limit = limit
		}
	}

	return filters
}

// assigneeToChoice maps a stored assignee value to the matching select option
// key used by the filter form. assignees is the list of known repo assignees;
// any value found in that list is treated as a direct selection, otherwise
// non-standard values fall back to "custom".
func assigneeToChoice(assignee string, assignees []string) string {
	switch assignee {
	case "":
		return ""
	case "@me":
		return "@me"
	}
	for _, a := range assignees {
		if a == assignee {
			return assignee
		}
	}
	return "custom"
}

// assigneeOptions builds the dropdown options for the assignee field. It
// always includes Any, @me, Custom… and prepends the fetched repo assignees
// when available.
func assigneeOptions(assignees []string) []huh.Option[string] {
	opts := []huh.Option[string]{
		huh.NewOption("Any", ""),
		huh.NewOption("@me", "@me"),
	}
	for _, a := range assignees {
		opts = append(opts, huh.NewOption(a, a))
	}
	opts = append(opts, huh.NewOption("Custom…", "custom"))
	return opts
}

// resolveAssignee maps the assignee select choice plus optional custom value
// back to the filter string.
func resolveAssignee(choice, custom string) string {
	if choice == "custom" {
		return strings.TrimSpace(custom)
	}
	return choice
}

// splitCSV splits a comma-separated label string into a slice of trimmed values,
// dropping empties.
func splitCSV(s string) []string {
	if s == "" {
		return nil
	}
	parts := strings.Split(s, ",")
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		if v := strings.TrimSpace(p); v != "" {
			out = append(out, v)
		}
	}
	return out
}
