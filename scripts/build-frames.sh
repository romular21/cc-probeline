#!/usr/bin/env sh
# build-frames.sh — regenerate the README screenshots in assets/screenshots/.
#
# The frames are rendered from the code itself, not screenshotted by hand: the
# emitter in tests/frames/ writes one ANSI file per scenario, which is then
# converted to PNG. That keeps the images honest (they always show what the
# current build actually renders) and reproducible by anyone with the repo.
#
# Pipeline:
#
#   tests/frames emitter  →  <scenario>.ansi  →  freeze  →  SVG  →  PNG
#
# Requirements:
#   - Go (to run the emitter)
#   - freeze          go install github.com/charmbracelet/freeze@latest
#   - an SVG renderer: rsvg-convert, inkscape or ImageMagick (whichever is found)
#
# freeze can emit PNG directly, but that path crashes on these inputs (v0.2.2),
# so the script goes through SVG and hands the rasterising to a dedicated tool.
#
# Usage:
#   scripts/build-frames.sh            # write into assets/screenshots/
#   OUT=/tmp/preview scripts/build-frames.sh   # write somewhere else first
#
# Frame → scenario map (mirrors the header of tests/frames/frames_test.go):
#
#   01  full dashboard              s1-rich-baseline
#   02  table close-up              s2-table-config-hint   (header lines cropped)
#   03  extra-usage red             s4-quota-100-extra-commit
#   04  quota warning               s3-quota-split-ctx-warn
#   05  cache-rebuild close-up      s5-cache-rebuild       (header lines cropped)

set -eu

ROOT=$(cd "$(dirname "$0")/.." && pwd)
OUT=${OUT:-"$ROOT/assets/screenshots"}
WORK=$(mktemp -d)
trap 'rm -rf "$WORK"' EXIT

# Rendering knobs. DejaVu Sans Mono is the default because it ships nearly
# everywhere and — unlike many "programming" fonts — carries the full
# box-drawing range at a single advance width, so the table columns stay
# aligned. Override FONT to match your own terminal.
FONT=${FONT:-"DejaVu Sans Mono"}
FONT_SIZE=${FONT_SIZE:-13}
DPI=${DPI:-180}
BG=${BG:-"#0d1117"}

command -v freeze >/dev/null 2>&1 || {
	echo "build-frames: freeze not found — go install github.com/charmbracelet/freeze@latest" >&2
	exit 1
}

# Pick whichever SVG rasteriser is installed.
if command -v rsvg-convert >/dev/null 2>&1; then
	RASTER=rsvg
elif command -v inkscape >/dev/null 2>&1; then
	RASTER=inkscape
elif command -v magick >/dev/null 2>&1 || command -v convert >/dev/null 2>&1; then
	RASTER=magick
else
	echo "build-frames: need one of rsvg-convert, inkscape or ImageMagick to rasterise SVG" >&2
	exit 1
fi

# FLATTEN_DIM rewrites the status line's "faint" attribute (SGR 2, used to fade
# turns from older requests) into explicit dark colours. Terminals honour SGR 2;
# the SVG renderers below silently ignore it, which would paint the whole table
# at full strength and lose the fresh-vs-history contrast entirely. Set it to 0
# only when screenshotting the .ansi files in a real terminal instead.
FLATTEN_DIM=${FLATTEN_DIM:-1}

echo "build-frames: emitting ANSI frames"
CC_PROBELINE_EMIT_DIR="$WORK" CC_PROBELINE_EMIT_FLATTEN_DIM="$FLATTEN_DIM" \
	go test "$ROOT/tests/frames/" -run EmitANSIFrames >/dev/null

# render <scenario> <output-name> [lines-to-crop-from-the-top]
render() {
	scenario=$1
	name=$2
	crop=${3:-0}

	src="$WORK/$scenario.ansi"
	[ -f "$src" ] || {
		echo "build-frames: emitter produced no $scenario.ansi" >&2
		exit 1
	}

	# Frames 02 and 05 are close-ups of the table: drop the header lines so the
	# image starts at the top border.
	if [ "$crop" -gt 0 ]; then
		tail -n +$((crop + 1)) "$src" >"$WORK/$name.ansi"
		src="$WORK/$name.ansi"
	fi

	freeze --language ansi \
		--font.family "$FONT" --font.size "$FONT_SIZE" --line-height 1.25 \
		--padding 24 --margin 0 --border.radius 8 --background "$BG" \
		--output "$WORK/$name.svg" "$src" >/dev/null

	case $RASTER in
	rsvg) rsvg-convert --dpi-x="$DPI" --dpi-y="$DPI" -o "$OUT/$name.png" "$WORK/$name.svg" ;;
	inkscape) inkscape --export-type=png --export-filename="$OUT/$name.png" \
		--export-dpi="$DPI" "$WORK/$name.svg" >/dev/null 2>&1 ;;
	magick)
		if command -v magick >/dev/null 2>&1; then
			magick -density "$DPI" "$WORK/$name.svg" "$OUT/$name.png"
		else
			convert -density "$DPI" "$WORK/$name.svg" "$OUT/$name.png"
		fi
		;;
	esac

	echo "  $name.png  ← $scenario"
}

mkdir -p "$OUT"
echo "build-frames: rendering with $FONT ${FONT_SIZE}pt at ${DPI}dpi via $RASTER → $OUT"

render s1-rich-baseline 01
render s2-table-config-hint 02 2
render s4-quota-100-extra-commit 03
render s3-quota-split-ctx-warn 04
render s5-cache-rebuild 05 2

echo "build-frames: done"
