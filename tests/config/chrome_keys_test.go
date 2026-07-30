// Package config_test — Phase 7.5 vertical-footprint keys.
//
// Covers the five new [general] chrome toggles plus the widened table_rows
// range (0 is now a valid "no table" setting):
//
//	table_legend, table_frame, table_dividers, header_line0, header_line1
//
// The invariant under test throughout is that every key defaults to true, so a
// config file written before Phase 7.5 renders exactly as it did before, and
// that ToProbesConfig inverts them into the negative Hide* runtime flags.
package config_test

import (
	"path/filepath"
	"testing"

	"github.com/labzink/cc-probeline/internal/config"
)

// ─── defaults ────────────────────────────────────────────────────────────────

// T-CHR1: every chrome key defaults to true — an existing config that predates
// Phase 7.5 must keep the full bordered layout.
func TestChromeKeys_DefaultToTrue(t *testing.T) {
	d := config.Default()

	cases := []struct {
		name string
		got  bool
	}{
		{"TableLegend", d.General.TableLegend},
		{"TableFrame", d.General.TableFrame},
		{"TableDividers", d.General.TableDividers},
		{"HeaderLine0", d.General.HeaderLine0},
		{"HeaderLine1", d.General.HeaderLine1},
	}
	for _, c := range cases {
		if !c.got {
			t.Errorf("Default().General.%s = false, want true", c.name)
		}
	}
}

// T-CHR2: a partial config file that omits the chrome keys keeps them true.
// This is the regression guard for the loader's "start from Default()" pass:
// if that ever changed to a zero-value decode, every user would silently lose
// their table borders on upgrade.
func TestChromeKeys_OmittedKeysKeepDefaults(t *testing.T) {
	tmp := t.TempDir()
	p := seedConfig(t, tmp, "version = 1\n\n[general]\ntable_rows = 5\n")

	cfg := loadField(t, p)

	if cfg.General.TableRows != 5 {
		t.Errorf("TableRows: got %d, want 5", cfg.General.TableRows)
	}
	if !cfg.General.TableLegend || !cfg.General.TableFrame || !cfg.General.TableDividers {
		t.Errorf("omitted table_* keys must stay true, got legend=%v frame=%v dividers=%v",
			cfg.General.TableLegend, cfg.General.TableFrame, cfg.General.TableDividers)
	}
	if !cfg.General.HeaderLine0 || !cfg.General.HeaderLine1 {
		t.Errorf("omitted header_line* keys must stay true, got line0=%v line1=%v",
			cfg.General.HeaderLine0, cfg.General.HeaderLine1)
	}
}

// ─── setters ─────────────────────────────────────────────────────────────────

// T-CHR3: each boolean setter round-trips through the TOML file.
func TestChromeSetters_RoundTrip(t *testing.T) {
	cases := []struct {
		name string
		set  func(path string, v bool) error
		read func(c *config.Config) bool
	}{
		{"table_legend", config.SetTableLegend, func(c *config.Config) bool { return c.General.TableLegend }},
		{"table_frame", config.SetTableFrame, func(c *config.Config) bool { return c.General.TableFrame }},
		{"table_dividers", config.SetTableDividers, func(c *config.Config) bool { return c.General.TableDividers }},
		{"header_line0", func(p string, v bool) error { return config.SetHeaderLine(p, 0, v) },
			func(c *config.Config) bool { return c.General.HeaderLine0 }},
		{"header_line1", func(p string, v bool) error { return config.SetHeaderLine(p, 1, v) },
			func(c *config.Config) bool { return c.General.HeaderLine1 }},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			p := filepath.Join(t.TempDir(), "config.toml")

			if err := c.set(p, false); err != nil {
				t.Fatalf("set(%s, false): %v", c.name, err)
			}
			if got := c.read(loadField(t, p)); got {
				t.Errorf("%s after set false: got true, want false", c.name)
			}

			if err := c.set(p, true); err != nil {
				t.Fatalf("set(%s, true): %v", c.name, err)
			}
			if got := c.read(loadField(t, p)); !got {
				t.Errorf("%s after set true: got false, want true", c.name)
			}
		})
	}
}

// T-CHR4: SetHeaderLine rejects any line index other than 0 or 1 and leaves the
// file untouched.
func TestSetHeaderLine_RejectsUnknownIndex(t *testing.T) {
	p := filepath.Join(t.TempDir(), "config.toml")

	if err := config.SetHeaderLine(p, 2, false); err == nil {
		t.Fatal("SetHeaderLine(2) returned nil error, want a rejection")
	}
	if err := config.SetHeaderLine(p, -1, false); err == nil {
		t.Fatal("SetHeaderLine(-1) returned nil error, want a rejection")
	}
}

// T-CHR5: setting one chrome key must not disturb its neighbours — the setters
// round-trip the whole Config, so a bug here would silently reset other keys.
func TestChromeSetters_PreserveOtherKeys(t *testing.T) {
	tmp := t.TempDir()
	p := seedConfig(t, tmp, "version = 1\n\n[general]\ntable_rows = 7\ntable_frame = false\n")

	if err := config.SetTableLegend(p, false); err != nil {
		t.Fatalf("SetTableLegend: %v", err)
	}

	cfg := loadField(t, p)
	if cfg.General.TableRows != 7 {
		t.Errorf("TableRows clobbered: got %d, want 7", cfg.General.TableRows)
	}
	if cfg.General.TableFrame {
		t.Error("TableFrame clobbered: got true, want false")
	}
	if cfg.General.TableLegend {
		t.Error("TableLegend: got true, want false")
	}
	if !cfg.General.TableDividers {
		t.Error("TableDividers should still hold its default true")
	}
}

// ─── adapter ─────────────────────────────────────────────────────────────────

// T-CHR6: ToProbesConfig inverts the positive config keys into the negative
// runtime flags, and maps table_rows = 0 onto HideTable.
func TestToProbesConfig_InvertsChromeKeys(t *testing.T) {
	// All chrome on (the default) → nothing hidden.
	pc := config.ToProbesConfig(*config.Default())
	if pc.HideTable || pc.HideTableLegend || pc.HideTableFrame ||
		pc.HideTableDividers || pc.HideHeaderLine0 || pc.HideHeaderLine1 {
		t.Errorf("default config must hide nothing, got %+v", pc)
	}

	// All chrome off → everything hidden.
	cfg := *config.Default()
	cfg.General.TableRows = 0
	cfg.General.TableLegend = false
	cfg.General.TableFrame = false
	cfg.General.TableDividers = false
	cfg.General.HeaderLine0 = false
	cfg.General.HeaderLine1 = false

	pc = config.ToProbesConfig(cfg)
	if !pc.HideTable {
		t.Error("table_rows = 0 must set HideTable")
	}
	if !pc.HideTableLegend || !pc.HideTableFrame || !pc.HideTableDividers {
		t.Errorf("table_* = false must set the Hide* flags, got %+v", pc)
	}
	if !pc.HideHeaderLine0 || !pc.HideHeaderLine1 {
		t.Errorf("header_line* = false must set the Hide* flags, got %+v", pc)
	}
}

// T-CHR7: a non-zero table_rows never sets HideTable — only an explicit 0 does.
func TestToProbesConfig_NonZeroRowsKeepsTable(t *testing.T) {
	cfg := *config.Default()
	cfg.General.TableRows = 1

	if pc := config.ToProbesConfig(cfg); pc.HideTable {
		t.Error("table_rows = 1 must not set HideTable")
	}
}
