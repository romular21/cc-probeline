package config

// setinplace.go — surgical, comment-preserving edits of single TOML values.
//
// Every setter in this package routes through setScalar. The alternative —
// unmarshal into Config, marshal back out — silently rewrites the whole file:
// comments vanish, key order is normalised, and the user's annotated config
// comes back as a bare dump. That is a poor trade for changing one boolean,
// especially in a file people are told to read and edit by hand.
//
// The edit is line-level: find the assignment, replace what follows '=', leave
// every other byte alone. The file is validated as TOML before the edit, so a
// broken config is reported rather than clobbered.

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/pelletier/go-toml/v2"
)

// tomlBool renders a Go bool as a TOML literal.
func tomlBool(v bool) string { return strconv.FormatBool(v) }

// tomlInt renders a Go int as a TOML literal.
func tomlInt(v int) string { return strconv.Itoa(v) }

// tomlString renders a Go string as a quoted TOML literal.
func tomlString(v string) string { return strconv.Quote(v) }

// insertHint is the comment attached to a key the setter has to create.
//
// It applies on insertion only. A user who configures entirely through the CLI
// would otherwise accumulate a file of bare assignments and have to go read the
// shipped example to learn what else a key accepts — the config is meant to be
// readable on its own. Replacing an existing value never touches comments:
// whatever the user wrote there is theirs.
var insertHint = map[string]string{
	"table_rows":        "0 = no table at all, max 40",
	"table_legend":      "the column labels under the table (2 rows)",
	"table_frame":       "the ┌─┐ / └─┘ borders and outer bars (2 rows)",
	"table_dividers":    "the inner │ separators — costs no rows, only noise",
	"header_line0":      "email • project • quota",
	"header_line1":      "model • ctx • cost • time • git",
	"header_merge":      "put both header rows on one line when they fit",
	"bar_width":         "3-20 segments; 5 and 10 pre-round, others stay proportional",
	"bar_style":         `block | line | low | dot | none (percentage instead of a bar)`,
	"alerts":            "the notice row: cache rebuilt, config errors, update available",
	"stale_marker":      `the "~" on per-model figures Claude Code has stopped refreshing`,
	"effort_style":      "glyph (○ ◔ ◑ ◕ ●) | word (xhigh)",
	"model_variant_tag": `keep the "[1m]" tag in the model name`,
	"tutorial_hints":    "rotating tips; alerts surface either way",
	"price_check":       "one optional price/version check a day",
	"no_color":          "plain monochrome output",
	"mode":              "standard | super-compact",

	// [colors]
	"green":   `bars below the notice threshold — "#rrggbb" or raw SGR`,
	"sage":    "healthy band for values shown as text (bar_style = none)",
	"amber":   "notice band for values shown as text (bar_style = none)",
	"yellow":  "notice band on bars, TTL and agent labels",
	"orange":  "warn band",
	"red":     "critical band",
	"cyan":    "git branch and the orchestrator label",
	"magenta": "reasoning-effort indicator",
}

// setScalar updates [table].key to rawValue in the TOML file at path, leaving
// the rest of the file byte-for-byte intact.
//
// rawValue must already be a TOML literal — use tomlBool / tomlInt / tomlString.
//
// Behaviour mirrors the previous per-key implementation:
//   - path missing        → parent directories created, minimal file written.
//   - path invalid TOML   → error; the file is NOT overwritten, so a typo in a
//     hand-edited config can never cost the user the rest of it.
//   - key present         → its value is replaced, indentation and any inline
//     comment preserved.
//   - table without key   → key inserted directly after the table header.
//   - table absent        → table appended at the end of the file.
func setScalar(path, table, key, rawValue string) error {
	data, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return createMinimalFile(path, table, key, rawValue)
		}
		return fmt.Errorf("read %s: %w", path, err)
	}

	if err := toml.Unmarshal(data, Default()); err != nil {
		return fmt.Errorf("existing config is invalid; run 'cc-probeline check-config' for details, then fix or remove %s: %w", path, err)
	}

	return atomicWrite(path, setTOMLValue(data, table, key, rawValue))
}

// setTOMLValue returns data with [table].key set to rawValue, editing only that
// value. The input is assumed to be valid TOML (validated by the caller).
func setTOMLValue(data []byte, table, key, rawValue string) []byte {
	lines := strings.Split(string(data), "\n")
	dotted := table + "." + key

	inTable := false
	headerIdx := -1

	for i, line := range lines {
		trimmed := strings.TrimSpace(line)

		// Top-level dotted form: general.tutorial_hints = …
		if !inTable && isAssignment(trimmed, dotted) {
			lines[i] = replaceTOMLValue(line, rawValue)
			return []byte(strings.Join(lines, "\n"))
		}

		// Table header: [general], [widgets], …
		if strings.HasPrefix(trimmed, "[") && strings.HasSuffix(trimmed, "]") {
			name := strings.TrimSpace(trimmed[1 : len(trimmed)-1])
			inTable = name == table
			if inTable {
				headerIdx = i
			}
			continue
		}

		if inTable && isAssignment(trimmed, key) {
			lines[i] = replaceTOMLValue(line, rawValue)
			return []byte(strings.Join(lines, "\n"))
		}
	}

	// Table present, key absent → insert right after the header so the new key
	// lands inside the table rather than at the end of the file, where it would
	// belong to whichever table happens to be last.
	if headerIdx >= 0 {
		insert := []string{withHint(key, rawValue)}
		lines = append(lines[:headerIdx+1], append(insert, lines[headerIdx+1:]...)...)
		return []byte(strings.Join(lines, "\n"))
	}

	// No such table → append one.
	out := string(data)
	if len(out) > 0 && !strings.HasSuffix(out, "\n") {
		out += "\n"
	}
	out += "\n[" + table + "]\n" + withHint(key, rawValue) + "\n"
	return []byte(out)
}

// withHint renders a freshly inserted assignment, appending the accepted values
// as a comment when one is known for the key.
func withHint(key, rawValue string) string {
	line := key + " = " + rawValue
	if h := insertHint[key]; h != "" {
		return line + "  # " + h
	}
	return line
}

// createMinimalFile writes a new config containing only the version marker and
// the one setting being changed. A full template is deliberately not written:
// the user asked to change one value, not to adopt a generated file.
func createMinimalFile(path, table, key, rawValue string) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return fmt.Errorf("mkdir parent of %s: %w", path, err)
	}
	content := fmt.Sprintf("version = 1\n\n[%s]\n%s = %s\n", table, key, rawValue)
	return atomicWrite(path, []byte(content))
}
