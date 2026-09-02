package ui

import (
	"strings"
	"testing"

	"github.com/adsouza/africa2ice/pkg/gameapi"
	"github.com/adsouza/africa2ice/pkg/render"
)

func liveabilityFrame() *gameapi.Frame {
	tile := func(id gameapi.TileID, food, water float64) gameapi.Tile {
		return gameapi.Tile{
			ID: id, X: int(id), Y: 0, Land: true, Explored: true, Biome: gameapi.Savanna, Region: gameapi.EastAfrica,
			BaselineK: 150, EcologicalK: 150, FloraStock: food, FloraCap: 810, WaterStock: water, WaterCap: 500,
			NaturalShelter: 0.5, MovementCost: 1.2,
		}
	}
	return &gameapi.Frame{
		Tiles: []gameapi.Tile{tile(0, 212, 315), tile(1, 640, 480), {ID: 2, X: 2, Land: true, Explored: false}},
		Bands: []gameapi.Band{
			{
				ID: 3, Species: gameapi.HomoSapiens, Population: 68, TileID: 0, Health: 1,
				SeasonalMortalityRate: 0.0003, ChronicMortalityRate: 0.0016,
				LastFoodReport:      gameapi.FoodTurnReport{Turn: 12, RequiredFU: 300},
				MigrationCandidates: []gameapi.MigrationCandidate{{TileID: 1, SeasonalMortalityRate: 0.0003, ChronicMortalityRate: 0.002}},
			},
			{ID: 9, Species: gameapi.ArchaicHominin, Population: 40, TileID: 1},
		},
	}
}

func TestTargetTilePrecedenceIsCursorThenQueuedThenHover(t *testing.T) {
	band := &gameapi.Band{ID: 3, HasQueuedMigration: true, QueuedMigration: 5}
	preview := render.MigrationPreview{BandID: 3, TileID: 4, Visible: true}
	hover := render.TileHover{TileID: 6, Visible: true}
	if tile, source := TargetTile(band, preview, hover); tile != 4 || source != TargetCursor {
		t.Fatalf("cursor precedence = (%d, %v)", tile, source)
	}
	if tile, source := TargetTile(band, render.MigrationPreview{BandID: 8, TileID: 4, Visible: true}, hover); tile != 5 || source != TargetQueued {
		t.Fatalf("queued precedence = (%d, %v)", tile, source)
	}
	if tile, source := TargetTile(&gameapi.Band{ID: 3}, render.MigrationPreview{}, hover); tile != 6 || source != TargetHover {
		t.Fatalf("hover = (%d, %v)", tile, source)
	}
	if _, source := TargetTile(&gameapi.Band{ID: 3}, render.MigrationPreview{}, render.TileHover{}); source != TargetNone {
		t.Fatalf("no target = %v", source)
	}
	if TargetNone.Label() != "hover or click an outlined tile" || TargetCursor.Label() != "cursor" || TargetQueued.Label() != "queued" || TargetHover.Label() != "hover" {
		t.Fatal("target source labels drifted from the spec")
	}
}

func TestLiveabilityRowsColorAbsoluteStateAndMarkRelativeDelta(t *testing.T) {
	frame := liveabilityFrame()
	band := &frame.Bands[0]
	here := CurrentTileLiveability(frame, band)
	target := TargetTileLiveability(frame, band, 1)
	if !here.Available || !target.Available || !target.Reachable {
		t.Fatalf("summaries = here %+v target %+v", here, target)
	}
	rows := LiveabilityRows(band, here, target)
	byLabel := map[string]LiveabilityRow{}
	for _, row := range rows {
		byLabel[row.Label] = row
	}
	if len(rows) != 8 {
		t.Fatalf("row count = %d, want 8", len(rows))
	}
	// Food 212 is below last turn's 300 FU requirement: red here, normal there, target better.
	if food := byLabel["Food"]; food.HereTier != TierRed || food.TargetTier != TierNormal || food.Delta != 1 || food.Here != "212 / 810" {
		t.Fatalf("food row = %+v", food)
	}
	// Water 315/500 = 0.63 of cap: normal; 480 is better.
	if water := byLabel["Water"]; water.HereTier != TierNormal || water.Delta != 1 {
		t.Fatalf("water row = %+v", water)
	}
	// Chronic 0.16% -> 0.20%: worse, both below the 0.004 amber tier.
	if mortality := byLabel["Mortality"]; mortality.Delta != -1 || mortality.HereTier != TierNormal {
		t.Fatalf("mortality row = %+v", mortality)
	}
	if archaic := byLabel["Archaic"]; archaic.TargetTier != TierAmber || archaic.Target != "1 band · pop 40" || archaic.Here != "none" {
		t.Fatalf("archaic row = %+v", archaic)
	}
	if route := byLabel["Route"]; route.Target != "×1.20 · 1 turn" || route.Here != "—" {
		t.Fatalf("route row = %+v", route)
	}
}

