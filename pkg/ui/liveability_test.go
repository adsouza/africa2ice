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
	byLabel := rowsByLabel(band, here, target)
	if len(byLabel) != 8 {
		t.Fatalf("row count = %d, want 8", len(byLabel))
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
	// Neighbours are reported, never tiered; see TestOthersRowIsInformationalNotAWarning.
	if others := byLabel["Others"]; others.TargetTier != TierNormal || others.Target != "1 archaic · pop 40" || others.Here != "none" {
		t.Fatalf("others row = %+v", others)
	}
	if route := byLabel["Route"]; route.Target != "×1.20 · 1 turn" || route.Here != "—" {
		t.Fatalf("route row = %+v", route)
	}
}

func TestLiveabilityTiersUseTheSpecThresholds(t *testing.T) {
	frame := liveabilityFrame()
	band := &frame.Bands[0]
	frame.Tiles[0].WaterStock = 120 // 0.24 of cap
	// 30% degraded, so capacity is 105 of a 150 baseline: occupancy is
	// 68/105 = 0.65, just under the crowding threshold, which leaves
	// degradation alone to drive the amber tier.
	frame.Tiles[0].Degradation, frame.Tiles[0].EcologicalK = 0.3, 105
	band.SeasonalMortalityRate, band.ChronicMortalityRate = 0.005, 0.004
	frame.Tiles[0].NaturalShelter = 0.2
	byLabel := rowsByLabel(band, CurrentTileLiveability(frame, band), TileLiveability{})
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

	mortality := rowsByLabel(band, CurrentTileLiveability(frame, band), TargetTileLiveability(frame, band, 1))["Mortality"]
	// Delta should be +1 (better) because seasonal+chronic rates are lower, ignoring crowding
	if mortality.Delta != 1 {
		t.Fatalf("mortality delta = %d, want 1 (target rates are lower despite high crowding)", mortality.Delta)
	}
	// Target string should show crowding
	if !strings.Contains(mortality.Target, "crowd −12") {
		t.Fatalf("mortality target = %q, want crowd −12", mortality.Target)
	}
}

// rowsByLabel indexes LiveabilityRows by its Label for assertions.
func rowsByLabel(band *gameapi.Band, here, target TileLiveability) map[string]LiveabilityRow {
	byLabel := map[string]LiveabilityRow{}
	for _, row := range LiveabilityRows(band, here, target) {
		byLabel[row.Label] = row
	}
	return byLabel
}

// A ▲▼ mark is a claim about the two numbers the player can see. Comparing raw
// metrics behind a rounded formatter marked differences that never reach the
// screen: 211.6 and 212.4 both render "212 / 810" yet earned a red ▼.
func TestDeltaIgnoresDifferencesTooSmallToDisplay(t *testing.T) {
	frame := liveabilityFrame()
	band := &frame.Bands[0]
	frame.Tiles[0].FloraStock, frame.Tiles[1].FloraStock = 211.6, 212.4
	food := rowsByLabel(band, CurrentTileLiveability(frame, band), TargetTileLiveability(frame, band, 1))["Food"]
	if food.Here != food.Target {
		t.Fatalf("fixture no longer renders both sides identically: here %q target %q", food.Here, food.Target)
	}
	if food.Delta != 0 {
		t.Fatalf("food delta = %d for identical displayed values %q, want 0", food.Delta, food.Here)
	}
}

// The Capacity tier asks whether the tile could carry this band once it
// arrives, which is how the domain already defines crowding: World.BandStress
// divides a tile's *total* resident population by its capacity. Tiering bare
// degradation instead flagged a smaller-but-ample target red (user-reported).
func TestCapacityTierMeasuresOccupancyAfterTheBandArrives(t *testing.T) {
	frame := liveabilityFrame()
	band := &frame.Bands[0] // population 68
	// Tile 1 holds an archaic band of 40 and has capacity 150, so arriving
	// makes 108/150 = 0.72 — past the 0.67 split-stress threshold — while
	// tile 0 carries this band alone at 68/150 = 0.45.
	capacity := rowsByLabel(band, CurrentTileLiveability(frame, band), TargetTileLiveability(frame, band, 1))["Capacity"]
	if capacity.HereTier != TierNormal || capacity.TargetTier != TierAmber {
		t.Fatalf("capacity tiers = here %v target %v, want normal then amber: %+v", capacity.HereTier, capacity.TargetTier, capacity)
	}
	if capacity.Here != "150 · 45% full" || capacity.Target != "150 · 72% full" {
		t.Fatalf("capacity values = here %q target %q", capacity.Here, capacity.Target)
	}
	// Occupancy drives the mark, not raw capacity: both tiles have K 150, so a
	// comparison on EcologicalK reports no difference at all.
	if capacity.Delta != -1 || !capacity.DeltaMaterial {
		t.Fatalf("capacity delta = %d material %v, want -1 and material (the target would be crowded)", capacity.Delta, capacity.DeltaMaterial)
	}
}

