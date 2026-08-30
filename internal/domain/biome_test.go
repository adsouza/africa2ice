package domain

import (
	"math"
	"testing"
)

func TestBiomeClassifierPrecedenceAndThresholds(t *testing.T) {
	if got := ClassifyBiome(1.0000000000000002, true, 1, 20); got != MountainousHighlands {
		t.Fatalf("highland precedence: %v", got)
	}
	if got := ClassifyBiome(1, true, 1, 20); got != CoastalShrubland {
		t.Fatalf("strict highland boundary: %v", got)
	}
	if got := ClassifyBiome(0, false, 0.2, -1); got != GlacialTundra {
		t.Fatalf("cold low-V: %v", got)
	}
	if got := ClassifyBiome(0, false, 0.45, 10); got != Savanna {
		t.Fatalf("right-continuous savanna threshold: %v", got)
	}
	if got := ClassifyBiome(0, false, 0.75, 10); got != RiverineWoodland {
		t.Fatalf("woodland threshold: %v", got)
	}
}

func TestCapacityAndMovementReferences(t *testing.T) {
	if got := BaselineKCurve(0.60); math.Abs(got-120) > 1e-12 {
		t.Fatalf("BaselineKCurve(.6) = %v", got)
	}
	if got := BaselineK(0.60, CoastalShrubland); math.Abs(got-138) > 1e-12 {
		t.Fatalf("coastal K = %v", got)
	}
	if got := BaselineK(0.60, MountainousHighlands); math.Abs(got-78) > 1e-12 {
		t.Fatalf("highland K = %v", got)
	}
	if got := ComposedMovementCost(0.60, CoastalShrubland); got != 1.25 {
		t.Fatalf("coastal movement = %v", got)
	}
	if got := ComposedMovementCost(0.60, MountainousHighlands); got != 2.5 {
		t.Fatalf("highland movement = %v", got)
	}
}

func TestLatitudeTemperatureAnchors(t *testing.T) {
	north, err := SeaLevelTemperatureC(0)
	if err != nil {
		t.Fatal(err)
	}
	south, err := SeaLevelTemperatureC(63)
	if err != nil {
		t.Fatal(err)
	}
	if math.Abs(north-(-8.180339887498945)) > 1e-12 {
		t.Fatalf("north edge = %.17g", north)
	}
	if math.Abs(south-5.909430734646929) > 1e-12 {
		t.Fatalf("south edge = %.17g", south)
	}
	sea, _ := HabitatTemperatureC(20, 0, 0)
	high, _ := HabitatTemperatureC(20, 1, 0)
	if sea-high != 6.5 {
		t.Fatalf("lapse difference = %v", sea-high)
	}
}
