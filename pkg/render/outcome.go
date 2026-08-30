package render

import (
	"sort"
	"strconv"
	"strings"

	"github.com/adsouza/africa2ice/pkg/gameapi"
)

type outcomeCause struct {
	label     string
	magnitude float64
}

type bandOutcomeSummary struct {
	available            bool
	turn                 int
	populationDelta      int64
	healthDeltaPoints    float64
	populationLossCauses []outcomeCause
	healthLossCauses     []outcomeCause
}

// summarizeBandOutcome translates completed-turn actuals into presentation-
// safe trends and ranked causes. It never reconstructs simulation results.
func summarizeBandOutcome(band *gameapi.Band) bandOutcomeSummary {
	if band == nil || band.LastOutcomeReport.Turn == 0 {
		return bandOutcomeSummary{}
	}
	report := band.LastOutcomeReport
	summary := bandOutcomeSummary{
		available:         true,
		turn:              report.Turn,
		populationDelta:   int64(report.EndingPopulation) - int64(report.StartingPopulation),
		healthDeltaPoints: 100 * (report.EndingHealth - report.StartingHealth),
	}
	if summary.populationDelta < 0 {
		if report.Growth < 0 {
			summary.populationLossCauses = append(summary.populationLossCauses, outcomeCause{"crowding / habitat limits", -report.Growth})
		}
		mortality := []outcomeCause{
			{"starvation", band.LastMortality.Starvation},
			{"seasonal hazards", band.LastMortality.Seasonal},
			{"chronic hazards", band.LastMortality.Chronic},
			{"eruption", band.LastMortality.Macro},
			{"acute incident", band.LastMortality.Acute},
		}
		for _, cause := range mortality {
			if cause.magnitude > 0 {
				summary.populationLossCauses = append(summary.populationLossCauses, cause)
			}
		}
	}
	if summary.healthDeltaPoints < -1e-9 {
		health := []outcomeCause{
			{"food shortage", -report.NutritionDelta},
			{"water shortage", report.WaterHealthLoss},
			{"endemic disease", report.DiseaseHealthLoss},
			{"adaptation trade-offs", report.GeneticBurdenHealthLoss},
			{"eruption", report.MacroHealthLoss},
			{"disease outbreak", report.AcuteDiseaseHealthLoss},
		}
		for _, cause := range health {
			if cause.magnitude > 0 {
				summary.healthLossCauses = append(summary.healthLossCauses, cause)
			}
		}
	}
	sortCauses(summary.populationLossCauses)
	sortCauses(summary.healthLossCauses)
	return summary
}

func formatOutcomeCauses(causes []outcomeCause, limit int) string {
	if len(causes) == 0 {
		return "demographic pressure"
	}
	if limit <= 0 || limit > len(causes) {
		limit = len(causes)
	}
	labels := make([]string, limit)
	for index := range labels {
		labels[index] = causes[index].label
	}
	result := strings.Join(labels, ", ")
	if remaining := len(causes) - limit; remaining > 0 {
		result += ", +" + strconv.Itoa(remaining) + " more"
	}
	return result
}

func sortCauses(causes []outcomeCause) {
	sort.SliceStable(causes, func(left, right int) bool {
		return causes[left].magnitude > causes[right].magnitude
	})
}
