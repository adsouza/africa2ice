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
