// Package claudejson_test — model-scoped rate-limit windows read from the usage
// snapshot Claude Code caches in ~/.claude.json.
//
// The fixtures below mirror the real file's shape: cachedUsageUtilization holds
// a fetchedAtMs stamp and a limits array in which each entry is tagged by kind
// ("session", "weekly_all", "weekly_scoped") and only the scoped ones carry a
// model name.
package claudejson_test

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/labzink/cc-probeline/internal/claudejson"
)

// writeUsage writes body as ~/.claude.json for the test and points the reader at
// it. Each call uses a fresh directory so the package-level mtime cache cannot
// serve a previous test's value.
func writeUsage(t *testing.T, body string) string {
	t.Helper()
	p := filepath.Join(t.TempDir(), ".claude.json")
	if err := os.WriteFile(p, []byte(body), 0o600); err != nil {
		t.Fatalf("write fixture: %v", err)
	}
	t.Setenv("CC_PROBELINE_CLAUDE_JSON", p)
	return p
}

// realShapeUsage reproduces the live file: a session window, an all-models
// weekly window, and one model-scoped weekly window for Fable.
const realShapeUsage = `{
  "oauthAccount": {"accessToken": "secret-must-be-ignored"},
  "cachedUsageUtilization": {
    "fetchedAtMs": 1785427359130,
    "accountUuid": "c9ac2812-eac2-43ff-b825-dc71a9fdf5c4",
    "utilization": {
      "five_hour": {"utilization": 10, "resets_at": "2026-07-30T17:20:00.589026+00:00"},
      "seven_day": {"utilization": 9, "resets_at": "2026-08-06T01:00:00.589045+00:00"},
      "limits": [
        {"kind": "session", "group": "session", "percent": 10,
         "resets_at": "2026-07-30T17:20:00.589026+00:00", "scope": null},
        {"kind": "weekly_all", "group": "weekly", "percent": 9,
         "resets_at": "2026-08-06T01:00:00.589045+00:00", "scope": null},
        {"kind": "weekly_scoped", "group": "weekly", "percent": 5,
         "resets_at": "2026-08-06T00:59:59.589374+00:00",
         "scope": {"model": {"id": null, "display_name": "Fable"}, "surface": null}}
      ]
    }
  }
}`

// T-MQ1: the scoped window is extracted, and only that one — the account-wide
// session/weekly entries must not leak into the result.
func TestScopedWeekly_ExtractsOnlyScopedWindows(t *testing.T) {
	writeUsage(t, realShapeUsage)

	limits, fetchedAt := claudejson.ScopedWeekly()

	if len(limits) != 1 {
		t.Fatalf("expected exactly the scoped window, got %d: %+v", len(limits), limits)
	}
	l := limits[0]
	if l.Model != "Fable" {
		t.Errorf("Model: got %q, want %q", l.Model, "Fable")
	}
	if l.Percent != 5 {
		t.Errorf("Percent: got %v, want 5", l.Percent)
	}
	want := time.Date(2026, 8, 6, 0, 59, 59, 589374000, time.UTC)
	if !l.ResetsAt.Equal(want) {
		t.Errorf("ResetsAt: got %v, want %v", l.ResetsAt, want)
	}
	if got := fetchedAt.UnixMilli(); got != 1785427359130 {
		t.Errorf("fetchedAt: got %d, want 1785427359130", got)
	}
}

// T-MQ2: an account with no model-scoped limit yields nothing, so the probe can
// stay invisible rather than render an empty label.
func TestScopedWeekly_NoScopedWindow(t *testing.T) {
	writeUsage(t, `{"cachedUsageUtilization": {"fetchedAtMs": 1785427359130,
	  "utilization": {"limits": [
	    {"kind": "session", "percent": 10, "resets_at": "2026-07-30T17:20:00Z", "scope": null},
	    {"kind": "weekly_all", "percent": 9, "resets_at": "2026-08-06T01:00:00Z", "scope": null}
	  ]}}}`)

	if limits, _ := claudejson.ScopedWeekly(); len(limits) != 0 {
		t.Errorf("expected no scoped windows, got %+v", limits)
	}
}

// T-MQ3: several scoped windows are all returned, in file order.
func TestScopedWeekly_MultipleModels(t *testing.T) {
	writeUsage(t, `{"cachedUsageUtilization": {"utilization": {"limits": [
	  {"kind": "weekly_scoped", "percent": 5, "resets_at": "2026-08-06T00:59:59Z",
	   "scope": {"model": {"display_name": "Fable"}}},
	  {"kind": "weekly_scoped", "percent": 42, "resets_at": "2026-08-06T00:59:59Z",
	   "scope": {"model": {"display_name": "Opus"}}}
	]}}}`)

	limits, _ := claudejson.ScopedWeekly()
	if len(limits) != 2 {
		t.Fatalf("expected 2 scoped windows, got %d: %+v", len(limits), limits)
	}
	if limits[0].Model != "Fable" || limits[1].Model != "Opus" {
		t.Errorf("wrong models or order: %+v", limits)
	}
	if limits[1].Percent != 42 {
		t.Errorf("second window percent: got %v, want 42", limits[1].Percent)
	}
}

