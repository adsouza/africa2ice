#!/usr/bin/env bash
# Regenerates pkg/render/assets/biomeglyphs.ttf from upstream Noto Emoji.
#
# The subset is committed rather than fetched at build time so the exact bytes
# that ship are reviewable and the build stays hermetic. Re-run this only when
# the glyph vocabulary in pkg/render/glyph.go changes, and commit the result.
#
# Requires: python3 with fonttools (pip install fonttools), curl.
set -euo pipefail

readonly UPSTREAM="https://raw.githubusercontent.com/google/fonts/main/ofl/notoemoji/NotoEmoji%5Bwght%5D.ttf"
readonly OUT="$(dirname "$0")/../pkg/render/assets/biomeglyphs.ttf"
# Riverine woodland, savanna, coastal shrubland, mountainous highlands,
# semi-arid desert, glacial tundra, and water (legend swatch only).
readonly UNICODES="U+1F333,U+1F33E,U+1F41A,U+26F0,U+1F335,U+2744,U+1F30A"

work="$(mktemp -d)"
trap 'rm -rf "$work"' EXIT

curl -sSL --fail --max-time 120 -o "$work/upstream.ttf" "$UPSTREAM"

# Instance the weight axis away first: a variable font carries gvar deltas for
# every glyph, and nothing here varies weight at runtime.
python3 -m fontTools.varLib.instancer "$work/upstream.ttf" wght=400 -o "$work/static.ttf"

# pkg/render/assets/ does not exist until this script creates it.
mkdir -p "$(dirname "$OUT")"

# Subset into a scratch path first: pyftsubset writing straight to $OUT would
# let an interrupted run (killed, disk full, OOM) leave truncated bytes at the
# committed path, and the EXIT trap only cleans up $work. Only mv into $OUT
# below, after the verification block passes.
scratch="$work/biomeglyphs.ttf"

python3 -m fontTools.subset "$work/static.ttf" \
  --unicodes="$UNICODES" \
  --output-file="$scratch" \
  --no-hinting --desubroutinize \
  --layout-features='' \
  --drop-tables+=DSIG,GSUB,GPOS

# Pin head.modified to the upstream font's own value. fontTools stamps a fresh
# wall-clock second into head.modified on every save (TTFont defaults to
# recalcTimestamp=True), so without this, two runs of this script over the
# exact same upstream input produce different bytes and every regeneration
# shows up as a spurious `git diff`. Carrying the upstream font's own
# head.modified keeps the field honest (it still moves when upstream actually
# changes) while making output deterministic for a fixed input. Every other
# table (glyf, cmap, hmtx, ...) is already reproducible; this is the only
# source of run-to-run drift.
python3 - "$work/upstream.ttf" "$scratch" <<'PY'
import sys
from fontTools.ttLib import TTFont

upstream_path, scratch_path = sys.argv[1], sys.argv[2]
upstream_modified = TTFont(upstream_path)["head"].modified

font = TTFont(scratch_path, recalcTimestamp=False)
font["head"].modified = upstream_modified
font.save(scratch_path)
PY

python3 - "$scratch" <<'PY'
import sys
from fontTools.ttLib import TTFont
font = TTFont(sys.argv[1])
cmap, glyf = font.getBestCmap(), font["glyf"]
missing = [
    hex(cp) for cp in (0x1F333, 0x1F33E, 0x1F41A, 0x26F0, 0x1F335, 0x2744, 0x1F30A)
    if cmap.get(cp) is None or glyf[cmap[cp]].numberOfContours == 0
]
if missing:
    sys.exit("subset is missing or has empty outlines for: %s" % ", ".join(missing))
print("subset OK: 7 glyphs, %d bytes" % len(open(sys.argv[1], "rb").read()))
PY

mv "$scratch" "$OUT"
