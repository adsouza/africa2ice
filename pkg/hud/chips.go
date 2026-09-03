package hud

import (
	"fmt"
	"image"
	"image/color"

	"github.com/adsouza/africa2ice/pkg/gameapi"
	"github.com/adsouza/africa2ice/pkg/ui"
	"github.com/ebitenui/ebitenui/widget"
)

const maxVisibleChips = 8

// Band list window geometry (spec §4 item 2). Past roughly 35 bands the
// computed height would push rows and the Close button below the 720 DIP
// presentation, so the window caps at bandListMaxHeight and scrolls instead.
const (
	bandListWidth     = 330.0
	bandListHeaderH   = 28.0
	bandListFooterH   = 34.0
	bandListRowHeight = 16.0
	bandListChromeMin = 40.0
	bandListMaxHeight = 420.0
)

// visibleChipIDs returns at most eight band IDs in attention order, always
// including the selected band, and the count left over for the +N chip.
func visibleChipIDs(frame *gameapi.Frame, selected gameapi.BandID) ([]gameapi.BandID, int) {
	ordered := ui.SapiensBandIDsByAttention(frame.Bands)
	if len(ordered) <= maxVisibleChips {
		return ordered, 0
	}
	visible := append([]gameapi.BandID(nil), ordered[:maxVisibleChips-1]...)
	included := false
	for _, id := range visible {
		if id == selected {
			included = true
		}
	}
	if !included {
		for _, id := range ordered {
			if id == selected {
				visible[len(visible)-1] = id
			}
		}
	}
	return visible, len(ordered) - len(visible)
}

// chipColors picks the chip's Move-status border/fill/border-width: the
// border colour always signals Move status (gold open, green done), and
// selection adds a distinct fill plus a thicker ring rather than overriding
// the status colour (spec §4 item 2).
func chipColors(done, selected bool) (border, fill color.RGBA, borderPx float64) {
	border = colorGold
	if done {
		border = colorGreen
	}
	fill = colorButtonIdle
	if selected {
		fill = colorButtonHover
	}
	borderPx = 1
	if selected {
		borderPx = 2
	}
	return border, fill, borderPx
}

func (p *Panel) chipButton(band gameapi.Band, selected bool) *widget.Button {
	t := p.theme
	done := ui.MoveDone(band)
	border, fill, borderPx := chipColors(done, selected)
	label := fmt.Sprintf("B%d", band.ID)
	if marker := ui.ConditionForSapiensBand(band).Marker(); marker != "" {
		label = marker + " " + label
	}
	width := t.px(borderPx)
	images := &widget.ButtonImage{
		Idle:    t.bordered(fill, border, width),
		Hover:   t.bordered(colorButtonHover, border, width),
		Pressed: t.bordered(colorButtonDown, border, width),
	}
	id := band.ID
	button := widget.NewButton(
		widget.ButtonOpts.Image(images),
		widget.ButtonOpts.Text(label, t.face(10.5), t.buttonText(border)),
		widget.ButtonOpts.TextPadding(t.insets(1, 8, 8, 1)),
		widget.ButtonOpts.ClickedHandler(func(*widget.ButtonClickedEventArgs) { p.emit(Intent{Kind: IntentSelectBand, Band: id}) }),
		widget.ButtonOpts.WidgetOpts(widget.WidgetOpts.CursorHovered("pointer")),
	)
	return button
}

// buildChips is the band row (spec §4 item 2): a 4-column grid so eight chips
// occupy two lines, plus a +N chip past eight bands.
func (p *Panel) buildChips(state State) widget.PreferredSizeLocateableWidget {
	t := p.theme
	grid := widget.NewContainer(
		widget.ContainerOpts.Layout(widget.NewGridLayout(widget.GridLayoutOpts.Columns(4), widget.GridLayoutOpts.Spacing(t.px(4), t.px(4)))),
		widget.ContainerOpts.WidgetOpts(stretch()),
	)
	visible, remaining := visibleChipIDs(state.Frame, state.SelectedBand)
	for _, id := range visible {
		for _, band := range state.Frame.Bands {
			if band.ID != id {
				continue
			}
			chip := p.chipButton(band, band.ID == state.SelectedBand)
			p.handles.chips[uint32(band.ID)] = chip
			grid.AddChild(chip)
		}
	}
	if remaining > 0 {
		more := t.button(fmt.Sprintf("+%d", remaining), 10.5, colorDim, colorText, func() { p.emit(Intent{Kind: IntentToggleBandList}) })
		p.handles.more = more
		grid.AddChild(more)
	}
	if state.BandListOpen {
		p.openBandList(state)
	}
	return grid
}

