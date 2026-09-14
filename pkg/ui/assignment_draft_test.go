package ui

import (
	"testing"

	"github.com/adsouza/africa2ice/pkg/gameapi"
)

func TestAssignmentDraftPreservesEditsAcrossSameBandPublication(t *testing.T) {
	original := [gameapi.AssignmentCount]uint16{2000, 2000, 2000, 2000, 2000}
	band := gameapi.Band{ID: 7, Species: gameapi.HomoSapiens, AllocationBP: original}
	var draft AssignmentDraft
	draft.Sync(&band, false)
	draft.Edit(100)
	want := [gameapi.AssignmentCount]uint16{2100, 2000, 2000, 2000, 2000}
	if draft.Allocation() != want || draft.Valid() || band.AllocationBP != original {
		t.Fatal("edit rebalanced another share or mutated the frame")
	}
	if _, ready := draft.Command(); ready {
		t.Fatal("invalid draft produced a command")
	}
	replacement := band
	replacement.AllocationBP = [gameapi.AssignmentCount]uint16{3000, 1000, 2000, 2000, 2000}
	draft.Sync(&replacement, false)
	if draft.Allocation() != want {
		t.Fatal("new publication overwrote explicit edits")
	}
	draft.Role = gameapi.HuntingAndFishing
	draft.Edit(-100)
	command, ready := draft.Command()
	if !ready || command.BandID != band.ID {
		t.Fatal("balanced draft did not produce a command")
	}
	draft.Edit(-100)
	if command.AllocationBP[gameapi.HuntingAndFishing] != 1900 {
		t.Fatal("later edits changed the captured command")
	}
	if !draft.Discard() || draft.Allocation() != original || draft.Dirty() {
		t.Fatal("discard lost the accepted baseline")
	}
	draft.Sync(&replacement, true)
	if draft.Allocation() != replacement.AllocationBP || draft.Dirty() {
		t.Fatal("accepted replacement did not establish a clean baseline")
	}
}

func TestAssignmentDraftHidesWhenSelectionCannotBeEdited(t *testing.T) {
	var draft AssignmentDraft
	band := gameapi.Band{ID: 1, Species: gameapi.HomoSapiens, AllocationBP: [gameapi.AssignmentCount]uint16{2000, 2000, 2000, 2000, 2000}}
	draft.Sync(&band, false)
	draft.Edit(100)
	archaic := gameapi.Band{ID: 2, Species: gameapi.ArchaicHominin}
	draft.Sync(&archaic, false)
	if draft.Visible() || draft.Dirty() || draft.Valid() {
		t.Fatal("archaic selection retained an active editor")
	}
	if _, ready := draft.Command(); ready {
		t.Fatal("hidden draft produced a command")
	}
	draft.Sync(&band, false)
	if draft.Dirty() || draft.Allocation() != band.AllocationBP {
		t.Fatal("returning to a band did not use its accepted assignment")
	}
	draft.Sync(nil, false)
	if draft.Visible() {
		t.Fatal("missing selection retained an editor")
	}
}
