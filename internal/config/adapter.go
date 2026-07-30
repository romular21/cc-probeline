package config

import (
	"github.com/labzink/cc-probeline/internal/probes"
)

// ToProbesConfig maps the high-level Config into the per-probe runtime config
// consumed by Probe.Visible / Probe.Render. Pure function; no errors.
func ToProbesConfig(cfg Config) probes.Config {
	return probes.Config{
		ModelEnabled:   cfg.Widgets.Model,
		EffortEnabled:  cfg.Widgets.Effort,
		CostEnabled:    cfg.Widgets.Cost,
		ProjectEnabled: cfg.Widgets.Project,
		EmailEnabled:   cfg.Widgets.Email,
		TimeEnabled:    cfg.Widgets.Time,
		CtxEnabled:     cfg.Widgets.Ctx,
		// CacheEnabled and SubagentEnabled are hardcoded true: the widget toggle
		// fields were removed from Widgets (dead config) but probes still read
		// these flags internally (cache.go:46, subagent.go:34). Hardcoding true
		// keeps probe visibility unchanged without cascading deletes into probes/.
		CacheEnabled:    true,
		QuotaEnabled:      cfg.Widgets.Quota,
		GitEnabled:        cfg.Widgets.Git,
		SubagentEnabled:   true,
		ModelQuotaEnabled: cfg.Widgets.QuotaModel,

		TableRows: cfg.General.TableRows,

		// Positive config keys → negative runtime flags, so that a zero-value
		// probes.Config still renders the original full-chrome layout.
		// table_rows = 0 is the explicit "no table" setting; the loader starts
		// from Default() (10), so an omitted key can never reach this branch.
		HideTable:         cfg.General.TableRows == 0,
		HideTableLegend:   !cfg.General.TableLegend,
		HideTableFrame:    !cfg.General.TableFrame,
		HideTableDividers: !cfg.General.TableDividers,
		HideHeaderLine0:   !cfg.General.HeaderLine0,
		HideHeaderLine1:   !cfg.General.HeaderLine1,
		HideHints:         !cfg.General.TutorialHints,

		Email: cfg.Probes.Email.Address,

		CostBudgetUSD:        cfg.Thresholds.CostBudgetUSD,
		CtxNoticeRatio:       cfg.Thresholds.CtxNoticeRatio,
		CtxWarnRatio:         cfg.Thresholds.CtxWarnRatio,
		CtxCriticalRatio:     cfg.Thresholds.CtxCriticalRatio,
		Quota5hNoticeRatio:   cfg.Thresholds.Quota5hNoticeRatio,
		Quota5hWarnRatio:     cfg.Thresholds.Quota5hWarnRatio,
		Quota5hCriticalRatio: cfg.Thresholds.Quota5hCriticalRatio,
		Quota7dNoticeRatio:   cfg.Thresholds.Quota7dNoticeRatio,
		Quota7dWarnRatio:     cfg.Thresholds.Quota7dWarnRatio,
		Quota7dCriticalRatio: cfg.Thresholds.Quota7dCriticalRatio,
		OrchTTLMinutes:       cfg.Thresholds.OrchTTLMinutes,
		SubagentGapMinutes:   cfg.Thresholds.SubagentGapMinutes,
	}
}
