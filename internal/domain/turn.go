package domain

import (
	"fmt"
	"sort"
)

// tileOccupant pairs a live band with the tile it stands on, so the
// subsistence pass can walk tiles without rescanning every band.
type tileOccupant struct {
	tile  TileID
	index int
}

type turnBandWork struct {
	floraDemand, huntingDemand, megafaunaDemand, waterDemand float64
	aquaticShare                                             float64
	floraAllocated, huntingAllocated, megafaunaAllocated     float64
	waterAllocated                                           float64
	animalFoodShare                                          float64
	researchGain                                             float64
	selection                                                HeritableState
	acuteRisk                                                [AcuteKindCount]float64
	crossedPassage                                           PassageID
	crossed                                                  bool
}

func (world *World) AdvanceTurn() error {
	if world.result != CampaignOngoing || world.turn >= MaxCampaignTurn {
		return ErrCampaignComplete
	}
	next := *world
	next.bands = append([]Band(nil), world.bands...)
	next.events = append([]Event(nil), world.events...)
	var err error
	next.rng, err = world.rng.clone()
	if err != nil {
		return err
	}
	if err := next.advanceTurn(); err != nil {
		return err
	}
	*world = next
	return nil
}

// advanceTurn mutates a private candidate. AdvanceTurn publishes it only after
// the complete transition passes world validation, so any failure leaves the
// live aggregate and its RNG position unchanged.
func (world *World) advanceTurn() error {
	nextTurn := world.turn + 1
	nextHabitat, nextClimate, err := BuildHabitat(world.grid, world.seed, nextTurn)
	if err != nil {
		return err
	}
	planning, err := world.planArchaicOwned()
	if err != nil {
		return err
	}
	nextBands := planning.bands
	nextBandID := planning.nextBandID
	nextTiles := &world.tiles
	season, _ := SeasonForTurn(nextTurn)

	for index := range nextBands {
		nextBands[index].StoredFood = FU(float64(nextBands[index].StoredFood) - float64(float64(nextBands[index].StoredFood)*FoodSpoilageRate))
	}

	populationByTile := [TileCount]uint64{}
	macroImpacts := [TileCount]MacroImpact{}
	for _, band := range nextBands {
		populationByTile[band.TileID] += uint64(band.Population)
	}
	for id := range TileCount {
		geography, _ := world.grid.Tile(TileID(id))
		macroImpacts[id] = MacroImpactAt(geography, nextTurn)
		if nextHabitat[id].BaselineK <= 0 {
			nextTiles[id] = TileState{}
			continue
		}
		nextTiles[id].Degradation = NextDegradation(nextTiles[id].Degradation, float64(populationByTile[id]), nextHabitat[id].BaselineK)
		caps := ResourceCaps(nextHabitat[id].Biome, season, nextTiles[id].Degradation, nextHabitat[id].BaselineK)
		caps.Flora = float64(caps.Flora * macroImpacts[id].FloraFactor)
		caps.Fauna = float64(caps.Fauna * macroImpacts[id].FaunaFactor)
		caps.Water = float64(caps.Water * macroImpacts[id].WaterFactor)
		nextTiles[id] = nextTiles[id].Regenerate(caps, ResourceVector{})
	}

	work := make([]turnBandWork, len(nextBands))
	for index, band := range nextBands {
		if band.Population <= 0 || nextHabitat[band.TileID].BaselineK <= 0 {
			continue
		}
		geography, _ := world.grid.Tile(band.TileID)
		profile, _ := FaunaFor(geography.Region, nextHabitat[band.TileID].Biome, true)
		hunting := HuntingRates(band, geography.Region, profile)
		work[index].floraDemand = float64(band.Workers(Foraging) * ForagingRate(band, nextHabitat[band.TileID].Biome))
		work[index].huntingDemand = float64(band.Workers(HuntingAndFishing) * hunting.Total)
		work[index].megafaunaDemand = float64(band.Workers(MegafaunaTracking) * MegafaunaRate(band, profile, hunting.Terrestrial))
		work[index].waterDemand = WaterRequired(band, nextHabitat[band.TileID].LocalTemperatureC)
		if hunting.Total > 0 {
			work[index].aquaticShare = hunting.Aquatic / hunting.Total
		}
		huntingShare := float64(band.Allocation[HuntingAndFishing]) / AllocationBasisPoints
		megafaunaShare := float64(band.Allocation[MegafaunaTracking]) / AllocationBasisPoints
		if hunting.Total > 0 {
			work[index].acuteRisk[AcutePredation] += float64(0.04 * huntingShare)
			work[index].acuteRisk[AcuteExposureFall] += float64(0.01 * huntingShare)
		}
		if work[index].megafaunaDemand > 0 {
			work[index].acuteRisk[AcutePredation] += float64(0.10 * megafaunaShare)
			work[index].acuteRisk[AcuteExposureFall] += float64(0.04 * megafaunaShare)
		}
	}

	// Bands are grouped by tile in a single pass. Asking every tile which
	// bands stood on it cost TileCount*len(nextBands) iterations a turn —
	// 1.57M at full occupancy — and each one bound a 360-byte Band by value,
	// so the scan moved well over half a gigabyte per turn to find at most a
	// handful of occupants per tile. It was also the map's most
	// alignment-sensitive loop: an eight-byte change to this function's stack
	// frame moved its cost by a factor of eight, which made unrelated edits
	// look like performance regressions. Tiles are still visited in ascending
	// order, so this pass is indistinguishable from the scan it replaces.
	occupants := make([]tileOccupant, 0, len(nextBands))
	for index := range nextBands {
		if nextBands[index].Population > 0 {
			occupants = append(occupants, tileOccupant{tile: nextBands[index].TileID, index: index})
		}
	}
	sort.Slice(occupants, func(left, right int) bool {
		if occupants[left].tile != occupants[right].tile {
			return occupants[left].tile < occupants[right].tile
		}
		return occupants[left].index < occupants[right].index
	})
	for start := 0; start < len(occupants); {
		tileID := occupants[start].tile
		end := start + 1
		for end < len(occupants) && occupants[end].tile == tileID {
			end++
		}
		group := occupants[start:end]
		start = end
		floraDemands, waterDemands := make([]float64, len(group)), make([]float64, len(group))
		faunaDemands := make([]float64, 0, len(group)*2)
		for offset, occupant := range group {
			floraDemands[offset], waterDemands[offset] = work[occupant.index].floraDemand, work[occupant.index].waterDemand
			faunaDemands = append(faunaDemands, work[occupant.index].huntingDemand, work[occupant.index].megafaunaDemand)
		}
		floraAllocated := ProportionalAllocate(nextTiles[tileID].Stock.Flora, floraDemands)
		faunaAllocated := ProportionalAllocate(nextTiles[tileID].Stock.Fauna, faunaDemands)
		waterAllocated := ProportionalAllocate(nextTiles[tileID].Stock.Water, waterDemands)
		floraUsed, faunaUsed, waterUsed := 0.0, 0.0, 0.0
		for offset, occupant := range group {
			work[occupant.index].floraAllocated = floraAllocated[offset]
			work[occupant.index].huntingAllocated = faunaAllocated[offset*2]
			work[occupant.index].megafaunaAllocated = faunaAllocated[offset*2+1]
			work[occupant.index].waterAllocated = waterAllocated[offset]
			floraUsed += floraAllocated[offset]
			faunaUsed += faunaAllocated[offset*2] + faunaAllocated[offset*2+1]
			waterUsed += waterAllocated[offset]
		}
		nextTiles[tileID].Stock.Flora -= floraUsed
		nextTiles[tileID].Stock.Fauna -= faunaUsed
		nextTiles[tileID].Stock.Water -= waterUsed
		if nextTiles[tileID].Stock.Flora < 0 {
			nextTiles[tileID].Stock.Flora = 0
		}
		if nextTiles[tileID].Stock.Fauna < 0 {
			nextTiles[tileID].Stock.Fauna = 0
		}
		if nextTiles[tileID].Stock.Water < 0 {
			nextTiles[tileID].Stock.Water = 0
		}
	}

	// Counted before phase 3 changes any population, so a band's kin bonus
	// reflects start-of-turn neighbours rather than whichever bands this loop
	// happens to have already resolved.
	kinContacts := kinContactCounts(nextBands, world.grid)

	for index := range nextBands {
		band := &nextBands[index]
		if band.Population <= 0 || nextHabitat[band.TileID].BaselineK <= 0 {
			band.Population = 0
			continue
		}
		startingPopulation := band.Population
		startPopulation := float64(startingPopulation)
		startingHealth := band.Health
		ordinaryCatch := work[index].huntingAllocated - float64(work[index].huntingAllocated*work[index].aquaticShare)
		aquaticCatch := work[index].huntingAllocated - ordinaryCatch
		metabolism := float64(band.Heritable[FattyAcidMetabolism])
		freshFood := float64(work[index].floraAllocated * FattyAcidConversion(PlantFood, metabolism))
		freshFood += float64((ordinaryCatch + work[index].megafaunaAllocated) * FattyAcidConversion(AnimalFood, metabolism))
		freshFood += float64(aquaticCatch * FattyAcidConversion(AquaticFood, metabolism))
		availableFood := float64(band.StoredFood) + freshFood
		foodConsumed, remainingFood, deficit, deficitFraction := FoodDeficit(startPopulation, availableFood)
		if foodConsumed > 0 && availableFood > 0 {
			animalFood := float64((ordinaryCatch + work[index].megafaunaAllocated) * FattyAcidConversion(AnimalFood, metabolism))
			aquaticFood := float64(aquaticCatch * FattyAcidConversion(AquaticFood, metabolism))
			work[index].animalFoodShare = clamp01((animalFood + aquaticFood) / availableFood)
		}
		band.StoredFood = FU(remainingFood)
		band.LastFoodReport = FoodTurnReport{Turn: nextTurn, RequiredFU: startPopulation, DeficitFU: deficit}
		waterDeficitFraction := 0.0
		if work[index].waterDemand > 0 {
			waterDeficitFraction = clamp01((work[index].waterDemand - work[index].waterAllocated) / work[index].waterDemand)
		}
		geography, _ := world.grid.Tile(band.TileID)
		diseaseHealthLoss, geneticHealthLoss := Phase3HealthLosses(*band, geography, nextHabitat[band.TileID], season)
		nutritionDelta := NutritionHealthDelta(deficitFraction)
		waterHealthLoss := float64(WaterHealthLossRate * waterDeficitFraction)
		health := float64(band.Health) + nutritionDelta - waterHealthLoss - diseaseHealthLoss - geneticHealthLoss
		band.Health = Health(clamp01(health))
		band.LastOutcomeReport = OutcomeReport{
			Turn: nextTurn, StartingPopulation: startingPopulation,
			StartingHealth: startingHealth, EndingHealth: band.Health,
			NutritionDelta: nutritionDelta, WaterHealthLoss: waterHealthLoss,
			DiseaseHealthLoss: diseaseHealthLoss, GeneticBurdenHealthLoss: geneticHealthLoss,
		}
		work[index].researchGain = band.Technology.PlannedResearchGain(band.Workers(Toolcraft))
		work[index].selection = SelectionDeltas(*band, geography, nextHabitat[band.TileID], season, work[index].animalFoodShare)
		effectiveK := float64(float64(nextHabitat[band.TileID].BaselineK*(1-nextTiles[band.TileID].Degradation)) * band.Technology.CapacityMultiplier())
		effectiveK = float64(effectiveK * macroImpacts[band.TileID].HabitatFactor)
		growth := LogisticGrowth(startPopulation, float64(populationByTile[band.TileID]), effectiveK, deficitFraction)
		band.LastOutcomeReport.Growth = growth
		grownPopulation := startPopulation + growth
		if grownPopulation < 0 {
			grownPopulation = 0
		}
		seasonalRate, chronicRate := Phase3MortalityRates(*band, geography, nextHabitat[band.TileID], season)
		rawStarvation := StarvationLoss(startPopulation, deficitFraction)
		rawSeasonal := float64(startPopulation * seasonalRate)
		rawChronic := float64(startPopulation * chronicRate)
		rawMortality := rawStarvation + rawSeasonal + rawChronic
		mortalityScale := 1.0
		if rawMortality > grownPopulation && rawMortality > 0 {
			mortalityScale = grownPopulation / rawMortality
		}
		starvation := float64(rawStarvation * mortalityScale)
		seasonalLoss := float64(rawSeasonal * mortalityScale)
		chronicLoss := float64(rawChronic * mortalityScale)
		population := grownPopulation - starvation - seasonalLoss - chronicLoss
		if population < 0 {
			population = 0
		}
		band.Population, err = RoundPopulation(population, world.rng)
		if err != nil {
			return err
		}
		band.LastMortality = MortalityReport{Starvation: starvation, Seasonal: seasonalLoss, Chronic: chronicLoss}
		band.LastOutcomeReport.EndingPopulation = band.Population
	}

	for index := range nextBands {
		band := &nextBands[index]
		if band.HasQueuedMigration && band.Population > 0 && nextHabitat[band.QueuedMigration].BaselineK > 0 {
			valid := band.TileID == band.QueuedOrigin
			if valid && band.QueueUsesPassage {
				if band.QueuedPassage >= PassageCount {
					valid = false
				} else {
					passage := passageCatalog[band.QueuedPassage]
					destination, atEndpoint := passageDestination(passage, band.TileID)
					valid = atEndpoint && destination == band.QueuedMigration && passageAvailability(*band, passage, nextHabitat, nextClimate.LongTermTempOffset, false) == PassageAvailable
				}
			} else if valid {
				valid = false
				for _, edge := range world.grid.OrdinaryEdges(band.TileID) {
					if edge.To == band.QueuedMigration {
						valid = true
						break
					}
				}
			}
			if valid {
				band.TileID = band.QueuedMigration
				geography, _ := world.grid.Tile(band.TileID)
				world.appendBandEvent(*band, Event{Turn: nextTurn, Kind: EventMigration, BandID: band.ID, TileID: band.TileID, Region: geography.Region, Summary: fmt.Sprintf("Band %d migrated.", band.ID)})
				if band.QueueUsesPassage {
					work[index].crossed, work[index].crossedPassage = true, band.QueuedPassage
				}
			}
		}
		band.HasQueuedMigration, band.QueueUsesPassage = false, false
		capacity := FoodStorageCapacity(band.Population)
		if float64(band.StoredFood) > capacity {
			band.StoredFood = FU(capacity)
		}
		band.SpatialActionUsed = false
	}

	for index := range nextBands {
		band := &nextBands[index]
		if band.Population <= 0 {
			continue
		}
		geography, _ := world.grid.Tile(band.TileID)
		impact := macroImpacts[band.TileID]
		if impact.Active && impact.Zone != MacroUnaffected {
			before := float64(band.Population)
			healthBefore := band.Health
			macroLoss := float64(before * impact.LossFraction)
			band.Population, err = RoundPopulation(before-macroLoss, world.rng)
			if err != nil {
				return err
			}
			band.Health = Health(clamp01(float64(band.Health) - impact.HealthLoss))
			band.LastMortality.Macro = before - float64(band.Population)
			band.LastOutcomeReport.MacroHealthLoss = float64(healthBefore - band.Health)
			world.appendBandEvent(*band, Event{Turn: nextTurn, Kind: EventMacroEpisode, BandID: band.ID, TileID: band.TileID, Region: geography.Region, Summary: fmt.Sprintf("Band %d was affected by the Campanian eruption.", band.ID)})
		}
		healthBeforeAcute := band.Health
		kind, loss, occurred, err := ResolveAcute(band, geography, nextHabitat[band.TileID], season, work[index].acuteRisk, work[index].crossed, work[index].crossedPassage, kinContacts[index], world.rng)
		if err != nil {
			return err
		}
		if occurred {
			band.LastMortality.Acute = loss
			world.appendBandEvent(*band, Event{Turn: nextTurn, Kind: EventAcuteIncident, BandID: band.ID, TileID: band.TileID, Region: geography.Region, Summary: fmt.Sprintf("Band %d suffered %s.", band.ID, acuteIncidentPhrase(kind))})
		}
		band.LastOutcomeReport.AcuteDiseaseHealthLoss = float64(healthBeforeAcute - band.Health)
		band.LastOutcomeReport.EndingPopulation = band.Population
		band.LastOutcomeReport.EndingHealth = band.Health
		capacity := FoodStorageCapacity(band.Population)
		if float64(band.StoredFood) > capacity {
			band.StoredFood = FU(capacity)
		}
	}

	researchGains := make(map[BandID]float64, len(nextBands))
	selectionDeltas := make(map[BandID]HeritableState, len(nextBands))
	for index, band := range nextBands {
		researchGains[band.ID] = work[index].researchGain
		selectionDeltas[band.ID] = work[index].selection
	}
	// This filter is the one place a band leaves the world, so it is also the
	// only place that can report the loss: without a line here a band the
	// player was watching simply stopped existing.
	live := nextBands[:0]
	for _, band := range nextBands {
		if band.Population > 0 {
			live = append(live, band)
			continue
		}
		geography, _ := world.grid.Tile(band.TileID)
		world.appendBandEvent(band, Event{Turn: nextTurn, Kind: EventExtinction, BandID: band.ID, TileID: band.TileID, Region: geography.Region, Summary: extinctionSummary(band)})
	}
	nextBands = live
	preGainBands := append([]Band(nil), nextBands...)
	completedInterbreeding := applyKnowledgeAndGenetics(nextBands, world.grid, researchGains, selectionDeltas, world.rng)
	for index := range nextBands {
		learned := nextBands[index].Technology.Acquired &^ preGainBands[index].Technology.Acquired
		for technology := Technology(0); technology < TechCount; technology++ {
			if learned&(1<<technology) != 0 {
				geography, _ := world.grid.Tile(nextBands[index].TileID)
				world.appendBandEvent(nextBands[index], Event{Turn: nextTurn, Kind: EventTechnology, BandID: nextBands[index].ID, TileID: nextBands[index].TileID, Region: geography.Region, Summary: fmt.Sprintf("Band %d learned %s.", nextBands[index].ID, technologyName(technology))})
			}
		}
		if completedInterbreeding[nextBands[index].ID] {
			geography, _ := world.grid.Tile(nextBands[index].TileID)
			world.appendBandEvent(nextBands[index], Event{Turn: nextTurn, Kind: EventInterbreeding, BandID: nextBands[index].ID, TileID: nextBands[index].TileID, Region: geography.Region, Summary: fmt.Sprintf("Band %d interbred with an archaic band.", nextBands[index].ID)})
		}
		nextBands[index].HasInterbreedTarget = false
	}
	result := CampaignOngoing
	hasSapiens := false
	previousEstablished := world.establishedRegions
	established := previousEstablished
	for _, band := range nextBands {
		if band.Species != HomoSapiens {
			continue
		}
		hasSapiens = true
		if band.Population >= MinEstablishedBand {
			geography, _ := world.grid.Tile(band.TileID)
			established |= 1 << geography.Region
		}
	}
	newlyEstablished := established &^ previousEstablished
	for region := Region(0); region < RegionCount; region++ {
		if newlyEstablished&(1<<region) != 0 {
			world.appendEvent(Event{Turn: nextTurn, Kind: EventAchievement, Region: region, Summary: fmt.Sprintf("Sapiens established %s.", region)})
		}
	}
	if !hasSapiens {
		result = CampaignExtinction
	} else if nextTurn == MaxCampaignTurn {
		if established&destinationMask() != 0 {
			result = CampaignVictory
		} else {
			result = CampaignDispersalFailed
		}
	}

	world.turn, world.result, world.bands = nextTurn, result, nextBands
	world.nextBandID = nextBandID
	world.exploredTiles = planning.exploredTiles
	world.habitat, world.climate, world.establishedRegions = nextHabitat, nextClimate, established
	world.revealFromSapiens()
	return world.validate()
}
