package domain

import "testing"

func TestViabilityCalibrationConstants(t *testing.T) {
	if PopulationGrowthRate != 0.020 || SeasonalMortalityScale != 0.10 ||
		ChronicMortalityScale != 0.10 || AcuteProbabilityScale != 0.10 ||
		SplitStressThreshold != 0.67 || MaxCrowdingDeclineFraction != 0.25 {
		t.Fatalf("viability calibration drifted: growth=%v seasonal=%v chronic=%v acute=%v split=%v crowdingDecline=%v",
			PopulationGrowthRate, SeasonalMortalityScale, ChronicMortalityScale, AcuteProbabilityScale,
			SplitStressThreshold, MaxCrowdingDeclineFraction)
	}
}

func TestShelterCurveUsesNaturalShelterOnlyAsEfficiency(t *testing.T) {
	if got := ShelterCurve(0, 1); got != 0 {
		t.Fatalf("zero work mitigation = %v", got)
	}
	plain := ShelterCurve(0.20, 0)
	cave := ShelterCurve(0.20, 1)
	if !(plain > 0 && cave > plain && cave < MaxShelterMitigation) {
		t.Fatalf("plain/cave mitigation = %v/%v", plain, cave)
	}
}

func TestHealthAndMortalityProtectionAreComponentSpecific(t *testing.T) {
	world, _ := NewWorld(7)
	band := world.bands[0]
	tile, _ := world.grid.Tile(band.TileID)
	habitat := world.habitat[band.TileID]
	season, _ := SeasonForTurn(0)
	diseaseWithout, _ := Phase3HealthLosses(band, tile, habitat, season)
	seasonalWithout, chronicWithout := Phase3MortalityRates(band, tile, habitat, season)
	band.Allocation = [AssignmentCount]AssignmentBP{2000, 2000, 2000, 0, 4000}
	diseaseWith, _ := Phase3HealthLosses(band, tile, habitat, season)
	seasonalWith, chronicWith := Phase3MortalityRates(band, tile, habitat, season)
	if diseaseWith >= diseaseWithout || seasonalWith >= seasonalWithout || chronicWith >= chronicWithout {
		t.Fatalf("shelter did not reduce covered risk: disease %v/%v seasonal %v/%v chronic %v/%v", diseaseWithout, diseaseWith, seasonalWithout, seasonalWith, chronicWithout, chronicWith)
	}
}

func TestAcuteProbabilitiesAreCappedAndCrossingSpecific(t *testing.T) {
	world, _ := NewWorld(8)
	band := world.bands[0]
	tile, _ := world.grid.Tile(band.TileID)
	habitat := world.habitat[band.TileID]
	season, _ := SeasonForTurn(0)
	work := [AcuteKindCount]float64{1, 1, 1, 1, 0}
	ordinary := AcuteProbabilities(band, tile, habitat, season, work, false, NorthWallacea, 0)
	crossing := AcuteProbabilities(band, tile, habitat, season, work, true, NorthWallacea, 0)
	total := 0.0
	for _, probability := range crossing {
		total += probability
	}
	if total > MaxAcuteProbability+1e-12 || ordinary[AcuteCrossingMishap] != 0 || crossing[AcuteCrossingMishap] <= 0 {
		t.Fatalf("ordinary/crossing probabilities = %#v / %#v", ordinary, crossing)
	}
}

func TestInnateImmuneReactivityReducesChronicUncoveredDisease(t *testing.T) {
	world, _ := NewWorld(9)
	band := world.bands[0]
	tile, _ := world.grid.Tile(band.TileID)
	habitat := world.habitat[band.TileID]
	season, _ := SeasonForTurn(0)
	original := chronicRisk[habitat.Biome]
	chronicRisk[habitat.Biome] = chronicRiskRow{uncovered: 0.02}
	defer func() { chronicRisk[habitat.Biome] = original }()
	band.Heritable[PigmentationLevel] = 1

	band.Heritable[InnateImmuneReactivity] = 0
	_, without := Phase3MortalityRates(band, tile, habitat, season)
	band.Heritable[InnateImmuneReactivity] = 1
	_, with := Phase3MortalityRates(band, tile, habitat, season)
	want := float64(without * InnateImmuneRemainingRisk(1))
	if with != want {
		t.Fatalf("immune chronic uncovered rate = %v, want %v from base %v", with, want, without)
	}
}
