package application

import (
	"testing"

	"github.com/adsouza/africa2ice/internal/domain"
)

// The HUD tells a player whether blocked terrain is closed for one cold snap or
// finished for the campaign, and it can only do that if the projection carries
// the tile's habitability trajectory. Without this the field is silently zero,
// which reads as "died on turn 0" for every tile on the map.
func TestFrameProjectsLastHabitableTurnForEveryTile(t *testing.T) {
	service, err := NewGameService(0x9e3779b97f4a7c15)
	if err != nil {
		t.Fatalf("NewGameService() error = %v", err)
	}
	frame, err := service.Snapshot()
	if err != nil {
		t.Fatalf("Snapshot() error = %v", err)
	}
	grid := service.world.Grid()
	mismatches, water, land := 0, 0, 0
	for id := range domain.TileCount {
		want := grid.LastHabitableTurn(domain.TileID(id))
		if got := frame.Tiles[id].LastHabitableTurn; got != want {
			if mismatches == 0 {
				t.Errorf("frame tile %d LastHabitableTurn = %d, want %d", id, got, want)
			}
			mismatches++
			continue
		}
		if want < 0 {
			water++
		} else {
			land++
		}
	}
	if mismatches > 0 {
		t.Errorf("%d of %d tiles carry the wrong habitability trajectory", mismatches, domain.TileCount)
	}
	// A projection that hard-coded -1, or one that hard-coded MaxCampaignTurn,
	// would satisfy the comparison above only if the grid agreed, but guard the
	// degenerate case where both sides collapse to a single value.
	if land == 0 || water == 0 {
		t.Errorf("expected both habitable and never-habitable tiles, got %d habitable and %d never", land, water)
	}
}
