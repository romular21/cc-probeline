package config

// sethints.go — line-level TOML editing primitives shared by every setter.
//
// BL-15 introduced surgical editing so that toggling one value keeps comments,
// blank lines, key order and unknown sections byte-for-byte intact. It was
// originally written for tutorial_hints alone; the generalised entry point now
// lives in setinplace.go and every setter goes through it. What remains here
// are the primitives: recognising an assignment, replacing the value on a key
// line, and writing the result atomically.

import (
	"os"
	"strings"
)

// isAssignment reports whether a trimmed line assigns to key (i.e. starts with
// key followed by optional spaces and '='), excluding comment lines.
func isAssignment(trimmed, key string) bool {
	if strings.HasPrefix(trimmed, "#") {
		return false
	}
	rest := strings.TrimPrefix(trimmed, key)
	if rest == trimmed {
		return false
	}
	rest = strings.TrimLeft(rest, " \t")
	return strings.HasPrefix(rest, "=")
}

// replaceTOMLValue replaces the value after '=' on a key line with val,
// preserving the key, its indentation, and any trailing inline comment.
//
// The comment is found with commentStart, which ignores a '#' inside a quoted
// string — colour values are written as "#rrggbb", and treating that hash as a
// comment marker would swallow the value and leave a stray quote behind.
func replaceTOMLValue(line, val string) string {
	eq := strings.Index(line, "=")
	if eq < 0 {
		return line
	}
	head := line[:eq+1]
	rest := line[eq+1:]

	comment := ""
	if h := commentStart(rest); h >= 0 {
		comment = " " + strings.TrimSpace(rest[h:])
	}
	return head + " " + val + comment
}

// commentStart returns the index of the '#' that begins an inline comment in s,
// or -1 when there is none. A '#' inside a single- or double-quoted string is
// part of the value, not a comment. Escapes are not interpreted: TOML basic
// strings allow \" but no setter writes one, and treating a backslash-quote as
// closing the string would only ever end the scan early on input we do not
// produce.
func commentStart(s string) int {
	var quote rune
	for i, r := range s {
		switch {
		case quote != 0:
			if r == quote {
				quote = 0
			}
		case r == '"' || r == '\'':
			quote = r
		case r == '#':
			return i
		}
	}
	return -1
}

// GlobalConfigPath returns the platform-appropriate global config location
// (XDG / HOME / APPDATA cascade). Re-exports the unexported globalConfigPath
// from path.go. Returns "" when no location can be determined.
func GlobalConfigPath() string { return globalConfigPath() }

// atomicWrite writes content to path via a .tmp sibling, then renames.
// os.Rename is atomic on POSIX (same-FS) and uses MoveFileEx on Windows.
// If WriteFile fails, path is not touched. If Rename fails, the .tmp file is
// removed (best-effort) to avoid leaving orphaned temporaries on disk.
// KISS: no flock (TOML editing is rare).
func atomicWrite(path string, content []byte) error {
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, content, 0o644); err != nil {
		return err
	}
	if err := os.Rename(tmp, path); err != nil {
		os.Remove(tmp) // best-effort cleanup; ignore error
		return err
	}
	return nil
}
