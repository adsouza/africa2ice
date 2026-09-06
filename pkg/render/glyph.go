package render

import (
	"bytes"
	_ "embed"
	"image/color"

	"github.com/adsouza/africa2ice/pkg/gameapi"
	"github.com/hajimehoshi/ebiten/v2/text/v2"
)

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

// biomeGlyphOverrides replaces a font glyph with a hand-drawn painter.
//
// Noto Emoji is line art, and line art is what dissolves at this size: a
// contour authored at ~20 font units lands at 0.2 px once scaled and vanishes
// into antialiasing, while a filled region merely shrinks. The six chosen
// glyphs measured on the survivable side, but riverine woodland is closest to
// the line at 173/576 px changed in TestRuneGlyphPaintsInkAtFocusTileSize,
// against 234-273/576 for the other five. This ships empty; an entry here
// needs no caller changes.
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
