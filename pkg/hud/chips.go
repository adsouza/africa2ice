package hud

import (
	"fmt"
	"image"
	"image/color"

	"github.com/adsouza/africa2ice/pkg/gameapi"
	"github.com/adsouza/africa2ice/pkg/ui"
	"github.com/ebitenui/ebitenui/widget"
	"github.com/hajimehoshi/ebiten/v2/text/v2"
)

const maxVisibleChips = 8

// chipRowScrollbarAllowance is the DIP width reserved for the scroll
// container's chrome. The panel draws no visible scrollbar (wireScrollWheel
// makes the wheel the only way to reach hidden content), but the available
// width for chips still needs a small margin so a chip row that exactly
// fills the column does not read as touching the edge.
const chipRowScrollbarAllowance = 12.0

// chipColumnPaddingDIP is chipButton's horizontal text padding
// (t.insets(1, 8, 8, 1): left 8 + right 8).
const chipColumnPaddingDIP = 16.0

// chipColumnSpacingDIP is the grid spacing buildChips passes to
// GridLayoutOpts.Spacing; it must match so the width math and the actual
// layout agree.
const chipColumnSpacingDIP = 4.0

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

// chipLabel is the text a chip button renders: the ConditionForSapiensBand
// marker prefix (empty, "!" or "!!") plus "B<id>". buildChips measures this
// same string to size the grid, so it lives here rather than being inlined
// into chipButton.
func chipLabel(band gameapi.Band) string {
	label := fmt.Sprintf("B%d", band.ID)
	if marker := ui.ConditionForSapiensBand(band).Marker(); marker != "" {
		label = marker + " " + label
	}
	return label
}

func (p *Panel) chipButton(band gameapi.Band, selected bool) *widget.Button {
	t := p.theme
	done := ui.MoveDone(band)
	border, fill, borderPx := chipColors(done, selected)
	label := chipLabel(band)
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

// chipColumns derives the chip grid's column count from the widest label
// actually being rendered. A fixed count cannot serve both ends: mostly
// single-digit band IDs waste roughly half the column's width at a fixed 4,
// but band IDs reach three digits and a suffering chip's "!!" prefix makes
// "!! B256" far wider than "B3", so a fixed 8 would overflow the column once
// IDs and warnings run long. columns is clamped to [4, maxVisibleChips] and
// never zero.
func (p *Panel) chipColumns(labels []string) int {
	t := p.theme
	var widest float64
	face := *t.face(10.5)
	for _, label := range labels {
		if width, _ := text.Measure(label, face, 0); width > widest {
			widest = width
		}
	}
	chipWidth := int(widest+0.5) + t.px(chipColumnPaddingDIP) + t.px(chipColumnSpacingDIP)
	if chipWidth <= 0 {
		return maxVisibleChips
	}
	available := t.px(panelWidth-2*panelPadding) - t.px(chipRowScrollbarAllowance)
	return max(4, min(maxVisibleChips, available/chipWidth))
}

// buildChips is the band row (spec §4 item 2): a grid sized to how many
// chips of the actual, current label width fit the column, plus a +N chip
// past eight bands.
func (p *Panel) buildChips(state State) widget.PreferredSizeLocateableWidget {
	t := p.theme
	visible, remaining := visibleChipIDs(state.Frame, state.SelectedBand)
	byID := make(map[gameapi.BandID]gameapi.Band, len(visible))
	for _, band := range state.Frame.Bands {
		byID[band.ID] = band
	}
	labels := make([]string, 0, len(visible)+1)
	for _, id := range visible {
		labels = append(labels, chipLabel(byID[id]))
	}
	moreLabel := fmt.Sprintf("+%d", remaining)
	if remaining > 0 {
		labels = append(labels, moreLabel)
	}
	columns := p.chipColumns(labels)
	grid := widget.NewContainer(
		widget.ContainerOpts.Layout(widget.NewGridLayout(widget.GridLayoutOpts.Columns(columns), widget.GridLayoutOpts.Spacing(t.px(chipColumnSpacingDIP), t.px(4)))),
		widget.ContainerOpts.WidgetOpts(stretch()),
	)
	for _, id := range visible {
		band := byID[id]
		chip := p.chipButton(band, band.ID == state.SelectedBand)
		p.handles.chips[uint32(band.ID)] = chip
		grid.AddChild(chip)
	}
	if remaining > 0 {
		more := t.button(moreLabel, 10.5, colorDim, colorText, func() { p.emit(Intent{Kind: IntentToggleBandList}) })
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
