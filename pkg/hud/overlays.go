package hud

import (
	"fmt"
	"image"

	"github.com/adsouza/africa2ice/pkg/ui"
	"github.com/ebitenui/ebitenui/widget"
)

const (
	overlayX      = 340.0
	overlayY      = 150.0
	overlayWidth  = 600.0
	overlayHeight = 420.0
)

// menuHelp is the Game Menu's help line; kept as a constant so tests can
// assert on it without inspecting widget internals.
const menuHelp = "Turns advance only when you explicitly end them."

// titleHeading is the title overlay's heading; kept as a constant so tests
// can assert on it without inspecting widget internals, and so the label
// drawn on screen cannot drift from what a test checks.
const titleHeading = "Africa 2 Ice: Paleolithic Dispersal"

// buildOverlay shows the modal for the current non-gameplay scene, or the
// shortcut sheet, as an ebitenui window that blocks input beneath it.
func (p *Panel) buildOverlay(state State) {
	var content *widget.Container
	switch {
	case state.Overlay.Scene == ui.SceneTitle:
		content = p.menuList(titleHeading, "Guide Homo sapiens from East Africa, 80,000–20,000 BP.", []menuEntry{
			{"Continue · Enter", Intent{Kind: IntentContinue}},
			{"New Campaign · N", Intent{Kind: IntentNewCampaign}},
			{"Load a checkpoint · L", Intent{Kind: IntentOpenStorage}},
		})
	case state.Overlay.Scene == ui.SceneMenu:
		content = p.menuList("Game Menu", menuHelp, []menuEntry{
			{"Back to game · Esc", Intent{Kind: IntentBack}},
			{"Save slots · S", Intent{Kind: IntentOpenStorage, Save: true}},
			{"Load or delete slots · L", Intent{Kind: IntentOpenStorage}},
			{"Settings · O", Intent{Kind: IntentOpenSettings}},
			{"Return to title · T", Intent{Kind: IntentReturnToTitle}},
		})
	case state.Overlay.Scene == ui.SceneStorage:
		content = p.storageList(state)
	case state.Overlay.Scene == ui.SceneSettings:
		content = p.settingsPanel(state)
	case state.ShortcutsOpen:
		content = p.shortcutSheet()
	default:
		return
	}
	rect := image.Rectangle(p.rect(overlayX, overlayY, overlayWidth, overlayHeight))
	window := widget.NewWindow(widget.WindowOpts.Contents(content), widget.WindowOpts.Modal(), widget.WindowOpts.CloseMode(widget.NONE), widget.WindowOpts.Location(rect))
	p.ui.AddWindow(window)
	p.handles.overlay = window
}

type menuEntry struct {
	label  string
	intent Intent
}

func (p *Panel) overlayFrame(heading, help string) *widget.Container {
	t := p.theme
	frame := t.column(10, t.insets(24, 28, 28, 20), t.bordered(colorPanel, colorGoldDeep, t.px(2)), widget.WidgetOpts.MinSize(t.px(overlayWidth), t.px(overlayHeight)))
	frame.AddChild(t.label(heading, 25, colorTitle))
	if help != "" {
		frame.AddChild(t.label(help, 11, colorDim))
	}
	return frame
}

func (p *Panel) menuList(heading, help string, entries []menuEntry) *widget.Container {
	t := p.theme
	frame := p.overlayFrame(heading, help)
	for _, entry := range entries {
		intent := entry.intent
		button := t.button(entry.label, 14, colorPanelEdge, colorText, func() { p.emit(intent) })
		button.GetWidget().LayoutData = widget.RowLayoutData{Stretch: true}
		p.handles.overlayButtons = append(p.handles.overlayButtons, button)
		frame.AddChild(button)
	}
	return frame
}

func (p *Panel) storageList(state State) *widget.Container {
	t := p.theme
	help := "Click a slot · Delete removes it · Esc back"
	if state.Overlay.StorageBusy != "" {
		help = state.Overlay.StorageBusy
	}
	frame := p.overlayFrame(state.Overlay.StorageHeading, help)
	saving := state.Overlay.StorageSaving
	for _, row := range state.Overlay.StorageRows {
		if row.Label == "" {
			continue
		}
		slot := row.Slot
		line := t.rowOf(8, stretch())
		var intent Intent
		switch {
		case saving:
			intent = Intent{Kind: IntentSaveSlot, Slot: slot}
		default:
			intent = Intent{Kind: IntentLoadSlot, Slot: slot}
		}
		button := t.button(fmt.Sprintf("%-10s %s", row.Label, row.Detail), 12, colorPanelEdge, colorText, func() { p.emit(intent) })
		button.GetWidget().Disabled = state.Overlay.StorageBusy != "" || (saving && !row.Writable) || (!saving && !row.Occupied)
		button.GetWidget().LayoutData = widget.RowLayoutData{Stretch: true}
		p.handles.overlayButtons = append(p.handles.overlayButtons, button)
		remove := t.button("Delete", 10, colorRed, colorRed, func() { p.emit(Intent{Kind: IntentDeleteSlot, Slot: slot}) })
		remove.GetWidget().Disabled = state.Overlay.StorageBusy != "" || !row.Occupied
		p.handles.deleteButtons = append(p.handles.deleteButtons, remove)
		line.AddChild(button, remove)
		frame.AddChild(line)
	}
	frame.AddChild(t.button("Back · Esc", 12, colorGoldDeep, colorGoldDeep, func() { p.emit(Intent{Kind: IntentBack}) }))
	return frame
}

