package application

import (
	"reflect"
	"testing"

	"github.com/adsouza/africa2ice/internal/domain"
	"github.com/adsouza/africa2ice/pkg/gameapi"
)

func TestSnapshotNamesLakesAtGeographicAnchors(t *testing.T) {
	service, err := NewGameService(1)
	if err != nil {
		t.Fatal(err)
	}
	frame, err := service.Snapshot()
	if err != nil {
		t.Fatal(err)
	}
	count := 0
	for _, tile := range frame.Tiles {
		if tile.Explored && !tile.Land {
			if tile.WaterBody != service.world.Grid().WaterBodyName(domain.TileID(tile.ID)) || tile.WaterBody == "" {
				t.Fatalf("explored water tile lacks its geographic name: %+v", tile)
			}
		} else if tile.WaterBody != "" {
			t.Fatalf("water name exposed on land or in fog: %+v", tile)
		}
		if tile.NearbyLake != "" {
			count++
			if !tile.Land || !tile.Explored {
				t.Fatalf("lake context exposed on unsuitable tile: %+v", tile)
			}
		}
	}
	if count < 2 {
		t.Fatalf("named lake tiles = %d, want at least 2", count)
	}
	for index, want := range map[int]string{1: "Lake Turkana", 2: "Lake Victoria"} {
		if got := frame.Tiles[domain.StartingTileIDs[index]].NearbyLake; got != want {
			t.Fatalf("anchor %d lake = %q, want %q", index, got, want)
		}
	}
	save, err := service.ExportSaveState()
	if err != nil {
		t.Fatal(err)
	}
	service.world, err = save.RestoreWorld()
	if err != nil {
		t.Fatal(err)
	}
	loaded, err := service.Snapshot()
	if err != nil {
		t.Fatal(err)
	}
	for id, tile := range frame.Tiles {
		if loaded.Tiles[id].NearbyLake != tile.NearbyLake || !reflect.DeepEqual(loaded.Tiles[id].Lakes, tile.Lakes) {
			t.Fatalf("lake name changed after load at tile %d", id)
		}
	}
}

// The frame must never carry a migration candidate pointing at a tile the
// player has not explored. The domain deliberately gives archaic bands
// hidden-target candidates so the computer policy can route them (see
// migration.go's HomoSapiens-only explored filter), but every consumer of the
// frame is player-facing: pkg/render draws a highlight per candidate, pkg/ui
// derives affordances from them, and none of those gate on Explored. Filtering
// once here is what makes all of them correct.
//
// The precondition assertion matters as much as the invariant: without it a
// fixture that happens to expose no hidden candidates would let this test pass
// while proving nothing.
func TestFrameCarriesNoCandidateOnAnUnexploredTile(t *testing.T) {
	service, err := NewGameService(0x5eed)
	if err != nil {
		t.Fatalf("NewGameService: %v", err)
	}

	hidden := 0
	for _, entry := range service.world.MigrationCandidatesByBand() {
		for _, candidate := range entry.Candidates {
			if !service.world.IsExplored(candidate.TileID) {
				hidden++
			}
		}
	}
	if hidden == 0 {
		t.Fatal("fixture exposes no hidden-target candidates in the domain, so this test would prove nothing")
	}
	t.Logf("domain exposes %d candidates on unexplored tiles", hidden)

	frame, err := service.Snapshot()
	if err != nil {
		t.Fatalf("Snapshot: %v", err)
	}
	leaked := 0
	for _, band := range frame.Bands {
		for _, candidate := range band.MigrationCandidates {
			if !frame.Tiles[candidate.TileID].Explored {
				leaked++
				if leaked <= 3 {
					t.Errorf("band %d (%v) has a candidate on unexplored tile %d", band.ID, band.Species, candidate.TileID)
				}
			}
		}
	}
	if leaked > 0 {
		t.Fatalf("%d frame candidates point at unexplored tiles", leaked)
	}

	// The other half of the invariant. A filter that dropped everything would
	// satisfy the check above perfectly, so assert the sapiens bands the player
	// actually commands still have somewhere to go -- their candidates were
	// already explored-only in the domain, so this filter must be a no-op for
	// them.
	sapiensWithCandidates := 0
	for _, band := range frame.Bands {
		if band.Species == gameapi.HomoSapiens && len(band.MigrationCandidates) > 0 {
			sapiensWithCandidates++
		}
	}
	if sapiensWithCandidates == 0 {
		t.Fatal("no sapiens band kept a migration candidate; the filter is too aggressive")
	}
	t.Logf("%d sapiens bands retain candidates", sapiensWithCandidates)
}

