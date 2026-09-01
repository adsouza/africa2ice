package application

import (
	"math"
	"testing"

	"github.com/adsouza/africa2ice/internal/domain"
)

func TestMoistureBalanceGate(t *testing.T) {
	report, err := BuildMoistureBalanceReport()
	if err != nil {
		t.Fatal(err)
	}
	if err := ValidateMoistureBalanceReport(report); err != nil {
		t.Fatal(err)
	}
	for index, turn := range []int{0, 100, 200, 300, 400} {
		if report.Samples[index].Turn != turn {
			t.Fatalf("sample %d turn = %d, want %d", index, report.Samples[index].Turn, turn)
		}
	}
	if report.DesertResidency.TileID != 700 || report.DesertResidency.StartTurn != 0 || report.DesertResidency.EndTurn != 35 {
		t.Fatalf("best desert window = tile %d turns %d-%d", report.DesertResidency.TileID, report.DesertResidency.StartTurn, report.DesertResidency.EndTurn)
	}
}

func TestDesertResidencyConsidersLaterQualifyingWindows(t *testing.T) {
	grid, err := (domain.WorldGenerator{}).Generate()
	if err != nil {
		t.Fatal(err)
	}
	history := make([]*domain.Habitat, domain.MaxCampaignTurn+1)
	for turn := 0; turn <= domain.MaxCampaignTurn; turn++ {
		history[turn], _, err = domain.BuildHabitat(grid, domain.BalanceSeedCorpus[0], turn)
		if err != nil {
			t.Fatal(err)
		}
	}
	windows := qualifyingDesertWindows(grid, history)
	firstDesertStartByTile := make(map[domain.TileID]int)
	for id := range domain.TileCount {
		for turn := 0; turn <= domain.MaxCampaignTurn; turn++ {
			if history[turn][id].Biome == domain.SemiAridDesert {
				firstDesertStartByTile[domain.TileID(id)] = turn
				break
			}
		}
	}
	laterWindows := 0
	for _, window := range windows {
		if window.startTurn > firstDesertStartByTile[window.tileID] {
			laterWindows++
		}
	}
	if laterWindows != 37 {
		t.Fatalf("later qualifying desert runs = %d, want 37 (all windows %d)", laterWindows, len(windows))
	}
}

func TestMoistureBalanceRejectsNonFiniteDesertCoverage(t *testing.T) {
	report := validMoistureValidationReport()
	report.DesertResidency.MeanWaterCoverage = math.NaN()
	if err := ValidateMoistureBalanceReport(report); err == nil {
		t.Fatal("moisture gate accepted NaN water coverage")
	}
}

func TestCoverageAfterDeficitRejectsInvalidDenominatorsAndValues(t *testing.T) {
	for _, test := range []struct {
		total, deficit float64
	}{
		{total: 0},
		{total: math.NaN()},
		{total: 1, deficit: math.NaN()},
		{total: 1, deficit: 2},
	} {
		if _, err := coverageAfterDeficit(test.total, test.deficit); err == nil {
			t.Fatalf("coverageAfterDeficit(%v, %v) succeeded", test.total, test.deficit)
		}
	}
	if got, err := coverageAfterDeficit(40, 10); err != nil || got != 0.75 {
		t.Fatalf("coverageAfterDeficit(40, 10) = (%v, %v)", got, err)
	}
}

func TestFaunaRecoveryMeasurementCanInvalidateDwellGate(t *testing.T) {
	report := validMoistureValidationReport()
	report.FaunaNinetyPercentTurns = faunaNinetyPercentRecoveryTurns(domain.FaunaRegenerationRate)
	report.DerivedDwellTurns = report.FaunaNinetyPercentTurns
	report.ConfiguredDwellTurns = report.FaunaNinetyPercentTurns
	if report.FaunaNinetyPercentTurns != faunaNinetyPercentRecoveryTurns(domain.FaunaRegenerationRate) {
		t.Fatalf("reported recovery turns = %d", report.FaunaNinetyPercentTurns)
	}
	report.FaunaNinetyPercentTurns = report.ConfiguredDwellTurns + 1
	if err := ValidateMoistureBalanceReport(report); err == nil {
		t.Fatal("dwell gate accepted a measured recovery longer than its configured floor")
	}
}

func validMoistureValidationReport() MoistureBalanceReport {
	sample := MoistureBalanceSample{LandTiles: 1, FloraCapacity: 1, FaunaCapacity: 1, WaterCapacity: 1}
	sample.BiomeTiles[0] = 1
	sample.BiomeFraction[0] = 1
	return MoistureBalanceReport{
		Version: moistureBalanceReportVersion, Samples: []MoistureBalanceSample{sample, sample, sample, sample, sample},
		DerivedDwellTurns: 1, ConfiguredDwellTurns: 1, FaunaNinetyPercentTurns: 1,
		DesertResidency: DesertResidencyReport{
			Found: true, CalendarYears: 10_500, Survived: true, EndingPopulation: 1, EndingHealth: 0.5,
			MinimumFoodCoverage: 0.5, MeanFoodCoverage: 0.75, MinimumWaterCoverage: 0.5, MeanWaterCoverage: 0.75,
		},
		Recovery: RecoveryReport{OscillatingWithinSteady: true},
	}
}
