// Package cli provides terminal output helpers for spec-forge CLI.
package cli

import (
	"fmt"
	"io"
	"os"

	"github.com/fatih/color"
)

var (
	green  = color.New(color.FgGreen).FprintfFunc()
	red    = color.New(color.FgRed, color.Bold).FprintfFunc()
	cyan   = color.New(color.FgCyan).FprintfFunc()
	yellow = color.New(color.FgYellow).FprintfFunc()
	dim    = color.New(color.Faint).FprintfFunc()
)

func init() {
	initColorState()
}

// initColorState configures color output based on the NO_COLOR environment variable.
// Per the NO_COLOR convention (https://no-color.org/), mere presence of the variable
// disables color output regardless of its value (including empty string).
func initColorState() {
	if _, ok := os.LookupEnv("NO_COLOR"); ok {
		color.NoColor = true
	}
}

// ColorEnabled reports whether colored output is active.
func ColorEnabled() bool {
	return !color.NoColor
}

// Successf prints a green success message with checkmark prefix.
func Successf(w io.Writer, format string, args ...any) {
	msg := fmt.Sprintf(format, args...)
	green(w, "✅ %s\n", msg)
}

// Skipf prints a dim skip message with skip prefix.
func Skipf(w io.Writer, format string, args ...any) {
	msg := fmt.Sprintf(format, args...)
	dim(w, "⏭️  %s\n", msg)
}

// Errorf prints a red error message.
func Errorf(w io.Writer, format string, args ...any) {
	msg := fmt.Sprintf(format, args...)
	red(w, "❌ %s\n", msg)
}

// Hintf prints a cyan hint prefix with yellow hint content.
func Hintf(w io.Writer, format string, args ...any) {
	msg := fmt.Sprintf(format, args...)
	cyan(w, "💡 Hint: ")
	yellow(w, "%s\n", msg)
}

// Statusf prints a neutral status message.
func Statusf(w io.Writer, format string, args ...any) {
	fmt.Fprintf(w, format+"\n", args...)
}
