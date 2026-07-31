---
description: Configure cc-probeline (table size, vertical footprint, display mode, colour, probes) through an interactive widget
---

# Configure cc-probeline: /cc-probeline-config

Interactive configuration for **cc-probeline**, shown as **two** native question widgets (arrow-navigable): page 1 covers **layout** — how many terminal rows the status line occupies — and page 2 covers **probes**. The user checks **only what they want to change** — anything left untouched stays as it is — and all changes are applied in a single batch at the end via the `cc-probeline` CLI setters. Never edits the TOML config directly.

Two pages are required because the widget caps at four questions of four options each, and the settings no longer fit on one.

## Usage

```
/cc-probeline-config
```

No arguments.

---

## Step 0 — Preconditions + read current state

1. If `cc-probeline` is not found in PATH, tell the user the binary is not installed
   yet and suggest running `/cc-probeline-install` to install it, then stop:

   ```
   cc-probeline binary not found in PATH.
   Run /cc-probeline-install to install it, then run /cc-probeline-config again.
   ```

2. Read the current effective config (needed to show 🟢/🔴 and the current table size):

   ```bash
   cc-probeline config show
   ```

   This prints JSON. Extract these values:
   - `general.table_rows` (int)
   - `general.table_legend`, `general.table_frame`, `general.table_dividers` (bool each)
   - `general.header_line0`, `general.header_line1` (bool each)
   - `general.mode` (`"standard"` | `"super-compact"`)
   - `general.no_color` (bool) — note: **colour is ON when `no_color` is false**
   - `general.tutorial_hints` (bool)
   - `general.quota_age_note` (bool) — the `(as of Xm ago)` suffix on the 5h/7d block
   - `general.price_check` (bool) — once-per-day network price/version check (opt-out)
   - `widgets.model`, `widgets.cost`, `widgets.project`, `widgets.email`, `widgets.time`, `widgets.ctx`, `widgets.quota`, `widgets.quota_model`, `widgets.git` (bool each)

---

## Step 1 — Page 1: layout (ONE AskUserQuestion call, four questions)

Call the **AskUserQuestion** tool **once** with the four questions below (do NOT make four separate calls, and never print a plain-text menu).

Fill every "currently …" description and 🟢/🔴 marker from the values read in Step 0 (🟢 = currently on/shown, 🔴 = currently off/hidden). For toggles, the wording must state the flip direction (e.g. "Currently on. Check to hide.").

**Option spacing & indent (required for readability — applies to ALL four questions):** every option's `description` MUST begin with a literal tab (`\t`) and end with a literal newline (`\n`):

