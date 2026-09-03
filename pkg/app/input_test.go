package app

import (
	"testing"

	"github.com/adsouza/africa2ice/pkg/gameapi"
	"github.com/adsouza/africa2ice/pkg/hud"
	"github.com/adsouza/africa2ice/pkg/ui"
	"github.com/hajimehoshi/ebiten/v2"
)

func TestBestTileHotkeySharesTheButtonPath(t *testing.T) {
	stub := &gameStub{frame: migrationPreviewFrame()}
	game := New(stub)

	game.moveToBestTile()
	command, ok := stub.appliedCommand.(gameapi.QueueMigration)
	if !ok || command.BandID != 7 || command.TileID != 2 {
		t.Fatalf("moveToBestTile applied %#v, want QueueMigration for band 7 to tile 2", stub.appliedCommand)
	}

	frame := migrationPreviewFrame()
	frame.Bands[0].MigrationCandidates = []gameapi.MigrationCandidate{{TileID: 2, RequiresPassage: true}}
	passageOnlyStub := &gameStub{frame: frame}
	passageOnlyGame := New(passageOnlyStub)

	passageOnlyGame.moveToBestTile()
	if passageOnlyStub.appliedCommand != nil || passageOnlyGame.notice != "No reachable land tile to move to this turn." {
		t.Fatalf("passage-only moveToBestTile = command %#v, notice %q", passageOnlyStub.appliedCommand, passageOnlyGame.notice)
	}

	mouseStub := &gameStub{frame: migrationPreviewFrame()}
	mouseGame := New(mouseStub)
	mouseGame.handleIntents([]hud.Intent{{Kind: hud.IntentMoveToBest}})
	mouseCommand, ok := mouseStub.appliedCommand.(gameapi.QueueMigration)
	if !ok || mouseCommand != command {
		t.Fatalf("mouse path applied %#v, want the same command as the keyboard path %#v", mouseStub.appliedCommand, command)
	}
}

func TestArrowsBelongToTheOpenRow(t *testing.T) {
	stub := &gameStub{frame: migrationPreviewFrame()}
	game := New(stub)
	game.openRow = ui.RowMove
	// migrationPreviewFrame's only migration candidate sits diagonally from the
	// band's home tile; MoveMigrationPreview accepts only single-axis steps
	// (see TestKeyboardMigrationPreviewCanTurnIntoReachableCorner), so reaching
	// it takes Up then Left.
	game.handleRowKey(ebiten.KeyArrowUp, false)
	game.handleRowKey(ebiten.KeyArrowLeft, false)
	if !game.hasMigrationPreview {
		t.Fatal("arrows in the Move row did not move the destination cursor")
	}
	game.clearMigrationPreview()

	game.openRow, game.rowChosen = ui.RowWorkforce, true
	game.assignmentRole = gameapi.Foraging
	game.handleRowKey(ebiten.KeyArrowDown, false)
	if game.assignmentRole != gameapi.HuntingAndFishing {
		t.Fatalf("Down in Workforce selected %v", game.assignmentRole)
	}
	game.handleRowKey(ebiten.KeyArrowRight, false)
	game.handleRowKey(ebiten.KeyArrowRight, true)
	if got := game.assignmentDraft[gameapi.HuntingAndFishing]; got != 600 {
		t.Fatalf("Right then Shift+Right = %d BP, want 600", got)
	}
	if game.hasMigrationPreview {
		t.Fatal("Workforce arrows leaked into the migration cursor")
	}
	game.handleRowKey(ebiten.KeyEnter, false)
	if stub.appliedCommand != nil {
		t.Fatal("Enter applied an invalid (>100%) draft")
	}
	game.handleRowKey(ebiten.KeyArrowLeft, true)
	game.handleRowKey(ebiten.KeyArrowLeft, false)
	game.handleRowKey(ebiten.KeyEnter, false)
	if _, ok := stub.appliedCommand.(gameapi.SetAssignment); ok {
		t.Fatal("Enter applied a draft equal to the baseline (nothing to apply)")
	}

	// − and + belong to the Workforce row exactly like Left/Right, so they
	// step the selected role there and do nothing while another row is open.
	game.openRow, game.rowChosen = ui.RowWorkforce, true
	game.assignmentRole = gameapi.Foraging
	game.handleRowKey(ebiten.KeyEqual, true)
	game.handleRowKey(ebiten.KeyMinus, false)
	if got := game.assignmentDraft[gameapi.Foraging]; got != 400 {
		t.Fatalf("Shift++ then − in the Workforce row = %d BP, want 400", got)
	}
	game.openRow = ui.RowMove
	game.handleRowKey(ebiten.KeyMinus, false)
	game.handleRowKey(ebiten.KeyEqual, true)
	if got := game.assignmentDraft[gameapi.Foraging]; got != 400 {
		t.Fatalf("−/+ with the Move row open changed the draft to %d BP, want 400", got)
	}

	game.openRow = ui.RowResearch
	game.researchCursor = gameapi.Firecraft
	game.handleRowKey(ebiten.KeyArrowDown, false)
	game.handleRowKey(ebiten.KeyEnter, false)
	if command, ok := stub.appliedCommand.(gameapi.ResearchTech); !ok || command.Tech != gameapi.HaftedTools {
		t.Fatalf("Research Down+Enter applied %#v", stub.appliedCommand)
	}
}

func TestPageKeysAndShiftArrowsChangeTheOpenRow(t *testing.T) {
	game := New(&gameStub{frame: migrationPreviewFrame()})
	game.openRow = ui.RowMove
	game.changeOpenRow(1)
	game.changeOpenRow(1)
	if game.openRow != ui.RowWorkforce || !game.rowChosen {
		t.Fatalf("two steps down = %v chosen %t", game.openRow, game.rowChosen)
	}
	game.changeOpenRow(1)
	if game.openRow != ui.RowMove {
		t.Fatal("open row did not wrap")
	}
	game.changeOpenRow(-1)
	if game.openRow != ui.RowWorkforce {
		t.Fatal("open row did not wrap backwards")
	}
}

func TestEscapePeelsOneLayerAtATime(t *testing.T) {
	game := New(&gameStub{frame: migrationPreviewFrame()})
	// See TestArrowsBelongToTheOpenRow: reaching the frame's only migration
	// candidate takes two single-axis moves, Up then Left.
	game.handleDirectionalMigration(0, -1)
	game.handleDirectionalMigration(-1, 0)
	game.bandListOpen = true
	if !game.escape() || game.hasMigrationPreview || !game.bandListOpen {
		t.Fatal("first Esc should clear only the cursor")
	}
	if !game.escape() || game.bandListOpen {
		t.Fatal("second Esc should close the band list")
	}
	if game.escape() {
		t.Fatal("third Esc has nothing to peel and should return false so the menu opens")
	}
}
