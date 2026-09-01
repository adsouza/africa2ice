package render

import (
	"testing"

	"github.com/adsouza/africa2ice/pkg/gameapi"
)

func TestVisibleSapiensBandWindowPagesToTheSelection(t *testing.T) {
	bands := interleavedBandWindowFixture(12)
	tests := []struct {
		name        string
		selected    gameapi.BandID
		wantFirst   int
		wantLabel   string
		wantBandIDs []gameapi.BandID
	}{
		{name: "first page", selected: 1, wantFirst: 0, wantLabel: "Bands 1–5/12", wantBandIDs: []gameapi.BandID{1, 2, 3, 4, 5}},
		{name: "second page", selected: 6, wantFirst: 5, wantLabel: "Bands 6–10/12", wantBandIDs: []gameapi.BandID{6, 7, 8, 9, 10}},
		{name: "partial final page", selected: 12, wantFirst: 10, wantLabel: "Bands 11–12/12", wantBandIDs: []gameapi.BandID{11, 12}},
		{name: "missing selection uses first page", selected: 999, wantFirst: 0, wantLabel: "Bands 1–5/12", wantBandIDs: []gameapi.BandID{1, 2, 3, 4, 5}},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			window := visibleSapiensBandWindow(bands, test.selected)
			if window.total != 12 || window.first != test.wantFirst || window.count != len(test.wantBandIDs) || window.label() != test.wantLabel {
				t.Fatalf("window = %+v", window)
			}
			for row, wantID := range test.wantBandIDs {
				band := bands[window.indices[row]]
				if band.Species != gameapi.HomoSapiens || band.ID != wantID {
					t.Fatalf("row %d = %#v, want sapiens band %d", row, band, wantID)
				}
			}
		})
	}
}

func TestVisibleSapiensBandWindowDoesNotPageSmallLists(t *testing.T) {
	bands := interleavedBandWindowFixture(4)
	window := visibleSapiensBandWindow(bands, 4)
	if window.total != 4 || window.first != 0 || window.count != 4 || window.label() != "Bands: 4" {
		t.Fatalf("window = %+v", window)
	}
}

func TestMaximumBandPageFitsBeforeOutcomeDetails(t *testing.T) {
	if bandListOriginY+maxVisibleSapiensBandRows*bandRowHeight > bandOutcomeOriginY {
		t.Fatal("band page overlaps the outcome details")
	}
}

func TestSapiensBandActionLabelDistinguishesPlanningStates(t *testing.T) {
	tests := []struct {
		name string
		band gameapi.Band
		want string
	}{
		{name: "ready", band: gameapi.Band{}, want: bandActionReady},
		{name: "queued migration", band: gameapi.Band{SpatialActionUsed: true, HasQueuedMigration: true}, want: bandActionMoveSet},
		{name: "accepted interbreeding", band: gameapi.Band{SpatialActionUsed: true, HasInterbreedTarget: true}, want: bandActionInterbreed},
		{name: "other completed action", band: gameapi.Band{SpatialActionUsed: true}, want: bandActionDone},
		{name: "exact intent wins", band: gameapi.Band{SpatialActionUsed: true, HasQueuedMigration: true, HasInterbreedTarget: true}, want: bandActionMoveSet},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if got := sapiensBandActionLabel(test.band); got != test.want {
				t.Fatalf("sapiensBandActionLabel(%#v) = %q, want %q", test.band, got, test.want)
			}
		})
	}
}

func TestConditionForSapiensBandUsesActualSufferingAndProjectedDanger(t *testing.T) {
	tests := []struct {
		name string
		band gameapi.Band
		want sapiensBandCondition
	}{
		{name: "stable", band: gameapi.Band{Health: 1, SeasonalMortalityRate: 0.001, ChronicMortalityRate: 0.001}, want: bandConditionStable},
		{name: "population fell", band: gameapi.Band{Health: 1, LastOutcomeReport: gameapi.OutcomeReport{Turn: 4, StartingPopulation: 20, EndingPopulation: 19, StartingHealth: 1, EndingHealth: 1}}, want: bandConditionSuffering},
		{name: "health fell", band: gameapi.Band{Health: 0.95, LastOutcomeReport: gameapi.OutcomeReport{Turn: 4, StartingPopulation: 20, EndingPopulation: 20, StartingHealth: 1, EndingHealth: 0.95}}, want: bandConditionSuffering},
		{name: "food shortfall", band: gameapi.Band{Health: 1, LastFoodReport: gameapi.FoodTurnReport{Turn: 4, RequiredFU: 20, DeficitFU: 1}}, want: bandConditionSuffering},
		{name: "critically low health", band: gameapi.Band{Health: 0.49}, want: bandConditionSuffering},
		{name: "reduced health", band: gameapi.Band{Health: 0.79}, want: bandConditionDanger},
		{name: "projected mortality", band: gameapi.Band{Health: 1, SeasonalMortalityRate: 0.0025, ChronicMortalityRate: 0.0015}, want: bandConditionDanger},
		{name: "danger boundary", band: gameapi.Band{Health: bandDangerHealthThreshold, SeasonalMortalityRate: bandDangerMortalityRate}, want: bandConditionDanger},
		{name: "suffering overrides danger", band: gameapi.Band{Health: 0.79, SeasonalMortalityRate: 0.01, LastFoodReport: gameapi.FoodTurnReport{Turn: 4, RequiredFU: 20, DeficitFU: 1}}, want: bandConditionSuffering},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if got := conditionForSapiensBand(test.band); got != test.want {
				t.Fatalf("conditionForSapiensBand(%#v) = %d, want %d", test.band, got, test.want)
			}
		})
	}
}

func interleavedBandWindowFixture(sapiensCount int) []gameapi.Band {
	bands := make([]gameapi.Band, 0, sapiensCount*2)
	for id := 1; id <= sapiensCount; id++ {
		bands = append(bands,
			gameapi.Band{ID: gameapi.BandID(10_000 + id), Species: gameapi.ArchaicHominin},
			gameapi.Band{ID: gameapi.BandID(id), Species: gameapi.HomoSapiens},
		)
	}
	return bands
}
