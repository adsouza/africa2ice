package domain

import (
	"math"
	"testing"
)

// DESIGN.md §12 step 4 says what this step adds over §7's fixtures is "scale and
// authoring": the seed-independent habitat quantities are swept exhaustively for
// all 6,144 tiles at all 401 turns rather than sampled. That exhaustiveness is
// what makes the configuration-time gates real instead of aspirational.
//
// Only the seed-independent quantities are swept here. LocalTemperatureC carries
// the per-tile climate noise and is therefore seed-dependent by construction; it
// is covered by the noise bounds below instead.

func TestSeedIndependentHabitatSweepsOverEveryTileAndTurn(t *testing.T) {
	grid, err := (WorldGenerator{}).Generate()
	if err != nil {
		t.Fatal(err)
	}
	reference := BalanceSeedCorpus[0]
	// One contrasting seed is enough to prove independence at every one of the
	// 2,463,744 tile-turns; the corpus-wide check below covers the rest at a
	// coarser turn resolution so the sweep stays inside the CI budget.
	contrast := BalanceSeedCorpus[len(BalanceSeedCorpus)-1]

	for turn := 0; turn <= MaxCampaignTurn; turn++ {
		first, _, err := BuildHabitat(grid, reference, turn)
		if err != nil {
			t.Fatalf("turn %d: %v", turn, err)
		}
		second, _, err := BuildHabitat(grid, contrast, turn)
		if err != nil {
			t.Fatalf("turn %d: %v", turn, err)
		}
		for id := range TileCount {
			left, right := first[id], second[id]
			if left.HabitatTemperatureC != right.HabitatTemperatureC {
				t.Fatalf("turn %d tile %d: HabitatTemperatureC moved with the seed: %v vs %v",
					turn, id, left.HabitatTemperatureC, right.HabitatTemperatureC)
			}
			if left.EffectiveMoisture != right.EffectiveMoisture {
				t.Fatalf("turn %d tile %d: EffectiveMoisture moved with the seed: %v vs %v",
					turn, id, left.EffectiveMoisture, right.EffectiveMoisture)
			}
			if left.VegetationIndex != right.VegetationIndex {
				t.Fatalf("turn %d tile %d: VegetationIndex moved with the seed: %v vs %v",
					turn, id, left.VegetationIndex, right.VegetationIndex)
			}
			if left.Biome != right.Biome {
				t.Fatalf("turn %d tile %d: biome moved with the seed: %v vs %v", turn, id, left.Biome, right.Biome)
			}
			if left.BaselineK != right.BaselineK {
				t.Fatalf("turn %d tile %d: BaselineK moved with the seed: %v vs %v", turn, id, left.BaselineK, right.BaselineK)
			}
			if left.MovementCost != right.MovementCost {
				t.Fatalf("turn %d tile %d: composed movement cost moved with the seed: %v vs %v",
					turn, id, left.MovementCost, right.MovementCost)
			}
		}
	}
}

// TestHabitatQuantitiesStayFiniteAndBoundedEverywhere is the other half of the
// sweep: every value the simulation reads must be finite and in range at every
// tile-turn, not merely at the sampled ones a fixture can name.
func TestHabitatQuantitiesStayFiniteAndBoundedEverywhere(t *testing.T) {
	grid, err := (WorldGenerator{}).Generate()
	if err != nil {
		t.Fatal(err)
	}
	seed := BalanceSeedCorpus[0]
	for turn := 0; turn <= MaxCampaignTurn; turn++ {
		habitat, _, err := BuildHabitat(grid, seed, turn)
		if err != nil {
			t.Fatalf("turn %d: %v", turn, err)
		}
		for id := range TileCount {
			tile := habitat[id]
			for name, value := range map[string]float64{
				"HabitatTemperatureC": tile.HabitatTemperatureC,
				"LocalTemperatureC":   tile.LocalTemperatureC,
				"EffectiveMoisture":   tile.EffectiveMoisture,
				"VegetationIndex":     tile.VegetationIndex,
				"BaselineK":           tile.BaselineK,
				"MovementCost":        tile.MovementCost,
			} {
				if math.IsNaN(value) || math.IsInf(value, 0) {
					t.Fatalf("turn %d tile %d: %s is not finite (%v)", turn, id, name, value)
				}
			}
			if tile.EffectiveMoisture < 0 || tile.EffectiveMoisture > 1 {
				t.Fatalf("turn %d tile %d: EffectiveMoisture %v is outside [0, 1]", turn, id, tile.EffectiveMoisture)
			}
			if tile.VegetationIndex < 0 || tile.VegetationIndex > 1 {
				t.Fatalf("turn %d tile %d: VegetationIndex %v is outside [0, 1]", turn, id, tile.VegetationIndex)
			}
			if tile.BaselineK < 0 {
				t.Fatalf("turn %d tile %d: BaselineK %v is negative", turn, id, tile.BaselineK)
			}
			if tile.MovementCost < 0 || tile.MovementCost > MaxMovementCost {
				t.Fatalf("turn %d tile %d: MovementCost %v is outside [0, %v]", turn, id, tile.MovementCost, MaxMovementCost)
			}
			if tile.Biome >= BiomeCount {
				t.Fatalf("turn %d tile %d: biome %d is outside the closed catalog", turn, id, tile.Biome)
			}
		}
	}
}

// TestSeedIndependenceHoldsAcrossTheWholeCorpus widens the seed axis at a
// coarser turn resolution, so the claim is about the corpus and not about one
// contrasting pair.
func TestSeedIndependenceHoldsAcrossTheWholeCorpus(t *testing.T) {
	grid, err := (WorldGenerator{}).Generate()
	if err != nil {
		t.Fatal(err)
	}
	for _, turn := range []int{0, 50, 100, 150, 200, 250, 300, 350, 400} {
		want, _, err := BuildHabitat(grid, BalanceSeedCorpus[0], turn)
		if err != nil {
			t.Fatal(err)
		}
		for _, seed := range BalanceSeedCorpus[1:] {
			got, _, err := BuildHabitat(grid, seed, turn)
			if err != nil {
				t.Fatal(err)
			}
			for id := range TileCount {
				if got[id].Biome != want[id].Biome || got[id].BaselineK != want[id].BaselineK ||
					got[id].MovementCost != want[id].MovementCost {
					t.Fatalf("seed %#x turn %d tile %d: seed-independent habitat differs", seed, turn, id)
				}
			}
		}
	}
}
