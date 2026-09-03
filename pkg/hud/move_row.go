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

func deltaMark(delta int) (string, color.RGBA) {
	switch {
	case delta > 0:
		return " ▲", colorGreen
	case delta < 0:
		return " ▼", colorRed
	default:
		return "", colorText
	}
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

// buildMoveBody is the open Move row (spec §4.1).
func (p *Panel) buildMoveBody(state State, band *gameapi.Band) widget.PreferredSizeLocateableWidget {
	t := p.theme
	body := t.column(4, t.insets(6, 24, 10, 8), solid(colorRowOpen), stretch())
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
	grid.AddChild(t.label("", 9.5, colorDim), t.label("HERE", 9.5, colorGold), t.label(targetHeader, 9.5, colorCyan))
	for _, row := range ui.LiveabilityRows(band, here, target) {
		mark, markColor := deltaMark(row.Delta)
		grid.AddChild(t.label(row.Label, 9.5, colorDim))
		grid.AddChild(t.label(row.Here, 9.5, tierColor(row.HereTier)))
		targetLabel := t.label(row.Target+mark, 9.5, tierColor(row.TargetTier))
		if mark != "" && row.TargetTier == ui.TierNormal {
			targetLabel.SetColor(markColor)
		}
		grid.AddChild(targetLabel)
	}
	if !target.Available && source != ui.TargetNone {
		grid.AddChild(t.label("", 9, colorDim), t.label("", 9, colorDim), t.label(target.Status, 9, colorDim))
	}
	body.AddChild(grid)

	buttons := t.rowOf(6)
	done := ui.MoveDone(*band)
	canMove := !done && target.Reachable && source != ui.TargetQueued
	moveHere := t.button("Move here · Enter", 10.5, colorCyan, colorCyan, func() { p.emit(Intent{Kind: IntentMoveTo, Tile: targetTile}) })
	moveHere.GetWidget().Disabled = !canMove
	p.handles.moveHere = moveHere
	best := t.button("Best tile", 10.5, colorGoldDeep, colorGoldDeep, func() { p.emit(Intent{Kind: IntentMoveToBest}) })
	best.GetWidget().Disabled = done || len(band.MigrationCandidates) == 0
	p.handles.best = best
	split := t.button("Split · N", 10.5, colorGoldDeep, colorGoldDeep, func() { p.emit(Intent{Kind: IntentSplit}) })
	split.GetWidget().Disabled = done
	p.handles.split = split
	partner := state.InterbreedFocus
	if partner == 0 && len(band.InterbreedCandidateIDs) > 0 {
		partner = band.InterbreedCandidateIDs[0]
	}
	interbreed := t.button("Interbreed · I", 10.5, colorInterbreed, colorInterbreed, func() { p.emit(Intent{Kind: IntentInterbreed, Band: partner}) })
	interbreed.GetWidget().Disabled = done || len(band.InterbreedCandidateIDs) == 0
	p.handles.interbreed = interbreed
	buttons.AddChild(moveHere, best, split, interbreed)
	body.AddChild(buttons)
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
	hint := "Arrows move a cursor instead of the pointer · Esc clears it · staying put is fine"
	switch {
	case band.Species == gameapi.ArchaicHominin:
		hint = "Computer controlled · no player actions"
	case source == ui.TargetNone:
		hint = "Hover or click an outlined tile · arrows move a cursor · Esc clears"
	}
	body.AddChild(t.label(hint, 8.5, colorDim))
	return body
}
