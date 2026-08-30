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
		biome := ClassifyBiome(geography.ElevationKm, geography.Coastal, vegetation, habitatTemperature)
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