// T-MQ4: a scoped entry with no usable model name is skipped — there would be
// nothing to label its bar with.
func TestScopedWeekly_SkipsUnnamedScope(t *testing.T) {
	writeUsage(t, `{"cachedUsageUtilization": {"utilization": {"limits": [
	  {"kind": "weekly_scoped", "percent": 5, "resets_at": "2026-08-06T00:59:59Z", "scope": null},
	  {"kind": "weekly_scoped", "percent": 6, "resets_at": "2026-08-06T00:59:59Z",
	   "scope": {"model": {"display_name": ""}}}
	]}}}`)

	if limits, _ := claudejson.ScopedWeekly(); len(limits) != 0 {
		t.Errorf("unnamed scopes must be skipped, got %+v", limits)
	}
}

// T-MQ5: an unparseable resets_at leaves the countdown unknown but keeps the
// window — the percentage is still worth showing.
func TestScopedWeekly_BadResetsAtKeepsWindow(t *testing.T) {
	writeUsage(t, `{"cachedUsageUtilization": {"utilization": {"limits": [
	  {"kind": "weekly_scoped", "percent": 5, "resets_at": "not-a-timestamp",
	   "scope": {"model": {"display_name": "Fable"}}}
	]}}}`)

	limits, _ := claudejson.ScopedWeekly()
	if len(limits) != 1 {
		t.Fatalf("window dropped over a bad timestamp: %+v", limits)
	}
	if !limits[0].ResetsAt.IsZero() {
		t.Errorf("ResetsAt should be zero when unparseable, got %v", limits[0].ResetsAt)
	}
}

// T-MQ6: fail-soft — a missing file or invalid JSON must never panic, and must
// keep serving the last good value.
//
// Claude Code rewrites ~/.claude.json constantly, so a read that lands mid-write
// is routine; blanking the segment on every such race would make it flicker.
// The contract therefore mirrors HasExtraUsageEnabled: on any read/parse failure
// the previously cached value stands.
func TestScopedWeekly_FailSoft(t *testing.T) {
	// Establish a known-good cached value first.
	writeUsage(t, realShapeUsage)
	good, _ := claudejson.ScopedWeekly()
	if len(good) != 1 {
		t.Fatalf("fixture: expected one scoped window, got %+v", good)
	}

	t.Run("missing file keeps last good value", func(t *testing.T) {
		t.Setenv("CC_PROBELINE_CLAUDE_JSON", filepath.Join(t.TempDir(), "absent.json"))
		limits, _ := claudejson.ScopedWeekly()
		if len(limits) != 1 || limits[0].Model != "Fable" {
			t.Errorf("missing file should keep the cached window, got %+v", limits)
		}
	})

	t.Run("invalid json keeps last good value", func(t *testing.T) {
		writeUsage(t, `{"cachedUsageUtilization": {`)
		limits, _ := claudejson.ScopedWeekly()
		if len(limits) != 1 || limits[0].Model != "Fable" {
			t.Errorf("invalid json should keep the cached window, got %+v", limits)
		}
	})

	// A well-formed file that simply has no scoped window is NOT a failure: it
	// is an authoritative "this account has none", and must clear the value.
	t.Run("valid file without scoped window clears it", func(t *testing.T) {
		writeUsage(t, `{"oauthAccount": {"hasExtraUsageEnabled": true}}`)
		if limits, _ := claudejson.ScopedWeekly(); len(limits) != 0 {
			t.Errorf("a valid file with no scoped window must clear the value, got %+v", limits)
		}
	})
}

// T-MQ-MC: multi-client setups select the subscription via CLAUDE_CONFIG_DIR
// (e.g. `claude2() { CLAUDE_CONFIG_DIR=$HOME/.claude2 command claude "$@"; }`).
// The status line inherits that variable from the client that spawned it, and
// the reader must follow it to that client's own .claude.json instead of the
// default ~/.claude.json — otherwise every client renders subscription 1's
// figures.
func TestScopedWeekly_HonoursClaudeConfigDir(t *testing.T) {
	t.Setenv("CC_PROBELINE_CLAUDE_JSON", "") // fall through to real resolution

	cfgDir := t.TempDir()
	body := `{"cachedUsageUtilization": {"fetchedAtMs": 1786942169858, "utilization": {"limits": [
	  {"kind": "weekly_scoped", "percent": 22,
	   "resets_at": "2026-08-20T01:00:00.000000+00:00",
	   "scope": {"model": {"display_name": "Fable"}}}
	]}}}`
	if err := os.WriteFile(filepath.Join(cfgDir, ".claude.json"), []byte(body), 0o600); err != nil {
		t.Fatalf("write fixture: %v", err)
	}
	// HOME points at a directory with NO .claude.json: if the resolver ignored
	// CLAUDE_CONFIG_DIR it would find nothing and return the cached previous
	// value, not this fixture's 22%.
	t.Setenv("HOME", t.TempDir())
	t.Setenv("CLAUDE_CONFIG_DIR", cfgDir)

	limits, _ := claudejson.ScopedWeekly()

	if len(limits) != 1 || limits[0].Model != "Fable" || limits[0].Percent != 22 {
		t.Fatalf("expected Fable 22%% from CLAUDE_CONFIG_DIR fixture, got %+v", limits)
	}
}
