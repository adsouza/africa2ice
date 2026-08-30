package domain

import (
	"math"
	"testing"
)

func TestClimateEndpointsAndBounds(t *testing.T) {
	for turn := 0; turn <= MaxCampaignTurn; turn++ {
		state, err := ClimateAt(42, turn)
		if err != nil {
			t.Fatal(err)
		}
		if state.AridityIndex < 0 || state.AridityIndex > 1 || math.IsNaN(state.GlobalTempOffset) || math.IsInf(state.GlobalTempOffset, 0) {
			t.Fatalf("invalid climate at %d: %#v", turn, state)
		}
		for _, offset := range state.RegionalAbrupt {
			if math.Abs(offset) > 2.5 {
				t.Fatalf("abrupt cap exceeded at %d: %v", turn, offset)
			}
		}
	}
	final, _ := ClimateAt(42, 400)
	if final.LongTermTempOffset != -LGMCooling || final.LongTermMoistureOffset != -AridificationAmplitude {
		t.Fatalf("final trend = %#v", final)
	}
	if !BeringiaOpen(final.LongTermTempOffset) {
		t.Fatal("Beringia is not open at terminal cooling")
	}
}

func TestClimateEpochReferenceTrajectory(t *testing.T) {
	if got := ClimateEpochForTurn(0); got != HumidOptimum {
		t.Fatalf("turn 0 epoch = %v", got)
	}
	seen := [3]bool{}
	for turn := 0; turn <= MaxCampaignTurn; turn++ {
		seen[ClimateEpochForTurn(turn)] = true
	}
	if !seen[0] || !seen[1] || !seen[2] {
		t.Fatalf("epoch trajectory misses a state: %v", seen)
	}
}

func TestClimateTablesMatchGenerationDefinitions(t *testing.T) {
	for turn := 0; turn <= MaxCampaignTurn; turn++ {
		date, _ := CampaignDate(turn)
		want := math.Sin(8 * math.Pi * date.CalendarProgress)
		got := math.Float64frombits(orbitalSinBits[turn])
		if math.Abs(got-want) > 1e-15 {
			t.Fatalf("orbital table[%d] = %.17g, want %.17g", turn, got, want)
		}
	}
}
