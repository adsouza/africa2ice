package application

import (
	"github.com/adsouza/africa2ice/internal/domain"
	"github.com/adsouza/africa2ice/pkg/gameapi"
)

// ProjectSaveState restores and projects a validated save without installing
// it into a running service. It is used by the display-only maximum-render
// browser fixture; callers receive the same immutable boundary frame as an
// accepted load would publish.
func ProjectSaveState(state SaveState) (*gameapi.Frame, error) {
	world, err := state.RestoreWorld()
	if err != nil {
		return nil, err
	}
	return newProjectionService(world, state.WorldRevision).projectFrame()
}

// newProjectionService builds a service that only ever projects. It still
// installs a clock: pollAutosaveClock dereferences that field unconditionally,
// so leaving it nil would make the projection path one accidental method call
// away from a nil-interface panic.
func newProjectionService(world *domain.World, revision uint64) *GameService {
	return &GameService{world: world, worldRevision: revision, terrainRevision: 1, clock: systemMonotonicClock{}}
}

func (service *GameService) projectFrame() (*gameapi.Frame, error) {
	date, err := domain.CampaignDate(service.world.Turn())
	if err != nil {
		return nil, err
	}
	season, err := domain.SeasonForTurn(service.world.Turn())
	if err != nil {
		return nil, err
	}
	climate := service.world.Climate()
	frame := &gameapi.Frame{
		EasyMode:      service.world.EasyMode(),
		WorldRevision: service.worldRevision, TerrainRevision: service.terrainRevision,
		Turn: date.Turn, YearBP: date.YearBP, Era: mapEra(date.Era), CalendarProgress: date.CalendarProgress,
		Season: mapSeason(season), CampaignResult: mapResult(service.world.Result()),
		Climate: gameapi.ClimateSummary{
			LongTermTempOffset: climate.LongTermTempOffset, SeasonalTempOffset: climate.SeasonalTempOffset,
			ClimateNoise: climate.ClimateNoise, GlobalTempOffset: climate.GlobalTempOffset,
			MoistureOffset: climate.LongTermMoistureOffset, AridityIndex: climate.AridityIndex, Epoch: mapEpoch(climate.Epoch),
		},
	}
	frame.MacroEpisodes = append(frame.MacroEpisodes, gameapi.MacroEpisodeSummary{
		Episode: gameapi.CampanianIgnimbrite,
		Warned:  domain.MacroEpisodeWarned(date.Turn), Current: domain.MacroEpisodeActive(date.Turn),
		Elapsed: date.YearBP < domain.CampanianYearBP && !domain.MacroEpisodeActive(date.Turn),
	})
	for region := domain.Region(0); region < domain.RegionCount; region++ {
		frame.Climate.RegionalAbrupt[mapRegion(region)] = climate.RegionalAbrupt[region]
	}
	habitat, tileStates, grid := service.world.Habitat(), service.world.TileStates(), service.world.Grid()
	frame.Tiles = make([]gameapi.Tile, domain.TileCount)
	for id := range domain.TileCount {
		geography, _ := grid.Tile(domain.TileID(id))
		publicTile := gameapi.Tile{
			ID: gameapi.TileID(id), X: geography.X, Y: geography.Y, Latitude: geography.Latitude, Longitude: geography.Longitude,
			Land: geography.Land, ElevationKm: geography.ElevationKm, Explored: service.world.IsExplored(domain.TileID(id)),
			NaturalShelter: geography.NaturalShelter, BaseMoisture: geography.BaseMoisture,
			LastHabitableTurn: grid.LastHabitableTurn(domain.TileID(id)),
		}
		if publicTile.Explored && !geography.Land {
			publicTile.WaterBody = grid.WaterBodyName(domain.TileID(id))
		}
		if geography.Land {
			macroImpact := domain.MacroImpactAt(geography, date.Turn)
			publicTile.Region, publicTile.Biome = mapRegion(geography.Region), mapBiome(habitat[id].Biome)
			publicTile.LocalTemperatureC, publicTile.MovementCost = habitat[id].LocalTemperatureC, habitat[id].MovementCost
			publicTile.VegetationIndex, publicTile.BaselineK = habitat[id].VegetationIndex, habitat[id].BaselineK
			publicTile.Degradation = tileStates[id].Degradation
			publicTile.EcologicalK = habitat[id].BaselineK * (1 - tileStates[id].Degradation) * macroImpact.HabitatFactor
			caps := domain.ResourceCaps(habitat[id].Biome, season, tileStates[id].Degradation, habitat[id].BaselineK)
			caps.Flora *= macroImpact.FloraFactor
			caps.Fauna *= macroImpact.FaunaFactor
			caps.Water *= macroImpact.WaterFactor
			publicTile.FloraStock, publicTile.FloraCap = tileStates[id].Stock.Flora, caps.Flora
			publicTile.FaunaStock, publicTile.FaunaCap = tileStates[id].Stock.Fauna, caps.Fauna
			publicTile.WaterStock, publicTile.WaterCap = tileStates[id].Stock.Water, caps.Water
			profile, _ := domain.FaunaFor(geography.Region, habitat[id].Biome, habitat[id].BaselineK > 0)
			for group := domain.FaunaGroup(0); group < domain.FaunaGroupCount; group++ {
				publicTile.Fauna.Weights[mapFaunaGroup(group)] = profile.Weights[group]
			}
			publicTile.Fauna.HuntingSupported, publicTile.Fauna.MegafaunaSupported = profile.HuntingSupported, profile.MegafaunaSupported
			if publicTile.Explored && macroImpact.Active && macroImpact.Zone != domain.MacroUnaffected {
				publicTile.VisibleMacroImpact = gameapi.MacroImpactSummary{Visible: true, Episode: gameapi.CampanianIgnimbrite, ResourceFactor: macroImpact.FloraFactor, HabitatFactor: macroImpact.HabitatFactor}
			}
		}
		frame.Tiles[id] = publicTile
	}

	for region := domain.Region(0); region < domain.RegionCount; region++ {
		if service.world.EstablishedRegions()&(1<<region) != 0 {
			frame.SapiensEstablishedRegions = append(frame.SapiensEstablishedRegions, mapRegion(region))
		}
	}
	for _, edge := range grid.Escarpments() {
		if !service.world.IsExplored(edge.First) || !service.world.IsExplored(edge.Second) {
			continue
		}
		frame.Escarpments = append(frame.Escarpments, gameapi.Escarpment{
			Name: edge.Name, First: gameapi.TileID(edge.First), Second: gameapi.TileID(edge.Second),
		})
	}
	for _, passage := range domain.Passages() {
		status := gameapi.PassageOpen
		if passage.ClimateGated && !domain.BeringiaOpen(climate.LongTermTempOffset) {
			status = gameapi.PassageLocked
		}
		frame.Passages = append(frame.Passages, gameapi.Passage{
			ID: gameapi.PassageID(passage.ID), From: gameapi.TileID(passage.From), To: gameapi.TileID(passage.To),
			Cost: passage.Cost, Status: status, Explored: service.world.IsExplored(passage.From) || service.world.IsExplored(passage.To),
		})
	}
	// Lake labels describe fixed geographic anchors, not their starting bands.
	// Rebuild them for every frame, including loaded games.
	for index, anchor := range domain.StartingAnchors {
		id := domain.StartingTileIDs[index]
		if frame.Tiles[id].Explored {
			frame.Tiles[id].NearbyLake = anchor.NearbyLake
		}
	}
	projectLakes(frame)
	for _, event := range service.world.Events() {
		frame.Events = append(frame.Events, gameapi.Event{Turn: event.Turn, Kind: mapEventKind(event.Kind), BandID: gameapi.BandID(event.BandID), TileID: gameapi.TileID(event.TileID), Region: mapRegion(event.Region), Summary: event.Summary})
	}
	allBands := service.world.Bands()
	migrationCandidates := service.world.MigrationCandidatesByBand()
	for bandIndex, band := range allBands {
		publicBand := gameapi.Band{
			ID: gameapi.BandID(band.ID), Species: mapSpecies(band.Species), TileID: gameapi.TileID(band.TileID),
			Population: uint32(band.Population), Health: float64(band.Health), StoredFood: float64(band.StoredFood),
			AcquiredTech: band.Technology.Acquired, HasResearchTarget: band.Technology.HasTarget,
			SpatialActionUsed: band.SpatialActionUsed, QueuedMigration: gameapi.TileID(band.QueuedMigration), HasQueuedMigration: band.HasQueuedMigration,
			HasInterbreedTarget: band.HasInterbreedTarget, InterbreedTargetID: gameapi.BandID(band.InterbreedTarget),
			Stress:         service.world.BandStress(band.ID),
			LastFoodReport: gameapi.FoodTurnReport{Turn: band.LastFoodReport.Turn, RequiredFU: band.LastFoodReport.RequiredFU, DeficitFU: band.LastFoodReport.DeficitFU},
			LastMortality:  gameapi.MortalityReport{Starvation: band.LastMortality.Starvation, Seasonal: band.LastMortality.Seasonal, Chronic: band.LastMortality.Chronic, Macro: band.LastMortality.Macro, Acute: band.LastMortality.Acute},
			LastOutcomeReport: gameapi.OutcomeReport{
				Turn: band.LastOutcomeReport.Turn, StartingPopulation: uint32(band.LastOutcomeReport.StartingPopulation), EndingPopulation: uint32(band.LastOutcomeReport.EndingPopulation), Growth: band.LastOutcomeReport.Growth,
				StartingHealth: float64(band.LastOutcomeReport.StartingHealth), EndingHealth: float64(band.LastOutcomeReport.EndingHealth), NutritionDelta: band.LastOutcomeReport.NutritionDelta,
				WaterHealthLoss: band.LastOutcomeReport.WaterHealthLoss, DiseaseHealthLoss: band.LastOutcomeReport.DiseaseHealthLoss, GeneticBurdenHealthLoss: band.LastOutcomeReport.GeneticBurdenHealthLoss,
				MacroHealthLoss: band.LastOutcomeReport.MacroHealthLoss, AcuteDiseaseHealthLoss: band.LastOutcomeReport.AcuteDiseaseHealthLoss,
			},
		}
		geography, _ := grid.Tile(band.TileID)
		publicBand.SeasonalMortalityRate, publicBand.ChronicMortalityRate = domain.Phase3MortalityRates(band, geography, habitat[band.TileID], season)
		for index := range publicBand.AllocationBP {
			publicBand.AllocationBP[index] = uint16(band.Allocation[index])
		}
		for technology := domain.Technology(0); technology < domain.TechCount; technology++ {
			publicTechnology := mapTech(technology)
			publicBand.ResearchProgress[publicTechnology] = band.Technology.Progress[technology]
			acquired := band.Technology.Has(technology)
			publicBand.ResearchOptions[publicTechnology] = gameapi.ResearchOption{
				Available:        !acquired && band.Technology.PrerequisitesMet(technology),
				Acquired:         acquired,
				Current:          band.Technology.HasTarget && band.Technology.Target == technology,
				Cost:             domain.ResearchCost[technology],
				PrerequisiteMask: domain.TechnologyPrerequisiteMask(technology),
			}
		}
		if band.Technology.HasTarget {
			publicBand.ResearchTarget = mapTech(band.Technology.Target)
			publicBand.OriginalResearchGainPreview = domain.ResearchGain(band.Workers(domain.Toolcraft))
		}
		for trait := domain.HeritableTrait(0); trait < domain.HeritableTraitCount; trait++ {
			publicBand.HeritableState[mapTrait(trait)] = float64(band.Heritable[trait])
		}
		for passageID := domain.PassageID(0); passageID < domain.PassageCount; passageID++ {
			status := service.world.PassageStatus(band.ID, passageID)
			switch status {
			case domain.PassageAvailable:
				publicBand.PassageStatuses[passageID] = gameapi.PassageOpen
			case domain.PassageNotAtEndpoint, domain.PassageDestinationUninhabitable:
				publicBand.PassageStatuses[passageID] = gameapi.PassageUnavailable
			default:
				publicBand.PassageStatuses[passageID] = gameapi.PassageLocked
			}
		}
		for _, candidate := range migrationCandidates[bandIndex].Candidates {
			// The domain hands archaic bands candidates on unexplored tiles on
			// purpose, so the computer policy can route them; its own filter is
			// HomoSapiens-only. Nothing downstream of this frame is, though --
			// the map draws a highlight per candidate and the panel derives
			// affordances from them, none of it species-aware -- so a hidden
			// target reaching the frame is a fog leak in every one of those
			// readers at once. Drop it here rather than at each of them.
			if !service.world.IsExplored(candidate.TileID) {
				continue
			}
			destination, _ := grid.Tile(candidate.TileID)
			seasonalMortalityRate, chronicMortalityRate := domain.Phase3MortalityRates(band, destination, habitat[candidate.TileID], season)
			publicBand.MigrationCandidates = append(publicBand.MigrationCandidates, gameapi.MigrationCandidate{
				TileID: gameapi.TileID(candidate.TileID), Cost: candidate.Cost, Attraction: candidate.Attraction,
				EcologicalK: candidate.EcologicalK, UsableFoodEquivalent: candidate.UsableFoodEquivalent,
				WaterSurvivalEquivalent: candidate.WaterSurvivalEquivalent, DestinationPopulation: candidate.DestinationPopulation,
				WarningSuitability:    candidate.WarningSuitability,
				SeasonalMortalityRate: seasonalMortalityRate, ChronicMortalityRate: chronicMortalityRate,
				CrowdingDecline: candidate.CrowdingDecline,
				Passage:         gameapi.PassageID(candidate.Passage),
				RequiresPassage: candidate.RequiresPassage,
			})
		}
		// Only a sapiens actor may interbreed, and only with a co-located
		// archaic band. Offering an archaic band a candidate list would
		// advertise an action the domain always rejects.
		if band.Species == domain.HomoSapiens {
			for _, other := range allBands {
				if other.ID != band.ID && other.TileID == band.TileID && other.Species == domain.ArchaicHominin {
					publicBand.InterbreedCandidateIDs = append(publicBand.InterbreedCandidateIDs, gameapi.BandID(other.ID))
				}
			}
		}
		// The split action chooses the first ordinary migration candidate. Use
		// the origin when none exists so the domain supplies the rejection in
		// its normal guard order, without duplicating eligibility rules here.
		splitDestination := band.TileID
		for _, candidate := range publicBand.MigrationCandidates {
			if !candidate.RequiresPassage {
				splitDestination = domain.TileID(candidate.TileID)
				break
			}
		}
		if err := service.world.ValidateSplit(band.ID, splitDestination, true); err != nil {
			publicBand.SplitBlock = domainErrorCode(err)
		}
		frame.Bands = append(frame.Bands, publicBand)
	}
	return frame, nil
}
