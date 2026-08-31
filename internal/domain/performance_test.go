package domain

import "testing"

func BenchmarkNewWorldWarm(b *testing.B) {
	if _, err := NewWorld(0); err != nil {
		b.Fatal(err)
	}
	b.ReportAllocs()
	b.ResetTimer()
	for index := range b.N {
		if _, err := NewWorld(uint64(index + 1)); err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkRestoreWorldWarm(b *testing.B) {
	world, err := NewWorld(1)
	if err != nil {
		b.Fatal(err)
	}
	state, err := world.ExportState()
	if err != nil {
		b.Fatal(err)
	}
	b.ReportAllocs()
	b.ResetTimer()
	for range b.N {
		if _, err := RestoreWorld(state); err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkAdvanceTurnMaximumWorkload(b *testing.B) {
	state := maximumWorkloadState(b)
	b.ReportAllocs()
	b.ResetTimer()
	for range b.N {
		b.StopTimer()
		world, err := RestoreWorld(state)
		if err != nil {
			b.Fatal(err)
		}
		b.StartTimer()
		if err := world.AdvanceTurn(); err != nil {
			b.Fatal(err)
		}
	}
}

func maximumWorkloadState(b *testing.B) State {
	b.Helper()
	world, err := NewWorld(0x9e3779b97f4a7c15)
	if err != nil {
		b.Fatal(err)
	}
	state, err := world.ExportState()
	if err != nil {
		b.Fatal(err)
	}
	starting := append([]Band(nil), state.Bands...)
	state.Bands = make([]Band, MaxBands)
	for index := range state.Bands {
		band := starting[index%len(starting)]
		band.ID = BandID(index + 1)
		band.Population = 40
		band.StoredFood = 0
		band.LastFoodReport = FoodTurnReport{}
		band.LastMortality = MortalityReport{}
		band.LastOutcomeReport = OutcomeReport{}
		band.SpatialActionUsed = false
		band.HasQueuedMigration = false
		band.HasInterbreedTarget = false
		state.Bands[index] = band
	}
	state.NextBandID = MaxBands + 1
	return state
}
