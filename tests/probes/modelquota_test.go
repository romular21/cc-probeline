// Package probes_test — ModelQuotaProbe: the per-model weekly windows shown
// next to the account-wide quota bars ("Fable 7d: ▒░░░░░░░░░ ↻ 6d.8h").
//
// The probe renders from probes.Data, which main fills from Claude Code's
// cached usage snapshot — so these tests need no files and no environment.
package probes_test

import (
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/labzink/cc-probeline/internal/claudejson"
	"github.com/labzink/cc-probeline/internal/probes"
	"github.com/labzink/cc-probeline/internal/renderer"
	"github.com/labzink/cc-probeline/internal/stdin"
)

// mqNow is the fixed clock these tests render against.
var mqNow = time.Date(2026, 7, 30, 12, 0, 0, 0, time.UTC)

// mqData returns Data carrying one Fable window at pct, resetting in 6 days.
func mqData(pct float64) probes.Data {
	return probes.Data{
		Now: mqNow,
		ScopedQuota: []claudejson.ScopedLimit{
			{Model: "Fable", Percent: pct, ResetsAt: mqNow.Add(6 * 24 * time.Hour)},
		},
		ScopedQuotaFetchedAt: mqNow.Add(-1 * time.Minute),
	}
}

// mqEnabled is a Config with the segment switched on.
var mqEnabled = probes.Config{ModelQuotaEnabled: true}

// plainTheme renders without ANSI so assertions compare bare text.
var plainTheme = renderer.Theme{AnsiEnabled: false}

// ─── visibility ──────────────────────────────────────────────────────────────

// T-MQP1: no scoped window in the snapshot → the segment does not appear at all,
// rather than rendering an empty label. Accounts without a per-model limit must
// see no change to their status line.
func TestModelQuota_HiddenWithoutData(t *testing.T) {
	p := &probes.ModelQuotaProbe{}

	if p.Visible(probes.Data{Now: mqNow}, mqEnabled) {
		t.Error("probe is visible with no scoped windows")
	}
	if got := p.Render(probes.Data{Now: mqNow}, mqEnabled, plainTheme, probes.LevelFull); got != "" {
		t.Errorf("Render with no data: got %q, want empty", got)
	}
}

// T-MQP2: the [widgets].quota_model toggle hides it even when data exists.
func TestModelQuota_HiddenWhenDisabled(t *testing.T) {
	p := &probes.ModelQuotaProbe{}

	if p.Visible(mqData(5), probes.Config{ModelQuotaEnabled: false}) {
		t.Error("probe is visible while quota_model is off")
	}
	if !p.Visible(mqData(5), mqEnabled) {
		t.Error("probe is hidden while quota_model is on and data exists")
	}
}

// ─── labelling ───────────────────────────────────────────────────────────────

// T-MQP3: the label must name both the model and the window, so a scoped bar can
// never be read as the account-wide one sitting next to it.
func TestModelQuota_LabelNamesModelAndWindow(t *testing.T) {
	got := (&probes.ModelQuotaProbe{}).Render(mqData(5), mqEnabled, plainTheme, probes.LevelFull)

	if !strings.HasPrefix(got, "Fable 7d: ") {
		t.Errorf("Full render must start with the model and window label, got %q", got)
	}
	if !strings.Contains(got, "↻") {
		t.Errorf("Full render is missing the reset countdown: %q", got)
	}
}

// T-MQP4: each level keeps the model identifiable while shrinking.
func TestModelQuota_LevelsStayIdentifiable(t *testing.T) {
	p := &probes.ModelQuotaProbe{}
	d := mqData(5)

	full := p.Render(d, mqEnabled, plainTheme, probes.LevelFull)
	compact := p.Render(d, mqEnabled, plainTheme, probes.LevelCompact)
	minimal := p.Render(d, mqEnabled, plainTheme, probes.LevelMinimal)

	if !strings.Contains(compact, "Fable") {
		t.Errorf("Compact dropped the model name: %q", compact)
	}
	if !strings.HasPrefix(minimal, "F ") {
		t.Errorf("Minimal must keep the model initial, got %q", minimal)
	}
	if !strings.Contains(minimal, "5%") {
		t.Errorf("Minimal must carry the percentage, got %q", minimal)
	}
	// Each level must be strictly shorter than the one above it.
	if !(len(full) > len(compact) && len(compact) > len(minimal)) {
		t.Errorf("levels do not shrink: full=%d compact=%d minimal=%d\n%q\n%q\n%q",
			len(full), len(compact), len(minimal), full, compact, minimal)
	}
}

// ─── several models ──────────────────────────────────────────────────────────

// T-MQP5: multiple scoped windows are all rendered, separated like the
// account-wide bars are.
func TestModelQuota_RendersEveryModel(t *testing.T) {
	d := probes.Data{
		Now: mqNow,
		ScopedQuota: []claudejson.ScopedLimit{
			{Model: "Fable", Percent: 5, ResetsAt: mqNow.Add(6 * 24 * time.Hour)},
			{Model: "Opus", Percent: 42, ResetsAt: mqNow.Add(6 * 24 * time.Hour)},
		},
	}

	got := (&probes.ModelQuotaProbe{}).Render(d, mqEnabled, plainTheme, probes.LevelFull)
	for _, want := range []string{"Fable 7d:", "Opus 7d:", " · "} {
		if !strings.Contains(got, want) {
			t.Errorf("render is missing %q: %q", want, got)
		}
	}
}

// ─── staleness ───────────────────────────────────────────────────────────────

