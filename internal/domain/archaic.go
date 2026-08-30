package domain

import (
	"errors"
	"fmt"
	"sort"
)

var archaicTechPriority = [...]Technology{
	PlantKnowledge,
	HaftedTools,
	Firecraft,
	TailoredClothing,
	Campcraft,
	MedicinalKnowledge,
	CordageAndNets,
	Trapping,
	CoastalNavigation,
}

type archaicPlanningResult struct {
	bands         []Band
	nextBandID    BandID
	exploredTiles [ExplorationWordCount]uint64
}

func (world *World) planArchaic() (archaicPlanningResult, error) {
	scratch := *world
	scratch.bands = append([]Band(nil), world.bands...)
	ids := make([]BandID, 0, len(scratch.bands))
	for _, band := range scratch.bands {
		if band.Species == ArchaicHominin && band.Population > 0 {
			ids = append(ids, band.ID)
		}
	}
	sort.Slice(ids, func(left, right int) bool { return ids[left] < ids[right] })

	for _, id := range ids {
		index := scratch.bandIndex(id)
		if index < 0 {
			continue
		}
		band := scratch.bands[index]
		geography, _ := scratch.grid.Tile(band.TileID)
		assignment, ok := ArchaicAssignment(geography.Region, scratch.habitat[band.TileID].Biome)
		if !ok {
			return archaicPlanningResult{}, fmt.Errorf("%w: missing archaic assignment", ErrInvalidValue)
		}
		if err := scratch.SetAssignment(id, assignment, false); err != nil {
			return archaicPlanningResult{}, err
		}
		index = scratch.bandIndex(id)
		if !scratch.bands[index].Technology.HasTarget {
			for _, technology := range archaicTechPriority {
				state := scratch.bands[index].Technology
				if state.Has(technology) || !state.PrerequisitesMet(technology) {
					continue
				}
				if err := scratch.Research(id, technology, false); err != nil {
					return archaicPlanningResult{}, err
				}
				break
			}
		}

		candidates := scratch.MigrationCandidates(id)
		if len(candidates) == 0 {
			continue
		}
		index = scratch.bandIndex(id)
		band = scratch.bands[index]
		if MacroEpisodeWarned(scratch.turn) {
			origin, _ := scratch.grid.Tile(band.TileID)
			originSuitability := 1 - MacroImpactAt(origin, scratch.turn+1).Intensity
			for _, candidate := range candidates {
				if candidate.WarningSuitability > originSuitability {
					if err := scratch.QueueMigration(id, candidate.TileID, false); err != nil {
						return archaicPlanningResult{}, err
					}
					candidates = nil
					break
				}
			}
			if candidates == nil {
				continue
			}
		}
		if scratch.BandStress(id) <= SplitStressThreshold {
			continue
		}
		archaicCount := 0
		for _, resident := range scratch.bands {
			if resident.Species == ArchaicHominin && resident.Population > 0 {
				archaicCount++
			}
		}
		if archaicCount < MaxArchaicBands {
			for _, candidate := range candidates {
				if candidate.RequiresPassage {
					continue
				}
				err := scratch.Split(id, candidate.TileID, false)
				if err == nil {
					candidates = nil
					break
				}
				if !expectedArchaicSplitRejection(err) {
					return archaicPlanningResult{}, err
				}
				break
			}
			if candidates == nil {
				continue
			}
		}
		if err := scratch.QueueMigration(id, candidates[0].TileID, false); err != nil {
			return archaicPlanningResult{}, err
		}
	}
	return archaicPlanningResult{bands: scratch.bands, nextBandID: scratch.nextBandID, exploredTiles: scratch.exploredTiles}, nil
}

func expectedArchaicSplitRejection(err error) bool {
	return errors.Is(err, ErrSplitPopulationTooLow) || errors.Is(err, ErrBandLimitReached) ||
		errors.Is(err, ErrBandIDExhausted) || errors.Is(err, ErrSplitDestinationNotAdjacent) ||
		errors.Is(err, ErrSplitDestinationUninhabitable)
}
