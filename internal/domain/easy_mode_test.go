package domain

import (
	"errors"
	"reflect"
	"testing"
)

func TestEasyModeExplorationAndSplit(t *testing.T) {
	world, _ := NewWorld(17)
	world.bands = world.bands[:1]
	world.exploredTiles = [ExplorationWordCount]uint64{}
	world.revealFromSapiens()
	x, y, _ := TileXY(world.bands[0].TileID)
	outer, _ := TileIDAt(x+2, y)
	if world.IsExplored(outer) {
		t.Fatal("normal mode revealed second ring")
	}
	world.SetEasyMode(true)
	if !world.IsExplored(outer) {
		t.Fatal("easy mode did not reveal second ring immediately")
	}
	world.SetEasyMode(false)
	if !world.IsExplored(outer) {
		t.Fatal("disabling easy mode erased exploration")
	}
	band := &world.bands[0]
	band.Population = 20
	destination := world.grid.OrdinaryEdges(band.TileID)[0].To
	world.markExplored(destination)
	if err := world.Split(band.ID, destination, true); !errors.Is(err, ErrSplitStressTooLow) {
		t.Fatalf("normal split: %v", err)
	}
	world.SetEasyMode(true)
	band.Population = 19
	if err := world.Split(band.ID, destination, true); !errors.Is(err, ErrSplitPopulationTooLow) {
		t.Fatalf("small split: %v", err)
	}
	band.Population = 20
	if err := world.Split(band.ID, destination, true); err != nil {
		t.Fatal(err)
	}
	if world.bands[0].Population != 10 || world.bands[1].Population != 10 {
		t.Fatal("split lost people")
	}
	if err := world.Split(world.bands[0].ID, destination, true); !errors.Is(err, ErrSpatialActionUsed) {
		t.Fatalf("easy mode bypassed action limit: %v", err)
	}
}

func TestEasyModeBoundsAllTurnLossesAndDisablesAcuteIncidents(t *testing.T) {
	for _, population := range []Population{1, 9, 19, 100, 10000} {
		world, _ := NewWorld(42)
		world.bands = world.bands[:1]
		world.bands[0].Population = population
		world.bands[0].Allocation = [AssignmentCount]AssignmentBP{0, 0, 10000, 0, 0}
		world.SetEasyMode(true)
		for turn := 0; turn < 30; turn++ {
			before := world.bands[0].Population
			if err := world.AdvanceTurn(); err != nil {
				t.Fatal(err)
			}
			if len(world.bands) != 1 || world.bands[0].Population < before-before/10 {
				t.Fatalf("population %d breached integer 10%% cap: %+v", before, world.bands)
			}
			band := world.bands[0]
			if band.LastMortality.Acute != 0 || band.LastOutcomeReport.AcuteDiseaseHealthLoss != 0 {
				t.Fatal("acute incident occurred")
			}
		}
		for _, event := range world.Events() {
			if event.Kind == EventAcuteIncident {
				t.Fatal("acute incident event emitted")
			}
		}
	}
}

func TestEasyModeSaveContinuesDeterministically(t *testing.T) {
	world, _ := NewWorld(42)
	world.SetEasyMode(true)
	if err := world.AdvanceTurn(); err != nil {
		t.Fatal(err)
	}
	state, _ := world.ExportState()
	restored, err := RestoreWorld(state)
	if err != nil {
		t.Fatal(err)
	}
	if !restored.EasyMode() {
		t.Fatal("lost difficulty")
	}
	for turn := 0; turn < 3; turn++ {
		if err := world.AdvanceTurn(); err != nil {
			t.Fatal(err)
		}
		if err := restored.AdvanceTurn(); err != nil {
			t.Fatal(err)
		}
	}
	a, _ := world.ExportState()
	b, _ := restored.ExportState()
	if !reflect.DeepEqual(a, b) {
		t.Fatal("difficulty restore changed continuation")
	}
}

func TestEasyModeCapsEruptionAndNormalModeRestoresLosses(t *testing.T) {
	for _, easy := range []bool{true, false} {
		world, _ := NewWorld(42)
		world.bands = world.bands[:1]
		activeTurn := 1
		for !MacroEpisodeActive(activeTurn) {
			activeTurn++
		}
		world.turn = activeTurn - 1
		habitat, climate, err := BuildHabitat(world.grid, world.seed, world.turn)
		if err != nil {
			t.Fatal(err)
		}
		world.habitat, world.climate = habitat, climate
		season, _ := SeasonForTurn(world.turn)
		for id := range TileCount {
			geography, _ := world.grid.Tile(TileID(id))
			world.tiles[id] = InitialTileState(world.seed, geography, habitat[id], season)
			if CampanianZone(TileID(id)) != MacroUnaffected && habitat[id].BaselineK > 0 {
				world.bands[0].TileID = TileID(id)
			}
		}
		world.bands[0].Population = 10000
		world.SetEasyMode(true)
		world.SetEasyMode(easy)
		if err := world.AdvanceTurn(); err != nil {
			t.Fatal(err)
		}
		if !easy && len(world.bands) == 0 {
			continue
		}
		if len(world.bands) != 1 {
			t.Fatal("easy mode lost band during eruption")
		}
		band := world.bands[0]
		if easy && band.Population < 9000 {
			t.Fatalf("eruption exceeded cap: %d", band.Population)
		}
		if !easy && band.Population >= 9000 {
			t.Fatalf("normal mode retained easy cap: %d", band.Population)
		}
		if band.LastOutcomeReport.MacroHealthLoss == 0 {
			t.Fatalf("fixture missed eruption: easy=%v tile=%d zone=%d turn=%d outcome=%+v", easy, band.TileID, CampanianZone(band.TileID), world.turn, band.LastOutcomeReport)
		}
	}
}

func TestEasyModeColdSurvivorCanBeSavedAndDifficultyDisabled(t *testing.T) {
	world, _ := NewWorld(42)
	world.bands = world.bands[:1]
	// Start on habitable land and find a turn where it becomes uninhabitable.
	var destination TileID
	found := false
	for turn := 1; turn <= MaxCampaignTurn && !found; turn++ {
		next, _, err := BuildHabitat(world.grid, world.seed, turn)
		if err != nil {
			t.Fatal(err)
		}
		for id := range TileCount {
			if world.habitat[id].BaselineK > 0 && next[id].BaselineK == 0 {
				destination, found = TileID(id), true
				break
			}
		}
		if !found {
			world.turn = turn
			world.habitat = next
		}
	}
	if !found {
		t.Fatal("no cooling boundary found")
	}
	season, _ := SeasonForTurn(world.turn)
	for id := range TileCount {
		geography, _ := world.grid.Tile(TileID(id))
		world.tiles[id] = InitialTileState(world.seed, geography, world.habitat[id], season)
	}
	world.bands[0].TileID = destination
	world.bands[0].Population = 100
	world.SetEasyMode(true)
	if err := world.AdvanceTurn(); err != nil {
		t.Fatal(err)
	}
	if len(world.bands) != 1 || world.bands[0].Population < 90 || world.habitat[destination].BaselineK != 0 {
		t.Fatal("cold bypassed easy-mode cap")
	}
	for _, easy := range []bool{true, false} {
		world.SetEasyMode(easy)
		state, err := world.ExportState()
		if err != nil {
			t.Fatal(err)
		}
		restored, err := RestoreWorld(state)
		if err != nil {
			t.Fatal(err)
		}
		if restored.EasyMode() != easy {
			t.Fatal("cold survivor lost difficulty setting")
		}
	}
}
