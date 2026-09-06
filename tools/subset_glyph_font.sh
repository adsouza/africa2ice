#!/usr/bin/env bash
# Regenerates pkg/render/assets/biomeglyphs.ttf from upstream Noto Emoji.
#
# The subset is committed rather than fetched at build time so the exact bytes
# that ship are reviewable and the build stays hermetic. Re-run this only when
# the glyph vocabulary in pkg/render/glyph.go changes, and commit the result.
#
# Requires: python3 with fonttools (pip install fonttools), curl.
#
# Lint with `shellcheck`.
set -euo pipefail

# The upstream input is pinned to a commit and verified against its digest,
# mirroring tools/install_golangci_lint.sh: `main` is a moving branch, so
# fetching from it makes "regenerates bit-identically" a claim about whatever
# google/fonts happens to hold today rather than about a fixed input, and a
# substituted or corrupted download would be subsetted and committed in
# silence. UPSTREAM_COMMIT is google/fonts' "Update Noto Emoji to the 3.002
# release" (2024-06-05), the newest commit touching this path; the version it
# reports is what THIRD_PARTY_NOTICES.md records and tools/check_audits
# asserts. Bumping the pin means replacing the digest and the recorded version
# too -- an unverifiable bump cannot pass.
readonly UPSTREAM_COMMIT=b979dba422e445492b0eb9951ac52ee0b4d648c3
readonly UPSTREAM_DIGEST=de6c18832938afc99caf132b39d6a30a19bac7f2e812e28db2535b4608d27551
readonly UPSTREAM_VERSION="Version 3.002"
readonly UPSTREAM="https://raw.githubusercontent.com/google/fonts/$UPSTREAM_COMMIT/ofl/notoemoji/NotoEmoji%5Bwght%5D.ttf"
readonly OUT="$(dirname "$0")/../pkg/render/assets/biomeglyphs.ttf"
# Riverine woodland, savanna, coastal shrubland, mountainous highlands,
# semi-arid desert, and glacial tundra.
readonly UNICODES="U+1F333,U+1F33E,U+1F41A,U+26F0,U+1F335,U+2744"

work="$(mktemp -d)"
trap 'rm -rf "$work"' EXIT

curl -sSL --fail --max-time 120 -o "$work/upstream.ttf" "$UPSTREAM"

# Runners disagree about which checksum tool exists, and a name existing does
# not mean the interface does: macOS ships a sha256sum with no --check at all.
# So compute the digest here and compare it ourselves.
digest_of() {
	for tool in "sha256sum" "shasum -a 256"; do
		# Intentional word splitting: $tool carries its arguments.
		# shellcheck disable=SC2086
		if value=$($tool "$1" 2>/dev/null | awk '{print $1}') &&
			[ "${#value}" -eq 64 ]; then
			printf '%s\n' "$value"
			return 0
		fi
	done
	return 1
}

measured=$(digest_of "$work/upstream.ttf") || {
	echo "subset_glyph_font: no working sha256 tool to verify the upstream font" >&2
	exit 1
}
if [ "$measured" != "$UPSTREAM_DIGEST" ]; then
	echo "subset_glyph_font: upstream font has digest" >&2
	echo "subset_glyph_font:   $measured" >&2
	echo "subset_glyph_font: but the pinned digest is" >&2
	echo "subset_glyph_font:   $UPSTREAM_DIGEST" >&2
	exit 1
fi

# The digest fixes the bytes; this fixes the *name* those bytes claim, because
# that name is what THIRD_PARTY_NOTICES.md records and check_audits asserts.
python3 - "$work/upstream.ttf" "$UPSTREAM_VERSION" <<'VERSION_CHECK'
import sys
from fontTools.ttLib import TTFont

path, expected = sys.argv[1], sys.argv[2]
names = {str(record) for record in TTFont(path)["name"].names if record.nameID == 5}
if expected not in names:
    sys.exit("upstream font reports version %s, expected %s" % (sorted(names), expected))
print("upstream OK: %s" % expected)
VERSION_CHECK

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
    hex(cp) for cp in (0x1F333, 0x1F33E, 0x1F41A, 0x26F0, 0x1F335, 0x2744)
    if cmap.get(cp) is None or glyf[cmap[cp]].numberOfContours == 0
]
if missing:
    sys.exit("subset is missing or has empty outlines for: %s" % ", ".join(missing))
print("subset OK: 6 glyphs, %d bytes" % len(open(sys.argv[1], "rb").read()))
PY

mv "$scratch" "$OUT"
