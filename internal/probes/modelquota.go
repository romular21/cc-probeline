package probes

import (
	"fmt"
	"time"

	"github.com/labzink/cc-probeline/internal/claudejson"
	"github.com/labzink/cc-probeline/internal/quota"
	"github.com/labzink/cc-probeline/internal/renderer"
)

// ModelQuotaProbe renders the per-model weekly rate-limit windows — the ones
// Claude Code's /usage screen labels "Current week (Fable)".
//
// These windows never reach the status-line payload (it carries only the
// account-wide five_hour / seven_day figures), so the numbers come from the
// snapshot Claude Code caches in ~/.claude.json. Reading it keeps the render
// path offline: no credentials, no network.
//
// It sits next to QuotaProbe on line 0 and is labelled with the model name plus
// the window ("Fable 7d:"), so a scoped bar can never be mistaken for the
// account-wide one beside it.
//
// Priority P1: it drops before the account-wide quota when the terminal is
// narrow, since the overall limit is the one you must not miss.
type ModelQuotaProbe struct{}

func (p *ModelQuotaProbe) Name() string  { return "quota_model" }
func (p *ModelQuotaProbe) Priority() int { return 1 }
func (p *ModelQuotaProbe) MinWidth() int { return len("F 0%") }

// modelQuotaStale is how old the cached usage snapshot may get before the
// figures are marked as aged.
//
// The value mirrors Claude Code's own contract rather than a taste call. It
// refreshes cachedUsageUtilization from live API responses, throttled to one
// write per 5 minutes, and its own reader discards the snapshot once it is
// older than an hour. So under an hour the figures are exactly what Claude
// Code itself would act on — flagging them earlier would nag about data that
// is, by its owner's definition, current. Past the hour the snapshot is what
// Claude Code would throw away, and saying so is the honest move.
const modelQuotaStale = 60 * time.Minute

// Visible reports whether the snapshot holds at least one model-scoped window.
// Accounts without a per-model limit (or Claude Code versions that do not cache
// one) render nothing at all rather than an empty label.
func (p *ModelQuotaProbe) Visible(d Data, c Config) bool {
	return c.ModelQuotaEnabled && len(d.ScopedQuota) > 0
}

// Render formats each model-scoped weekly window.
//
// Display levels:
//
//	Full:    "Fable 7d: <bar10> <reset>" [+ age suffix]
//	Compact: "Fable <bar5> <reset>"      [+ age suffix]
//	Minimal: "F <pct>%"
//
// Colours follow the same notice/warn/critical ratios as the account-wide 7d
// window, so a scoped bar reads identically to the one beside it.
func (p *ModelQuotaProbe) Render(d Data, c Config, t renderer.Theme, level Level) string {
	limits, fetchedAt := d.ScopedQuota, d.ScopedQuotaFetchedAt
	if len(limits) == 0 {
		return ""
	}

	notice, warn, critical := resolveRatios(c.Quota7dNoticeRatio, c.Quota7dWarnRatio, c.Quota7dCriticalRatio)

	colourReset := ""
	if t.AnsiEnabled {
		colourReset = t.Colors.Reset
	}

	// A scoped weekly window rolls over with the account-wide one — the two
	// timestamps differ by under a second in practice — so repeating the
	// countdown would spend a dozen columns restating what the 7d block beside
	// it already says. Resolve the account reset and suppress the scoped one
	// whenever they coincide; an account whose windows genuinely diverge still
	// gets both.
	acctReset, acctKnown := accountWeeklyReset(d)

	out := ""
	for i, l := range limits {
		if i > 0 {
			out += " · "
		}
		showReset := !acctKnown || l.ResetsAt.IsZero() ||
			absDuration(l.ResetsAt.Sub(acctReset)) > sharedResetTolerance
		out += renderScopedLimit(l, level, c, notice, warn, critical, d.Now, t, colourReset, showReset)
	}

	// Staleness marker: kept short (" · 2h old") because it rides on a line that
	// exists to be compact, and it only appears once the snapshot passes the age
	// at which Claude Code itself stops trusting it. Full level only — the
	// tighter levels have no room to spare.
	if level == LevelFull && !fetchedAt.IsZero() {
		if age := d.Now.Sub(fetchedAt); age > modelQuotaStale {
			out += " · " + formatAge(age) + " old"
		}
	}

	return out
}

