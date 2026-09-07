package domain

import (
	"fmt"
	"math"
	"reflect"
	"testing"
)

func TestNewWorldInitializer(t *testing.T) {
	world, err := NewWorld(17)
	if err != nil {
		t.Fatal(err)
	}
	if world.Turn() != 0 || world.Result() != CampaignOngoing {
		t.Fatalf("new world state: turn=%d result=%d", world.Turn(), world.Result())
	}
	bands := world.Bands()
	if len(bands) != len(StartingAnchors) {
		t.Fatalf("bands = %d", len(bands))
	}
	var sapiens, archaic uint64
	for index, band := range bands {
		if band.ID != BandID(index+1) || band.TileID != StartingTileIDs[index] || band.Population != StartingAnchors[index].Population || band.Health != 1 || band.StoredFood != 0 {
			t.Fatalf("band %d: %#v", index, band)
		}
		if band.Species == HomoSapiens {
			sapiens += uint64(band.Population)
		} else {
			archaic += uint64(band.Population)
		}
	}
	if sapiens != 480 || archaic != 354 || world.nextBandID != BandID(len(StartingAnchors)+1) {
		t.Fatalf("scenario totals sapiens=%v archaic=%v next=%d", sapiens, archaic, world.nextBandID)
	}
}

func TestStateRoundTripIsolatedAndContinuesRNG(t *testing.T) {
	world, _ := NewWorld(42)
	state, err := world.ExportState()
	if err != nil {
		t.Fatal(err)
	}
	restored, err := RestoreWorld(state)
	if err != nil {
		t.Fatal(err)
	}
	restoredState, err := restored.ExportState()
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(state, restoredState) {
		t.Fatal("state changed across restore")
	}
	state.Bands[0].Population = 999
	state.RNGState[0] ^= 0xff
	if world.Bands()[0].Population == 999 {
		t.Fatal("export aliases live bands")
	}
	nextOriginal, nextRestored := world.rng.Uint64(), restored.rng.Uint64()
	if nextOriginal != nextRestored {
		t.Fatalf("RNG continuation differs: %d != %d", nextOriginal, nextRestored)
	}
}

func TestWorldsReuseCanonicalSeedIndependentGrid(t *testing.T) {
	first, err := NewWorld(1)
	if err != nil {
		t.Fatal(err)
	}
	second, err := NewWorld(2)
	if err != nil {
		t.Fatal(err)
	}
	state, err := first.ExportState()
	if err != nil {
		t.Fatal(err)
	}
	restored, err := RestoreWorld(state)
	if err != nil {
		t.Fatal(err)
	}
	if first.grid != second.grid || first.grid != restored.grid {
		t.Fatal("world construction rebuilt seed-independent geography")
	}
	if first.habitat == second.habitat || first.rng == second.rng {
		t.Fatal("canonical grid reuse aliased per-world state")
	}
}

func TestInitialExplorationIsSapiensOnly(t *testing.T) {
	world, _ := NewWorld(0)
	for _, band := range world.Bands() {
		if band.Species == ArchaicHominin && world.IsExplored(band.TileID) {
			t.Fatalf("archaic tile %d was revealed", band.TileID)
		}
		if band.Species == HomoSapiens && !world.IsExplored(band.TileID) {
			t.Fatalf("sapiens tile %d is hidden", band.TileID)
		}
	}
}

func TestSplitAppendsBoundedPersistentEvent(t *testing.T) {
	world, err := NewWorld(18)
	if err != nil {
		t.Fatal(err)
	}
	world.bands[0].Population = 10_000
	candidates := world.MigrationCandidates(world.bands[0].ID)
	var destination TileID
	found := false
	for _, candidate := range candidates {
		if !candidate.RequiresPassage {
			destination, found = candidate.TileID, true
			break
		}
	}
	if !found {
		t.Fatal("starting band has no ordinary split destination")
	}
	if err := world.Split(world.bands[0].ID, destination, true); err != nil {
		t.Fatal(err)
	}
	if len(world.events) != 1 || world.events[0].Kind != EventSplit || world.events[0].BandID != world.bands[len(world.bands)-1].ID {
		t.Fatalf("split events = %#v", world.events)
	}
	state, err := world.ExportState()
	if err != nil {
		t.Fatal(err)
	}
	restored, err := RestoreWorld(state)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(world.events, restored.events) {
		t.Fatalf("restored events = %#v, want %#v", restored.events, world.events)
	}
}

