package application

import (
	"fmt"

	"github.com/adsouza/africa2ice/internal/domain"
)

// MaximumRenderFixture expands an ordinary reference save into the locked
// maximum-state rendering workload. It exists for the checked-in performance
// fixture generator; gameplay never calls it.
func MaximumRenderFixture(state SaveState) (SaveState, error) {
	for index := range state.ExploredTiles {
		state.ExploredTiles[index] = ^uint64(0)
	}
	var sapiensTemplate, archaicTemplate BandSave
	var hasSapiens, hasArchaic bool
	archaicCount := 0
	for _, band := range state.Bands {
		if band.Species == uint8(domain.HomoSapiens) && !hasSapiens {
			sapiensTemplate, hasSapiens = band, true
		}
		if band.Species == uint8(domain.ArchaicHominin) {
			archaicCount++
			if !hasArchaic {
				archaicTemplate, hasArchaic = band, true
			}
		}
	}
	if !hasSapiens || !hasArchaic {
		return SaveState{}, fmt.Errorf("reference state lacks a species template")
	}
	nextID := state.NextBandID
	appendBand := func(template BandSave) {
		template.ID = nextID
		template.SpatialActionUsed = false
		template.HasQueuedMigration = false
		template.HasInterbreedTarget = false
		template.LastFoodReport = FoodReportSave{}
		template.LastMortality = MortalitySave{}
		template.LastOutcomeReport = OutcomeReportSave{}
		state.Bands = append(state.Bands, template)
		nextID++
	}
	for archaicCount < domain.MaxArchaicBands {
		appendBand(archaicTemplate)
		archaicCount++
	}
	for len(state.Bands) < domain.MaxBands {
		appendBand(sapiensTemplate)
	}
	state.NextBandID = nextID
	if _, err := state.RestoreWorld(); err != nil {
		return SaveState{}, fmt.Errorf("generated fixture does not restore: %w", err)
	}
	return state, nil
}
