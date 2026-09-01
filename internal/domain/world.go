package domain

import (
	"fmt"
	"math"
	"sort"
)

type World struct {
	seed               uint64
	turn               int
	result             CampaignResult
	nextBandID         BandID
	bands              []Band
	events             []Event
	tiles              [TileCount]TileState
	exploredTiles      [ExplorationWordCount]uint64
	establishedRegions uint16
	rng                *WorldRNG
	grid               *Grid
	habitat            *Habitat
	climate            ClimateState
}

func NewWorld(seed uint64) (*World, error) {
	grid, err := canonicalGrid()
	if err != nil {
		return nil, err
	}
	habitat, climate, err := BuildHabitat(grid, seed, 0)
	if err != nil {
		return nil, err
	}
	resolved, err := ResolveStartingTiles(grid, habitat)
	if err != nil {
		return nil, err
	}
	if resolved != StartingTileIDs {
		return nil, fmt.Errorf("%w: starting tile fixture changed", ErrInvalidValue)
	}
	world := &World{seed: seed, result: CampaignOngoing, nextBandID: BandID(len(StartingAnchors) + 1), rng: NewWorldRNG(seed), grid: grid, habitat: habitat, climate: climate}
	season, _ := SeasonForTurn(0)
	for id := range TileCount {
		geography, _ := grid.Tile(TileID(id))
		world.tiles[id] = InitialTileState(seed, geography, habitat[id], season)
	}
	sapiensAllocation := [AssignmentCount]AssignmentBP{3500, 3000, 1500, 500, 1500}
	for index, anchor := range StartingAnchors {
		allocation := sapiensAllocation
		if anchor.Species == ArchaicHominin {
			allocation, _ = ArchaicAssignment(anchor.Region, habitat[StartingTileIDs[index]].Biome)
		}
		traits, ok := StartingHeritableState(anchor.Species, anchor.Region)
		if !ok {
			return nil, fmt.Errorf("%w: missing starting traits", ErrInvalidValue)
		}
		world.bands = append(world.bands, Band{ID: BandID(index + 1), Species: anchor.Species, TileID: StartingTileIDs[index], Population: anchor.Population, Health: 1, Allocation: allocation, Heritable: traits})
	}
	world.revealInitialEastAfrica()
	world.revealFromSapiens()
	if err := world.validate(); err != nil {
		return nil, err
	}
	return world, nil
}

func RestoreWorld(state State) (*World, error) {
	grid, err := canonicalGrid()
	if err != nil {
		return nil, err
	}
	habitat, climate, err := BuildHabitat(grid, state.Seed, state.Turn)
	if err != nil {
		return nil, err
	}
	rng, err := RestoreWorldRNG(append([]byte(nil), state.RNGState...))
	if err != nil {
		return nil, fmt.Errorf("%w: RNG state: %v", ErrInvalidValue, err)
	}
	world := &World{
		seed: state.Seed, turn: state.Turn, result: state.Result, nextBandID: state.NextBandID,
		bands: append([]Band(nil), state.Bands...), events: append([]Event(nil), state.Events...), tiles: state.Tiles, exploredTiles: state.ExploredTiles,
		establishedRegions: state.EstablishedRegions, rng: rng, grid: grid, habitat: habitat, climate: climate,
	}
	if err := world.validate(); err != nil {
		return nil, err
	}
	return world, nil
}

func (world *World) ExportState() (State, error) {
	if err := world.validate(); err != nil {
		return State{}, err
	}
	rngState, err := world.rng.MarshalBinary()
	if err != nil {
		return State{}, err
	}
	return State{
		Seed: world.seed, Turn: world.turn, Result: world.result, NextBandID: world.nextBandID,
		Bands: append([]Band(nil), world.bands...), Events: append([]Event(nil), world.events...), Tiles: world.tiles, ExploredTiles: world.exploredTiles,
		EstablishedRegions: world.establishedRegions, RNGState: append([]byte(nil), rngState...),
	}, nil
}

func (world *World) Turn() int              { return world.turn }
func (world *World) Seed() uint64           { return world.seed }
func (world *World) Result() CampaignResult { return world.result }
func (world *World) Climate() ClimateState  { return world.climate }
func (world *World) Grid() *Grid            { return world.grid }
func (world *World) Habitat() Habitat {
	if world.habitat == nil {
		return Habitat{}
	}
	return *world.habitat
}
func (world *World) TileStates() [TileCount]TileState { return world.tiles }
func (world *World) Bands() []Band                    { return append([]Band(nil), world.bands...) }
func (world *World) Events() []Event                  { return append([]Event(nil), world.events...) }
func (world *World) EstablishedRegions() uint16       { return world.establishedRegions }

func (world *World) IsExplored(id TileID) bool {
	if id >= TileCount {
		return false
	}
	return world.exploredTiles[id/64]&(uint64(1)<<(id%64)) != 0
}

func (world *World) markExplored(id TileID) {
	if id < TileCount {
		world.exploredTiles[id/64] |= uint64(1) << (id % 64)
	}
}

