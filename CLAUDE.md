# cc-probeline

A Claude Code status line: a single Go binary that reads the session's JSONL log
and the status-line payload, and renders one to eight terminal rows.

This fork adds a configurable vertical footprint and per-model quota windows on
top of [labzink/cc-probeline](https://github.com/labzink/cc-probeline).

## Build and test

Go lives in `~/.local/go` and is **not** on PATH. Shell state does not persist
between commands, so export it every time:

```sh
export PATH=$HOME/.local/go/bin:$PATH
go build ./... && go test ./... && go vet ./...
```

`go test ./...` should report 22 passing packages and no failures. Tests are
hermetic — they touch no real config, no `~/.claude`, and no network.

To try a change against a real session without installing:

```sh
printf '%s' "$PAYLOAD" | COLUMNS=200 CC_PROBELINE_CONFIG=/tmp/try.toml NO_COLOR=1 ./cc-probeline
```

`CC_PROBELINE_CONFIG` wins over the project-local and global config, so it is
the safe way to preview a layout without touching the user's own file.

## The rule that governs most decisions

**Defaults never change.** Every key added here defaults to the behaviour the
plugin already shipped, so an existing config renders exactly as it did before.
Someone who installs this fork over the original should see no difference until
they ask for one. Two consequences show up constantly in the code:

- **Runtime flags are stated negatively** — `HideTableLegend`, `HideHeaderLine0`,
  `HideTable` — so the zero value of `probes.Config` reproduces the original
  layout. Callers that build a Config directly (tests, older code paths) then
  need to set nothing. `ToProbesConfig` inverts the positive `[general]` keys
  into these fields. `MergeHeaderLines` and `BarStyle` are the exceptions:
  their zero values already mean "as before".
- **A new segment renders nothing when it has no data.** The per-model quota
  probe is on by default but invisible on accounts without a scoped limit.

## Architecture notes worth knowing before editing

**Probes are pure functions of `probes.Data`.** `main` gathers everything —
JSONL, payload, state, the cached usage snapshot — and probes only render. When
the per-model probe read `~/.claude.json` directly it dragged process-global
cache state into rendering and made the golden tests machine-dependent. Data in,
string out.

**Config edits are line-level, never a round-trip.** Every setter goes through
`setScalar` in `internal/config/setinplace.go`, which finds the assignment and
replaces what follows `=`. Unmarshalling into `Config` and marshalling back
rewrites the whole file: comments gone, key order normalised. Users are invited
to read and annotate `config.toml`, so that is not an acceptable cost for
changing one boolean. A file that does not parse is reported, never overwritten.

**Colour goes through the palette, not through literals.** `renderer.ColorScheme`
holds one entry per semantic role, and `[colors]` lets any role be repointed.
`sage` and `amber` exist separately from `green` and `yellow` because a value
rendered as text needs a calmer hue than the same value rendered as a bar —
dimming was tried instead and made the number too faint to read.

**Two data sources, and they are not interchangeable.** The account-wide 5h/7d
figures arrive in the status-line payload on every render and are always fresh.
Per-model weekly windows never reach the payload at all; they come from
`cachedUsageUtilization` in `~/.claude.json`, which Claude Code refreshes from
live API responses at most once per five minutes and discards past an hour.
Staleness thresholds here mirror those numbers rather than being chosen by
taste. Reading that file is read-only and touches no credentials — keep it that
way, the privacy claim in README and PRIVACY.md depends on it.

## Screenshots

`assets/screenshots/*.png` are rendered from the code by
`scripts/build-frames.sh`, not captured by hand. Any change to default
rendering — a label, a glyph, a default value — leaves the committed images a
step behind, so rerun the script and commit the result.

Two traps it already works around: the SVG renderers ignore ANSI faint (SGR 2),
so history rows would come out as bright as the current request; and line 0 is
long enough that a narrow frame width silently collapses the header to its
Compact form, dropping the `5h:` / `7d:` labels.

## Tests

Tests live in `tests/`, mirroring the package layout, and are written to state
what the behaviour is *for* rather than to restate the implementation. Two
kinds need care:

- **Golden snapshots** (`tests/statusline/testdata/golden/`) — regenerate with
  `go test ./tests/statusline/ -run Golden -update`, then read the diff. A
  golden that changed for a reason you cannot name means the change was wider
  than intended.
- **Layout tests** assert on rendered line counts, because the whole point of
  the footprint keys is how many terminal rows the line occupies.

## Adding a config key

The path is the same every time, and all of it is required — a key that renders
but does not appear in `check-config` or the wizard is only half-added:

1. `internal/config/config.go` — the field, with a doc comment saying what the
   default is and why.
2. `internal/config/load.go` — the default in `Default()`.
3. `internal/probes/probe.go` — the runtime field, negatively stated if the
   zero value would otherwise change the layout.
4. `internal/config/adapter.go` — the mapping, inverting if needed.
5. `internal/config/config_set.go` — a setter, validating before it writes.
6. `cmd/cc-probeline/main.go` + `config_set.go` — the CLI subcommand.
7. `cmd/cc-probeline/checkconfig.go` — the `check-config` entry.
8. `commands/cc-probeline-config.md` — the wizard. Note its widget caps at four
   questions of four options, which is why it is split across two pages.
9. `scripts/config.toml.example`, `README.md`, `CHANGELOG.md`.
