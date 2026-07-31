package config

// Config is the top-level configuration structure for cc-probeline.
// It is unmarshalled from a TOML file found via the cascade (see LoadCascade).
// Use Default() to obtain the built-in defaults when no file exists.
type Config struct {
	// Version is the config file format version. Currently always 1.
	// Future breaking changes will increment this number so the loader can
	// apply migration logic before unmarshalling into this struct.
	Version int `toml:"version" json:"version"`

	// General groups settings that affect the overall behaviour of the tool:
	// colours, font glyphs, and hints. Change these to tune the visual style
	// without touching individual widget settings.
	General General `toml:"general" json:"general"`

	// Widgets controls which probes are rendered. Set a field to false to
	// permanently hide the corresponding status-line widget. All widgets are
	// visible by default to preserve Phase 4-5 behaviour.
	Widgets Widgets `toml:"widgets" json:"widgets"`

	// Thresholds defines numeric cutoffs used by probes to emit warnings.
	// CostBudgetUSD=0 disables budget warnings. Ratios are in the [0,1] range.
	Thresholds Thresholds `toml:"thresholds" json:"thresholds"`

	// Colors overrides individual palette roles. Omitted roles keep the
	// built-in colour; an unusable value is skipped rather than applied.
	Colors Colors `toml:"colors" json:"colors"`

	// Probes groups per-probe configuration that is not covered by the widget
	// toggles above. Currently only the Email probe requires extra settings.
	Probes Probes `toml:"probes" json:"probes"`
}

