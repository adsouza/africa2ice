package domain

const MinEstablishedBand Population = 20

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
	nextTurn := world.turn + 1
	nextHabitat, nextClimate, err := BuildHabitat(world.grid, world.seed, nextTurn)
	if err != nil {
		return err
	}
	planning, err := world.planArchaic()
	if err != nil {
		return err
	}
	nextBands := planning.bands
	nextBandID := planning.nextBandID
	nextTiles := world.tiles
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
		caps.Flora *= macroImpacts[id].FloraFactor
		caps.Fauna *= macroImpacts[id].FaunaFactor
		caps.Water *= macroImpacts[id].WaterFactor
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
			work[index].acuteRisk[AcutePredation] += 0.04 * huntingShare
			work[index].acuteRisk[AcuteExposureFall] += 0.01 * huntingShare
		}
		if work[index].megafaunaDemand > 0 {
			work[index].acuteRisk[AcutePredation] += 0.10 * megafaunaShare
			work[index].acuteRisk[AcuteExposureFall] += 0.04 * megafaunaShare
		}
	}

	for tileID := range TileCount {
		indices := make([]int, 0, 4)
		for index, band := range nextBands {
			if int(band.TileID) == tileID && band.Population > 0 {
				indices = append(indices, index)
			}
		}
		if len(indices) == 0 {
			continue
		}
		floraDemands, waterDemands := make([]float64, len(indices)), make([]float64, len(indices))
		faunaDemands := make([]float64, 0, len(indices)*2)
		for offset, index := range indices {
			floraDemands[offset], waterDemands[offset] = work[index].floraDemand, work[index].waterDemand
			faunaDemands = append(faunaDemands, work[index].huntingDemand, work[index].megafaunaDemand)
		}
		floraAllocated := ProportionalAllocate(nextTiles[tileID].Stock.Flora, floraDemands)
		faunaAllocated := ProportionalAllocate(nextTiles[tileID].Stock.Fauna, faunaDemands)
		waterAllocated := ProportionalAllocate(nextTiles[tileID].Stock.Water, waterDemands)
		floraUsed, faunaUsed, waterUsed := 0.0, 0.0, 0.0
		for offset, index := range indices {
			work[index].floraAllocated = floraAllocated[offset]
			work[index].huntingAllocated = faunaAllocated[offset*2]
			work[index].megafaunaAllocated = faunaAllocated[offset*2+1]
			work[index].waterAllocated = waterAllocated[offset]
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
		if band.Technology.HasTarget && !band.Technology.Has(band.Technology.Target) && band.Technology.PrerequisitesMet(band.Technology.Target) {
			work[index].researchGain = ResearchGain(band.Workers(Toolcraft))
			remaining := ResearchCost[band.Technology.Target] - band.Technology.Progress[band.Technology.Target]
			if work[index].researchGain > remaining {
				work[index].researchGain = remaining
			}
		}
		work[index].selection = SelectionDeltas(*band, geography, nextHabitat[band.TileID], season, work[index].animalFoodShare)
		effectiveK := float64(float64(nextHabitat[band.TileID].BaselineK*(1-nextTiles[band.TileID].Degradation)) * band.Technology.CapacityMultiplier())
		effectiveK *= macroImpacts[band.TileID].HabitatFactor
		growth := LogisticGrowth(startPopulation, float64(populationByTile[band.TileID]), effectiveK, deficitFraction)
		band.LastOutcomeReport.Growth = growth
		grownPopulation := startPopulation + growth
		if grownPopulation < 0 {
			grownPopulation = 0
		}
		seasonalRate, chronicRate := Phase3MortalityRates(*band, geography, nextHabitat[band.TileID], season)
		rawStarvation := StarvationLoss(startPopulation, deficitFraction)
		rawSeasonal := startPopulation * seasonalRate
		rawChronic := startPopulation * chronicRate
		rawMortality := rawStarvation + rawSeasonal + rawChronic
		mortalityScale := 1.0
		if rawMortality > grownPopulation && rawMortality > 0 {
			mortalityScale = grownPopulation / rawMortality
		}
		starvation := rawStarvation * mortalityScale
		seasonalLoss := rawSeasonal * mortalityScale
		chronicLoss := rawChronic * mortalityScale
		population := grownPopulation - starvation - seasonalLoss - chronicLoss
		if population < 0 {
			population = 0
		}
		band.Population, err = RoundPopulation(population)
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
			macroLoss := before * impact.LossFraction
			band.Population, err = RoundPopulation(before - macroLoss)
			if err != nil {
				return err
			}
			band.Health = Health(clamp01(float64(band.Health) - impact.HealthLoss))
			band.LastMortality.Macro = before - float64(band.Population)
			band.LastOutcomeReport.MacroHealthLoss = float64(healthBefore - band.Health)
		}
		healthBeforeAcute := band.Health
		_, loss, occurred, err := ResolveAcute(band, geography, nextHabitat[band.TileID], season, work[index].acuteRisk, work[index].crossed, work[index].crossedPassage, world.rng)
		if err != nil {
			return err
		}
		if occurred {
			band.LastMortality.Acute = loss
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
	live := nextBands[:0]
	for _, band := range nextBands {
		if band.Population > 0 {
			live = append(live, band)
		}
	}
	nextBands = live
	applyKnowledgeAndGenetics(nextBands, world.grid, researchGains, selectionDeltas, world.rng)
	for index := range nextBands {
		nextBands[index].HasInterbreedTarget = false
	}
	result := CampaignOngoing
	hasSapiens := false
	established := world.establishedRegions
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
	if !hasSapiens {
		result = CampaignExtinction
	} else if nextTurn == MaxCampaignTurn {
		destinations := uint16(1<<Frangistan | 1<<SouthAsia | 1<<YellowRiverBasin | 1<<Sahul | 1<<Beringia)
		if established&destinations != 0 {
			result = CampaignVictory
		} else {
			result = CampaignDispersalFailed
		}
	}

	world.turn, world.result, world.bands, world.tiles = nextTurn, result, nextBands, nextTiles
	world.nextBandID = nextBandID
	world.exploredTiles = planning.exploredTiles
	world.habitat, world.climate, world.establishedRegions = nextHabitat, nextClimate, established
	world.revealFromSapiens()
	return world.validate()
}
