# Tile Biome Glyphs Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Draw a monochrome biome pictograph on each land tile at focus zoom, and in each biome legend swatch, as a shape channel redundant with the existing L\* colour ladder.

**Architecture:** A new `pkg/render/glyph.go` owns a per-biome painter table, an ink rule, and a zoom gate. Glyphs are drawn in a **separate pass after the terrain blit** — never baked into the terrain cache, which is a fixed 8 px-per-tile image that gets nearest-neighbour upscaled at focus and would render any baked glyph as blocky mush. Text rasterisation reuses ebiten `text/v2` exactly as `MapScene.drawText` already does, with a second `GoTextFaceSource` built from an embedded, committed font subset.

**Tech Stack:** Go 1.26.4, ebiten v2.9.10 (`text/v2`), `golang.org/x/image` v0.43.0, Noto Emoji (OFL 1.1) subset embedded via `go:embed`.

**Spec:** `docs/superpowers/specs/2026-09-05-tile-biome-glyphs-design.md`

## Global Constraints

- **No new module dependency.** `go.mod` must not gain a `require`. `pkg/render`'s archtest allow-list (`internal/archtest/arch_test.go:132`) permits only `pkg/gameapi`, `github.com/hajimehoshi/ebiten/v2`, and `golang.org/x/image`. `embed` is standard library and `isStandardLibrary` exempts it — do not edit the allow-list.
- **No changes to** `internal/domain`, `internal/application`, `pkg/gameapi`, saves, state hashes, or the turn contract.
- **Compressed wasm ceiling** `MaxCompressedWasmBytes = 3_600_000`. Release build before this work is 3,288,292 B brotli. The one-way ratchet in DESIGN.md §10 must not be touched.
- **Colour source of truth** is `climateBiomeColor` (`pkg/render/map.go:883`). Do not modify it.
- **Ink rule is maximum contrast, never a lightness threshold.** See spec §6.
- **Comment style:** this codebase writes comments that explain *why*, with measured values (see `pkg/render/lightness.go`, `pkg/render/biome_contrast_test.go`). Match that. Do not add narration comments that restate the code.
- **Reuse `linearize`** from `pkg/render/lightness.go:35`. Do not write a second sRGB linearisation.
- **Run `./tools/check_release_readiness.sh` locally before the final commit** — it needs no CI artifacts, and skipping it has cost a red push before.

## Deviations from the spec

Two, both deliberate. A reviewer should check these are still the right call rather than assume the spec was followed.

1. **No bespoke mask cache.** Spec §7.2 specifies a mask cache keyed `(rune, physicalPx, ink)` and invalidated on `canvas.scale` change. This plan instead rasterises through `text.Draw`, which already maintains ebiten's internal glyph-image cache — a second parallel cache would duplicate it and add an invalidation bug surface for no gain. The property §7.2 was protecting (rasterise at physical pixels, not DIP) is still tested, by `TestBiomeGlyphsRasterizeAtPhysicalScale` in Task 4.
2. **Ink-coverage assertions instead of golden images.** Spec §12 asks for golden images per biome. This plan asserts ink pixel *coverage* per biome instead: it catches the failure mode that actually threatens this feature — a line-art glyph dissolving into a hairline — without committing binary fixtures that must be regenerated whenever the palette or font moves. If a future change needs to pin exact glyph shapes, golden images can be added then.

## File Structure

| File | Responsibility |
|---|---|
| `tools/subset_glyph_font.sh` (create) | Reproducibly regenerate the font subset from an upstream Noto Emoji release |
| `pkg/render/assets/biomeglyphs.ttf` (create) | Committed 6-glyph subset, the embedded build input |
| `pkg/render/glyph.go` (create) | Ink rule, zoom gate, biome→painter table, font-backed painter, override hook |
| `pkg/render/glyph_test.go` (create) | Unit tests for all of the above |
| `pkg/render/map.go` (modify) | `MapScene` gains a glyph face source + draw counter; new glyph pass in `drawFrame`; legend layout |
| `pkg/render/map_test.go` (modify) | Zoom-gate integration assertions |
| `pkg/render/legend_test.go` (modify) | Legend layout assertions |
| `THIRD_PARTY_NOTICES.md` (modify) | OFL 1.1 section |
| `tools/check_audits/main.go` (modify) | Assert the new licence section is present |
| `docs/DESIGN.md` (modify) | Appendix C row + §6 presentation note |

---

### Task 1: Commit a reproducible font subset with its licence and audit

**Files:**
- Create: `tools/subset_glyph_font.sh`
- Create: `pkg/render/assets/biomeglyphs.ttf`
- Modify: `THIRD_PARTY_NOTICES.md`
- Modify: `tools/check_audits/main.go:57-64`

**Interfaces:**
- Consumes: nothing.
- Produces: `pkg/render/assets/biomeglyphs.ttf`, a TrueType font containing exactly the runes `🌳 🌾 🐚 ⛰ 🌵 ❄` (U+1F333, U+1F33E, U+1F41A, U+26F0, U+1F335, U+2744), all with non-empty outlines.

- [ ] **Step 1: Write the subsetting script**

`tools/subset_glyph_font.sh`:

```bash
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
# semi-arid desert, and glacial tundra.
readonly UNICODES="U+1F333,U+1F33E,U+1F41A,U+26F0,U+1F335,U+2744"

work="$(mktemp -d)"
trap 'rm -rf "$work"' EXIT

curl -sSL --fail --max-time 120 -o "$work/upstream.ttf" "$UPSTREAM"

# Instance the weight axis away first: a variable font carries gvar deltas for
# every glyph, and nothing here varies weight at runtime.
python3 -m fontTools.varLib.instancer "$work/upstream.ttf" wght=400 -o "$work/static.ttf"

python3 -m fontTools.subset "$work/static.ttf" \
  --unicodes="$UNICODES" \
  --output-file="$OUT" \
  --no-hinting --desubroutinize \
  --layout-features='' \
  --drop-tables+=DSIG,GSUB,GPOS

python3 - "$OUT" <<'PY'
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
```

