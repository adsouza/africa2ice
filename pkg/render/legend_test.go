package render

import (
	"testing"

	"github.com/adsouza/africa2ice/pkg/gameapi"
)

func TestMapLegendExplainsEveryRenderedTileClass(t *testing.T) {
	entries := mapLegendEntries(0.4)
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
	humid := mapLegendEntries(0)
	arid := mapLegendEntries(1)
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
