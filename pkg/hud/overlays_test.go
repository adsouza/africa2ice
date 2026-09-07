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
	if panel.handles.overlay == nil || len(panel.handles.overlayButtons) != 5 {
		t.Fatalf("menu buttons = %d, want 5 now that Field Notes (always reachable via F/Shift+F and its own edge controls) is gone from the menu", len(panel.handles.overlayButtons))
	}
	want := []IntentKind{IntentBack, IntentOpenStorage, IntentOpenStorage, IntentOpenSettings, IntentReturnToTitle}
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

	// Save mode is carried as data, not inferred from the heading text.
	saving := New()
	saveState := testState(testFrame(1), 1)
	saveState.Overlay = OverlayState{Scene: ui.SceneStorage, StorageHeading: "Save / Delete", StorageSaving: true}
	saveState.Overlay.StorageRows[0] = StorageRow{Label: "Manual 1", Detail: "Turn 4 · 78,800 BP · sapiens 490", Slot: 1, Occupied: true, Writable: true}
	saveState.Overlay.StorageRows[1] = StorageRow{Label: "Quick", Detail: "Turn 6 · 78,000 BP · sapiens 500", Slot: 99, Occupied: true}
	saving.Update(saveState)
	saving.handles.overlayButtons[0].Click()
	if intents := saving.Update(saveState); len(intents) != 1 || intents[0].Kind != IntentSaveSlot || intents[0].Slot != 1 {
		t.Fatalf("writable row click in save mode = %+v", intents)
	}
	if !saving.handles.overlayButtons[1].GetWidget().Disabled {
		t.Fatal("a quick slot is writable in the save browser")
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
	// The button must read as the dialog's primary action: a filled gold
	// idle background over dark text, the same treatment End turn gets
	// (end_turn.go), not black text over the theme's dark button fill.
	// Button colors have no public getter, so this checks the theme's
	// memoized solid-fill cache the way
	// TestHiddenBarControlMatchesTheBreakthroughAccent checks the border
	// cache: colorGoldDeep only ever reaches t.solid (as opposed to
	// t.bordered) from buildEndTurn's idle fill and buildEndScene's, and
	// End turn is never built while the campaign is over.
	sawGoldDeepFill := false
	for key := range panel.theme.solids {
		if key == colorGoldDeep {
			sawGoldDeepFill = true
		}
	}
	if !sawGoldDeepFill {
		t.Fatal("New Campaign button's idle image is not the gold fill (colorGoldDeep); it should match End turn's primary treatment")
	}
	panel.handles.newCampaign.Click()
	if intents := panel.Update(state); len(intents) != 1 || intents[0].Kind != IntentNewCampaign {
		t.Fatalf("new campaign = %+v", intents)
	}
}

func TestSettingsEasyModeCheckbox(t *testing.T) {
	panel := New()
	state := testState(testFrame(1), 1)
	state.Overlay = OverlayState{Scene: ui.SceneSettings, EasyMode: true}
	panel.Update(state)
	checkbox := panel.handles.easyModeCheckbox
	if checkbox == nil || checkbox.State() != widget.WidgetChecked {
		t.Fatal("easy mode is not checked")
	}
	checkbox.Click()
	intents := panel.Update(state)
	if len(intents) != 1 || intents[0].Kind != IntentToggleEasyMode {
		t.Fatalf("checkbox intents: %+v", intents)
	}
	state.Overlay.EasyMode = false
	state.Overlay.SettingsDisabled = true
	panel.Update(state)
	checkbox = panel.handles.easyModeCheckbox
	if checkbox.State() != widget.WidgetUnchecked || !checkbox.GetWidget().Disabled {
		t.Fatal("checkbox state/loading guard incorrect")
	}
}
