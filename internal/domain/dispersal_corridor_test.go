package domain

import (
	"fmt"
	"strings"
	"testing"
)

// lastViableDepartureTurn is the latest turn at which a sapiens band may still
// leave its founding region and reach a destination region before the campaign
// ends. It is a design line, not a measurement: a player who spends the early
// eras consolidating in Africa should still be able to attempt the dispersal,
// and only the final era (turn 300, 25,000 BP) is late enough that starting the
// journey is a genuine forfeit. Beyond it the map may close.
//
// Nothing in the UI announces this deadline, so a corridor that shuts earlier
// than this leaves the player playing a campaign that cannot be won without
// ever being told. That is the failure this gate exists to catch.
const lastViableDepartureTurn = 300

// destinationReachability solves the campaign backwards as a time-expanded
// graph: reachable[turn] holds every tile from which some destination region is
// still reachable by turn MaxCampaignTurn. A band may wait, take an ordinary
// land edge, or take a named passage, and in every case the tile it will stand
// on must be habitable on both sides of the turn boundary, because
// QueueMigration validates against the planning frame and turn resolution
// validates again after phase-1 climate.
//
// Technology gating is deliberately ignored, so this is an upper bound on what
// any player could achieve. A failure here cannot be answered with "research
// Coastal Navigation first".
func destinationReachability(t *testing.T, grid *Grid) [][ExplorationWordCount]uint64 {
	t.Helper()
	habitable := make([][ExplorationWordCount]uint64, MaxCampaignTurn+1)
	beringiaOpen := make([]bool, MaxCampaignTurn+1)
	for turn := 0; turn <= MaxCampaignTurn; turn++ {
		habitat, climate, err := BuildHabitat(grid, 0, turn)
		if err != nil {
			t.Fatalf("build habitat at turn %d: %v", turn, err)
		}
		beringiaOpen[turn] = BeringiaOpen(climate.LongTermTempOffset)
		for id := range TileCount {
			if habitat[id].BaselineK > 0 {
				habitable[turn][id/64] |= uint64(1) << (id % 64)
			}
		}
	}
	isHabitable := func(turn int, tile TileID) bool {
		return habitable[turn][tile/64]&(uint64(1)<<(tile%64)) != 0
	}

	destination := [TileCount]bool{}
	for id := range TileCount {
		geography, ok := grid.Tile(TileID(id))
		if !ok || !geography.Land {
			continue
		}
		for _, region := range DestinationRegions {
			if geography.Region == region {
				destination[id] = true
				break
			}
		}
	}

	reachable := make([][ExplorationWordCount]uint64, MaxCampaignTurn+1)
	mark := func(turn int, tile TileID) { reachable[turn][tile/64] |= uint64(1) << (tile % 64) }
	canReach := func(turn int, tile TileID) bool {
		return reachable[turn][tile/64]&(uint64(1)<<(tile%64)) != 0
	}
	edges := make([]GridEdge, 0, MaxGridNeighbors)
	for turn := MaxCampaignTurn; turn >= 0; turn-- {
		for id := range TileCount {
			tile := TileID(id)
			if !isHabitable(turn, tile) {
				continue
			}
			if destination[id] {
				mark(turn, tile)
				continue
			}
			if turn == MaxCampaignTurn {
				continue
			}
			step := func(to TileID) bool {
				return isHabitable(turn, to) && isHabitable(turn+1, to) && canReach(turn+1, to)
			}
			if step(tile) {
				mark(turn, tile)
				continue
			}
			reached := false
			for _, edge := range grid.AppendOrdinaryEdges(edges[:0], tile) {
				if step(edge.To) {
					reached = true
					break
				}
			}
			if !reached {
				for _, passage := range passageCatalog {
					to, atEndpoint := passageDestination(passage, tile)
					if !atEndpoint || (passage.ClimateGated && !beringiaOpen[turn+1]) {
						continue
					}
					if step(to) {
						reached = true
						break
					}
				}
			}
			if reached {
				mark(turn, tile)
			}
		}
	}
	return reachable
}

