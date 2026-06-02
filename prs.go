// prs.go
package main

import (
	"fmt"
	"strconv"
	"strings"

	"hubcap/internal/github"

	"github.com/charmbracelet/huh"
)

func configurePRFilters(state *AppState) github.PRFilters {
	filters := state.PRFilters

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
	draftChoice := filters.Draft
	reviewStatusInput := filters.ReviewStatus
	limitInput := fmt.Sprintf("%d", filters.Limit)
	actionChoice := "save"

	groupFields := []huh.Field{
		huh.NewSelect[string]().
			Title("State").
			Options(
				huh.NewOption("open", "open"),
				huh.NewOption("closed", "closed"),
				huh.NewOption("merged", "merged"),
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
				Description("Space to toggle. Matches PRs with ALL selected labels.").
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
		huh.NewSelect[string]().
			Title("Draft").
			Options(
				huh.NewOption("all", ""),
				huh.NewOption("draft only", "true"),
				huh.NewOption("non-draft only", "false"),
			).
			Value(&draftChoice),
		huh.NewInput().
			Title("Review status").
			Placeholder("approved, changes-requested, etc.").
			Value(&reviewStatusInput),
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
		return github.PRFilters{State: "open", Limit: 50}
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
	if draftChoice != "" {
		filters.Draft = draftChoice
	}
	if reviewStatusInput != "" {
		filters.ReviewStatus = strings.TrimSpace(reviewStatusInput)
	}
	if limitInput != "" {
		limit, err := strconv.Atoi(limitInput)
		if err == nil && limit > 0 {
			filters.Limit = limit
		}
	}

	return filters
}
