package hud

import (
	"testing"

	"github.com/adsouza/africa2ice/pkg/ui"
)

func TestCampControlsReplaceMapAndReturnWithoutRebuildOnIdle(t *testing.T) {
	panel := New()
	state := testState(testFrame(1), 1)
	state.Overlay.Scene = ui.SceneCamp
	panel.Update(state)
	if panel.handles.endTurn != nil || panel.handles.panelMiddle != nil || panel.handles.drawerBody != nil || len(panel.handles.overlayButtons) != 1 {
		t.Fatal("camp retained interactive map controls")
	}
	builds := panel.builds
	idleKey := panel.PresentationKey()
	panel.ui.SetFocusedWidget(panel.handles.overlayButtons[0])
	if panel.PresentationKey() == idleKey {
		t.Fatal("keyboard focus did not invalidate the camp caption cache")
	}
	panel.handles.overlayButtons[0].Click()
	intents := panel.Update(state)
	if len(intents) != 1 || intents[0].Kind != IntentBack || panel.builds != builds {
		t.Fatalf("return button intents = %+v; rebuilt idle camp", intents)
	}
	state.Overlay.Scene = ui.SceneGameplay
	state.DetailsOpen = true
	panel.Sync(state)
	if panel.handles.endTurn == nil || panel.handles.panelMiddle == nil {
		t.Fatal("returning did not restore map controls before Draw")
	}
	panel.handles.campButton.Click()
	if intents := panel.Update(state); len(intents) != 1 || intents[0].Kind != IntentViewCamp {
		t.Fatalf("View camp button intents = %+v", intents)
	}
}
