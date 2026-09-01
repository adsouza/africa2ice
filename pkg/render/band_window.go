package render

import (
	"fmt"

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
		return fmt.Sprintf("Bands: %d", window.total)
	}
	return fmt.Sprintf("Bands %d–%d/%d", window.first+1, window.first+window.count, window.total)
}

func visibleSapiensBandWindow(bands []gameapi.Band, selectedBand gameapi.BandID) sapiensBandWindow {
	selectedOrdinal := -1
	window := sapiensBandWindow{}
	for _, band := range bands {
		if band.Species != gameapi.HomoSapiens {
			continue
		}
		if band.ID == selectedBand {
			selectedOrdinal = window.total
		}
		window.total++
	}
	if selectedOrdinal >= 0 {
		window.first = selectedOrdinal / maxVisibleSapiensBandRows * maxVisibleSapiensBandRows
	}

	ordinal := 0
	for index, band := range bands {
		if band.Species != gameapi.HomoSapiens {
			continue
		}
		if ordinal >= window.first && ordinal < window.first+maxVisibleSapiensBandRows {
			window.indices[window.count] = index
			window.count++
		}
		ordinal++
		if window.count == maxVisibleSapiensBandRows {
			break
		}
	}
	return window
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