- [ ] **Step 2: Run it and verify the subset**

```bash
chmod +x tools/subset_glyph_font.sh && ./tools/subset_glyph_font.sh
```

Expected: `subset OK: 6 glyphs, 6052 bytes` (a byte count within ~200 of 6052 is fine — upstream Noto Emoji may have advanced). If it reports missing glyphs, stop: the vocabulary and the upstream font have diverged and that is a design question, not a build fix.

- [ ] **Step 3: Add the OFL section to THIRD_PARTY_NOTICES.md**

Append a section, mirroring the existing `## Bundled Go Regular font` section's shape (it lives around line 469):

```markdown
## Bundled Noto Emoji subset

`pkg/render/assets/biomeglyphs.ttf` is a six-glyph subset of Noto Emoji, regenerated by
`tools/subset_glyph_font.sh` from `google/fonts` at `ofl/notoemoji/NotoEmoji[wght].ttf`. It
contains only the biome pictographs the map legend and terrain layer draw. Its complete license
is reproduced below.

### Noto Emoji — SIL Open Font License 1.1

```text
<paste the full verbatim contents of https://raw.githubusercontent.com/google/fonts/main/ofl/notoemoji/OFL.txt here>
```
```

Fetch the licence text with `curl -sSL https://raw.githubusercontent.com/google/fonts/main/ofl/notoemoji/OFL.txt` and paste it verbatim. Do not summarise or truncate it.

- [ ] **Step 4: Write the failing audit assertion**

In `tools/check_audits/main.go`, inside the existing `for _, required := range [...]string{` block (around line 65), add one entry:

```go
		"### Noto Emoji — SIL Open Font License 1.1",
```

And after the existing Go Regular checks (around line 64), add:

```go
	if !strings.Contains(string(notices), "## Bundled Noto Emoji subset") {
		fail(errors.New("THIRD_PARTY_NOTICES.md does not include the bundled Noto Emoji subset license"))
	}
```

- [ ] **Step 5: Run the audit**

Run: `go run ./tools/check_audits`
Expected: exits 0. If it fails, the notices section heading does not match the asserted string exactly — fix the heading, not the assertion.

- [ ] **Step 6: Commit**

```bash
git add tools/subset_glyph_font.sh pkg/render/assets/biomeglyphs.ttf THIRD_PARTY_NOTICES.md tools/check_audits/main.go
git commit -m "build: vendor a six-glyph Noto Emoji subset with its OFL licence

The subset is committed rather than fetched at build time so the bytes that
ship are reviewable and the build stays hermetic. check_audits now asserts the
licence section the way it already does for Go Regular.

Co-Authored-By: Claude Opus 5 <noreply@anthropic.com>"
```

---

### Task 2: Ink selection by maximum contrast

**Files:**
- Create: `pkg/render/glyph.go`
- Create: `pkg/render/glyph_test.go`

**Interfaces:**
- Consumes: `linearize(channel uint8) float64` from `pkg/render/lightness.go:35`.
- Produces:
  - `glyphInkLight`, `glyphInkDark` — package-level `color.RGBA` constants.
  - `glyphInk(background color.RGBA) color.RGBA`
  - `contrastRatio(first, second color.RGBA) float64`

- [ ] **Step 1: Write the failing test**

`pkg/render/glyph_test.go`:

```go
package render

import (
	"image/color"
	"testing"

	"github.com/adsouza/africa2ice/pkg/gameapi"
)

// minGlyphContrast is WCAG 2.2 SC 1.4.11's ratio for non-text graphical
// objects. The 4.5:1 text ratio deliberately does not apply: these are
// pictographs redundant with the tile fill, not text carrying meaning alone.
const minGlyphContrast = 3.0

func TestGlyphInkPicksTheHigherContrastOfTheTwoInks(t *testing.T) {
	for biome := gameapi.Biome(0); biome < gameapi.BiomeCount; biome++ {
		background := climateBiomeColor(biome, 0)
		chosen := glyphInk(background)
		rejected := glyphInkDark
		if chosen == glyphInkDark {
			rejected = glyphInkLight
		}
		chosenRatio := contrastRatio(background, chosen)
		rejectedRatio := contrastRatio(background, rejected)
		if chosenRatio < rejectedRatio {
			t.Errorf("%v: chose ink at %.2f over ink at %.2f", biome, chosenRatio, rejectedRatio)
		}
		if chosenRatio < minGlyphContrast {
			t.Errorf("%v: chosen ink contrast %.2f is below %.2f", biome, chosenRatio, minGlyphContrast)
		}
	}
}

// A lightness threshold at L* 55 -- the rule this implementation replaced --
// assigns light ink to coastal shrubland at 4.20 when dark scores 4.42. This
// pins the case that motivated maximizing contrast instead, so a future
// "simplification" back to a threshold fails here rather than in play.
func TestCoastalShrublandTakesDarkInkDespiteSittingBelowLStar55(t *testing.T) {
	background := climateBiomeColor(gameapi.CoastalShrubland, 0)
	if lightness := cieLightness(background); lightness > 55 {
		t.Fatalf("fixture no longer holds: coastal shrubland L* = %.1f, expected below 55", lightness)
	}
	if got := glyphInk(background); got != glyphInkDark {
		t.Errorf("coastal shrubland ink = %v, want dark", got)
	}
}

func TestContrastRatioIsSymmetricAndBoundedByBlackOnWhite(t *testing.T) {
	white, black := color.RGBA{R: 255, G: 255, B: 255, A: 255}, color.RGBA{A: 255}
	if forward, reverse := contrastRatio(white, black), contrastRatio(black, white); forward != reverse {
		t.Errorf("contrastRatio is not symmetric: %.4f vs %.4f", forward, reverse)
	}
	if got := contrastRatio(white, black); got < 20.9 || got > 21.1 {
		t.Errorf("black on white = %.2f, want 21", got)
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./pkg/render/ -run 'TestGlyphInk|TestCoastalShrubland|TestContrastRatio' -v`
Expected: FAIL — `undefined: glyphInk`, `undefined: glyphInkDark`, `undefined: contrastRatio`.

