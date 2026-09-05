package ui

import (
	"sort"

	"github.com/adsouza/africa2ice/pkg/gameapi"
)

// Presentation-only warning tiers (DESIGN.md "Sapiens band warning thresholds").
// The simulation never reads these.
const (
	bandSufferingHealthThreshold = 0.50
	bandDangerHealthThreshold    = 0.80
	bandDangerMortalityRate      = 0.004
)

type SapiensBandCondition uint8

const (
	BandStable SapiensBandCondition = iota
	BandDanger
	BandSuffering
)

// Marker is the text prefix a chip carries so the tier is never color-only.
func (condition SapiensBandCondition) Marker() string {
	switch condition {
	case BandDanger:
		return "!"
	case BandSuffering:
		return "!!"
	default:
		return ""
	}
}

func ConditionForSapiensBand(band gameapi.Band) SapiensBandCondition {
	outcome := band.LastOutcomeReport
	declinedLastTurn := outcome.Turn != 0 &&
		(outcome.EndingPopulation < outcome.StartingPopulation || outcome.EndingHealth < outcome.StartingHealth-1e-9)
	shortOfFoodLastTurn := band.LastFoodReport.Turn != 0 && band.LastFoodReport.DeficitFU > 0
	if declinedLastTurn || shortOfFoodLastTurn || band.Health < bandSufferingHealthThreshold {
		return BandSuffering
	}
	if band.Health < bandDangerHealthThreshold || band.SeasonalMortalityRate+band.ChronicMortalityRate >= bandDangerMortalityRate {
		return BandDanger
	}
	return BandStable
}

// MoveDone is the checklist predicate for the spatial action (spec §5.1). Chips
// and the Move row both read it, so they cannot disagree.
func MoveDone(band gameapi.Band) bool {
	return band.SpatialActionUsed || band.HasQueuedMigration || band.HasInterbreedTarget
}

// BandsNeedingMove counts living sapiens bands whose spatial action is open.
func BandsNeedingMove(bands []gameapi.Band) int {
	count := 0
	for _, band := range bands {
		if band.Species == gameapi.HomoSapiens && band.Population > 0 && !MoveDone(band) {
			count++
		}
	}
	return count
}

// SapiensBandIDsByAttention returns living player bands in the presentation-
// only attention order: suffering before danger before stable, then lower
// health, larger last-turn food deficit, higher projected mortality, larger
// proportional loss, ascending ID.
func SapiensBandIDsByAttention(bands []gameapi.Band) []gameapi.BandID {
	indices := sapiensBandIndicesByAttention(bands)
	ids := make([]gameapi.BandID, len(indices))
	for ordinal, index := range indices {
		ids[ordinal] = bands[index].ID
	}
	return ids
}

// SapiensBandOrdinal reports a band's zero-based position in attention order.
func SapiensBandOrdinal(bands []gameapi.Band, id gameapi.BandID) (int, bool) {
	for ordinal, bandID := range SapiensBandIDsByAttention(bands) {
		if bandID == id {
			return ordinal, true
		}
	}
	return 0, false
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
	leftCondition, rightCondition := ConditionForSapiensBand(left), ConditionForSapiensBand(right)
	if leftCondition != rightCondition {
		return leftCondition > rightCondition
	}
	if left.Health != right.Health {
		return left.Health < right.Health
	}
	leftDeficit, rightDeficit := left.LastFoodReport.DeficitFraction(), right.LastFoodReport.DeficitFraction()
	if leftDeficit != rightDeficit {
		return leftDeficit > rightDeficit
	}
	leftMortality := left.SeasonalMortalityRate + left.ChronicMortalityRate
	rightMortality := right.SeasonalMortalityRate + right.ChronicMortalityRate
	if leftMortality != rightMortality {
		return leftMortality > rightMortality
	}
	leftLoss, rightLoss := lastTurnPopulationLossFraction(left), lastTurnPopulationLossFraction(right)
	if leftLoss != rightLoss {
		return leftLoss > rightLoss
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
