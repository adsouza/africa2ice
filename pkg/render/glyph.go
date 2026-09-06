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
