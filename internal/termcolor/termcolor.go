// Package termcolor gates ANSI colors for CLI output.
package termcolor

import (
	"io"
	"os"

	"golang.org/x/term"
)

// ANSI SGR codes (no external color dependency).
const (
	Reset = "\033[0m"
	Green = "\033[32m"
	Red   = "\033[31m"
)

// Enabled reports whether ANSI colors should be written to w.
//
// Off when NO_COLOR is set, or when w is not an *os.File.
// On when FORCE_COLOR is set, or when w is a terminal.
func Enabled(w io.Writer) bool {
	if os.Getenv("NO_COLOR") != "" {
		return false
	}
	f, ok := w.(*os.File)
	if !ok {
		return false
	}
	if os.Getenv("FORCE_COLOR") != "" {
		return true
	}
	return term.IsTerminal(int(f.Fd()))
}

// Wrap surrounds s with code/Reset when Enabled(w); otherwise returns s unchanged.
func Wrap(w io.Writer, code, s string) string {
	if !Enabled(w) {
		return s
	}
	return code + s + Reset
}
