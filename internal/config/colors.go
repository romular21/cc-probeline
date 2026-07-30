package config

// colors.go — user overrides for the palette.
//
// The built-in palette sticks to 16-colour ANSI so it reads on whatever theme
// the terminal happens to use, with two 256-colour exceptions (orange, amber)
// for hues 16-colour ANSI simply does not have. That is a sound default and a
// poor straitjacket: terminal themes vary enormously, and what reads as a calm
// green on one is acid on another. [colors] lets any role be repointed without
// a rebuild.

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"

	"github.com/labzink/cc-probeline/internal/renderer"
)

// Colors overrides individual palette roles. Every field is optional; an empty
// value keeps the built-in colour.
//
// Values accept two forms:
//
//	"#rrggbb"     a hex colour, emitted as a 24-bit truecolour escape
//	"38;5;179"    raw SGR parameters, emitted verbatim between ESC[ and m
//
// The raw form is the escape hatch: anything the terminal understands — 256
// colour, bold combinations, even blink — can be written directly.
type Colors struct {
	// Green is the healthy band: usage below the notice threshold.
	Green string `toml:"green" json:"green"`

	// Amber is the notice band for values rendered as text (bar_style = "none").
	Amber string `toml:"amber" json:"amber"`

	// Sage is the healthy band for values rendered as text (bar_style = "none").
	Sage string `toml:"sage" json:"sage"`

	// Yellow is the notice band elsewhere, plus TTL and agent labels.
	Yellow string `toml:"yellow" json:"yellow"`

	// Orange is the warn band.
	Orange string `toml:"orange" json:"orange"`

	// Red is the critical band.
	Red string `toml:"red" json:"red"`

	// Cyan is the git branch and the "orchestrator" role label.
	Cyan string `toml:"cyan" json:"cyan"`

	// Magenta is the reasoning-effort indicator.
	Magenta string `toml:"magenta" json:"magenta"`
}

// hexRe matches a six-digit hex colour with a leading '#'.
var hexRe = regexp.MustCompile(`^#[0-9a-fA-F]{6}$`)

// sgrRe matches a raw SGR parameter list: digits separated by semicolons.
var sgrRe = regexp.MustCompile(`^[0-9]+(;[0-9]+)*$`)

// ParseColour converts a config colour value into an ANSI escape sequence.
// Returns ("", false) when the value is not a form we recognise, so the caller
// can keep the built-in colour rather than emitting garbage into the terminal.
func ParseColour(v string) (string, bool) {
	v = strings.TrimSpace(v)
	if v == "" {
		return "", false
	}

	if hexRe.MatchString(v) {
		r, _ := strconv.ParseUint(v[1:3], 16, 8)
		g, _ := strconv.ParseUint(v[3:5], 16, 8)
		b, _ := strconv.ParseUint(v[5:7], 16, 8)
		return fmt.Sprintf("\x1b[38;2;%d;%d;%dm", r, g, b), true
	}

	if sgrRe.MatchString(v) {
		return "\x1b[" + v + "m", true
	}

	return "", false
}

// ApplyColors returns pal with every valid override in c applied. Invalid
// values are skipped: a typo in the config costs the user that one colour, not
// their status line.
func ApplyColors(pal renderer.ColorScheme, c Colors) renderer.ColorScheme {
	set := func(dst *string, v string) {
		if esc, ok := ParseColour(v); ok {
			*dst = esc
		}
	}

	set(&pal.Green, c.Green)
	set(&pal.Amber, c.Amber)
	set(&pal.Sage, c.Sage)
	set(&pal.Yellow, c.Yellow)
	set(&pal.Orange, c.Orange)
	set(&pal.Red, c.Red)
	set(&pal.Cyan, c.Cyan)
	set(&pal.Magenta, c.Magenta)

	// The pre-computed bold combinations are derived, so they follow their base
	// colour rather than silently keeping the old hue.
	if esc, ok := ParseColour(c.Green); ok {
		pal.BoldGreen = pal.Bold + esc
	}
	if esc, ok := ParseColour(c.Yellow); ok {
		pal.BoldYellow = pal.Bold + esc
	}
	if esc, ok := ParseColour(c.Red); ok {
		pal.BoldRed = pal.Bold + esc
	}

	return pal
}

// colourKeys lists the [colors] roles SetColor accepts.
var colourKeys = []string{"green", "sage", "amber", "yellow", "orange", "red", "cyan", "magenta"}

// SetColor atomically updates one role in [colors]. The value is validated
// before it is written, so an unusable colour never reaches the file.
func SetColor(path, name, value string) error {
	if !isValidColourKey(name) {
		return fmt.Errorf("unknown colour %q: accepted names are %s",
			name, strings.Join(colourKeys, ", "))
	}
	if _, ok := ParseColour(value); !ok {
		return fmt.Errorf("invalid colour %q: use \"#rrggbb\" or raw SGR parameters such as \"38;5;179\"", value)
	}
	return setScalar(path, "colors", name, tomlString(value))
}

// isValidColourKey reports whether name is a known [colors] role.
func isValidColourKey(name string) bool {
	for _, k := range colourKeys {
		if k == name {
			return true
		}
	}
	return false
}
