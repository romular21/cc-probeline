// Package format — visual length helpers for status-line truncation.
package format

import (
	"regexp"

	"github.com/mattn/go-runewidth"
)

// markerRE matches our marker grammar: `{{` + a lowercase letter + zero or
// more of [a-z0-9:_-] + `}}`. The grammar is closed: the assembler emits
// only the recognised palette ({{color:NAME}}, {{dim}}, {{bold}},
// {{italic}}, {{reset}}), but the regex is generic so unknown tokens with
// the same shape also strip. Malformed sequences (single braces, missing
// closer, leading digit) pass through unchanged.
var markerRE = regexp.MustCompile(`\{\{[a-z][a-z0-9:_-]*\}\}`)

// ansiRE matches a CSI escape sequence — ESC [ params, then a final byte —
// which is the shape of every colour code the palette holds.
var ansiRE = regexp.MustCompile("\x1b\\[[0-9;?]*[ -/]*[@-~]")

// StripMarkers removes marker tokens and raw ANSI escapes from s.
//
// Both forms have to go. The design intends probes to emit markers and the
// renderer to convert them to escapes at the very end — but the palette also
// exposes raw escape strings (t.Colors.Green and friends), and the quota probes
// have always concatenated those directly. Since colours became user
// configurable there is no route back to markers for them: the marker grammar
// is a closed vocabulary, and "#7cb27c" has no token in it.
//
// Leaving escapes in place was not harmless. Control bytes have a runewidth of
// zero, but the printable tail of an escape — "[38;5;108m" — counts as ten
// columns, so a coloured segment measured about eleven columns wider than it
// draws. Every caller inherited the error: FitLine compressed segments that
// still fit, and the header merge refused to join two rows with room to spare.
// The symptom was a status line that behaved differently with colour on than
// with NO_COLOR set — the one difference a test suite written with colour off
// cannot see.
func StripMarkers(s string) string {
	return ansiRE.ReplaceAllString(markerRE.ReplaceAllString(s, ""), "")
}

// VisualLen returns the terminal column width of s after StripMarkers, using
// runewidth so wide UTF-8 glyphs (CJK) count as 2.
func VisualLen(s string) int {
	return runewidth.StringWidth(StripMarkers(s))
}
