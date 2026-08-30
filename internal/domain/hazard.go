package domain

const (
	MaxShelterMitigation          = 0.60
	ShelterHalfSaturation         = 0.25
	NaturalShelterEfficiencyBonus = 1.00
	CampSecurityScale             = 0.75
	CampHygieneScale              = 0.40
	MaxChronicRate                = 0.25
)

type seasonalRiskRow struct{ exposure, campDisease, uncovered float64 }
type chronicRiskRow struct{ exposure, campPredation, campDisease, uncovered float64 }

var seasonalRisk = [BiomeCount][SeasonCount]seasonalRiskRow{
	RiverineWoodland:     {{0.001, 0.001, 0.002}, {0.001, 0.001, 0.002}, {0.002, 0.002, 0.002}, {0.001, 0.001, 0.002}},
	Savanna:              {{0.006, 0.000, 0.004}, {0.004, 0.000, 0.002}, {0.003, 0.000, 0.001}, {0.004, 0.000, 0.002}},
	CoastalShrubland:     {{0.003, 0.000, 0.005}, {0.002, 0.000, 0.004}, {0.003, 0.000, 0.005}, {0.002, 0.000, 0.004}},
	MountainousHighlands: {{0.005, 0.000, 0.003}, {0.008, 0.000, 0.004}, {0.014, 0.000, 0.006}, {0.008, 0.000, 0.004}},
	SemiAridDesert:       {{0.012, 0.000, 0.008}, {0.007, 0.000, 0.005}, {0.005, 0.000, 0.003}, {0.007, 0.000, 0.005}},
	GlacialTundra:        {{0.008, 0.000, 0.002}, {0.015, 0.000, 0.003}, {0.030, 0.000, 0.005}, {0.015, 0.000, 0.003}},
}

var chronicRisk = [BiomeCount]chronicRiskRow{
	RiverineWoodland:     {0.001, 0.002, 0.008, 0.007},
	Savanna:              {0.002, 0.004, 0.002, 0.004},
	CoastalShrubland:     {0.002, 0.002, 0.003, 0.005},
	MountainousHighlands: {0.005, 0.002, 0.001, 0.004},
	SemiAridDesert:       {0.004, 0.002, 0.001, 0.003},
	GlacialTundra:        {0.008, 0.002, 0.001, 0.004},
}

func ShelterCurve(share, naturalShelter float64) float64 {
	share, naturalShelter = clamp01(share), clamp01(naturalShelter)
	effort := share * (1 + NaturalShelterEfficiencyBonus*naturalShelter)
	if effort <= 0 {
		return 0
	}
	return MaxShelterMitigation * (effort / (effort + ShelterHalfSaturation))
}

func remainingRisk(technologyMitigation, campMitigation float64) float64 {
	return (1 - clamp01(technologyMitigation)) * (1 - clamp01(campMitigation))
}

func combinedTechnologyMitigation(effects ...float64) float64 {
	remaining := 1.0
	for _, effect := range effects {
		remaining *= 1 - effect
	}
	return 1 - remaining
}

func exposureTechnologyMitigation(band Band) float64 {
	effects := make([]float64, 0, 3)
	if band.Technology.Has(Firecraft) {
		effects = append(effects, 0.20)
	}
	if band.Technology.Has(TailoredClothing) {
		effects = append(effects, 0.50)
	}
	if band.Technology.Has(Campcraft) {
		effects = append(effects, 0.30)
	}
	return combinedTechnologyMitigation(effects...)
}

func campDiseaseTechnologyMitigation(band Band, medicinal bool) float64 {
	effects := make([]float64, 0, 2)
	if band.Technology.Has(Campcraft) {
		effects = append(effects, 0.30)
	}
	if medicinal && band.Technology.Has(MedicinalKnowledge) {
		effects = append(effects, 0.45)
	}
	return combinedTechnologyMitigation(effects...)
}

func Phase3HealthLosses(band Band, tile TileGeography, habitat HabitatTile, season Season) (disease, genetic float64) {
	share := float64(band.Allocation[Shelter]) / AllocationBasisPoints
	hygiene := CampHygieneScale * ShelterCurve(share, 0)
	rates := diseaseHealthRates[habitat.Biome]
	immuneRemaining := InnateImmuneRemainingRisk(float64(band.Heritable[InnateImmuneReactivity]))
	camp := rates.camp * remainingRisk(campDiseaseTechnologyMitigation(band, true), hygiene) * immuneRemaining
	nonCampTech := 0.0
	if band.Technology.Has(MedicinalKnowledge) {
		nonCampTech = 0.45
	}
	nonCamp := rates.nonCamp * remainingRisk(nonCampTech, 0) * immuneRemaining
	disease = camp + nonCamp
	pathogenPressure := clamp01((rates.camp + rates.nonCamp) / 0.04)
	immune := float64(band.Heritable[InnateImmuneReactivity])
	inflammation := 0.010 * immune * (1 - pathogenPressure)
	uv := UVExposure(tile, season)
	pigmentation := float64(band.Heritable[PigmentationLevel])
	vitamin := 0.0
	if pigmentation > uv {
		vitamin = 0.010 * (1 - uv) * (pigmentation - uv)
	}
	return disease, inflammation + vitamin
}