// The UI reads arrival crowding from the frame instead of recomputing it, so
// each projected candidate must carry exactly the domain's ArrivalStress.
func TestFrameCandidatesCarryDomainArrivalStress(t *testing.T) {
	service, err := NewGameService(0x5eed)
	if err != nil {
		t.Fatalf("NewGameService: %v", err)
	}
	want := map[[2]uint32]float64{}
	for _, entry := range service.world.MigrationCandidatesByBand() {
		for _, candidate := range entry.Candidates {
			want[[2]uint32{uint32(entry.BandID), uint32(candidate.TileID)}] = candidate.ArrivalStress
		}
	}
	frame, err := service.Snapshot()
	if err != nil {
		t.Fatalf("Snapshot: %v", err)
	}
	checked := 0
	for _, band := range frame.Bands {
		for _, candidate := range band.MigrationCandidates {
			expected := want[[2]uint32{uint32(band.ID), uint32(candidate.TileID)}]
			if expected <= 0 {
				t.Fatalf("band %d candidate %d: domain ArrivalStress %v is not positive", band.ID, candidate.TileID, expected)
			}
			if candidate.ArrivalStress != expected {
				t.Fatalf("band %d candidate %d: frame ArrivalStress %v, domain %v", band.ID, candidate.TileID, candidate.ArrivalStress, expected)
			}
			checked++
		}
	}
	if checked == 0 {
		t.Fatal("frame has no candidates, so this test would prove nothing")
	}
}

// The step 9 warning contract: on the turn before the eruption an explored
// affected tile shows the impact it is about to take, flagged as a warning;
// while the eruption strikes it shows the current impact; an unexplored tile
// or an unaffected one shows nothing. Before the projection only carried the
// active impact, so the warning turn, the one a player can act on, was blank.
func TestVisibleMacroImpactProjectsTheWarningTurn(t *testing.T) {
	world, err := domain.NewWorld(1)
	if err != nil {
		t.Fatal(err)
	}
	grid := world.Grid()
	activeTurn := 1
	for !domain.MacroEpisodeActive(activeTurn) {
		activeTurn++
	}
	var direct, unaffected domain.TileGeography
	for id := range domain.TileCount {
		tile, _ := grid.Tile(domain.TileID(id))
		switch {
		case !tile.Land:
		case domain.CampanianZone(tile.ID) == domain.MacroDirect:
			direct = tile
		case domain.CampanianZone(tile.ID) == domain.MacroUnaffected:
			unaffected = tile
		}
	}
	if !direct.Land {
		t.Fatal("no land tile in the direct zone")
	}
	warned := visibleMacroImpact(direct, activeTurn-1, true)
	if !warned.Visible || !warned.Warned || warned.Intensity != 1 || warned.HabitatFactor != 0.5 {
		t.Fatalf("warning-turn impact = %+v, want next turn's direct impact flagged as a warning", warned)
	}
	active := visibleMacroImpact(direct, activeTurn, true)
	if !active.Visible || active.Warned || active.Intensity != 1 {
		t.Fatalf("active-turn impact = %+v, want the current impact, not a warning", active)
	}
	for name, got := range map[string]gameapi.MacroImpactSummary{
		"unexplored on the warning turn": visibleMacroImpact(direct, activeTurn-1, false),
		"unaffected on the warning turn": visibleMacroImpact(unaffected, activeTurn-1, true),
		"two turns before":               visibleMacroImpact(direct, activeTurn-2, true),
		"after the episode":              visibleMacroImpact(direct, activeTurn+1, true),
	} {
		if got.Visible {
			t.Errorf("%s shows %+v", name, got)
		}
	}
}
