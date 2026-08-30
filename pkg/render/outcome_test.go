package render

import (
	"math"
	"reflect"
	"testing"

	"github.com/adsouza/africa2ice/pkg/gameapi"
)

func TestSummarizeBandOutcomeRanksIndependentPopulationAndHealthCauses(t *testing.T) {
	band := &gameapi.Band{
		LastMortality: gameapi.MortalityReport{Starvation: 1, Seasonal: 4, Chronic: 2},
		LastOutcomeReport: gameapi.OutcomeReport{
			Turn: 3, StartingPopulation: 100, EndingPopulation: 96, Growth: 3,
			StartingHealth: 1, EndingHealth: 0.97, NutritionDelta: 0.05,
			WaterHealthLoss: 0.02, DiseaseHealthLoss: 0.06, GeneticBurdenHealthLoss: 0.001,
		},
	}
	summary := summarizeBandOutcome(band)
	if !summary.available || summary.populationDelta != -4 || math.Abs(summary.healthDeltaPoints-(-3)) > 1e-12 {
		t.Fatalf("summary = %#v", summary)
	}
	populationLabels := []string{summary.populationLossCauses[0].label, summary.populationLossCauses[1].label, summary.populationLossCauses[2].label}
	if !reflect.DeepEqual(populationLabels, []string{"seasonal hazards", "chronic hazards", "starvation"}) {
		t.Fatalf("population causes = %v", populationLabels)
	}
	healthLabels := []string{summary.healthLossCauses[0].label, summary.healthLossCauses[1].label, summary.healthLossCauses[2].label}
	if !reflect.DeepEqual(healthLabels, []string{"endemic disease", "water shortage", "adaptation trade-offs"}) {
		t.Fatalf("health causes = %v", healthLabels)
	}
	if got := formatOutcomeCauses(summary.healthLossCauses, 2); got != "endemic disease, water shortage, +1 more" {
		t.Fatalf("formatted causes = %q", got)
	}
}

func TestSummarizeBandOutcomeIsUnavailableWithoutCompletedTurn(t *testing.T) {
	if got := summarizeBandOutcome(&gameapi.Band{}); got.available {
		t.Fatalf("zero report summarized as available: %#v", got)
	}
}

// The crowding decline is the one population loss the domain does not record as
// a mortality cause: it arrives as a negative Growth, and every MortalityReport
// field reads zero while it happens. This summary is therefore the only place a
// player is told about it, and the branch that produces it was previously
// untested.
func TestSummarizeBandOutcomeAttributesCrowdingDecline(t *testing.T) {
	// Taken from a traced collapse: a band of 112 on a tile whose effective
	// capacity had fallen far below it, losing the bounded maximum with no
	// mortality cause recorded at all.
	band := &gameapi.Band{
		LastMortality: gameapi.MortalityReport{},
		LastOutcomeReport: gameapi.OutcomeReport{
			Turn: 143, StartingPopulation: 112, EndingPopulation: 84, Growth: -28,
			StartingHealth: 1, EndingHealth: 1,
		},
	}
	summary := summarizeBandOutcome(band)
	if !summary.available || summary.populationDelta != -28 {
		t.Fatalf("summary = %#v", summary)
	}
	if len(summary.populationLossCauses) != 1 {
		t.Fatalf("population causes = %#v, want exactly the crowding cause", summary.populationLossCauses)
	}
	if got := summary.populationLossCauses[0]; got.label != "crowding / habitat limits" || math.Abs(got.magnitude-28) > 1e-12 {
		t.Fatalf("cause = %#v", got)
	}
	// Without the branch the player would be told "demographic pressure", which
	// names nothing and is what the fallback exists for.
	if got := formatOutcomeCauses(summary.populationLossCauses, 2); got != "crowding / habitat limits" {
		t.Fatalf("formatted = %q", got)
	}
}