func Phase3MortalityRates(band Band, tile TileGeography, habitat HabitatTile, season Season) (seasonal, chronic float64) {
	share := float64(band.Allocation[Shelter]) / AllocationBasisPoints
	exposureCamp := ShelterCurve(share, tile.NaturalShelter)
	hygiene := CampHygieneScale * ShelterCurve(share, 0)
	security := CampSecurityScale * ShelterCurve(share, 0)
	coldPressure := clamp01((10 - habitat.LocalTemperatureC) / 30)
	hypoxiaPressure := clamp01((tile.ElevationKm - 1.5) / 2.5)
	coldRemaining := ColdRemainingRisk(float64(band.Heritable[ColdAdaptation]), coldPressure)
	altitudeRemaining := HighAltitudeRemainingRisk(float64(band.Heritable[HighAltitudeAdaptation]), hypoxiaPressure)
	immuneRemaining := InnateImmuneRemainingRisk(float64(band.Heritable[InnateImmuneReactivity]))

	seasonalBase := seasonalRisk[habitat.Biome][season]
	seasonal = seasonalBase.exposure*remainingRisk(exposureTechnologyMitigation(band), exposureCamp)*coldRemaining +
		seasonalBase.campDisease*remainingRisk(campDiseaseTechnologyMitigation(band, false), hygiene)*immuneRemaining +
		seasonalBase.uncovered

	chronicBase := chronicRisk[habitat.Biome]
	predationTech := 0.0
	if band.Technology.Has(Campcraft) {
		predationTech = 0.40
	}
	chronic = chronicBase.exposure*remainingRisk(exposureTechnologyMitigation(band), exposureCamp)*coldRemaining*altitudeRemaining +
		chronicBase.campPredation*remainingRisk(predationTech, security) +
		chronicBase.campDisease*remainingRisk(campDiseaseTechnologyMitigation(band, true), hygiene)*immuneRemaining +
		chronicBase.uncovered
	chronic *= 1 + (1 - float64(band.Health))
	uv := UVExposure(tile, season)
	pigmentation := float64(band.Heritable[PigmentationLevel])
	if uv > pigmentation {
		chronic += 0.012 * uv * (uv - pigmentation)
	}
	if chronic > MaxChronicRate {
		chronic = MaxChronicRate
	}
	return seasonal, chronic
}

type AcuteKind uint8

const (
	AcutePredation AcuteKind = iota
	AcuteDiseaseOutbreak
	AcuteFloodStorm
	AcuteExposureFall
	AcuteCrossingMishap
	AcuteKindCount
)

type protectionClass uint8

const (
	protectionExposure protectionClass = iota
	protectionCampPredation
	protectionCampDisease
	protectionUncovered
	protectionClassCount
)

const MaxAcuteProbability = 0.25

var acuteBiomeBase = [BiomeCount][4]float64{
	RiverineWoodland:     {0.015, 0.035, 0.030, 0.010},
	Savanna:              {0.025, 0.015, 0.010, 0.015},
	CoastalShrubland:     {0.015, 0.020, 0.040, 0.015},
	MountainousHighlands: {0.015, 0.010, 0.020, 0.045},
	SemiAridDesert:       {0.010, 0.010, 0.010, 0.040},
	GlacialTundra:        {0.020, 0.010, 0.010, 0.055},
}

var acuteSeasonFactor = [SeasonCount][4]float64{
	SeasonWarm:    {1.0, 1.2, 1.2, 0.8},
	SeasonCooling: {1.0, 1.0, 1.0, 1.0},
	SeasonCold:    {0.9, 0.8, 0.8, 1.3},
	SeasonWarming: {1.0, 1.0, 1.1, 1.0},
}

var acutePartition = [AcuteKindCount][protectionClassCount]float64{
	AcutePredation:       {0, 0.40, 0, 0.60},
	AcuteDiseaseOutbreak: {0, 0, 0.50, 0.50},
	AcuteFloodStorm:      {0, 0, 0, 1.00},
	AcuteExposureFall:    {0.70, 0, 0, 0.30},
	AcuteCrossingMishap:  {0, 0, 0, 1.00},
}

var minimumAcuteLoss = [AcuteKindCount]float64{0.02, 0.03, 0.02, 0.01, 0.05}
var maximumAcuteLoss = [AcuteKindCount]float64{0.08, 0.12, 0.10, 0.06, 0.15}
var passageAcuteRisk = [PassageCount]float64{0.08, 0.10, 0.04}

func campMitigation(band Band, class protectionClass, naturalShelter float64) float64 {
	share := float64(band.Allocation[Shelter]) / AllocationBasisPoints
	switch class {
	case protectionExposure:
		return ShelterCurve(share, naturalShelter)
	case protectionCampPredation:
		return CampSecurityScale * ShelterCurve(share, 0)
	case protectionCampDisease:
		return CampHygieneScale * ShelterCurve(share, 0)
	default:
		return 0
	}
}

