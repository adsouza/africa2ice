package application

import (
	"fmt"
	"math"

	"github.com/adsouza/africa2ice/internal/domain"
	"github.com/adsouza/africa2ice/pkg/gameapi"
)

const moistureBalanceReportVersion = 1

type MoistureBalanceReport struct {
	Version                 int                     `json:"version"`
	Seed                    uint64                  `json:"seed"`
	Samples                 []MoistureBalanceSample `json:"samples"`
	DerivedDwellTurns       int                     `json:"derived_dwell_turns"`
	ConfiguredDwellTurns    int                     `json:"configured_dwell_turns"`
	FaunaNinetyPercentTurns int                     `json:"fauna_ninety_percent_turns"`
	DesertResidency         DesertResidencyReport   `json:"desert_residency"`
	Recovery                RecoveryReport          `json:"recovery"`
}

func ValidateMoistureBalanceReport(report MoistureBalanceReport) error {
	if report.Version != moistureBalanceReportVersion || len(report.Samples) != 5 {
		return fmt.Errorf("invalid moisture report shape")
	}
	for _, sample := range report.Samples {
		if sample.LandTiles <= 0 || sample.FloraCapacity <= 0 || sample.FaunaCapacity <= 0 || sample.WaterCapacity <= 0 {
			return fmt.Errorf("turn %d has an empty resource envelope", sample.Turn)
		}
		total := 0
		fraction := 0.0
		for biome := range sample.BiomeTiles {
			total += sample.BiomeTiles[biome]
			fraction += sample.BiomeFraction[biome]
		}
		if total != sample.LandTiles || math.Abs(fraction-1) > 1e-12 {
			return fmt.Errorf("turn %d biome mix does not cover the land raster", sample.Turn)
		}
	}
	if report.DerivedDwellTurns != report.ConfiguredDwellTurns || report.ConfiguredDwellTurns < report.FaunaNinetyPercentTurns {
		return fmt.Errorf("biome dwell floor no longer covers fauna recovery")
	}
	desert := report.DesertResidency
	if !desert.Found || desert.CalendarYears < 10_500 || !desert.Survived || desert.EndingPopulation < 1 || desert.EndingHealth <= 0 {
		return fmt.Errorf("no survivable full half-cycle desert residency was found")
	}
	if desert.MinimumFoodCoverage <= 0 || desert.MinimumFoodCoverage >= 1 || desert.MinimumWaterCoverage <= 0 || desert.MinimumWaterCoverage >= 1 {
		return fmt.Errorf("desert residency is not pressured in both food and water")
	}
	if !report.Recovery.OscillatingWithinSteady {
		return fmt.Errorf("an oscillating tile recovers farther from fauna cap than its steady-biome counterfactual")
	}
	return nil
}

type MoistureBalanceSample struct {
	Turn          int                         `json:"turn"`
	YearBP        int                         `json:"year_bp"`
	LandTiles     int                         `json:"land_tiles"`
	BiomeTiles    [gameapi.BiomeCount]int     `json:"biome_tiles"`
	BiomeFraction [gameapi.BiomeCount]float64 `json:"biome_fraction"`
	FloraCapacity float64                     `json:"flora_capacity"`
	FaunaCapacity float64                     `json:"fauna_capacity"`
	WaterCapacity float64                     `json:"water_capacity"`
}

type DesertResidencyReport struct {
	Found                bool    `json:"found"`
	TileID               uint16  `json:"tile_id"`
	Region               string  `json:"region"`
	StartTurn            int     `json:"start_turn"`
	EndTurn              int     `json:"end_turn"`
	CalendarYears        int     `json:"calendar_years"`
	Population           int     `json:"population"`
	EndingPopulation     float64 `json:"ending_population"`
	EndingHealth         float64 `json:"ending_health"`
	Survived             bool    `json:"survived"`
	MinimumFoodCoverage  float64 `json:"minimum_food_coverage"`
	MeanFoodCoverage     float64 `json:"mean_food_coverage"`
	MinimumWaterCoverage float64 `json:"minimum_water_coverage"`
	MeanWaterCoverage    float64 `json:"mean_water_coverage"`
}

