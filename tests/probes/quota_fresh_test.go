// Package probes_test — RED tests for Phase 6.8.b quota freshness.
//
// Tests T-Q3 and T-Q4 verify that QuotaProbe reads from quota.Freshest()
// across sessions and applies bold_red colour when usage exceeds 95%.
//
// T-Q3: probe renders the freshest snapshot stored via quota.Update; when the
//
//	snapshot is stale, renders "as of Xm ago" suffix.
//
// T-Q4: FiveHourPct > 95 or SevenDayPct > 95 → {{color:bold_red}} in raw render;
//
//	at or below 95 → no bold_red marker.
//
// Isolation: CC_PROBELINE_QUOTA_DIR is set to t.TempDir() so tests do not
// touch real user state files.
package probes_test

import (
	"strings"
	"testing"
	"time"

	"github.com/labzink/cc-probeline/internal/probes"
	"github.com/labzink/cc-probeline/internal/quota"
	"github.com/labzink/cc-probeline/internal/renderer"
	"github.com/labzink/cc-probeline/internal/stdin"
)

// TestQuotaProbe_FreshAcrossSessions (T-Q3 / T-12) verifies that QuotaProbe
// renders data from quota.Freshest(), not from d.Stdin.RateLimits alone.
//
// The test simulates the "fresh across sessions" contract:
//  1. A previous session wrote a snapshot with FiveHourPct=67 via quota.Update.
//  2. The current session has d.Stdin.RateLimits with FiveHourPct=30 (stale payload).
//  3. QuotaProbe.Render must output "67" (from Freshest), not "30" (from Stdin).
//
// Sub-case 2 verifies that when the snapshot is older than the staleness
// threshold (> 10 minutes ago), the rendered output contains "as of" and
// a minute-count (e.g. "as of 15m ago").
func TestQuotaProbe_FreshAcrossSessions(t *testing.T) {
	p := &probes.QuotaProbe{}
	cfg := probes.Config{QuotaEnabled: true}
	th := renderer.Theme{} // plain-text; colour tested separately in T-Q4

	// Sub-case A: fresh snapshot (just written) — probe renders freshest pct.
	t.Run("renders_freshest_pct", func(t *testing.T) {
		t.Setenv("CC_PROBELINE_QUOTA_DIR", t.TempDir())

		now := time.Date(2026, 6, 1, 12, 0, 0, 0, time.UTC)
		snap := quota.Snapshot{
			TS:          now.UnixMilli(),
			FiveHourPct: 67.0,
			SevenDayPct: 42.0,
		}
		if err := quota.Update(snap); err != nil {
			t.Fatalf("quota.Update: %v", err)
		}

		// d.Stdin.RateLimits carries stale 30% — probe must prefer Freshest().
		rl := &stdin.RateLimits{
			FiveHour: stdin.RateWindow{UsedPercentage: 30.0},
			SevenDay: stdin.RateWindow{UsedPercentage: 20.0},
		}
		d := probes.Data{
			Now:   now,
			Stdin: stdin.Payload{RateLimits: rl},
		}

		got := p.Render(d, cfg, th, probes.LevelMinimal)
		if !strings.Contains(got, "67") {
			t.Errorf("T-Q3 render_freshest_pct: want '67' (from Freshest) in %q; probe must read from quota.Freshest(), not d.Stdin.RateLimits", got)
		}
		// Must NOT contain the stale payload value '30' as the leading pct.
		if strings.HasPrefix(got, "30") {
			t.Errorf("T-Q3 render_freshest_pct: got stale value '30' as prefix in %q; probe must use Freshest, not Stdin.RateLimits", got)
		}
	})

	// Sub-case B: snapshot older than staleness threshold — output must contain "as of".
	t.Run("stale_snapshot_shows_age", func(t *testing.T) {
		t.Setenv("CC_PROBELINE_QUOTA_DIR", t.TempDir())

		now := time.Date(2026, 6, 1, 12, 0, 0, 0, time.UTC)
		// TS is 15 minutes in the past — exceeds any reasonable staleness threshold.
		staleTS := now.Add(-15 * time.Minute).UnixMilli()
		snap := quota.Snapshot{
			TS:          staleTS,
			FiveHourPct: 55.0,
			SevenDayPct: 33.0,
		}
		if err := quota.Update(snap); err != nil {
			t.Fatalf("quota.Update: %v", err)
		}

		rl := &stdin.RateLimits{
			FiveHour: stdin.RateWindow{UsedPercentage: 55.0},
			SevenDay: stdin.RateWindow{UsedPercentage: 33.0},
		}
		d := probes.Data{
			Now:   now,
			Stdin: stdin.Payload{RateLimits: rl},
		}

		got := p.Render(d, cfg, th, probes.LevelFull)
		if !strings.Contains(got, "as of") {
			t.Errorf("T-Q3 stale_snapshot_shows_age: want 'as of' in render output for stale snapshot, got %q", got)
		}
		// Must contain a minute count (e.g. "15m").
		if !strings.Contains(got, "m ago") {
			t.Errorf("T-Q3 stale_snapshot_shows_age: want 'Xm ago' in render output, got %q", got)
		}
	})
}

