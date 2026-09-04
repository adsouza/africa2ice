package hud

import (
	"fmt"
	"image/color"

	"github.com/adsouza/africa2ice/pkg/gameapi"
	"github.com/adsouza/africa2ice/pkg/ui"
	"github.com/ebitenui/ebitenui/widget"
)

func tierColor(tier ui.LiveabilityTier) color.RGBA {
	switch tier {
	case ui.TierAmber:
		return colorAmber
	case ui.TierRed:
		return colorRed
	default:
		return colorText
	}
}

// deltaMark is the TARGET column's comparison mark and its colour. A
// difference the band would not feel keeps the mark — it still ranks the two
// tiles — but drops to the dim colour: colouring an inconsequential
// difference red or green cried wolf (user-reported). Both buildMoveBody and
// refreshTarget go through here so the two paths cannot disagree.
func deltaMark(row ui.LiveabilityRow) (string, color.RGBA) {
	mark := ""
	switch {
	case row.Delta > 0:
		mark = " ▲"
	case row.Delta < 0:
		mark = " ▼"
	default:
		return "", colorText
	}
	if !row.DeltaMaterial {
		return mark, colorDim
	}
	if row.Delta > 0 {
		return mark, colorGreen
	}
	return mark, colorRed
}

// moveGridFontDIP is the HERE / TARGET grid's text size. Named because
// TestMoveGridValuesFitTheirColumns depends on the value cells and their
// column being measured at the same size.
const moveGridFontDIP = 9.5

// moveTargetCell is one TARGET value and its comparison mark, as two labels in
// a row. A widget.Text carries a single colour, so appending the mark to the
// value forced one colour on both — a dim mark would have greyed out the value
// beside it. buildPartnerGenetics splits its cells for the same reason.
func (p *Panel) moveTargetCell(index int, row ui.LiveabilityRow) *widget.Container {
	t := p.theme
	cell := t.rowOf(0)
	value := t.label(row.Target, moveGridFontDIP, tierColor(row.TargetTier))
	mark, markColor := deltaMark(row)
	markLabel := t.label(mark, moveGridFontDIP, markColor)
	p.handles.moveTargetValues[index], p.handles.moveTargetMarks[index] = value, markLabel
	p.handles.moveTargetCells[index] = cell
	cell.AddChild(value, markLabel)
	return cell
}

// rowHeader is the always-visible line of a checklist row: number badge,
// title, and one-line summary. Clicking it opens the row.
func (p *Panel) rowHeader(row ui.ChecklistRow, done, open bool, summary string) *widget.Button {
	t := p.theme
	badge := colorPanelEdge
	if done {
		badge = colorGreen
	} else if row != ui.RowWorkforce {
		badge = colorGold
	}
	border := colorPanelEdge
	if open {
		border = colorGold
	}
	label := fmt.Sprintf("%d  %-10s %s", int(row)+1, row.Title(), summary)
	button := widget.NewButton(
		widget.ButtonOpts.Image(t.buttonImages(border)),
		widget.ButtonOpts.Text(label, t.face(11), t.buttonText(badge)),
		widget.ButtonOpts.TextPadding(t.insets(6, 10, 10, 6)),
		widget.ButtonOpts.TextPosition(widget.TextPositionStart, widget.TextPositionCenter),
		widget.ButtonOpts.ClickedHandler(func(*widget.ButtonClickedEventArgs) { p.emit(Intent{Kind: IntentOpenRow, Row: row}) }),
		widget.ButtonOpts.WidgetOpts(stretch(), widget.WidgetOpts.CursorHovered("pointer")),
	)
	p.handles.rowHeader[row] = button
	return button
}

