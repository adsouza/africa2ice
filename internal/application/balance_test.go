package application

import (
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
	if report.DesertResidency.TileID != 5302 || report.DesertResidency.StartTurn != 26 || report.DesertResidency.EndTurn != 61 {
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
	firstStartByTile := make(map[domain.TileID]int)
	laterWindows := 0
	for _, window := range windows {
		first, ok := firstStartByTile[window.tileID]
		if !ok {
			firstStartByTile[window.tileID] = window.startTurn
			continue
		}
		if window.startTurn > first {
			laterWindows++
		}
	}
	if laterWindows == 0 {
		t.Fatalf("%d qualifying windows contain no later start for the same tile", len(windows))
	}
}

func TestFaunaRecoveryMeasurementCanInvalidateDwellGate(t *testing.T) {
	report, err := BuildMoistureBalanceReport()
	if err != nil {
		t.Fatal(err)
	}
	if report.FaunaNinetyPercentTurns != faunaNinetyPercentRecoveryTurns(domain.FaunaRegenerationRate) {
		t.Fatalf("reported recovery turns = %d", report.FaunaNinetyPercentTurns)
	}
	report.FaunaNinetyPercentTurns = report.ConfiguredDwellTurns + 1
	if err := ValidateMoistureBalanceReport(report); err == nil {
		t.Fatal("dwell gate accepted a measured recovery longer than its configured floor")
	}
}