func TestLiveabilityTiersUseTheSpecThresholds(t *testing.T) {
	frame := liveabilityFrame()
	band := &frame.Bands[0]
	frame.Tiles[0].WaterStock = 120 // 0.24 of cap
	frame.Tiles[0].Degradation = 0.3
	band.SeasonalMortalityRate, band.ChronicMortalityRate = 0.005, 0.004
	frame.Tiles[0].NaturalShelter = 0.2
	rows := LiveabilityRows(band, CurrentTileLiveability(frame, band), TileLiveability{})
	byLabel := map[string]LiveabilityRow{}
	for _, row := range rows {
		byLabel[row.Label] = row
	}
	if byLabel["Water"].HereTier != TierRed || byLabel["Capacity"].HereTier != TierAmber || byLabel["Mortality"].HereTier != TierRed || byLabel["Shelter"].HereTier != TierAmber {
		t.Fatalf("tiers = water %v capacity %v mortality %v shelter %v", byLabel["Water"].HereTier, byLabel["Capacity"].HereTier, byLabel["Mortality"].HereTier, byLabel["Shelter"].HereTier)
	}
	if byLabel["Biome"].Target != "—" {
		t.Fatalf("unavailable target biome = %q, want an em dash", byLabel["Biome"].Target)
	}
}

func TestTargetTileLiveabilityHidesFogAndExplainsUnreachable(t *testing.T) {
	frame := liveabilityFrame()
	band := &frame.Bands[0]
	if fog := TargetTileLiveability(frame, band, 2); fog.Available || fog.Status != "Unexplored · details hidden" {
		t.Fatalf("fog summary = %+v", fog)
	}
	frame.Tiles[1].Explored = true
	band.MigrationCandidates = nil
	if far := TargetTileLiveability(frame, band, 1); !far.Available || far.Reachable || far.Status != "not reachable" {
		t.Fatalf("unreachable summary = %+v", far)
	}
}

func TestMortalityDeltaIgnoresCrowdingHeadcount(t *testing.T) {
	frame := liveabilityFrame()
	band := &frame.Bands[0]
	// Set candidate rates much lower than band rates, with high crowding
	frame.Bands[0].MigrationCandidates[0].SeasonalMortalityRate = 0.0001
	frame.Bands[0].MigrationCandidates[0].ChronicMortalityRate = 0.0005
	frame.Bands[0].MigrationCandidates[0].CrowdingDecline = 12

	here := CurrentTileLiveability(frame, band)
	target := TargetTileLiveability(frame, band, 1)
	rows := LiveabilityRows(band, here, target)

	byLabel := map[string]LiveabilityRow{}
	for _, row := range rows {
		byLabel[row.Label] = row
	}

	mortality := byLabel["Mortality"]
	// Delta should be +1 (better) because seasonal+chronic rates are lower, ignoring crowding
	if mortality.Delta != 1 {
		t.Fatalf("mortality delta = %d, want 1 (target rates are lower despite high crowding)", mortality.Delta)
	}
	// Target string should show crowding
	if !strings.Contains(mortality.Target, "crowding −12") {
		t.Fatalf("mortality target = %q, want crowding −12", mortality.Target)
	}
}