// buildMoveBody is the open Move row (spec §4.1). Every widget in its TARGET
// column (the header cell, the eight comparison values, the optional
// "unavailable" status row, moveHere's Disabled state, and the hint line) is
// also refreshed in place by refreshTarget, so a hover, arrow-key cursor, or
// queued-migration change repaints without rebuilding the tree.
func (p *Panel) buildMoveBody(state State, band *gameapi.Band) widget.PreferredSizeLocateableWidget {
	t := p.theme
	body := t.column(4, t.insets(6, 24, 10, 8), t.solid(colorRowOpen), stretch())
	here := ui.CurrentTileLiveability(state.Frame, band)
	targetTile, source := ui.TargetTile(band, state.Preview, state.Hover)
	target := ui.TileLiveability{}
	if source != ui.TargetNone {
		target = ui.TargetTileLiveability(state.Frame, band, targetTile)
	}
	grid := widget.NewContainer(
		widget.ContainerOpts.Layout(widget.NewGridLayout(widget.GridLayoutOpts.Columns(3), widget.GridLayoutOpts.Spacing(t.px(8), t.px(2)), widget.GridLayoutOpts.Stretch([]bool{false, true, true}, nil))),
		widget.ContainerOpts.WidgetOpts(stretch()),
	)
	targetHeader := "TARGET"
	if source != ui.TargetNone {
		targetHeader = "TARGET · " + source.Label()
	}
	targetHeaderLabel := t.label(targetHeader, 9.5, colorCyan)
	p.handles.moveTargetHeader = targetHeaderLabel
	grid.AddChild(t.label("", 9.5, colorDim), t.label("HERE", 9.5, colorGold), targetHeaderLabel)
	for index, row := range ui.LiveabilityRows(band, here, target) {
		grid.AddChild(t.label(row.Label, moveGridFontDIP, colorDim))
		grid.AddChild(t.label(row.Here, moveGridFontDIP, tierColor(row.HereTier)))
		grid.AddChild(p.moveTargetCell(index, row))
	}
	p.handles.moveTargetStatus = nil
	if !target.Available && source != ui.TargetNone {
		status := t.label(target.Status, 9, colorDim)
		p.handles.moveTargetStatus = status
		grid.AddChild(t.label("", 9, colorDim), t.label("", 9, colorDim), status)
	}
	body.AddChild(grid)

	// Two rows of two, each button stretched to share its row evenly: a
	// single row of four is wider than the 352 DIP column (spec §4 item 4;
	// user-reported), clipping Interbreed at the panel's right edge.
	buttonRow1, buttonRow2 := t.rowOf(6, stretch()), t.rowOf(6, stretch())
	done := ui.MoveDone(*band)
	// A computer-controlled selection is read only: the row still shows the
	// HERE/TARGET comparison, but none of its four actions may be sent.
	readOnly := band.Species != gameapi.HomoSapiens || state.CampaignOver
	canMove := !done && !readOnly && target.Reachable && source != ui.TargetQueued
	moveHere := t.button("Move here · Enter", 10.5, colorCyan, colorCyan, func() { p.emit(Intent{Kind: IntentMoveTo, Tile: targetTile}) })
	moveHere.GetWidget().Disabled = !canMove
	moveHere.GetWidget().LayoutData = widget.RowLayoutData{Stretch: true}
	p.handles.moveHere = moveHere
	best := t.button("Best tile · B", 10.5, colorGoldDeep, colorGoldDeep, func() { p.emit(Intent{Kind: IntentMoveToBest}) })
	// A non-empty candidate list is not enough: moveToBestTile needs one that
	// requires no passage (user-reported, same defect as Split above).
	best.GetWidget().Disabled = done || readOnly || !ui.HasOrdinaryLandCandidate(*band)
	best.GetWidget().LayoutData = widget.RowLayoutData{Stretch: true}
	p.handles.best = best
	split := t.button("Split · N", 10.5, colorGoldDeep, colorGoldDeep, func() { p.emit(Intent{Kind: IntentSplit}) })
	// ui.DiagnoseSplit subsumes done and the species check and adds the guards
	// the panel could not see before: crowding pressure, the minimum viable
	// population, the band limit, and whether any adjacent land is available to
	// settle. Without them the button was offered and the click only produced a
	// notice (user-reported).
	split.GetWidget().Disabled = readOnly || ui.DiagnoseSplit(state.Frame, band) != ""
	split.GetWidget().LayoutData = widget.RowLayoutData{Stretch: true}
	p.handles.split = split
	partner := state.InterbreedFocus
	if partner == 0 && len(band.InterbreedCandidateIDs) > 0 {
		partner = band.InterbreedCandidateIDs[0]
	}
	interbreed := t.button("Interbreed · I", 10.5, colorInterbreed, colorInterbreed, func() { p.emit(Intent{Kind: IntentInterbreed, Band: partner}) })
	interbreed.GetWidget().Disabled = done || readOnly || len(band.InterbreedCandidateIDs) == 0
	interbreed.GetWidget().LayoutData = widget.RowLayoutData{Stretch: true}
	p.handles.interbreed = interbreed
	buttonRow1.AddChild(moveHere, best)
	buttonRow2.AddChild(split, interbreed)
	body.AddChild(buttonRow1, buttonRow2)
	if len(band.InterbreedCandidateIDs) > 1 && !done {
		picker := t.rowOf(4)
		picker.AddChild(t.label("Partner:", 9, colorDim))
		for _, candidate := range band.InterbreedCandidateIDs {
			id := candidate
			border := colorPanelEdge
			if id == partner {
				border = colorInterbreed
			}
			picker.AddChild(t.button(fmt.Sprintf("B%d", id), 9, border, colorInterbreed, func() { p.emit(Intent{Kind: IntentInterbreed, Band: id}) }))
		}
		body.AddChild(picker)
	}
	if band.Species == gameapi.HomoSapiens && !done && len(band.InterbreedCandidateIDs) > 0 {
		if partnerBand := bandWithID(state.Frame, partner); partnerBand != nil {
			body.AddChild(p.buildPartnerGenetics(band, partnerBand))
		}
	}
	hint := "Arrows move a cursor instead of the pointer · Esc clears it · staying put is fine"
	switch {
	case band.Species == gameapi.ArchaicHominin:
		hint = "Computer controlled · no player actions"
	case source == ui.TargetNone:
		hint = "Hover or click an outlined tile · arrows move a cursor · Esc clears"
	}
	hintLabel := t.label(hint, 8.5, colorDim)
	p.handles.moveHint = hintLabel
	body.AddChild(hintLabel)
	return body
}

