package hud

import (
	"image"

	"github.com/adsouza/africa2ice/pkg/gameapi"
	"github.com/adsouza/africa2ice/pkg/ui"
	"github.com/ebitenui/ebitenui"
	"github.com/ebitenui/ebitenui/input"
	"github.com/ebitenui/ebitenui/widget"
	"github.com/hajimehoshi/ebiten/v2"
)

// Panel DIP geometry (spec §4). Map area: x 20–884, y 74–700.
const (
	panelX          = 908.0
	panelY          = 68.0
	panelWidth      = 352.0
	panelHeight     = 632.0
	panelPadding    = 18.0
	mapLeft         = 20.0
	mapRight        = 884.0
	mapBottom       = 700.0
	drawerCompactH  = 102.0
	drawerExpandedH = 300.0
	drawerTabW      = 150.0
	drawerTabH      = 18.0
)

// DrawerCompactHeight and DrawerExpandedHeight are exported so pkg/app can
// derive the map's camera-visible height (spec §6) from the same numbers the
// drawer itself draws at, rather than duplicating them.
const (
	DrawerCompactHeight  = drawerCompactH
	DrawerExpandedHeight = drawerExpandedH
)

// Panel owns the ebitenui tree for every piece of chrome. It rebuilds the tree
// whenever the State value changes and otherwise leaves widgets untouched so
// presses and hovers survive across ticks.
type Panel struct {
	ui      *ebitenui.UI
	theme   *theme
	root    *widget.Container
	last    State
	built   bool
	builds  int
	intents []Intent
	handles handles
}

// handles keeps pointers to widgets tests and refreshes need to reach. Later
// tasks add fields here as they wire up the widgets those fields point to
// (guideNext, guideX and friends): a field with no writer or reader anywhere
// in the package is exactly the dead code the unused linter exists to catch,
// so it is added alongside its first use, not ahead of it.
type handles struct {
	chips      map[uint32]*widget.Button // keyed by gameapi.BandID
	more       *widget.Button
	bandList   *widget.Window
	details    *widget.Button
	traits     map[gameapi.HeritableTrait]*widget.Button
	overlay    *widget.Window
	rowHeader  [ui.ChecklistRowCount]*widget.Button
	moveHere   *widget.Button
	best       *widget.Button
	split      *widget.Button
	interbreed *widget.Button
	research   [gameapi.TechCount]*widget.Button
	workforce  workforceHandles
	endTurn    *widget.Button
	drawerTab  *widget.Button
	drawerMore *widget.Button
	events     []*widget.Button
	camera     *widget.Button

	overlayButtons []*widget.Button
	deleteButtons  []*widget.Button
	newCampaign    *widget.Button
	volumeSlider   *widget.Slider
	volumeLabel    *widget.Text
}

func New() *Panel {
	panel := &Panel{theme: newTheme()}
	panel.root = widget.NewContainer(widget.ContainerOpts.Layout(fixedLayout{}))
	panel.ui = &ebitenui.UI{Container: panel.root}
	return panel
}

// Update rebuilds on change, runs ebitenui, and returns the intents clicks
// produced this tick. Call it before map input so Hovered is current.
func (p *Panel) Update(state State) []Intent {
	structural, lastStructural := state, p.last
	structural.Workforce, lastStructural.Workforce = WorkforceDraft{}, WorkforceDraft{}
	structural.Overlay.MasterVolume, lastStructural.Overlay.MasterVolume = 0, 0
	switch {
	case !p.built || structural != lastStructural:
		p.rebuild(state)
		p.built = true
	case state.Workforce != p.last.Workforce:
		p.last = state
		p.refreshWorkforce(state)
	case state.Overlay.MasterVolume != p.last.Overlay.MasterVolume:
		p.last = state
		p.refreshVolume(state)
	}
	p.ui.Update()
	intents := p.intents
	p.intents = nil
	return intents
}

func (p *Panel) Draw(screen *ebiten.Image) { p.ui.Draw(screen) }

// Hovered reports whether the pointer is over any chrome widget, so map input
// can yield. Valid after Update.
func (p *Panel) Hovered() bool { return input.UIHovered }

func (p *Panel) emit(intent Intent) { p.intents = append(p.intents, intent) }

// rect converts a DIP rectangle to render pixels including the letterbox offset.
func (p *Panel) rect(x, y, width, height float64) fixedRect {
	transform := p.last.Transform
	if transform.Scale <= 0 {
		transform.Scale = 1
	}
	left := int(transform.OffsetX + x*transform.Scale + 0.5)
	top := int(transform.OffsetY + y*transform.Scale + 0.5)
	return fixedRect(image.Rect(left, top, left+p.theme.px(width), top+p.theme.px(height)))
}