// TestQuotaProbe_AgeNoteSuppressed verifies that quota_age_note = false removes
// the "(as of Xm ago)" suffix and changes nothing else.
//
// The note is worth being able to switch off because of what it actually
// measures. Its age comes from DataTS — the last time any of the four stored
// numbers changed — not from when a payload last arrived. The 5h/7d figures
// ride in the status-line payload and are refreshed on every render, so an idle
// session, which spends nothing and therefore moves nothing, grows the age
// indefinitely while the percentages beside it stay exactly right. The suffix
// is sixteen columns wide, which is enough to wrap a merged header.
//
// Both windows keep their bars, percentages and countdowns: the key must reach
// the suffix only, so no one loses a number by quietening a note.
func TestQuotaProbe_AgeNoteSuppressed(t *testing.T) {
	t.Setenv("CC_PROBELINE_QUOTA_DIR", t.TempDir())

	p := &probes.QuotaProbe{}
	th := renderer.Theme{}
	now := time.Date(2026, 6, 1, 12, 0, 0, 0, time.UTC)

	snap := quota.Snapshot{
		TS:          now.Add(-65 * time.Minute).UnixMilli(),
		FiveHourPct: 6.0,
		SevenDayPct: 11.0,
	}
	if err := quota.Update(snap); err != nil {
		t.Fatalf("quota.Update: %v", err)
	}
	d := probes.Data{Now: now, Stdin: stdin.Payload{RateLimits: &stdin.RateLimits{
		FiveHour: stdin.RateWindow{UsedPercentage: 6.0},
		SevenDay: stdin.RateWindow{UsedPercentage: 11.0},
	}}}

	shown := p.Render(d, probes.Config{QuotaEnabled: true}, th, probes.LevelFull)
	if !strings.Contains(shown, "as of") {
		t.Fatalf("setup: a 65-minute-old snapshot must carry the age note, got %q", shown)
	}

	hidden := p.Render(d, probes.Config{QuotaEnabled: true, HideQuotaAgeNote: true}, th, probes.LevelFull)
	if strings.Contains(hidden, "as of") {
		t.Errorf("HideQuotaAgeNote: want no age note, got %q", hidden)
	}
	if want := strings.TrimSuffix(shown, " (as of 65m ago)"); hidden != want {
		t.Errorf("HideQuotaAgeNote must drop the suffix and nothing else:\n got  %q\n want %q", hidden, want)
	}
}

