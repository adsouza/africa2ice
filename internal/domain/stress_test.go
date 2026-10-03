package domain

import (
	"math"
	"testing"
)

// stressFixture places band 0 on a land tile that an active macro episode
// degrades, with a degraded stock, two capacity technologies, and a second
// resident band, so every factor in DESIGN.md's
// Stress = P_total / (BaselineK · (1 − Degradation) · MacroHabitatFactor · T_tech)
// differs from 1 and the whole-tile population differs from the band's own.
func stressFixture(t *testing.T) (*World, TileID) {
	t.Helper()
	world, err := NewWorld(2)
	if err != nil {
		t.Fatal(err)
	}
	for turn := 1; turn <= MaxCampaignTurn; turn++ {
		for id := range TileCount {
			geography, _ := world.grid.Tile(TileID(id))
			if !geography.Land || world.habitat[id].BaselineK <= 0 || MacroImpactAt(geography, turn).HabitatFactor >= 1 {
				continue
			}
			tile := TileID(id)
			for word := range world.exploredTiles {
				world.exploredTiles[word] = ^uint64(0)
			}
			world.turn = turn
			world.tiles[tile].Degradation = 0.5
			world.bands[0].TileID, world.bands[0].Population = tile, 300
			world.bands[0].Technology.Acquired = 1<<0 | 1<<5
			world.bands[1].TileID, world.bands[1].Population = tile, 200
			return world, tile
		}
	}
	t.Fatal("no macro-affected habitable tile in the campaign")
	return nil, 0
}

func specStress(world *World, tile TileID, population float64, technology TechnologyState) float64 {
	geography, _ := world.grid.Tile(tile)
	factor := MacroImpactAt(geography, world.turn).HabitatFactor
	return population / float64(world.habitat[tile].BaselineK*(1-world.tiles[tile].Degradation)*factor*technology.CapacityMultiplier())
}

func assertClose(t *testing.T, name string, got, want float64) {
	t.Helper()
	if math.Abs(got-want) > float64(1e-12*math.Abs(want)) {
		t.Fatalf("%s = %.15g, want %.15g", name, got, want)
	}
}

func TestBandStressDividesWholeTilePopulationByEcologicalKWithTechnology(t *testing.T) {
	world, tile := stressFixture(t)
	band := world.bands[0]
	assertClose(t, "BandStress", world.BandStress(band.ID), specStress(world, tile, 500, band.Technology))
}

func TestMigrationCandidateArrivalStressAddsTheArrivingBand(t *testing.T) {
	world, _ := stressFixture(t)
	band := world.bands[0]
	candidates := world.MigrationCandidates(band.ID)
	if len(candidates) == 0 {
		t.Fatal("fixture band has no migration candidates")
	}
	for _, candidate := range candidates {
		world.tiles[candidate.TileID].Degradation = 0.25
	}
	candidates = world.MigrationCandidates(band.ID)
	for _, candidate := range candidates {
		arriving := float64(candidate.DestinationPopulation) + float64(band.Population)
		assertClose(t, "ArrivalStress", candidate.ArrivalStress, specStress(world, candidate.TileID, arriving, band.Technology))
	}
}
