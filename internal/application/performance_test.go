package application

import (
	"testing"

	"github.com/adsouza/africa2ice/internal/domain"
)

var calibrationResult uint64

func BenchmarkFrameProjectionMaximumWorkload(b *testing.B) {
	world := maximumProjectionWorld(b)
	service := newProjectionService(world, 1)
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

// The calibration workload is frozen: it is the unit every gated ratio is
// expressed in, so it must not move when the game moves. The sizes are
// literals rather than domain constants for that reason — this loop used to
// be sized by TileCount, which would have rescaled every recorded ratio the
// day the map changed size. calibrationBlockSize is chosen so each block
// allocates 576 bytes, putting the mean allocation within a few percent of
// the gated workloads' own (roughly 580 bytes for the maximum turn).
const (
	calibrationBlocks    = 96
	calibrationBlockSize = 72
)

// BenchmarkCalibration is the denominator check_benchmarks divides the gated
// workloads by, so its one job is to move with the machine the way they do.
// It allocates deliberately. Both gated workloads spend their time in the
// allocator, the collector and memory traffic — around 1.9 MB and 3,200
// allocations for a maximum turn — while this benchmark was a cache-resident
// integer loop with zero allocations. Those profiles do not scale together
// across microarchitectures, so the ratio failed to normalise the very thing
// it exists to normalise: on one commit an Intel Xeon runner reported a
// maximum-turn ratio of 1,949 and an AMD EPYC runner 3,840, and the release
// lane failed on hardware alone. The same mismatch is why the interactive Mac
// read ~1,490 where the CI reference read ~2,368 for identical code.
func BenchmarkCalibration(b *testing.B) {
	b.ReportAllocs()
	b.ResetTimer()
	accumulator := uint64(0x9e3779b97f4a7c15)
	for range b.N {
		blocks := make([][]uint64, calibrationBlocks)
		for block := range blocks {
			values := make([]uint64, calibrationBlockSize)
			for index := range values {
				accumulator ^= uint64(index) + 0x9e3779b97f4a7c15 + accumulator<<6 + accumulator>>2
				values[index] = accumulator
			}
			blocks[block] = values
		}
		// Reading every block back before dropping it keeps the collector and
		// the memory system in the measurement rather than letting the
		// allocations retire untouched.
		for _, values := range blocks {
			for _, value := range values {
				accumulator ^= value >> 7
			}
		}
	}
	calibrationResult = accumulator
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
