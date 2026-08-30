package domain

import "testing"

func TestCampaignDateBoundaries(t *testing.T) {
	tests := []struct {
		turn int
		year int
		era  CampaignEra
	}{{0, 80_000, EraEarly}, {100, 50_000, EraMiddle}, {200, 35_000, EraLate}, {300, 25_000, EraFinal}, {400, 20_000, EraFinal}}
	for _, test := range tests {
		got, err := CampaignDate(test.turn)
		if err != nil || got.YearBP != test.year || got.Era != test.era {
			t.Fatalf("CampaignDate(%d) = %#v, %v", test.turn, got, err)
		}
	}
	if _, err := CampaignDate(401); err == nil {
		t.Fatal("turn 401 accepted")
	}
}

func TestSeasonCycle(t *testing.T) {
	for turn, want := range []Season{SeasonWarm, SeasonWarm, SeasonWarm, SeasonCooling, SeasonCooling, SeasonCooling, SeasonCold, SeasonCold, SeasonCold, SeasonWarming, SeasonWarming, SeasonWarming} {
		got, err := SeasonForTurn(turn)
		if err != nil || got != want {
			t.Fatalf("SeasonForTurn(%d) = %v, %v", turn, got, err)
		}
	}
}
