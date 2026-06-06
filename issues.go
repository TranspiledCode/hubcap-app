// issues.go
package main

import (
	"fmt"
	"strconv"
	"strings"

	"hubcap/internal/github"

	"github.com/charmbracelet/huh"
)

// IssueFilterVals holds the mutable values bound to the embedded filter form.
// It must be heap-allocated (use &IssueFilterVals{}) so huh's Value() pointers
// remain stable across BubbleTea value-receiver model copies.
type IssueFilterVals struct {
	State          string
	AssigneeChoice string
	AssigneeCustom string
	SelectedLabels []string
	LabelInput     string
	LimitInput     string
	ActionChoice   string
}

// InitIssueFilterVals pre-populates vals from the current filter settings so
// the form opens with the user's existing selections.
func InitIssueFilterVals(vals *IssueFilterVals, filters github.Filters, assignees []string) {
	vals.State = filters.State
	vals.AssigneeChoice = assigneeToChoice(filters.Assignee, assignees)
	vals.AssigneeCustom = ""
	if vals.AssigneeChoice == "custom" {
		vals.AssigneeCustom = filters.Assignee
	}
	vals.SelectedLabels = splitCSV(filters.Label)
	vals.LabelInput = filters.Label
	vals.LimitInput = fmt.Sprintf("%d", filters.Limit)
	vals.ActionChoice = "save"
}

// BuildIssueFilterForm constructs a *huh.Form bound to vals. Call form.Init()
// to start it; route messages through form.Update(msg) inside your model.
func BuildIssueFilterForm(vals *IssueFilterVals, assignees []string, labels []string) *huh.Form {
	groupFields := []huh.Field{
		huh.NewSelect[string]().
			Title("State").
			Options(
				huh.NewOption("open", "open"),
				huh.NewOption("closed", "closed"),
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
				Description("Space to toggle. Matches issues with ALL selected labels.").
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

// ResolveIssueFilters reads the completed vals and returns an updated Filters.
func ResolveIssueFilters(vals *IssueFilterVals, current github.Filters, labels []string) github.Filters {
	if vals.ActionChoice == "reset" {
		return github.Filters{State: "open", Limit: 50}
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
	if vals.LimitInput != "" {
		if limit, err := strconv.Atoi(vals.LimitInput); err == nil && limit > 0 {
			filters.Limit = limit
		}
	}
	return filters
}

// ── Shared form helpers ───────────────────────────────────────────────────────

// assigneeToChoice maps a stored assignee value to the matching select option.
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

// assigneeOptions builds the dropdown options for the assignee field.
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

// splitCSV splits a comma-separated label string into trimmed, non-empty values.
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
