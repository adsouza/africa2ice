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
