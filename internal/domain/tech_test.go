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
	if got := state.PlannedResearchGain(ResearchHalfSaturation); got != 10 {
		t.Fatalf("half-saturation gain = %v", got)
	}
	for state.HasTarget {
		state.AdvanceResearch(state.Target, state.PlannedResearchGain(1_000))
	}
	if !state.Has(Firecraft) || state.Progress[Firecraft] != ResearchCost[Firecraft] {
		t.Fatalf("research did not normalize: %#v", state)
	}
	if got := state.PlannedResearchGain(1_000); got != 0 {
		t.Fatalf("acquired technology still plans a gain: %v", got)
	}
	// Diffusion completes a technology nobody targeted, through the same rule.
	other := TechnologyState{}
	other.AdvanceResearch(HaftedTools, float64(ResearchCost[HaftedTools]*2))
	if !other.Has(HaftedTools) || other.Progress[HaftedTools] != ResearchCost[HaftedTools] {
		t.Fatalf("diffusion overshoot did not clamp: %#v", other)
	}
	// Prerequisites gate progress, not just selection. Trapping needs
	// CordageAndNets, which this state has never acquired.
	other.AdvanceResearch(Trapping, 1_000)
	if other.Has(Trapping) || other.Progress[Trapping] != 0 {
		t.Fatalf("locked technology accepted progress: %#v", other)
	}
	if math.IsNaN(state.CapacityMultiplier()) {
		t.Fatal("capacity multiplier is NaN")
	}
}
