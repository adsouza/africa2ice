package render

import (
	"testing"

	"github.com/adsouza/africa2ice/pkg/gameapi"
)

func interbreedBand() gameapi.Band {
	return gameapi.Band{
		ID: 7, Species: gameapi.HomoSapiens, TileID: 0,
		InterbreedCandidateIDs: []gameapi.BandID{9},
	}
}

// The HUD has to distinguish three states, because each one wants different
// pixels: nothing to say, an offer to make, and a choice already made.
func TestInterbreedStatusDistinguishesAvailableAcceptedAndUnavailable(t *testing.T) {
	band := interbreedBand()
	if got := interbreedStatus(band); got.state != interbreedAvailable {
		t.Fatalf("co-located band state = %v, want available", got.state)
	}

	band.HasInterbreedTarget, band.InterbreedTargetID, band.SpatialActionUsed = true, 9, true
	accepted := interbreedStatus(band)
	if accepted.state != interbreedAccepted {
		t.Fatalf("accepted intent state = %v, want accepted", accepted.state)
	}
	if accepted.targetID != 9 {
		t.Fatalf("accepted target = %d, want 9", accepted.targetID)
	}

	band = interbreedBand()
	band.InterbreedCandidateIDs = nil
	if got := interbreedStatus(band); got.state != interbreedUnavailable {
		t.Fatalf("band with no candidate state = %v, want unavailable", got.state)
	}
}

// A band that has spent its spatial action on something else cannot interbreed,
// so the control must not read as available.
func TestSpentSpatialActionIsNotOfferedAsAvailable(t *testing.T) {
	band := interbreedBand()
	band.SpatialActionUsed = true
	if got := interbreedStatus(band); got.state == interbreedAvailable {
		t.Fatal("a band with a spent spatial action was offered interbreeding")
	}
}

// An archaic band is never the actor, so it never advertises the control.
func TestArchaicBandNeverOffersInterbreeding(t *testing.T) {
	band := interbreedBand()
	band.Species = gameapi.ArchaicHominin
	if got := interbreedStatus(band); got.state != interbreedUnavailable {
		t.Fatalf("archaic band state = %v, want unavailable", got.state)
	}
}
