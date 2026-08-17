[![License: MIT](https://img.shields.io/github/license/romular21/cc-probeline)](LICENSE)
![Platforms](https://img.shields.io/badge/platforms-macOS%20·%20Linux%20·%20Windows-555)
![Fork](https://img.shields.io/badge/fork%20of-labzink%2Fcc--probeline-555)

# See where it leaks. Stop paying for it.

A live dashboard right in your status line — surfacing what Claude Code hides: the cost of every turn, what your subagents spend, how long your cache stays alive, plus current 5h/7d and Fable 7d limits, context and git.

And it takes exactly as much room as you let it: every row is configurable, down to a single line.

**Stop overpaying for inefficiency you can't see. Spend your limits on purpose.**

**[Why this fork exists →](#why-this-fork-exists)** · **[Install →](#install)**

![cc-probeline live dashboard: a Claude Code session where every turn lands priced, subagents bill in real time, the cache TTL ages ⏱ 60m → 0m and rebuilds in dollars, and the 5h limit fills to 100% with overage — all in the status line](assets/video/hero.gif)

## Why this fork exists

The original renders a fixed eight-row dashboard. For seeing everything at once
that is the right default, and its author's position is that anyone who wants it
smaller is free to adapt the code. This fork is that adaptation — kept in a shape
other people can use, so the answer to "make it shorter" is a config key rather
than a private patch.

Two things changed, and one thing deliberately did not.

**Every row became optional.** The legend, the borders, the per-turn table and
either header line each switch off from the config, and they stack. The eight
rows go down to one:

```
5h: 1% ↻ 4h:47m · 7d: 9% ↻ 6d.7h • Fable 7d: 5% • opus-5 xhigh • ctx: 30% 300K/1000K
```

**Per-model limits arrived.** Claude Code tracks a separate weekly window for
models that have one, but never puts it in the status-line payload — you only
see it by opening `/usage`. This fork reads it from the snapshot Claude Code
caches locally and puts it on the line next to the account-wide bars.

**Defaults did not move.** Every key added here defaults to the behaviour the
original shipped. Install this over the original and nothing looks different
until you ask for it.

| | original | this fork |
|---|---|---|
| Rows | fixed 8 | 1–8, every row a key |
| `table_rows = 0` | raised to 1 | valid: no table at all |
| Per-model limits | — | `Fable 7d: 5%` |
| Bars | 5 or 10 blocks | 3–20 segments, five glyph styles, or a bare percentage |
| Palette | fixed | `[colors]`, any role, hex or SGR |
| Effort | `◕` | `◕` or `xhigh` |
| Header rows | always two | merged into one when they fit |
| `tutorial_hints` | key existed, was never read | works |
| Config edits | comments stripped on every write | preserved |

The full list, including four bugs found along the way, is in
[CHANGELOG.md](CHANGELOG.md).

## What the probe pulls out

Most status lines count things — tokens, turns, running agents. **The probe prices them.** Everything below comes out of your session's local log: data Claude Code has, but never shows you.

- **Every turn, priced** — not one opaque session total: a live table where each step lands with its own cost.
- **What your subagents spend** — subagent work is invisible while it runs. The probe puts each agent on the bill, live, next to your own turns.
- **Cache rebuilds, in dollars** — idle past the TTL (60 min for the orchestrator, 5 for subagents), and your next turn quietly rewrites the whole cache. The probe ages it live (⏱ 60m → 0m) and prices the rebuild when it hits.
- **Extra usage in money, not percent** — past 100% of your plan, the overage shows up in dollars before the invoice does.
- **Prices that stay correct** — your dollars are only as honest as the price table behind them. cc-probeline refreshes its rates over the network — one optional, opt-out check a day, never during render — so when Anthropic changes prices your totals follow within a day, no reinstall. Offline or opted out, it falls back to the table baked into the build.
- **5h / 7d limits with reset clocks** — watch them fill, know exactly when they free up.
- **Per-model limits, not just the total** — Claude Code shows a separate weekly window for models that have one, but never puts it in the status-line payload: you only see it by opening `/usage`. cc-probeline reads it from the snapshot Claude Code caches locally and puts it on the line next to the account-wide bars — `Fable 7d: ▒░░░░░░░░░ ↻ 6d.8h` — every bar labelled, so the scoped figure is never mistaken for the total.

  This one segment is the only figure on the line that can go stale, and it is worth knowing why. The `5h` and `7d` bars come from the status-line payload, which Claude Code sends on every render, so they are always current — and if a window rolls over while a stored reading is in hand, the reading is discarded rather than shown. The per-model window has no such live source: it comes from a cache Claude Code refreshes only when certain API responses land, which in a long session can be over an hour apart. When that cache passes the age Claude Code itself stops trusting it, the figure is marked `~5%`; opening `/usage` refreshes it on the spot.
- **Fits the room you give it** — the full dashboard is eight rows; the legend, the borders, the per-turn table and either header line each switch off from the config, and they stack. Take it down to one line of quota bars, or leave it exactly as it ships. Every key defaults to the full layout.
- **Colour-coded zones** — numbers shift colour as they enter warning and critical territory, so the line catches your eye exactly when it should.
- Plus the table stakes: model, context, git, session time.

![Turn-by-turn cost table: orchestrator and subagent rows side by side, cache read/write per turn, per-turn dollars, config hint at the bottom](assets/screenshots/02.png)
**Every turn lands on its own line — orchestrator and subagents alike — priced as it happens. Finally you see where every dollar of your reasoning actually goes.**

**Built to fit your terminal.** Don't like a segment, the colours, the width — or how many rows it eats? Every one of those is a config key, and `cc-probeline <key> <value>` writes it for you — no hand-editing TOML. The `/cc-probeline-config` wizard does the same through a widget; it ships with the plugin, which this fork does not publish, so use it only if you have the original's plugin installed alongside.

![Status line past the plan limit: +$3.80 extra usage shown in red next to a filled 5h bar](assets/screenshots/03.png)
**The moment you cross 100%, you'll see it — and the extra bill stays under your control.**

![Quota warning: 5h window at 98% with its reset clock, plus a subagent cache-expired alert](assets/screenshots/04.png)
**You get warned while there's still time to act — not after you've hit the wall.**

![Cache rebuild caught live: 240K tokens rewritten for $3.02, TTL countdown showing fresh 60m next to stale 0m](assets/screenshots/05.png)
**Cache rebuilds stop being silent — you see the price the moment they happen.**

And nothing about your session ever leaves your machine — that's [why it's called a probe](#why-its-called-a-probe).

## Why it's called a probe

A probe is an instrument of observation, not intervention. Everything cc-probeline does is read and display — it never reaches into your account or reports on you.

- **What it reads:** your session's JSONL log (`~/.claude/projects/…`) and the status-line payload Claude Code pipes directly to it.
- **What it doesn't touch:** credentials, keychain, OAuth tokens — no telemetry, ever. Rendering is fully offline; the only network it ever makes is one optional, opt-out price/version check a day — a plain download of a public file, sending nothing about your session. Turn it off and it never touches the network at all.
- **The binary:** single compiled Go binary, no runtime dependencies, one run ≈ 5 ms.
- **Auditable:** MIT license, open source. This fork is built from source rather than distributed as a release, so there is nothing to verify a signature against — you compile the code you can read. (The upstream project publishes signed, checksummed release archives.)

## Install

This fork publishes no release archives, so the packaged channels — Homebrew,
Scoop, the curl one-liner, the plugin marketplace — all still point at the
**original** and would install that instead. Build from source to get this
version:

```sh
git clone https://github.com/romular21/cc-probeline
cd cc-probeline
go build -o ~/.local/bin/cc-probeline ./cmd/cc-probeline
cc-probeline install          # wires the Claude Code status line
```

Requires Go 1.22 or newer. `cc-probeline install` writes the `statusLine` block
into `~/.claude/settings.json`; an existing custom status line is left untouched
unless you pass `--merge-settings --force`. **Restart Claude Code** afterwards.

**Verify:**

```sh
cc-probeline --check
```

Prints `Installation OK`.

**Updating** is the same loop: `git pull && go build -o ~/.local/bin/cc-probeline ./cmd/cc-probeline`.
The `/cc-probeline-update` command shipped by the plugin upgrades through the
packaged channels and would pull the original over your build, so do not use it
here.

### Installing the original instead

If you want the upstream version — packaged, signed, with release archives —
follow [its own README](https://github.com/labzink/cc-probeline#install). In
short:

```sh
curl -fsSL https://raw.githubusercontent.com/labzink/cc-probeline/main/scripts/install.sh | sh
```

### Requirements

- Claude Code on macOS, Linux, or Windows.
- For the quota segment (5h / 7d limits, extra usage): Claude Code ≥ 2.1.80, which passes `rate_limits` in the status-line payload. On older versions the quota segment is hidden; everything else works normally.
- For the per-model segment (`Fable 7d: …`): a plan that actually has a model-scoped weekly limit. These windows never reach the status-line payload, so the figures are read from the usage snapshot Claude Code caches in `~/.claude.json` — locally, read-only, no credentials and no network. No scoped limit on the account means the segment simply does not render.

  That snapshot is refreshed by Claude Code, not by this tool, and only when certain API responses land — a session can run for an hour and a half of heavy traffic without one. Past the age at which Claude Code itself stops trusting it, the figure is marked with a leading `~`. **Opening `/usage` refreshes it immediately** (measured: a 98-minute-old snapshot updated the moment the command ran), and the `~` clears on the next render.

  The account-wide `5h` and `7d` bars are unaffected — those arrive in the status-line payload on every render and are always current.

  They do carry a note of their own, inherited from upstream: `(as of Xm ago)`, added once the stored numbers have gone ten minutes without changing. It times how long the figures have held still, which is not the same as doubting them — an idle hour spends nothing, so nothing changes, and the note appears beside percentages that are exactly right. It is also sixteen columns wide, enough to push a merged header onto a second row. `quota_age_note = false` drops it.

### Configuration

Run the interactive wizard from inside Claude Code:

```
/cc-probeline-config
```

It walks you through probes, table size and colours — and writes the TOML for you. Or edit `~/.config/cc-probeline/config.toml` directly (validate with `cc-probeline check-config`).

Every field is optional and every default is the full layout, so you only ever write the lines you want to change:

```toml
[general]
table_rows     = 10   # per-turn cost table: rows to keep (0 = no table, max 40)
table_legend   = true # the "# role model cache r/w …" labels      (2 rows)
table_frame    = true # ┌─┐ / └─┘ borders and the outer bars        (2 rows)
table_dividers = true # inner │ separators — costs no rows, only visual noise
header_line0   = true # email • project • quota
header_line1   = true # model • ctx • cost • time • git
columns        = 0    # layout width; 0 detects it — see `cc-probeline diag`
alerts         = true # the notice row: cache rebuilt, config errors, update
quota_age_note = true # "(as of Xm ago)" on 5h/7d — how long they held still
usage_refresh  = true # keep the /usage cache fresh via headless `claude -p "/usage"`
header_merge   = false # join the two header rows when they fit on one
bar_width      = 10   # progress-bar segments, 3-20 (8 keeps small values visible)
bar_style      = "block" # or line / low / dot / none (percentage instead of a bar)
effort_style   = "glyph" # or "word" — spell the effort out instead of ◕
model_variant_tag = true # false drops the "[1m]" tag from the model name
tutorial_hints = true # false stops the rotating tips; alerts still show

[widgets]             # flip any single segment off
email = false

[colors]              # repoint any palette role: "#rrggbb" or raw SGR
sage  = "38;5;108"    # green · sage · amber · yellow · orange · red · cyan · magenta
```

Config is read in precedence order: `CC_PROBELINE_CONFIG=/path` (explicit override) → `.cc-probeline.toml` in the current repo (project-local) → `~/.config/cc-probeline/config.toml` (global). An invalid value never breaks the status line — it falls back to the default.

Full reference — colour thresholds, cost budget, every widget: [`scripts/config.toml.example`](scripts/config.toml.example).

#### Four layouts to start from

The keys stack, so the dashboard scales from eight rows down to one. Pick whichever trade you want and paste it.

**As it ships — 8 rows.** Everything on: quota header, session header, and the last 10 turns priced in a bordered table with column labels. Nothing to configure.

**Sensible middle ground — 5 rows.** Keeps every number that costs you money and drops only the decoration around it:

```toml
[general]
table_rows     = 3      # the last few turns, still priced
table_legend   = false  # column labels stop paying rent once you know them
table_frame    = false  # columns stay aligned without the borders
tutorial_hints = false

[widgets]
email   = false         # you know whose account this is
project = false         # your terminal title already says it
```

```
5h: █░░░░░░░░░ ↻ 0h:43m · 7d: ▒░░░░░░░░░ ↻ 6d.8h • Fable 7d: ▒░░░░░░░░░ ↻ 6d.8h
opus-5 • cost: $4.22 • time: 08:34 • ⎇ main ⚠ 10
196 │ orchestrator │ opus-5 │ 326K     1K │   109 │   $0.18 │ Read          ⏱ 60m
195 │ orchestrator │ opus-5 │ 325K    883 │   255 │   $0.18 │ thinking...
194 ┼ orchestrator ┼ opus-5 ┼ 320K     5K ┼   825 ┼   $0.23 ┼ thinking...
```

**Limits only — 1 row.** For when the status line should answer exactly one question, *how much is left*, and get out of the way:

```toml
[general]
table_rows     = 0      # no per-turn table at all
header_line1   = false  # drop model • ctx • cost • time • git
tutorial_hints = false

[widgets]
email   = false
project = false
```

```
5h: █░░░░░░░░░ ↻ 0h:43m · 7d: ▒░░░░░░░░░ ↻ 6d.8h • Fable 7d: ▒░░░░░░░░░ ↻ 6d.8h
```

A header line whose every segment is switched off is dropped entirely rather than left blank, so the widget toggles reclaim the row too.

**The one this fork is run with — 1 row.** Same single line as above, but it spends those columns on the three things worth watching rather than on drawing. Every quota is a bare percentage instead of a bar, which is both shorter and more precise, and the row it frees goes to context and the model:

```toml
[general]
table_rows        = 0        # no per-turn table
header_merge      = true     # both header rows on one line while they fit
bar_style         = "none"   # a percentage says more than ten segments, in fewer columns
bar_width         = 8        # only reached if you put the bars back
effort_style      = "word"   # "xhigh" instead of ◕ — no legend to remember
model_variant_tag = false    # ctx already spells the window out
table_legend      = false
table_frame       = false
table_dividers    = false
tutorial_hints    = false
alerts            = false    # see the note below before copying this line
quota_age_note    = false    # "(as of Xm ago)" times how long 5h/7d held still, not staleness
columns           = 155      # your real terminal width — `cc-probeline diag` prints it

[widgets]
ctx     = true               # the one number that decides when to /compact
quota   = true
quota_model = true           # Fable 7d
model   = true
effort  = true
cost    = false              # the table is gone, so this is a total with no story
project = false              # the terminal title already says it
email   = false
time    = false
git     = false              # the editor shows the branch

[colors]                     # a bare percentage wants a calmer hue than a bar does
sage  = "#7cb27c"            # healthy band, values shown as text
amber = "38;5;179"           # notice band, values shown as text
```

```
5h: 12% ↻ 0h:40m · 7d: 4% ↻ 6d.7h • Fable 7d: 3% • opus-5 xhigh • ctx: 74% 743K/1000K
```

Two of those lines are opinions rather than savings, and both are worth a moment. `alerts = false` costs you `Cache rebuilt · 60-min idle TTL passed`, the one notice that explains a turn which suddenly cost more than its neighbours — keep it on until the row genuinely hurts. `columns` is per-machine and does nothing for anyone else: set it to your own width, or leave it at 0 and let detection try.

`Fable 7d` carries no countdown here because the scoped window rolls over with the account-wide one, so the `7d` block beside it already answers the question.

### Updating

```sh
git pull && go build -o ~/.local/bin/cc-probeline ./cmd/cc-probeline
```

`/cc-probeline-update` upgrades through the packaged channels, so it would pull
the original over your build — do not use it here.

The status line also carries an `↑ update: vX → vY` notice driven by the same
once-a-day check. In this fork it reports **upstream's** latest version, which is
not what you are running; the check is still worth keeping for the other half of
its job, refreshing the price table so cost estimates track Anthropic's rates.
Turn the whole thing off with `price_check = false` if the notice bothers you
more than stale prices would.


### Uninstall

```sh
cc-probeline uninstall        # restores the status line you had before
rm ~/.local/bin/cc-probeline
```

The restore is byte-for-byte if cc-probeline replaced an existing status line.
**Restart Claude Code afterwards.** The packaged uninstall paths (`brew
uninstall`, the curl script's `--uninstall`) belong to the original and have
nothing to remove here.

## The experiment

The original is a personal experiment by its author: can you hand programming
over to AI **entirely** — every line of code, every design decision — and still
end up with a product that matches the operator's vision exactly? Their answer,
and the build log that goes with it, is in
[the upstream README](https://github.com/labzink/cc-probeline#the-experiment).

This fork was built the same way and is a second data point for the same
question: the changes here were specified in conversation, written by Claude,
and reviewed decision by decision — including the four upstream bugs the work
turned up, which were found by reading the code rather than by hitting them.

**Contributing:** bug reports and ideas are welcome — open an issue.

MIT License. This is a fork of [cc-probeline](https://github.com/labzink/cc-probeline) by Konstantin Labzin, extended with configurable vertical footprint and per-model quota windows.
