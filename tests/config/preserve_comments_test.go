// Package config_test — every setter must edit its value in place, leaving the
// rest of the user's config.toml byte-for-byte intact.
//
// This file is the regression guard for a promise the project makes in its
// changelog and breaks easily: a setter that round-trips the file through
// Config rewrites the whole thing, dropping comments and normalising key order.
// The user is invited to read and annotate this file by hand, so losing their
// notes to change one boolean is not an acceptable trade.
package config_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/labzink/cc-probeline/internal/config"
)

// annotatedSeed is a config in the shape a user would actually keep: comments
// above sections, inline comments after values, aligned assignments, blank
// lines, and a key order that is not the struct's.
const annotatedSeed = `version = 1

# ─── how much room the line may take ───
[general]
tutorial_hints = false      # learned these already
table_rows     = 3          # last few turns, still priced
bar_width      = 8          # 6.25% resolution
effort_style   = "word"
mode           = 'standard'

# segments I actually read
[widgets]
ctx   = true                # the one I watch
cost  = false
email = false
`

// writeSeed writes annotatedSeed to a fresh temp config and returns its path.
func writeSeed(t *testing.T) string {
	t.Helper()
	p := filepath.Join(t.TempDir(), "config.toml")
	if err := os.WriteFile(p, []byte(annotatedSeed), 0o644); err != nil {
		t.Fatalf("seed: %v", err)
	}
	return p
}

// readFile returns the file contents as a string.
func readFile(t *testing.T, p string) string {
	t.Helper()
	b, err := os.ReadFile(p)
	if err != nil {
		t.Fatalf("read: %v", err)
	}
	return string(b)
}

// T-PC1: every setter keeps the surrounding file intact. Each case changes one
// value and asserts that the comments, the blank lines and the untouched
// assignments all survive verbatim.
func TestSetters_PreserveCommentsAndLayout(t *testing.T) {
	cases := []struct {
		name string
		set  func(path string) error
		want string // the line the setter should have produced
	}{
		{"table_rows", func(p string) error { return config.SetTableRows(p, 7) }, "table_rows     = 7"},
		{"bar_width", func(p string) error { return config.SetBarWidth(p, 10) }, "bar_width      = 10"},
		{"table_legend", func(p string) error { return config.SetTableLegend(p, false) }, "table_legend = false"},
		{"header_merge", func(p string) error { return config.SetHeaderMerge(p, true) }, "header_merge = true"},
		{"effort_style", func(p string) error { return config.SetEffortStyle(p, "glyph") }, `effort_style   = "glyph"`},
		{"mode", func(p string) error { return config.SetMode(p, "super-compact") }, `mode           = "super-compact"`},
		{"widget", func(p string) error { return config.SetWidget(p, "cost", true) }, "cost  = true"},
		{"hints", func(p string) error { return config.SetTutorialHints(p, true) }, "tutorial_hints = true"},
	}

	// Fragments that must survive every single edit.
	survivors := []string{
		"# ─── how much room the line may take ───",
		"# segments I actually read",
		"# learned these already",
		"# last few turns, still priced",
		"# the one I watch",
		"[general]",
		"[widgets]",
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			p := writeSeed(t)

			if err := c.set(p); err != nil {
				t.Fatalf("setter: %v", err)
			}

			got := readFile(t, p)
			for _, s := range survivors {
				if !strings.Contains(got, s) {
					t.Errorf("edit dropped %q from the file:\n%s", s, got)
				}
			}
			if !strings.Contains(got, c.want) {
				t.Errorf("expected line %q in:\n%s", c.want, got)
			}
			// The file must stay loadable.
			if _, errs := config.Load(p); len(errs) > 0 {
				for _, e := range errs {
					if e.Severity == config.SeverityError {
						t.Errorf("file no longer parses: %s", e.Message)
					}
				}
			}
		})
	}
}

// T-PC2: an inline comment on the edited line itself is kept — only the value
// between '=' and the '#' changes.
func TestSetters_KeepInlineCommentOnEditedLine(t *testing.T) {
	p := writeSeed(t)

	if err := config.SetTableRows(p, 12); err != nil {
		t.Fatalf("SetTableRows: %v", err)
	}

	got := readFile(t, p)
	if !strings.Contains(got, "# last few turns, still priced") {
		t.Errorf("inline comment on the edited line was dropped:\n%s", got)
	}
	if strings.Contains(got, "table_rows     = 3") {
		t.Errorf("old value still present:\n%s", got)
	}
}

// T-PC3: a key absent from the file is inserted into its table rather than
// appended at the end, where it would silently belong to whichever table
// happens to be last.
func TestSetters_InsertMissingKeyIntoItsTable(t *testing.T) {
	p := writeSeed(t)

	if err := config.SetHeaderLine(p, 1, false); err != nil {
		t.Fatalf("SetHeaderLine: %v", err)
	}

	cfg, errs := config.Load(p)
	for _, e := range errs {
		if e.Severity == config.SeverityError {
			t.Fatalf("load after insert: %s", e.Message)
		}
	}
	if cfg.General.HeaderLine1 {
		t.Error("header_line1 did not take effect — the key likely landed in the wrong table")
	}
	// And it must not have leaked into [widgets].
	got := readFile(t, p)
	widgetsIdx := strings.Index(got, "[widgets]")
	keyIdx := strings.Index(got, "header_line1")
	if widgetsIdx >= 0 && keyIdx > widgetsIdx {
		t.Errorf("header_line1 was inserted after [widgets]:\n%s", got)
	}
}

// T-PC4: a broken config is reported, never overwritten — a typo in a
// hand-edited file must not cost the user the rest of it.
func TestSetters_RefuseToClobberBrokenFile(t *testing.T) {
	p := filepath.Join(t.TempDir(), "config.toml")
	broken := "version = 1\n\n[general\ntable_rows = 3\n" // missing ]
	if err := os.WriteFile(p, []byte(broken), 0o644); err != nil {
		t.Fatalf("seed: %v", err)
	}

	if err := config.SetTableRows(p, 9); err == nil {
		t.Fatal("SetTableRows accepted a broken config")
	}
	if got := readFile(t, p); got != broken {
		t.Errorf("broken file was modified:\ngot  %q\nwant %q", got, broken)
	}
}

// T-PC5: a missing file is created with just the version marker and the one
// setting asked for — not a generated full template.
func TestSetters_CreateMinimalFile(t *testing.T) {
	p := filepath.Join(t.TempDir(), "nested", "config.toml")

	if err := config.SetBarWidth(p, 6); err != nil {
		t.Fatalf("SetBarWidth: %v", err)
	}

	got := readFile(t, p)
	if !strings.Contains(got, "[general]") || !strings.Contains(got, "bar_width = 6") {
		t.Errorf("minimal file missing the setting:\n%s", got)
	}
	if strings.Contains(got, "[widgets]") || strings.Contains(got, "[thresholds]") {
		t.Errorf("a full template was written instead of a minimal file:\n%s", got)
	}
}

// T-PC6: out-of-range values are rejected with the file left alone.
func TestSetBarWidth_RejectsOutOfRange(t *testing.T) {
	p := writeSeed(t)
	before := readFile(t, p)

	for _, w := range []int{2, 21, -1} {
		if err := config.SetBarWidth(p, w); err == nil {
			t.Errorf("SetBarWidth(%d) was accepted", w)
		}
	}
	if got := readFile(t, p); got != before {
		t.Error("a rejected value still modified the file")
	}
}
