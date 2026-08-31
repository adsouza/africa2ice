package application

import (
	"testing"

	"github.com/adsouza/africa2ice/internal/domain"
)

var calibrationResult uint64

func BenchmarkFrameProjectionMaximumWorkload(b *testing.B) {
	world := maximumProjectionWorld(b)
	service := &GameService{world: world, worldRevision: 1, terrainRevision: 1}
	b.ReportAllocs()
	b.ResetTimer()
	for range b.N {
		frame, err := service.Snapshot()
		if err != nil {
			b.Fatal(err)
		}
		if len(frame.Tiles) != domain.TileCount || len(frame.Bands) != domain.MaxBands {
			b.Fatalf("incomplete projection: %d tiles, %d bands", len(frame.Tiles), len(frame.Bands))
		}
	}
}

func BenchmarkCalibration(b *testing.B) {
	values := make([]uint64, domain.TileCount)
	b.ReportAllocs()
	b.ResetTimer()
	var result uint64
	for range b.N {
		accumulator := uint64(0x9e3779b97f4a7c15)
		for index := range values {
			accumulator ^= uint64(index) + 0x9e3779b97f4a7c15 + accumulator<<6 + accumulator>>2
			values[index] = accumulator
		}
		result ^= accumulator
	}
	calibrationResult = result
}

func maximumProjectionWorld(b *testing.B) *domain.World {
	b.Helper()
	world, err := domain.NewWorld(0x9e3779b97f4a7c15)
	if err != nil {
		b.Fatal(err)
	}
	state, err := world.ExportState()
	if err != nil {
		b.Fatal(err)
	}
	starting := append([]domain.Band(nil), state.Bands...)
	state.Bands = make([]domain.Band, domain.MaxBands)
	for index := range state.Bands {
		band := starting[index%len(starting)]
		band.ID = domain.BandID(index + 1)
		band.Population = 40
		band.StoredFood = 0
		band.LastFoodReport = domain.FoodTurnReport{}
		band.LastMortality = domain.MortalityReport{}
		band.LastOutcomeReport = domain.OutcomeReport{}
		band.SpatialActionUsed = false
		band.HasQueuedMigration = false
		band.HasInterbreedTarget = false
		state.Bands[index] = band
	}
	state.NextBandID = domain.MaxBands + 1
	world, err = domain.RestoreWorld(state)
	if err != nil {
		b.Fatal(err)
	}
	return world
}