func (p *Panel) rebuild(state State) {
	p.builds++
	p.last = state
	scale := state.Transform.Scale
	if scale <= 0 {
		scale = 1
	}
	p.theme.setScale(scale)
	if p.handles.overlay != nil {
		p.handles.overlay.Close()
		p.handles.overlay = nil
	}
	if p.handles.bandList != nil {
		p.handles.bandList.Close()
		p.handles.bandList = nil
	}
	p.root.RemoveChildren()
	p.handles = handles{chips: map[uint32]*widget.Button{}, traits: map[gameapi.HeritableTrait]*widget.Button{}}
	if state.Frame == nil {
		return
	}
	p.root.AddChild(p.buildPanel(state))
	p.root.AddChild(p.buildDrawer(state))
	if state.Camera.FocusAvailable {
		p.root.AddChild(p.buildCameraButton(state))
	}
	if ending := p.buildEndScene(state); ending != nil {
		p.root.AddChild(ending)
	}
	p.buildOverlay(state)
}

// buildCameraButton is the map-corner Overview/Focus control (spec §6).
func (p *Panel) buildCameraButton(state State) widget.PreferredSizeLocateableWidget {
	t := p.theme
	label := "Focus · Z"
	if state.Camera.Focused {
		label = "Overview · Z"
	}
	holder := widget.NewContainer(widget.ContainerOpts.Layout(widget.NewAnchorLayout()),
		widget.ContainerOpts.WidgetOpts(widget.WidgetOpts.LayoutData(p.rect(mapRight-118, 80, 110, 22))))
	button := t.button(label, 9.5, colorGoldDeep, colorGoldDeep, func() { p.emit(Intent{Kind: IntentCameraToggle}) })
	button.GetWidget().LayoutData = widget.AnchorLayoutData{StretchHorizontal: true, StretchVertical: true}
	p.handles.camera = button
	holder.AddChild(button)
	return holder
}

// buildPanel is the full chrome column: header, chips, band line, details,
// guide card (Task 16 fills), checklist (Task 9), end turn (Task 9), footer.
func (p *Panel) buildPanel(state State) widget.PreferredSizeLocateableWidget {
	t := p.theme
	column := t.column(8, t.insets(14, panelPadding, panelPadding, 12), solid(colorPanel),
		widget.WidgetOpts.LayoutData(p.rect(panelX, panelY, panelWidth, panelHeight)))
	band := state.selectedBand()
	column.AddChild(p.buildHeader(state))
	column.AddChild(p.buildChips(state))
	column.AddChild(p.buildBandLine(state, band))
	if state.DetailsOpen && band != nil {
		column.AddChild(p.buildDetails(state, band))
	}
	if guide := p.buildGuideCard(state); guide != nil {
		column.AddChild(guide)
	}
	column.AddChild(p.buildChecklist(state, band))
	column.AddChild(t.label("Space ends the turn · Tab next band · ? shortcuts", 9.5, colorDim))
	return column
}

// buildGuideCard is a stub; Task 16 implements the first-turn guide overlay.
func (p *Panel) buildGuideCard(State) widget.PreferredSizeLocateableWidget { return nil }

// buildChecklist is the three-row Move/Research/Workforce checklist (spec
// §5) plus the End turn button.
func (p *Panel) buildChecklist(state State, band *gameapi.Band) widget.PreferredSizeLocateableWidget {
	t := p.theme
	column := t.column(4, nil, nil, stretch())
	column.AddChild(t.label("THIS TURN", 9.5, colorDim))
	if band == nil {
		column.AddChild(t.label("Select a band on the map or a chip above.", 10, colorText))
		column.AddChild(p.buildEndTurn(state))
		return column
	}
	summaries := [ui.ChecklistRowCount]string{
		ui.MoveSummary(state.Frame, *band), ui.ResearchSummary(*band), ui.WorkforceSummary(state.Workforce.AllocationBP, state.Workforce.Dirty),
	}
	dones := [ui.ChecklistRowCount]bool{ui.MoveDone(*band), ui.ResearchDone(*band), false}
	for row := ui.ChecklistRow(0); row < ui.ChecklistRowCount; row++ {
		open := row == state.OpenRow
		header := p.rowHeader(row, dones[row], open, summaries[row])
		if row == ui.RowWorkforce {
			p.handles.workforce.header = header
		}
		column.AddChild(header)
		if !open {
			continue
		}
		switch row {
		case ui.RowMove:
			column.AddChild(p.buildMoveBody(state, band))
		case ui.RowResearch:
			column.AddChild(p.buildResearchBody(state, band))
		case ui.RowWorkforce:
			if state.Workforce.Visible {
				column.AddChild(p.buildWorkforceBody(state, band))
			} else {
				column.AddChild(t.label("Computer controlled · allocation read only", 9.5, colorDim))
			}
		}
	}
	column.AddChild(p.buildEndTurn(state))
	return column
}