// refreshTarget updates the Move row's TARGET column (and moveHere's Disabled
// state) from a new hover, arrow-key cursor, or queued-migration target
// without rebuilding the tree. HERE is not touched here: it is keyed off the
// selected band's own tile, which only changes through fields already in the
// structural comparison.
//
// It reports whether the refresh could be applied in place. It cannot when
// the TARGET column's own shape would need to change: the Move row is not
// currently built for the open row (a rebuild is about to, or already did,
// handle that through the ordinary structural path), or the "unavailable"
// status row's presence would flip because the target tile's availability
// changed. The caller rebuilds instead in either case.
func (p *Panel) refreshTarget(state State) bool {
	band := state.selectedBand()
	built := p.handles.moveTargetHeader != nil
	if state.OpenRow != ui.RowMove || band == nil {
		return !built
	}
	if !built {
		return false
	}
	targetTile, source := ui.TargetTile(band, state.Preview, state.Hover)
	target := ui.TileLiveability{}
	if source != ui.TargetNone {
		target = ui.TargetTileLiveability(state.Frame, band, targetTile)
	}
	if statusRow := !target.Available && source != ui.TargetNone; statusRow != (p.handles.moveTargetStatus != nil) {
		return false
	}
	p.refreshes++

	targetHeader := "TARGET"
	if source != ui.TargetNone {
		targetHeader = "TARGET · " + source.Label()
	}
	p.handles.moveTargetHeader.Label = targetHeader

	here := ui.CurrentTileLiveability(state.Frame, band)
	for index, row := range ui.LiveabilityRows(band, here, target) {
		value, markLabel := p.handles.moveTargetValues[index], p.handles.moveTargetMarks[index]
		value.Label = row.Target
		value.SetColor(tierColor(row.TargetTier))
		mark, markColor := deltaMark(row)
		markLabel.Label = mark
		markLabel.SetColor(markColor)
	}
	if p.handles.moveTargetStatus != nil {
		p.handles.moveTargetStatus.Label = target.Status
	}

	done := ui.MoveDone(*band)
	readOnly := band.Species != gameapi.HomoSapiens || state.CampaignOver
	canMove := !done && !readOnly && target.Reachable && source != ui.TargetQueued
	p.handles.moveHere.GetWidget().Disabled = !canMove

	hint := "Arrows move a cursor instead of the pointer · Esc clears it · staying put is fine"
	switch {
	case band.Species == gameapi.ArchaicHominin:
		hint = "Computer controlled · no player actions"
	case source == ui.TargetNone:
		hint = "Hover or click an outlined tile · arrows move a cursor · Esc clears"
	}
	p.handles.moveHint.Label = hint
	return true
}

