package domain

import (
	"math"
	"testing"
)

func TestTechnologyDAGAndCapacityProduct(t *testing.T) {
	state := TechnologyState{}
	if err := state.Select(TailoredClothing); err != ErrMissingTechnologyPrerequisite {
		t.Fatalf("locked selection = %v", err)
	}
	for technology := Technology(0); technology < TechCount; technology++ {
		state.Acquired |= 1 << technology
	}
	if got := state.CapacityMultiplier(); got != 1.4772347614192902 {
		t.Fatalf("all-tech product = %.17g", got)
	}
}

func TestResearchProductionAndCompletion(t *testing.T) {
	state := TechnologyState{}
	if err := state.Select(Firecraft); err != nil {
		t.Fatal(err)
	}
	if got := state.ApplyResearch(ResearchHalfSaturation); got != 10 {
		t.Fatalf("half-saturation gain = %v", got)
	}
	for state.HasTarget {
		state.ApplyResearch(1_000)
	}
	if !state.Has(Firecraft) || state.Progress[Firecraft] != ResearchCost[Firecraft] {
		t.Fatalf("research did not normalize: %#v", state)
	}
	if math.IsNaN(state.CapacityMultiplier()) {
		t.Fatal("capacity multiplier is NaN")
	}
}
