package render

import (
	"image/color"
	"math"
	"testing"

	"github.com/adsouza/africa2ice/pkg/gameapi"
	"github.com/hajimehoshi/ebiten/v2"
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
	// Pin an intermediate point so a degenerate step function (0 below the
	// floor, 1 above it, no ramp) cannot pass: the function is documented to
	// ramp *linearly* from glyphMinCell (16) to mapTileSize*FocusScale (24),
	// so the midpoint cell 20 must land at 0.5, not jump straight to 1.
	const glyphAlphaEpsilon = 1e-9
	if got := glyphAlpha(20); math.Abs(got-0.5) > glyphAlphaEpsilon {
		t.Errorf("midpoint cell 20 alpha = %v, want 0.5 (linear ramp from glyphMinCell to mapTileSize*FocusScale)", got)
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
	painters, err := newBiomeGlyphs()
	if err != nil {
		t.Fatalf("newBiomeGlyphs: %v", err)
	}
	for biome := gameapi.Biome(0); biome < gameapi.BiomeCount; biome++ {
		if painters[biome] == nil {
			t.Errorf("%v has no painter", biome)
		}
	}
}

// The override map is the escape hatch for a glyph that reads poorly in a
// 24 DIP cell. It ships empty, so this proves the mechanism is live rather
// than assumed.
func TestBiomeGlyphOverrideTakesPrecedenceOverTheFontGlyph(t *testing.T) {
	stub := &countingGlyphPainter{}
	biomeGlyphOverrides[gameapi.Savanna] = stub
	defer delete(biomeGlyphOverrides, gameapi.Savanna)

	painters, err := newBiomeGlyphs()
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

// countingGlyphPainter only needs to satisfy glyphPainter; the override test
// distinguishes it from the font-backed painters by identity, not by call
// count.
type countingGlyphPainter struct{}

func (p *countingGlyphPainter) paint(logicalCanvas, float32, float32, float32, color.RGBA, float64) {
}

// A painted glyph must actually change pixels at the size it ships at, in the
// right amount, in the right *place*, and in the ink colour it was actually
// asked to paint. This is the one test that proves the ebiten text path
// renders these outlines at all; a painter that drew nothing, filled the whole
// tile solid, spilled over the cell edge, or ignored its ink argument must all
// fail here.
//
// The ink box is asserted, not just the pixel count, because a count alone
// cannot see extent: an edge-to-edge glyph and a well-margined one of the same
// area are the same number. That gap is exactly how a 20 DIP em -- 23.6 DIP of
// ink for savanna and mountainous highlands -- passed every earlier review of
// this test while leaving no margin at all.
func TestRuneGlyphPaintsInkAtFocusTileSize(t *testing.T) {
	painters, err := newBiomeGlyphs()
	if err != nil {
		t.Fatalf("newBiomeGlyphs: %v", err)
	}
	// The cell and em size the map actually paints at focus.
	const cell = mapTileSize * FocusScale
	size := float32(cell) * glyphCellFraction
	for biome := gameapi.Biome(0); biome < gameapi.BiomeCount; biome++ {
		background := climateBiomeColor(biome, 0)
		ink := glyphInk(background)
		target := ebiten.NewImage(cell, cell)
		target.Fill(background)
		painters[biome].paint(newLogicalCanvas(target, 1), cell/2, cell/2, size, ink, 1)

		changed := 0
		maxCoverage := 0.0 // how far toward pure ink the best-covered pixel gets, 0..1
		minX, minY, maxX, maxY := int(cell), int(cell), -1, -1
		for y := 0; y < cell; y++ {
			for x := 0; x < cell; x++ {
				r, g, b, _ := target.At(x, y).RGBA()
				pr, pg, pb := r>>8, g>>8, b>>8
				if pr == uint32(background.R) && pg == uint32(background.G) && pb == uint32(background.B) {
					continue
				}
				changed++
				minX, minY = min(minX, x), min(minY, y)
				maxX, maxY = max(maxX, x), max(maxY, y)

				// Antialiasing blends background and ink, so an exact-match
				// count is wrong -- a changed pixel is a coverage-weighted
				// blend of the two: pixel = background + t*(ink-background)
				// for some coverage t in (0,1]. Project the pixel onto that
				// line and check the leftover (perpendicular) component is
				// negligible: a painter that ignored its ink argument, or
				// otherwise painted some third colour, leaves a residual far
				// larger than 8-bit rounding noise (measured max
				// residual-squared across all six biomes is under 0.6).
				bgR, bgG, bgB := float64(background.R), float64(background.G), float64(background.B)
				inkR, inkG, inkB := float64(ink.R), float64(ink.G), float64(ink.B)
				dR, dG, dB := inkR-bgR, inkG-bgG, inkB-bgB
				cR, cG, cB := float64(pr)-bgR, float64(pg)-bgG, float64(pb)-bgB
				denom := dR*dR + dG*dG + dB*dB
				if denom == 0 {
					t.Fatalf("%v: ink and background are identical, cannot verify glyph colour", biome)
				}
				coverage := (dR*cR + dG*cG + dB*cB) / denom
				residualR, residualG, residualB := cR-coverage*dR, cG-coverage*dG, cB-coverage*dB
				if residual := residualR*residualR + residualG*residualG + residualB*residualB; residual > 50 {
					t.Errorf("%v: pixel (%d,%d) = (%d,%d,%d) is not a background/ink blend (residual^2=%.1f, coverage=%.2f)", biome, x, y, pr, pg, pb, residual, coverage)
				}
				if coverage > maxCoverage {
					maxCoverage = coverage
				}
			}
		}
		target.Deallocate()
		t.Logf("%v: %d/576 pixels changed, ink box x=[%d..%d] y=[%d..%d], max ink coverage %.2f",
			biome, changed, minX, maxX, minY, maxY, maxCoverage)

		// The margin spec 5.2 asks for is a property of ink extent, not of em
		// size, so assert the extent. Measured at the shipped 16.8 DIP em in
		// the 24 DIP cell, the widest ink box is x=[2..21] -- savanna and
		// mountainous highlands, the two glyphs that filled the cell edge to
		// edge at the old 20 DIP em -- and every box stays inside [2..21] on
		// both axes. So each glyph clears every cell edge by at least
		// glyphCellMargin, and adjacent tiles keep 4 DIP between their ink
		// across the 0.4 DIP gap drawFlatTerrain leaves.
		if minX < glyphCellMargin || maxX > cell-1-glyphCellMargin || minY < glyphCellMargin || maxY > cell-1-glyphCellMargin {
			t.Errorf("%v ink box x=[%d..%d] y=[%d..%d] does not clear the %d DIP cell by %d DIP on every side",
				biome, minX, maxX, minY, maxY, int(cell), glyphCellMargin)
		}

		// A glyph covering under 5% of a 576px cell is a hairline, which is the
		// failure mode line-art fonts have at this size.
		if changed < 29 {
			t.Errorf("%v glyph changed only %d of 576 pixels, want >= 29 (5%% floor)", biome, changed)
		}
		// A ceiling catches a painter that fills the whole tile solid instead
		// of drawing a glyph (576 changed pixels). Measured coverage across
		// all six biomes at the shipped 16.8 DIP em: riverine woodland 138,
		// savanna 186, shrubland 177, highlands 203, desert 184, tundra 202
		// (out of 576) -- a ceiling at 345 (60% of the tile) leaves
		// comfortable headroom above the real maximum (203, ~35%) while still
		// rejecting a solid fill outright.
		if changed > 345 {
			t.Errorf("%v glyph changed %d of 576 pixels, want <= 345 (60%% ceiling)", biome, changed)
		}
		// At least one pixel must actually reach close to full ink coverage,
		// not just a faint fringe -- otherwise a painter that barely tints
		// the background (rather than drawing ink) could still clear the
		// floor above on pixel count alone. Measured max coverage is >=0.92
		// for all six biomes.
		if maxCoverage < 0.8 {
			t.Errorf("%v: no pixel reached >= 0.8 ink coverage (max %.2f); glyph may not actually paint in ink", biome, maxCoverage)
		}
	}
}

// The embedded subset is a transfer-size cost against a ratcheting ceiling, so
// a wider vocabulary must be a visible decision rather than silent growth.
// Raise this only alongside a measured wasm-size check.
//
// 6500 balances two competing pressures against the measured six-glyph subset
// (6,052 B raw). It must be loose enough to absorb upstream Noto Emoji
// redrawing those same six glyphs (a few hundred bytes of drift is normal
// hinting/outline churn, not a vocabulary change), but tight enough to fire
// on the smallest real expansion: a seventh glyph costs roughly 1,008 B
// (6,052 B / 6 glyphs), landing the subset near 7,060 B. 8192 was loose
// enough to let a one-glyph addition through silently; 6500 sits below that
// 7,060 B line with ~450 B of headroom for redraw drift above the current
// 6,052 B, so it trips on any vocabulary addition while tolerating glyph
// reshaping.
func TestEmbeddedGlyphFontStaysWithinItsBudget(t *testing.T) {
	const maxBiomeGlyphFontBytes = 6500
	if len(biomeGlyphFont) == 0 {
		t.Fatal("embedded glyph font is empty")
	}
	if len(biomeGlyphFont) > maxBiomeGlyphFontBytes {
		t.Errorf("embedded glyph font = %d B, above the %d B budget", len(biomeGlyphFont), maxBiomeGlyphFontBytes)
	}
}