- [ ] **Step 3: Write minimal implementation**

`pkg/render/glyph.go`:

```go
package render

import "image/color"

// The two candidate inks for biome pictographs. They are near-black and
// near-white rather than pure, so a glyph never out-contrasts the HUD chrome
// it sits beside.
var (
	glyphInkLight = color.RGBA{R: 252, G: 252, B: 252, A: 255}
	glyphInkDark  = color.RGBA{R: 16, G: 16, B: 16, A: 255}
)

// glyphInk returns whichever ink has the greater contrast against background.
//
// A lightness threshold was measured and rejected. At L* 55 coastal shrubland
// (L* 51.1) takes light ink at 4.20 when dark scores 4.42 -- the threshold
// picks the worse ink for the one biome nearest a tie, which is exactly where
// a tuned constant fails. Maximizing contrast needs no constant and stays
// correct if a palette entry ever moves.
func glyphInk(background color.RGBA) color.RGBA {
	if contrastRatio(background, glyphInkDark) >= contrastRatio(background, glyphInkLight) {
		return glyphInkDark
	}
	return glyphInkLight
}

// contrastRatio is the WCAG 2.2 contrast ratio of two opaque sRGB colors.
func contrastRatio(first, second color.RGBA) float64 {
	lighter, darker := relativeLuminance(first), relativeLuminance(second)
	if lighter < darker {
		lighter, darker = darker, lighter
	}
	return (lighter + 0.05) / (darker + 0.05)
}

// relativeLuminance is WCAG's Y, which uses its own published coefficients
// rather than the D65 matrix in cieLab. linearize is shared with lightness.go.
func relativeLuminance(value color.RGBA) float64 {
	return 0.2126*linearize(value.R) + 0.7152*linearize(value.G) + 0.0722*linearize(value.B)
}
```

- [ ] **Step 4: Run test to verify it passes**

Run: `go test ./pkg/render/ -run 'TestGlyphInk|TestCoastalShrubland|TestContrastRatio' -v`
Expected: PASS, all three.

- [ ] **Step 5: Commit**

```bash
git add pkg/render/glyph.go pkg/render/glyph_test.go
git commit -m "feat: choose glyph ink by maximum contrast, not a lightness threshold

A threshold at L* 55 assigns light ink to coastal shrubland at 4.20 when dark
scores 4.42. Maximising contrast needs no tuned constant and holds if a palette
entry moves; a test pins the shrubland case so a revert to a threshold fails
here rather than in play.

Co-Authored-By: Claude Opus 5 <noreply@anthropic.com>"
```

---

### Task 3: Zoom gate, glyph vocabulary, and the font-backed painter

**Files:**
- Modify: `pkg/render/glyph.go`
- Modify: `pkg/render/glyph_test.go`

**Interfaces:**
- Consumes: `glyphInk` (Task 2); `logicalCanvas` (`pkg/render/canvas.go:13`); `mapTileSize = 8`, `FocusScale = 3.0`.
- Produces:
  - `glyphAlpha(cell float32) float64` — 0 at overview, ramps to 1 at focus.
  - `type glyphPainter interface { paint(destination logicalCanvas, centreX, centreY, sizeDIP float32, ink color.RGBA, alpha float64) }`
  - `type runeGlyph struct { source *text.GoTextFaceSource; value string }` implementing `glyphPainter`.
  - `newBiomeGlyphs() ([gameapi.BiomeCount]glyphPainter, glyphPainter, error)` — biome table plus the water painter used by the legend.
  - `biomeGlyphOverrides map[gameapi.Biome]glyphPainter` — empty; the hand-drawn escape hatch.

- [ ] **Step 1: Write the failing test**

Append to `pkg/render/glyph_test.go`:

