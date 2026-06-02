// create_forms.go
package main

import (
	"hubcap/internal/github"

	"github.com/charmbracelet/huh"
)

func runCreateIssueForm() error {
	clearScreen()
	var title, body string

	form := huh.NewForm(
		huh.NewGroup(
			huh.NewInput().
				Title("New Issue — Title").
				Placeholder("Short description").
				Value(&title),
			huh.NewText().
				Title("Body").
				Placeholder("Describe the issue (optional)").
				CharLimit(4000).
				Value(&body),
		),
	).WithTheme(huh.ThemeCatppuccin())

	if err := form.Run(); err != nil {
		return nil // cancelled
	}
	if title == "" {
		return nil
	}
	return github.CreateIssue(title, body, nil)
}

func runCreatePRForm() error {
	clearScreen()
	var title, body, base string
	var draft bool

	base = "main"

	form := huh.NewForm(
		huh.NewGroup(
			huh.NewInput().
				Title("New PR — Title").
				Placeholder("Short description").
				Value(&title),
			huh.NewText().
				Title("Body").
				Placeholder("Describe the changes (optional)").
				CharLimit(4000).
				Value(&body),
			huh.NewInput().
				Title("Base branch").
				Placeholder("main").
				Value(&base),
			huh.NewConfirm().
				Title("Draft PR?").
				Value(&draft),
		),
	).WithTheme(huh.ThemeCatppuccin())

	if err := form.Run(); err != nil {
		return nil // cancelled
	}
	if title == "" {
		return nil
	}
	return github.CreatePR(title, body, base, draft)
}
