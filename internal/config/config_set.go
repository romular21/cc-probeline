package config

// config_set.go — the setters behind the `cc-probeline <key> <value>` commands
// and the /cc-probeline-config wizard.
//
// Every one of them edits a single value in place via setScalar, so comments,
// key order and formatting in the user's config.toml survive untouched. None of
// them round-trips the file through Config: that would rewrite the whole file
// to change one boolean, which is a poor trade for a file people are invited to
// read and annotate by hand.

import (
	"fmt"
	"strings"
)

// tableRowsCap is the maximum value accepted by SetTableRows.
const tableRowsCap = 40

// tableRowsFloor is the minimum value accepted by SetTableRows. Negative values
// are raised to this floor. 0 is deliberately inside the accepted range: it is
// the "no table at all" setting, which drops the per-turn block (rows, legend
// and frame) from the status line.
const tableRowsFloor = 0

// barWidthMin and barWidthMax bound SetBarWidth. Below three segments a bar has
// no shape; past twenty it stops being readable at a glance.
const (
	barWidthMin = 3
	barWidthMax = 20
)

// validModes lists the accepted values for SetMode.
var validModes = []string{"standard", "super-compact"}

// effortStyles lists the values SetEffortStyle accepts.
var effortStyles = []string{"glyph", "word"}

// barStyles lists the values SetBarStyle accepts.
var barStyles = []string{"block", "line", "low", "dot", "none"}

// widgetKeys lists the [widgets] toggles SetWidget accepts, in the order the
// status line renders them so error messages read naturally.
var widgetKeys = []string{
	"model", "effort", "cost", "project", "email",
	"time", "ctx", "quota", "quota_model", "git",
}

// ─── [general] ───────────────────────────────────────────────────────────────

// SetTutorialHints atomically updates [general].tutorial_hints.
func SetTutorialHints(path string, value bool) error {
	return setScalar(path, "general", "tutorial_hints", tomlBool(value))
}

// SetMode atomically updates [general].mode.
// Accepted values: "standard", "super-compact". Any other value returns an
// error and leaves the file unchanged.
func SetMode(path, mode string) error {
	if !isValidMode(mode) {
		return fmt.Errorf("invalid mode %q: accepted values are %s",
			mode, strings.Join(validModes, ", "))
	}
	return setScalar(path, "general", "mode", tomlString(mode))
}

// SetNoColor atomically updates [general].no_color.
func SetNoColor(path string, value bool) error {
	return setScalar(path, "general", "no_color", tomlBool(value))
}

// SetPriceCheck atomically updates [general].price_check (Phase 7.46 Wave B /
// BL-36 — opt out of the once-per-day network price check).
func SetPriceCheck(path string, value bool) error {
	return setScalar(path, "general", "price_check", tomlBool(value))
}

// SetRefreshInterval atomically updates [general].refresh_interval_hint.
func SetRefreshInterval(path string, seconds int) error {
	return setScalar(path, "general", "refresh_interval_hint", tomlInt(seconds))
}

// SetTableRows atomically updates [general].table_rows. The value is clamped to
// [0, 40]: above 40 is capped, negatives are raised to 0. 0 is a valid setting
// and turns the per-turn table off entirely.
func SetTableRows(path string, rows int) error {
	if rows > tableRowsCap {
		rows = tableRowsCap
	}
	if rows < tableRowsFloor {
		rows = tableRowsFloor
	}
	return setScalar(path, "general", "table_rows", tomlInt(rows))
}

// SetTableLegend atomically updates [general].table_legend.
func SetTableLegend(path string, value bool) error {
	return setScalar(path, "general", "table_legend", tomlBool(value))
}

// SetTableFrame atomically updates [general].table_frame.
func SetTableFrame(path string, value bool) error {
	return setScalar(path, "general", "table_frame", tomlBool(value))
}

// SetTableDividers atomically updates [general].table_dividers.
func SetTableDividers(path string, value bool) error {
	return setScalar(path, "general", "table_dividers", tomlBool(value))
}

// SetHeaderMerge atomically updates [general].header_merge.
func SetHeaderMerge(path string, value bool) error {
	return setScalar(path, "general", "header_merge", tomlBool(value))
}

// SetHeaderLine atomically updates [general].header_line0 or header_line1.
// n selects the line (0 or 1); any other value returns an error and leaves the
// file unchanged.
func SetHeaderLine(path string, n int, value bool) error {
	if n != 0 && n != 1 {
		return fmt.Errorf("invalid header line %d: accepted values are 0, 1", n)
	}
	return setScalar(path, "general", fmt.Sprintf("header_line%d", n), tomlBool(value))
}

// SetBarWidth atomically updates [general].bar_width.
// Accepted range: 3–20 segments.
func SetBarWidth(path string, width int) error {
	if width < barWidthMin || width > barWidthMax {
		return fmt.Errorf("invalid bar_width %d: accepted range is %d-%d",
			width, barWidthMin, barWidthMax)
	}
	return setScalar(path, "general", "bar_width", tomlInt(width))
}

// SetBarStyle atomically updates [general].bar_style.
// Accepted values: block, line, low, dot, none.
func SetBarStyle(path, style string) error {
	if !isValidBarStyle(style) {
		return fmt.Errorf("invalid bar_style %q: accepted values are %s",
			style, strings.Join(barStyles, ", "))
	}
	return setScalar(path, "general", "bar_style", tomlString(style))
}

// SetEffortStyle atomically updates [general].effort_style.
// Accepted values: "glyph", "word".
func SetEffortStyle(path, style string) error {
	if style != "glyph" && style != "word" {
		return fmt.Errorf("invalid effort_style %q: accepted values are %s",
			style, strings.Join(effortStyles, ", "))
	}
	return setScalar(path, "general", "effort_style", tomlString(style))
}

// SetModelVariantTag atomically updates [general].model_variant_tag.
func SetModelVariantTag(path string, value bool) error {
	return setScalar(path, "general", "model_variant_tag", tomlBool(value))
}

// ─── [widgets] ───────────────────────────────────────────────────────────────

// SetWidget atomically updates the named toggle in [widgets].
// name must be one of the Widgets field TOML names (e.g. "model", "ctx").
// Unknown names return an error and leave the file unchanged.
func SetWidget(path, name string, value bool) error {
	if !isValidWidget(name) {
		return fmt.Errorf("unknown widget %q: accepted names are %s",
			name, strings.Join(widgetKeys, ", "))
	}
	return setScalar(path, "widgets", name, tomlBool(value))
}

// ─── validation helpers ──────────────────────────────────────────────────────

// isValidMode reports whether mode is one of validModes.
func isValidMode(mode string) bool {
	for _, v := range validModes {
		if mode == v {
			return true
		}
	}
	return false
}

// isValidBarStyle reports whether style is one of barStyles.
func isValidBarStyle(style string) bool {
	for _, v := range barStyles {
		if style == v {
			return true
		}
	}
	return false
}

// isValidWidget reports whether name is a known [widgets] key.
func isValidWidget(name string) bool {
	for _, k := range widgetKeys {
		if k == name {
			return true
		}
	}
	return false
}