```go
// The terrain cache is a fixed 8 px-per-tile image that drawTerrain scales up
// with FilterNearest, so a glyph baked into it would be upscaled 3x and read as
// mush. Glyphs are therefore a separate pass gated on the drawn cell size, and
// the gate is a pure function of Cell so it can be tested without a camera.
func TestGlyphAlphaIsZeroAtOverviewAndFullAtFocus(t *testing.T) {
	if got := glyphAlpha(mapTileSize); got != 0 {
		t.Errorf("overview cell %v alpha = %v, want 0", float32(mapTileSize), got)
	}
	if got := glyphAlpha(mapTileSize * FocusScale); got != 1 {
		t.Errorf("focus cell %v alpha = %v, want 1", float32(mapTileSize*FocusScale), got)
	}
	if got := glyphAlpha(glyphMinCell - 0.01); got != 0 {
		t.Errorf("just below the floor alpha = %v, want 0", got)
	}
	previous := 0.0
	for cell := float32(glyphMinCell); cell <= mapTileSize*FocusScale; cell += 0.5 {
		alpha := glyphAlpha(cell)
		if alpha < previous {
			t.Fatalf("alpha fell from %v to %v at cell %v", previous, alpha, cell)
		}
		previous = alpha
	}
}

func TestBiomeGlyphsCoverEveryBiome(t *testing.T) {
	painters, water, err := newBiomeGlyphs()
	if err != nil {
		t.Fatalf("newBiomeGlyphs: %v", err)
	}
	if water == nil {
		t.Error("water painter is nil")
	}
	for biome := gameapi.Biome(0); biome < gameapi.BiomeCount; biome++ {
		if painters[biome] == nil {
			t.Errorf("%v has no painter", biome)
		}
	}
}

// The override map is the escape hatch for a glyph that reads poorly at 20 DIP.
// It ships empty, so this proves the mechanism is live rather than assumed.
func TestBiomeGlyphOverrideTakesPrecedenceOverTheFontGlyph(t *testing.T) {
	stub := &countingGlyphPainter{}
	biomeGlyphOverrides[gameapi.Savanna] = stub
	defer delete(biomeGlyphOverrides, gameapi.Savanna)

	painters, _, err := newBiomeGlyphs()
	if err != nil {
		t.Fatalf("newBiomeGlyphs: %v", err)
	}
	if painters[gameapi.Savanna] != glyphPainter(stub) {
		t.Fatal("override did not replace the savanna painter")
	}
	if painters[gameapi.RiverineWoodland] == glyphPainter(stub) {
		t.Error("override leaked onto an unrelated biome")
	}
}

type countingGlyphPainter struct{ calls int }

func (p *countingGlyphPainter) paint(logicalCanvas, float32, float32, float32, color.RGBA, float64) {
	p.calls++
}

// A painted glyph must actually change pixels at the size it ships at. This is
// the one test that proves the ebiten text path renders these outlines at all;
// everything above it would pass against a painter that drew nothing.
func TestRuneGlyphPaintsInkAtFocusTileSize(t *testing.T) {
	painters, _, err := newBiomeGlyphs()
	if err != nil {
		t.Fatalf("newBiomeGlyphs: %v", err)
	}
	for biome := gameapi.Biome(0); biome < gameapi.BiomeCount; biome++ {
		background := climateBiomeColor(biome, 0)
		target := ebiten.NewImage(24, 24)
		target.Fill(background)
		painters[biome].paint(newLogicalCanvas(target, 1), 12, 12, 20, glyphInk(background), 1)

		changed := 0
		for y := 0; y < 24; y++ {
			for x := 0; x < 24; x++ {
				if r, g, b, _ := target.At(x, y).RGBA(); r>>8 != uint32(background.R) || g>>8 != uint32(background.G) || b>>8 != uint32(background.B) {
					changed++
				}
			}
		}
		target.Deallocate()
		// A glyph covering under 5% of a 576px cell is a hairline, which is the
		// failure mode line-art fonts have at this size.
		if changed < 29 {
			t.Errorf("%v glyph changed only %d of 576 pixels", biome, changed)
		}
	}
}
```

Add `"github.com/hajimehoshi/ebiten/v2"` to the test file's imports.

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./pkg/render/ -run 'TestGlyphAlpha|TestBiomeGlyph|TestRuneGlyph' -v`
Expected: FAIL — `undefined: glyphAlpha`, `undefined: newBiomeGlyphs`, `undefined: biomeGlyphOverrides`, `undefined: glyphMinCell`.

- [ ] **Step 3: Write minimal implementation**

Append to `pkg/render/glyph.go` (and extend its import block to `bytes`, `embed`, `image/color`, `github.com/hajimehoshi/ebiten/v2/text/v2`, `github.com/adsouza/africa2ice/pkg/gameapi`):

```go
//go:embed assets/biomeglyphs.ttf
var biomeGlyphFont []byte

const (
	// glyphMinCell is the drawn cell size below which no glyph is painted.
	// Measured: at the 7.6 px overview extent all six pictographs are
	// indistinguishable blobs, and no glyph choice rescues that -- the cell is
	// smaller than the stroke structure of any recognizable pictograph. The
	// floor sits at twice the overview cell so glyphs appear only in the
	// second half of the camera transition rather than crawling in from it.
	glyphMinCell = 16.0
)
```

**Do not declare `glyphCellFraction` here.** It belongs to Task 4, which holds its only caller. This repo's `golangci-lint` config enables `unused` (`.golangci.yml:15`), which flags any unexported symbol with zero references anywhere in the package — so a constant declared one task ahead of its first use fails the lint gate even though `go build` and every test pass.

```go

// biomeGlyphOverrides replaces a font glyph with a hand-drawn painter.
//
// Noto Emoji is line art, and line art is what dissolves at this size: a
// contour authored at ~20 font units lands at 0.2 px once scaled and vanishes
// into antialiasing, while a filled region merely shrinks. The six chosen
// glyphs measured on the survivable side, but savanna is closest to the line.
// This ships empty; an entry here needs no caller changes.
var biomeGlyphOverrides = map[gameapi.Biome]glyphPainter{}

// glyphPainter draws one pictograph centred at a DIP point.
type glyphPainter interface {
	paint(destination logicalCanvas, centreX, centreY, sizeDIP float32, ink color.RGBA, alpha float64)
}

// glyphAlpha ramps a glyph in with the drawn cell size. Cell lerps from
// mapTileSize to mapTileSize*FocusScale with the camera's progress
// (CameraGeometry), so gating on it needs no camera coupling.
func glyphAlpha(cell float32) float64 {
	const full = mapTileSize * FocusScale
	if cell <= glyphMinCell {
		return 0
	}
	if cell >= full {
		return 1
	}
	return float64(cell-glyphMinCell) / float64(full-glyphMinCell)
}

// runeGlyph paints a single rune from the embedded subset.
type runeGlyph struct {
	source *text.GoTextFaceSource
	value  string
}