func (p *Panel) settingsPanel(state State) *widget.Container {
	t := p.theme
	help := "Drag the slider · M mutes · O or Esc back"
	if state.Overlay.SettingsDisabled {
		help = "Loading preferences… controls disabled · O/Esc back"
	}
	frame := p.overlayFrame("Settings", help)
	volume := t.rowOf(10, stretch())
	label := t.label(fmt.Sprintf("Master volume %3.0f%%", state.Overlay.MasterVolume*100), 13, colorText)
	p.handles.volumeLabel = label
	volume.AddChild(label)
	slider := widget.NewSlider(
		widget.SliderOpts.Orientation(widget.DirectionHorizontal),
		widget.SliderOpts.MinMax(0, 100),
		widget.SliderOpts.InitialCurrent(int(state.Overlay.MasterVolume*100+0.5)),
		widget.SliderOpts.Images(&widget.SliderTrackImage{Idle: t.solid(colorPanelEdge), Hover: t.solid(colorPanelEdge)}, t.buttonImages(colorGoldDeep)),
		widget.SliderOpts.FixedHandleSize(t.px(12)),
		widget.SliderOpts.ChangedHandler(func(args *widget.SliderChangedEventArgs) {
			p.emit(Intent{Kind: IntentSetVolume, Volume: float64(args.Current) / 100})
		}),
		widget.SliderOpts.WidgetOpts(widget.WidgetOpts.MinSize(t.px(220), t.px(14))),
	)
	slider.GetWidget().Disabled = state.Overlay.SettingsDisabled
	p.handles.volumeSlider = slider
	volume.AddChild(slider)
	frame.AddChild(volume)
	mute := "Muted: off · M"
	if state.Overlay.Muted {
		mute = "Muted: on · M"
	}
	for _, entry := range []menuEntry{
		{mute, Intent{Kind: IntentToggleMute}},
		{"Show first-turn guide", Intent{Kind: IntentShowGuide}},
		{"Back · Esc", Intent{Kind: IntentBack}},
	} {
		intent := entry.intent
		button := t.button(entry.label, 13, colorPanelEdge, colorText, func() { p.emit(intent) })
		button.GetWidget().Disabled = state.Overlay.SettingsDisabled && intent.Kind != IntentBack
		p.handles.overlayButtons = append(p.handles.overlayButtons, button)
		frame.AddChild(button)
	}
	return frame
}

// refreshVolume updates the settings overlay's slider position and label
// without rebuilding the tree, so ebitenui's drag state (which lives on the
// *Slider instance) survives a mid-drag ChangedHandler round trip through
// the application and back into State.Overlay.MasterVolume.
func (p *Panel) refreshVolume(state State) {
	p.refreshes++
	if p.handles.volumeSlider != nil {
		if current := int(state.Overlay.MasterVolume*100 + 0.5); p.handles.volumeSlider.Current != current {
			p.handles.volumeSlider.Current = current
		}
	}
	if p.handles.volumeLabel != nil {
		p.handles.volumeLabel.Label = fmt.Sprintf("Master volume %3.0f%%", state.Overlay.MasterVolume*100)
	}
}

func (p *Panel) shortcutSheet() *widget.Container {
	t := p.theme
	frame := p.overlayFrame("Keyboard shortcuts", "? or Esc closes")
	lines := []string{
		"Space  end turn        Tab / Shift+Tab  next / previous band",
		"PgUp / PgDn or Shift+Up/Down  change the open row",
		"Move row:  arrows steer the cursor · Enter queues · Esc clears",
		"Research row:  Up/Down highlight · Enter chooses · 1–9 direct",
		"Workforce row:  Up/Down pick a role · Left/Right or −/+ step 1% · Shift 5% · Enter/A apply · D discard",
		"N split · I interbreed · J cycle partner · G cycle trait note · B best tile",
		"F notes · Shift+F expand notes · wheel scrolls notes · Z camera",
		"Ctrl/Cmd+S quick-save · F1–F3 save · Shift+F1–F3 load · M mute · Esc menu",
	}
	for _, line := range lines {
		frame.AddChild(t.label(line, 11.5, colorText))
	}
	frame.AddChild(t.button("Close · Esc", 12, colorGoldDeep, colorGoldDeep, func() { p.emit(Intent{Kind: IntentToggleShortcuts}) }))
	return frame
}
