package hud

import (
	"image"

	"github.com/adsouza/africa2ice/pkg/gameapi"
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
// (endTurn, rowHeader, guideNext, guideX and friends): a field with no writer
// or reader anywhere in the package is exactly the dead code the unused
// linter exists to catch, so it is added alongside its first use, not ahead
// of it.
type handles struct {
	chips    map[uint32]*widget.Button // keyed by gameapi.BandID
	more     *widget.Button
	bandList *widget.Window
	details  *widget.Button
	traits   map[gameapi.HeritableTrait]*widget.Button
	overlay  *widget.Window
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
	if !p.built || state != p.last {
		p.rebuild(state)
		p.last = state
		p.built = true
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
	if drawer := p.buildDrawer(state); drawer != nil {
		p.root.AddChild(drawer)
	}
	if ending := p.buildEndScene(state); ending != nil {
		p.root.AddChild(ending)
	}
	p.buildOverlay(state)
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

// buildChecklist is a stub; Task 9 implements the move/research/interbreed
// checklist rows.
func (p *Panel) buildChecklist(State, *gameapi.Band) widget.PreferredSizeLocateableWidget {
	return p.theme.column(6, nil, nil, stretch())
}

// The three builders below are temporary minimal stand-ins so the package
// compiles and the Panel lifecycle can be exercised; Tasks 9-13 replace them
// file by file with the real chrome.
func (p *Panel) buildDrawer(State) widget.PreferredSizeLocateableWidget   { return nil }
func (p *Panel) buildEndScene(State) widget.PreferredSizeLocateableWidget { return nil }
func (p *Panel) buildOverlay(State)                                       {}
