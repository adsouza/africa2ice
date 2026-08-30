package domain

import (
	"math"
	"testing"
)

// Stochastic rounding is what lets a sub-person change register at all. The
// deterministic +0.5 form created a deadband of half a person either side of
// zero net change, inside which nothing ever happened: with growth of ~0.05
// people per turn, every birth the model computed was discarded on the turn it
// was computed. Carrying a fractional remainder would mean storing a fraction
// of a person, which is not a thing; drawing instead keeps population strictly
// integral while making the expected value come out right over time.

// An exact integer must survive every draw, so a band with no net change never
// drifts on its own.
func TestExactPopulationsSurviveEveryDraw(t *testing.T) {
	cases := []struct {
		value float64
		want  Population
	}{{0, 0}, {1, 1}, {20, 20}, {41, 41}}
	for _, testCase := range cases {
		for seed := uint64(0); seed < 64; seed++ {
			got, err := RoundPopulation(testCase.value, NewWorldRNG(seed))
			if err != nil || got != testCase.want {
				t.Fatalf("RoundPopulation(%v) with seed %d = %v, %v; want %v", testCase.value, seed, got, err, testCase.want)
			}
		}
	}
	if got, err := RoundPopulation(float64(MaxPopulation), NewWorldRNG(1)); err != nil || got != MaxPopulation {
		t.Fatalf("RoundPopulation(max) = %v, %v", got, err)
	}
}

// A fractional value may only ever land on the two integers bracketing it.
func TestFractionalPopulationsLandOnAdjacentIntegers(t *testing.T) {
	cases := []struct {
		value        float64
		lower, upper Population
	}{{0.25, 0, 1}, {20.5, 20, 21}, {41.75, 41, 42}}
	rng := NewWorldRNG(7)
	for _, testCase := range cases {
		for draw := 0; draw < 500; draw++ {
			got, err := RoundPopulation(testCase.value, rng)
			if err != nil {
				t.Fatal(err)
			}
			if got != testCase.lower && got != testCase.upper {
				t.Fatalf("RoundPopulation(%v) = %v, want %v or %v", testCase.value, got, testCase.lower, testCase.upper)
			}
		}
	}
}

// Unbiased in expectation: that is the whole point of drawing rather than
// truncating, and it is what makes a 5%-of-a-person growth term real.
func TestPopulationRoundingIsUnbiased(t *testing.T) {
	const draws = 20000
	rng := NewWorldRNG(11)
	rounded := 0
	for i := 0; i < draws; i++ {
		got, err := RoundPopulation(20.25, rng)
		if err != nil {
			t.Fatal(err)
		}
		if got == 21 {
			rounded++
		}
	}
	share := float64(rounded) / float64(draws)
	if math.Abs(share-0.25) > 0.02 {
		t.Fatalf("rounded up %.4f of the time for a 0.25 fraction, want ~0.25", share)
	}
}

func TestPopulationRoundingConsumesExactlyOneDraw(t *testing.T) {
	used, reference := NewWorldRNG(23), NewWorldRNG(23)
	if _, err := RoundPopulation(3.5, used); err != nil {
		t.Fatal(err)
	}
	reference.Float64()
	if got, want := used.Float64(), reference.Float64(); got != want {
		t.Fatalf("rounding consumed a different number of draws: %v vs %v", got, want)
	}
}

func TestPopulationRoundingRejectsInvalidValues(t *testing.T) {
	quietNaN := math.Float64frombits(0x7FF8000000000000)
	positiveInf := math.Float64frombits(0x7FF0000000000000)
	for _, invalid := range []float64{-1, quietNaN, positiveInf, float64(MaxPopulation) + 1} {
		if _, err := RoundPopulation(invalid, NewWorldRNG(3)); err == nil {
			t.Fatalf("RoundPopulation(%v) accepted an invalid result", invalid)
		}
	}
}
