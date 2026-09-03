package hud

import (
	"fmt"

	"github.com/adsouza/africa2ice/pkg/gameapi"
	"github.com/ebitenui/ebitenui/widget"
)

var traitShortNames = [gameapi.HeritableTraitCount]string{"Cold", "Altitude", "Immune", "Arid", "Pigment", "Fat"}

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
	label := "details ▼"
	if state.DetailsOpen {
		label = "details ▲"
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
	column := t.column(4, t.insets(6, 10, 10, 8), solid(colorRow), stretch())
	food := "Food last turn: unavailable"
	if report := band.LastFoodReport; report.Turn > 0 {
		food = fmt.Sprintf("Turn %d food: need %.1f · ate %.1f · short %.1f (%.0f%%)", report.Turn, report.RequiredFU, report.ConsumedFU(), report.DeficitFU, report.DeficitFraction()*100)
	}
	column.AddChild(t.label(food, 9.5, colorText))
	deaths := "Deaths last turn: unavailable"
	if band.LastOutcomeReport.Turn > 0 {
		m := band.LastMortality
		deaths = fmt.Sprintf("Deaths: starvation %.2f · seasonal %.2f · chronic %.2f · macro %.2f · acute %.2f", m.Starvation, m.Seasonal, m.Chronic, m.Macro, m.Acute)
	}
	column.AddChild(t.label(deaths, 9.5, colorText))
	column.AddChild(t.label(fmt.Sprintf("Stored food %.1f FU", band.StoredFood), 9.5, colorText))

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
		widget.ContainerOpts.Layout(widget.NewGridLayout(widget.GridLayoutOpts.Columns(3), widget.GridLayoutOpts.Spacing(t.px(4), t.px(4)))),
		widget.ContainerOpts.WidgetOpts(stretch()),
	)
	for trait := gameapi.HeritableTrait(0); trait < gameapi.HeritableTraitCount; trait++ {
		focus := trait
		cell := t.button(fmt.Sprintf("%s %.3f @ %s", traitShortNames[trait], band.HeritableState[trait], pressures[trait]), 8.5, colorPanelEdge, colorText,
			func() { p.emit(Intent{Kind: IntentFocusTrait, Trait: focus}) })
		p.handles.traits[trait] = cell
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
		column.AddChild(t.wrapped(partners, 9, colorInterbreed, panelWidth-2*panelPadding-20))
	}
	return column
}
