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

	// glyphCellFraction is the font em size as a fraction of the drawn cell,
	// chosen so the widest glyph's *ink* fits the 20 DIP box spec 5.2 asks
	// for inside the 24 DIP cell -- leaving the ~2 DIP margin that keeps
	// neighbouring tiles' glyphs from merging across the 0.4 DIP gap
	// drawFlatTerrain leaves.
	//
	// Em size is not ink extent, and for this vocabulary the gap is large.
	// Measured from the embedded subset's own glyf bounding boxes (units of
	// a 2048 em): mountainous highlands spans 2420 = 1.182 em and savanna
	// 2340 = 1.143 em. A 20 DIP em therefore painted 23.6 DIP of ink, and
	// both -- savanna being the commonest biome on this map -- filled the
	// cell edge to edge and touched their neighbours, which is precisely
	// what the margin exists to prevent. 20/1.182 = 16.93 DIP is the largest
	// em whose ink still fits; 16.8 is that rounded down to a fraction the
	// cell divides exactly, and measures 19.85 DIP of ink for a 2.07 DIP
	// margin per side. TestRuneGlyphPaintsInkAtFocusTileSize pins the ink
	// box so a future size change cannot quietly reopen this.
	glyphCellFraction = 16.8 / (mapTileSize * FocusScale)

	// glyphCellMargin is the DIP clearance the above buys on each side of the
	// cell, rounded down to the whole pixel a rasterized ink box can be
	// asserted against.
	glyphCellMargin = 2
)

// biomeGlyphOverrides replaces a font glyph with a hand-drawn painter.
//
// Noto Emoji is line art, and line art is what dissolves at this size: a
// contour authored at ~20 font units lands at 0.16 px once scaled (20/2048 of
// the 16.8 DIP em) and vanishes into antialiasing, while a filled region merely
// shrinks. The six chosen glyphs measured on the survivable side, but riverine
// woodland is closest to the line at 138/576 px changed in
// TestRuneGlyphPaintsInkAtFocusTileSize, against 177-203/576 for the other
// five. This ships empty; an entry here needs no caller changes.
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

// newBiomeGlyphs builds the per-biome painter table, applying
// biomeGlyphOverrides last so a hand-drawn replacement wins.
func newBiomeGlyphs() ([gameapi.BiomeCount]glyphPainter, error) {
	var painters [gameapi.BiomeCount]glyphPainter
	source, err := text.NewGoTextFaceSource(bytes.NewReader(biomeGlyphFont))
	if err != nil {
		return painters, err
	}
	// Semi-arid desert is the cactus, not U+1F3DC: Noto draws that as a framed
	// desert scene, far too busy inside a 24 DIP cell.
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
	return painters, nil
}
