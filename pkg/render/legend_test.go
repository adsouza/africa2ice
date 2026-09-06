package render

import (
	"testing"

	"github.com/adsouza/africa2ice/pkg/gameapi"
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
	// Six biomes plus water; unknown and escarpment stay bare.
	if withGlyph != int(gameapi.BiomeCount)+1 {
		t.Errorf("%d legend entries carry a glyph, want %d", withGlyph, gameapi.BiomeCount+1)
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
	meaningBottom := legendMeaningY + 7*textLineSpacing
	if meaningBottom > mapLegendHeight {
		t.Errorf("meaning text box bottom %v (y=%v + 7*%v line spacing) overflows the %v row", meaningBottom, legendMeaningY, textLineSpacing, mapLegendHeight)
	}
}
