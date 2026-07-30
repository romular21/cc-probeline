package frames_test

// dim_test.go — flattenDim: rewrite ANSI "faint" (SGR 2) into explicit dark
// colours for the README asset pipeline.
//
// Why this exists: the status line marks history — turns from older
// orchestrator requests — by wrapping whole rows in SGR 2, so the current
// request reads bright against a faded backlog. Terminals honour that, but the
// SVG/PNG renderers used to turn the emitted .ansi files into screenshots do
// not: they drop the attribute and paint faint text at full strength, which
// flattens the table and sells a status line nobody sees.
//
// Rather than lose the distinction, the emitter can pre-flatten it: inside a
// faint region every colour is swapped for a darker equivalent, so the contrast
// survives in renderers that only understand plain colour. Gated behind
// CC_PROBELINE_EMIT_FLATTEN_DIM=1 — a real terminal needs no such help, and the
// unflattened output stays the default.

import (
	"regexp"
	"strconv"
	"strings"
	"testing"
)

// ─── tests ───────────────────────────────────────────────────────────────────

// T-FD1: the whole point — a faint row and a bright row must not come out the
// same colour, which is exactly what happens when SGR 2 is dropped.
func TestFlattenDim_FaintAndBrightDiffer(t *testing.T) {
	bright := "\x1b[36morchestrator\x1b[0m"
	faint := "\x1b[2m\x1b[36morchestrator\x1b[0m"

	gotBright := flattenDim(bright)
	gotFaint := flattenDim(faint)

	if gotBright != bright {
		t.Errorf("a bright row must pass through untouched:\ngot  %q\nwant %q", gotBright, bright)
	}
	if gotFaint == gotBright {
		t.Errorf("faint and bright rows flattened to the same output: %q", gotFaint)
	}
	if !strings.Contains(gotFaint, dimColour["36"]) {
		t.Errorf("faint cyan should become %s, got %q", dimColour["36"], gotFaint)
	}
	if strings.Contains(gotFaint, "\x1b[2m") {
		t.Errorf("SGR 2 should not survive flattening: %q", gotFaint)
	}
}

// T-FD2: plain (uncoloured) text inside a faint region still has to fade —
// most of a history row is uncoloured.
func TestFlattenDim_PlainTextFades(t *testing.T) {
	got := flattenDim("\x1b[2m 13 │ opus-4-8 \x1b[0m")

	if !strings.Contains(got, dimDefault) {
		t.Errorf("plain faint text should take the dim default %s, got %q", dimDefault, got)
	}
}

// T-FD3: a reset ends the region — colours after it must come back bright, or
// one faint row would bleed into the rest of the table.
func TestFlattenDim_ResetEndsRegion(t *testing.T) {
	got := flattenDim("\x1b[2mhistory\x1b[0m\x1b[36mfresh\x1b[0m")

	if !strings.HasSuffix(got, "\x1b[36mfresh\x1b[0m") {
		t.Errorf("colour after the reset must stay bright, got %q", got)
	}
}

// T-FD4: extended colours inside a faint region are deliberate palette choices
// (the orange context bar, for one) and must be preserved verbatim.
func TestFlattenDim_ExtendedColoursPreserved(t *testing.T) {
	in := "\x1b[2m\x1b[38;5;208mbar\x1b[0m"

	if got := flattenDim(in); !strings.Contains(got, "38;5;208") {
		t.Errorf("256-colour value must survive flattening, got %q", got)
	}
}

// T-FD5: text with no escapes at all is returned unchanged.
func TestFlattenDim_PlainInputUnchanged(t *testing.T) {
	in := "no escapes here"
	if got := flattenDim(in); got != in {
		t.Errorf("plain input changed: got %q, want %q", got, in)
	}
}

// sgrRe matches a CSI SGR sequence ("\x1b[…m") and captures its parameters.
var sgrRe = regexp.MustCompile(`\x1b\[([0-9;]*)m`)

// dimColour maps a bright SGR colour parameter to a dark 256-colour stand-in
// used inside a faint region. The targets are the same hues two or three steps
// down the 256-colour ramp, which is what a terminal's faint rendering
// approximates anyway.
var dimColour = map[string]string{
	"30": "38;5;236", // black
	"31": "38;5;88",  // red
	"32": "38;5;28",  // green
	"33": "38;5;94",  // yellow
	"34": "38;5;18",  // blue
	"35": "38;5;53",  // magenta
	"36": "38;5;30",  // cyan
	"37": "38;5;243", // white
	"39": "38;5;243", // default foreground
}

// dimDefault is the colour plain text takes inside a faint region.
const dimDefault = "38;5;243"

// flattenDim rewrites s so that faint regions carry explicit dark colours.
//
// It walks the SGR sequences tracking whether SGR 2 is in effect:
//   - "2" turns the region on and is replaced by the dim default colour;
//   - a colour set inside the region is replaced by its dark counterpart;
//   - "0" (reset) ends the region and passes through untouched.
//
// 256-colour and truecolour sequences (38;5;… / 38;2;…) inside a faint region
// are left alone: they are already explicit choices, and guessing a darker
// variant would distort hues the palette picked deliberately.
func flattenDim(s string) string {
	var out strings.Builder
	dim := false
	last := 0

	for _, loc := range sgrRe.FindAllStringSubmatchIndex(s, -1) {
		start, end := loc[0], loc[1]
		params := s[loc[2]:loc[3]]

		out.WriteString(s[last:start])
		last = end

		out.WriteString(rewriteSGR(params, &dim))
	}
	out.WriteString(s[last:])

	return out.String()
}

// rewriteSGR returns the replacement for one SGR sequence, updating dim to
// reflect whether a faint region is in effect after it.
func rewriteSGR(params string, dim *bool) string {
	if params == "" {
		params = "0" // "\x1b[m" is shorthand for a reset
	}

	fields := strings.Split(params, ";")
	rewritten := make([]string, 0, len(fields))

	for i := 0; i < len(fields); i++ {
		f := fields[i]
		switch {
		case f == "2":
			// Enter the faint region: SGR 2 itself is dropped (the renderers we
			// target ignore it) and stands in as the dim default colour. A colour
			// later in the same sequence overrides it, which is the intent.
			*dim = true
			rewritten = append(rewritten, dimDefault)

		case f == "0":
			*dim = false
			rewritten = append(rewritten, f)

		case *dim && isExtendedColour(f) && i+1 < len(fields):
			// 38;5;N or 38;2;R;G;B — already explicit, copy the whole run through.
			run := extendedColourRun(fields[i:])
			rewritten = append(rewritten, run...)
			i += len(run) - 1

		case *dim && dimColour[f] != "":
			rewritten = append(rewritten, dimColour[f])

		default:
			rewritten = append(rewritten, f)
		}
	}

	return "\x1b[" + strings.Join(rewritten, ";") + "m"
}

// isExtendedColour reports whether f introduces a 256-colour/truecolour run.
func isExtendedColour(f string) bool {
	n, err := strconv.Atoi(f)
	return err == nil && (n == 38 || n == 48)
}

// extendedColourRun returns the leading fields that form one extended-colour
// sequence: "38;5;N" (3 fields) or "38;2;R;G;B" (5). A malformed run falls back
// to the single introducer so nothing is swallowed.
func extendedColourRun(fields []string) []string {
	if len(fields) < 2 {
		return fields[:1]
	}
	switch fields[1] {
	case "5":
		if len(fields) >= 3 {
			return fields[:3]
		}
	case "2":
		if len(fields) >= 5 {
			return fields[:5]
		}
	}
	return fields[:1]
}
