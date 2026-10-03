package app

import (
	"testing"

	"github.com/adsouza/africa2ice/pkg/gameapi"
	"github.com/adsouza/africa2ice/pkg/ui"
)

// moved has spent its spatial action, so its default row is Research.
var moved = &gameapi.Band{ID: 1, SpatialActionUsed: true}

func TestDisclosureResetReturnsEveryToggleToItsDefault(t *testing.T) {
	disclosure := panelDisclosure{openRow: ui.RowWorkforce, rowChosen: true, detailsOpen: true, bandListOpen: true, endTurnArmed: true}
	disclosure.reset(moved)
	if disclosure != (panelDisclosure{openRow: ui.RowResearch}) {
		t.Fatalf("reset = %+v, want only the band's default row", disclosure)
	}
}

func TestDisclosureAdvanceYieldsToAChosenRow(t *testing.T) {
	var disclosure panelDisclosure
	disclosure.advance(moved)
	if disclosure.openRow != ui.RowResearch {
		t.Fatalf("unchosen advance opened %v, want the default row", disclosure.openRow)
	}
	disclosure.choose(ui.RowWorkforce)
	disclosure.advance(moved)
	if disclosure.openRow != ui.RowWorkforce || !disclosure.rowChosen {
		t.Fatalf("advance after choose = %+v, want the chosen row kept", disclosure)
	}
}

func TestDisclosureStepWrapsAndCountsAsAChoice(t *testing.T) {
	disclosure := panelDisclosure{openRow: ui.RowWorkforce}
	disclosure.step(1)
	if disclosure.openRow != ui.RowMove || !disclosure.rowChosen {
		t.Fatalf("step past the last row = %+v, want Move, chosen", disclosure)
	}
	disclosure.step(-1)
	if disclosure.openRow != ui.RowWorkforce {
		t.Fatalf("step before the first row = %v, want Workforce", disclosure.openRow)
	}
}

// Ending a turn with unmoved bands takes a second request, unless forced.
func TestDisclosureEndTurnNeedsConfirmationOnlyWhileBandsWait(t *testing.T) {
	var disclosure panelDisclosure
	if !disclosure.confirmEndTurn(0, false) {
		t.Fatal("no waiting bands still required confirmation")
	}
	if disclosure.confirmEndTurn(2, false) || !disclosure.endTurnArmed {
		t.Fatal("first request with waiting bands did not arm")
	}
	if !disclosure.confirmEndTurn(2, false) || disclosure.endTurnArmed {
		t.Fatal("second request did not confirm and disarm")
	}
	if !disclosure.confirmEndTurn(2, true) || disclosure.endTurnArmed {
		t.Fatal("forced request did not confirm immediately")
	}
}
