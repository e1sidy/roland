package main

import (
	"fmt"
	"os"

	"github.com/e1sidy/slate"
)

// ANSI color codes.
const (
	colorReset  = "\033[0m"
	colorRed    = "\033[31m"
	colorGreen  = "\033[32m"
	colorYellow = "\033[33m"
	colorBlue   = "\033[34m"
	colorCyan   = "\033[36m"
	colorGray   = "\033[90m"
	colorBold   = "\033[1m"
)

// noColor returns true if color output should be suppressed.
func noColor() bool {
	if _, ok := os.LookupEnv("NO_COLOR"); ok {
		return true
	}
	return !isTerminal()
}

// isTerminal checks if stdout is a terminal.
func isTerminal() bool {
	fi, err := os.Stdout.Stat()
	if err != nil {
		return false
	}
	return fi.Mode()&os.ModeCharDevice != 0
}

// colorize wraps text in ANSI color codes if color is enabled.
func colorize(color, text string) string {
	if noColor() {
		return text
	}
	return color + text + colorReset
}

// bold wraps text in bold ANSI codes if color is enabled.
func bold(text string) string {
	return colorize(colorBold, text)
}

// colorStatus returns a colored status string.
func colorStatus(status slate.Status) string {
	text := string(status)
	switch status {
	case slate.StatusOpen:
		return colorize(colorBlue, text)
	case slate.StatusInProgress:
		return colorize(colorGreen, text)
	case slate.StatusBlocked:
		return colorize(colorRed, text)
	case slate.StatusDeferred:
		return colorize(colorYellow, text)
	case slate.StatusClosed:
		return colorize(colorGray, text)
	case slate.StatusCancelled:
		return colorize(colorGray, text)
	default:
		return text
	}
}

// colorPriority returns a colored priority string.
func colorPriority(p slate.Priority) string {
	text := fmt.Sprintf("P%d", p)
	switch p {
	case slate.P0:
		return colorize(colorRed+colorBold, text)
	case slate.P1:
		return colorize(colorRed, text)
	case slate.P2:
		return colorize(colorYellow, text)
	case slate.P3:
		return colorize(colorBlue, text)
	case slate.P4:
		return colorize(colorGray, text)
	default:
		return text
	}
}
