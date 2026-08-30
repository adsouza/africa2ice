package domain

import (
	"math"
	"testing"
)

func TestSiberianColdResourceExample(t *testing.T) {
	cap := ResourceCaps(GlacialTundra, SeasonCold, 0, 1)
	if cap.Flora != 15 || cap.Fauna != 270 || cap.Water != 137.5 {
		t.Fatalf("caps = %#v", cap)
	}
	state := (TileState{}).Regenerate(cap, ResourceVector{})
	if state.Stock.Flora != 4.5 || state.Stock.Fauna != 40.5 || state.Stock.Water != 68.75 {
		t.Fatalf("regrowth = %#v", state.Stock)
	}
}

func TestRegenerationOrderingAndBounds(t *testing.T) {
	cap := ResourceVector{100, 100, 100}
	state := TileState{Stock: ResourceVector{200, 50, 0}}
	next := state.Regenerate(cap, ResourceVector{10, 70, 60})
	if next.Stock.Flora != 90 || next.Stock.Fauna < 0 || next.Stock.Water != 0 {
		t.Fatalf("next = %#v", next)
	}
	for _, value := range []float64{next.Stock.Flora, next.Stock.Fauna, next.Stock.Water} {
		if math.IsNaN(value) || value < 0 || value > 100 {
			t.Fatalf("stock out of bounds: %v", value)
		}
	}
}

func TestDegradationDamageRecoveryAndCap(t *testing.T) {
	if got := NextDegradation(0, 150, 100); math.Abs(got-0.015) > 1e-15 {
		t.Fatalf("damage = %v", got)
	}
	if got := NextDegradation(0.75, 0, 100); got != 0.72 {
		t.Fatalf("recovery = %v", got)
	}
	if got := NextDegradation(0.74, 1000, 100); got != MaxDegradation {
		t.Fatalf("cap = %v", got)
	}
}

func TestInitialStockDeterministicAndBounded(t *testing.T) {
	grid, _ := (WorldGenerator{}).Generate()
	habitat, _, _ := BuildHabitat(grid, 7, 0)
	season, _ := SeasonForTurn(0)
	for id := range TileCount {
		geography, _ := grid.Tile(TileID(id))
		a := InitialTileState(7, geography, habitat[id], season)
		b := InitialTileState(7, geography, habitat[id], season)
		if a != b {
			t.Fatalf("tile %d initialization not deterministic", id)
		}
		cap := ResourceCaps(habitat[id].Biome, season, 0, habitat[id].BaselineK)
		if a.Stock.Flora < 0 || a.Stock.Flora > cap.Flora || a.Stock.Fauna < 0 || a.Stock.Fauna > cap.Fauna || a.Stock.Water < 0 || a.Stock.Water > cap.Water {
			t.Fatalf("tile %d stock/cap mismatch: %#v %#v", id, a.Stock, cap)
		}
	}
}
