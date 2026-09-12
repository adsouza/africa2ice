package app

import (
	"testing"

	"github.com/adsouza/africa2ice/pkg/gameapi"
	"github.com/adsouza/africa2ice/pkg/hud"
	"github.com/adsouza/africa2ice/pkg/render"
	"github.com/adsouza/africa2ice/pkg/ui"
)

// cameraFrame is the camera's own fixture: band 7 opens the campaign with
// its spatial action available, band 8 has already acted, and band 9 is a
// second still-movable band to re-arm Focus with.
func cameraFrame() *gameapi.Frame {
	frame := migrationPreviewFrame()
	frame.Bands = append(frame.Bands,
		gameapi.Band{ID: 8, Species: gameapi.HomoSapiens, TileID: 1, Population: 120, SpatialActionUsed: true},
		gameapi.Band{ID: 9, Species: gameapi.HomoSapiens, TileID: 2, Population: 120},
	)
	return frame
}

// TestSpendingTheMoveLeavesTheCameraFocused pins the sticky half of the
// camera contract. The mode used to be re-derived every tick from
// "Move row open ∧ sapiens ∧ !MoveDone", so committing a move falsified two
// of those conjuncts at once — MoveDone flips, and advanceOpenRow walks the
// open row off Move — and the map zoomed out from under the player mid-plan.
// The mode is stored state now, and nothing about the band clears it.
func TestSpendingTheMoveLeavesTheCameraFocused(t *testing.T) {
	game := New(&gameStub{frame: cameraFrame()})
	if game.desiredCameraMode() != render.CameraFocus {
		t.Fatal("a fresh band with its move open should start in Focus")
	}
	game.frame.Bands[0].SpatialActionUsed = true
	game.advanceOpenRow()
	if game.desiredCameraMode() != render.CameraFocus {
		t.Fatal("spending the move zoomed out; Focus must hold")
	}
	game.openRow, game.rowChosen = ui.RowResearch, true
	if game.desiredCameraMode() != render.CameraFocus {
		t.Fatal("opening another row zoomed out; Focus must hold")
	}
}

// TestSwitchingBandsRecentersWithoutLeavingFocus is the other half of the
// ask. Band 8 has already acted, which under the derived mode was exactly
// the selection that snapped back to Overview; the camera should do nothing
// here but follow its new centre.
func TestSwitchingBandsRecentersWithoutLeavingFocus(t *testing.T) {
	game := New(&gameStub{frame: cameraFrame()})
	game.stepCamera()
	game.handleIntents([]hud.Intent{{Kind: hud.IntentSelectBand, Band: 8}})
	game.stepCamera()
	if game.camera.Mode != render.CameraFocus {
		t.Fatal("selecting an already-moved band zoomed out")
	}
	if game.camera.CenterTile != game.frame.Bands[1].TileID {
		t.Fatalf("camera centre = %d, want band 8's tile %d", game.camera.CenterTile, game.frame.Bands[1].TileID)
	}
}

// TestCameraToggleHoldsUntilANewMovableSelectionRearmsIt covers Z's new
// meaning. It sets the mode outright rather than inverting a derived answer,
// so it survives everything that happens to the band it was pressed on. A
// selection change onto a band that can still act re-arms Focus, and that is
// the only automatic zoom-in the camera keeps.
func TestCameraToggleHoldsUntilANewMovableSelectionRearmsIt(t *testing.T) {
	game := New(&gameStub{frame: cameraFrame()})
	game.toggleCameraFocus()
	if game.desiredCameraMode() != render.CameraOverview {
		t.Fatal("Z should leave Focus")
	}
	game.openRow, game.rowChosen = ui.RowMove, true
	if game.desiredCameraMode() != render.CameraOverview {
		t.Fatal("reopening Move on the same selection re-armed Focus")
	}
	game.frame.Bands[0].SpatialActionUsed = true
	if game.desiredCameraMode() != render.CameraOverview {
		t.Fatal("spending the move after Z zoomed in; the toggle sets the mode, it does not invert one")
	}
	game.handleIntents([]hud.Intent{{Kind: hud.IntentSelectBand, Band: 8}})
	if game.desiredCameraMode() != render.CameraOverview {
		t.Fatal("selecting a band that has already acted re-armed Focus")
	}
	game.handleIntents([]hud.Intent{{Kind: hud.IntentSelectBand, Band: 9}})
	if game.desiredCameraMode() != render.CameraFocus {
		t.Fatal("selecting a band that can still act should re-arm Focus")
	}
}