func (world *World) revealInitialEastAfrica() {
	for id := range TileCount {
		geography, _ := world.grid.Tile(TileID(id))
		if geography.Land && geography.Region == EastAfrica {
			world.markExplored(TileID(id))
			for dy := -1; dy <= 1; dy++ {
				for dx := -1; dx <= 1; dx++ {
					neighbor, err := TileIDAt(geography.X+dx, geography.Y+dy)
					if err == nil {
						neighborGeo, _ := world.grid.Tile(neighbor)
						if !neighborGeo.Land {
							world.markExplored(neighbor)
						}
					}
				}
			}
		}
	}
}

func (world *World) revealFromSapiens() {
	for _, band := range world.bands {
		if band.Species != HomoSapiens || band.Population <= 0 {
			continue
		}
		x, y, _ := TileXY(band.TileID)
		for dy := -1; dy <= 1; dy++ {
			for dx := -1; dx <= 1; dx++ {
				id, err := TileIDAt(x+dx, y+dy)
				if err == nil {
					world.markExplored(id)
				}
			}
		}
		for _, passage := range passageCatalog {
			if passageAvailability(band, passage, world.habitat, world.climate.LongTermTempOffset, false) != PassageAvailable {
				continue
			}
			destination, _ := passageDestination(passage, band.TileID)
			world.markExplored(destination)
		}
	}
}

func (world *World) bandIndex(id BandID) int {
	index := sort.Search(len(world.bands), func(index int) bool { return world.bands[index].ID >= id })
	if index >= len(world.bands) || world.bands[index].ID != id {
		return -1
	}
	return index
}