func (glyph *runeGlyph) paint(destination logicalCanvas, centreX, centreY, sizeDIP float32, ink color.RGBA, alpha float64) {
	if alpha <= 0 {
		return
	}
	scale := float64(destination.scale)
	options := &text.DrawOptions{}
	// Rasterize at physical pixels, not DIP: logicalCanvas scales coordinates
	// at draw time, so a DIP-sized glyph would be upscaled and blurry on a
	// high-DPI target. This mirrors drawText (map.go).
	options.GeoM.Translate(float64(centreX)*scale, float64(centreY)*scale)
	options.ColorScale.ScaleWithColor(ink)
	options.ColorScale.ScaleAlpha(float32(alpha))
	options.PrimaryAlign = text.AlignCenter
	options.SecondaryAlign = text.AlignCenter
	face := &text.GoTextFace{Source: glyph.source, Size: float64(sizeDIP) * scale}
	text.Draw(destination.image, glyph.value, face, options)
}

// newBiomeGlyphs builds the per-biome painter table and the legend's water
// painter, applying biomeGlyphOverrides last so a hand-drawn replacement wins.
func newBiomeGlyphs() ([gameapi.BiomeCount]glyphPainter, glyphPainter, error) {
	var painters [gameapi.BiomeCount]glyphPainter
	source, err := text.NewGoTextFaceSource(bytes.NewReader(biomeGlyphFont))
	if err != nil {
		return painters, nil, err
	}
	// Semi-arid desert is the cactus, not U+1F3DC: Noto draws that as a framed
	// desert scene, far too busy at 20 DIP.
	vocabulary := [gameapi.BiomeCount]string{
		gameapi.RiverineWoodland:     "\U0001F333",
		gameapi.Savanna:              "\U0001F33E",
		gameapi.CoastalShrubland:     "\U0001F41A",
		gameapi.MountainousHighlands: "⛰",
		gameapi.SemiAridDesert:       "\U0001F335",
		gameapi.GlacialTundra:        "❄",
	}
	for biome := gameapi.Biome(0); biome < gameapi.BiomeCount; biome++ {
		painters[biome] = &runeGlyph{source: source, value: vocabulary[biome]}
	}
	for biome, override := range biomeGlyphOverrides {
		if int(biome) < len(painters) {
			painters[biome] = override
		}
	}
	return painters, &runeGlyph{source: source, value: "\U0001F30A"}, nil
}
```

- [ ] **Step 4: Run test to verify it passes**

Run: `go test ./pkg/render/ -run 'TestGlyphAlpha|TestBiomeGlyph|TestRuneGlyph' -v`
Expected: PASS, all four.

If `TestRuneGlyphPaintsInkAtFocusTileSize` fails for one biome with a low pixel count, that biome's glyph is a hairline at this size — this is the risk spec §13 names. Do **not** lower the threshold. Record which biome, and raise it as an override candidate.

- [ ] **Step 5: Commit**

```bash
git add pkg/render/glyph.go pkg/render/glyph_test.go
git commit -m "feat: add biome glyph painters gated on drawn cell size

Gating on Cell rather than the camera keeps the rule a pure function: Cell
already lerps from 8 to 24 with camera progress. The override map is the escape
hatch for a glyph that reads as a hairline at 20 DIP, and ships empty with a
test proving it is live.

Co-Authored-By: Claude Opus 5 <noreply@anthropic.com>"
```

---

### Task 4: Draw the glyph pass at focus zoom

**Files:**
- Modify: `pkg/render/map.go` — `MapScene` struct (around line 45), `NewMapScene` (line 129), `drawFrame` (around line 275)
- Modify: `pkg/render/map_test.go`

**Interfaces:**
- Consumes: `newBiomeGlyphs`, `glyphAlpha`, `glyphInk` (Tasks 2–3); `MapGeometry.TilePoint`; `climateBiomeColor`.
- Produces: `MapScene.glyphDraws uint64`, a per-frame counter mirroring the existing `terrainRebuilds` instrumentation; `(*MapScene).drawBiomeGlyphs`; and `glyphCellFraction`.

**Declare `glyphCellFraction` in this task**, in `pkg/render/glyph.go`'s existing const block, because this task holds its only caller and `unused` (`.golangci.yml:15`) fails a constant declared ahead of its first reference:

```go
	// glyphCellFraction leaves a margin inside the cell so neighbouring tiles'
	// glyphs do not merge across the 0.4 DIP gap drawFlatTerrain leaves.
	glyphCellFraction = 20.0 / (mapTileSize * FocusScale)
```

- [ ] **Step 1: Write the failing test**

Append to `pkg/render/map_test.go`:

```go
// Glyphs must be a separate pass from the terrain cache, which is a fixed
// 8px-per-tile image that drawTerrain scales up with FilterNearest. Counting
// draws mirrors the terrainRebuilds instrumentation and is robust where pixel
// matching against an antialiased glyph would not be.
func TestBiomeGlyphsDrawOnlyAtFocusZoom(t *testing.T) {
	frame := representativeRenderFrame()
	screen := ebiten.NewImage(1280, 720)
	defer screen.Deallocate()
	scene := NewMapScene()

	scene.SetCamera(Camera{Mode: CameraOverview}, mapAreaHeight)
	scene.Draw(screen, frame, 7, MigrationPreview{}, "", EndScene{}, false)
	if scene.glyphDraws != 0 {
		t.Fatalf("overview drew %d biome glyphs, want 0", scene.glyphDraws)
	}

	scene.SetCamera(Camera{Mode: CameraFocus, Progress: 1}, mapAreaHeight)
	scene.Draw(screen, frame, 7, MigrationPreview{}, "", EndScene{}, false)
	if scene.glyphDraws == 0 {
		t.Fatal("focus drew no biome glyphs")
	}
	if scene.glyphDraws > TerrainGridWidth*TerrainGridHeight {
		t.Fatalf("focus drew %d glyphs, more than the %d tiles that exist",
			scene.glyphDraws, TerrainGridWidth*TerrainGridHeight)
	}
}

