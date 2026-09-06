package render

import (
	"testing"

	"github.com/adsouza/africa2ice/pkg/gameapi"
	"github.com/hajimehoshi/ebiten/v2"
)

func TestMapLegendExplainsEveryRenderedTileClass(t *testing.T) {
	entries := NewMapScene().mapLegendEntries(0.4)
	if len(entries) != int(gameapi.BiomeCount)+3 {
		t.Fatalf("legend entries = %d, want %d", len(entries), gameapi.BiomeCount+3)
	}
	seen := map[string]bool{}
	for _, entry := range entries {
		if entry.label == "" || entry.meaning == "" {
			t.Fatalf("incomplete legend entry: %#v", entry)
		}
		if seen[entry.label] {
			t.Fatalf("duplicate legend label %q", entry.label)
		}
		seen[entry.label] = true
	}
	for _, label := range []string{"Riverine woodland", "Savanna", "Coastal shrubland", "Mountain highlands", "Semi-arid desert", "Glacial tundra", "Water", "Unknown", "Escarpment"} {
		if !seen[label] {
			t.Fatalf("legend is missing %q", label)
		}
	}
	if !entries[8].edge {
		t.Fatal("escarpment legend entry is not presented as an edge")
	}
}

func TestLegendUsesStableBiomeIdentityAndTheLiveAtmosphericWaterGrade(t *testing.T) {
	scene := NewMapScene()
	humid := scene.mapLegendEntries(0)
	arid := scene.mapLegendEntries(1)
	for biome := gameapi.Biome(0); biome < gameapi.BiomeCount; biome++ {
		if humid[biome].color != climateBiomeColor(biome, 0) || arid[biome].color != climateBiomeColor(biome, 1) {
			t.Fatalf("legend swatch for %s diverges from map palette", biome)
		}
		if humid[biome].color != arid[biome].color {
			t.Fatalf("atmospheric grade changed %s's biome identity", biome)
		}
	}
	if humid[6].color == arid[6].color || humid[7].color != arid[7].color || humid[8].color != arid[8].color {
		t.Fatal("water did not follow the grade, or fixed overlay colors changed")
	}
}

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

// The legend teaches the map's glyph vocabulary, so it must not show a glyph
// the map never draws. Water is deliberately glyph-less: drawBiomeGlyphs skips
// every !tile.Land tile, and water is already unambiguous from colour and
// coastline shape without a redundant channel.
func TestWaterLegendEntryHasNoGlyph(t *testing.T) {
	scene := NewMapScene()
	for _, entry := range scene.mapLegendEntries(0.4) {
		if entry.label == "Water" && entry.glyph != nil {
			t.Error("water legend entry carries a glyph, but the map never draws one")
		}
	}
}

// An 8x8 swatch cannot host a legible pictograph. This pins the enlarged
// geometry — including the y offsets that keep the swatch, the meaning text,
// and the row bottom from colliding — so a later tidy-up cannot silently
// shrink the swatch or let it collide with a neighbor again.
func TestLegendSwatchIsLargeEnoughForAGlyph(t *testing.T) {
	if legendSwatchSize < 12 {
		t.Errorf("legend swatch = %v, too small to host a glyph", legendSwatchSize)
	}
	if legendLabelX <= legendSwatchSize+legendSwatchX {
		t.Errorf("legend label at %v overlaps a %v swatch at x+%v", legendLabelX, legendSwatchSize, legendSwatchX)
	}
	swatchBottom := legendSwatchY + legendSwatchSize
	if swatchBottom >= legendMeaningY {
		t.Errorf("swatch bottom %v (y=%v + size %v) collides with meaning text at y=%v", swatchBottom, legendSwatchY, legendSwatchSize, legendMeaningY)
	}
	meaningBottom := legendMeaningY + legendMeaningSize*textLineSpacing
	if meaningBottom > mapLegendHeight {
		t.Errorf("meaning text box bottom %v (y=%v + %v*%v line spacing) overflows the %v row", meaningBottom, legendMeaningY, legendMeaningSize, textLineSpacing, mapLegendHeight)
	}
}

// The legend swatch is the smallest box a pictograph is asked to sit in, so it
// is where an ink extent larger than the em size shows first. At the original
// 11 DIP em every biome crossed the swatch's top edge and savanna and mountain
// highlands crossed both sides, painting over the 0.7 stroke and into the
// label gutter -- and nothing caught it, because the only glyph assertions in
// the package counted pixels rather than locating them.
//
// This paints at the exact coordinates drawMapLegend uses for the first entry
// (absolute position matters: the baseline is quantized to a whole physical
// pixel) and asserts every inked pixel lands inside the swatch, at both device
// scales the reference browsers run at.
func TestLegendGlyphInkStaysInsideItsSwatch(t *testing.T) {
	scene := NewMapScene()
	// Wide and tall enough that ink escaping the swatch is recorded rather
	// than clipped: the swatch sits at x 24-36, y 50-62 for the first entry.
	const canvasWidth, canvasHeight = 96, 96
	swatchLeft := float32(mapOriginX) + legendSwatchX
	swatchTop := float32(mapLegendOriginY) + legendSwatchY
	for _, scale := range []float64{1, 2} {
		for _, entry := range scene.mapLegendEntries(0.4) {
			if entry.glyph == nil {
				continue
			}
			target := ebiten.NewImage(int(canvasWidth*scale), int(canvasHeight*scale))
			target.Fill(entry.color)
			entry.glyph.paint(
				newLogicalCanvas(target, scale),
				swatchLeft+legendSwatchSize/2, swatchTop+legendSwatchSize/2,
				legendGlyphSize, glyphInk(entry.color), 1,
			)
			minX, minY, maxX, maxY, inked := canvasWidth*scale, canvasHeight*scale, -1.0, -1.0, 0
			for y := range int(canvasHeight * scale) {
				for x := range int(canvasWidth * scale) {
					r, g, b, _ := target.At(x, y).RGBA()
					if r>>8 == uint32(entry.color.R) && g>>8 == uint32(entry.color.G) && b>>8 == uint32(entry.color.B) {
						continue
					}
					inked++
					// A pixel covers [i, i+1) physical px, so its DIP extent
					// is [i/scale, (i+1)/scale).
					minX, minY = min(minX, float64(x)/scale), min(minY, float64(y)/scale)
					maxX, maxY = max(maxX, float64(x+1)/scale), max(maxY, float64(y+1)/scale)
				}
			}
			target.Deallocate()
			if inked == 0 {
				t.Fatalf("scale %v: %s legend glyph painted nothing", scale, entry.label)
			}
			t.Logf("scale %v %-20s ink x=[%.2f..%.2f] y=[%.2f..%.2f] over swatch x=[%.0f..%.0f] y=[%.0f..%.0f]",
				scale, entry.label, minX, maxX, minY, maxY,
				swatchLeft, swatchLeft+legendSwatchSize, swatchTop, swatchTop+legendSwatchSize)
			if minX < float64(swatchLeft) || maxX > float64(swatchLeft+legendSwatchSize) ||
				minY < float64(swatchTop) || maxY > float64(swatchTop+legendSwatchSize) {
				t.Errorf("scale %v: %s legend glyph ink x=[%.2f..%.2f] y=[%.2f..%.2f] escapes the swatch x=[%.0f..%.0f] y=[%.0f..%.0f]",
					scale, entry.label, minX, maxX, minY, maxY,
					swatchLeft, swatchLeft+legendSwatchSize, swatchTop, swatchTop+legendSwatchSize)
			}
		}
	}
}
