package domain

const (
	ReferenceFloraCapacity  = 300.0
	ReferenceFaunaCapacity  = 300.0
	ReferenceWaterCapacity  = 250.0
	FloraRegenerationRate   = 0.30
	FaunaRegenerationRate   = 0.15
	WaterRegenerationRate   = 0.50
	MaxDegradation          = 0.75
	DegradationDamageRate   = 0.06
	DegradationRecoveryRate = 0.03
)

type ResourceVector struct {
	Flora float64
	Fauna float64
	Water float64
}

type TileState struct {
	Degradation float64
	Stock       ResourceVector
}

var resourceCapIndex = [BiomeCount][SeasonCount]ResourceVector{
	RiverineWoodland: {
		{1.50, 1.20, 2.00}, {1.30, 1.10, 1.80}, {1.00, 1.00, 1.50}, {1.30, 1.10, 1.80},
	},
	Savanna: {
		{1.00, 1.00, 1.00}, {0.85, 0.95, 0.85}, {0.65, 0.80, 0.65}, {0.85, 0.95, 0.85},
	},
	CoastalShrubland: {
		{0.80, 1.10, 1.20}, {0.70, 1.00, 1.10}, {0.55, 0.90, 0.90}, {0.70, 1.00, 1.10},
	},
	MountainousHighlands: {
		{0.50, 0.65, 0.90}, {0.35, 0.55, 0.80}, {0.20, 0.45, 0.70}, {0.35, 0.55, 0.80},
	},
	SemiAridDesert: {
		{0.25, 0.35, 0.30}, {0.20, 0.30, 0.25}, {0.15, 0.25, 0.20}, {0.20, 0.30, 0.25},
	},
	GlacialTundra: {
		{0.20, 1.20, 0.70}, {0.10, 1.10, 0.65}, {0.05, 0.90, 0.55}, {0.10, 1.10, 0.65},
	},
}

var initialStockFraction = [BiomeCount]ResourceVector{
	RiverineWoodland:     {0.90, 0.85, 0.95},
	Savanna:              {0.80, 0.80, 0.75},
	CoastalShrubland:     {0.80, 0.85, 0.90},
	MountainousHighlands: {0.70, 0.75, 0.80},
	SemiAridDesert:       {0.60, 0.70, 0.55},
	GlacialTundra:        {0.50, 0.85, 0.75},
}

func ResourceCaps(biome Biome, season Season, degradation, baselineK float64) ResourceVector {
	if biome >= BiomeCount || season >= SeasonCount || baselineK <= 0 {
		return ResourceVector{}
	}
	factor := 1 - clamp01(degradation)
	index := resourceCapIndex[biome][season]
	return ResourceVector{
		Flora: float64(float64(ReferenceFloraCapacity*index.Flora) * factor),
		Fauna: float64(float64(ReferenceFaunaCapacity*index.Fauna) * factor),
		Water: float64(float64(ReferenceWaterCapacity*index.Water) * factor),
	}
}

func InitialTileState(seed uint64, geography TileGeography, habitat HabitatTile, season Season) TileState {
	cap := ResourceCaps(habitat.Biome, season, 0, habitat.BaselineK)
	if habitat.BaselineK <= 0 {
		return TileState{}
	}
	base := initialStockFraction[habitat.Biome]
	regionKey, tileKey := uint64(geography.Region), uint64(geography.ID)
	floraFraction := initialFraction(base.Flora, seed, regionKey, tileKey, FloraStock)
	faunaFraction := initialFraction(base.Fauna, seed, regionKey, tileKey, FaunaStock)
	waterFraction := initialFraction(base.Water, seed, regionKey, tileKey, WaterStock)
	return TileState{Stock: ResourceVector{
		Flora: float64(cap.Flora * floraFraction),
		Fauna: float64(cap.Fauna * faunaFraction),
		Water: float64(cap.Water * waterFraction),
	}}
}

func initialFraction(base float64, seed, region, tile uint64, stock ResourceStock) float64 {
	regionOffset := float64(0.08 * UnitResourceAbundanceV1(seed, ResourceRegion, region, stock))
	tileOffset := float64(0.04 * UnitResourceAbundanceV1(seed, ResourceTile, tile, stock))
	return clamp01(base + regionOffset + tileOffset)
}

func (state TileState) Regenerate(cap ResourceVector, extraction ResourceVector) TileState {
	return TileState{Degradation: state.Degradation, Stock: ResourceVector{
		Flora: regenerateStock(state.Stock.Flora, cap.Flora, extraction.Flora, FloraRegenerationRate),
		Fauna: regenerateStock(state.Stock.Fauna, cap.Fauna, extraction.Fauna, FaunaRegenerationRate),
		Water: regenerateStock(state.Stock.Water, cap.Water, extraction.Water, WaterRegenerationRate),
	}}
}

func regenerateStock(stock, capacity, extraction, rate float64) float64 {
	if capacity <= 0 {
		return 0
	}
	capped := stock
	if capped > capacity {
		capped = capacity
	}
	if capped < 0 {
		capped = 0
	}
	regrown := capped + float64(rate*(capacity-capped))
	next := regrown - extraction
	if next < 0 {
		return 0
	}
	if next > capacity {
		return capacity
	}
	return next
}

func NextDegradation(current, totalPopulation, baselineK float64) float64 {
	if baselineK <= 0 {
		return 0
	}
	pressure := totalPopulation / baselineK
	if pressure > 1 {
		excess := pressure - 1
		next := current + float64(DegradationDamageRate*float64(excess*excess))
		if next > MaxDegradation {
			return MaxDegradation
		}
		return next
	}
	next := current - float64(DegradationRecoveryRate*(1-pressure))
	if next < 0 {
		return 0
	}
	return next
}