// TestQuotaProbe_BoldRedAbove95 (T-Q4) verifies the bold_red colour rule:
//   - FiveHourPct > 95 → raw Render output contains "{{color:bold_red}}".
//   - SevenDayPct > 95 → raw Render output contains "{{color:bold_red}}".
//   - Both ≤ 95 → raw Render output does NOT contain "{{color:bold_red}}".
//
// The test inspects the raw marker string (before renderer.Apply) because
// the colour contract is expressed at the marker level; Apply converts it
// to an ANSI code in a separate step already tested in colour_test.go.
func TestQuotaProbe_BoldRedAbove95(t *testing.T) {
	p := &probes.QuotaProbe{}
	cfg := probes.Config{QuotaEnabled: true}
	// AnsiEnabled=true and BoldRed populated so that Apply would resolve the marker;
	// but we assert on the raw marker string, not on the post-Apply ANSI code.
	th := renderer.Theme{AnsiEnabled: true, Colors: renderer.DefaultPalette()}

	now := time.Date(2026, 6, 1, 12, 0, 0, 0, time.UTC)
	const boldRedMarker = "{{color:bold_red}}"

	cases := []struct {
		name        string
		fiveHourPct float64
		sevenDayPct float64
		wantBoldRed bool
	}{
		{
			name:        "five_hour_above_95_triggers_bold_red",
			fiveHourPct: 96.0,
			sevenDayPct: 50.0,
			wantBoldRed: true,
		},
		{
			name:        "seven_day_above_95_triggers_bold_red",
			fiveHourPct: 50.0,
			sevenDayPct: 97.5,
			wantBoldRed: true,
		},
		{
			name:        "both_at_95_no_bold_red",
			fiveHourPct: 95.0,
			sevenDayPct: 95.0,
			wantBoldRed: false,
		},
		{
			name:        "both_below_95_no_bold_red",
			fiveHourPct: 60.0,
			sevenDayPct: 40.0,
			wantBoldRed: false,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Setenv("CC_PROBELINE_QUOTA_DIR", t.TempDir())

			snap := quota.Snapshot{
				TS:          now.UnixMilli(),
				FiveHourPct: tc.fiveHourPct,
				SevenDayPct: tc.sevenDayPct,
			}
			if err := quota.Update(snap); err != nil {
				t.Fatalf("quota.Update: %v", err)
			}

			rl := &stdin.RateLimits{
				FiveHour: stdin.RateWindow{UsedPercentage: tc.fiveHourPct},
				SevenDay: stdin.RateWindow{UsedPercentage: tc.sevenDayPct},
			}
			d := probes.Data{
				Now:   now,
				Stdin: stdin.Payload{RateLimits: rl},
			}

			// Inspect raw render output (before Apply) for the marker.
			raw := p.Render(d, cfg, th, probes.LevelMinimal)

			hasBoldRed := strings.Contains(raw, boldRedMarker)
			if tc.wantBoldRed && !hasBoldRed {
				t.Errorf("T-Q4 %s: FiveHourPct=%.1f SevenDayPct=%.1f; want %q in raw render, got %q",
					tc.name, tc.fiveHourPct, tc.sevenDayPct, boldRedMarker, raw)
			}
			if !tc.wantBoldRed && hasBoldRed {
				t.Errorf("T-Q4 %s: FiveHourPct=%.1f SevenDayPct=%.1f ≤ 95; must NOT contain %q in raw render, got %q",
					tc.name, tc.fiveHourPct, tc.sevenDayPct, boldRedMarker, raw)
			}
		})
	}
}

// TestQuotaProbe_BothWindowsRoundHalfUp verifies the payload-fallback display
// rounding: the 5-hour window rounds the shown percentage half-up (99.6
// → "100%") so the number stops parking on 99 right before the wall. Since the
// official-figures alignment, the 7-day window rounds half-up too: every
// official surface (claude.ai settings, the /usage screen) shows the
// server-rounded integer, and truncation sat a point below it. Both windows
// carry the same 99.6 here, so a single render proves the shared rule.
//
// The rounding is display-only: it must NOT arm the paid-overage badge. The
// badge is driven by d.ExtraActive (set in main from the RAW payload pct, never
// from this rounded number), so with ExtraActive unset no "+$"/"extra" appears
// even though the 5h number reads "100%".
func TestQuotaProbe_BothWindowsRoundHalfUp(t *testing.T) {
	t.Setenv("CC_PROBELINE_QUOTA_DIR", t.TempDir())

	p := &probes.QuotaProbe{}
	cfg := probes.Config{QuotaEnabled: true}
	th := renderer.Theme{} // plain-text: assert on bare digits, no colour markers
	now := time.Date(2026, 6, 1, 12, 0, 0, 0, time.UTC)

	snap := quota.Snapshot{
		TS:          now.UnixMilli(),
		FiveHourPct: 99.6,
		SevenDayPct: 99.6,
	}
	if err := quota.Update(snap); err != nil {
		t.Fatalf("quota.Update: %v", err)
	}

	d := probes.Data{Now: now, Stdin: stdin.Payload{RateLimits: &stdin.RateLimits{
		FiveHour: stdin.RateWindow{UsedPercentage: 99.6},
		SevenDay: stdin.RateWindow{UsedPercentage: 99.6},
	}}}

	got := p.Render(d, cfg, th, probes.LevelMinimal)

	// Output is "<5h> · <7d>"; split so each window is asserted independently.
	parts := strings.SplitN(got, " · ", 2)
	if len(parts) != 2 {
		t.Fatalf("render %q: want two windows split by ' · '", got)
	}
	if !strings.Contains(parts[0], "100%") {
		t.Errorf("5h at 99.6 must round half-up to 100%%, got %q", parts[0])
	}
	if !strings.Contains(parts[1], "100%") {
		t.Errorf("7d at 99.6 must round half-up to 100%% like every official surface, got %q", parts[1])
	}
	// Rounded display must not leak into the paid-overage badge.
	if strings.Contains(got, "+$") || strings.Contains(got, "extra") {
		t.Errorf("rounded 5h display must not arm the overage badge, got %q", got)
	}
}

