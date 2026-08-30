package domain

import "math"

const (
	ForagingReferenceRate = 2.5
	HuntingReferenceRate  = 2.5
	MegafaunaRateFactor   = 1.75
	BaseWaterPerPerson    = 1.0
	HealthLossRate        = 0.20
	HealthRecoveryRate    = 0.05
	WaterHealthLossRate   = 0.40
	StarvationCoefficient = 0.10
	PopulationGrowthRate  = 0.002

	// MaxCrowdingDeclineFraction bounds how much of a band the logistic crowding
	// term may remove in a single turn.
	MaxCrowdingDeclineFraction = 0.25
)

var biomeForagingIndex = [BiomeCount]float64{1.20, 1.00, 0.75, 0.60, 0.40, 0.30}

func ForagingRate(band Band, biome Biome) float64 {
	if biome >= BiomeCount {
		return 0
	}
	multiplier := 1.0
	if band.Technology.Has(PlantKnowledge) {
		multiplier = 1.30
	}
	return float64(float64(ForagingReferenceRate*biomeForagingIndex[biome]) * multiplier)
}

func GroupHuntingRate(band Band, region Region, profile FaunaProfile, group FaunaGroup) float64 {
	if region >= RegionCount || group >= FaunaGroupCount || group == Megafauna || !profile.HuntingSupported || profile.Weights[group] <= 0 {
		return 0
	}
	if group == PelagicAquatic && !band.Technology.Has(CordageAndNets) {
		return 0
	}
	multiplier := 1.0
	if band.Technology.Has(HaftedTools) {
		if group == LargeGame {
			multiplier = float64(multiplier * 1.30)
		}
		if group == InshoreAquatic {
			multiplier = float64(multiplier * 1.15)
		}
	}
	if band.Technology.Has(CordageAndNets) && group == InshoreAquatic {
		multiplier = float64(multiplier * 1.25)
	}
	if band.Technology.Has(Trapping) {
		switch group {
		case SmallGame:
			multiplier = float64(multiplier * 1.35)
		case MediumGame:
			multiplier = float64(multiplier * 1.20)
		case InshoreAquatic:
			multiplier = float64(multiplier * 1.20)
		}
	}
	if band.Technology.Has(CoastalNavigation) && group == PelagicAquatic {
		multiplier = float64(multiplier * 1.40)
	}
	base := float64(HuntingReferenceRate * huntingIndex[region])
	return float64(float64(base*profile.Weights[group]) * multiplier)
}

type HuntingRateSummary struct{ Terrestrial, Aquatic, Total float64 }

func HuntingRates(band Band, region Region, profile FaunaProfile) HuntingRateSummary {
	summary := HuntingRateSummary{}
	for group := FaunaGroup(0); group < FaunaGroupCount; group++ {
		rate := GroupHuntingRate(band, region, profile, group)
		switch group {
		case SmallGame, MediumGame, LargeGame:
			summary.Terrestrial += rate
		case InshoreAquatic, PelagicAquatic:
			summary.Aquatic += rate
		}
	}
	summary.Total = summary.Terrestrial + summary.Aquatic
	return summary
}

func MegafaunaRate(band Band, profile FaunaProfile, ordinaryTerrestrialRate float64) float64 {
	if !profile.MegafaunaSupported || profile.Weights[Megafauna] <= 0 || ordinaryTerrestrialRate <= 0 {
		return 0
	}
	multiplier := 1.0
	if band.Technology.Has(HaftedTools) {
		multiplier = 1.30
	}
	return float64(float64(MegafaunaRateFactor*ordinaryTerrestrialRate) * multiplier)
}

func ProportionalAllocate(available float64, demands []float64) []float64 {
	result := make([]float64, len(demands))
	if available <= 0 || len(demands) == 0 {
		return result
	}
	maximum := 0.0
	for _, demand := range demands {
		if demand > maximum && !math.IsInf(demand, 0) && !math.IsNaN(demand) {
			maximum = demand
		}
	}
	if maximum == 0 {
		return result
	}
	scaledTotal := 0.0
	for _, demand := range demands {
		if demand > 0 {
			scaledTotal += demand / maximum
		}
	}
	if scaledTotal == 0 {
		return result
	}
	if maximum <= available/scaledTotal {
		copy(result, demands)
		return result
	}
	used := 0.0
	last := -1
	for index, demand := range demands {
		if demand <= 0 {
			continue
		}
		share := (demand / maximum) / scaledTotal
		allocation := float64(available * share)
		result[index] = allocation
		used += allocation
		last = index
	}
	if used > available && last >= 0 {
		result[last] -= used - available
	}
	return result
}

func WaterDemandMultiplier(localTemperatureC float64, aridAdaptation TraitValue) float64 {
	heat := localTemperatureC - 20
	if heat < 0 {
		heat = 0
	}
	if heat > 15 {
		heat = 15
	}
	increment := float64(0.02 * heat)
	return 1 + float64(increment*AridHeatRemaining(float64(aridAdaptation)))
}

func WaterRequired(band Band, localTemperatureC float64) float64 {
	multiplier := WaterDemandMultiplier(localTemperatureC, band.Heritable[AridClimateAdaptation])
	return float64(float64(float64(band.Population)*BaseWaterPerPerson) * multiplier)
}

func FoodDeficit(required, available float64) (consumed, remaining, deficit, fraction float64) {
	if required <= 0 {
		return 0, available, 0, 0
	}
	consumed = available
	if consumed > required {
		consumed = required
	}
	remaining = available - consumed
	deficit = required - consumed
	fraction = deficit / required
	return
}

func NutritionHealthDelta(deficitFraction float64) float64 {
	if deficitFraction > 0 {
		return -float64(HealthLossRate * clamp01(deficitFraction))
	}
	return HealthRecoveryRate
}

func StarvationLoss(population, deficitFraction float64) float64 {
	fractionSquared := float64(deficitFraction * deficitFraction)
	return float64(float64(population*StarvationCoefficient) * fractionSquared)
}

// LogisticGrowth returns the band's requested change in people for the turn.
//
// The decline it can request is bounded. §7 describes the crowding factor as
// stopping growth for everyone standing on a full tile, but the same expression
// is unbounded below: at thirty times capacity it removes most of the band in one
// turn. That matters more than the magnitude suggests, because crowding is
// reported as growth and belongs to no MortalityReport cause, so the people are
// simply gone with every category reading zero and nothing in the UI able to
// account for them. Bounding the decline to a fraction of the band makes an
// over-capacity tile a sustained squeeze the player can see coming and answer,
// by splitting and moving through a chokepoint in smaller groups, instead of a
// single unexplained cull on the turn of arrival.
//
// A tile that cannot support anyone takes the same bound rather than annihilating
// its occupants, so the function stays continuous as capacity approaches zero and
// a climate shift under a settled band remains survivable long enough to answer.
func LogisticGrowth(population, totalPopulation, effectiveK, deficitFraction float64) float64 {
	if population <= 0 {
		return 0
	}
	decline := -float64(population * MaxCrowdingDeclineFraction)
	if effectiveK <= 0 {
		return decline
	}
	base := float64(float64(PopulationGrowthRate*population) * (1 - totalPopulation/effectiveK))
	if base > 0 {
		return float64(base * (1 - clamp01(deficitFraction)))
	}
	return max(base, decline)
}
