package hud

import (
	"fmt"
	"image/color"

	"github.com/adsouza/africa2ice/pkg/gameapi"
	"github.com/ebitenui/ebitenui/widget"
)

var traitShortNames = [gameapi.HeritableTraitCount]string{"Cold", "Altitude", "Immune", "Arid", "Pigment", "Fat"}

// traitCellColors marks the variant whose Field Note is currently displayed.
func traitCellColors(focused bool) (border, text color.RGBA, borderPx float64) {
	if focused {
		return colorGold, colorGold, 2
	}
	return colorPanelEdge, colorText, 1
}

// detailsTextWidth bounds every text line in buildDetails to the panel's
// inner width, so the food, deaths, stored-food, and interbreeding lines
// wrap instead of spilling past the 352 DIP column (spec §4 item 4).
const detailsTextWidth = panelWidth - 2*panelPadding - 24

// buildBandLine is the one-line band identity with the details toggle
// (spec §4 item 3).
func (p *Panel) buildBandLine(state State, band *gameapi.Band) widget.PreferredSizeLocateableWidget {
	t := p.theme
	row := t.rowOf(8, stretch())
	if band == nil {
		row.AddChild(t.label("No band selected", 13, colorText))
		return row
	}
	identity := fmt.Sprintf("Band %d", band.ID)
	detail := fmt.Sprintf("· Pop %d", band.Population)
	if report := band.LastOutcomeReport; report.Turn > 0 {
		if delta := int64(report.EndingPopulation) - int64(report.StartingPopulation); delta != 0 {
			detail += fmt.Sprintf(" (%+d)", delta)
		}
	}
	detail += fmt.Sprintf(" · Health %.0f%%", band.Health*100)
	if band.Species == gameapi.ArchaicHominin {
		detail += " · Computer controlled · read only"
	}
	row.AddChild(t.label(identity, 13, colorText))
	detailLabel := t.label(detail, 10.5, colorDim)
	p.handles.bandDetail = detailLabel
	row.AddChild(detailLabel)
	label := "details ▼ · D"
	if state.DetailsOpen {
		label = "details ▲ · D"
	}
	toggle := t.button(label, 10, colorGoldDeep, colorGoldDeep, func() { p.emit(Intent{Kind: IntentToggleDetails}) })
	toggle.GetWidget().LayoutData = widget.RowLayoutData{Position: widget.RowLayoutPositionEnd}
	p.handles.details = toggle
	row.AddChild(toggle)
	return row
}

// buildDetails is the expanded disclosure (spec §4 item 4): food last turn,
// deaths last turn, heritable variants as focus buttons, interbreeding partners.
func (p *Panel) buildDetails(state State, band *gameapi.Band) widget.PreferredSizeLocateableWidget {
	t := p.theme
	column := t.column(4, t.insets(6, 10, 10, 8), t.solid(colorRow), stretch())
	p.handles.detailsBody = column

	foodHeading, foodLine := "FOOD LAST TURN", "unavailable"
	if report := band.LastFoodReport; report.Turn > 0 {
		foodHeading = fmt.Sprintf("FOOD LAST TURN (T%d)", report.Turn)
		foodLine = fmt.Sprintf("need %.0f · ate %.0f · short %.0f (%.0f%%)", report.RequiredFU, report.ConsumedFU(), report.DeficitFU, report.DeficitFraction()*100)
	}
	column.AddChild(t.label(foodHeading, 8.5, colorDim))
	column.AddChild(t.wrapped(foodLine, 9.5, colorText, detailsTextWidth))

	deathsLine := "unavailable"
	if band.LastOutcomeReport.Turn > 0 {
		m := band.LastMortality
		deathsLine = fmt.Sprintf("starvation %.2f · seasonal %.2f · chronic %.2f · macro %.2f · acute %.2f", m.Starvation, m.Seasonal, m.Chronic, m.Macro, m.Acute)
	}
	column.AddChild(t.label("DEATHS LAST TURN", 8.5, colorDim))
	column.AddChild(t.wrapped(deathsLine, 9.5, colorText, detailsTextWidth))

	column.AddChild(t.wrapped(fmt.Sprintf("Stored food %.1f FU", band.StoredFood), 9.5, colorText, detailsTextWidth))

	column.AddChild(t.label("HERITABLE VARIANTS · click one for its Field Note · G cycles", 8.5, colorDim))
	var tile gameapi.Tile
	if int(band.TileID) < len(state.Frame.Tiles) {
		tile = state.Frame.Tiles[band.TileID]
	}
	pressures := [gameapi.HeritableTraitCount]string{
		fmt.Sprintf("%.0f °C", tile.LocalTemperatureC), fmt.Sprintf("%.1f km", tile.ElevationKm), tile.Biome.String(),
		fmt.Sprintf("moisture %.2f", tile.BaseMoisture), fmt.Sprintf("%.0f° lat", tile.Latitude), "diet",
	}
	grid := widget.NewContainer(
		widget.ContainerOpts.Layout(widget.NewGridLayout(
			widget.GridLayoutOpts.Columns(2),
			widget.GridLayoutOpts.Spacing(t.px(4), t.px(4)),
			widget.GridLayoutOpts.Stretch([]bool{true, true}, []bool{false, false, false}),
		)),
		widget.ContainerOpts.WidgetOpts(stretch()),
	)
	for trait := gameapi.HeritableTrait(0); trait < gameapi.HeritableTraitCount; trait++ {
		focus := trait
		focused := state.Note.HasTrait && state.Note.Trait == trait
		border, textColor, borderPx := traitCellColors(focused)
		width := t.px(borderPx)
		images := &widget.ButtonImage{
			Idle:    t.bordered(colorButtonIdle, border, width),
			Hover:   t.bordered(colorButtonHover, border, width),
			Pressed: t.bordered(colorButtonDown, border, width),
		}
		cell := widget.NewButton(
			widget.ButtonOpts.Image(images),
			widget.ButtonOpts.Text(fmt.Sprintf("%s %.2f · %s", traitShortNames[trait], band.HeritableState[trait], pressures[trait]), t.face(8.5), t.buttonText(textColor)),
			widget.ButtonOpts.TextPadding(t.insets(3, 8, 8, 3)),
			widget.ButtonOpts.ClickedHandler(func(*widget.ButtonClickedEventArgs) { p.emit(Intent{Kind: IntentFocusTrait, Trait: focus}) }),
			widget.ButtonOpts.WidgetOpts(widget.WidgetOpts.CursorHovered("pointer")),
		)
		p.handles.traits[trait] = cell
		if focused {
			p.handles.traitFocused = cell
		}
		grid.AddChild(cell)
	}
	column.AddChild(grid)

	if len(band.InterbreedCandidateIDs) > 0 || band.HasInterbreedTarget {
		partners := "Interbreeding partners:"
		for _, candidate := range band.InterbreedCandidateIDs {
			partners += fmt.Sprintf(" B%d", candidate)
		}
		if band.HasInterbreedTarget {
			partners = fmt.Sprintf("Interbreeding accepted with B%d · gene flow resolves at end of turn", band.InterbreedTargetID)
		}
		column.AddChild(t.wrapped(partners, 9, colorInterbreed, detailsTextWidth))
	}
	return column
}