func TestCompletedMigrationAppendsHistoricalEvent(t *testing.T) {
	world, err := NewWorld(20)
	if err != nil {
		t.Fatal(err)
	}
	band := world.bands[0]
	var destination TileID
	found := false
	for _, candidate := range world.MigrationCandidates(band.ID) {
		if world.IsExplored(candidate.TileID) {
			destination, found = candidate.TileID, true
			break
		}
	}
	if !found {
		t.Fatal("starting band has no explored migration destination")
	}
	if err := world.QueueMigration(band.ID, destination, true); err != nil {
		t.Fatal(err)
	}
	if err := world.AdvanceTurn(); err != nil {
		t.Fatal(err)
	}
	found = false
	for _, event := range world.Events() {
		if event.Kind == EventMigration && event.BandID == band.ID && event.TileID == destination && event.Turn == 1 {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("migration event missing from %#v", world.Events())
	}
}

func TestEventFeedDropsItsOldestEntryAtCapacity(t *testing.T) {
	world, _ := NewWorld(21)
	geography, _ := world.grid.Tile(world.bands[0].TileID)
	for index := 0; index <= MaxEvents; index++ {
		world.appendEvent(Event{Turn: 0, Kind: EventAchievement, BandID: world.bands[0].ID, TileID: world.bands[0].TileID, Region: geography.Region, Summary: fmt.Sprintf("event %d", index)})
	}
	if len(world.events) != MaxEvents || world.events[0].Summary != "event 1" || world.events[MaxEvents-1].Summary != fmt.Sprintf("event %d", MaxEvents) {
		t.Fatalf("bounded event feed = first %q last %q length %d", world.events[0].Summary, world.events[MaxEvents-1].Summary, len(world.events))
	}
	if err := world.validate(); err != nil {
		t.Fatal(err)
	}
}

func TestRestoreRejectsContradictoryTerminalState(t *testing.T) {
	world, _ := NewWorld(19)
	state, _ := world.ExportState()
	state.Turn = MaxCampaignTurn
	state.Result = CampaignOngoing
	if _, err := RestoreWorld(state); err == nil {
		t.Fatal("turn-400 ongoing state restored")
	}

	state, _ = world.ExportState()
	state.Result = CampaignVictory
	if _, err := RestoreWorld(state); err == nil {
		t.Fatal("pre-terminal victory restored")
	}

	state, _ = world.ExportState()
	state.Bands = nil
	state.Result = CampaignOngoing
	if _, err := RestoreWorld(state); err == nil {
		t.Fatal("ongoing extinct state restored")
	}
}

func TestRestoreRejectsLivingBandOnWater(t *testing.T) {
	world, _ := NewWorld(23)
	state, _ := world.ExportState()
	for id := range TileCount {
		geography, _ := world.grid.Tile(TileID(id))
		if !geography.Land {
			state.Bands[0].TileID = TileID(id)
			if _, err := RestoreWorld(state); err == nil {
				t.Fatal("living band on water restored")
			}
			return
		}
	}
	t.Fatal("fixture has no water tile")
}

func TestRestoreRejectsInfiniteFoodReport(t *testing.T) {
	world, _ := NewWorld(24)
	if err := world.AdvanceTurn(); err != nil {
		t.Fatal(err)
	}
	positiveInfinity := math.MaxFloat64
	positiveInfinity += math.MaxFloat64
	state, _ := world.ExportState()
	state.Bands[0].LastFoodReport.RequiredFU = positiveInfinity
	if _, err := RestoreWorld(state); err == nil {
		t.Fatal("infinite required food restored")
	}
	state, _ = world.ExportState()
	state.Bands[0].LastFoodReport.DeficitFU = positiveInfinity
	if _, err := RestoreWorld(state); err == nil {
		t.Fatal("infinite food deficit restored")
	}
}
