// Package statusline_test — Phase 7.5 vertical-footprint toggles.
//
// These tests assert on the rendered line COUNT and on the presence of specific
// glyphs, because the whole point of the feature is how many terminal rows the
// status line occupies. They run through Assembler.Render (the production path)
// rather than calling the renderer directly, so a toggle that is parsed but
// never wired would still fail here.
package statusline_test

import (
	"strings"
	"testing"
	"time"

	"github.com/labzink/cc-probeline/internal/format"
	"github.com/labzink/cc-probeline/internal/mode"
	"github.com/labzink/cc-probeline/internal/parser"
	"github.com/labzink/cc-probeline/internal/probes"
	"github.com/labzink/cc-probeline/internal/renderer"
	"github.com/labzink/cc-probeline/internal/statusline"
	"github.com/labzink/cc-probeline/internal/stdin"
)

// chromeConfig returns a Config whose header probes actually render something
// (a zero-value Config has every widget switched off, which would make both
// header lines empty and therefore indistinguishable from a hidden line).
// Tutorial hints are off so the rotating tip never perturbs the line count.
func chromeConfig(mutate func(*probes.Config)) probes.Config {
	c := probes.Config{
		ModelEnabled:   true,
		ProjectEnabled: true,
		HideHints:      true,
	}
	if mutate != nil {
		mutate(&c)
	}
	return c
}

// chromeAssembler builds a Standard-mode Assembler carrying cfg.
func chromeAssembler(cfg probes.Config) *statusline.Assembler {
	return &statusline.Assembler{
		Mode:   mode.Standard,
		Theme:  renderer.Theme{AnsiEnabled: false},
		Cols:   120,
		Config: cfg,
	}
}

// chromeData builds a two-turn, single-group session — enough to produce a
// table with one anchor (notch) row — plus the stdin fields the two header
// probes need (model → line1, cwd → line0).
func chromeData() probes.Data {
	base := time.Date(2026, 6, 1, 12, 0, 0, 0, time.UTC)

	older := parser.Turn{
		Index:     1,
		Role:      "orch",
		Model:     "claude-sonnet-4-6",
		UUID:      "uuid-1",
		GroupID:   1,
		Timestamp: base,
		Tokens:    parser.TokenCounts{Output: 100},
		ToolUse:   "ToolA",
	}
	newer := parser.Turn{
		Index:     2,
		Role:      "orch",
		Model:     "claude-sonnet-4-6",
		UUID:      "uuid-2",
		GroupID:   1,
		Timestamp: base.Add(20 * time.Second),
		Tokens:    parser.TokenCounts{Output: 200},
		ToolUse:   "ToolB",
	}

	return probes.Data{
		Stdin: stdin.Payload{
			Model: stdin.Model{ID: "claude-sonnet-4-6", Name: "Sonnet 4.6"},
			Cwd:   "/home/dev/demo-project",
		},
		Session: &parser.SessionStats{
			Turns:     []parser.Turn{newer, older}, // newest-first
			TurnCount: 2,
		},
		SessionID:    "chrome-session",
		Now:          base.Add(30 * time.Second),
		TerminalCols: 120,
	}
}

// renderLines runs the assembler and returns the output lines with the render
// markers stripped, so assertions compare what the user actually sees.
func renderLines(t *testing.T, cfg probes.Config) []string {
	t.Helper()
	t.Setenv("XDG_CACHE_HOME", t.TempDir())
	t.Setenv("HOME", t.TempDir())

	out := chromeAssembler(cfg).Render(chromeData())
	if out == "" {
		return nil
	}
	lines := strings.Split(out, "\n")
	for i := range lines {
		lines[i] = format.StripMarkers(lines[i])
	}
	return lines
}

// tableBlock returns only the lines belonging to the per-turn table, i.e.
// everything after the two header lines.
func tableBlock(t *testing.T, cfg probes.Config) []string {
	t.Helper()
	lines := renderLines(t, cfg)
	if len(lines) < 2 {
		t.Fatalf("expected the two header lines, got %d: %q", len(lines), lines)
	}
	return lines[2:]
}

// ─── T-CT0: fixture sanity ───────────────────────────────────────────────────

// Guards every count-based test below: the baseline must be two header lines
// plus a framed table with a legend, and no hint row.
func TestChrome_Baseline_HasHeadersAndFullTable(t *testing.T) {
	lines := renderLines(t, chromeConfig(func(c *probes.Config) { c.TableRows = 2 }))

	// 2 headers + top border + 2 data rows + legend separator + legend + bottom.
	if len(lines) != 8 {
		t.Fatalf("baseline should be 8 lines, got %d:\n%s", len(lines), strings.Join(lines, "\n"))
	}
	joined := strings.Join(lines, "\n")
	for _, want := range []string{"demo-project", "sonnet-4-6", "┌", "└", "~cost"} {
		if !strings.Contains(joined, want) {
			t.Errorf("baseline is missing %q:\n%s", want, joined)
		}
	}
}

