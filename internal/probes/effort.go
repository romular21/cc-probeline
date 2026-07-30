package probes

import (
	"github.com/labzink/cc-probeline/internal/renderer"
)

// effortIcon maps effort level strings to Unicode circle icons.
// "off" maps to the empty string (probe hidden via Visible).
var effortIcon = map[string]string{
	"low":    "○",
	"medium": "◔",
	"high":   "◑",
	"xhigh":  "◕",
	"max":    "●",
	"off":    "",
}

// effortWord maps effort level strings to their spelled-out name, used when
// [general].effort_style = "word". The circle icons encode the level by how
// full the circle is, which is compact but needs a legend to read; the word
// needs none.
var effortWord = map[string]string{
	"low":    "low",
	"medium": "medium",
	"high":   "high",
	"xhigh":  "xhigh",
	"max":    "max",
	"off":    "",
}

// EffortProbe renders the effort level as a Unicode circle icon.
// All three display levels render the same icon (effort is P0; icon is never dropped).
type EffortProbe struct{}

func (p *EffortProbe) Name() string  { return "effort" }
func (p *EffortProbe) Priority() int { return 0 }
func (p *EffortProbe) MinWidth() int { return 1 } // single rune icon

// Visible returns false when EffortEnabled is false or the effort level is "off" or empty.
func (p *EffortProbe) Visible(d Data, c Config) bool {
	if !c.EffortEnabled {
		return false
	}
	lvl := d.Stdin.Effort.Level
	return lvl != "" && lvl != "off"
}

// Render returns the Unicode icon for the effort level, colour-wrapped per
// B3 §5 (see effortGlyph). Returns "" for "off" or unrecognised levels.
func (p *EffortProbe) Render(d Data, c Config, t renderer.Theme, level Level) string {
	return effortMark(d.Stdin.Effort.Level, t.AnsiEnabled, c.EffortWord)
}

// effortColorMarker returns the opening colour marker token for the given effort
// level when used as a wrapper around a text segment (model name or icon):
//
//	low            → "{{dim}}"
//	high/xhigh/max → "{{color:magenta}}"
//	medium or ""   → "" (no marker; caller renders text plain)
//
// The caller is responsible for appending "{{reset}}" when the marker is non-empty.
// This is the single source of truth for effort-level colour selection, shared by
// effortGlyph (icon wrapping) and ModelProbe.Render (model-name wrapping).
func effortColorMarker(lvl string) string {
	switch lvl {
	case "low":
		return "{{dim}}"
	case "high", "xhigh", "max":
		return "{{color:magenta}}"
	default:
		// medium, empty, "off", or any unrecognised level — no colour marker.
		return ""
	}
}

// effortGlyph returns the effort icon for lvl, colour-wrapped per B3 §5 when
// ansiEnabled. It delegates colour selection to effortColorMarker so both the
// icon and the model name share the same colour semantics:
//
//	low            → {{dim}}…{{reset}}
//	medium         → no marker (default colour)
//	high/xhigh/max → {{color:magenta}}…{{reset}}
//
// Returns "" for "off" or unrecognised levels (caller drops the segment).
func effortGlyph(lvl string, ansiEnabled bool) string {
	return effortMark(lvl, ansiEnabled, false)
}

// effortMark renders the effort indicator as either the circle icon or the
// spelled-out level, applying the same colour treatment to both.
func effortMark(lvl string, ansiEnabled, asWord bool) string {
	table := effortIcon
	if asWord {
		table = effortWord
	}
	icon, ok := table[lvl]
	if !ok || icon == "" {
		return ""
	}
	if !ansiEnabled {
		return icon
	}
	marker := effortColorMarker(lvl)
	if marker == "" {
		// medium and any future default levels — no colour marker.
		return icon
	}
	return marker + icon + "{{reset}}"
}
