// diag.go implements `cc-probeline diag` — what the renderer sees.
package main

import (
	"fmt"
	"io"
	"os"
	"runtime"
	"strconv"

	"github.com/labzink/cc-probeline/internal/config"
	"github.com/labzink/cc-probeline/internal/renderer"
	"golang.org/x/term"
)

// runDiag reports how terminal width resolves, step by step.
//
// Width is the one input the renderer cannot get wrong quietly: when it is
// wrong the line still renders, just laid out for a terminal that does not
// exist — segments collapse to their compact form and a merged header splits in
// two. Because Claude Code runs this command with stdout on a pipe, the usual
// checks fail there while succeeding when you run it by hand, which makes the
// problem look intermittent. This prints each step so the difference is visible
// rather than inferred.
func runDiag(args []string) int {
	return runDiagImpl(args, os.Stdout, os.Stderr)
}

func runDiagImpl(_ []string, stdout, _ io.Writer) int {
	fmt.Fprintln(stdout, "cc-probeline diag —", versionString())
	fmt.Fprintln(stdout)
	fmt.Fprintln(stdout, "Terminal width, in the order it is resolved:")

	// 1. $COLUMNS.
	if s := os.Getenv("COLUMNS"); s != "" {
		if n, err := strconv.Atoi(s); err == nil && n > 0 {
			fmt.Fprintf(stdout, "  1. $COLUMNS            %d\n", n)
		} else {
			fmt.Fprintf(stdout, "  1. $COLUMNS            set to %q, not a positive number\n", s)
		}
	} else {
		fmt.Fprintln(stdout, "  1. $COLUMNS            not set  (shells do not export it to children)")
	}

	// 2. stdout.
	if w, _, err := term.GetSize(int(os.Stdout.Fd())); err == nil && w > 0 {
		fmt.Fprintf(stdout, "  2. stdout              %d\n", w)
	} else {
		fmt.Fprintln(stdout, "  2. stdout              not a terminal  (a pipe when Claude Code calls us)")
	}

	// 3. controlling terminal.
	if runtime.GOOS == "windows" {
		fmt.Fprintln(stdout, "  3. /dev/tty            n/a on Windows")
	} else if f, err := os.Open("/dev/tty"); err != nil {
		fmt.Fprintln(stdout, "  3. /dev/tty            cannot open  (no controlling terminal)")
	} else {
		w, _, gerr := term.GetSize(int(f.Fd()))
		f.Close() //nolint:errcheck
		if gerr == nil && w > 0 {
			fmt.Fprintf(stdout, "  3. /dev/tty            %d\n", w)
		} else {
			fmt.Fprintln(stdout, "  3. /dev/tty            opened, but size unavailable")
		}
	}
	fmt.Fprintln(stdout, "  4. fallback            80")

	cwd, _ := os.Getwd()
	cfg, _, _ := config.LoadCascade(cwd)

	fmt.Fprintln(stdout)
	if cfg.General.Columns > 0 {
		fmt.Fprintf(stdout, "  [general].columns      %d  — overrides all of the above\n", cfg.General.Columns)
		fmt.Fprintf(stdout, "  effective              %d\n", cfg.General.Columns)
	} else {
		fmt.Fprintf(stdout, "  [general].columns      0  (auto)\n")
		fmt.Fprintf(stdout, "  effective              %d\n", renderer.DetectCols())
	}

	fmt.Fprintln(stdout)
	fmt.Fprintln(stdout, "Run this from your shell and the numbers above describe your shell, not the")
	fmt.Fprintln(stdout, "status line: Claude Code invokes us with different streams. If the line keeps")
	fmt.Fprintln(stdout, "wrapping at a width that is not yours, set it outright:")
	fmt.Fprintln(stdout)
	fmt.Fprintln(stdout, "    cc-probeline columns <width>      # 0 restores auto-detect")

	return 0
}
