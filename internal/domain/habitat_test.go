package domain

import (
	"math"
	"testing"
)

func TestHabitatSweepFiniteAndSeedIndependent(t *testing.T) {
	grid, err := (WorldGenerator{}).Generate()
	if err != nil {
		t.Fatal(err)
	}
	for turn := 0; turn <= MaxCampaignTurn; turn++ {
		a, _, err := BuildHabitat(grid, 0, turn)
		if err != nil {
			t.Fatal(err)
		}
		b, _, err := BuildHabitat(grid, ^uint64(0), turn)
		if err != nil {
			t.Fatal(err)
		}
		for id := range TileCount {
			geography, _ := grid.Tile(TileID(id))
			left, right := a[id], b[id]
			if !geography.Land {
				if left.BaselineK != 0 {
					t.Fatalf("water tile %d is habitable", id)
				}
				continue
			}
			for _, value := range []float64{left.EffectiveMoisture, left.VegetationIndex, left.HabitatTemperatureC, left.LocalTemperatureC, left.BaselineK, left.MovementCost} {
				if math.IsNaN(value) || math.IsInf(value, 0) {
					t.Fatalf("non-finite tile %d turn %d", id, turn)
				}
			}
			if left.EffectiveMoisture < 0 || left.EffectiveMoisture > 1 || left.VegetationIndex < 0 || left.VegetationIndex > 1 || left.BaselineK < 0 || left.MovementCost < 1 || left.MovementCost > MaxMovementCost {
				t.Fatalf("out-of-range habitat tile %d turn %d: %#v", id, turn, left)
			}
			if left.Biome != right.Biome || left.HabitatTemperatureC != right.HabitatTemperatureC || left.EffectiveMoisture != right.EffectiveMoisture || left.BaselineK != right.BaselineK {
				t.Fatalf("seed changed habitat tile %d turn %d", id, turn)
			}
			if left.LocalTemperatureC == right.LocalTemperatureC && UnitNoiseV1(0, turn) != UnitNoiseV1(^uint64(0), turn) {
				t.Fatalf("seed noise did not reach local temperature")
			}
		}
	}
}

func TestTurnZeroEastAfricaHasSavannaAndWoodland(t *testing.T) {
	grid, _ := (WorldGenerator{}).Generate()
	habitat, _, _ := BuildHabitat(grid, 0, 0)
	var savanna, woodland bool
	for id := range TileCount {
		geography, _ := grid.Tile(TileID(id))
		if !geography.Land || geography.Region != EastAfrica {
			continue
		}
		savanna = savanna || habitat[id].Biome == Savanna
		woodland = woodland || habitat[id].Biome == RiverineWoodland
	}
	if !savanna || !woodland {
		t.Fatalf("turn-zero East Africa savanna=%v woodland=%v", savanna, woodland)
	}
}
