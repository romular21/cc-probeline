package claudejson

// usage.go reads the model-scoped rate-limit windows that Claude Code caches in
// ~/.claude.json under "cachedUsageUtilization" — the same numbers its /usage
// screen shows as "Current week (Fable)".
//
// Why this file exists: the status-line payload Claude Code pipes to us carries
// only the account-wide five_hour / seven_day windows. Per-model windows never
// reach the payload, so the cached snapshot on disk is the only local source.
// Reading it keeps rendering fully offline — no credentials, no network.
//
// Security note (same contract as claudejson.go): ~/.claude.json holds OAuth
// tokens. The structs below decode ONLY the usage fields; every other key is
// dropped by the decoder. File contents are never logged — only the fact of a
// failure is.

import (
	"encoding/json"
	"log/slog"
	"os"
	"sync"
	"time"
)

// kindWeeklyScoped is the "limits" entry kind Claude Code uses for a window that
// applies to one model rather than the whole account.
const kindWeeklyScoped = "weekly_scoped"

// ScopedLimit is one model-scoped rate-limit window.
type ScopedLimit struct {
	// Model is the human-readable model name as Claude Code labels it
	// (scope.model.display_name — e.g. "Fable"). Never empty.
	Model string
	// Percent is the share of the window already used (0–100).
	Percent float64
	// ResetsAt is when the window rolls over. Zero when Claude Code did not
	// record a reset time.
	ResetsAt time.Time
}

// usageFile decodes only cachedUsageUtilization from ~/.claude.json.
type usageFile struct {
	Cached struct {
		FetchedAtMs int64 `json:"fetchedAtMs"`
		Utilization struct {
			FiveHour *struct {
				Utilization *float64 `json:"utilization"`
			} `json:"five_hour"`
			SevenDay *struct {
				Utilization *float64 `json:"utilization"`
			} `json:"seven_day"`
			Limits []struct {
				Kind     string  `json:"kind"`
				Percent  float64 `json:"percent"`
				ResetsAt string  `json:"resets_at"`
				Scope    *struct {
					Model *struct {
						DisplayName string `json:"display_name"`
					} `json:"model"`
				} `json:"scope"`
			} `json:"limits"`
		} `json:"utilization"`
	} `json:"cachedUsageUtilization"`
}

// usageCacheEntry holds the parsed scoped limits and the mtime they were read at.
type usageCacheEntry struct {
	mu        sync.Mutex
	limits    []ScopedLimit
	acct5h    *float64
	acct7d    *float64
	fetchedAt time.Time
	mtime     time.Time
	valid     bool
}

// usageCache is the package-level mtime cache for the usage snapshot. It is
// separate from pkgCache so the two readers cannot invalidate each other.
var usageCache usageCacheEntry

// ScopedWeekly returns the model-scoped rate-limit windows from the cached
// usage snapshot, together with the moment Claude Code fetched them.
//
// Fail-soft contract, matching HasExtraUsageEnabled:
//   - File missing, unreadable, invalid, or carrying no scoped window → nil.
//   - A previously cached value is returned when the file goes temporarily away.
//   - Only the fact of a failure is logged, never the file contents.
//
// mtime-cache: the file is re-read only when its mtime changed since the last
// successful read. ~/.claude.json is rewritten often, but parsing it is cheap
// next to the render budget, and the cache keeps the common case free.
func ScopedWeekly() (limits []ScopedLimit, fetchedAt time.Time) {
	usageCache.mu.Lock()
	defer usageCache.mu.Unlock()
	refreshUsageLocked()
	return usageCache.limits, usageCache.fetchedAt
}

// AccountWindows returns the account-wide five-hour and seven-day used
// percentages from the cached usage snapshot, together with the moment Claude
// Code fetched them. These are the server-rounded integers every official
// surface displays — the claude.ai settings page, the /usage screen — unlike
// the fractional used_percentage the status-line payload carries. ok is false
// when the snapshot is absent or carries no account windows.
//
// Same fail-soft and mtime-cache contract as ScopedWeekly.
func AccountWindows() (fiveHour, sevenDay float64, fetchedAt time.Time, ok bool) {
	usageCache.mu.Lock()
	defer usageCache.mu.Unlock()
	refreshUsageLocked()
	if usageCache.acct5h == nil || usageCache.acct7d == nil {
		return 0, 0, time.Time{}, false
	}
	return *usageCache.acct5h, *usageCache.acct7d, usageCache.fetchedAt, true
}

// refreshUsageLocked re-reads the usage snapshot when its mtime moved,
// populating the package cache. Caller must hold usageCache.mu.
func refreshUsageLocked() {
	p := claudeJSONPath()
	if p == "" {
		slog.Warn("claudejson: HOME not set; cannot locate ~/.claude.json")
		return
	}

	fi, err := os.Stat(p)
	if err != nil {
		if !os.IsNotExist(err) {
			slog.Warn("claudejson: usage stat failed")
		}
		return
	}

	mtime := fi.ModTime()
	if usageCache.valid && mtime.Equal(usageCache.mtime) {
		return
	}

	data, err := os.ReadFile(p)
	if err != nil {
		slog.Warn("claudejson: usage read failed")
		return
	}

	var parsed usageFile
	if err := json.Unmarshal(data, &parsed); err != nil {
		slog.Warn("claudejson: usage parse failed")
		return
	}

	out := make([]ScopedLimit, 0, 2)
	for _, l := range parsed.Cached.Utilization.Limits {
		if l.Kind != kindWeeklyScoped {
			continue
		}
		if l.Scope == nil || l.Scope.Model == nil || l.Scope.Model.DisplayName == "" {
			// A scoped window with no model name has nothing to label it with.
			continue
		}
		sl := ScopedLimit{
			Model:   l.Scope.Model.DisplayName,
			Percent: l.Percent,
		}
		// resets_at carries a fractional-second RFC 3339 stamp
		// ("2026-08-06T00:59:59.589374+00:00"); a stamp we cannot parse simply
		// leaves the countdown unknown rather than dropping the whole window.
		if l.ResetsAt != "" {
			if ts, err := time.Parse(time.RFC3339Nano, l.ResetsAt); err == nil {
				sl.ResetsAt = ts.UTC()
			}
		}
		out = append(out, sl)
	}

	usageCache.limits = out
	if ms := parsed.Cached.FetchedAtMs; ms > 0 {
		usageCache.fetchedAt = time.UnixMilli(ms)
	} else {
		usageCache.fetchedAt = time.Time{}
	}
	usageCache.acct5h, usageCache.acct7d = nil, nil
	if u := parsed.Cached.Utilization.FiveHour; u != nil && u.Utilization != nil {
		usageCache.acct5h = u.Utilization
	}
	if u := parsed.Cached.Utilization.SevenDay; u != nil && u.Utilization != nil {
		usageCache.acct7d = u.Utilization
	}
	usageCache.mtime = mtime
	usageCache.valid = true
}
