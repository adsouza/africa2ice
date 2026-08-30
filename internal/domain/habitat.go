package domain

type HabitatTile struct {
	ID                  TileID
	Biome               Biome
	EffectiveMoisture   float64
	VegetationIndex     float64
	HabitatTemperatureC float64
	LocalTemperatureC   float64
	BaselineK           float64
	MovementCost        float64
}

type Habitat [TileCount]HabitatTile

type biomeDerivationState struct {
	current      Biome
	candidate    Biome
	candidateRun int
}

// deriveBiomeHistory turns the instantaneous classifier into a stable authored
// campaign table. A different raw class must persist for MinBiomeDwellTurns
// before it is published, preventing seasonal threshold noise from making
// resource and movement categories flicker every few turns.
func (grid *Grid) deriveBiomeHistory() error {
	grid.biomes = make([]Biome, (MaxCampaignTurn+1)*TileCount)
	states := [TileCount]biomeDerivationState{}
	for turn := 0; turn <= MaxCampaignTurn; turn++ {
		climate, err := ClimateAt(0, turn)
		if err != nil {
			return err
		}
		for id := range TileCount {
			geography := grid.tiles[id]
			if !geography.Land {
				continue
			}
			habitatOffset := climate.LongTermTempOffset + climate.SeasonalTempOffset + climate.RegionalAbrupt[geography.Region]
			temperature, err := HabitatTemperatureC(geography.Y, geography.ElevationKm, habitatOffset)
			if err != nil {
				return err
			}
			moisture := EffectiveMoisture(geography.BaseMoisture, geography.Region, climate)
			vegetation := VegetationIndex(moisture, temperature)
			raw := ClassifyBiome(geography.ElevationKm, geography.Coastal, vegetation, temperature)
			state := &states[id]
			switch {
			case turn == 0:
				state.current = raw
			case raw == state.current:
				state.candidateRun = 0
			default:
				if raw == state.candidate {
					state.candidateRun++
				} else {
					state.candidate = raw
					state.candidateRun = 1
				}
				if state.candidateRun >= MinBiomeDwellTurns {
					state.current = state.candidate
					state.candidateRun = 0
				}
			}
			grid.biomes[turn*TileCount+id] = state.current
		}
	}
	return nil
}

func BuildHabitat(grid *Grid, seed uint64, turn int) (*Habitat, ClimateState, error) {
	climate, err := ClimateAt(seed, turn)
	if err != nil {
		return nil, ClimateState{}, err
	}
	result := &Habitat{}
	for id := range TileCount {
		geography, ok := grid.Tile(TileID(id))
		if !ok {
			return nil, ClimateState{}, ErrInvalidCoordinate
		}
		tile := HabitatTile{ID: TileID(id)}
		if !geography.Land {
			result[id] = tile
			continue
		}
		habitatOffset := climate.LongTermTempOffset + climate.SeasonalTempOffset + climate.RegionalAbrupt[geography.Region]
		habitatTemperature, err := HabitatTemperatureC(geography.Y, geography.ElevationKm, habitatOffset)
		if err != nil {
			return nil, ClimateState{}, err
		}
		localTemperature := habitatTemperature + climate.ClimateNoise
		moisture := EffectiveMoisture(geography.BaseMoisture, geography.Region, climate)
		vegetation := VegetationIndex(moisture, habitatTemperature)
		biome, ok := grid.biomeAt(turn, TileID(id))
		if !ok {
			return nil, ClimateState{}, ErrInvalidCoordinate
		}
		tile.Biome = biome
		tile.EffectiveMoisture = moisture
		tile.VegetationIndex = vegetation
		tile.HabitatTemperatureC = habitatTemperature
		tile.LocalTemperatureC = localTemperature
		tile.BaselineK = BaselineK(vegetation, biome)
		tile.MovementCost = ComposedMovementCost(vegetation, biome)
		result[id] = tile
	}
	return result, climate, nil
}