func TestDispersalCorridorStaysOpenUntilTheFinalEra(t *testing.T) {
	grid, err := WorldGenerator{}.Generate()
	if err != nil {
		t.Fatalf("generate grid: %v", err)
	}
	reachable := destinationReachability(t, grid)

	firstClosedTurn := -1
	for turn := 0; turn <= lastViableDepartureTurn; turn++ {
		open := false
		for index, anchor := range StartingAnchors {
			if anchor.Species != HomoSapiens {
				continue
			}
			tile := StartingTileIDs[index]
			if reachable[turn][tile/64]&(uint64(1)<<(tile%64)) != 0 {
				open = true
				break
			}
		}
		if !open {
			firstClosedTurn = turn
			break
		}
	}
	if firstClosedTurn >= 0 {
		date, _ := CampaignDate(firstClosedTurn)
		t.Errorf("no sapiens founding tile can still reach any destination region from turn %d (%d BP); "+
			"the gate requires the corridor to stay open through turn %d, because nothing tells the player it has shut",
			firstClosedTurn, date.YearBP, lastViableDepartureTurn)
	}
}

// TestCaucasusPassCarriesTheEasternCorridor pins the specific geography the
// eastern route depends on. Between the Black Sea and the Caspian the land
// narrows to two tile columns, and if both are authored as 2 km highlands the
// campaign's long-term cooling drives their vegetation index under
// VegetationColdCutoffC and the whole eastern corridor shuts for good. One
// column must stay low enough to keep its capacity.
func TestCaucasusPassCarriesTheEasternCorridor(t *testing.T) {
	grid, err := WorldGenerator{}.Generate()
	if err != nil {
		t.Fatalf("generate grid: %v", err)
	}
	// The land bridge rows between the two seas, from the Levant side north to
	// the open Frangistan steppe.
	rows := []int{17, 16, 15, 14}
	habitats := make([]*Habitat, MaxCampaignTurn+1)
	for turn := 0; turn <= MaxCampaignTurn; turn++ {
		habitat, _, err := BuildHabitat(grid, 0, turn)
		if err != nil {
			t.Fatalf("build habitat at turn %d: %v", turn, err)
		}
		habitats[turn] = habitat
	}
	for _, y := range rows {
		open := false
		var report strings.Builder
		for x := 26; x <= 29; x++ {
			tile, err := TileIDAt(x, y)
			if err != nil {
				continue
			}
			geography, ok := grid.Tile(tile)
			if !ok || !geography.Land {
				continue
			}
			habitableTurns := 0
			for turn := 0; turn <= MaxCampaignTurn; turn++ {
				if habitats[turn][tile].BaselineK > 0 {
					habitableTurns++
				}
			}
			fmt.Fprintf(&report, " tile %d (%.1fE/%.1fN elev=%.2fkm habitable %d/%d turns)",
				tile, geography.Longitude, geography.Latitude, geography.ElevationKm, habitableTurns, MaxCampaignTurn+1)
			if habitableTurns == MaxCampaignTurn+1 {
				open = true
			}
		}
		if !open {
			t.Errorf("no tile on the land bridge row y=%d stays habitable for the whole campaign:%s", y, report.String())
		}
	}
}

// TestLastHabitableTurnMatchesTheHabitatItSummarizes checks the precomputed
// trajectory against the authority it summarizes, on every land tile rather
// than a sample, so a tile whose capacity flickers late cannot be recorded as
// dead early.
func TestLastHabitableTurnMatchesTheHabitatItSummarizes(t *testing.T) {
	grid, err := WorldGenerator{}.Generate()
	if err != nil {
		t.Fatalf("generate grid: %v", err)
	}
	want := [TileCount]int{}
	for id := range TileCount {
		want[id] = -1
	}
	for turn := 0; turn <= MaxCampaignTurn; turn++ {
		habitat, _, err := BuildHabitat(grid, 0, turn)
		if err != nil {
			t.Fatalf("build habitat at turn %d: %v", turn, err)
		}
		for id := range TileCount {
			if habitat[id].BaselineK > 0 {
				want[id] = turn
			}
		}
	}
	for id := range TileCount {
		if got := grid.LastHabitableTurn(TileID(id)); got != want[id] {
			t.Fatalf("LastHabitableTurn(%d) = %d, want %d", id, got, want[id])
		}
	}
	if grid.LastHabitableTurn(TileCount) != -1 {
		t.Errorf("LastHabitableTurn() of an out-of-range tile = %d, want -1", grid.LastHabitableTurn(TileCount))
	}
}