// T-MQP6: the age marker is tied to Claude Code's own one-hour read TTL — it
// refreshes this cache from live API responses (throttled to one write per five
// minutes) and discards the snapshot past an hour. Anything younger is what
// Claude Code itself would act on, so flagging it would be nagging; anything
// older is what Claude Code would throw away, so it must be called out.
func TestModelQuota_AgeMarkerFollowsClaudeCodeTTL(t *testing.T) {
	p := &probes.ModelQuotaProbe{}

	// Well inside the TTL, and just inside it: silent in both cases.
	for _, age := range []time.Duration{1 * time.Minute, 45 * time.Minute} {
		d := mqData(5)
		d.ScopedQuotaFetchedAt = mqNow.Add(-age)
		if got := p.Render(d, mqEnabled, plainTheme, probes.LevelFull); strings.Contains(got, "old") {
			t.Errorf("a %v-old snapshot is still current for Claude Code; must not be marked: %q", age, got)
		}
	}

	// Past the TTL: marked, and compactly.
	stale := mqData(5)
	stale.ScopedQuotaFetchedAt = mqNow.Add(-2 * time.Hour)
	got := p.Render(stale, mqEnabled, plainTheme, probes.LevelFull)
	if !strings.HasSuffix(got, " · 2h old") {
		t.Errorf("a 2h-old snapshot must carry a compact age marker, got %q", got)
	}
}

// T-MQP7: the age marker never appears at the tighter levels, where the line is
// already fighting for room.
func TestModelQuota_AgeMarkerOnlyAtFull(t *testing.T) {
	p := &probes.ModelQuotaProbe{}
	stale := mqData(5)
	stale.ScopedQuotaFetchedAt = mqNow.Add(-2 * time.Hour)

	for _, lvl := range []probes.Level{probes.LevelCompact, probes.LevelMinimal} {
		if got := p.Render(stale, mqEnabled, plainTheme, lvl); strings.Contains(got, "old") {
			t.Errorf("level %v must not carry the age marker: %q", lvl, got)
		}
	}
}

// ─── shared reset ────────────────────────────────────────────────────────────

// mqWithAccountReset returns Data whose account-wide 7d window resets at
// accountReset, alongside a Fable window resetting at fableReset.
func mqWithAccountReset(accountReset, fableReset time.Time) probes.Data {
	raw, _ := json.Marshal(accountReset.Unix())
	return probes.Data{
		Now: mqNow,
		Stdin: stdin.Payload{
			RateLimits: &stdin.RateLimits{
				SevenDay: stdin.RateWindow{UsedPercentage: 9, ResetsAt: raw},
			},
		},
		ScopedQuota: []claudejson.ScopedLimit{
			{Model: "Fable", Percent: 5, ResetsAt: fableReset},
		},
		ScopedQuotaFetchedAt: mqNow,
	}
}

// T-MQP9: the scoped weekly window rolls over with the account-wide one — Claude
// Code stamps the two about a second apart — so repeating the countdown would
// spend a dozen columns restating what the 7d block beside it already says.
func TestModelQuota_SharedResetIsNotRepeated(t *testing.T) {
	reset := mqNow.Add(6 * 24 * time.Hour)
	// One second apart, exactly as the real snapshot records them.
	d := mqWithAccountReset(reset, reset.Add(-time.Second))

	got := (&probes.ModelQuotaProbe{}).Render(d, mqEnabled, plainTheme, probes.LevelFull)

	if strings.Contains(got, "↻") {
		t.Errorf("countdown repeated although both windows share a rollover: %q", got)
	}
	if !strings.HasPrefix(got, "Fable 7d: ") {
		t.Errorf("label lost with the countdown: %q", got)
	}
}

// T-MQP10: an account whose scoped window genuinely rolls over at a different
// time still gets its own countdown — suppression must be about duplication,
// not about hiding information.
func TestModelQuota_DivergentResetKeepsCountdown(t *testing.T) {
	accountReset := mqNow.Add(6 * 24 * time.Hour)
	d := mqWithAccountReset(accountReset, accountReset.Add(-48*time.Hour))

	got := (&probes.ModelQuotaProbe{}).Render(d, mqEnabled, plainTheme, probes.LevelFull)

	if !strings.Contains(got, "↻") {
		t.Errorf("a window resetting two days earlier must keep its countdown: %q", got)
	}
}

// T-MQP11: with no account reset to compare against, the scoped countdown is
// kept — dropping it would lose the only rollover information on the line.
func TestModelQuota_UnknownAccountResetKeepsCountdown(t *testing.T) {
	d := mqData(5) // no RateLimits in Stdin

	got := (&probes.ModelQuotaProbe{}).Render(d, mqEnabled, plainTheme, probes.LevelFull)

	if !strings.Contains(got, "↻") {
		t.Errorf("countdown dropped although no account reset was known: %q", got)
	}
}

// T-MQP12: a window with no parseable reset time still renders its percentage —
// the countdown degrades to "??m" rather than dropping the bar.
func TestModelQuota_UnknownResetStillRenders(t *testing.T) {
	d := probes.Data{
		Now:         mqNow,
		ScopedQuota: []claudejson.ScopedLimit{{Model: "Fable", Percent: 5}},
	}

	got := (&probes.ModelQuotaProbe{}).Render(d, mqEnabled, plainTheme, probes.LevelFull)
	if !strings.HasPrefix(got, "Fable 7d: ") {
		t.Errorf("window dropped when reset time is unknown: %q", got)
	}
	if !strings.Contains(got, "??m") {
		t.Errorf("unknown reset should render ??m, got %q", got)
	}
}
