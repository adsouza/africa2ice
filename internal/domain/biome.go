package domain

const (
	VegetationColdCutoffC = -5.0
	VegetationWarmthC     = 10.0
	TundraSplitTempC      = 0.0
	CanopyClosureV        = 0.90
	MaxMovementCost       = 2.75
	BiomeChurnCap         = 12
	MinBiomeDwellTurns    = 15
)

var biomeCapacityFactor = [BiomeCount]float64{1, 1, 1.15, 0.65, 1, 1}
var biomeMovementFactor = [BiomeCount]float64{1, 1, 1.25, 2.50, 1, 1}

func ThermalSuitability(temperatureC float64) float64 {
	return clamp01((temperatureC - VegetationColdCutoffC) / (VegetationWarmthC - VegetationColdCutoffC))
}

func VegetationIndex(effectiveMoisture, habitatTemperatureC float64) float64 {
	return clamp01(float64(effectiveMoisture * ThermalSuitability(habitatTemperatureC)))
}

func ClassifyBiome(elevationKm float64, coastal bool, vegetationIndex, habitatTemperatureC float64) Biome {
	if elevationKm > HighlandElevationKm {
		return MountainousHighlands
	}
	if coastal {
		return CoastalShrubland
	}
	if vegetationIndex < 0.30 {
		if habitatTemperatureC < TundraSplitTempC {
			return GlacialTundra
		}
		return SemiAridDesert
	}
	if vegetationIndex < 0.45 {
		return SemiAridDesert
	}
	if vegetationIndex < 0.75 {
		return Savanna
	}
	return RiverineWoodland
}

func BaselineKCurve(v float64) float64 {
	v = clamp01(v)
	switch {
	case v <= 0.15:
		return interpolate(v, 0, 0, 0.15, 9)
	case v <= 0.30:
		return interpolate(v, 0.15, 9, 0.30, 34)
	case v < 0.45:
		return interpolate(v, 0.30, 34, 0.45, 77)
	case v == 0.45:
		return 103
	case v <= 0.75:
		return interpolate(v, 0.45, 103, 0.75, 137)
	case v <= 0.90:
		return interpolate(v, 0.75, 137, 0.90, 171)
	default:
		return interpolate(v, 0.90, 171, 1, 145)
	}
}

func BaselineK(v float64, biome Biome) float64 {
	if biome >= BiomeCount {
		return 0
	}
	return float64(BaselineKCurve(v) * biomeCapacityFactor[biome])
}

func MovementCurve(v float64) float64 {
	v = clamp01(v)
	switch {
	case v <= 0.075:
		return 2.20
	case v <= 0.225:
		return interpolate(v, 0.075, 2.20, 0.225, 2.00)
	case v <= 0.375:
		return interpolate(v, 0.225, 2.00, 0.375, 1.50)
	case v <= 0.600:
		return interpolate(v, 0.375, 1.50, 0.600, 1.00)
	case v <= 0.900:
		return 1.00
	default:
		return interpolate(v, 0.900, 1.00, 1.000, 1.20)
	}
}

func ComposedMovementCost(v float64, biome Biome) float64 {
	if biome >= BiomeCount {
		return MaxMovementCost
	}
	cost := float64(MovementCurve(v) * biomeMovementFactor[biome])
	if cost < 1 {
		return 1
	}
	if cost > MaxMovementCost {
		return MaxMovementCost
	}
	return cost
}

func interpolate(x, x0, y0, x1, y1 float64) float64 {
	fraction := (x - x0) / (x1 - x0)
	return y0 + float64(fraction*(y1-y0))
}
