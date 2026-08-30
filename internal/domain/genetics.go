package domain

import "math"

type HeritableState [HeritableTraitCount]TraitValue

var sapiensEastAfricaTraits = HeritableState{0.10, 0.00, 0.45, 0.55, 0.85, 0.40}
var archaicLevantTraits = HeritableState{0.45, 0.00, 0.60, 0.25, 0.65, 0.55}
var archaicFrangistanTraits = HeritableState{0.75, 0.10, 0.55, 0.00, 0.45, 0.65}

func StartingHeritableState(species Species, region Region) (HeritableState, bool) {
	switch {
	case species == HomoSapiens && region == EastAfrica:
		return sapiensEastAfricaTraits, true
	case species == ArchaicHominin && region == Levant:
		return archaicLevantTraits, true
	case species == ArchaicHominin && region == Frangistan:
		return archaicFrangistanTraits, true
	default:
		return HeritableState{}, false
	}
}

func ColdRemainingRisk(value, coldPressure float64) float64 {
	return 1 - float64(0.50*float64(value*clamp01(coldPressure)))
}
func HighAltitudeRemainingRisk(value, pressure float64) float64 {
	return 1 - float64(0.60*float64(value*clamp01(pressure)))
}
func InnateImmuneRemainingRisk(value float64) float64 { return 1 - float64(0.45*clamp01(value)) }
func AridHeatRemaining(value float64) float64         { return 1 - float64(0.40*clamp01(value)) }

type FoodSource uint8

const (
	PlantFood FoodSource = iota
	AnimalFood
	AquaticFood
)

func FattyAcidConversion(source FoodSource, value float64) float64 {
	value = clamp01(value)
	switch source {
	case PlantFood:
		return 1.10 - float64(0.20*value)
	case AnimalFood:
		return 0.95 + float64(0.10*value)
	case AquaticFood:
		return 0.90 + float64(0.20*value)
	default:
		return 0
	}
}

var diseaseHealthRates = [BiomeCount]struct{ camp, nonCamp float64 }{
	RiverineWoodland:     {0.020, 0.010},
	Savanna:              {0.010, 0.010},
	CoastalShrubland:     {0.010, 0.010},
	MountainousHighlands: {0.005, 0.005},
	SemiAridDesert:       {0.005, 0.005},
	GlacialTundra:        {0.005, 0.005},
}

var seasonUVFactor = [SeasonCount]float64{1.00, 0.85, 0.70, 0.85}

func UVExposure(tile TileGeography, season Season) float64 {
	if tile.ID >= TileCount || season >= SeasonCount || !tile.Land {
		return 0
	}
	latitudeUV := 1 - math.Float64frombits(latitudeSinSquaredBits[tile.Y])
	altitudeFactor := 1 + float64(0.10*tile.ElevationKm)
	return clamp01(float64(float64(latitudeUV*seasonUVFactor[season]) * altitudeFactor))
}

func SelectionDeltas(band Band, tile TileGeography, habitat HabitatTile, season Season, animalFoodShare float64) HeritableState {
	state := band.Heritable
	coldPressure := clamp01((10 - habitat.LocalTemperatureC) / 30)
	hypoxiaPressure := clamp01((tile.ElevationKm - 1.5) / 2.5)
	rates := diseaseHealthRates[habitat.Biome]
	pathogenPressure := clamp01((rates.camp + rates.nonCamp) / 0.04)
	heatPressure := clamp01((habitat.LocalTemperatureC - 20) / 15)
	uv := UVExposure(tile, season)
	result := HeritableState{}
	result[ColdAdaptation] = TraitValue(float64(0.012 * (coldPressure - 0.25) * float64(state[ColdAdaptation]) * (1 - float64(state[ColdAdaptation]))))
	result[HighAltitudeAdaptation] = TraitValue(float64(0.012 * (hypoxiaPressure - 0.15) * float64(state[HighAltitudeAdaptation]) * (1 - float64(state[HighAltitudeAdaptation]))))
	result[InnateImmuneReactivity] = TraitValue(float64(0.010 * (pathogenPressure - 0.45) * float64(state[InnateImmuneReactivity]) * (1 - float64(state[InnateImmuneReactivity]))))
	result[AridClimateAdaptation] = TraitValue(float64(0.010 * (heatPressure - 0.25) * float64(state[AridClimateAdaptation]) * (1 - float64(state[AridClimateAdaptation]))))
	result[PigmentationLevel] = TraitValue(float64(0.005 * (uv - float64(state[PigmentationLevel]))))
	result[FattyAcidMetabolism] = TraitValue(float64(0.012 * (clamp01(animalFoodShare) - 0.50) * float64(state[FattyAcidMetabolism]) * (1 - float64(state[FattyAcidMetabolism]))))
	return result
}

const (
	SameSpeciesGeneFlowRate = 0.025
	InterbreedGeneFlowRate  = 0.10
	MaxGeneFlowPerTurn      = 0.20
)

const (
	MutationProbabilityCold         = 0.00020
	MutationProbabilityHighAltitude = 0.00015
	MutationProbabilityImmune       = 0.00025
	MutationProbabilityArid         = 0.00020
	MutationProbabilityPigmentation = 0.0
	MutationProbabilityFattyAcid    = 0.00020
)

var MutationEntryFrequency = [HeritableTraitCount]float64{0.020, 0.020, 0.015, 0.020, 0, 0.020}
var MutationProbability = [HeritableTraitCount]float64{
	MutationProbabilityCold,
	MutationProbabilityHighAltitude,
	MutationProbabilityImmune,
	MutationProbabilityArid,
	MutationProbabilityPigmentation,
	MutationProbabilityFattyAcid,
}