type RecoveryReport struct {
	HalfCycleEndTurn        int     `json:"half_cycle_end_turn"`
	SteadyFaunaGap          float64 `json:"steady_fauna_gap"`
	MaximumOscillatingGap   float64 `json:"maximum_oscillating_gap"`
	OscillatingWithinSteady bool    `json:"oscillating_within_steady"`
}

// BuildMoistureBalanceReport calculates the step-12 calibration evidence from
// the same domain APIs used by the game. It does not advance or mutate a world.
func BuildMoistureBalanceReport() (MoistureBalanceReport, error) {
	grid, err := (domain.WorldGenerator{}).Generate()
	if err != nil {
		return MoistureBalanceReport{}, err
	}
	report := MoistureBalanceReport{Version: moistureBalanceReportVersion, Seed: domain.BalanceSeedCorpus[0]}
	for _, turn := range []int{0, 100, 200, 300, 400} {
		sample, sampleErr := moistureBalanceSample(grid, report.Seed, turn)
		if sampleErr != nil {
			return MoistureBalanceReport{}, sampleErr
		}
		report.Samples = append(report.Samples, sample)
	}
	report.DerivedDwellTurns = int(math.Ceil(math.Log(0.10) / math.Log(1-domain.FaunaRegenerationRate)))
	report.FaunaNinetyPercentTurns = report.DerivedDwellTurns
	report.ConfiguredDwellTurns = domain.MinBiomeDwellTurns
	report.DesertResidency, err = desertResidencyReport(grid, report.Seed)
	if err != nil {
		return MoistureBalanceReport{}, err
	}
	report.Recovery, err = recoveryReport(grid, report.Seed)
	if err != nil {
		return MoistureBalanceReport{}, err
	}
	return report, nil
}

func moistureBalanceSample(grid *domain.Grid, seed uint64, turn int) (MoistureBalanceSample, error) {
	habitat, _, err := domain.BuildHabitat(grid, seed, turn)
	if err != nil {
		return MoistureBalanceSample{}, err
	}
	date, err := domain.CampaignDate(turn)
	if err != nil {
		return MoistureBalanceSample{}, err
	}
	season, err := domain.SeasonForTurn(turn)
	if err != nil {
		return MoistureBalanceSample{}, err
	}
	sample := MoistureBalanceSample{Turn: turn, YearBP: date.YearBP}
	for id := range domain.TileCount {
		geography, _ := grid.Tile(domain.TileID(id))
		if !geography.Land {
			continue
		}
		sample.LandTiles++
		biome := habitat[id].Biome
		sample.BiomeTiles[biome]++
		cap := domain.ResourceCaps(biome, season, 0, habitat[id].BaselineK)
		sample.FloraCapacity += cap.Flora
		sample.FaunaCapacity += cap.Fauna
		sample.WaterCapacity += cap.Water
	}
	for biome := range sample.BiomeTiles {
		sample.BiomeFraction[biome] = float64(sample.BiomeTiles[biome]) / float64(sample.LandTiles)
	}
	return sample, nil
}

type desertWindow struct {
	tileID             domain.TileID
	region             domain.Region
	startTurn, endTurn int
	years              int
}

