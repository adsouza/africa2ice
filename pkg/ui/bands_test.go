package ui

import (
	"testing"

	"github.com/adsouza/africa2ice/pkg/gameapi"
)

func TestConditionForSapiensBandUsesSpecTiers(t *testing.T) {
	tests := []struct {
		name string
		band gameapi.Band
		want SapiensBandCondition
	}{
		{name: "healthy", band: gameapi.Band{Health: 0.95}, want: BandStable},
		{name: "health below 0.80 is danger", band: gameapi.Band{Health: 0.79}, want: BandDanger},
		{name: "mortality at 0.004 is danger", band: gameapi.Band{Health: 1, SeasonalMortalityRate: 0.002, ChronicMortalityRate: 0.002}, want: BandDanger},
		{name: "health below 0.50 is suffering", band: gameapi.Band{Health: 0.49}, want: BandSuffering},
		{name: "food shortfall last turn is suffering", band: gameapi.Band{Health: 1, LastFoodReport: gameapi.FoodTurnReport{Turn: 3, RequiredFU: 10, DeficitFU: 1}}, want: BandSuffering},
		{name: "population decline last turn is suffering", band: gameapi.Band{Health: 1, LastOutcomeReport: gameapi.OutcomeReport{Turn: 3, StartingPopulation: 100, EndingPopulation: 99, StartingHealth: 1, EndingHealth: 1}}, want: BandSuffering},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if got := ConditionForSapiensBand(test.band); got != test.want {
				t.Fatalf("condition = %d, want %d", got, test.want)
			}
		})
	}
	if BandStable.Marker() != "" || BandDanger.Marker() != "!" || BandSuffering.Marker() != "!!" {
		t.Fatal("condition markers do not match the spec's chip prefixes")
	}
}

func TestSapiensBandIDsByAttentionOrdersSufferingFirstThenHealthThenID(t *testing.T) {
	bands := []gameapi.Band{
		{ID: 1, Species: gameapi.HomoSapiens, Population: 10, Health: 0.9},
		{ID: 2, Species: gameapi.ArchaicHominin, Population: 10, Health: 0.1},
		{ID: 3, Species: gameapi.HomoSapiens, Population: 10, Health: 0.4},
		{ID: 4, Species: gameapi.HomoSapiens, Population: 0, Health: 0.1},
		{ID: 5, Species: gameapi.HomoSapiens, Population: 10, Health: 0.9},
		{ID: 6, Species: gameapi.HomoSapiens, Population: 10, Health: 0.7},
	}
	got := SapiensBandIDsByAttention(bands)
	want := []gameapi.BandID{3, 6, 1, 5}
	if len(got) != len(want) {
		t.Fatalf("order = %v, want %v", got, want)
	}
	for index := range want {
		if got[index] != want[index] {
			t.Fatalf("order = %v, want %v", got, want)
		}
	}
	if ordinal, ok := SapiensBandOrdinal(bands, 5); !ok || ordinal != 3 {
		t.Fatalf("ordinal of band 5 = (%d, %t), want (3, true)", ordinal, ok)
	}
	if _, ok := SapiensBandOrdinal(bands, 2); ok {
		t.Fatal("archaic band received an attention ordinal")
	}
}

func TestMoveDoneAndBandsNeedingMove(t *testing.T) {
	if MoveDone(gameapi.Band{}) {
		t.Fatal("fresh band reported as done")
	}
	for _, band := range []gameapi.Band{{SpatialActionUsed: true}, {HasQueuedMigration: true}, {HasInterbreedTarget: true}} {
		if !MoveDone(band) {
			t.Fatalf("band %+v not reported as done", band)
		}
	}
	bands := []gameapi.Band{
		{ID: 1, Species: gameapi.HomoSapiens, Population: 5},
		{ID: 2, Species: gameapi.HomoSapiens, Population: 5, HasQueuedMigration: true},
		{ID: 3, Species: gameapi.ArchaicHominin, Population: 5},
		{ID: 4, Species: gameapi.HomoSapiens, Population: 0},
	}
	if got := BandsNeedingMove(bands); got != 1 {
		t.Fatalf("BandsNeedingMove = %d, want 1", got)
	}
}