// The reported symptom: a target tile with lower capacity was coloured red
// even though the band's population fitted it several times over. The mark
// still records that the target is tighter; only the alarm goes away.
func TestLowerCapacityTheBandComfortablyFitsIsNotAWarning(t *testing.T) {
	frame := liveabilityFrame()
	band := &frame.Bands[0] // population 68
	frame.Tiles[0].EcologicalK, frame.Tiles[1].EcologicalK = 400, 200
	capacity := rowsByLabel(band, CurrentTileLiveability(frame, band), TargetTileLiveability(frame, band, 1))["Capacity"]
	// 68/400 = 17% against (68+40)/200 = 54%: tighter, but neither side is
	// near the 67% crowding point.
	if capacity.HereTier != TierNormal || capacity.TargetTier != TierNormal {
		t.Fatalf("capacity tiers = here %v target %v, want both normal: %+v", capacity.HereTier, capacity.TargetTier, capacity)
	}
	if capacity.Delta != -1 {
		t.Fatalf("capacity delta = %d, want -1: the target really is tighter", capacity.Delta)
	}
	if capacity.DeltaMaterial {
		t.Fatal("capacity delta is material between two tiles the band fits comfortably")
	}
}

// A tile with less capacity than the arriving band needs is red, not amber:
// the band cannot be carried there at all.
func TestCapacityIsRedWhenTheBandWouldNotFit(t *testing.T) {
	frame := liveabilityFrame()
	band := &frame.Bands[0] // population 68, joining an archaic band of 40
	frame.Tiles[1].EcologicalK = 100
	capacity := rowsByLabel(band, CurrentTileLiveability(frame, band), TargetTileLiveability(frame, band, 1))["Capacity"]
	if capacity.TargetTier != TierRed {
		t.Fatalf("capacity target tier = %v for 108 people on a capacity of 100, want red: %+v", capacity.TargetTier, capacity)
	}
	if capacity.Target != "100 · 108% full" {
		t.Fatalf("capacity target = %q, want an occupancy over 100%%", capacity.Target)
	}
	if !capacity.DeltaMaterial {
		t.Fatal("capacity delta onto an over-full tile is not material")
	}
}

// An archaic neighbour is the one opportunity this row exists to advertise —
// it is the precondition for interbreeding — so it must not be coloured as a
// hazard (user-reported). The pressure a neighbour does create now lives on
// Capacity, where the occupancy figure causing it is visible.
func TestOthersRowIsInformationalNotAWarning(t *testing.T) {
	frame := liveabilityFrame()
	band := &frame.Bands[0]
	others := rowsByLabel(band, CurrentTileLiveability(frame, band), TargetTileLiveability(frame, band, 1))["Others"]
	if others.HereTier != TierNormal || others.TargetTier != TierNormal {
		t.Fatalf("others tiers = here %v target %v, want both normal: %+v", others.HereTier, others.TargetTier, others)
	}
	// Naming the species keeps the interbreeding cue the amber tier used to carry.
	if others.Here != "none" || others.Target != "1 archaic · pop 40" {
		t.Fatalf("others values = here %q target %q", others.Here, others.Target)
	}
}

// A sapiens neighbour competes for the same food and water, so the row counts
// every species; the band being read for never counts itself.
func TestOthersRowCountsEverySpeciesButNotTheBandItself(t *testing.T) {
	frame := liveabilityFrame()
	frame.Bands = append(frame.Bands, gameapi.Band{ID: 11, Species: gameapi.HomoSapiens, Population: 25, TileID: 1})
	band := &frame.Bands[0] // taken after the append: appending can reallocate
	others := rowsByLabel(band, CurrentTileLiveability(frame, band), TargetTileLiveability(frame, band, 1))["Others"]
	if others.Target != "2 bands · pop 65" {
		t.Fatalf("mixed-species others = %q, want a species-neutral count of both", others.Target)
	}
	if others.Here != "none" {
		t.Fatalf("here others = %q, want none: the selected band must not count itself", others.Here)
	}
}

// A degraded tile names both its present and its baseline capacity, in the
// same "current / potential" idiom the Food and Water rows use. That states
// the loss exactly, where a third percentage on the line overflowed the
// column (TestMoveGridValuesFitTheirColumns).
func TestDegradedCapacityNamesItsBaseline(t *testing.T) {
	frame := liveabilityFrame()
	band := &frame.Bands[0] // population 68
	frame.Tiles[0].EcologicalK, frame.Tiles[0].Degradation = 75, 0.5
	byLabel := rowsByLabel(band, CurrentTileLiveability(frame, band), TargetTileLiveability(frame, band, 1))
	if capacity := byLabel["Capacity"]; capacity.Here != "75/150 · 91% full" {
		t.Fatalf("degraded capacity = %q, want its baseline alongside", capacity.Here)
	}
	// An undegraded tile stays on the short form.
	if capacity := byLabel["Capacity"]; capacity.Target != "150 · 72% full" {
		t.Fatalf("undegraded capacity = %q, want no baseline pair", capacity.Target)
	}
}
