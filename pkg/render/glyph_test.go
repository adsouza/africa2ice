package render

import (
	"image/color"
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
		t.Logf("%v: %d/576 pixels changed", biome, changed)
		// A glyph covering under 5% of a 576px cell is a hairline, which is the
		// failure mode line-art fonts have at this size.
		if changed < 29 {
			t.Errorf("%v glyph changed only %d of 576 pixels", biome, changed)
		}
	}
}
