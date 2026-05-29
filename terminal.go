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
)

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

func menu(reader *bufio.Reader, options []string) string {
	if len(options) == 0 {
		return ""
	}

	selected := 0

	if err := enableRawMode(); err != nil {
		return numberedMenu(reader, options)
	}
	defer disableRawMode()

	renderMenu := func() {
		fmt.Print("\033[?25l")
		for index, option := range options {
			prefix := "  "
			if index == selected {
				prefix = "> "
			}
			fmt.Printf("%s%s\033[K\r\n", prefix, option)
		}
		fmt.Print("\r\n")
		fmt.Print("↑/↓ navigate • enter submit • 1-9 jump • q quit/back\033[K")
		fmt.Printf("\033[%dF", len(options)+1)
	}

	renderMenu()
	defer fmt.Print("\033[?25h")

	buffer := make([]byte, 3)

	for {
		n, err := os.Stdin.Read(buffer)
		if err != nil || n == 0 {
			return ""
		}
		key := string(buffer[:n])
		switch key {
		case "\r", "\n":
			fmt.Print("\033[?25h")
			fmt.Printf("\033[%dB\r\n", len(options)+1)
			return options[selected]
		case "q", "Q", "\x03", "\x1b":
			fmt.Print("\033[?25h")
			fmt.Printf("\033[%dB\r\n", len(options)+1)
			return ""
		case "\x1b[A":
			selected--
			if selected < 0 {
				selected = len(options) - 1
			}
			renderMenu()
		case "\x1b[B":
			selected++
			if selected >= len(options) {
				selected = 0
			}
			renderMenu()
		default:
			if len(key) == 1 && key[0] >= '1' && key[0] <= '9' {
				index := int(key[0] - '1')
				if index >= 0 && index < len(options) {
					selected = index
					renderMenu()
				}
			}
		}
	}
}

func numberedMenu(reader *bufio.Reader, options []string) string {
	for index, option := range options {
		fmt.Printf("%d) %s\n", index+1, option)
	}
	for {
		input := strings.TrimSpace(prompt(reader, "Choose: "))
		if input == "" {
			return ""
		}
		number, err := strconv.Atoi(input)
		if err != nil || number < 1 || number > len(options) {
			fmt.Println("Enter a number from the menu.")
			continue
		}
		return options[number-1]
	}
}
