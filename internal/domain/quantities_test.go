package domain

import (
	"math"
	"testing"
)

func TestPopulationRoundingIsBoundedAndDeterministic(t *testing.T) {
	for input, want := range map[float64]Population{0.49: 0, 0.5: 1, 41.49: 41, 41.5: 42} {
		got, err := RoundPopulation(input)
		if err != nil {
			t.Fatalf("RoundPopulation(%v): %v", input, err)
		}
		if got != want {
			t.Fatalf("RoundPopulation(%v) = %v; want %v", input, got, want)
		}
	}
	if got, err := RoundPopulation(float64(MaxPopulation)); err != nil || got != MaxPopulation {
		t.Fatalf("RoundPopulation(max) = %v, %v", got, err)
	}
	quietNaN := math.Float64frombits(0x7FF8000000000000)
	positiveInf := math.Float64frombits(0x7FF0000000000000)
	for _, invalid := range []float64{-1, quietNaN, positiveInf, float64(MaxPopulation) + 1} {
		if _, err := RoundPopulation(invalid); err == nil {
			t.Fatalf("RoundPopulation(%v) accepted an invalid result", invalid)
		}
	}
}