// General groups top-level display settings that are not widget-specific.
type General struct {
	// TutorialHints enables inline hints shown in the status line when the
	// session is fresh or when notable events occur. Set to false to suppress.
	TutorialHints bool `toml:"tutorial_hints" json:"tutorial_hints"`

	// NoColor forces plain-text output with no ANSI colour codes. Equivalent
	// to setting NO_COLOR=1 in the environment. The environment variable takes
	// precedence over this field.
	NoColor bool `toml:"no_color" json:"no_color"`

	// NerdFont enables Nerd Font glyph icons in widget output. Set to true
	// when your terminal uses a patched Nerd Font. The terminal auto-detection
	// at startup may also set this automatically.
	NerdFont bool `toml:"nerd_font" json:"nerd_font"`

	// RefreshIntervalHint is the suggested refresh cadence in seconds, passed
	// to Claude Code via the hook handshake. Does not affect the actual
	// rendering interval — CC controls that. Range: 1-60.
	RefreshIntervalHint int `toml:"refresh_interval_hint" json:"refresh_interval_hint"`

	// TableRows is the maximum number of per-turn rows shown in the subagent
	// table. Defaults to 10. SetTableRows clamps the value to [0, 40].
	// 0 is a valid setting and means "no table at all": the per-turn block
	// (data rows, legend and frame alike) is dropped from the status line.
	TableRows int `toml:"table_rows" json:"table_rows"`

	// TableLegend shows the legend row ("# role model cache r/w out ~cost
	// tool") under the per-turn table, together with its ├─┼─┤ separator.
	// Default true. Set false to reclaim two status-line rows once the column
	// order is familiar.
	TableLegend bool `toml:"table_legend" json:"table_legend"`

	// TableFrame draws the table frame: the ┌─┬─┐ / └─┴─┘ border lines and the
	// outer left/right bars on every row. Default true. Set false to reclaim
	// two more status-line rows; the columns stay aligned without it.
	TableFrame bool `toml:"table_frame" json:"table_frame"`

	// TableDividers draws the inner vertical column separators (│) and the
	// group-boundary notch glyphs (├ ┼ ┤) that mark where one orchestrator
	// request ends and the next begins. Default true. Set false for a
	// whitespace-separated table; costs no rows, only visual noise.
	TableDividers bool `toml:"table_dividers" json:"table_dividers"`

	// HeaderLine0 shows the first status-line row — the account/workspace
	// header carrying email, project and the 5h/7d quota bars. Default true.
	// Set false to drop the whole row regardless of the individual [widgets]
	// toggles.
	HeaderLine0 bool `toml:"header_line0" json:"header_line0"`

	// ModelVariantTag keeps the bracketed variant tag that large-context models
	// carry in their id ("opus-5[1m]"). Default true. Set false to drop it: the
	// context segment beside it already spells the window out ("437K/1000K"),
	// so the tag repeats information the line is about to give anyway.
	ModelVariantTag bool `toml:"model_variant_tag" json:"model_variant_tag"`

	// Alerts shows the notice row under the status line: cache rebuilt, subagent
	// cache expired, context compacted, config errors, a pending update. Default
	// true. Set false to drop the row entirely.
	//
	// It is not decoration — "Cache rebuilt · 60-min idle TTL passed" is telling
	// you that an idle gap past the prompt-cache TTL made the next turn rewrite
	// the whole cache, which is billed at the cache-write rate and is usually the
	// reason a turn suddenly cost more. Turning it off trades that explanation
	// for the row it occupies.
	Alerts bool `toml:"alerts" json:"alerts"`

	// StaleMarker prefixes a per-model figure with "~" once Claude Code's cached
	// usage snapshot passes the age at which Claude Code itself stops trusting
	// it (one hour). Default true. Set false to always read the number as-is —
	// the figure is a weekly window, so an hour of drift is usually small.
	StaleMarker bool `toml:"stale_marker" json:"stale_marker"`

	// BarStyle picks the glyphs the Full-level progress bars are drawn with. A
	// terminal line has one height, so a visually shorter bar can only come
	// from glyphs that occupy less of it:
	//
	//	"block" — █ ▒ ░, the original, filling the cell (default)
	//	"line"  — ━ ╾ ─, a rule through the middle of the line
	//	"low"   — ▄ ▂ ▁, sitting on the baseline
	//	"dot"   — ● ◐ ○, echoing the effort indicator
	//	"none"  — no bar; the percentage stands in for it
	//
	// Any other value falls back to "block".
	BarStyle string `toml:"bar_style" json:"bar_style"`

	// EffortStyle selects how the reasoning-effort indicator is drawn next to
	// the model name: "glyph" (default) draws the filled-circle icon — ○ low,
	// ◔ medium, ◑ high, ◕ xhigh, ● max — and "word" spells the level out. The
	// icon is compact but needs a legend to read; the word needs none.
	// Any other value falls back to "glyph".
	EffortStyle string `toml:"effort_style" json:"effort_style"`

	// BarWidth is how many segments the Full-level progress bars use — the
	// quota windows, the per-model windows and the context bar alike, so they
	// always match each other. Accepts 10 (default, 5% precision) or 5 (half
	// the width, 10% precision); any other value falls back to 10.
	BarWidth int `toml:"bar_width" json:"bar_width"`

	// HeaderMerge joins the two header rows into one whenever their combined
	// width still fits the terminal, falling back to two rows when it does not.
	// Default false, which keeps the original two-row header. Set true to stop
	// spending a second terminal row on content that already fits beside the
	// first.
	HeaderMerge bool `toml:"header_merge" json:"header_merge"`

	// HeaderLine1 shows the second status-line row — the session header
	// carrying model, git, context, cost and elapsed time. Default true.
	// Set false to drop the whole row regardless of the individual [widgets]
	// toggles.
	HeaderLine1 bool `toml:"header_line1" json:"header_line1"`

	// Mode selects the display mode: "standard" (default) or "super-compact".
	// CORE reads this field to switch the assembler layout. Setters write it
	// via SetMode; the legacy per-session mode file is superseded by this field.
	Mode string `toml:"mode" json:"mode"`

	// PriceCheck enables the once-per-day network price/version check (Phase 7.46
	// Wave B / BL-36): a single GET of our public prices.json, 24h-cached and
	// fail-soft offline, used to self-heal the cost table and surface the latest
	// version. Default true. Set false to keep the tool fully offline — the baked
	// price table (correct as of build time) is then used and no network is
	// touched. The render path itself is always network-free; this governs only
	// the background refresh.
	PriceCheck bool `toml:"price_check" json:"price_check"`
}

