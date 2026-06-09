// terminal.go
package main

import (
	"bufio"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"runtime"
	"strconv"
	"strings"

	"github.com/charmbracelet/huh"
)

// confirmAction prompts the user with a styled yes/no confirmation dialog.
// Returns true if the user confirms, false otherwise (including on error).
func confirmAction(title, description string, affirmative string) bool {
	confirmed := false
	form := huh.NewForm(
		huh.NewGroup(
			huh.NewConfirm().
				Title(title).
				Description(description).
				Affirmative(affirmative).
				Negative("Cancel").
				Value(&confirmed),
		),
	).WithTheme(huh.ThemeCatppuccin())
	if err := form.Run(); err != nil {
		return false
	}
	return confirmed
}

func enableRawMode() error {
	cmd := exec.Command("stty", "raw", "-echo")
	cmd.Stdin = os.Stdin
	return cmd.Run()
}

func disableRawMode() {
	cmd := exec.Command("stty", "-raw", "echo")
	cmd.Stdin = os.Stdin
	_ = cmd.Run()
}

func prompt(reader *bufio.Reader, label string) string {
	fmt.Print(label)
	value, err := reader.ReadString('\n')
	if err != nil {
		return ""
	}
	return strings.TrimRight(value, "\r\n")
}

func pause(reader *bufio.Reader) {
	fmt.Println()
	fmt.Print("Press Enter to continue...")
	_, _ = reader.ReadString('\n')
}

func promptBranchName(reader *bufio.Reader, defaultName string) (string, bool) {
	fmt.Printf("Branch name [%s]: ", defaultName)
	value, err := reader.ReadString('\n')
	if err != nil {
		return "", false
	}
	value = strings.TrimRight(value, "\r\n")
	if strings.TrimSpace(value) == "" {
		value = defaultName
	}
	if len(value) > 45 {
		fmt.Printf("Name is %d chars (max 45). Try again.\n", len(value))
		return "", false
	}
	return value, true
}

func clearScreen() {
	fmt.Print("\033[H\033[2J")
}

func termSize() (rows, cols int) {
	cmd := exec.Command("stty", "size")
	cmd.Stdin = os.Stdin
	out, err := cmd.Output()
	if err != nil {
		return 40, 80
	}
	parts := strings.Fields(strings.TrimSpace(string(out)))
	if len(parts) != 2 {
		return 40, 80
	}
	r, err1 := strconv.Atoi(parts[0])
	c, err2 := strconv.Atoi(parts[1])
	if err1 != nil || err2 != nil || r <= 0 || c <= 0 {
		return 40, 80
	}
	return r, c
}

func bodyBudget(fixedLines, menuItems int) (visualRows, termCols int) {
	rows, cols := termSize()
	budget := rows - fixedLines - (menuItems + 2) - 2
	if budget < 3 {
		budget = 3
	}
	return budget, cols
}

func require(name string) error {
	_, err := exec.LookPath(name)
	if err != nil {
		return fmt.Errorf("%s is required but was not found in PATH", name)
	}
	return nil
}

func copyText(text string) error {
	switch runtime.GOOS {
	case "darwin":
		cmd := exec.Command("pbcopy")
		cmd.Stdin = strings.NewReader(text)
		return cmd.Run()
	case "linux":
		if _, err := exec.LookPath("wl-copy"); err == nil {
			cmd := exec.Command("wl-copy")
			cmd.Stdin = strings.NewReader(text)
			return cmd.Run()
		}
		if _, err := exec.LookPath("xclip"); err == nil {
			cmd := exec.Command("xclip", "-selection", "clipboard")
			cmd.Stdin = strings.NewReader(text)
			return cmd.Run()
		}
	}
	return errors.New("no clipboard command available")
}

// emptyTabAction reads a single keypress for empty-list and error screens.
// switchTarget is the tab to switch to on Tab/Shift-Tab.
// Returns "quit", "switch", "filters", or "" (retry/refresh).
func emptyTabAction(reader *bufio.Reader, state *AppState, switchTarget TabID) string {
	if err := enableRawMode(); err != nil {
		input := prompt(reader, "> ")
		switch strings.TrimSpace(strings.ToLower(input)) {
		case "q", "quit", "b":
			return "quit"
		case "f":
			return "filters"
		}
		return ""
	}
	defer disableRawMode()
	var buf [4]byte
	n, _ := os.Stdin.Read(buf[:])
	key := string(buf[:n])
	switch key {
	case "\t", "\x1b[Z":
		state.ActiveTab = switchTarget
		return "switch"
	case "f", "F":
		return "filters"
	case "q", "Q", "b", "B", "\x03", "\x1b":
		return "quit"
	}
	return ""
}