// ─── T-CT1: legend ───────────────────────────────────────────────────────────

// The legend row and its ├─┼─┤ separator are worth two terminal rows; hiding
// them must remove exactly two lines and take the column labels with them.
func TestChrome_LegendOff_RemovesTwoLines(t *testing.T) {
	on := renderLines(t, chromeConfig(func(c *probes.Config) { c.TableRows = 2 }))
	off := renderLines(t, chromeConfig(func(c *probes.Config) {
		c.TableRows = 2
		c.HideTableLegend = true
	}))

	if got := len(on) - len(off); got != 2 {
		t.Errorf("hiding the legend freed %d lines, want 2 (on=%d off=%d)", got, len(on), len(off))
	}
	if strings.Contains(strings.Join(off, "\n"), "~cost") {
		t.Error("legend labels still present after HideTableLegend")
	}
	// The data rows themselves must survive.
	if !strings.Contains(strings.Join(off, "\n"), "ToolB") {
		t.Error("hiding the legend also removed the data rows")
	}
}

// ─── T-CT2: frame ────────────────────────────────────────────────────────────

// Frame off drops the top border, the bottom border and the legend separator
// (a frame element whose ├ ┤ corners only line up with the outer bars).
func TestChrome_FrameOff_RemovesBorders(t *testing.T) {
	on := renderLines(t, chromeConfig(func(c *probes.Config) { c.TableRows = 2 }))
	off := renderLines(t, chromeConfig(func(c *probes.Config) {
		c.TableRows = 2
		c.HideTableFrame = true
	}))

	if got := len(on) - len(off); got != 3 {
		t.Errorf("hiding the frame freed %d lines, want 3 (on=%d off=%d)", got, len(on), len(off))
	}

	joined := strings.Join(off, "\n")
	for _, glyph := range []string{"┌", "┐", "└", "┘"} {
		if strings.Contains(joined, glyph) {
			t.Errorf("corner glyph %q still present after HideTableFrame:\n%s", glyph, joined)
		}
	}
	// The labels stay — the frame toggle must not swallow the legend.
	if !strings.Contains(joined, "~cost") {
		t.Error("hiding the frame also removed the legend labels")
	}
}

// ─── T-CT3: dividers ─────────────────────────────────────────────────────────

// Dividers cost no rows — they are pure visual noise — so the line count must
// be unchanged while the inner separators and notch glyphs disappear. The outer
// bars belong to the frame and legitimately survive.
func TestChrome_DividersOff_KeepsLineCount(t *testing.T) {
	on := renderLines(t, chromeConfig(func(c *probes.Config) { c.TableRows = 2 }))
	off := renderLines(t, chromeConfig(func(c *probes.Config) {
		c.TableRows = 2
		c.HideTableDividers = true
	}))

	if len(on) != len(off) {
		t.Errorf("hiding dividers changed the line count: on=%d off=%d", len(on), len(off))
	}

	for _, line := range tableBlock(t, chromeConfig(func(c *probes.Config) {
		c.TableRows = 2
		c.HideTableDividers = true
	})) {
		for _, glyph := range []string{"┼", "┬", "┴"} {
			if strings.Contains(line, glyph) {
				t.Errorf("join glyph %q still present after HideTableDividers: %q", glyph, line)
			}
		}
		// Inner separators are gone: a data row keeps at most the two outer bars.
		if n := strings.Count(line, "│"); n > 2 {
			t.Errorf("row still carries %d │ after HideTableDividers: %q", n, line)
		}
	}
}

// T-CT4: with frame AND dividers off no box-drawing glyph may remain at all.
func TestChrome_FrameAndDividersOff_LeavesPlainText(t *testing.T) {
	block := tableBlock(t, chromeConfig(func(c *probes.Config) {
		c.TableRows = 2
		c.HideTableFrame = true
		c.HideTableDividers = true
	}))

	joined := strings.Join(block, "\n")
	for _, glyph := range []string{"│", "┼", "├", "┤", "┬", "┴", "┌", "┐", "└", "┘", "─"} {
		if strings.Contains(joined, glyph) {
			t.Errorf("box glyph %q survived frame+dividers off:\n%s", glyph, joined)
		}
	}
	// Content is still there and still column-aligned.
	if !strings.Contains(joined, "ToolB") || !strings.Contains(joined, "orch") {
		t.Errorf("row content lost:\n%s", joined)
	}
}