func (world *World) validate() error {
	if world == nil || world.grid == nil || world.habitat == nil || world.rng == nil {
		return fmt.Errorf("%w: incomplete world", ErrInvalidValue)
	}
	if _, err := CampaignDate(world.turn); err != nil {
		return err
	}
	if world.result > CampaignDispersalFailed || world.establishedRegions&^uint16((1<<RegionCount)-1) != 0 {
		return fmt.Errorf("%w: campaign state", ErrInvalidValue)
	}
	season, _ := SeasonForTurn(world.turn)
	for id, tile := range world.tiles {
		geography, _ := world.grid.Tile(TileID(id))
		habitat := world.habitat[id]
		values := [...]float64{tile.Degradation, tile.Stock.Flora, tile.Stock.Fauna, tile.Stock.Water}
		for _, value := range values {
			if math.IsNaN(value) || math.IsInf(value, 0) || value < 0 {
				return fmt.Errorf("%w: tile resources", ErrInvalidValue)
			}
		}
		if tile.Degradation > MaxDegradation {
			return fmt.Errorf("%w: tile degradation", ErrInvalidValue)
		}
		caps := ResourceCaps(habitat.Biome, season, tile.Degradation, habitat.BaselineK)
		if !geography.Land || habitat.BaselineK <= 0 {
			if tile != (TileState{}) {
				return fmt.Errorf("%w: uninhabitable tile state", ErrInvalidValue)
			}
		} else if tile.Stock.Flora > caps.Flora+1e-9 || tile.Stock.Fauna > caps.Fauna+1e-9 || tile.Stock.Water > caps.Water+1e-9 {
			return fmt.Errorf("%w: tile resource above capacity", ErrInvalidValue)
		}
	}
	if len(world.bands) > MaxBands {
		return ErrBandLimitReached
	}
	previous := BandID(0)
	for index := range world.bands {
		band := world.bands[index]
		if band.ID == 0 || (index > 0 && band.ID <= previous) {
			return fmt.Errorf("%w: band IDs not strictly ordered", ErrInvalidValue)
		}
		previous = band.ID
		if band.TileID >= TileCount || band.Species >= SpeciesCount {
			return fmt.Errorf("%w: band identity", ErrInvalidValue)
		}
		if band.Population > 0 && world.habitat[band.TileID].BaselineK <= 0 {
			return fmt.Errorf("%w: living band on uninhabitable tile", ErrInvalidValue)
		}
		if err := ValidateHealth(band.Health); err != nil {
			return err
		}
		if err := ValidateFU(band.StoredFood); err != nil {
			return err
		}
		if float64(band.StoredFood) > FoodStorageCapacity(band.Population) {
			return fmt.Errorf("%w: stored food above capacity", ErrInvalidValue)
		}
		if err := ValidateAssignments(band.Allocation); err != nil {
			return err
		}
		if err := band.Technology.Validate(); err != nil {
			return err
		}
		for _, trait := range band.Heritable {
			if err := ValidateTraitValue(trait); err != nil {
				return err
			}
		}
		if band.LastFoodReport.Turn == 0 {
			if band.LastFoodReport.RequiredFU != 0 || band.LastFoodReport.DeficitFU != 0 {
				return fmt.Errorf("%w: unavailable food report has values", ErrInvalidValue)
			}
		} else if band.LastFoodReport.Turn != world.turn || band.LastFoodReport.RequiredFU < 0 || band.LastFoodReport.DeficitFU < 0 || band.LastFoodReport.DeficitFU > band.LastFoodReport.RequiredFU || math.IsNaN(band.LastFoodReport.RequiredFU) || math.IsNaN(band.LastFoodReport.DeficitFU) || math.IsInf(band.LastFoodReport.RequiredFU, 0) || math.IsInf(band.LastFoodReport.DeficitFU, 0) {
			return fmt.Errorf("%w: food report", ErrInvalidValue)
		}
		mortality := [...]float64{band.LastMortality.Starvation, band.LastMortality.Seasonal, band.LastMortality.Chronic, band.LastMortality.Macro, band.LastMortality.Acute}
		for _, value := range mortality {
			if math.IsNaN(value) || math.IsInf(value, 0) || value < 0 {
				return fmt.Errorf("%w: mortality report", ErrInvalidValue)
			}
		}
		outcome := band.LastOutcomeReport
		if outcome.Turn == 0 {
			if outcome != (OutcomeReport{}) {
				return fmt.Errorf("%w: unavailable outcome report has values", ErrInvalidValue)
			}
		} else {
			if outcome.Turn != world.turn || outcome.StartingPopulation == 0 || outcome.EndingPopulation != band.Population || outcome.EndingHealth != band.Health {
				return fmt.Errorf("%w: outcome report identity", ErrInvalidValue)
			}
			if err := ValidateHealth(outcome.StartingHealth); err != nil {
				return err
			}
			if err := ValidateHealth(outcome.EndingHealth); err != nil {
				return err
			}
			if math.IsNaN(outcome.Growth) || math.IsInf(outcome.Growth, 0) || math.IsNaN(outcome.NutritionDelta) || math.IsInf(outcome.NutritionDelta, 0) {
				return fmt.Errorf("%w: outcome report change", ErrInvalidValue)
			}
			healthLosses := [...]float64{outcome.WaterHealthLoss, outcome.DiseaseHealthLoss, outcome.GeneticBurdenHealthLoss, outcome.MacroHealthLoss, outcome.AcuteDiseaseHealthLoss}
			for _, value := range healthLosses {
				if math.IsNaN(value) || math.IsInf(value, 0) || value < 0 {
					return fmt.Errorf("%w: outcome health report", ErrInvalidValue)
				}
			}
		}
		if band.HasQueuedMigration {
			valid := band.SpatialActionUsed && band.QueuedOrigin == band.TileID && band.QueuedMigration < TileCount && world.habitat[band.QueuedMigration].BaselineK > 0
			if valid {
				switch {
				case band.QueueUsesPassage:
					if band.QueuedPassage >= PassageCount {
						valid = false
					} else {
						passage := passageCatalog[band.QueuedPassage]
						destination, atEndpoint := passageDestination(passage, band.QueuedOrigin)
						valid = atEndpoint && destination == band.QueuedMigration && passageAvailability(band, passage, world.habitat, world.climate.LongTermTempOffset, false) == PassageAvailable
					}
				default:
					valid = false
					for _, edge := range world.grid.OrdinaryEdges(band.TileID) {
						if edge.To == band.QueuedMigration {
							valid = true
							break
						}
					}
				}
			}
			if !valid {
				return fmt.Errorf("%w: queued migration", ErrInvalidValue)
			}
		}
		if band.HasInterbreedTarget {
			targetIndex := world.bandIndex(band.InterbreedTarget)
			if !band.SpatialActionUsed || targetIndex < 0 || world.bands[targetIndex].Species == band.Species || world.bands[targetIndex].TileID != band.TileID {
				return fmt.Errorf("%w: interbreed target", ErrInvalidValue)
			}
		}
	}
	if len(world.bands) != 0 && world.nextBandID <= world.bands[len(world.bands)-1].ID {
		return fmt.Errorf("%w: next band ID", ErrInvalidValue)
	}
	previousEventTurn := -1
	for _, event := range world.events {
		if event.Turn < 0 || event.Turn > world.turn || event.Turn < previousEventTurn || event.Kind >= EventKindCount ||
			event.BandID >= world.nextBandID || event.TileID >= TileCount || event.Region >= RegionCount || event.Summary == "" || len(event.Summary) > 256 {
			return fmt.Errorf("%w: event feed", ErrInvalidValue)
		}
		previousEventTurn = event.Turn
	}
	if len(world.events) > MaxEvents {
		return fmt.Errorf("%w: event feed capacity", ErrInvalidValue)
	}
	hasSapiens := false
	for _, band := range world.bands {
		if band.Species == HomoSapiens {
			hasSapiens = true
			break
		}
	}
	wantResult := CampaignOngoing
	switch {
	case !hasSapiens:
		wantResult = CampaignExtinction
	case world.turn == MaxCampaignTurn && world.establishedRegions&destinationMask() != 0:
		wantResult = CampaignVictory
	case world.turn == MaxCampaignTurn:
		wantResult = CampaignDispersalFailed
	}
	if world.result != wantResult {
		return fmt.Errorf("%w: terminal campaign state", ErrInvalidValue)
	}
	return nil
}