// TestCameraNeedsASelectionToFocus guards the one clamp on the stored flag:
// stepCamera refreshes CenterTile only while a band is selected, so Focus
// with nothing selected would zoom a stale tile. The clamp reads the flag
// without clearing it, so reselecting restores the player's zoom.
func TestCameraNeedsASelectionToFocus(t *testing.T) {
	game := New(&gameStub{frame: cameraFrame()})
	game.selectedBand = 0
	if game.desiredCameraMode() != render.CameraOverview {
		t.Fatal("Focus with nothing selected should fall back to Overview")
	}
	game.selectedBand = 7
	if game.desiredCameraMode() != render.CameraFocus {
		t.Fatal("reselecting should restore the stored Focus")
	}
}

// TestCameraToggleStaysAvailableAfterTheMoveIsSpent keeps the map-corner
// button reachable. It used to be gated on the same predicate the mode was
// derived from, so a Focus that outlives the move would otherwise strand the
// player zoomed in with Z as the only way back out.
func TestCameraToggleStaysAvailableAfterTheMoveIsSpent(t *testing.T) {
	game := New(&gameStub{frame: cameraFrame()})
	game.frame.Bands[0].SpatialActionUsed = true
	if !game.hudState().Camera.ToggleAvailable {
		t.Fatal("the map-corner toggle vanished with the move it must outlive")
	}
}

func TestCameraStepsEachUpdateAndTracksTheDrawer(t *testing.T) {
	game := New(&gameStub{frame: migrationPreviewFrame()})
	for tick := 0; tick < render.CameraTransitionTicks; tick++ {
		game.stepCamera()
	}
	if !game.camera.Settled() || game.camera.Mode != render.CameraFocus || game.camera.CenterTile != game.frame.Bands[0].TileID {
		t.Fatalf("camera after transition = %+v", game.camera)
	}
	game.notesMode = hud.NotesExpanded
	if game.mapVisibleHeight() != 626-340 {
		t.Fatalf("visible height with expanded drawer = %.0f", game.mapVisibleHeight())
	}
	game.notesMode = hud.NotesHidden
	if game.mapVisibleHeight() != 626-hud.DrawerHiddenHeight {
		t.Fatalf("visible height with hidden drawer = %.0f", game.mapVisibleHeight())
	}
}

// TestClickAndHoverPickTheSameTileMidTransition guards against hover and
// click resolving different tiles mid-transition: both must read the live
// camera through the same pickTile, not a copy refreshed only in Draw.
// migrationPreviewFrame's tiles carry placeholder X/Y (negative, off any
// real grid row) since no test before this one ever fed them through the
// camera's pixel geometry; row/column 2,0 is this test's own copy of the
// frame, set to a real grid position so TilePoint/TileAt round-trip to the
// migration candidate tile (2) instead of landing off the map entirely.
func TestClickAndHoverPickTheSameTileMidTransition(t *testing.T) {
	game := New(&gameStub{frame: migrationPreviewFrame()})
	game.frame.Tiles[2].X, game.frame.Tiles[2].Y = 2, 0
	game.camera = render.Camera{Mode: render.CameraFocus, CenterTile: game.frame.Bands[0].TileID, Progress: 0.4}
	target := game.frame.Tiles[2]
	px, py := render.CameraGeometry(game.camera, game.frame, game.mapVisibleHeight()).TilePoint(target)
	x, y := int(px), int(py)

	hoverID, hoverOK := game.exploredHoverTile(x, y, true)
	clickID, clickOK := game.pickTile(x, y)
	if !hoverOK || !clickOK || hoverID != 2 || clickID != 2 {
		t.Fatalf("hover = (%d,%t), click = (%d,%t), want (2,true) both", hoverID, hoverOK, clickID, clickOK)
	}
}
