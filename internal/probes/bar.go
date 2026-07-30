package probes

import (
	"fmt"
	"strings"

	"github.com/labzink/cc-probeline/internal/renderer"
)

// defaultBarWidth is the segment count Full-level bars use when [general].bar_width
// is unset. It matches the layout the status line has always shipped.
const defaultBarWidth = 10

// minBarWidth and maxBarWidth bound [general].bar_width. Below three segments a
// bar carries no shape at all; past twenty it stops being a glance-value.
const (
	minBarWidth = 3
	maxBarWidth = 20
)

// Bar styles accepted by [general].bar_style.
//
// A terminal line has one height, so a "shorter" bar can only come from glyphs
// that occupy less of it. block fills the cell; line draws through the middle;
// low sits on the baseline; dot echoes the effort circles; none drops the bar
// altogether and prints the number the bar was standing in for.
const (
	barStyleBlock = "block" // █ ▒ ░ — the original, full cell height
	barStyleLine  = "line"  // ━ ╾ ─ — a rule through the middle of the line
	barStyleLow   = "low"   // ▄ ▂ ▁ — sits on the baseline
	barStyleDot   = "dot"   // ● ◐ ○ — matches the effort indicator's circles
	barStyleNone  = "none"  // no bar at all: the percentage stands in for it
)

// barGlyphs maps a style to its full/half/empty runes. The renderer always
// produces block glyphs; these are substituted afterwards, so the fill maths
// and every existing snapshot stay untouched.
var barGlyphs = map[string][3]string{
	barStyleLine: {"━", "╾", "─"},
	barStyleLow:  {"▄", "▂", "▁"},
	barStyleDot:  {"●", "◐", "○"},
}

// barStyleNoBar reports whether the configured style replaces the bar with a
// bare percentage. Callers use it to suppress the percentage they would
// otherwise append beside the bar, which would then appear twice.
func barStyleNoBar(c Config) bool { return c.BarStyle == barStyleNone }

// usageBar renders a Full-level progress bar at the width and in the style the
// config asks for.
//
// Widths 5 and 10 keep their historical renderers, which pre-round the value so
// the bar holds still as a percentage drifts. Every other width uses the
// proportional renderer, which does not round and therefore keeps a small
// value visible instead of flooring it away. An unset or out-of-range width
// falls back to the default rather than failing — a bad config value must never
// cost the user their bar.
func usageBar(percent float64, c Config) string {
	if barStyleNoBar(c) {
		return fmt.Sprintf("%d%%", displayPctInt(percent, false))
	}
	return restyleBar(rawUsageBar(percent, c), c)
}

// compactBar renders the 5-segment bar used at Compact level, in the configured
// style. Width is fixed here: Compact exists precisely because the line has run
// out of room, so it is not the place to honour a width preference.
func compactBar(percent float64, c Config) string {
	if barStyleNoBar(c) {
		return fmt.Sprintf("%d%%", displayPctInt(percent, false))
	}
	return restyleBar(renderer.ProgressBar(percent), c)
}

// rawUsageBar picks the renderer for the configured width, always in block
// glyphs.
func rawUsageBar(percent float64, c Config) string {
	switch w := c.BarWidth; {
	case w == 5:
		return renderer.ProgressBar(percent)
	case w >= minBarWidth && w <= maxBarWidth && w != defaultBarWidth:
		return renderer.ProgressBarN(percent, w)
	default:
		return renderer.ProgressBar10(percent)
	}
}

// restyleBar substitutes the block glyphs for the configured style's. An unset
// or unrecognised style leaves the bar as it is, which is the original look.
func restyleBar(bar string, c Config) string {
	g, ok := barGlyphs[c.BarStyle]
	if !ok {
		return bar
	}
	r := strings.NewReplacer("█", g[0], "▒", g[1], "░", g[2])
	return r.Replace(bar)
}
