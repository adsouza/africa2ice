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
	if game.mapVisibleHeight() != 626 {
		t.Fatalf("visible height with hidden drawer = %.0f", game.mapVisibleHeight())
	}
}