// partnerTraitColor is the comparison colour for one heritable trait cell in
// buildPartnerGenetics: green when the partner is higher, dim when lower,
// and the neutral text colour when equal. This is a difference display, not
// a judgement about which value is better.
func partnerTraitColor(self, other float64) color.RGBA {
	switch {
	case other > self:
		return colorGreen
	case other < self:
		return colorDim
	default:
		return colorText
	}
}

// bandWithID finds a band by ID in the frame, or nil. Unlike
// State.selectedBand it takes an arbitrary ID, so buildPartnerGenetics can
// look up the focused interbreeding candidate rather than the selection.
func bandWithID(frame *gameapi.Frame, id gameapi.BandID) *gameapi.Band {
	if frame == nil {
		return nil
	}
	for index := range frame.Bands {
		if frame.Bands[index].ID == id {
			return &frame.Bands[index]
		}
	}
	return nil
}

// buildPartnerGenetics is the compact block shown below the partner picker
// while interbreeding is still open (spec F2): interbreeding moves the
// band's heritable traits toward the partner's, but until now the panel
// never showed the partner's own values, so a choice among more than one
// candidate was blind. Every trait cell splits into two labels so only the
// arrow-and-partner-value half carries the comparison colour — the self
// value stays neutral, since this is a difference display, not a judgement
// about which value is better.
func (p *Panel) buildPartnerGenetics(band, partner *gameapi.Band) widget.PreferredSizeLocateableWidget {
	t := p.theme
	column := t.column(3, t.insets(4, 0, 2, 0), nil, stretch())
	p.handles.partnerGenetics = column
	heading := t.label(fmt.Sprintf("PARTNER B%d · pop %d", partner.ID, partner.Population), 9, colorInterbreed)
	p.handles.partnerGeneticsHeading = heading
	column.AddChild(heading)
	grid := widget.NewContainer(
		widget.ContainerOpts.Layout(widget.NewGridLayout(
			widget.GridLayoutOpts.Columns(2),
			widget.GridLayoutOpts.Spacing(t.px(8), t.px(2)),
			widget.GridLayoutOpts.Stretch([]bool{true, true}, nil),
		)),
		widget.ContainerOpts.WidgetOpts(stretch()),
	)
	for trait := gameapi.HeritableTrait(0); trait < gameapi.HeritableTraitCount; trait++ {
		self, other := band.HeritableState[trait], partner.HeritableState[trait]
		valueColor := partnerTraitColor(self, other)
		cell := t.rowOf(2)
		cell.AddChild(t.label(fmt.Sprintf("%s %.2f", traitShortNames[trait], self), 8.5, colorDim))
		value := t.label(fmt.Sprintf("→ %.2f", other), 8.5, valueColor)
		p.handles.partnerGeneticsValues[trait] = value
		cell.AddChild(value)
		grid.AddChild(cell)
	}
	column.AddChild(grid)
	return column
}