// TestQuotaProbe_OfficialFigureOverridesPayload verifies the official-figures
// alignment: while the usage snapshot is fresh, the displayed number is the
// server-rounded integer from AcctQuota (what claude.ai settings and /usage
// show), not the payload's fractional value — observed live as probeline "11%"
// against an official "12%". The ±2 guard drops the override once the live
// payload has moved materially past the snapshot.
func TestQuotaProbe_OfficialFigureOverridesPayload(t *testing.T) {
	t.Setenv("CC_PROBELINE_QUOTA_DIR", t.TempDir()) // hermetic: no machine snapshot
	p := &probes.QuotaProbe{}
	cfg := probes.Config{}
	th := renderer.Theme{}
	now := time.Date(2026, 8, 17, 12, 0, 0, 0, time.UTC)

	official5h, official7d := 8.0, 12.0
	d := probes.Data{
		Now: now,
		Stdin: stdin.Payload{RateLimits: &stdin.RateLimits{
			FiveHour: stdin.RateWindow{UsedPercentage: 7.2},
			SevenDay: stdin.RateWindow{UsedPercentage: 11.7},
		}},
		AcctQuota5h: &official5h,
		AcctQuota7d: &official7d,
	}

	got := p.Render(d, cfg, th, probes.LevelMinimal)
	parts := strings.SplitN(got, " · ", 2)
	if len(parts) != 2 {
		t.Fatalf("render %q: want two windows split by ' · '", got)
	}
	if !strings.Contains(parts[0], "8%") {
		t.Errorf("5h: want official 8%% over payload 7.2, got %q", parts[0])
	}
	if !strings.Contains(parts[1], "12%") {
		t.Errorf("7d: want official 12%% over payload 11.7, got %q", parts[1])
	}

	// bar_style none prints the number via usageBar rather than pctSuffix —
	// the path the default block-bar style hides. The official figure must win
	// there too (this is exactly where the live 11-vs-12 drift was observed).
	cfg.BarStyle = "none"
	got = p.Render(d, cfg, th, probes.LevelFull)
	parts = strings.SplitN(got, " · ", 2)
	if len(parts) != 2 {
		t.Fatalf("no-bar render %q: want two windows split by ' · '", got)
	}
	if !strings.Contains(parts[0], "8%") {
		t.Errorf("no-bar 5h: want official 8%% over payload 7.2, got %q", parts[0])
	}
	if !strings.Contains(parts[1], "12%") {
		t.Errorf("no-bar 7d: want official 12%% over payload 11.7, got %q", parts[1])
	}
	cfg.BarStyle = ""

	// Divergence guard: payload far past the snapshot → the snapshot no longer
	// speaks for the present; fall back to the rounded live value.
	d.Stdin.RateLimits.SevenDay.UsedPercentage = 15.4
	got = p.Render(d, cfg, th, probes.LevelMinimal)
	parts = strings.SplitN(got, " · ", 2)
	if !strings.Contains(parts[1], "15%") {
		t.Errorf("7d diverged: want rounded payload 15%%, got %q", parts[1])
	}
}
