package hud

import (
	"image"

	"github.com/adsouza/africa2ice/pkg/gameapi"
	"github.com/adsouza/africa2ice/pkg/render"
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
	drawerHiddenH   = 20.0
	drawerCompactH  = 102.0
	drawerExpandedH = 300.0
	drawerTabW      = 150.0
	drawerTabH      = 18.0
	panelFooterH    = 62.0
	// panelHeaderMinH and panelHeaderMaxH bound the header region's height
	// once it is measured from its own content (spec §4 item 1): a
	// pathological measurement — an empty frame, a future field that grows
	// unboundedly — cannot eat into the scrollable middle or push the
	// footer below the panel this way.
	panelHeaderMinH = 92.0
	panelHeaderMaxH = 140.0
)

// DrawerHiddenHeight, DrawerCompactHeight and DrawerExpandedHeight are
// exported so pkg/app can derive the map's camera-visible height (spec §6)
// from the same numbers the drawer itself draws at, rather than duplicating
// them.
const (
	DrawerHiddenHeight   = drawerHiddenH
	DrawerCompactHeight  = drawerCompactH
	DrawerExpandedHeight = drawerExpandedH
)

// Panel owns the ebitenui tree for every piece of chrome. It rebuilds the tree
// whenever the State value changes and otherwise leaves widgets untouched so
// presses and hovers survive across ticks.
type Panel struct {
	ui     *ebitenui.UI
	theme  *theme
	root   *widget.Container
	last   State
	built  bool
	builds int
	// refreshes counts in-place widget writes made outside a rebuild
	// (refreshWorkforce, refreshVolume, refreshTarget), so PresentationKey
	// can see appearance changes that never touch p.builds.
	refreshes int
	intents   []Intent
	handles   handles
}