- The **leading `\t`** indents the description text (and its 🟢/🔴 marker) from the line start.
- The **trailing `\n`** makes the native widget render a blank line *between* options.
- Keep each description to a **single line** — never add a second description line. The blank line must fall *between* options, not inside one (a `\n` in the middle would split one option's text instead).

So each description has the shape `"\t<🟢/🔴 marker + wording>\n"`. These are verified levers for the `AskUserQuestion` widget.

### Question 1 — `Table rows` (single-select)

- **header:** `Table rows`
- **question:** `Rows in the per-turn table? Current value is first — pick it to keep. (Other = custom, 0–40)`
- **options:** list the **current value first**, labelled `"<N> (current)"`, then three other presets chosen from {0, 5, 10, 20} that are not the current value. The widget auto-offers an **"Other"** free-text choice — rely on it for a custom number (do not add a separate prompt). The binary clamps any value to **0–40**.
- **`0` means no table at all** — rows, legend and frame all go. Label it `0 (no table)` and say so in its description; it is the single biggest vertical saving.

Example when current = 10:
- `10 (current)` — keep the current value, nothing changes.
- `0 (no table)` — drop the per-turn table entirely (frees ~5 rows).
- `5` — minimal table.
- `20` — max useful height.

### Question 2 — `Table chrome` (multi-select, flip)

- **header:** `Table chrome`
- **question:** `Table decoration. Check ONLY what you want to change. (🟢 = currently shown, 🔴 = hidden)`
- **multiSelect:** true
- **options** (3) — state the row saving in each description, it is the whole point:
  - **Legend row** — `🟢 Currently shown. Check to hide (frees 2 rows).` / `🔴 Currently hidden. Check to show.` per `table_legend`.
  - **Frame / borders** — `🟢 Currently shown. Check to hide (frees 2 rows).` / `🔴 …show.` per `table_frame`.
  - **Column dividers** — `🟢 Currently shown. Check to hide (no rows saved, just quieter).` / `🔴 …show.` per `table_dividers`.

Skip this question entirely when `table_rows` is 0 **and** the user did not pick a non-zero value in Question 1 — there is no table to decorate.

### Question 3 — `Header lines` (multi-select, flip)

- **header:** `Header lines`
- **question:** `The two header rows. Check ONLY what you want to change. (🟢 = currently shown, 🔴 = hidden)`
- **multiSelect:** true
- **options** (3):
  - **Account line** — `🟢 Currently shown (email • project • quota). Check to hide.` / `🔴 …show.` per `header_line0`.
  - **Session line** — `🟢 Currently shown (model • ctx • cost • time • git). Check to hide.` / `🔴 …show.` per `header_line1`.
  - **Quota age note** — `🟢 Currently shown ("as of Xm ago"). Check to hide — it times how long 5h/7d held still, not whether they are stale.` / `🔴 Currently hidden. Check to show.` per `quota_age_note`. Sixteen columns wide, so it is a common cause of a merged header wrapping.

Mention in the question text that hiding individual segments instead is done on page 2, and that a header line whose every segment is off disappears on its own.

### Question 4 — `General` (multi-select, flip)

- **header:** `General`
- **question:** `General settings. Check ONLY what you want to change — anything unchecked stays as it is. (🟢 = currently on, 🔴 = currently off)`
- **multiSelect:** true
- **options** (4):
  - **Display mode** — description reflects current: `🟢 Now: standard. Check to switch to super-compact.` (or the reverse when current is super-compact).
  - **Colour output** — `🟢 Currently on. Check to turn it off (monochrome).` (or `🔴 Currently off. Check to turn it on.` when `no_color` is true).
  - **Tutorial hints** — `🟢 Currently on. Check to turn off (alerts still show).` / `🔴 Currently off. Check to turn on.`
  - **Price check (network)** — `🟢 Currently on. Check to turn off (stay fully offline; baked prices).` / `🔴 Currently off. Check to turn on (daily price/version check).` per `price_check`.

---

## Step 2 — Page 2: probes (a second AskUserQuestion call, three questions)

Call **AskUserQuestion** a second time with these three questions. Each carries three probes, so all nine fit within the widget's four-options-per-question limit.

Every option description follows the same pattern as page 1: `🟢 Currently shown. Check to hide.` or `🔴 Currently hidden. Check to show.`, filled from the values read in Step 0.

### Question 1 — `Probes 1` (multi-select, flip)

- **header:** `Probes 1`
- **question:** `Probes (1 of 3) — line 0. Check ONLY the probes you want to flip. (🟢 shown / 🔴 hidden)`
- **multiSelect:** true
- **options** (3): `email`, `project`, `quota` (the account-wide 5h / 7d bars).

### Question 2 — `Probes 2` (multi-select, flip)

- **header:** `Probes 2`
- **question:** `Probes (2 of 3). Check ONLY the probes you want to flip. (🟢 shown / 🔴 hidden)`
- **multiSelect:** true
- **options** (3):
  - **quota_model** — the per-model weekly window (`Fable 7d: …`). Word it as `🟢 Currently shown (per-model weekly limits). Check to hide.` and note that it renders nothing when the account has no model-scoped limit.
  - **git**, and **model** — the latter controls **both** the model name and the effort indicator.

### Question 3 — `Probes 3` (multi-select, flip)

- **header:** `Probes 3`
- **question:** `Probes (3 of 3) — line 1. Check ONLY the probes you want to flip. (🟢 shown / 🔴 hidden)`
- **multiSelect:** true
- **options** (3): `ctx`, `cost`, `time`.

Skip page 2 entirely when the user hid **both** header lines on page 1 — the probes have nowhere left to render.

---

## Step 3 — Apply the choices (single batched run)

Translate the answers from **both** pages into CLI setters and run them in **one** chained bash command (`&&`). Skip anything the user did not change.

- **Table rows:** if the chosen value differs from the current one, add `cc-probeline table-rows <N>`.
- **Legend row** checked → `cc-probeline table-legend off` if currently shown, else `… on`.
- **Frame / borders** checked → `cc-probeline table-frame off` / `… on`.
- **Column dividers** checked → `cc-probeline table-dividers off` / `… on`.
- **Account line** checked → `cc-probeline header-line 0 off` if currently shown, else `… on`.
- **Session line** checked → `cc-probeline header-line 1 off` / `… on`.
- **Quota age note** checked → `cc-probeline quota-age-note off` if currently shown, else `… on`.
- **Display mode** checked → `cc-probeline mode <the other value>` (standard ↔ super-compact).
- **Colour output** checked → flip colour: `cc-probeline no-color on` if colour is currently on (i.e. `no_color` false), else `cc-probeline no-color off`.
- **Tutorial hints** checked → `cc-probeline hints off` if currently on, else `cc-probeline hints on`.
- **Price check (network)** checked → `cc-probeline price-check off` if currently on (`price_check` true), else `cc-probeline price-check on`.
- **Each probe** checked → flip it: `cc-probeline widgets <name> off` if currently shown, else `cc-probeline widgets <name> on`.
  - **`model`** is special: emit **two** commands with the same new state — `cc-probeline widgets model <state>` **and** `cc-probeline widgets effort <state>`.

If nothing was changed, do not run any setter — just print the confirmation.

After the setters run, output exactly one line:

```
Settings saved. Changes take effect on the next status-line refresh.
```

---

## Rules

- **Two AskUserQuestion calls: page 1 (layout, four questions) then page 2 (probes, three questions).** Never split a page into separate calls and never print plain-text menus. Page 2 is skipped when both header lines end up hidden.
- **Spacing & indent:** prefix every option `description` with `\t` and suffix it with `\n` (leading tab = indent, trailing newline = blank line between options). Single-line descriptions only — no second line.
- **Flip semantics:** an unchecked toggle means "leave unchanged". Only checked toggles are flipped. The default (nothing checked) changes nothing.
- **Never hand-edit TOML.** All writes go through the CLI setters.
- **Fill current state from `cc-probeline config show`** — the 🟢/🔴 markers and the "(current)" table-rows option must reflect real values, otherwise the flip is unsafe.
- User-facing wording calls them **probes** (the CLI setter is historically `widgets`).
