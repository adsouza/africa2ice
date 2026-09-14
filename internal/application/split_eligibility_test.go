package application

import (
	"fmt"
	"testing"

	"github.com/adsouza/africa2ice/pkg/gameapi"
)

// splitDestinationForTest mirrors the choice both the projection and
// pkg/app.splitSelectedBand make: the first ordinary migration candidate, or
// the band's own tile when it has none.
func splitDestinationForTest(band gameapi.Band) gameapi.TileID {
	for _, candidate := range band.MigrationCandidates {
		if !candidate.RequiresPassage {
			return candidate.TileID
		}
	}
	return band.TileID
}

func firstBandOfSpecies(frame *gameapi.Frame, species gameapi.Species) (gameapi.Band, bool) {
	for _, band := range frame.Bands {
		if band.Species == species {
			return band, true
		}
	}
	return gameapi.Band{}, false
}

// TestProjectedSplitEligibilityMatchesAppliedCommand pins the seam between the
// projected Band.SplitBlock and the SplitBand command: the same destination, and
// the same error code out of domainErrorCode. Every case names the code it
// expects, because a suite that only ever reached the empty, split-allowed
// answer would still pass with SplitBlock hardcoded to "" -- which is what the
// two-value easy-mode loop this replaced actually did.
func TestProjectedSplitEligibilityMatchesAppliedCommand(t *testing.T) {
	for _, tc := range []struct {
		name    string
		species gameapi.Species
		prepare func(*testing.T, *GameService, gameapi.Band)
		want    gameapi.ErrorCode
	}{
		{name: "allowed", species: gameapi.HomoSapiens},
		{
			name:    "spatial action already spent",
			species: gameapi.HomoSapiens,
			prepare: func(t *testing.T, service *GameService, band gameapi.Band) {
				t.Helper()
				if _, err := service.Apply(gameapi.QueueMigration{BandID: band.ID, TileID: splitDestinationForTest(band)}); err != nil {
					t.Fatal(err)
				}
			},
			want: gameapi.ErrSpatialActionUsed,
		},
		{name: "computer controlled band", species: gameapi.ArchaicHominin, want: gameapi.ErrComputerControlledBand},
	} {
		for _, easy := range []bool{false, true} {
			t.Run(fmt.Sprintf("%s/easy=%v", tc.name, easy), func(t *testing.T) {
				service, err := NewGameService(2)
				if err != nil {
					t.Fatal(err)
				}
				frame, err := service.Apply(gameapi.SetEasyMode{Enabled: easy})
				if err != nil {
					t.Fatal(err)
				}
				band, ok := firstBandOfSpecies(frame, tc.species)
				if !ok {
					t.Fatalf("no %v band to exercise", tc.species)
				}
				if tc.prepare != nil {
					tc.prepare(t, service, band)
					if frame, err = service.Snapshot(); err != nil {
						t.Fatal(err)
					}
					if band, ok = firstBandOfSpecies(frame, tc.species); !ok {
						t.Fatalf("no %v band after setup", tc.species)
					}
				}
				if band.SplitBlock != tc.want {
					t.Fatalf("projected %q, want %q", band.SplitBlock, tc.want)
				}

				before, _ := service.StateHash()
				projected, err := service.Snapshot()
				if err != nil {
					t.Fatal(err)
				}
				after, _ := service.StateHash()
				if before != after || projected.WorldRevision != frame.WorldRevision {
					t.Fatal("projection changed campaign")
				}

				_, err = service.Apply(gameapi.SplitBand{BandID: band.ID, Destination: splitDestinationForTest(band)})
				got := gameapi.ErrorCode("")
				if err != nil {
					got = err.(*gameapi.GameError).Code
				}
				if got != band.SplitBlock {
					t.Fatalf("projected %q, command %q", band.SplitBlock, got)
				}
			})
		}
	}
}
