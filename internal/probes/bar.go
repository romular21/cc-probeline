package probes

import "github.com/labzink/cc-probeline/internal/renderer"

// defaultBarWidth is the segment count Full-level bars use when [general].bar_width
// is unset. It matches the layout the status line has always shipped.
const defaultBarWidth = 10

// minBarWidth and maxBarWidth bound [general].bar_width. Below three segments a
// bar carries no shape at all; past twenty it stops being a glance-value.
const (
	minBarWidth = 3
	maxBarWidth = 20
)

// usageBar renders a Full-level progress bar at the width the config asks for.
//
// Widths 5 and 10 keep their historical renderers, which pre-round the value so
// the bar holds still as a percentage drifts. Every other width uses the
// proportional renderer, which does not round and therefore keeps a small
// value visible instead of flooring it away. An unset or out-of-range width
// falls back to the default rather than failing — a bad config value must never
// cost the user their bar.
func usageBar(percent float64, c Config) string {
	switch w := c.BarWidth; {
	case w == 5:
		return renderer.ProgressBar(percent)
	case w >= minBarWidth && w <= maxBarWidth && w != defaultBarWidth:
		return renderer.ProgressBarN(percent, w)
	default:
		return renderer.ProgressBar10(percent)
	}
}