func desertResidencyReport(grid *domain.Grid, seed uint64) (DesertResidencyReport, error) {
	history := make([]*domain.Habitat, domain.MaxCampaignTurn+1)
	for turn := 0; turn <= domain.MaxCampaignTurn; turn++ {
		habitat, _, err := domain.BuildHabitat(grid, seed, turn)
		if err != nil {
			return DesertResidencyReport{}, err
		}
		history[turn] = habitat
	}
	best := DesertResidencyReport{}
	for id := range domain.TileCount {
		geography, _ := grid.Tile(domain.TileID(id))
		if !geography.Land {
			continue
		}
		for start := 0; start <= domain.MaxCampaignTurn; start++ {
			if history[start][id].Biome != domain.SemiAridDesert {
				continue
			}
			startDate, _ := domain.CampaignDate(start)
			for end := start; end <= domain.MaxCampaignTurn && history[end][id].Biome == domain.SemiAridDesert; end++ {
				endDate, _ := domain.CampaignDate(end)
				years := startDate.YearBP - endDate.YearBP
				if years >= 10_500 {
					candidate := simulateDesertWindow(grid, history, seed, desertWindow{tileID: domain.TileID(id), region: geography.Region, startTurn: start, endTurn: end, years: years})
					if !best.Found || candidate.EndingHealth > best.EndingHealth || candidate.EndingHealth == best.EndingHealth && candidate.EndingPopulation > best.EndingPopulation || candidate.EndingHealth == best.EndingHealth && candidate.EndingPopulation == best.EndingPopulation && candidate.TileID < best.TileID {
						best = candidate
					}
					break
				}
			}
			break
		}
	}
	return best, nil
}

func simulateDesertWindow(grid *domain.Grid, history []*domain.Habitat, seed uint64, window desertWindow) DesertResidencyReport {
	const population = domain.Population(40)
	allocation := [domain.AssignmentCount]domain.AssignmentBP{3500, 3000, 1500, 500, 1500}
	traits, ok := domain.StartingHeritableState(domain.HomoSapiens, window.region)
	if !ok {
		traits, _ = domain.StartingHeritableState(domain.HomoSapiens, domain.EastAfrica)
	}
	band := domain.Band{Population: population, Allocation: allocation, Heritable: traits}
	simulatedPopulation := float64(population)
	health := 1.0
	geography, _ := grid.Tile(window.tileID)
	startSeason, _ := domain.SeasonForTurn(window.startTurn)
	state := domain.InitialTileState(seed, geography, history[window.startTurn][window.tileID], startSeason)
	result := DesertResidencyReport{Found: true, TileID: uint16(window.tileID), Region: gameapi.Region(window.region).String(), StartTurn: window.startTurn, EndTurn: window.endTurn, CalendarYears: window.years, Population: int(population), MinimumFoodCoverage: 1, MinimumWaterCoverage: 1}
	count := 0
	for turn := window.startTurn; turn <= window.endTurn; turn++ {
		band.Population = domain.Population(max(1, int(math.Round(simulatedPopulation))))
		habitat := history[turn][window.tileID]
		season, _ := domain.SeasonForTurn(turn)
		cap := domain.ResourceCaps(habitat.Biome, season, 0, habitat.BaselineK)
		regenerated := state.Regenerate(cap, domain.ResourceVector{})
		profile, _ := domain.FaunaFor(window.region, habitat.Biome, true)
		hunting := domain.HuntingRates(band, window.region, profile)
		floraDemand := band.Workers(domain.Foraging) * domain.ForagingRate(band, habitat.Biome)
		faunaDemand := band.Workers(domain.HuntingAndFishing) * hunting.Total
		faunaDemand += band.Workers(domain.MegafaunaTracking) * domain.MegafaunaRate(band, profile, hunting.Terrestrial)
		waterDemand := domain.WaterRequired(band, habitat.LocalTemperatureC)
		flora := min(floraDemand, regenerated.Stock.Flora)
		fauna := min(faunaDemand, regenerated.Stock.Fauna)
		water := min(waterDemand, regenerated.Stock.Water)
		food := flora * domain.FattyAcidConversion(domain.PlantFood, float64(band.Heritable[domain.FattyAcidMetabolism]))
		food += fauna * domain.FattyAcidConversion(domain.AnimalFood, float64(band.Heritable[domain.FattyAcidMetabolism]))
		foodCoverage := min(1, food/simulatedPopulation)
		waterCoverage := min(1, water/waterDemand)
		foodDeficit := 1 - foodCoverage
		waterDeficit := 1 - waterCoverage
		health = max(0, min(1, health+domain.NutritionHealthDelta(foodDeficit)-domain.WaterHealthLossRate*waterDeficit))
		growth := domain.LogisticGrowth(simulatedPopulation, simulatedPopulation, habitat.BaselineK, foodDeficit)
		simulatedPopulation = max(0, simulatedPopulation+growth-domain.StarvationLoss(simulatedPopulation, foodDeficit))
		result.MinimumFoodCoverage = min(result.MinimumFoodCoverage, foodCoverage)
		result.MinimumWaterCoverage = min(result.MinimumWaterCoverage, waterCoverage)
		result.MeanFoodCoverage += foodCoverage
		result.MeanWaterCoverage += waterCoverage
		count++
		regenerated.Stock.Flora -= flora
		regenerated.Stock.Fauna -= fauna
		regenerated.Stock.Water -= water
		state = regenerated
	}
	result.MeanFoodCoverage /= float64(count)
	result.MeanWaterCoverage /= float64(count)
	result.EndingPopulation = simulatedPopulation
	result.EndingHealth = health
	result.Survived = simulatedPopulation >= 1
	return result
}

