package application

import (
	"os"
	"testing"

	"github.com/adsouza/africa2ice/internal/domain"
)

func TestPerformanceProfileSaveIsAValidMaximumRenderFixture(t *testing.T) {
	payload, err := os.ReadFile("../../testdata/performance_profile_save.json")
	if err != nil {
		t.Fatal(err)
	}
	state, err := DecodeSaveState(payload)
	if err != nil {
		t.Fatal(err)
	}
	if state.Turn != 300 || len(state.Bands) != domain.MaxBands {
		t.Fatalf("profile fixture = turn %d bands %d", state.Turn, len(state.Bands))
	}
	archaic := 0
	for _, band := range state.Bands {
		if band.Species == uint8(domain.ArchaicHominin) {
			archaic++
		}
	}
	if archaic != domain.MaxArchaicBands {
		t.Fatalf("profile fixture archaic bands = %d, want %d", archaic, domain.MaxArchaicBands)
	}
	for tile := 0; tile < domain.TileCount; tile++ {
		if state.ExploredTiles[tile/64]&(uint64(1)<<uint(tile%64)) == 0 {
			t.Fatalf("profile fixture tile %d is not explored", tile)
		}
	}
	world, err := state.RestoreWorld()
	if err != nil {
		t.Fatal(err)
	}
	if world.Turn() != 300 || len(world.Bands()) != domain.MaxBands {
		t.Fatalf("restored profile fixture = turn %d bands %d", world.Turn(), len(world.Bands()))
	}
	frame, err := ProjectSaveState(state)
	if err != nil {
		t.Fatal(err)
	}
	if frame.Turn != 300 || len(frame.Bands) != domain.MaxBands || len(frame.Tiles) != domain.TileCount {
		t.Fatalf("projected profile fixture = turn %d bands %d tiles %d", frame.Turn, len(frame.Bands), len(frame.Tiles))
	}
}
