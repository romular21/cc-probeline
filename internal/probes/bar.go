package probes

import "github.com/labzink/cc-probeline/internal/renderer"

// defaultBarWidth is the segment count Full-level bars use when [general].bar_width
// is unset. It matches the layout the status line has always shipped.
const defaultBarWidth = 10

// usageBar renders a Full-level progress bar at the width the config asks for.
//
// Only two widths exist because only two are legible: ten segments resolve 5%,
// five resolve 10%. Anything between would imply a precision the glyphs cannot
// carry. An unset or unrecognised width falls back to ten rather than failing —
// a bad config value must never cost the user their bar.
func usageBar(percent float64, c Config) string {
	if c.BarWidth == 5 {
		return renderer.ProgressBar(percent)
	}
	return renderer.ProgressBar10(percent)
}