func recoveryReport(grid *domain.Grid, seed uint64) (RecoveryReport, error) {
	endTurn := 0
	for turn := 0; turn <= domain.MaxCampaignTurn; turn++ {
		date, _ := domain.CampaignDate(turn)
		if 80_000-date.YearBP >= 10_500 {
			endTurn = turn
			break
		}
	}
	history := make([]*domain.Habitat, endTurn+1)
	for turn := 0; turn <= endTurn; turn++ {
		habitat, _, err := domain.BuildHabitat(grid, seed, turn)
		if err != nil {
			return RecoveryReport{}, err
		}
		history[turn] = habitat
	}
	actualStocks := [domain.TileCount]float64{}
	steadyStocks := [domain.TileCount]float64{}
	maximumActualGap := 0.0
	maximumSteadyGap := 0.0
	withinSteady := true
	for turn := 0; turn <= endTurn; turn++ {
		season, _ := domain.SeasonForTurn(turn)
		for id := range domain.TileCount {
			geography, _ := grid.Tile(domain.TileID(id))
			if !geography.Land || history[turn][id].BaselineK <= 0 {
				continue
			}
			actualCap := domain.ResourceCaps(history[turn][id].Biome, season, 0, history[turn][id].BaselineK).Fauna
			steadyBiome := history[endTurn][id].Biome
			steadyCap := domain.ResourceCaps(steadyBiome, season, 0, history[turn][id].BaselineK).Fauna
			actualStock := min(actualStocks[id], actualCap)
			steadyStock := min(steadyStocks[id], steadyCap)
			actualStocks[id] = actualStock + domain.FaunaRegenerationRate*(actualCap-actualStock)
			steadyStocks[id] = steadyStock + domain.FaunaRegenerationRate*(steadyCap-steadyStock)
			if turn == endTurn && actualCap > 0 && steadyCap > 0 {
				actualGap := (actualCap - actualStocks[id]) / actualCap
				steadyGap := (steadyCap - steadyStocks[id]) / steadyCap
				maximumActualGap = max(maximumActualGap, actualGap)
				maximumSteadyGap = max(maximumSteadyGap, steadyGap)
				if actualGap > steadyGap+1e-12 {
					withinSteady = false
				}
			}
		}
	}
	return RecoveryReport{HalfCycleEndTurn: endTurn, SteadyFaunaGap: maximumSteadyGap, MaximumOscillatingGap: maximumActualGap, OscillatingWithinSteady: withinSteady}, nil
}
