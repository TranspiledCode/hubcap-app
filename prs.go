// prs.go
package main

import (
	"fmt"
	"strconv"
	"strings"

	"hubcap/internal/github"

	"github.com/charmbracelet/huh"
)

// PRFilterVals holds the mutable values bound to the embedded PR filter form.
// Must be heap-allocated (use &PRFilterVals{}) for stable huh Value() pointers.
type PRFilterVals struct {
	State          string
	AssigneeChoice string
	AssigneeCustom string
	SelectedLabels []string
	LabelInput     string
	Draft          string
	ReviewStatus   string
	LimitInput     string
	ActionChoice   string
}

// InitPRFilterVals pre-populates vals from the current PR filter settings.
func InitPRFilterVals(vals *PRFilterVals, filters github.PRFilters, assignees []string) {
	vals.State = filters.State
	vals.AssigneeChoice = assigneeToChoice(filters.Assignee, assignees)
	vals.AssigneeCustom = ""
	if vals.AssigneeChoice == "custom" {
		vals.AssigneeCustom = filters.Assignee
	}
	vals.SelectedLabels = splitCSV(filters.Label)
	vals.LabelInput = filters.Label
	vals.Draft = filters.Draft
	vals.ReviewStatus = filters.ReviewStatus
	vals.LimitInput = fmt.Sprintf("%d", filters.Limit)
	vals.ActionChoice = "save"
}

// BuildPRFilterForm constructs a *huh.Form bound to vals. Call form.Init()
// to start it; route messages through form.Update(msg) inside your model.
func BuildPRFilterForm(vals *PRFilterVals, assignees []string, labels []string) *huh.Form {
	groupFields := []huh.Field{
		huh.NewSelect[string]().
			Title("State").
			Options(
				huh.NewOption("open", "open"),
				huh.NewOption("closed", "closed"),
				huh.NewOption("merged", "merged"),
				huh.NewOption("all", "all"),
			).
			Value(&vals.State),
		huh.NewSelect[string]().
			Title("Assignee").
			Options(assigneeOptions(assignees)...).
			Value(&vals.AssigneeChoice),
		huh.NewInput().
			Title("Custom assignee").
			Placeholder("GitHub username").
			Value(&vals.AssigneeCustom).
			DescriptionFunc(func() string {
				if vals.AssigneeChoice == "custom" {
					return "Required when Custom is selected."
				}
				return ""
			}, &vals.AssigneeChoice),
	}

	if len(labels) > 0 {
		labelOptions := make([]huh.Option[string], 0, len(labels))
		for _, name := range labels {
			labelOptions = append(labelOptions, huh.NewOption(name, name))
		}
		height := len(labelOptions)
		if height > 8 {
			height = 8
		}
		groupFields = append(groupFields,
			huh.NewMultiSelect[string]().
				Title("Labels").
				Description("Space to toggle. Matches PRs with ALL selected labels.").
				Options(labelOptions...).
				Height(height).
				Value(&vals.SelectedLabels),
		)
	} else {
		groupFields = append(groupFields,
			huh.NewInput().
				Title("Label").
				Placeholder("Label name (comma-separated) or blank for any").
				Value(&vals.LabelInput),
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
			Value(&vals.Draft),
		huh.NewInput().
			Title("Review status").
			Placeholder("approved, changes-requested, etc.").
			Value(&vals.ReviewStatus),
		huh.NewInput().
			Title("Limit").
			Placeholder("e.g. 50").
			Value(&vals.LimitInput),
		huh.NewSelect[string]().
			Title("Action").
			Options(
				huh.NewOption("Save filters", "save"),
				huh.NewOption("Reset to defaults", "reset"),
			).
			Value(&vals.ActionChoice),
	)

	return huh.NewForm(huh.NewGroup(groupFields...)).WithTheme(huh.ThemeCatppuccin())
}

// ResolvePRFilters reads the completed vals and returns an updated PRFilters.
func ResolvePRFilters(vals *PRFilterVals, current github.PRFilters, labels []string) github.PRFilters {
	if vals.ActionChoice == "reset" {
		return github.PRFilters{State: "open", Limit: 50}
	}
	filters := current
	if vals.State != "" {
		filters.State = vals.State
	}
	filters.Assignee = resolveAssignee(vals.AssigneeChoice, vals.AssigneeCustom)
	if len(labels) > 0 {
		filters.Label = strings.Join(vals.SelectedLabels, ",")
	} else if vals.LabelInput != "" {
		filters.Label = strings.TrimSpace(vals.LabelInput)
	} else {
		filters.Label = ""
	}
	filters.Draft = vals.Draft
	if vals.ReviewStatus != "" {
		filters.ReviewStatus = strings.TrimSpace(vals.ReviewStatus)
	}
	if vals.LimitInput != "" {
		if limit, err := strconv.Atoi(vals.LimitInput); err == nil && limit > 0 {
			filters.Limit = limit
		}
	}
	return filters
}
