package render

import (
	"strings"
	"testing"

	"github.com/adsouza/africa2ice/pkg/gameapi"
	"github.com/hajimehoshi/ebiten/v2/text/v2"
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

func TestTileLiveabilityComparesCurrentAndArrowTarget(t *testing.T) {
	frame := tileInfoFixture()
	band := &frame.Bands[0]
	current := currentTileSummary(frame, band)
	target := targetTileSummary(frame, band, MigrationPreview{BandID: band.ID, TileID: 1, Visible: true}, TileHover{})

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
	for _, expected := range []string{"1.0 km", "Food stock 160/280 FU", "Water 90/120 WU", "Capacity 120/140", "Seasonal 0.30% · chronic 0.40%", "Shelter 40%", "travel ×1.25"} {
		if !strings.Contains(strings.Join(lines[:], "\n"), expected) {
			t.Fatalf("target details %q do not contain %q", lines, expected)
		}
	}
}

func TestTileLiveabilityDoesNotRevealUnexploredTerrain(t *testing.T) {
	frame := tileInfoFixture()
	band := &frame.Bands[0]
	summary := targetTileSummary(frame, band, MigrationPreview{BandID: band.ID, TileID: 2, Visible: true}, TileHover{})
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
	water := targetTileSummary(frame, band, MigrationPreview{BandID: band.ID, TileID: 3, Visible: true}, TileHover{})
	if water.showDetails || !strings.Contains(water.status, "cannot occupy") {
		t.Fatalf("water summary = %#v", water)
	}

	band.HasQueuedMigration, band.QueuedMigration = true, 1
	queued := targetTileSummary(frame, band, MigrationPreview{}, TileHover{})
	if !queued.showDetails || !strings.Contains(queued.status, "queued") {
		t.Fatalf("queued summary = %#v", queued)
	}
}

func TestTileHoverFillsTheInspectorWithoutReplacingExplicitMigrationIntent(t *testing.T) {
	frame := tileInfoFixture()
	band := &frame.Bands[0]
	hover := TileHover{TileID: 1, Visible: true}

	hovered := targetTileSummary(frame, band, MigrationPreview{}, hover)
	if !hovered.showDetails || !strings.Contains(hovered.status, "pointer hover · reachable") {
		t.Fatalf("hover summary = %#v", hovered)
	}

	band.HasQueuedMigration, band.QueuedMigration = true, 3
	queued := targetTileSummary(frame, band, MigrationPreview{}, hover)
	if !strings.Contains(queued.status, "cannot occupy") || strings.Contains(queued.status, "pointer hover") {
		t.Fatalf("queued migration did not override hover: %#v", queued)
	}

	preview := MigrationPreview{BandID: band.ID, TileID: 1, Visible: true}
	arrow := targetTileSummary(frame, band, preview, hover)
	if !strings.Contains(arrow.status, "arrow cursor") {
		t.Fatalf("arrow cursor did not override queued migration and hover: %#v", arrow)
	}
}

func TestTileLiveabilityCallsOutArchaicBandsWithoutLeakingThroughFog(t *testing.T) {
	frame := tileInfoFixture()
	frame.Bands = append(frame.Bands,
		gameapi.Band{ID: 8, Species: gameapi.ArchaicHominin, TileID: 0, Population: 60},
		gameapi.Band{ID: 9, Species: gameapi.ArchaicHominin, TileID: 1, Population: 75},
		gameapi.Band{ID: 10, Species: gameapi.ArchaicHominin, TileID: 1, Population: 90},
		gameapi.Band{ID: 11, Species: gameapi.ArchaicHominin, TileID: 2, Population: 150},
		gameapi.Band{ID: 12, Species: gameapi.ArchaicHominin, TileID: 1, Population: 0},
	)
	band := &frame.Bands[0]

	current := currentTileSummary(frame, band)
	if current.archaicBandCount != 1 || current.archaicPopulation != 60 {
		t.Fatalf("current archaic presence = %d bands/%d population", current.archaicBandCount, current.archaicPopulation)
	}
	if line := liveabilityLines(current)[archaicPresenceLineIndex]; line != "Archaic hominins: 1 band · pop 60" {
		t.Fatalf("current archaic line = %q", line)
	}

	target := targetTileSummary(frame, band, MigrationPreview{BandID: band.ID, TileID: 1, Visible: true}, TileHover{})
	if target.archaicBandCount != 2 || target.archaicPopulation != 165 {
		t.Fatalf("target archaic presence = %d bands/%d population", target.archaicBandCount, target.archaicPopulation)
	}
	if line := liveabilityLines(target)[archaicPresenceLineIndex]; line != "Archaic hominins: 2 bands · pop 165" {
		t.Fatalf("target archaic line = %q", line)
	} else {
		face := &text.GoTextFace{Source: NewMapScene().faceSource, Size: tileInspectorTextSize}
		width, height := text.Measure(line, face, 0)
		if width > 151 {
			t.Fatalf("target archaic line width = %.1fpx, want at most 151px", width)
		}
		lastLineY := float64(tileInspectorOriginY+27) + float64(len(liveabilityLines(target))-1)*tileInspectorRowGap
		if lastLineY+height > interbreedPanelLineY {
			t.Fatalf("tile details end at %.1fpx, overlapping the footer at %dpx", lastLineY+height, interbreedPanelLineY)
		}
	}

	hidden := targetTileSummary(frame, band, MigrationPreview{BandID: band.ID, TileID: 2, Visible: true}, TileHover{})
	if hidden.archaicBandCount != 0 || hidden.archaicPopulation != 0 {
		t.Fatalf("unexplored tile leaked archaic presence: %#v", hidden)
	}
}

func tileInfoFixture() *gameapi.Frame {
	return &gameapi.Frame{
		Tiles: []gameapi.Tile{
			{ID: 0, Land: true, Explored: true, Biome: gameapi.Savanna, Region: gameapi.EastAfrica, BaselineK: 100, EcologicalK: 90, FloraStock: 60, FloraCap: 120, FaunaStock: 40, FaunaCap: 80, WaterStock: 70, WaterCap: 100, LocalTemperatureC: 25, ElevationKm: 0.2, NaturalShelter: 0.2, MovementCost: 1},
			{ID: 1, Land: true, Explored: true, Biome: gameapi.CoastalShrubland, Region: gameapi.RestOfAfrica, BaselineK: 140, EcologicalK: 120, Degradation: 0.1, FloraStock: 70, FloraCap: 130, FaunaStock: 90, FaunaCap: 150, WaterStock: 90, WaterCap: 120, LocalTemperatureC: 23, ElevationKm: 1.0, NaturalShelter: 0.4, MovementCost: 1.25},
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
	line := liveabilityLines(summary)[7]
	if !strings.Contains(strings.ToLower(line), "crowding") {
		t.Fatalf("risk line = %q, want the dominant cause named", line)
	}
	if !strings.Contains(line, "28") {
		t.Fatalf("risk line = %q, want the projected loss in people", line)
	}

	// With room at the destination the line must stay exactly as it was, so the
	// warning carries meaning by its absence too.
	summary.crowdingDecline = 0
	if line := liveabilityLines(summary)[7]; line != "Seasonal 0.05% · chronic 0.06%" {
		t.Fatalf("uncrowded risk line = %q", line)
	}
}