// Water, unexplored and halo tiles assert nothing a pictograph could restate,
// so a glyph there would be new information rather than a redundant channel.
// gameapi.Tile spells this as Land, not Water -- there is no Water field.
func TestBiomeGlyphsSkipUnexploredAndWaterTiles(t *testing.T) {
	frame := representativeRenderFrame()
	explored := 0
	for _, tile := range frame.Tiles {
		if tile.Explored && tile.Land {
			explored++
		}
	}
	if explored == 0 {
		t.Fatal("fixture has no explored land tiles")
	}
	screen := ebiten.NewImage(1280, 720)
	defer screen.Deallocate()
	scene := NewMapScene()
	scene.SetCamera(Camera{Mode: CameraFocus, Progress: 1}, mapAreaHeight)
	scene.Draw(screen, frame, 7, MigrationPreview{}, "", EndScene{}, false)
	if scene.glyphDraws > uint64(explored) {
		t.Fatalf("drew %d glyphs for %d explored land tiles", scene.glyphDraws, explored)
	}
}
```

Add a third test, covering high-DPI correctness. Spec §7.2 called for a bespoke mask cache keyed on physical size and invalidated when `canvas.scale` changes; this plan **deliberately deviates** by rasterising through `text.Draw`, which already maintains ebiten's own glyph-image cache — building a second, parallel cache would duplicate it. The underlying concern the spec was protecting still needs a test: a glyph must be rasterised at physical pixels, not DIP, or it is blurry at 2x.

```go
// logicalCanvas scales coordinates at draw time, so a glyph sized in DIP would
// be upscaled and blurry on a high-DPI target. runeGlyph.paint multiplies size
// by canvas scale, mirroring drawText; this pins that it actually does.
func TestBiomeGlyphsRasterizeAtPhysicalScale(t *testing.T) {
	painters, _, err := newBiomeGlyphs()
	if err != nil {
		t.Fatalf("newBiomeGlyphs: %v", err)
	}
	background := climateBiomeColor(gameapi.RiverineWoodland, 0)
	ink := glyphInk(background)

	inkPixels := func(scale float64, size int) int {
		target := ebiten.NewImage(size, size)
		defer target.Deallocate()
		target.Fill(background)
		centre := float32(size) / 2 / float32(scale)
		painters[gameapi.RiverineWoodland].paint(newLogicalCanvas(target, scale), centre, centre, 20, ink, 1)
		count := 0
		for y := 0; y < size; y++ {
			for x := 0; x < size; x++ {
				if r, g, b, _ := target.At(x, y).RGBA(); r>>8 != uint32(background.R) || g>>8 != uint32(background.G) || b>>8 != uint32(background.B) {
					count++
				}
			}
		}
		return count
	}

	standard, highDPI := inkPixels(1, 24), inkPixels(2, 48)
	// A glyph rasterized at DIP and blitted up would cover the same fraction
	// but from 4x fewer source pixels. Rasterizing at physical size means the
	// 2x target carries close to 4x the ink pixels.
	if float64(highDPI) < 3*float64(standard) {
		t.Errorf("high-DPI glyph covered %d px against %d at 1x; expected near 4x", highDPI, standard)
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./pkg/render/ -run TestBiomeGlyphs -v`
Expected: FAIL — `scene.glyphDraws undefined`.

- [ ] **Step 3: Write minimal implementation**

In `pkg/render/map.go`, add to the `MapScene` struct:

```go
	biomeGlyphs  [gameapi.BiomeCount]glyphPainter
	waterGlyph   glyphPainter
	glyphDraws   uint64
```

In `NewMapScene`, after the existing face source is built:

```go
	painters, water, err := newBiomeGlyphs()
	if err != nil {
		panic(err)
	}
	return &MapScene{faceSource: source, biomeGlyphs: painters, waterGlyph: water}
```

Add the pass:

```go
// drawBiomeGlyphs paints one pictograph per explored land tile, giving biome
// identity a shape channel alongside the L* fill ladder. It is deliberately not
// baked into the terrain cache: that image is a fixed 8 px-per-tile raster the
// camera scales up with FilterNearest, so a baked glyph would be upscaled 3x
// into mush at exactly the zoom where it is meant to be legible.
func (scene *MapScene) drawBiomeGlyphs(screen logicalCanvas, geometry MapGeometry, frame *gameapi.Frame) {
	alpha := glyphAlpha(geometry.Cell)
	if alpha <= 0 {
		return
	}
	size := geometry.Cell * glyphCellFraction
	for _, tile := range frame.Tiles {
		if !tile.Explored || !tile.Land || int(tile.Biome) >= len(scene.biomeGlyphs) {
			continue
		}
		x, y := geometry.TilePoint(tile)
		// TilePoint is valid for every tile; the SubImage clip drops the ones
		// off-screen, so no visibility test is needed here.
		background := climateBiomeColor(tile.Biome, frame.Climate.AridityIndex)
		scene.biomeGlyphs[tile.Biome].paint(screen, x, y, size, glyphInk(background), alpha)
		scene.glyphDraws++
	}
}
```

In `drawFrame`, reset the counter where the frame is composed and call the pass immediately after `drawHalo` — glyphs belong under the selection overlays, markers and rings so band discs stay readable:

```go
	scene.glyphDraws = 0
	scene.drawTerrain(mapCanvas, geometry, frame)
	scene.drawHalo(mapCanvas, geometry, frame)
	scene.drawBiomeGlyphs(mapCanvas, geometry, frame)
	scene.drawReachableTiles(mapCanvas, geometry, frame, selectedBand)
```

- [ ] **Step 4: Run test to verify it passes**

Run: `go test ./pkg/render/ -run TestBiomeGlyphs -v`
Expected: PASS, all three.

Then run the whole package: `go test ./pkg/render/`
Expected: PASS. If `performance_test.go` regresses, the per-frame glyph cost is the cause — report the measured numbers rather than loosening the budget.

- [ ] **Step 5: Commit**

```bash
git add pkg/render/map.go pkg/render/map_test.go
git commit -m "feat: paint biome glyphs in a pass above the terrain cache

The terrain cache is a fixed 8px-per-tile raster scaled with FilterNearest, so
a baked glyph would be upscaled 3x into mush at exactly the zoom meant to make
it legible. The pass sits above it and below the selection overlays.

Co-Authored-By: Claude Opus 5 <noreply@anthropic.com>"
```

---

### Task 5: Glyphs in the legend swatch

**Files:**
- Modify: `pkg/render/map.go:807-822` (`drawMapLegend`)
- Modify: `pkg/render/legend.go` — `mapLegendEntry` gains a biome field
- Modify: `pkg/render/legend_test.go`

**Interfaces:**
- Consumes: `MapScene.biomeGlyphs`, `MapScene.waterGlyph`, `glyphInk` (Tasks 2–4).
- Produces: `mapLegendEntry.glyph glyphPainter` — nil for entries with no pictograph.

- [ ] **Step 1: Write the failing test**

Append to `pkg/render/legend_test.go`:

```go
// The legend is where the glyph vocabulary is taught rather than guessed, so
// every biome swatch must carry the same pictograph its tiles do.
func TestEveryBiomeLegendEntryCarriesAGlyph(t *testing.T) {
	scene := NewMapScene()
	entries := scene.mapLegendEntries(0.4)
	withGlyph := 0
	for _, entry := range entries {
		if entry.glyph != nil {
			withGlyph++
		}
	}
	// The six biomes carry glyphs; water, unknown and escarpment stay bare.
	if withGlyph != int(gameapi.BiomeCount) {
		t.Errorf("%d legend entries carry a glyph, want %d", withGlyph, gameapi.BiomeCount)
	}
}

// An 8x8 swatch cannot host a legible pictograph. This pins the enlarged
// geometry so a later tidy-up cannot silently shrink it back.
func TestLegendSwatchIsLargeEnoughForAGlyph(t *testing.T) {
	if legendSwatchSize < 12 {
		t.Errorf("legend swatch = %v, too small to host a glyph", legendSwatchSize)
	}
	if legendLabelX <= legendSwatchSize+4 {
		t.Errorf("legend label at %v overlaps a %v swatch at x+4", legendLabelX, legendSwatchSize)
	}
	if legendMeaningY+7 > mapLegendHeight {
		t.Errorf("meaning text at %v overflows the %v row", legendMeaningY, mapLegendHeight)
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./pkg/render/ -run TestLegend -run TestEveryBiome -v`
Expected: FAIL — `undefined: legendSwatchSize`, and `mapLegendEntries` is not a method.

- [ ] **Step 3: Write minimal implementation**

In `pkg/render/legend.go`, add `glyph glyphPainter` to `mapLegendEntry` and make `mapLegendEntries` a method on `*MapScene` so it can reach the painter table. Set `glyph: scene.biomeGlyphs[gameapi.RiverineWoodland]` and so on for the six biomes, `glyph: scene.waterGlyph` for water, and leave `unknown` and `escarpment` without one.

In `pkg/render/map.go`, replace the literals in `drawMapLegend` with named constants and draw the glyph:

```go
const (
	// An 8x8 swatch cannot host a legible pictograph; 12 can at 11 DIP.
	legendSwatchSize = float32(12)
	legendLabelX     = float32(19)
	legendMeaningY   = float32(14)
	legendGlyphSize  = float32(11)
)
```

Then in the non-edge branch, after the swatch fill and stroke:

```go
		vector.FillRect(screen, x+4, mapLegendOriginY+4, legendSwatchSize, legendSwatchSize, entry.color, false)
		vector.StrokeRect(screen, x+4, mapLegendOriginY+4, legendSwatchSize, legendSwatchSize, 0.7, color.RGBA{R: 210, G: 216, B: 210, A: 180}, false)
		if entry.glyph != nil {
			centre := x + 4 + legendSwatchSize/2
			entry.glyph.paint(screen, centre, mapLegendOriginY+4+legendSwatchSize/2, legendGlyphSize, glyphInk(entry.color), 1)
		}
```

and update the two `drawText` calls to use `x+legendLabelX` and `mapLegendOriginY+legendMeaningY`.

- [ ] **Step 4: Run tests to verify they pass**

Run: `go test ./pkg/render/ -v`
Expected: PASS.

Making `mapLegendEntries` a method breaks three existing call sites, which must be updated to construct a scene first:

- `legend_test.go:10` in `TestMapLegendExplainsEveryRenderedTileClass` — `entries := mapLegendEntries(0.4)` becomes `entries := NewMapScene().mapLegendEntries(0.4)`
- `legend_test.go:35-36` in `TestLegendUsesStableBiomeIdentityAndTheLiveAtmosphericWaterGrade` — `humid := mapLegendEntries(0)` and `arid := mapLegendEntries(1)` become `scene := NewMapScene()` then `scene.mapLegendEntries(0)` and `scene.mapLegendEntries(1)`
- `map.go:809` in `drawMapLegend` — `entries := mapLegendEntries(aridity)` becomes `entries := scene.mapLegendEntries(aridity)`

Both existing tests assert entry labels, meanings and colours; none assert pixel offsets, so their bodies need no other change. Do not revert the layout constants to make anything pass.

- [ ] **Step 5: Commit**

```bash
git add pkg/render/legend.go pkg/render/legend_test.go pkg/render/map.go
git commit -m "feat: teach the glyph vocabulary in the map legend

An 8x8 swatch cannot host a legible pictograph, so the swatch grows to 12 and
the label and meaning offsets move with it. Named constants replace the inline
literals so the geometry is assertable.

Co-Authored-By: Claude Opus 5 <noreply@anthropic.com>"
```

---

### Task 6: Record the decision and prove the budget

**Files:**
- Modify: `docs/DESIGN.md` — Appendix C row and a §6 presentation note
- Modify: `pkg/render/glyph_test.go`

**Interfaces:**
- Consumes: `biomeGlyphFont` (Task 3).
- Produces: nothing consumed downstream.

- [ ] **Step 1: Write the failing test**

Append to `pkg/render/glyph_test.go`:

```go
// The embedded subset is a transfer-size cost against a ratcheting ceiling, so
// a wider vocabulary must be a visible decision rather than silent growth.
// Raise this only alongside a measured wasm-size check.
func TestEmbeddedGlyphFontStaysWithinItsBudget(t *testing.T) {
	const maxBiomeGlyphFontBytes = 8192
	if len(biomeGlyphFont) == 0 {
		t.Fatal("embedded glyph font is empty")
	}
	if len(biomeGlyphFont) > maxBiomeGlyphFontBytes {
		t.Errorf("embedded glyph font = %d B, above the %d B budget", len(biomeGlyphFont), maxBiomeGlyphFontBytes)
	}
}
```

- [ ] **Step 2: Run test to verify it passes immediately**

Run: `go test ./pkg/render/ -run TestEmbeddedGlyphFont -v`
Expected: PASS at ~6,052 B. This one is a ratchet, not a red-green cycle — it exists to fail on a future change, so verify it fails when you temporarily lower the constant to 1024, then restore it.

- [ ] **Step 3: Add the DESIGN.md entries**

Add an Appendix C row recording the locked decision, matching the table's existing column shape:

| Decision | Value | Status | Section |
|---|---|---|---|
| Biome glyph channel | Monochrome Noto Emoji subset, focus zoom only, ink by maximum contrast | Locked | §6 |

And a §6 presentation note stating: biome identity carries a redundant shape channel at focus zoom; glyphs never encode information absent from the tile fill; band discs deliberately carry no glyph, because at their ~15 px inscribed square two species glyphs are indistinguishable and the figure occludes the wedge boundary it would reinforce.

- [ ] **Step 4: Verify the whole build and the size gate**

```bash
go test ./... && go run ./tools/check_audits && ./tools/check_release_readiness.sh
```

Expected: all pass. Then measure the real transfer cost:

```bash
./build_web.sh --release && brotli -q 11 -f -k web/main.wasm && stat -f%z web/main.wasm.br
```

Expected: under 3,600,000, and within ~6 KB of the 3,288,292 baseline. If it exceeds the ceiling, stop and report — do not touch the ratchet.

- [ ] **Step 5: Commit**

```bash
git add docs/DESIGN.md pkg/render/glyph_test.go
git commit -m "docs: lock the biome glyph channel and budget its embedded subset

Records why band discs carry no glyph, so the rejected half is not revived
without new evidence, and ratchets the embedded font size so a wider vocabulary
is a visible decision.

Co-Authored-By: Claude Opus 5 <noreply@anthropic.com>"
```

---

## Notes for the executor

- **The riskiest step is Task 3, Step 4.** If a biome's glyph changes fewer than 29 of 576 pixels it is a hairline, which is the exact failure mode spec §13 predicts for line-art fonts. Do not lower the threshold to make it pass. Report which biome, and treat it as the first `biomeGlyphOverrides` entry.
- **Spec §13 asks for one in-engine confirmation before building on the offline measurements.** Task 3's `TestRuneGlyphPaintsInkAtFocusTileSize` is that confirmation — it runs inside the `TestMain` ebiten loop (`pkg/render/main_test.go`) where `.At()` is available. If it fails for *every* biome, stop: the offline rasterisation did not predict the engine, and the design needs revisiting rather than the plan continuing.
- **`git diff` is not a patch here** — `diff.external=difft` is configured. Use `git diff --no-ext-diff` when you need machine-readable output.

---

### Task 5b: Remove the water glyph (added mid-run, from user feedback)

This task was not in the original plan. While the feature was being built, the user observed that
the water glyph appeared in the legend but never on map tiles, and chose to remove it rather than
start drawing it on water.

The behaviour was intentional and specified, but the spec was wrong: the legend's job here is to
teach the map's glyph vocabulary, so a glyph the map never draws does not belong in it. Water is
already unambiguous from colour and coastline shape — it was the one legend entry that never needed
a redundant channel, and stamping a pictograph across every sea tile at focus zoom would have been
heavy repeated ink for a distinction the colour already makes.

**Tasks 1, 3, 4 and 5 above are left as originally written.** They record what was instructed at the
time, and this task records what changed. Reading the plan in order gives the true final state; do
not "tidy" the earlier tasks to match, or the plan stops being a usable build log.

**What changed:**
- `pkg/render/legend.go` — the Water entry drops its `glyph`; it keeps its colour swatch.
- `pkg/render/glyph.go` — `newBiomeGlyphs()` returns `([gameapi.BiomeCount]glyphPainter, error)`;
  the water painter is gone.
- `pkg/render/map.go` — `MapScene.waterGlyph` removed.
- `tools/subset_glyph_font.sh` and `pkg/render/assets/biomeglyphs.ttf` — U+1F30A dropped, so the
  vocabulary is six codepoints and the subset is 6052 B raw / 4313 B brotli.
- `TestWaterLegendEntryHasNoGlyph` pins the rule so it cannot silently regress.
