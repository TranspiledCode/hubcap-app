// dashboard.go
package main

import (
	"bufio"
	"fmt"
	"os"
	"strconv"
	"strings"
	"sync"

	"hubcap/internal/github"

	"github.com/charmbracelet/huh"
)

// ── Data types ────────────────────────────────────────────────────────────────

type dashboardResult struct {
	reviewRequests  []github.PullRequest
	myPRs           []github.PullRequest
	assignedIssues  []github.Issue
	availableIssues []github.Issue
	errs            [4]error
}

// dashRow is one entry in the flat navigable list rendered on screen.
type dashRow struct {
	isHeader  bool
	sectionID int  // 0=reviewRequests, 1=myPRs, 2=assignedIssues, 3=availableIssues
	itemIdx   int  // index within section (-1 for headers)
	isIssue   bool // true = Issue row, false = PullRequest row
}

const (
	secReviewRequests = 0
	secMyPRs          = 1
	secAssigned       = 2
	secAvailable      = 3
)

var sectionNames = [4]string{
	"REVIEW REQUESTS",
	"MY OPEN PRs",
	"ASSIGNED TO ME",
	"AVAILABLE TO GRAB",
}

// sectionLen returns the item count for a given section.
func sectionLen(data dashboardResult, sec int) int {
	switch sec {
	case secReviewRequests:
		return len(data.reviewRequests)
	case secMyPRs:
		return len(data.myPRs)
	case secAssigned:
		return len(data.assignedIssues)
	case secAvailable:
		return len(data.availableIssues)
	}
	return 0
}

// buildRows constructs the flat navigable row list from dashboard data.
// Empty sections are omitted entirely. Collapsed sections show only their header.
func buildRows(data dashboardResult, collapsed [4]bool) []dashRow {
	var rows []dashRow
	for sec := 0; sec < 4; sec++ {
		count := sectionLen(data, sec)
		if count == 0 && data.errs[sec] == nil {
			continue // hidden when empty and no error
		}
		rows = append(rows, dashRow{isHeader: true, sectionID: sec, itemIdx: -1})
		if collapsed[sec] {
			continue
		}
		isIssue := sec == secAssigned || sec == secAvailable
		for i := 0; i < count; i++ {
			rows = append(rows, dashRow{isHeader: false, sectionID: sec, itemIdx: i, isIssue: isIssue})
		}
	}
	return rows
}

// ── Concurrent data fetch ─────────────────────────────────────────────────────

func fetchDashboard(cfg Config) dashboardResult {
	var wg sync.WaitGroup
	var result dashboardResult

	wg.Add(4)

	go func() {
		defer wg.Done()
		result.reviewRequests, result.errs[secReviewRequests] = github.FetchReviewRequests(25)
	}()
	go func() {
		defer wg.Done()
		result.myPRs, result.errs[secMyPRs] = github.FetchPRs(github.PRFilters{
			Author: "@me", State: "open", Limit: 25,
		})
	}()
	go func() {
		defer wg.Done()
		result.assignedIssues, result.errs[secAssigned] = github.FetchIssues(github.Filters{
			Assignee: "@me", State: "open", Limit: 25,
		})
	}()
	go func() {
		defer wg.Done()
		result.availableIssues, result.errs[secAvailable] = github.FetchIssues(cfg.AvailableFilter)
	}()

	wg.Wait()
	return result
}

// ── Navigation loop ───────────────────────────────────────────────────────────

