package app

import (
	"testing"

	"github.com/adsouza/africa2ice/pkg/gameapi"
)

func TestRejectedWorkforceApplyPreservesExactDraft(t *testing.T) {
	frame := migrationPreviewFrame()
	frame.Bands[0].AllocationBP = [gameapi.AssignmentCount]uint16{2000, 2000, 2000, 2000, 2000}
	stub := &gameStub{frame: frame}
	game := New(stub)
	stub.applyErrorAt = len(stub.appliedCommands) + 1
	game.editAssignmentDraft(100)
	game.workforce.Role = gameapi.HuntingAndFishing
	game.editAssignmentDraft(-100)
	want := game.workforce.Allocation()
	game.applyAssignmentDraft()
	if !game.workforce.Dirty() || game.workforce.Allocation() != want || game.workforce.Role != gameapi.HuntingAndFishing {
		t.Fatal("command rejection changed the user's draft")
	}
	if game.frame != frame || frame.Bands[0].AllocationBP != ([gameapi.AssignmentCount]uint16{2000, 2000, 2000, 2000, 2000}) {
		t.Fatal("rejected apply replaced the accepted frame")
	}
	game.syncAssignmentDraft(false)
	if game.workforce.Allocation() != want {
		t.Fatal("refresh after rejection lost the draft")
	}
}
