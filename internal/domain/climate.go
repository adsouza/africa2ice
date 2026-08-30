package domain

import "math"

//go:generate go run ../../tools/generate_climate_tables -output climate_tables_generated.go

const (
	LGMCooling             = 6.1
	OrbitalAmplitude       = 1.5
	SeasonalAmplitude      = 2.0
	NoiseAmplitude         = 0.35
	AridificationAmplitude = 0.35
	PrecessionAmplitude    = 0.10
	BeringiaOpenFraction   = 0.85
	EpochLowerThreshold    = 0.25
	EpochUpperThreshold    = 0.96
	EpochHysteresis        = 0.02
)

type ClimateEpoch uint8

const (
	HumidOptimum ClimateEpoch = iota
	AridTransition
	GlacialMaximum
)

type ClimateState struct {
	Turn                   int
	LongTermTempOffset     float64
	SeasonalTempOffset     float64
	ClimateNoise           float64
	GlobalTempOffset       float64
	LongTermMoistureOffset float64
	AridityIndex           float64
	Epoch                  ClimateEpoch
	RegionalAbrupt         [RegionCount]float64
}

type climatePulse struct {
	startBP, peakBP, endBP int
	amplitude              float64
}

var climatePulses = [...]climatePulse{
	{72_290, 71_990, 70_790, 1.5}, {64_050, 63_750, 62_550, 1.5},
	{54_170, 53_870, 52_670, 2.5}, {46_810, 46_660, 46_060, 2.0},
	{38_170, 38_020, 37_420, 2.0}, {32_450, 32_350, 31_950, 1.5},
	{23_290, 23_240, 23_040, 1.0},
}

var regionalAbruptWeight = [RegionCount]float64{0.10, 0.15, 0.20, 0.50, 1.00, 0.65, 0.20, 0.10, 0.35, 0.40, 0.05, 0.55, 0.35}
var regionalAridityWeight = [RegionCount]float64{0.55, 0.90, 1.00, 0.60, 0.50, 0.75, 0.65, 0.30, 0.55, 0.70, 0.80, 0.45, 0.40}

func ClimateAt(seed uint64, turn int) (ClimateState, error) {
	date, err := CampaignDate(turn)
	if err != nil {
		return ClimateState{}, err
	}
	u := date.CalendarProgress
	smooth := smoothstep(u)
	orbital := math.Float64frombits(orbitalSinBits[turn])
	longTerm := -float64(LGMCooling*smooth) + float64(OrbitalAmplitude*float64((1-u)*orbital))
	seasonal := float64(SeasonalAmplitude * math.Float64frombits(seasonalCosBits[turn%12]))
	noise := float64(NoiseAmplitude * UnitNoiseV1(seed, turn))
	precession := math.Float64frombits(precessionSinBits[turn])
	moisture := -float64(AridificationAmplitude*smooth) + float64(PrecessionAmplitude*float64((1-u)*precession))
	aridity := clamp01(-moisture / AridificationAmplitude)
	state := ClimateState{
		Turn: turn, LongTermTempOffset: longTerm, SeasonalTempOffset: seasonal,
		ClimateNoise: noise, GlobalTempOffset: longTerm + seasonal + noise,
		LongTermMoistureOffset: moisture, AridityIndex: aridity,
	}
	for region := Region(0); region < RegionCount; region++ {
		state.RegionalAbrupt[region] = AbruptClimateOffset(region, date.YearBP)
	}
	state.Epoch = ClimateEpochForTurn(turn)
	return state, nil
}

func smoothstep(value float64) float64 {
	value = clamp01(value)
	squared := float64(value * value)
	return float64(squared * (3 - float64(2*value)))
}

func AbruptClimateOffset(region Region, yearBP int) float64 {
	if region >= RegionCount {
		return 0
	}
	total := 0.0
	for _, pulse := range climatePulses {
		total += float64(pulse.amplitude * pulseEnvelope(pulse, yearBP))
	}
	return float64(total * regionalAbruptWeight[region])
}

func pulseEnvelope(pulse climatePulse, yearBP int) float64 {
	if yearBP > pulse.startBP || yearBP < pulse.endBP {
		return 0
	}
	if yearBP >= pulse.peakBP {
		progress := float64(pulse.startBP-yearBP) / float64(pulse.startBP-pulse.peakBP)
		return smoothstep(progress)
	}
	progress := float64(pulse.peakBP-yearBP) / float64(pulse.peakBP-pulse.endBP)
	return 1 - smoothstep(progress)
}

func TileMoistureOffset(region Region, climate ClimateState) float64 {
	if region >= RegionCount {
		return 0
	}
	return float64(climate.LongTermMoistureOffset * regionalAridityWeight[region])
}

func EffectiveMoisture(base float64, region Region, climate ClimateState) float64 {
	return clamp01(base + TileMoistureOffset(region, climate))
}

func BeringiaOpen(longTermTempOffset float64) bool {
	threshold := -float64(BeringiaOpenFraction * LGMCooling)
	return longTermTempOffset <= threshold
}

func ClimateEpochForTurn(turn int) ClimateEpoch {
	epoch := HumidOptimum
	for current := 0; current <= turn && current <= MaxCampaignTurn; current++ {
		date, _ := CampaignDate(current)
		u := date.CalendarProgress
		moisture := -float64(AridificationAmplitude*smoothstep(u)) + float64(PrecessionAmplitude*float64((1-u)*math.Float64frombits(precessionSinBits[current])))
		x := clamp01(-moisture / AridificationAmplitude)
		if current == 0 {
			switch {
			case x < EpochLowerThreshold:
				epoch = HumidOptimum
			case x < EpochUpperThreshold:
				epoch = AridTransition
			default:
				epoch = GlacialMaximum
			}
			continue
		}
		switch epoch {
		case HumidOptimum:
			if x >= EpochUpperThreshold+EpochHysteresis {
				epoch = GlacialMaximum
			} else if x >= EpochLowerThreshold+EpochHysteresis {
				epoch = AridTransition
			}
		case AridTransition:
			if x >= EpochUpperThreshold+EpochHysteresis {
				epoch = GlacialMaximum
			} else if x <= EpochLowerThreshold-EpochHysteresis {
				epoch = HumidOptimum
			}
		case GlacialMaximum:
			if x <= EpochLowerThreshold-EpochHysteresis {
				epoch = HumidOptimum
			} else if x <= EpochUpperThreshold-EpochHysteresis {
				epoch = AridTransition
			}
		}
	}
	return epoch
}
