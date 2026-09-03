package hud

import (
	"strings"
	"testing"

	"github.com/adsouza/africa2ice/pkg/gameapi"
	"github.com/adsouza/africa2ice/pkg/ui"
	"github.com/ebitenui/ebitenui/widget"
)

// TestGameMenuDescribesTurnBasedBehavior covers the Game Menu overlay's help
// text via the constant buildOverlay renders it from, rather than digging
// through widget internals.
func TestGameMenuDescribesTurnBasedBehavior(t *testing.T) {
	if !strings.Contains(menuHelp, "explicitly end") {
		t.Fatalf("menu help = %q, want it to explain turns advance only when ended", menuHelp)
	}
	if strings.Contains(menuHelp, "pause") {
		t.Fatalf("menu help = %q, should not describe pausing", menuHelp)
	}
}

func TestMenuOverlayButtonsEmitNavigationIntents(t *testing.T) {
	panel := New()
	state := testState(testFrame(1), 1)
	state.Overlay = OverlayState{Scene: ui.SceneMenu}
	panel.Update(state)
	if panel.handles.overlay == nil || len(panel.handles.overlayButtons) != 6 {
		t.Fatalf("menu buttons = %d", len(panel.handles.overlayButtons))
	}
	want := []IntentKind{IntentBack, IntentOpenStorage, IntentOpenStorage, IntentOpenSettings, IntentSetNotesMode, IntentReturnToTitle}
	for index, kind := range want {
		panel.handles.overlayButtons[index].Click()
		intents := panel.Update(state)
		if len(intents) != 1 || intents[0].Kind != kind {
			t.Fatalf("menu button %d = %+v, want %v", index, intents, kind)
		}
		if index == 1 && !intents[0].Save || index == 2 && intents[0].Save {
			t.Fatalf("storage mode for button %d = %+v", index, intents)
		}
	}
}

func TestStorageOverlayRowsCarrySlots(t *testing.T) {
	panel := New()
	state := testState(testFrame(1), 1)
	state.Overlay = OverlayState{Scene: ui.SceneStorage, StorageHeading: "Load / Delete"}
	state.Overlay.StorageRows[0] = StorageRow{Label: "Manual 1", Detail: "Turn 4 · 78,800 BP · sapiens 490", Slot: 1, Occupied: true, Writable: true}
	state.Overlay.StorageRows[1] = StorageRow{Label: "Quick", Detail: "Empty", Slot: 99}
	panel.Update(state)
	if len(panel.handles.overlayButtons) != 2 {
		t.Fatalf("rows with labels = %d, want 2", len(panel.handles.overlayButtons))
	}
	panel.handles.overlayButtons[0].Click()
	if intents := panel.Update(state); len(intents) != 1 || intents[0].Kind != IntentLoadSlot || intents[0].Slot != 1 {
		t.Fatalf("occupied row click = %+v", intents)
	}
	if !panel.handles.overlayButtons[1].GetWidget().Disabled {
		t.Fatal("empty slot is clickable in the load browser")
	}
	if len(panel.handles.deleteButtons) != 2 || !panel.handles.deleteButtons[1].GetWidget().Disabled {
		t.Fatal("delete controls: want one per labelled row, disabled for empty slots")
	}
}

// TestTitleOverlayShowsHeadingAndNewCampaign covers the title overlay's
// content and its New Campaign row, which had no dedicated coverage before.
func TestTitleOverlayShowsHeadingAndNewCampaign(t *testing.T) {
	panel := New()
	state := testState(testFrame(1), 1)
	state.Overlay = OverlayState{Scene: ui.SceneTitle}
	panel.Update(state)
	if panel.handles.overlay == nil || len(panel.handles.overlayButtons) != 3 {
		t.Fatalf("title buttons = %d", len(panel.handles.overlayButtons))
	}
	if label := panel.handles.overlayButtons[1].Text().Label; !strings.Contains(label, "New Campaign") {
		t.Fatalf("title button 1 = %q, want it to contain New Campaign", label)
	}
	panel.handles.overlayButtons[1].Click()
	if intents := panel.Update(state); len(intents) != 1 || intents[0].Kind != IntentNewCampaign {
		t.Fatalf("new campaign click = %+v", intents)
	}
	content := panel.handles.overlay.GetContainer().Children()[0].(*widget.Container)
	heading := content.Children()[0].(*widget.Text)
	if heading.Label != titleHeading {
		t.Fatalf("title heading = %q, want %q", heading.Label, titleHeading)
	}
}

// TestSettingsVolumeRefreshesWithoutRebuilding guards against a rebuild on
// every slider tick: ebitenui's drag state lives on the *Slider instance, so
// a rebuild mid-drag (recreating the slider) would kill the drag after the
// first tick. A volume-only state change must go through refreshVolume
// instead of Panel.rebuild.
func TestSettingsVolumeRefreshesWithoutRebuilding(t *testing.T) {
	panel := New()
	state := testState(testFrame(1), 1)
	state.Overlay = OverlayState{Scene: ui.SceneSettings, MasterVolume: 0.5}
	panel.Update(state)
	builds := panel.builds

	state.Overlay.MasterVolume = 0.6
	panel.Update(state)

	if panel.builds != builds {
		t.Fatalf("builds = %d, want %d (volume-only change must not rebuild)", panel.builds, builds)
	}
	if panel.handles.volumeSlider.Current != 60 {
		t.Fatalf("slider current = %d, want 60", panel.handles.volumeSlider.Current)
	}
	if !strings.Contains(panel.handles.volumeLabel.Label, "60%") {
		t.Fatalf("volume label = %q, want it to contain 60%%", panel.handles.volumeLabel.Label)
	}
}

func TestEndSceneShowsNewCampaignButton(t *testing.T) {
	panel := New()
	frame := testFrame(1)
	frame.CampaignResult = gameapi.Extinction
	state := testState(frame, 1)
	state.Ending = ui.CampaignEndScene(frame)
	panel.Update(state)
	if panel.handles.newCampaign == nil {
		t.Fatal("end scene has no New Campaign button")
	}
	panel.handles.newCampaign.Click()
	if intents := panel.Update(state); len(intents) != 1 || intents[0].Kind != IntentNewCampaign {
		t.Fatalf("new campaign = %+v", intents)
	}
}
