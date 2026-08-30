package domain

import (
	"math"
	"sort"
)

const (
	MaxGridNeighbors       = 8
	MaxPassageEdgesPerTile = 2
)

type MigrationCandidate struct {
	TileID                  TileID
	Cost                    float64
	Attraction              float64
	EcologicalK             float64
	UsableFoodEquivalent    float64
	WaterSurvivalEquivalent float64
	DestinationPopulation   uint64
	WarningSuitability      float64
	// CrowdingDecline is the people the band would lose to the logistic crowding
	// term on its first turn at this destination, or zero if the tile has room.
	// It is a magnitude, not a signed growth value.
	CrowdingDecline float64
	Passage         PassageID
	RequiresPassage bool
}

func (world *World) MigrationCandidates(id BandID) []MigrationCandidate {
	index := world.bandIndex(id)
	if index < 0 {
		return nil
	}
	band := world.bands[index]
	populationByTile := [TileCount]uint64{}
	for _, resident := range world.bands {
		if resident.Population > 0 {
			populationByTile[resident.TileID] += uint64(resident.Population)
		}
	}
	result := make([]MigrationCandidate, 0, MaxGridNeighbors+MaxPassageEdgesPerTile)
	for _, edge := range world.grid.OrdinaryEdges(band.TileID) {
		if world.habitat[edge.To].BaselineK <= 0 || band.Species == HomoSapiens && !world.IsExplored(edge.To) {
			continue
		}
		result = append(result, world.scoreMigration(band, edge.To, float64(edge.StepLength*world.habitat[edge.To].MovementCost), populationByTile[edge.To], false, 0))
	}
	for _, passage := range passageCatalog {
		if passageAvailability(band, passage, world.habitat, world.climate.LongTermTempOffset, true) != PassageAvailable {
			continue
		}
		destination, _ := passageDestination(passage, band.TileID)
		if band.Species == HomoSapiens && !world.IsExplored(destination) {
			continue
		}
		result = append(result, world.scoreMigration(band, destination, passage.Cost, populationByTile[destination], true, passage.ID))
	}
	sort.Slice(result, func(left, right int) bool {
		if result[left].Attraction != result[right].Attraction {
			return result[left].Attraction > result[right].Attraction
		}
		return result[left].TileID < result[right].TileID
	})
	return result
}

func (world *World) scoreMigration(band Band, destination TileID, cost float64, destinationPopulation uint64, requiresPassage bool, passage PassageID) MigrationCandidate {
	habitat := world.habitat[destination]
	tile := world.tiles[destination]
	geography, _ := world.grid.Tile(destination)
	ecologicalK := float64(habitat.BaselineK * (1 - tile.Degradation))
	if ecologicalK < 0 {
		ecologicalK = 0
	}
	usableFood := usableFoodEquivalent(band, geography.Region, habitat.Biome, tile.Stock)
	waterDemand := float64(BaseWaterPerPerson * WaterDemandMultiplier(habitat.LocalTemperatureC, band.Heritable[AridClimateAdaptation]))
	waterSurvival := 0.0
	if waterDemand > 0 {
		waterSurvival = float64(tile.Stock.Water) / waterDemand
		if waterSurvival > ecologicalK {
			waterSurvival = ecologicalK
		}
	}
	warningSuitability := 1.0
	if MacroEpisodeWarned(world.turn) {
		warningSuitability -= MacroImpactAt(geography, world.turn+1).Intensity
	}
	resources := usableFood + waterSurvival
	attraction := 0.0
	denominator := float64(cost * (1 + float64(destinationPopulation)))
	if ecologicalK > 0 && resources > 0 && warningSuitability > 0 && denominator > 0 {
		attraction = float64(ecologicalK*resources*warningSuitability) / denominator
	}
	if math.IsNaN(attraction) || math.IsInf(attraction, 0) || attraction < 0 {
		attraction = 0
	}
	// What the crowding term will cost on arrival. The band joins whoever already
	// stands there, so the capacity it is measured against is the whole tile's,
	// matching the phase-3 rule that crowding uses the whole tile while growth
	// uses the band. The deficit fraction is zero because a non-positive logistic
	// result is never scaled by the fed fraction, so feeding cannot change it.
	arrivalK := float64(ecologicalK * band.Technology.CapacityMultiplier())
	crowdingDecline := 0.0
	arriving := float64(band.Population)
	if growth := LogisticGrowth(arriving, float64(destinationPopulation)+arriving, arrivalK, 0); growth < 0 {
		crowdingDecline = -growth
	}
	return MigrationCandidate{
		TileID: destination, Cost: cost, Attraction: attraction, EcologicalK: ecologicalK,
		UsableFoodEquivalent: usableFood, WaterSurvivalEquivalent: waterSurvival,
		DestinationPopulation: destinationPopulation, WarningSuitability: warningSuitability,
		CrowdingDecline: crowdingDecline,
		Passage:         passage, RequiresPassage: requiresPassage,
	}
}

func usableFoodEquivalent(band Band, region Region, biome Biome, stock ResourceVector) float64 {
	profile, ok := FaunaFor(region, biome, true)
	if !ok {
		return 0
	}
	terrestrialWeight, aquaticWeight := 0.0, 0.0
	for group := FaunaGroup(0); group < FaunaGroupCount; group++ {
		weight := profile.Weights[group]
		if weight <= 0 {
			continue
		}
		switch group {
		case SmallGame, MediumGame, LargeGame:
			if profile.HuntingSupported {
				terrestrialWeight += weight
			}
		case Megafauna:
			if profile.MegafaunaSupported {
				terrestrialWeight += weight
			}
		case InshoreAquatic:
			if profile.HuntingSupported {
				aquaticWeight += weight
			}
		case PelagicAquatic:
			if profile.HuntingSupported && band.Technology.Has(CordageAndNets) {
				aquaticWeight += weight
			}
		}
	}
	metabolism := float64(band.Heritable[FattyAcidMetabolism])
	plant := float64(stock.Flora * FattyAcidConversion(PlantFood, metabolism))
	animal := float64(float64(stock.Fauna) * terrestrialWeight * FattyAcidConversion(AnimalFood, metabolism))
	aquatic := float64(float64(stock.Fauna) * aquaticWeight * FattyAcidConversion(AquaticFood, metabolism))
	return plant + animal + aquatic
}