// T-CT5: dividers off must preserve column alignment — a │ is replaced by a
// space of the same visible width, never dropped.
func TestChrome_DividersOff_PreservesColumnWidth(t *testing.T) {
	on := tableBlock(t, chromeConfig(func(c *probes.Config) { c.TableRows = 2 }))
	off := tableBlock(t, chromeConfig(func(c *probes.Config) {
		c.TableRows = 2
		c.HideTableDividers = true
	}))

	if len(on) < 2 || len(off) < 2 {
		t.Fatalf("no data rows rendered (on=%d off=%d)", len(on), len(off))
	}
	// Index 1 is a data row in both renders (index 0 is the top border).
	onWidth := len([]rune(on[1]))
	offWidth := len([]rune(off[1]))
	if onWidth != offWidth {
		t.Errorf("data row width changed with dividers off: on=%d off=%d\n on: %q\noff: %q",
			onWidth, offWidth, on[1], off[1])
	}
}

// ─── T-CT6: table_rows = 0 ───────────────────────────────────────────────────

// HideTable drops the entire per-turn block — rows, legend and frame alike —
// leaving only the two header lines.
func TestChrome_HideTable_LeavesOnlyHeaders(t *testing.T) {
	lines := renderLines(t, chromeConfig(func(c *probes.Config) { c.HideTable = true }))

	if len(lines) != 2 {
		t.Errorf("HideTable should leave the two header lines, got %d:\n%s",
			len(lines), strings.Join(lines, "\n"))
	}
	joined := strings.Join(lines, "\n")
	for _, glyph := range []string{"┌", "└", "│"} {
		if strings.Contains(joined, glyph) {
			t.Errorf("table glyph %q survived HideTable", glyph)
		}
	}
}

// ─── T-CT7: header lines ─────────────────────────────────────────────────────

// Each header line can be dropped independently, and dropping one must not
// leave an empty row behind.
func TestChrome_HeaderLines_DropIndependently(t *testing.T) {
	both := renderLines(t, chromeConfig(func(c *probes.Config) { c.HideTable = true }))
	if len(both) != 2 {
		t.Fatalf("fixture: expected 2 header lines, got %d: %q", len(both), both)
	}

	noLine1 := renderLines(t, chromeConfig(func(c *probes.Config) {
		c.HideTable = true
		c.HideHeaderLine1 = true
	}))
	if len(noLine1) != 1 {
		t.Errorf("HideHeaderLine1: got %d lines, want 1: %q", len(noLine1), noLine1)
	} else if noLine1[0] != both[0] {
		t.Errorf("HideHeaderLine1 kept the wrong row:\ngot  %q\nwant %q", noLine1[0], both[0])
	}

	noLine0 := renderLines(t, chromeConfig(func(c *probes.Config) {
		c.HideTable = true
		c.HideHeaderLine0 = true
	}))
	if len(noLine0) != 1 {
		t.Errorf("HideHeaderLine0: got %d lines, want 1: %q", len(noLine0), noLine0)
	} else if noLine0[0] != both[1] {
		t.Errorf("HideHeaderLine0 kept the wrong row:\ngot  %q\nwant %q", noLine0[0], both[1])
	}
}

// T-CT8: a header line whose every probe is switched off must not be emitted as
// a blank row — turning off the last widget removes the line.
func TestChrome_EmptyHeaderLine_IsNotEmitted(t *testing.T) {
	// ProjectEnabled=false leaves line0 with no visible probe at all.
	lines := renderLines(t, probes.Config{
		ModelEnabled: true,
		HideHints:    true,
		HideTable:    true,
	})

	if len(lines) != 1 {
		t.Errorf("an all-off header line must be dropped, got %d lines: %q", len(lines), lines)
	}
	for _, l := range lines {
		if strings.TrimSpace(l) == "" {
			t.Errorf("blank line emitted: %q", lines)
		}
	}
}

// T-CT9: everything off at once renders nothing at all — no stray blank line.
func TestChrome_EverythingOff_RendersEmpty(t *testing.T) {
	lines := renderLines(t, chromeConfig(func(c *probes.Config) {
		c.HideTable = true
		c.HideHeaderLine0 = true
		c.HideHeaderLine1 = true
	}))

	if joined := strings.Join(lines, "\n"); strings.TrimSpace(joined) != "" {
		t.Errorf("expected empty output with every row hidden, got:\n%q", joined)
	}
}

// ─── T-CT10: zero value ──────────────────────────────────────────────────────

// A Config with no Hide* flag set must reproduce the pre-Phase-7.5 layout.
// This is the guard that keeps every existing caller (and every older test)
// honest when new chrome flags are added.
func TestChrome_NoHideFlags_KeepsFullChrome(t *testing.T) {
	joined := strings.Join(renderLines(t, probes.Config{
		ModelEnabled:   true,
		ProjectEnabled: true,
		HideHints:      true,
		TableRows:      2,
	}), "\n")

	for _, want := range []string{"┌", "└", "│", "~cost"} {
		if !strings.Contains(joined, want) {
			t.Errorf("Config without Hide* flags lost %q from the default layout:\n%s", want, joined)
		}
	}
}
