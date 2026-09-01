package render

import (
	"fmt"
	"sort"

	"github.com/adsouza/africa2ice/pkg/gameapi"
)

const maxVisibleSapiensBandRows = 5

const (
	bandActionReady      = "READY"
	bandActionMoveSet    = "MOVE SET"
	bandActionInterbreed = "INTERBREED"
	bandActionDone       = "DONE"

	bandSufferingHealthThreshold = 0.50
	bandDangerHealthThreshold    = 0.80
	bandDangerMortalityRate      = 0.004
)

type sapiensBandCondition uint8

const (
	bandConditionStable sapiensBandCondition = iota
	bandConditionDanger
	bandConditionSuffering
)

// sapiensBandWindow is a bounded, allocation-free page into Frame.Bands. The
// frame may interleave computer-controlled archaic bands, so the stored values
// are source-slice indices rather than a contiguous range.
type sapiensBandWindow struct {
	indices [maxVisibleSapiensBandRows]int
	count   int
	first   int
	total   int
}

func (window sapiensBandWindow) label() string {
	if window.total <= maxVisibleSapiensBandRows {
		return fmt.Sprintf("Priority bands: %d", window.total)
	}
	return fmt.Sprintf("Priority %d–%d/%d", window.first+1, window.first+window.count, window.total)
}

func visibleSapiensBandWindow(bands []gameapi.Band, selectedBand gameapi.BandID) sapiensBandWindow {
	orderedIndices := sapiensBandIndicesByAttention(bands)
	selectedOrdinal := -1
	window := sapiensBandWindow{total: len(orderedIndices)}
	for ordinal, index := range orderedIndices {
		band := bands[index]
		if band.ID == selectedBand {
			selectedOrdinal = ordinal
			break
		}
	}
	if selectedOrdinal >= 0 {
		window.first = selectedOrdinal / maxVisibleSapiensBandRows * maxVisibleSapiensBandRows
	}

	end := min(window.first+maxVisibleSapiensBandRows, len(orderedIndices))
	for _, index := range orderedIndices[window.first:end] {
		window.indices[window.count] = index
		window.count++
	}
	return window
}

// SapiensBandIDsByAttention returns living player bands in the same
// presentation-only order used by the HUD. Callers may use the order for
// selection navigation; simulation and persistence never read it.
func SapiensBandIDsByAttention(bands []gameapi.Band) []gameapi.BandID {
	indices := sapiensBandIndicesByAttention(bands)
	bandIDs := make([]gameapi.BandID, len(indices))
	for ordinal, index := range indices {
		bandIDs[ordinal] = bands[index].ID
	}
	return bandIDs
}

func sapiensBandIndicesByAttention(bands []gameapi.Band) []int {
	indices := make([]int, 0, len(bands))
	for index, band := range bands {
		if band.Species == gameapi.HomoSapiens && band.Population > 0 {
			indices = append(indices, index)
		}
	}
	sort.Slice(indices, func(left, right int) bool {
		return sapiensBandNeedsMoreAttention(bands[indices[left]], bands[indices[right]])
	})
	return indices
}

func sapiensBandNeedsMoreAttention(left, right gameapi.Band) bool {
	leftCondition, rightCondition := conditionForSapiensBand(left), conditionForSapiensBand(right)
	if leftCondition != rightCondition {
		return leftCondition > rightCondition
	}
	if left.Health != right.Health {
		return left.Health < right.Health
	}
	leftFoodDeficit, rightFoodDeficit := left.LastFoodReport.DeficitFraction(), right.LastFoodReport.DeficitFraction()
	if leftFoodDeficit != rightFoodDeficit {
		return leftFoodDeficit > rightFoodDeficit
	}
	leftMortality := left.SeasonalMortalityRate + left.ChronicMortalityRate
	rightMortality := right.SeasonalMortalityRate + right.ChronicMortalityRate
	if leftMortality != rightMortality {
		return leftMortality > rightMortality
	}
	leftPopulationLoss, rightPopulationLoss := lastTurnPopulationLossFraction(left), lastTurnPopulationLossFraction(right)
	if leftPopulationLoss != rightPopulationLoss {
		return leftPopulationLoss > rightPopulationLoss
	}
	return left.ID < right.ID
}

func lastTurnPopulationLossFraction(band gameapi.Band) float64 {
	report := band.LastOutcomeReport
	if report.Turn == 0 || report.StartingPopulation == 0 || report.EndingPopulation >= report.StartingPopulation {
		return 0
	}
	return float64(report.StartingPopulation-report.EndingPopulation) / float64(report.StartingPopulation)
}

// sapiensBandActionLabel reports the accepted-frame planning state without
// guessing which action produced a bare SpatialActionUsed marker. Exact queued
// intents take precedence over the generic spent state they imply.
func sapiensBandActionLabel(band gameapi.Band) string {
	switch {
	case band.HasQueuedMigration:
		return bandActionMoveSet
	case band.HasInterbreedTarget:
		return bandActionInterbreed
	case band.SpatialActionUsed:
		return bandActionDone
	default:
		return bandActionReady
	}
}

// conditionForSapiensBand combines the latest completed-turn actuals with the
// current projected health and background mortality rates. It is a UI warning
// tier only; the simulation never reads these presentation thresholds.
func conditionForSapiensBand(band gameapi.Band) sapiensBandCondition {
	outcome := band.LastOutcomeReport
	declinedLastTurn := outcome.Turn != 0 &&
		(outcome.EndingPopulation < outcome.StartingPopulation || outcome.EndingHealth < outcome.StartingHealth-1e-9)
	shortOfFoodLastTurn := band.LastFoodReport.Turn != 0 && band.LastFoodReport.DeficitFU > 0
	if declinedLastTurn || shortOfFoodLastTurn || band.Health < bandSufferingHealthThreshold {
		return bandConditionSuffering
	}
	if band.Health < bandDangerHealthThreshold || band.SeasonalMortalityRate+band.ChronicMortalityRate >= bandDangerMortalityRate {
		return bandConditionDanger
	}
	return bandConditionStable
}