// Widgets controls visibility for each status-line probe widget.
// All fields default to true (all widgets visible).
type Widgets struct {
	// Model shows the active Claude model name (e.g. "claude-sonnet-4-5").
	Model bool `toml:"model" json:"model"`

	// Effort shows the effort level indicator ([high], [normal], etc.).
	Effort bool `toml:"effort" json:"effort"`

	// Cost shows the running session cost estimate in USD.
	Cost bool `toml:"cost" json:"cost"`

	// Project shows the project/working-directory name.
	Project bool `toml:"project" json:"project"`

	// Email shows the user email address from the CC session.
	Email bool `toml:"email" json:"email"`

	// Time shows the elapsed session time.
	Time bool `toml:"time" json:"time"`

	// Ctx shows the context window usage as a progress bar.
	Ctx bool `toml:"ctx" json:"ctx"`

	// Quota shows the daily/monthly quota usage if available.
	Quota bool `toml:"quota" json:"quota"`

	// QuotaModel shows the per-model weekly windows (e.g. "Fable 7d: ▒░░░░ ↻ 6d")
	// next to the account-wide bars. The figures come from the usage snapshot
	// Claude Code caches in ~/.claude.json, since the status-line payload carries
	// only account-wide limits. Renders nothing when the account has no
	// model-scoped limit. Default true.
	QuotaModel bool `toml:"quota_model" json:"quota_model"`

	// Git shows the current git branch and dirty-state indicator.
	Git bool `toml:"git" json:"git"`
}

// Thresholds defines numeric cutoffs used by probes to decide when to emit
// warnings or change colour. All values are optional overrides.
type Thresholds struct {
	// CostBudgetUSD is the per-session cost budget in USD. When the running
	// cost exceeds this value the cost probe turns red. 0 disables the check.
	CostBudgetUSD float64 `toml:"cost_budget_usd" json:"cost_budget_usd"`

	// CtxNoticeRatio is the context-window fill ratio at which the Ctx probe
	// first turns yellow (the lower of two warning levels). Range: (0, 1).
	// Default: 0.50. Must satisfy notice < warn < critical.
	CtxNoticeRatio float64 `toml:"ctx_notice_ratio" json:"ctx_notice_ratio"`

	// CtxWarnRatio is the context-window fill ratio at which the Ctx probe
	// switches to orange (the upper warning level). Range: (0, 1). Default: 0.70.
	// Must satisfy notice < warn < critical.
	CtxWarnRatio float64 `toml:"ctx_warn_ratio" json:"ctx_warn_ratio"`

	// CtxCriticalRatio is the fill ratio at which the Ctx probe turns red.
	// Range: (0, 1). Default: 0.90. Must satisfy notice < warn < critical.
	CtxCriticalRatio float64 `toml:"ctx_critical_ratio" json:"ctx_critical_ratio"`

	// Quota5h{Notice,Warn,Critical}Ratio are the three colour-flip ratios for the
	// 5-hour rate-limit window (yellow/orange/red). Range: (0, 1).
	// Defaults: 0.50/0.70/0.90. Must satisfy notice < warn < critical.
	Quota5hNoticeRatio   float64 `toml:"quota_5h_notice_ratio" json:"quota_5h_notice_ratio"`
	Quota5hWarnRatio     float64 `toml:"quota_5h_warn_ratio" json:"quota_5h_warn_ratio"`
	Quota5hCriticalRatio float64 `toml:"quota_5h_critical_ratio" json:"quota_5h_critical_ratio"`

	// Quota7d{Notice,Warn,Critical}Ratio are the three colour-flip ratios for the
	// 7-day rate-limit window (yellow/orange/red). Range: (0, 1).
	// Defaults: 0.50/0.70/0.90. Must satisfy notice < warn < critical.
	Quota7dNoticeRatio   float64 `toml:"quota_7d_notice_ratio" json:"quota_7d_notice_ratio"`
	Quota7dWarnRatio     float64 `toml:"quota_7d_warn_ratio" json:"quota_7d_warn_ratio"`
	Quota7dCriticalRatio float64 `toml:"quota_7d_critical_ratio" json:"quota_7d_critical_ratio"`

	// OrchTTLMinutes is the orchestrator idle timeout in minutes. The
	// subagent probe emits a warning when the orchestrator has been idle
	// longer than this value. Default: 60.
	OrchTTLMinutes int `toml:"orch_ttl_minutes" json:"orch_ttl_minutes"`

	// SubagentGapMinutes is the expected maximum gap between subagent
	// heartbeats in minutes. A larger gap triggers a stale-agent warning.
	// Default: 5.
	SubagentGapMinutes int `toml:"subagent_gap_minutes" json:"subagent_gap_minutes"`
}

// Probes groups per-probe configuration values that are not widget toggles.
type Probes struct {
	// Email holds configuration specific to the Email probe.
	Email EmailOpts `toml:"email" json:"email"`
}

// EmailOpts holds configuration for the Email probe.
type EmailOpts struct {
	// Address overrides the email address shown by the Email probe.
	// When empty the probe reads the address from the CC session JSONL.
	Address string `toml:"address" json:"address"`
}