// openBandList shows every band in attention order as a modal list; clicking
// one selects it, and the application closes the list on selection. The
// window height is capped and the rows scroll (spec §4 item 2): sized from
// the count of living sapiens bands actually shown, not len(Frame.Bands),
// which also counts archaic and extinct entries the list never lists.
func (p *Panel) openBandList(state State) {
	t := p.theme
	ids := ui.SapiensBandIDsByAttention(state.Frame.Bands)

	windowX, windowY := panelX-340, panelY+60
	windowHeight := min(bandListChromeMin+bandListRowHeight*float64(len(ids)), bandListMaxHeight)
	if minHeight := bandListHeaderH + bandListFooterH + bandListRowHeight; windowHeight < minHeight {
		windowHeight = minHeight
	}

	body := widget.NewContainer(
		widget.ContainerOpts.Layout(fixedLayout{}),
		widget.ContainerOpts.BackgroundImage(t.bordered(colorRowOpen, colorGoldDeep, t.px(1))),
	)

	header := t.column(3, t.insets(12, 14, 14, 4), nil,
		widget.WidgetOpts.LayoutData(p.rect(windowX, windowY, bandListWidth, bandListHeaderH)))
	header.AddChild(t.label("ALL BANDS · priority order", 10, colorGoldDeep))
	body.AddChild(header)

	rows := t.column(3, t.insets(0, 14, 14, 0), nil, stretch())
	for _, id := range ids {
		for _, band := range state.Frame.Bands {
			if band.ID != id {
				continue
			}
			label := fmt.Sprintf("%-3s B%-3d pop %d · health %.0f%% · %s", ui.ConditionForSapiensBand(band).Marker(), band.ID, band.Population, band.Health*100, ui.MoveSummary(state.Frame, band))
			bandID := band.ID
			rows.AddChild(t.button(label, 10, colorPanelEdge, colorText, func() { p.emit(Intent{Kind: IntentSelectBand, Band: bandID}) }))
		}
	}
	// rows is wrapped the same way buildPanel's scroll content is (see
	// scrollContent's doc comment): its own natural preferred width would
	// otherwise blow out to the widest band's summary line, and
	// StretchContentWidth only ever grows content narrower than the
	// viewport, never shrinks content that measures wider.
	scrollHeight := windowHeight - bandListHeaderH - bandListFooterH
	content := scrollContent{Container: rows, widthPx: t.px(bandListWidth)}
	scroll := widget.NewScrollContainer(
		widget.ScrollContainerOpts.Content(content),
		widget.ScrollContainerOpts.StretchContentWidth(),
		widget.ScrollContainerOpts.Image(&widget.ScrollContainerImage{
			Idle: t.solid(colorRowOpen), Disabled: t.solid(colorRowOpen), Mask: t.solid(colorRowOpen),
		}),
		widget.ScrollContainerOpts.WidgetOpts(widget.WidgetOpts.LayoutData(
			p.rect(windowX, windowY+bandListHeaderH, bandListWidth, scrollHeight))),
	)
	p.wireScrollWheel(scroll, content)
	body.AddChild(scroll)

	footer := t.column(0, t.insets(4, 14, 14, 12), nil,
		widget.WidgetOpts.LayoutData(p.rect(windowX, windowY+windowHeight-bandListFooterH, bandListWidth, bandListFooterH)))
	footer.AddChild(t.button("Close · Esc", 10, colorGoldDeep, colorGoldDeep, func() { p.emit(Intent{Kind: IntentToggleBandList}) }))
	body.AddChild(footer)

	rect := image.Rectangle(p.rect(windowX, windowY, bandListWidth, windowHeight))
	window := widget.NewWindow(widget.WindowOpts.Contents(body), widget.WindowOpts.Modal(), widget.WindowOpts.CloseMode(widget.NONE), widget.WindowOpts.Location(rect))
	p.ui.AddWindow(window)
	p.handles.bandList = window
}