func browseDashboard(reader *bufio.Reader, state *AppState, cfg *Config) string {
	var data dashboardResult
	var rows []dashRow
	collapsed := [4]bool{}
	needsRefresh := true

	for {
		if needsRefresh {
			clearScreen()
			renderHeader(state, false)
			fmt.Println("Loading...")
			data = fetchDashboard(*cfg)
			rows = buildRows(data, collapsed)
			state.DashboardStatus = dashboardStatus(data)
			needsRefresh = false
		}

		cursor := state.DashboardCursor
		if cursor >= len(rows) {
			cursor = 0
		}

		if err := enableRawMode(); err != nil {
			// Fallback: non-raw mode
			renderDashboard(state, data, rows, cursor, collapsed, false)
			input := prompt(reader, "Number to open, r refresh, c config, q quit: ")
			input = strings.TrimSpace(strings.ToLower(input))
			switch input {
			case "q", "quit", "b":
				return "quit"
			case "r":
				needsRefresh = true
				continue
			case "c":
				*cfg = configureHubcap(reader, state, *cfg)
				data = fetchDashboard(*cfg)
				rows = buildRows(data, collapsed)
				continue
			}
			num, err := strconv.Atoi(input)
			if err == nil {
				openDashboardItem(reader, state, data, rows, num)
			}
			continue
		}

		renderDashboard(state, data, rows, cursor, collapsed, true)

		buf := make([]byte, 3)
		n, err := os.Stdin.Read(buf)
		disableRawMode()
		if err != nil || n == 0 {
			return "quit"
		}

		key := string(buf[:n])
		switch key {
		case "q", "Q", "\x03", "\x1b":
			state.DashboardCursor = cursor
			return "quit"

		case "\t", "\x1b[Z":
			state.DashboardCursor = cursor
			state.ActiveTab = nextTab(state.ActiveTab)
			return ""

		case "1":
			state.ActiveTab = TabDashboard
			return ""
		case "2":
			state.ActiveTab = TabIssues
			return ""
		case "3":
			state.ActiveTab = TabPRs
			return ""

		case "r", "R":
			needsRefresh = true

		case "n", "N":
			clearScreen()
			renderHeader(state, false)
			fmt.Println("Ctrl+C to cancel.")
			github.RunCommandPassthrough("gh", "issue", "create")
			needsRefresh = true

		case "p", "P":
			clearScreen()
			renderHeader(state, false)
			fmt.Println("Ctrl+C to cancel.")
			github.RunCommandPassthrough("gh", "pr", "create")
			needsRefresh = true

		case "c", "C":
			*cfg = configureHubcap(reader, state, *cfg)
			needsRefresh = true

		case "\x1b[A": // up
			cursor--
			if cursor < 0 {
				cursor = len(rows) - 1
			}
			state.DashboardCursor = cursor

		case "\x1b[B": // down
			cursor++
			if cursor >= len(rows) {
				cursor = 0
			}
			state.DashboardCursor = cursor

		case "\x1b[D": // left — collapse current section
			if len(rows) > 0 {
				sec := rows[cursor].sectionID
				collapsed[sec] = true
				rows = buildRows(data, collapsed)
				// keep cursor on the section header
				for i, r := range rows {
					if r.isHeader && r.sectionID == sec {
						cursor = i
						break
					}
				}
				state.DashboardCursor = cursor
			}

		case "\r", "\n":
			if len(rows) == 0 {
				continue
			}
			row := rows[cursor]
			if row.isHeader {
				// toggle collapse
				collapsed[row.sectionID] = !collapsed[row.sectionID]
				rows = buildRows(data, collapsed)
				// keep cursor on header
				for i, r := range rows {
					if r.isHeader && r.sectionID == row.sectionID {
						cursor = i
						break
					}
				}
				state.DashboardCursor = cursor
			} else {
				// open item
				state.DashboardCursor = cursor
				openDashboardItemByRow(reader, state, data, row)
				needsRefresh = true
			}
		}
	}
}

// openDashboardItemByRow opens the detail view for the item described by row.
func openDashboardItemByRow(reader *bufio.Reader, state *AppState, data dashboardResult, row dashRow) {
	switch {
	case row.sectionID == secReviewRequests:
		viewPR(reader, state, data.reviewRequests[row.itemIdx].Number)
	case row.sectionID == secMyPRs:
		viewPR(reader, state, data.myPRs[row.itemIdx].Number)
	case row.sectionID == secAssigned:
		viewIssue(reader, state, data.assignedIssues[row.itemIdx].Number)
	case row.sectionID == secAvailable:
		viewIssue(reader, state, data.availableIssues[row.itemIdx].Number)
	}
}

// openDashboardItem finds an item by number across all sections and opens it.
func openDashboardItem(reader *bufio.Reader, state *AppState, data dashboardResult, rows []dashRow, number int) {
	for _, row := range rows {
		if row.isHeader {
			continue
		}
		switch row.sectionID {
		case secReviewRequests:
			if data.reviewRequests[row.itemIdx].Number == number {
				viewPR(reader, state, number)
				return
			}
		case secMyPRs:
			if data.myPRs[row.itemIdx].Number == number {
				viewPR(reader, state, number)
				return
			}
		case secAssigned:
			if data.assignedIssues[row.itemIdx].Number == number {
				viewIssue(reader, state, number)
				return
			}
		case secAvailable:
			if data.availableIssues[row.itemIdx].Number == number {
				viewIssue(reader, state, number)
				return
			}
		}
	}
	fmt.Printf("Item #%d not found in current view.\n", number)
	pause(reader)
}

// ensure fmt is used — will be used by render functions in later tasks.
var _ = fmt.Sprintf

// ── Rendering ─────────────────────────────────────────────────────────────────

