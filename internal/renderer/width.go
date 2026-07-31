// Package renderer — terminal width detection.
package renderer

import (
	"os"
	"runtime"
	"strconv"

	"golang.org/x/term"
)

// DetectCols returns the terminal width in columns. Resolution order:
//
//  1. Environment variable $COLUMNS, if it parses as a positive integer.
//  2. term.GetSize on os.Stdout.
//  3. term.GetSize on the controlling terminal, /dev/tty.
//  4. Fallback to 80.
//
// Each step only commits when its result is strictly positive; anything else
// falls through to the next.
//
// Step 3 exists because of how this binary is actually run. Claude Code invokes
// the status-line command with stdout on a pipe so it can capture the output,
// so step 2 always fails there — and $COLUMNS is a shell variable that is not
// exported to child processes, so step 1 usually fails too. Every render would
// therefore lay out for 80 columns no matter how wide the terminal really is,
// which shows up as segments collapsing to their compact form and merged header
// rows splitting in two for no visible reason. The controlling terminal is
// still reachable through /dev/tty even when the standard streams are
// redirected, and asking it is the one way to get the real answer.
func DetectCols() int {
	if s := os.Getenv("COLUMNS"); s != "" {
		if n, err := strconv.Atoi(s); err == nil && n > 0 {
			return n
		}
	}
	if w, _, err := term.GetSize(int(os.Stdout.Fd())); err == nil && w > 0 {
		return w
	}
	if w, ok := controllingTerminalWidth(); ok {
		return w
	}
	return 80
}

// controllingTerminalWidth asks /dev/tty for its size.
//
// Windows has no /dev/tty; the open fails there and the caller falls back to
// the default, which is what it did before this step existed. A process with no
// controlling terminal at all — cron, a CI runner — fails the same way.
func controllingTerminalWidth() (int, bool) {
	if runtime.GOOS == "windows" {
		return 0, false
	}
	f, err := os.Open("/dev/tty")
	if err != nil {
		return 0, false
	}
	defer f.Close() //nolint:errcheck

	w, _, err := term.GetSize(int(f.Fd()))
	if err != nil || w <= 0 {
		return 0, false
	}
	return w, true
}
