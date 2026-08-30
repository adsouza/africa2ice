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

// The control hints must not advertise a key that does nothing.
func TestControlHintsMentionInterbreedOnlyWhenAvailable(t *testing.T) {
	if hint := spatialControlHint(interbreedStatus(interbreedBand())); !contains(hint, "I:") {
		t.Fatalf("available interbreeding is missing from the hint %q", hint)
	}
	band := interbreedBand()
	band.InterbreedCandidateIDs = nil
	if hint := spatialControlHint(interbreedStatus(band)); contains(hint, "I:") {
		t.Fatalf("hint %q advertises interbreeding with no candidate", hint)
	}
}

func TestInspectorAndFieldNotesHaveDedicatedVerticalSpace(t *testing.T) {
	if mapLegendOriginY+mapLegendHeight > mapOriginY {
		t.Fatal("map legend overlaps the map")
	}
	if interbreedPanelLineY < tileInspectorOriginY || interbreedPanelLineY+8 > tileInspectorOriginY+tileInspectorHeight {
		t.Fatal("interbreeding line falls outside the tile inspector")
	}
	if tileInspectorOriginY+tileInspectorHeight >= fieldNotesPanelOriginY {
		t.Fatal("Field Notes panel covers the tile inspector")
	}
	if fieldNotesPanelOriginY+fieldNotesPanelHeight > 610 {
		t.Fatal("Field Notes panel overlaps the controls")
	}
}

func contains(haystack, needle string) bool {
	return len(haystack) >= len(needle) && (haystack == needle || indexOf(haystack, needle) >= 0)
}

func indexOf(haystack, needle string) int {
	for i := 0; i+len(needle) <= len(haystack); i++ {
		if haystack[i:i+len(needle)] == needle {
			return i
		}
	}
	return -1
}
