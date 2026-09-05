package app

import (
	"testing"

	"github.com/adsouza/africa2ice/pkg/hud"
	"github.com/adsouza/africa2ice/pkg/render"
	"github.com/adsouza/africa2ice/pkg/ui"
)

func TestCameraFocusesWhileTheMoveRowIsOpenAndTheActionIsAvailable(t *testing.T) {
	game := New(&gameStub{frame: migrationPreviewFrame()})
	if game.desiredCameraMode() != render.CameraFocus {
		t.Fatal("fresh band with Move open should request Focus")
	}
	game.openRow, game.rowChosen = ui.RowResearch, true
	if game.desiredCameraMode() != render.CameraOverview {
		t.Fatal("Research open should return to Overview")
	}
	game.openRow = ui.RowMove
	game.frame.Bands[0].SpatialActionUsed = true
	if game.desiredCameraMode() != render.CameraOverview {
		t.Fatal("spent action should return to Overview")
	}
	game.frame.Bands[0].SpatialActionUsed = false
	game.toggleCameraOverride()
	if game.desiredCameraMode() != render.CameraOverview {
		t.Fatal("Z should invert the automatic choice")
	}
	game.handleIntents([]hud.Intent{{Kind: hud.IntentSelectBand, Band: 7}})
	game.resetDisclosure()
	if game.desiredCameraMode() != render.CameraFocus {
		t.Fatal("override should reset with disclosure")
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
	if game.mapVisibleHeight() != 626-300 {
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