func renderDashboard(state *AppState, data dashboardResult, rows []dashRow, cursor int, collapsed [4]bool, rawMode bool) {
	nl := "\n"
	cr := ""
	if rawMode {
		nl = "\033[K\r\n"
		fmt.Print("\033[H")
	} else {
		clearScreen()
	}
	renderHeader(state, rawMode)

	if len(rows) == 0 {
		fmt.Printf("No items to show. Press r to refresh.%s", nl)
		return
	}

	for i, row := range rows {
		sel := ""
		if i == cursor {
			sel = styleCyan.Render(">") + " "
		} else {
			sel = "  "
		}

		if row.isHeader {
			if i > 0 {
				fmt.Print(nl)
			}
			count := sectionLen(data, row.sectionID)
			errMark := ""
			if data.errs[row.sectionID] != nil {
				errMark = " ⚠ could not load"
				count = 0
			}
			arrow := "▾"
			if collapsed[row.sectionID] {
				arrow = "▸"
			}
			fmt.Printf("%s%s %s (%d)%s%s%s",
				sel, arrow, sectionNames[row.sectionID], count, errMark, cr, nl)
			continue
		}

		// item row
		switch {
		case row.sectionID == secReviewRequests || row.sectionID == secMyPRs:
			pr := data.reviewRequests
			if row.sectionID == secMyPRs {
				pr = data.myPRs
			}
			p := pr[row.itemIdx]
			indicator := stateIndicator(p.State, p.IsDraft)
			checks := summarizeChecks(p.StatusRollup)
			fmt.Printf("  %s%s #%-5d %-52s %s%s%s",
				sel, indicator, p.Number, truncate(p.Title, 52), checks, cr, nl)
		case row.isIssue:
			var issue github.Issue
			if row.sectionID == secAssigned {
				issue = data.assignedIssues[row.itemIdx]
			} else {
				issue = data.availableIssues[row.itemIdx]
			}
			indicator := stateIndicator(issue.State, false)
			labels := coloredLabelsCompact(issue.Labels, 24)
			fmt.Printf("  %s%s #%-5d %-48s %s%s%s",
				sel, indicator, issue.Number, truncate(cleanLine(issue.Title), 48), labels, cr, nl)
		}
	}

	fmt.Print(nl)
	stats := fmt.Sprintf("%d reviews · %d PRs · %d assigned · %d available",
		len(data.reviewRequests), len(data.myPRs), len(data.assignedIssues), len(data.availableIssues))
	if rawMode {
		fmt.Print(hintSep(true))
		fmt.Print(hintBar("↑↓", "move", "enter", "open", "←", "fold", "tab", "switch", "n", "issue", "p", "PR", "r", "refresh", "c", "config", "q", "quit") + "\033[K\r\n")
		fmt.Print(renderStatusBar(state, stats) + "\033[K\r\n")
		fmt.Print("\033[J")
	} else {
		fmt.Print("\n" + hintSep(false))
		fmt.Print(hintBar("number", "open", "n", "issue", "p", "PR", "r", "refresh", "c", "config", "q", "quit") + "\n")
		fmt.Print(renderStatusBar(state, stats) + "\n")
	}
}

// ── Config screen ─────────────────────────────────────────────────────────────

func configureHubcap(reader *bufio.Reader, state *AppState, cfg Config) Config {
	for {
		clearScreen()
		renderHeader(state, false)
		fmt.Println("Configure hubcap")
		fmt.Println()

		choice := menu(reader, []string{
			"Change \"Available to Grab\" filter",
			"Reset to defaults",
			"Back",
		})

		switch choice {
		case "Change \"Available to Grab\" filter":
			cfg.AvailableFilter = configureAvailableFilter(reader, state, cfg.AvailableFilter)
			if err := saveConfig(cfg); err != nil {
				fmt.Println(errorBox(fmt.Sprintf("Could not save config: %v", err)))
				pause(reader)
			}
		case "Reset to defaults":
			if !confirmAction(
				"Reset configuration to defaults?",
				"This will overwrite your current hubcap config.",
				"Reset",
			) {
				continue
			}
			cfg = defaultConfig()
			if err := saveConfig(cfg); err != nil {
				fmt.Println(errorBox(fmt.Sprintf("Could not save config: %v", err)))
				pause(reader)
			} else {
				fmt.Println(successBox("Reset to defaults."))
				pause(reader)
			}
		case "Back", "":
			return cfg
		}
	}
}

func configureAvailableFilter(reader *bufio.Reader, state *AppState, filters github.Filters) github.Filters {
	// Initialize with current values
	stateChoice := filters.State
	availableAssignees, _ := github.FetchAssignees()
	assigneeChoice := assigneeToChoice(filters.Assignee, availableAssignees)
	assigneeCustom := ""
	if assigneeChoice == "custom" {
		assigneeCustom = filters.Assignee
	}
	labelInput := filters.Label
	selectedLabels := splitCSV(filters.Label)
	limitInput := fmt.Sprintf("%d", filters.Limit)
	actionChoice := "save"

	availableLabels, _ := github.FetchLabels()

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
		return github.Filters{State: "open", Limit: 25}
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

func dashboardStatus(data dashboardResult) string {
	pipe := styleGray.Render("  ·  ")
	item := func(n int, singular, plural string) string {
		label := plural
		if n == 1 {
			label = singular
		}
		return styleCyan.Render(strconv.Itoa(n)) + styleGray.Render(" "+label)
	}
	return strings.Join([]string{
		item(len(data.reviewRequests), "review request", "review requests"),
		item(len(data.myPRs), "open PR", "open PRs"),
		item(len(data.assignedIssues), "assigned", "assigned"),
		item(len(data.availableIssues), "available", "available"),
	}, pipe)
}
