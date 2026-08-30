package render

import (
	"strings"
	"testing"

	"github.com/adsouza/africa2ice/pkg/gameapi"
)

func TestMapLegendExplainsEveryRenderedTileClass(t *testing.T) {
	entries := mapLegendEntries(0.4)
	if len(entries) != int(gameapi.BiomeCount)+2 {
		t.Fatalf("legend entries = %d, want %d", len(entries), gameapi.BiomeCount+2)
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
	for _, label := range []string{"Riverine woodland", "Savanna", "Coastal shrubland", "Mountain highlands", "Semi-arid desert", "Glacial tundra", "Water", "Unknown"} {
		if !seen[label] {
			t.Fatalf("legend is missing %q", label)
		}
	}
}

func TestLegendBiomeSwatchesFollowTheLiveClimatePalette(t *testing.T) {
	humid := mapLegendEntries(0)
	arid := mapLegendEntries(1)
	for biome := gameapi.Biome(0); biome < gameapi.BiomeCount; biome++ {
		if humid[biome].color != climateBiomeColor(biome, 0) || arid[biome].color != climateBiomeColor(biome, 1) {
			t.Fatalf("legend swatch for %s diverges from map palette", biome)
		}
	}
	if humid[6].color != arid[6].color || humid[7].color != arid[7].color {
		t.Fatal("water or unexplored swatch changed with aridity")
	}
}

func TestTileLiveabilityComparesCurrentAndArrowTarget(t *testing.T) {
	frame := tileInfoFixture()
	band := &frame.Bands[0]
	current := currentTileSummary(frame, band)
	target := targetTileSummary(frame, band, MigrationPreview{BandID: band.ID, TileID: 1, Visible: true})

	if !current.showDetails || current.foodStock != 100 || current.foodCap != 200 || current.seasonalRisk != 0.001 || current.chronicRisk != 0.002 {
		t.Fatalf("current summary = %#v", current)
	}
	if !target.showDetails || !strings.Contains(target.status, "reachable") || target.foodStock != 160 || target.waterStock != 90 {
		t.Fatalf("target summary = %#v", target)
	}
	if target.seasonalRisk != 0.003 || target.chronicRisk != 0.004 {
		t.Fatalf("target risks = %v/%v", target.seasonalRisk, target.chronicRisk)
	}
	lines := liveabilityLines(target)
	for _, expected := range []string{"Food stock 160/280 FU", "Water 90/120 WU", "Seasonal 0.30% · chronic 0.40%", "Shelter 40%", "travel ×1.25"} {
		if !strings.Contains(strings.Join(lines[:], "\n"), expected) {
			t.Fatalf("target details %q do not contain %q", lines, expected)
		}
	}
}

func TestTileLiveabilityDoesNotRevealUnexploredTerrain(t *testing.T) {
	frame := tileInfoFixture()
	band := &frame.Bands[0]
	summary := targetTileSummary(frame, band, MigrationPreview{BandID: band.ID, TileID: 2, Visible: true})
	if summary.showDetails || summary.biome != "" || summary.foodStock != 0 || summary.waterStock != 0 {
		t.Fatalf("unexplored summary leaked tile details: %#v", summary)
	}
	if lines := liveabilityLines(summary); !strings.Contains(lines[0], "details hidden") {
		t.Fatalf("unexplored explanation = %q", lines[0])
	}
}

func TestTileLiveabilityExplainsWaterAndFallsBackToQueuedTarget(t *testing.T) {
	frame := tileInfoFixture()
	band := &frame.Bands[0]
	water := targetTileSummary(frame, band, MigrationPreview{BandID: band.ID, TileID: 3, Visible: true})
	if water.showDetails || !strings.Contains(water.status, "cannot occupy") {
		t.Fatalf("water summary = %#v", water)
	}

	band.HasQueuedMigration, band.QueuedMigration = true, 1
	queued := targetTileSummary(frame, band, MigrationPreview{})
	if !queued.showDetails || !strings.Contains(queued.status, "queued") {
		t.Fatalf("queued summary = %#v", queued)
	}
}

func tileInfoFixture() *gameapi.Frame {
	return &gameapi.Frame{
		Tiles: []gameapi.Tile{
			{ID: 0, Land: true, Explored: true, Biome: gameapi.Savanna, Region: gameapi.EastAfrica, BaselineK: 100, EcologicalK: 90, FloraStock: 60, FloraCap: 120, FaunaStock: 40, FaunaCap: 80, WaterStock: 70, WaterCap: 100, LocalTemperatureC: 25, NaturalShelter: 0.2, MovementCost: 1},
			{ID: 1, Land: true, Explored: true, Biome: gameapi.CoastalShrubland, Region: gameapi.RestOfAfrica, BaselineK: 140, EcologicalK: 120, Degradation: 0.1, FloraStock: 70, FloraCap: 130, FaunaStock: 90, FaunaCap: 150, WaterStock: 90, WaterCap: 120, LocalTemperatureC: 23, NaturalShelter: 0.4, MovementCost: 1.25},
			{ID: 2, Land: true, Explored: false, Biome: gameapi.GlacialTundra, Region: gameapi.Beringia, BaselineK: 999, FloraStock: 999, WaterStock: 999},
			{ID: 3, Land: false, Explored: true},
		},
		Bands: []gameapi.Band{{
			ID: 7, Species: gameapi.HomoSapiens, TileID: 0,
			SeasonalMortalityRate: 0.001, ChronicMortalityRate: 0.002,
			MigrationCandidates: []gameapi.MigrationCandidate{{TileID: 1, SeasonalMortalityRate: 0.003, ChronicMortalityRate: 0.004}},
		}},
	}
}

// The risk line named seasonal and chronic — the two smallest contributors to
// population loss — and said nothing about crowding, which dominates whenever a
// band moves onto a tile too small for it. In a traced case a band of 112 lost
// 99 people in one turn with seasonal at 0.05% and chronic at 0.06%. The player
// could read the target's capacity but was never told what taking this band
// there would cost.
func TestLiveabilityRiskLineNamesTheDominantCause(t *testing.T) {
	summary := tileLiveabilitySummary{
		heading: "TARGET", status: "arrow cursor · reachable", showDetails: true,
		biome: "Savanna", region: "East Africa", ecologicalK: 11,
		seasonalRisk: 0.0005, chronicRisk: 0.0006, hasRisk: true,
		crowdingDecline: 28,
	}
	line := liveabilityLines(summary)[6]
	if !strings.Contains(strings.ToLower(line), "crowding") {
		t.Fatalf("risk line = %q, want the dominant cause named", line)
	}
	if !strings.Contains(line, "28") {
		t.Fatalf("risk line = %q, want the projected loss in people", line)
	}

	// With room at the destination the line must stay exactly as it was, so the
	// warning carries meaning by its absence too.
	summary.crowdingDecline = 0
	if line := liveabilityLines(summary)[6]; line != "Seasonal 0.05% · chronic 0.06%" {
		t.Fatalf("uncrowded risk line = %q", line)
	}
}
