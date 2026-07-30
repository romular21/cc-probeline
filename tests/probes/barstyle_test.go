// Package probes_test — [general].bar_style: which glyphs the progress bars are
// drawn with, including the style that replaces the bar with its percentage.
//
// A terminal line has a fixed height, so a visually lighter bar can only come
// from glyphs that occupy less of the cell. These tests pin the substitution
// down and, more importantly, guard the one thing a style change must never do:
// state the same percentage twice.
package probes_test

import (
	"strings"
	"testing"

	"github.com/labzink/cc-probeline/internal/probes"
	"github.com/labzink/cc-probeline/internal/renderer"
	"github.com/labzink/cc-probeline/internal/stdin"
)

// ctxData returns Data with a context window at the given fill.
func ctxData(used, size int) probes.Data {
	return probes.Data{
		Stdin: stdin.Payload{
			ContextWindow: stdin.ContextWindow{
				Size:         size,
				CurrentUsage: map[string]int{"cache_read_input_tokens": used},
			},
		},
		Now: mqNow,
	}
}

// ctxCfg returns a Config with the ctx probe on and the given bar style.
func ctxCfg(style string) probes.Config {
	return probes.Config{CtxEnabled: true, BarStyle: style, BarWidth: 10}
}

// T-BS1: each style draws the bar with its own glyphs and none of the others'.
func TestBarStyle_SubstitutesGlyphs(t *testing.T) {
	cases := []struct {
		style   string
		want    string // a glyph this style must use
		unwant  string // a glyph it must not
		wantPct bool   // whether the percentage stands in for the bar
	}{
		{"block", "█", "━", false},
		{"line", "━", "█", false},
		{"low", "▄", "█", false},
		{"dot", "●", "█", false},
		{"none", "", "█", true},
	}

	p := &probes.CtxProbe{}
	d := ctxData(500_000, 1_000_000) // 50% — guarantees at least one full segment

	for _, c := range cases {
		t.Run(c.style, func(t *testing.T) {
			got := p.Render(d, ctxCfg(c.style), plainTheme, probes.LevelFull)

			if c.want != "" && !strings.Contains(got, c.want) {
				t.Errorf("style %q should draw with %q: %q", c.style, c.want, got)
			}
			if strings.Contains(got, c.unwant) {
				t.Errorf("style %q still uses %q: %q", c.style, c.unwant, got)
			}
			if c.wantPct && !strings.Contains(got, "50%") {
				t.Errorf("style %q must render the percentage in the bar's place: %q", c.style, got)
			}
		})
	}
}

// T-BS2: an unset or unrecognised style renders exactly like "block", so a
// typo in the config costs nothing.
func TestBarStyle_UnknownFallsBackToBlock(t *testing.T) {
	p := &probes.CtxProbe{}
	d := ctxData(500_000, 1_000_000)

	block := p.Render(d, ctxCfg("block"), plainTheme, probes.LevelFull)
	for _, style := range []string{"", "nonsense", "BLOCK"} {
		if got := p.Render(d, ctxCfg(style), plainTheme, probes.LevelFull); got != block {
			t.Errorf("style %q should render as block:\ngot   %q\nblock %q", style, got, block)
		}
	}
}

// T-BS3: with the bar replaced by a percentage, the percentage must appear
// exactly once — the context segment used to append its own "(NN%)" as well.
func TestBarStyle_NonePrintsPercentOnce(t *testing.T) {
	got := (&probes.CtxProbe{}).Render(
		ctxData(430_000, 1_000_000), ctxCfg("none"), plainTheme, probes.LevelFull)

	if n := strings.Count(got, "43%"); n != 1 {
		t.Errorf("percentage appears %d times, want 1: %q", n, got)
	}
}

// T-BS4: the per-model window follows the same style as everything else, so a
// line never mixes two kinds of bar.
func TestBarStyle_AppliesToModelQuota(t *testing.T) {
	d := mqData(50)
	cfg := mqEnabled
	cfg.BarStyle = "line"
	cfg.BarWidth = 10

	got := (&probes.ModelQuotaProbe{}).Render(d, cfg, plainTheme, probes.LevelFull)

	if !strings.Contains(got, "━") {
		t.Errorf("model quota ignored bar_style: %q", got)
	}
	if strings.Contains(got, "█") {
		t.Errorf("model quota still drawing block glyphs: %q", got)
	}
}

// T-BS5: Compact level honours the style too — it is reached by narrowing the
// terminal, which must not silently change how the bars look.
func TestBarStyle_AppliesAtCompactLevel(t *testing.T) {
	got := (&probes.CtxProbe{}).Render(
		ctxData(500_000, 1_000_000), ctxCfg("dot"), renderer.Theme{AnsiEnabled: false}, probes.LevelCompact)

	if !strings.Contains(got, "●") {
		t.Errorf("Compact level ignored bar_style: %q", got)
	}
	if strings.Contains(got, "█") {
		t.Errorf("Compact level still drawing block glyphs: %q", got)
	}
}