// sharedResetTolerance is how far apart the account-wide and scoped weekly
// resets may sit and still count as the same rollover. Claude Code stamps them
// about a second apart, and they render identically well beyond that, so a
// minute is generous without ever merging two genuinely different windows.
const sharedResetTolerance = time.Minute

// accountWeeklyReset resolves the account-wide 7d reset from the same two
// sources QuotaProbe uses: the live payload first, then the persisted snapshot.
func accountWeeklyReset(d Data) (time.Time, bool) {
	var live []byte
	if d.Stdin.RateLimits != nil {
		live = d.Stdin.RateLimits.SevenDay.ResetsAt
	}
	var snapReset int64
	if snap, ok := quota.Freshest(); ok {
		snapReset = snap.SevenDayReset
	}
	return resolveReset(live, snapReset)
}

// absDuration returns the magnitude of d.
func absDuration(d time.Duration) time.Duration {
	if d < 0 {
		return -d
	}
	return d
}

// renderScopedLimit formats one model-scoped window at the given level.
//
// showReset drops the countdown when the window rolls over together with the
// account-wide 7d block rendered beside it.
func renderScopedLimit(l claudejson.ScopedLimit, level Level, c Config,
	notice, warn, critical float64, now time.Time, t renderer.Theme, colourReset string,
	showReset bool) string {

	// The reset countdown reuses the account-wide 7d colour thresholds; a scoped
	// window rolls over on the same weekly cadence.
	reset := ""
	if showReset {
		var resetUnix int64
		if !l.ResetsAt.IsZero() {
			resetUnix = l.ResetsAt.Unix()
		}
		reset = " " + formatReset(nil, resetUnix, now, sevenDayThresholds)
	}

	switch level {
	case LevelFull:
		// Same 10-segment bar the account-wide blocks use. A shorter bar would
		// save a few columns but read as a different kind of measurement sitting
		// on the same line; bars that mean the same thing should look the same.
		// The width is bought back by dropping the duplicated countdown instead.
		bar := quotaUsageColor(l.Percent, notice, warn, critical, t) +
			usageBar(l.Percent, c) + colourReset
		return fmt.Sprintf("%s 7d: %s%s", l.Model, bar, reset)
	case LevelCompact:
		bar := quotaUsageColor(l.Percent, notice, warn, critical, t) +
			compactBar(l.Percent, c) + colourReset
		return fmt.Sprintf("%s %s%s", l.Model, bar, reset)
	default: // LevelMinimal
		// Minimal keeps the model's first letter as the label — enough to tell it
		// apart from the account-wide percentages, which carry no letter at all.
		initial := string([]rune(l.Model)[0])
		pct := fmt.Sprintf("%d%%", displayPctInt(l.Percent, false))
		if !t.AnsiEnabled {
			return initial + " " + pct
		}
		return initial + " " + scopedPctColour(l.Percent, notice, warn, critical, pct)
	}
}

// formatAge renders a staleness age compactly: minutes below an hour, then
// whole hours, then days. The exact figure stops mattering once the snapshot is
// this old — the point is that it is no longer current.
func formatAge(age time.Duration) string {
	switch {
	case age < time.Hour:
		return fmt.Sprintf("%dm", int(age.Minutes()))
	case age < 24*time.Hour:
		return fmt.Sprintf("%dh", int(age.Hours()))
	default:
		return fmt.Sprintf("%dd", int(age.Hours()/24))
	}
}

// scopedPctColour wraps a percentage in its usage colour, mirroring the
// account-wide minimal-level rules (green · yellow · orange · red, bold_red
// above 95%).
func scopedPctColour(pct, notice, warn, critical float64, s string) string {
	var marker string
	switch {
	case pct > 95.0:
		marker = "{{color:bold_red}}"
	case pct >= critical*100.0:
		marker = "{{color:red}}"
	case pct >= warn*100.0:
		marker = "{{color:orange}}"
	case pct >= notice*100.0:
		marker = "{{color:yellow}}"
	default:
		marker = "{{color:green}}"
	}
	return marker + s + "{{reset}}"
}
