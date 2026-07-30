# Changelog

All notable changes to `cc-probeline` are documented here.

The format follows [Keep a Changelog](https://keepachangelog.com/en/1.1.0/), and the project follows [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

### Added

- **Per-model limits, not just the total** — the status line now shows the weekly window for each model that has its own, right beside the account-wide bars: `5h: █░░░░░░░░░ ↻ 0h:59m · 7d: ▒░░░░░░░░░ ↻ 6d.8h • Fable 7d: ▒░░░░░░░░░ ↻ 6d.8h`. Every bar is labelled, so the scoped figure can never be mistaken for the total. Claude Code never puts these windows in the status-line payload — only `five_hour` and `seven_day` — so the numbers come from the usage snapshot it caches in `~/.claude.json`, the same source its `/usage` screen reads. Rendering stays fully offline: the file is read-only, no credentials are touched, and nothing leaves the machine. Accounts without a model-scoped limit see no change. Toggle with `[widgets] quota_model`.

- **Take back your vertical space** — every row the status line draws can now be switched off from the config, and the keys stack. `table_legend` drops the `# role model cache r/w …` labels and their separator (2 rows), `table_frame` drops the `┌─┐ / └─┘` borders and the outer bars (2 rows), `header_line0` / `header_line1` drop the account and session headers, and `table_dividers` quietens the inner `│` separators without costing a row. The full eight-row dashboard can be taken down to a single line of quota bars. All keys default to `true`, so an existing config renders exactly as before.
- **`table_rows = 0`** — 0 is now a valid table size and means "no per-turn table at all": rows, legend and frame go together. Previously 0 was silently raised to 1.
- **CLI toggles for the new keys** — `cc-probeline table-legend on|off`, `table-frame`, `table-dividers`, and `header-line 0|1 on|off`, alongside the existing `table-rows`. All five show up in `cc-probeline check-config`.
- **`header_merge`** — puts both header rows on one line whenever they fit the terminal, and splits them again when they do not. A second row is only worth a terminal line when the content actually needs the width.
- **`bar_width`** — any width from 3 to 20 segments, default 10. It applies to the quota windows, the per-model windows and the context bar together, so the bars on a line always match each other. Widths 5 and 10 keep their historical renderers, which pre-round the value so the bar holds still as a percentage drifts; every other width uses a proportional renderer that does not round, so a window well below one segment's worth still shows as half instead of vanishing — at 8 segments a 9% window reads `▒░░░░░░░` rather than empty.
- **`bar_style`** — which glyphs the bars are drawn with. A terminal line has one height, so a visually lighter bar can only come from glyphs that take up less of the cell: `"line"` (`━ ╾ ─`) draws a rule through the middle, `"low"` (`▄ ▂ ▁`) sits on the baseline, `"dot"` (`● ◐ ○`) echoes the effort indicator, and `"none"` drops the bar entirely and prints the percentage in its place — `5h: 9% ↻ 6d.7h`. Default stays `"block"`. The style applies to every bar on the line at once, so a line never mixes two kinds.
- **`effort_style`** — `"word"` spells the reasoning effort out (`opus-5 xhigh`) instead of drawing the filled-circle icon (`opus-5 ◕`). The icon is compact but needs the legend to read; the word needs none. Default stays `"glyph"`.
- **`model_variant_tag`** — set false to drop the bracketed variant tag large-context models carry in their id (`opus-5[1m]` → `opus-5`). The context segment beside it already spells the window out, so the tag repeats what the line is about to say. Default stays true.
- **`scripts/build-frames.sh`** — regenerates every README screenshot from the code itself (emitter → ANSI → SVG → PNG), so the images always show what the current build renders instead of being screenshotted by hand. The code already referenced this script; it had never been committed. It also flattens the status line's faint attribute into explicit dark colours first, because the SVG renderers ignore SGR 2 and would otherwise paint history rows as brightly as the current request, losing the distinction the table is built around.

### Changed

- **The per-model window no longer repeats the weekly countdown.** A scoped weekly window rolls over with the account-wide one — Claude Code stamps the two about a second apart — so `Fable 7d: ▒░░░░░░░░░ ↻ 6d.8h` was spending a dozen columns restating the `7d` block beside it. The countdown is now shown only when the two genuinely diverge.
- **The context segment is labelled like every other one.** It rendered as `ctx ███░░░` while the quota segments used `5h:` / `7d:`; it is now `ctx:`, and it follows `bar_width` so its bar always matches the bars next to it.

### Fixed

- **Config edits no longer destroy your comments.** Only `tutorial_hints` edited its value in place; every other setter — `mode`, `no_color`, `widgets`, `table_rows`, `price_check` and the rest — round-tripped the file through the config struct, which rewrote it wholesale: comments gone, key order normalised, alignment lost. Changing one boolean should not cost you an annotated file. All setters now share the same surgical edit, so comments, blank lines, key order and unknown sections survive byte-for-byte, and a broken config is reported rather than overwritten. (The changelog has promised this since 0.1.1; it was only ever true for one key.)
- **`tutorial_hints = false` now actually works** — the key, its `cc-probeline hints off` command and its `check-config` entry all existed, but nothing read the value at render time, so the rotating tips kept appearing. Switching them off now stops the rotation; genuine alerts (config errors, expired cache, a pending update) still surface, since those report something you need to act on.
- **No more blank status-line rows** — a header line whose every segment was switched off used to be emitted as an empty line, costing a terminal row while showing nothing. It is now dropped, so the `[widgets]` toggles reclaim the row as well.

## [0.1.3] — 2026-06-17

### Changed

- **One-step install on every channel** — Homebrew and Scoop now wire the Claude Code status line on install, the same way the curl script already does. Install through any channel, restart Claude Code, and you're done — no separate wiring command. An existing custom status line is never overwritten (switch explicitly with `cc-probeline install --merge-settings --force`).
- **Uninstall restores your status line everywhere** — `brew uninstall` now restores the status line you had before removing the binary, and the curl script gained `--uninstall` (`install.sh … | sh -s -- --uninstall`) to do the same. On Scoop, run `cc-probeline uninstall` before `scoop uninstall` (Scoop has no uninstall hook).

## [0.1.2] — 2026-06-17

### Fixed

- **macOS `brew install`** — the Homebrew cask now strips the `com.apple.quarantine` attribute on install, so Gatekeeper no longer blocks the unsigned binary ("cannot verify the developer" / `zsh: killed`). `brew install` works out of the box on macOS again.

## [0.1.1] — 2026-06-17

### Added

- **Self-healing prices + update notice** — cc-probeline refreshes its price table over the network (one optional check a day, opt-out, never during render) so cost estimates track Anthropic's rates without a reinstall; offline or opted out, it uses the table baked in at build time. When a newer release is out, the status line shows an `↑ update: vX → vY — run /cc-probeline-update` hint. Disable the check with `price_check = false` or via the `/cc-probeline-config` wizard.
- **Update from the plugin** — a `/cc-probeline-update` command upgrades the binary through the channel it was installed with (Homebrew / Scoop / curl), or installs it if missing.
- **Install from the plugin** — the marketplace plugin ships a `/cc-probeline-install` command that detects your OS, installs the binary through the right channel (Homebrew / Scoop / curl) and wires the status line, asking before it runs anything. A session-start check offers it automatically when the binary is missing.
- **Build provenance** — release archives are signed with keyless build provenance attestation, verifiable with `gh attestation verify <file> --repo labzink/cc-probeline`.

### Changed

- **Config edits preserve comments** — toggling `tutorial_hints` (and the `/cc-probeline-config` wizard) now edits the value in place, keeping comments, formatting, and key order in your `config.toml` intact.

### Fixed

- **curl install path** — the documented `curl … | sh` one-liner now points at `scripts/install.sh` (the published location).

## [0.1.0] — 2026-06-16

First public release.

### Added

- **Status line core** — reads the active session JSONL from `~/.claude/projects/<slug>/*.jsonl` and renders a single status line: current model, token usage (input / output / cache_read / cache_create), and approximate session cost.
- **Per-turn table** — compact breakdown of cost and tokens per turn, with a configurable number of rows.
- **Context window indicator** — shows how much of the model context window the session is using.
- **Quota warnings** — block-limit and weekly rate-limit indicators derived from the official Claude Code source data (delta only, no bundled pricing tables).
- **Active subagent indicator** — surfaces subagents running in the session.
- **Probes** — model, cost, project, email, time, context, quota, and git, each individually toggleable.
- **Semantic colour** — 16-colour ANSI palette readable on both light and dark terminals, with `NO_COLOR` support and a monochrome mode.
- **Configuration** — TOML config file plus a `cc-probeline config` CLI and the `/cc-probeline-config` interactive wizard (also shipped as a plugin command).
- **Distribution** — Homebrew tap, Scoop bucket, and `install.sh` / `install.ps1` installers; GitHub Releases with SHA256 checksums for five targets (darwin arm64/amd64, linux arm64/amd64, windows amd64).
- **Plugin marketplace** — `.claude-plugin/marketplace.json` + `plugin.json` so the plugin is installable via `/plugin marketplace add labzink/cc-probeline` and `/plugin install cc-probeline@cc-probeline`.

### Notes

- The main status line is wired into Claude Code by `install.sh` / Homebrew / Scoop (or manually in `settings.json`). Claude Code plugins cannot set the main `statusLine` automatically, so the marketplace plugin provides discovery and the `/cc-probeline-config` wizard rather than auto-installing the status line.
