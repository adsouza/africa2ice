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

// TestBestTileKeepsTheMoveRowOpen covers spec §3.4's requirement that a
// destination the game picked for the player (the Best tile button, or its B
// hotkey via moveToBestTile) leaves the Move row open so the player can see
// what got queued, unlike a destination the player picked themselves.
//
// Both setups pre-mark the band's spatial action as already having a queued
// migration (HasQueuedMigration) without SpatialActionUsed, so
// DiagnoseMigration still allows queuing but DefaultOpenRow already treats
// Move as done and Research as the next undone row — this is what makes
// advanceOpenRow's default recompute actually move away from RowMove,
// exposing the collapse this test guards against.
func TestBestTileKeepsTheMoveRowOpen(t *testing.T) {
	newGame := func() (*Game, *gameStub) {
		frame := migrationPreviewFrame()
		frame.Bands[0].HasQueuedMigration = true
		stub := &gameStub{frame: frame}
		game := New(stub)
		game.openRow = ui.RowMove
		game.rowChosen = false
		return game, stub
	}

	game, stub := newGame()
	game.moveToBestTile()
	if _, ok := stub.appliedCommand.(gameapi.QueueMigration); !ok {
		t.Fatalf("moveToBestTile did not queue a migration: %#v", stub.appliedCommand)
	}
	if game.openRow != ui.RowMove || !game.rowChosen {
		t.Fatalf("moveToBestTile left openRow=%v rowChosen=%t, want RowMove/true", game.openRow, game.rowChosen)
	}

	// Contrast: a row the player did not open via the Best tile shortcut
	// still auto-advances exactly as it did before this change.
	researchGame, researchStub := newGame()
	researchGame.chooseResearchTechnology(gameapi.Firecraft)
	if _, ok := researchStub.appliedCommand.(gameapi.ResearchTech); !ok {
		t.Fatalf("chooseResearchTechnology did not apply a ResearchTech command: %#v", researchStub.appliedCommand)
	}
	if researchGame.openRow == ui.RowMove {
		t.Fatalf("chooseResearchTechnology left the row at Move; expected it to still advance away, want %v", ui.RowResearch)
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
	game.workforce.Role = gameapi.Foraging
	game.handleRowKey(ebiten.KeyArrowDown, false)
	if game.workforce.Role != gameapi.HuntingAndFishing {
		t.Fatalf("Down in Workforce selected %v", game.workforce.Role)
	}
	game.handleRowKey(ebiten.KeyArrowRight, false)
	game.handleRowKey(ebiten.KeyArrowRight, true)
	if got := game.workforce.Allocation()[gameapi.HuntingAndFishing]; got != 600 {
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
	game.workforce.Role = gameapi.Foraging
	game.handleRowKey(ebiten.KeyEqual, true)
	game.handleRowKey(ebiten.KeyMinus, false)
	if got := game.workforce.Allocation()[gameapi.Foraging]; got != 400 {
		t.Fatalf("Shift++ then − in the Workforce row = %d BP, want 400", got)
	}
	game.openRow = ui.RowMove
	game.handleRowKey(ebiten.KeyMinus, false)
	game.handleRowKey(ebiten.KeyEqual, true)
	if got := game.workforce.Allocation()[gameapi.Foraging]; got != 400 {
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

func TestShortcutSheetSwallowsGameplayKeys(t *testing.T) {
	stub := &gameStub{frame: migrationPreviewFrame()}
	game := New(stub)
	initialBand := game.selectedBand

	game.shortcutsOpen = true
	if game.gameplayKeysActive() {
		t.Fatal("gameplayKeysActive() = true while the shortcut sheet is open, want false")
	}

	// One call path: handleGameplayKeys itself must respect the guard (no
	// real key is pressed in this headless test either way, but the guarded
	// early return must not panic or otherwise misbehave on the path that a
	// live Space/Tab press would take).
	game.handleGameplayKeys()
	if stub.endTurns != 0 || game.selectedBand != initialBand {
		t.Fatalf("handleGameplayKeys acted while the shortcut sheet was open: endTurns=%d selectedBand=%d", stub.endTurns, game.selectedBand)
	}

	if !game.escape() || game.shortcutsOpen {
		t.Fatal("escape() did not close the shortcut sheet")
	}
	if !game.gameplayKeysActive() {
		t.Fatal("gameplayKeysActive() = false after the sheet closed, want true")
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

// TestDetailsHotkeyIsRowOwnedAgainstWorkforce covers D2: `D` toggles the band
// details disclosure everywhere except while the Workforce row is open,
// where it keeps its old meaning (discard the workforce draft) instead, the
// same row-owned model arrows/Enter/-+ already use. Real key state cannot be
// injected (see gameplayKeysActive's comment), so this drives the same seam
// TestArrowsBelongToTheOpenRow does: handleRowKey directly for the row-owned
// half, and the Game method the global switch calls for the other half.
func TestDetailsHotkeyIsRowOwnedAgainstWorkforce(t *testing.T) {
	game := New(&gameStub{frame: migrationPreviewFrame()})
	game.openRow = ui.RowMove

	// With Move open, handleRowKey does not own D at all (RowMove's switch
	// has no KeyD case) — it is the global toggleDetails path that owns it,
	// and that path must leave the workforce draft alone.
	draftBefore := game.workforce.Allocation()
	game.handleRowKey(ebiten.KeyD, false)
	if game.workforce.Allocation() != draftBefore {
		t.Fatal("handleRowKey's D case touched the workforce draft with the Move row open")
	}
	detailsBefore := game.detailsOpen
	game.toggleDetails()
	if game.detailsOpen == detailsBefore {
		t.Fatal("toggleDetails did not flip detailsOpen")
	}
	game.toggleDetails()
	if game.detailsOpen != detailsBefore {
		t.Fatal("toggleDetails did not flip detailsOpen back")
	}

	// With Workforce open and a dirty draft, D is row-owned: it discards the
	// draft and must not touch detailsOpen.
	game.openRow, game.rowChosen = ui.RowWorkforce, true
	game.workforce.Role = gameapi.Foraging
	game.handleRowKey(ebiten.KeyArrowRight, false)
	if !game.workforce.Dirty() {
		t.Fatal("setup: Right should have dirtied the workforce draft")
	}
	detailsBefore = game.detailsOpen
	game.handleRowKey(ebiten.KeyD, false)
	if game.workforce.Dirty() {
		t.Fatal("D did not discard the dirty workforce draft with the Workforce row open")
	}
	if game.detailsOpen != detailsBefore {
		t.Fatal("D changed detailsOpen while the Workforce row was open; that row should own D instead")
	}
}