func acuteTechnologyMitigation(band Band, kind AcuteKind, class protectionClass) float64 {
	effects := make([]float64, 0, 3)
	add := func(technology Technology, effect float64) {
		if band.Technology.Has(technology) {
			effects = append(effects, effect)
		}
	}
	switch {
	case kind == AcutePredation && class == protectionCampPredation:
		add(Firecraft, 0.15)
		add(Campcraft, 0.40)
	case kind == AcutePredation && class == protectionUncovered:
		add(HaftedTools, 0.20)
		add(Trapping, 0.20)
	case kind == AcuteDiseaseOutbreak && class == protectionCampDisease:
		add(Campcraft, 0.30)
		add(MedicinalKnowledge, 0.40)
	case kind == AcuteDiseaseOutbreak && class == protectionUncovered:
		add(MedicinalKnowledge, 0.40)
	case kind == AcuteExposureFall && class == protectionExposure:
		add(Firecraft, 0.20)
		add(TailoredClothing, 0.40)
		add(Campcraft, 0.30)
	case kind == AcuteCrossingMishap && class == protectionUncovered:
		add(CoastalNavigation, 0.50)
	}
	return combinedTechnologyMitigation(effects...)
}

func AcuteProbabilities(band Band, tile TileGeography, habitat HabitatTile, season Season, workRisk [AcuteKindCount]float64, crossed bool, passage PassageID) [AcuteKindCount]float64 {
	result := [AcuteKindCount]float64{}
	for kind := AcuteKind(0); kind < AcuteKindCount; kind++ {
		environmentWeight := 0.0
		if kind != AcuteCrossingMishap {
			environmentWeight = acuteBiomeBase[habitat.Biome][kind] * acuteSeasonFactor[season][kind]
		}
		for class := protectionClass(0); class < protectionClassCount; class++ {
			base := environmentWeight * acutePartition[kind][class]
			if class == protectionUncovered {
				base += workRisk[kind]
				if kind == AcuteCrossingMishap && crossed && passage < PassageCount {
					base += passageAcuteRisk[passage]
				}
			}
			geneticRemaining := 1.0
			if kind == AcuteDiseaseOutbreak {
				geneticRemaining = InnateImmuneRemainingRisk(float64(band.Heritable[InnateImmuneReactivity]))
			}
			result[kind] += base * remainingRisk(acuteTechnologyMitigation(band, kind, class), campMitigation(band, class, tile.NaturalShelter)) * geneticRemaining
		}
	}
	total := 0.0
	for _, probability := range result {
		total += probability
	}
	if total > MaxAcuteProbability {
		scale := MaxAcuteProbability / total
		for index := range result {
			result[index] *= scale
		}
	}
	return result
}

func ResolveAcute(band *Band, tile TileGeography, habitat HabitatTile, season Season, workRisk [AcuteKindCount]float64, crossed bool, passage PassageID, rng *WorldRNG) (AcuteKind, float64, bool, error) {
	probabilities := AcuteProbabilities(*band, tile, habitat, season, workRisk, crossed, passage)
	draw := rng.Float64()
	cumulative := 0.0
	selected := AcuteKindCount
	for kind, probability := range probabilities {
		cumulative += probability
		if draw < cumulative {
			selected = AcuteKind(kind)
			break
		}
	}
	if selected == AcuteKindCount {
		return 0, 0, false, nil
	}
	severityDraw := rng.Float64()
	lossFraction := minimumAcuteLoss[selected] + (maximumAcuteLoss[selected]-minimumAcuteLoss[selected])*severityDraw
	severityMitigation := 0.0
	switch selected {
	case AcutePredation:
		if band.Technology.Has(HaftedTools) {
			severityMitigation = 0.15
		}
	case AcuteDiseaseOutbreak:
		if band.Technology.Has(MedicinalKnowledge) {
			severityMitigation = 0.35
		}
	case AcuteCrossingMishap:
		if band.Technology.Has(CoastalNavigation) {
			severityMitigation = 0.35
		}
	}
	lossFraction *= 1 - severityMitigation
	before := float64(band.Population)
	loss := before * clamp01(lossFraction)
	if loss > before {
		loss = before
	}
	population, err := RoundPopulation(before - loss)
	if err != nil {
		return 0, 0, false, err
	}
	band.Population = population
	loss = before - float64(band.Population)
	if selected == AcuteDiseaseOutbreak && band.Population > 0 {
		healthLoss := 0.05 + 0.10*severityDraw
		if band.Technology.Has(MedicinalKnowledge) {
			healthLoss *= 0.50
		}
		healthLoss *= InnateImmuneRemainingRisk(float64(band.Heritable[InnateImmuneReactivity]))
		band.Health = Health(clamp01(float64(band.Health) - healthLoss))
	}
	return selected, loss, true, nil
}
