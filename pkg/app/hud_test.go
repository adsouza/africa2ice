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

// TestGContinuesPastAClickedVariant covers the traitFocus cursor fix: before
// this fix, IntentFocusTrait set g.traitFocus to the trait just clicked, so
// focusNextTraitNote's next call (G) re-displayed that same variant instead
// of advancing — the two paths disagreed on whether traitFocus means "the
// current one" or "the next one". Now both leave traitFocus pointing at the
// next variant G will show.
func TestGContinuesPastAClickedVariant(t *testing.T) {
	stub := &gameStub{frame: migrationPreviewFrame()}
	game := New(stub)
	game.handleIntents([]hud.Intent{{Kind: hud.IntentFocusTrait, Trait: gameapi.ColdAdaptation}})
	if !game.fieldNote.HasTrait || game.fieldNote.Trait != gameapi.ColdAdaptation {
		t.Fatalf("clicked trait note = %#v", game.fieldNote)
	}
	game.focusNextTraitNote()
	if !game.fieldNote.HasTrait || game.fieldNote.Trait != gameapi.HighAltitudeAdaptation {
		t.Fatalf("G right after a click = %#v, want it to move on to HighAltitudeAdaptation", game.fieldNote)
	}
}

func TestHidingNotesPreservesTheLastChosenHeight(t *testing.T) {
	game := New(&gameStub{frame: migrationPreviewFrame()})
	game.setNotesMode(hud.NotesExpanded)
	game.setNotesMode(hud.NotesHidden)
	game.toggleFieldNotes()
	if game.notesMode != hud.NotesExpanded || !game.settings.FieldNotesExpanded {
		t.Fatalf("after hide+toggle: notesMode %v expanded %t, want NotesExpanded preserved", game.notesMode, game.settings.FieldNotesExpanded)
	}
}

func TestAcceptedActionsAdvanceOpenRowUnlessThePlayerChoseOne(t *testing.T) {
	newGameWithMoveDone := func() (*Game, *gameStub) {
		before := migrationPreviewFrame()
		before.Bands[0].SpatialActionUsed = true
		stub := &gameStub{frame: before}
		game := New(stub)
		after := migrationPreviewFrame()
		after.Bands[0].SpatialActionUsed = true
		after.Bands[0].HasResearchTarget = true
		stub.frame = after
		return game, stub
	}

	game, _ := newGameWithMoveDone()
	if game.openRow != ui.RowResearch || game.rowChosen {
		t.Fatalf("setup: open row %v chosen %t, want Research open and unchosen", game.openRow, game.rowChosen)
	}
	game.chooseResearchTechnology(gameapi.Firecraft)
	if game.openRow != ui.RowMove {
		t.Fatalf("unchosen: open row after research accepted = %v, want RowMove (both rows done)", game.openRow)
	}

	chosenGame, _ := newGameWithMoveDone()
	chosenGame.openRow, chosenGame.rowChosen = ui.RowWorkforce, true
	chosenGame.chooseResearchTechnology(gameapi.Firecraft)
	if chosenGame.openRow != ui.RowWorkforce {
		t.Fatalf("chosen: open row after research accepted = %v, want unchanged RowWorkforce", chosenGame.openRow)
	}
}

func TestSpaceEndsTheTurnPastTheSoftBlock(t *testing.T) {
	stub := &gameStub{frame: migrationPreviewFrame()}
	game := New(stub)
	game.endTurn(false)
	if stub.endTurns != 0 || !game.endTurnArmed {
		t.Fatalf("unforced end turn with a band still needing a move = turns %d armed %t", stub.endTurns, game.endTurnArmed)
	}
	stub2 := &gameStub{frame: migrationPreviewFrame()}
	game2 := New(stub2)
	game2.endTurn(true)
	if stub2.endTurns != 1 {
		t.Fatalf("forced end turn (Space) = turns %d, want 1 on the first call", stub2.endTurns)
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

func TestGuideFollowsTheTurnAndPersistsDismissal(t *testing.T) {
	stub := &gameStub{frame: migrationPreviewFrame()}
	game := New(stub)
	if game.guide.Step != ui.GuideMove {
		t.Fatalf("fresh guide = %+v", game.guide)
	}
	game.handleIntents([]hud.Intent{{Kind: hud.IntentMoveTo, Tile: 2}})
	frame := migrationPreviewFrame()
	frame.Bands[0].HasQueuedMigration, frame.Bands[0].QueuedMigration = true, 2
	stub.frame = frame
	game.handleIntents([]hud.Intent{{Kind: hud.IntentMoveTo, Tile: 2}}) // second apply returns the queued frame
	if game.guide.Step != ui.GuideResearch {
		t.Fatalf("guide after a queued move = %+v", game.guide)
	}
	game.handleIntents([]hud.Intent{{Kind: hud.IntentGuideDismiss}})
	if game.guide.Visible() || !game.settings.GuideDismissed {
		t.Fatalf("dismissal not persisted: guide %+v settings %+v", game.guide, game.settings)
	}
	game.handleIntents([]hud.Intent{{Kind: hud.IntentShowGuide}})
	if game.guide.Step != ui.GuideMove || game.settings.GuideDismissed {
		t.Fatal("Show first-turn guide did not restore the card")
	}
}

// TestBandActionIntentsStopWhenTheCampaignEnds covers spec §5.3's other half:
// the epilogue's chrome may still be clickable underneath the end scene, so
// planning intents must be refused while overlay navigation keeps working.
func TestBandActionIntentsStopWhenTheCampaignEnds(t *testing.T) {
	frame := migrationPreviewFrame()
	frame.CampaignResult = gameapi.Extinction
	stub := &gameStub{frame: frame}
	game := New(stub)
	for _, kind := range []hud.IntentKind{
		hud.IntentMoveTo, hud.IntentMoveToBest, hud.IntentSplit, hud.IntentInterbreed,
		hud.IntentChooseResearch, hud.IntentAdjustRole, hud.IntentApplyWorkforce, hud.IntentEndTurn,
	} {
		game.handleIntents([]hud.Intent{{Kind: kind, Tile: 2, Delta: 100}})
		if stub.appliedCommand != nil {
			t.Fatalf("intent %v applied %#v after the campaign ended", kind, stub.appliedCommand)
		}
	}
	game.handleIntents([]hud.Intent{{Kind: hud.IntentToggleDetails}})
	if !game.detailsOpen {
		t.Fatal("overlay/disclosure intents stopped working after the campaign ended")
	}
	game.handleIntents([]hud.Intent{{Kind: hud.IntentNewCampaign}})
	if stub.newCampaigns != 1 {
		t.Fatalf("New Campaign requests = %d, want 1 after the campaign ended", stub.newCampaigns)
	}
}
