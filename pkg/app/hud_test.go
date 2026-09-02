package app

import (
	"testing"

	"github.com/adsouza/africa2ice/pkg/gameapi"
	"github.com/adsouza/africa2ice/pkg/hud"
	"github.com/adsouza/africa2ice/pkg/ui"
)

func TestHUDStateDerivesRowsAndGateFromTheFrame(t *testing.T) {
	frame := migrationPreviewFrame()
	frame.Bands[0].HasQueuedMigration, frame.Bands[0].QueuedMigration = true, 2
	game := New(&gameStub{frame: frame})
	state := game.hudState()
	if state.SelectedBand != 7 || state.OpenRow != ui.RowResearch {
		t.Fatalf("loaded queued migration: selected %d open row %v, want band 7 with Research open", state.SelectedBand, state.OpenRow)
	}
	if !state.EndTurn.Enabled || state.EndTurn.Soft {
		t.Fatalf("gate with every band moved = %+v", state.EndTurn)
	}
	if state.NotesMode != hud.NotesCompact {
		t.Fatalf("default notes mode = %v", state.NotesMode)
	}
}

func TestEndTurnIntentHonorsTheDirtyDraftGuardLikeSpace(t *testing.T) {
	stub := &gameStub{frame: migrationPreviewFrame()}
	game := New(stub)
	game.editAssignmentDraft(100)
	game.handleIntents([]hud.Intent{{Kind: hud.IntentEndTurn}})
	if stub.endTurns != 0 || game.notice != "Apply or discard workforce changes before ending the turn" {
		t.Fatalf("dirty-draft end turn = %d turns, notice %q", stub.endTurns, game.notice)
	}
	game.discardAssignmentDraft()
	game.handleIntents([]hud.Intent{{Kind: hud.IntentEndTurn}})
	if stub.endTurns != 0 || !game.endTurnArmed {
		t.Fatalf("soft block: turns %d armed %t", stub.endTurns, game.endTurnArmed)
	}
	game.handleIntents([]hud.Intent{{Kind: hud.IntentEndTurn, Force: true}})
	if stub.endTurns != 1 || game.endTurnArmed {
		t.Fatalf("armed click: turns %d armed %t", stub.endTurns, game.endTurnArmed)
	}
}

func TestIntentsReuseHotkeyPaths(t *testing.T) {
	stub := &gameStub{frame: migrationPreviewFrame()}
	game := New(stub)
	game.handleIntents([]hud.Intent{{Kind: hud.IntentMoveTo, Tile: 2}})
	if command, ok := stub.appliedCommand.(gameapi.QueueMigration); !ok || command.TileID != 2 {
		t.Fatalf("MoveTo applied %#v", stub.appliedCommand)
	}
	game.handleIntents([]hud.Intent{{Kind: hud.IntentChooseResearch, Tech: gameapi.Firecraft}})
	if command, ok := stub.appliedCommand.(gameapi.ResearchTech); !ok || command.Tech != gameapi.Firecraft {
		t.Fatalf("ChooseResearch applied %#v", stub.appliedCommand)
	}
	game.handleIntents([]hud.Intent{{Kind: hud.IntentAdjustRole, Role: gameapi.Toolcraft, Delta: 100}})
	if game.assignmentRole != gameapi.Toolcraft || !game.assignmentDraftDirty() {
		t.Fatalf("AdjustRole: role %v dirty %t", game.assignmentRole, game.assignmentDraftDirty())
	}
	game.handleIntents([]hud.Intent{{Kind: hud.IntentDiscardWorkforce}})
	game.handleIntents([]hud.Intent{{Kind: hud.IntentOpenRow, Row: ui.RowWorkforce}, {Kind: hud.IntentToggleDetails}})
	if game.openRow != ui.RowWorkforce || !game.detailsOpen {
		t.Fatal("OpenRow/ToggleDetails not applied")
	}
	game.handleIntents([]hud.Intent{{Kind: hud.IntentSetNotesMode, Notes: hud.NotesExpanded}})
	if game.notesMode != hud.NotesExpanded || !game.settings.FieldNotesExpanded || !game.settings.FieldNotesVisible {
		t.Fatalf("SetNotesMode persisted %+v", game.settings)
	}
}

func TestSelectionChangeResetsDisclosure(t *testing.T) {
	frame := migrationPreviewFrame()
	frame.Bands = append(frame.Bands, gameapi.Band{ID: 9, Species: gameapi.HomoSapiens, Population: 50, TileID: 0, HasResearchTarget: true, SpatialActionUsed: true})
	game := New(&gameStub{frame: frame})
	game.detailsOpen, game.openRow, game.rowChosen = true, ui.RowWorkforce, true
	game.handleIntents([]hud.Intent{{Kind: hud.IntentSelectBand, Band: 9}})
	if game.selectedBand != 9 || game.detailsOpen || game.openRow != ui.RowMove || game.rowChosen {
		t.Fatalf("after select: band %d details %t row %v chosen %t", game.selectedBand, game.detailsOpen, game.openRow, game.rowChosen)
	}
}
