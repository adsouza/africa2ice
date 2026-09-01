package application

import (
	"fmt"
	"math"
	"runtime"
	"sync"

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
	for _, value := range []float64{
		desert.EndingPopulation, desert.EndingHealth,
		desert.MinimumFoodCoverage, desert.MeanFoodCoverage,
		desert.MinimumWaterCoverage, desert.MeanWaterCoverage,
	} {
		if !finite(value) {
			return fmt.Errorf("desert residency contains a non-finite or negative value")
		}
	}
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
// the same domain APIs and complete World.AdvanceTurn transition used by the
// game. Its isolated scenarios never touch a player's world.
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
	report.FaunaNinetyPercentTurns = faunaNinetyPercentRecoveryTurns(domain.FaunaRegenerationRate)
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

// faunaNinetyPercentRecoveryTurns measures the discrete toward-cap recurrence
// independently of the logarithmic dwell-floor derivation. Keeping both paths
// lets the balance gate catch a changed recurrence or an off-by-one derivation.
func faunaNinetyPercentRecoveryTurns(rate float64) int {
	stock := 0.0
	for turns := 0; ; turns++ {
		if stock >= 0.90 {
			return turns
		}
		stock += rate * (1 - stock)
	}
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
	template, err := domain.NewWorld(seed)
	if err != nil {
		return DesertResidencyReport{}, err
	}
	templateState, err := template.ExportState()
	if err != nil {
		return DesertResidencyReport{}, err
	}
	windows := qualifyingDesertWindows(grid, history)
	if len(windows) == 0 {
		return DesertResidencyReport{}, nil
	}
	type simulationResult struct {
		report DesertResidencyReport
		err    error
	}
	results := make([]simulationResult, len(windows))
	jobs := make(chan int)
	workerCount := min(runtime.GOMAXPROCS(0), len(windows))
	var workers sync.WaitGroup
	workers.Add(workerCount)
	for range workerCount {
		go func() {
			defer workers.Done()
			for index := range jobs {
				results[index].report, results[index].err = runDesertWindowThroughWorld(grid, history, seed, templateState.RNGState, windows[index])
			}
		}()
	}
	for index := range windows {
		jobs <- index
	}
	close(jobs)
	workers.Wait()

	best := DesertResidencyReport{}
	for _, result := range results {
		if result.err != nil {
			return DesertResidencyReport{}, result.err
		}
		if betterDesertResidency(result.report, best) {
			best = result.report
		}
	}
	return best, nil
}

func betterDesertResidency(candidate, best DesertResidencyReport) bool {
	if !best.Found || candidate.EndingHealth != best.EndingHealth {
		return !best.Found || candidate.EndingHealth > best.EndingHealth
	}
	if candidate.EndingPopulation != best.EndingPopulation {
		return candidate.EndingPopulation > best.EndingPopulation
	}
	if candidate.TileID != best.TileID {
		return candidate.TileID < best.TileID
	}
	if candidate.StartTurn != best.StartTurn {
		return candidate.StartTurn < best.StartTurn
	}
	return candidate.EndTurn < best.EndTurn
}

func qualifyingDesertWindows(grid *domain.Grid, history []*domain.Habitat) []desertWindow {
	windows := make([]desertWindow, 0)
	for id := range domain.TileCount {
		geography, _ := grid.Tile(domain.TileID(id))
		if !geography.Land {
			continue
		}
		for start := 0; start <= domain.MaxCampaignTurn; {
			if history[start][id].Biome != domain.SemiAridDesert {
				start++
				continue
			}
			runEnd := start
			for runEnd+1 <= domain.MaxCampaignTurn && history[runEnd+1][id].Biome == domain.SemiAridDesert {
				runEnd++
			}
			startDate, _ := domain.CampaignDate(start)
			for end := start; end <= runEnd; end++ {
				endDate, _ := domain.CampaignDate(end)
				years := startDate.YearBP - endDate.YearBP
				if years >= 10_500 {
					windows = append(windows, desertWindow{tileID: domain.TileID(id), region: geography.Region, startTurn: start, endTurn: end, years: years})
					break
				}
			}
			start = runEnd + 1
		}
	}
	return windows
}

func runDesertWindowThroughWorld(grid *domain.Grid, history []*domain.Habitat, seed uint64, rngState []byte, window desertWindow) (DesertResidencyReport, error) {
	const population = domain.Population(40)
	allocation := [domain.AssignmentCount]domain.AssignmentBP{3500, 3000, 1500, 500, 1500}
	traits, ok := domain.StartingHeritableState(domain.HomoSapiens, window.region)
	if !ok {
		traits, _ = domain.StartingHeritableState(domain.HomoSapiens, domain.EastAfrica)
	}
	result := DesertResidencyReport{Found: true, TileID: uint16(window.tileID), Region: gameapi.Region(window.region).String(), StartTurn: window.startTurn, EndTurn: window.endTurn, CalendarYears: window.years, Population: int(population), MinimumFoodCoverage: 1, MinimumWaterCoverage: 1}
	state := domain.State{
		Seed: seed, Turn: window.startTurn, Result: domain.CampaignOngoing, NextBandID: 2,
		Bands: []domain.Band{{
			ID: 1, Species: domain.HomoSapiens, TileID: window.tileID,
			Population: population, Health: 1, Allocation: allocation, Heritable: traits,
		}},
		RNGState: append([]byte(nil), rngState...),
	}
	season, err := domain.SeasonForTurn(window.startTurn)
	if err != nil {
		return DesertResidencyReport{}, err
	}
	for id := range domain.TileCount {
		geography, _ := grid.Tile(domain.TileID(id))
		state.Tiles[id] = domain.InitialTileState(seed, geography, history[window.startTurn][id], season)
	}
	world, err := domain.RestoreWorld(state)
	if err != nil {
		return DesertResidencyReport{}, fmt.Errorf("restore desert tile %d at turn %d: %w", window.tileID, window.startTurn, err)
	}
	count := 0
	for world.Turn() < window.endTurn {
		if err := world.AdvanceTurn(); err != nil {
			return DesertResidencyReport{}, fmt.Errorf("advance desert tile %d to turn %d: %w", window.tileID, world.Turn()+1, err)
		}
		bands := world.Bands()
		if len(bands) == 0 {
			result.MinimumFoodCoverage = 0
			result.MinimumWaterCoverage = 0
			count++
			break
		}
		band := bands[0]
		foodCoverage, coverageErr := coverageAfterDeficit(band.LastFoodReport.RequiredFU, band.LastFoodReport.DeficitFU)
		if coverageErr != nil {
			return DesertResidencyReport{}, fmt.Errorf("desert tile %d turn %d food coverage: %w", window.tileID, world.Turn(), coverageErr)
		}
		waterCoverage, coverageErr := coverageAfterDeficit(domain.WaterHealthLossRate, band.LastOutcomeReport.WaterHealthLoss)
		if coverageErr != nil {
			return DesertResidencyReport{}, fmt.Errorf("desert tile %d turn %d water coverage: %w", window.tileID, world.Turn(), coverageErr)
		}
		result.MinimumFoodCoverage = min(result.MinimumFoodCoverage, foodCoverage)
		result.MinimumWaterCoverage = min(result.MinimumWaterCoverage, waterCoverage)
		result.MeanFoodCoverage += foodCoverage
		result.MeanWaterCoverage += waterCoverage
		count++
	}
	if count == 0 {
		return DesertResidencyReport{}, fmt.Errorf("desert tile %d has an empty residency interval", window.tileID)
	}
	result.MeanFoodCoverage /= float64(count)
	result.MeanWaterCoverage /= float64(count)
	if bands := world.Bands(); len(bands) != 0 {
		result.EndingPopulation = float64(bands[0].Population)
		result.EndingHealth = float64(bands[0].Health)
		result.Survived = bands[0].Population > 0
	}
	return result, nil
}

func coverageAfterDeficit(total, deficit float64) (float64, error) {
	if !finite(total) || !finite(deficit) || total <= 0 || deficit > total {
		return 0, fmt.Errorf("invalid total %.6f or deficit %.6f", total, deficit)
	}
	coverage := (total - deficit) / total
	if !finite(coverage) || coverage > 1 {
		return 0, fmt.Errorf("invalid derived coverage %.6f", coverage)
	}
	return coverage, nil
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