// handles keeps pointers to widgets tests and refreshes need to reach. Later
// tasks add fields here as they wire up the widgets those fields point to
// (guideNext, guideX and friends): a field with no writer or reader anywhere
// in the package is exactly the dead code the unused linter exists to catch,
// so it is added alongside its first use, not ahead of it.
type handles struct {
	chips    map[uint32]*widget.Button // keyed by gameapi.BandID
	more     *widget.Button
	bandList *widget.Window
	// bandListScroll is the band list window's own scroll container, kept
	// so PresentationKey can see its ScrollTop: the mouse wheel mutates it
	// directly (wireScrollWheel) without touching p.builds, p.refreshes, or
	// any of PresentationKey's other fields.
	bandListScroll *widget.ScrollContainer
	details        *widget.Button
	// headerContent, macroWarning and panelMiddle let TestHeaderRegionFitsItsContent
	// measure the header region against its own content instead of the old
	// fixed constant: headerContent is buildHeader's returned column (its
	// Rect.Max.Y is the bottom of its last child, the population line);
	// macroWarning is the optional warning line inside it; panelMiddle is
	// the scroll container the header's height must leave room above.
	headerContent *widget.Container
	macroWarning  *widget.Text
	panelMiddle   *widget.ScrollContainer
	bandDetail    *widget.Text
	detailsBody   *widget.Container
	traits        map[gameapi.HeritableTrait]*widget.Button
	// traitFocused is the trait cell matching state.Note.Trait when
	// state.Note.HasTrait is true, or nil when the drawer shows no trait
	// note (or details are collapsed). buildDetails sets it directly rather
	// than making callers search handles.traits, since ebitenui exposes no
	// colour getter on a built widget to verify the highlight another way.
	traitFocused *widget.Button
	overlay      *widget.Window
	rowHeader    [ui.ChecklistRowCount]*widget.Button
	moveHere     *widget.Button
	best         *widget.Button
	split        *widget.Button
	interbreed   *widget.Button
	// The TARGET column of the open Move row (moveTargetHeader,
	// moveTargetValues, moveTargetStatus) and moveHint are refreshed in place
	// by refreshTarget as the hover/cursor/queued target changes; nil when
	// the Move row is not open (refreshTarget is then a no-op).
	moveTargetHeader *widget.Text
	moveTargetValues [8]*widget.Text
	moveTargetStatus *widget.Text
	moveHint         *widget.Text
	// partnerGenetics, partnerGeneticsHeading and partnerGeneticsValues are
	// the focused-partner block below the interbreed picker; nil unless the
	// selected sapiens band has an open spatial action and at least one
	// interbreed candidate (see buildPartnerGenetics).
	partnerGenetics        *widget.Container
	partnerGeneticsHeading *widget.Text
	partnerGeneticsValues  [gameapi.HeritableTraitCount]*widget.Text
	research               [gameapi.TechCount]*widget.Button
	workforce              workforceHandles
	endTurn                *widget.Button
	drawerTab              *widget.Button
	drawerMore             *widget.Button
	// drawerBar and drawerBarEvent are the hidden-mode full-width bar and its
	// left-hand clickable event text; nil in compact/expanded mode, where the
	// edge tab (drawerTab/drawerMore above) is what tests and refreshes reach.
	drawerBar      *widget.Container
	drawerBarEvent *widget.Button
	// notesScroll is the Field Notes drawer body's own ScrollContainer (see
	// buildDrawer): pkg/hud owns its scrolling the same way it owns
	// panelMiddle's and bandListScroll's, so ScrollTop is directly readable
	// here for PresentationKey, and tests use this handle to prove a
	// hover-driven Update did not discard and recreate it (see
	// refreshTarget).
	notesScroll *widget.ScrollContainer
	events      []*widget.Button
	camera      *widget.Button
	guideNext   *widget.Button
	guideX      *widget.Button

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
	structural.Hover, lastStructural.Hover = render.TileHover{}, render.TileHover{}
	if !p.built || structural != lastStructural {
		p.rebuild(state)
		p.built = true
	} else {
		// Each applicable refresh runs independently rather than through a
		// mutually-exclusive switch: two of Workforce, Overlay.MasterVolume
		// and Hover can change in the same tick, and a switch would run only
		// the first matching case while p.last = state still absorbed every
		// field's new value — silently dropping the other field's refresh
		// with no way for a later tick to detect it.
		if state.Workforce != p.last.Workforce {
			p.refreshWorkforce(state)
		}
		if state.Overlay.MasterVolume != p.last.Overlay.MasterVolume {
			p.refreshVolume(state)
		}
		if state.Hover != p.last.Hover {
			if !p.refreshTarget(state) {
				p.rebuild(state)
				p.built = true
			}
		}
		p.last = state
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

// PresentationKey is a comparable snapshot of everything that can change what
// this Panel draws. pkg/app hashes it into an opaque uint64 (pkg/render must
// not learn about pkg/hud or ebitenui) and feeds it to the map scene as its
// chrome revision, so a chrome-only appearance change still forces a repaint
// even though the map's own frame key is unchanged.
type PresentationKey struct {
	builds      int
	refreshes   int
	cursorX     int
	cursorY     int
	mouseLeft   bool
	mouseRight  bool
	mouseMiddle bool
	uiHovered   bool
	panelScroll float64
	bandListTop float64
	notesTop    float64
}

// PresentationKey returns a value that changes whenever anything this Panel
// draws could look different this tick:
//   - p.builds, bumped by every structural rebuild;
//   - p.refreshes, bumped by every in-place refresh that actually writes to
//     a widget (refreshWorkforce, refreshVolume, refreshTarget) — these
//     mutate widgets Update's structural comparison deliberately excludes;
//   - the cursor position and mouse button state, which drive ebitenui's
//     hover and pressed visuals without going through Update at all;
//   - input.UIHovered, ebitenui's own hover flag;
//   - the panel's own scroll offset (panelMiddle.ScrollTop), the band list
//     window's scroll offset (bandListScroll.ScrollTop, 0 when the window is
//     closed), and the Field Notes drawer body's scroll offset
//     (notesScroll.ScrollTop, 0 when the drawer is hidden) — wireScrollWheel
//     mutates a *widget.ScrollContainer's ScrollTop directly from the mouse
//     wheel, with no other side effect any of the fields above would catch,
//     so a scroll-only tick would otherwise report the same key as the tick
//     before it and the map would skip repainting over the newly-scrolled
//     chrome. (Opening or closing the band list window, or switching the
//     drawer's mode, is itself a rebuild, already covered by p.builds.)
//
// A widget added later with animation of its own that none of these fields
// already track (a spinner, a blinking caret, a hover-delayed tooltip) must
// extend this key too, or that animation will not repaint on an otherwise-
// idle frame.
func (p *Panel) PresentationKey() PresentationKey {
	x, y := ebiten.CursorPosition()
	var panelScroll float64
	if p.handles.panelMiddle != nil {
		panelScroll = p.handles.panelMiddle.ScrollTop
	}
	var bandListTop float64
	if p.handles.bandListScroll != nil {
		bandListTop = p.handles.bandListScroll.ScrollTop
	}
	var notesTop float64
	if p.handles.notesScroll != nil {
		notesTop = p.handles.notesScroll.ScrollTop
	}
	return PresentationKey{
		builds:      p.builds,
		refreshes:   p.refreshes,
		cursorX:     x,
		cursorY:     y,
		mouseLeft:   ebiten.IsMouseButtonPressed(ebiten.MouseButtonLeft),
		mouseRight:  ebiten.IsMouseButtonPressed(ebiten.MouseButtonRight),
		mouseMiddle: ebiten.IsMouseButtonPressed(ebiten.MouseButtonMiddle),
		uiHovered:   input.UIHovered,
		panelScroll: panelScroll,
		bandListTop: bandListTop,
		notesTop:    notesTop,
	}
}

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

// buildPanel is the panel's background plus three fixed-rect regions (spec
// §4): a header, a scrollable middle holding chips through the checklist
// rows, and a footer pinning End turn and the hint line to the bottom. A
// single RowLayout column for the whole panel could exceed the 632 DIP
// column height with the guide card and details both open, pushing End turn
// and the footer off-screen; RowLayout itself neither shrinks nor scrolls,
// so the middle region is a ScrollContainer instead.
func (p *Panel) buildPanel(state State) widget.PreferredSizeLocateableWidget {
	t := p.theme
	background := widget.NewContainer(
		widget.ContainerOpts.Layout(fixedLayout{}),
		widget.ContainerOpts.BackgroundImage(t.solid(colorPanel)),
		widget.ContainerOpts.WidgetOpts(widget.WidgetOpts.LayoutData(p.rect(panelX, panelY, panelWidth, panelHeight))),
	)
	band := state.selectedBand()

	// The header region used to be a fixed constant sized for the busiest
	// case (a macro warning line present), which left dead space above the
	// chips whenever that line was absent. Building the inner content first
	// and measuring it means the region always fits what it actually holds,
	// with panelHeaderMinH/MaxH as a sane backstop rather than a tuned
	// constant that only ever matched one case.
	headerInsets := t.insets(14, panelPadding, panelPadding, 6)
	headerContent := p.buildHeader(state)
	_, contentHeightPx := headerContent.PreferredSize()
	headerHeightPx := contentHeightPx + headerInsets.Top + headerInsets.Bottom
	headerHeightPx = max(t.px(panelHeaderMinH), min(t.px(panelHeaderMaxH), headerHeightPx))
	scale := p.theme.scale
	if scale <= 0 {
		scale = 1
	}
	headerHeightDIP := float64(headerHeightPx) / scale

	header := t.column(2, headerInsets, nil,
		widget.WidgetOpts.LayoutData(p.rect(panelX, panelY, panelWidth, headerHeightDIP)))
	header.AddChild(headerContent)
	background.AddChild(header)

	body := t.column(8, t.insets(8, panelPadding, panelPadding, 8), nil, stretch())
	body.AddChild(p.buildChips(state))
	body.AddChild(p.buildBandLine(state, band))
	if state.DetailsOpen && band != nil {
		body.AddChild(p.buildDetails(state, band))
	}
	if guide := p.buildGuideCard(state); guide != nil {
		body.AddChild(guide)
	}
	body.AddChild(p.buildChecklistRows(state, band))

	middleHeight := panelHeight - headerHeightDIP - panelFooterH
	content := scrollContent{Container: body, widthPx: t.px(panelWidth)}
	scroll := widget.NewScrollContainer(
		widget.ScrollContainerOpts.Content(content),
		widget.ScrollContainerOpts.StretchContentWidth(),
		widget.ScrollContainerOpts.Image(&widget.ScrollContainerImage{
			Idle: t.solid(colorPanel), Disabled: t.solid(colorPanel), Mask: t.solid(colorPanel),
		}),
		widget.ScrollContainerOpts.WidgetOpts(widget.WidgetOpts.LayoutData(
			p.rect(panelX, panelY+headerHeightDIP, panelWidth, middleHeight))),
	)
	p.handles.panelMiddle = scroll
	p.wireScrollWheel(scroll, content)
	background.AddChild(scroll)

	footer := t.column(6, t.insets(8, panelPadding, panelPadding, 12), nil,
		widget.WidgetOpts.LayoutData(p.rect(panelX, panelY+panelHeight-panelFooterH, panelWidth, panelFooterH)))
	// Once the campaign is over there is no turn left to end, so the button
	// is omitted rather than drawn as an empty disabled bar (spec §5.3).
	if state.Frame.CampaignResult == gameapi.Ongoing {
		footer.AddChild(p.buildEndTurn(state))
	}
	footer.AddChild(t.label("Space ends the turn · Tab next band · ? shortcuts", 9.5, colorDim))
	background.AddChild(footer)

	return background
}

// scrollContent wraps the scrollable middle's column and reports a fixed
// width regardless of what its descendants would naturally prefer. Without
// this, a single non-stretched wide widget anywhere in the checklist (a
// pre-existing overflow in, say, an open row's button strip — out of scope
// here) would inflate the whole column's measured PreferredSize, and
// ScrollContainer's StretchContentWidth only ever grows content that measures
// narrower than the viewport; it never shrinks content that measures wider.
// That inflated width would then flow back down through every *stretched*
// descendant (including buildDetails' own bounded container), widening each
// past the panel's right edge even though its own text is correctly bounded.
// Height still reports the column's true preferred height, so vertical
// scrolling measures correctly; only the reported width is pinned.
type scrollContent struct {
	*widget.Container
	widthPx int
}

func (c scrollContent) PreferredSize() (int, int) {
	_, height := c.Container.PreferredSize()
	return c.widthPx, height
}

// wireScrollWheel lets the mouse wheel move the panel's scrollable middle
// region. There is no visible scrollbar (the panel already has a details
// disclosure and a checklist for revealing more content), so the wheel is
// the only way to reach content below the fold.
func (p *Panel) wireScrollWheel(scroll *widget.ScrollContainer, content widget.PreferredSizeLocateableWidget) {
	const wheelStepDIP = 48.0
	scroll.GetWidget().ScrolledEvent.AddHandler(func(args interface{}) {
		event, ok := args.(*widget.WidgetScrolledEventArgs)
		if !ok {
			return
		}
		overflow := float64(content.GetWidget().Rect.Dy() - scroll.ViewRect().Dy())
		if overflow <= 0 {
			return
		}
		scroll.ScrollTop -= event.Y * float64(p.theme.px(wheelStepDIP)) / overflow
	})
}

// buildChecklistRows is the three-row Move/Research/Workforce checklist
// (spec §5); End turn is built and placed separately, in the panel's footer.
func (p *Panel) buildChecklistRows(state State, band *gameapi.Band) widget.PreferredSizeLocateableWidget {
	t := p.theme
	column := t.column(4, nil, nil, stretch())
	column.AddChild(t.label("THIS TURN", 9.5, colorDim))
	if band == nil {
		column.AddChild(t.label("Select a band on the map or a chip above.", 10, colorText))
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
	return column
}
