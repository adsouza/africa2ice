package application

import (
	"testing"

	"github.com/adsouza/africa2ice/internal/domain"
	"github.com/adsouza/africa2ice/pkg/gameapi"
)

// A queued migration that has resolved is gone, so the save must not keep its
// destination, origin, or passage behind a false flag: those values are dead
// state, yet they reached every save and the canonical campaign-state hash.
func TestResolvedMigrationLeavesNoQueuedValuesInTheSave(t *testing.T) {
	service, err := NewGameService(1)
	if err != nil {
		t.Fatal(err)
	}
	frame, err := service.Snapshot()
	if err != nil {
		t.Fatal(err)
	}
	var band gameapi.Band
	for _, candidate := range frame.Bands {
		if candidate.Species == gameapi.HomoSapiens && len(candidate.MigrationCandidates) > 0 {
			band = candidate
			break
		}
	}
	if band.ID == 0 {
		t.Fatal("no sapiens band with a migration candidate")
	}
	destination := band.MigrationCandidates[0].TileID
	if _, err := service.Apply(gameapi.QueueMigration{BandID: band.ID, TileID: destination}); err != nil {
		t.Fatal(err)
	}
	if _, err := service.EndTurn(); err != nil {
		t.Fatal(err)
	}
	state, err := service.ExportSaveState()
	if err != nil {
		t.Fatal(err)
	}
	for _, saved := range state.Bands {
		if saved.ID != uint64(band.ID) {
			continue
		}
		if saved.HasQueuedMigration || saved.QueuedMigration != 0 || saved.QueuedOrigin != 0 || saved.QueuedPassage != 0 || saved.QueueUsesPassage {
			t.Fatalf("resolved band %d saved queued fields %+v", band.ID, []any{saved.HasQueuedMigration, saved.QueuedMigration, saved.QueuedOrigin, saved.QueuedPassage, saved.QueueUsesPassage})
		}
		if saved.TileID != uint16(destination) {
			t.Fatalf("band %d is on tile %d, want the queued destination %d: the move did not resolve", band.ID, saved.TileID, destination)
		}
		return
	}
	t.Fatalf("band %d missing from the save", band.ID)
}

// A band that completes its last technology has no research target left, so
// the save must not keep the finished technology behind a false flag.
func TestCompletedResearchLeavesNoTargetInTheSave(t *testing.T) {
	service, err := NewGameService(1)
	if err != nil {
		t.Fatal(err)
	}
	state, err := service.world.ExportState()
	if err != nil {
		t.Fatal(err)
	}
	last := domain.TechCount - 1
	index := -1
	for candidate, band := range state.Bands {
		if band.Species == domain.HomoSapiens {
			index = candidate
			break
		}
	}
	band := &state.Bands[index]
	band.Technology.Acquired = (uint16(1)<<domain.TechCount - 1) &^ (1 << last) // every technology but the last
	for technology := range last {
		band.Technology.Progress[technology] = domain.ResearchCost[technology] // acquired means fully paid
	}
	band.Technology.Progress[last] = domain.ResearchCost[last] - 0.001
	if err := band.Technology.Select(last); err != nil {
		t.Fatal(err)
	}
	if service.world, err = domain.RestoreWorld(state); err != nil {
		t.Fatal(err)
	}
	if _, err := service.EndTurn(); err != nil {
		t.Fatal(err)
	}
	saved, err := service.ExportSaveState()
	if err != nil {
		t.Fatal(err)
	}
	for _, item := range saved.Bands {
		if item.ID != uint64(band.ID) {
			continue
		}
		if item.AcquiredTech&(1<<last) == 0 {
			t.Fatalf("band %d did not complete technology %d, so this test proves nothing", band.ID, last)
		}
		if item.HasResearchTarget || item.ResearchTarget != 0 {
			t.Fatalf("completed tree saved research target %d (has %t), want none", item.ResearchTarget, item.HasResearchTarget)
		}
		return
	}
	t.Fatalf("band %d missing from the save", band.ID)
}
