package hud

import (
	"fmt"

	"github.com/adsouza/africa2ice/pkg/gameapi"
	"github.com/ebitenui/ebitenui/widget"
)

var eraRanges = [gameapi.CampaignEraCount]string{"80,000–50,000 BP", "50,000–35,000 BP", "35,000–25,000 BP", "25,000–20,000 BP"}

func macroWarning(frame *gameapi.Frame) string {
	for _, episode := range frame.MacroEpisodes {
		switch {
		case episode.Current:
			return "ACTIVE · " + episode.Episode.String()
		case episode.Warned:
			return "WARNING · " + episode.Episode.String()
		}
	}
	return ""
}

// buildHeader is the title row, clock, era/season/epoch, macro warning, and
// the sapiens total (spec §4 item 1).
func (p *Panel) buildHeader(state State) widget.PreferredSizeLocateableWidget {
	frame := state.Frame
	t := p.theme
	column := t.column(2, nil, nil, stretch())
	title := t.rowOf(8, stretch())
	title.AddChild(t.label("Africa 2 Ice", 24, colorTitle))
	menu := t.button("☰ Menu · Esc", 10.5, colorGoldDeep, colorGoldDeep, func() { p.emit(Intent{Kind: IntentOpenMenu}) })
	menu.GetWidget().LayoutData = widget.RowLayoutData{Position: widget.RowLayoutPositionEnd}
	title.AddChild(menu)
	column.AddChild(title)
	column.AddChild(t.label(fmt.Sprintf("%s BP · Turn %d/400", formatThousands(frame.YearBP), frame.Turn), 15, colorText))
	era := frame.Era.String() + " era"
	if frame.Era < gameapi.CampaignEraCount {
		era += " · " + eraRanges[frame.Era]
	}
	column.AddChild(t.label(era+" · "+frame.Season.String()+" · "+frame.Climate.Epoch.String(), 10.5, colorDim))
	if warning := macroWarning(frame); warning != "" {
		column.AddChild(t.label(warning, 9.5, colorAmber))
	}
	var total uint64
	living := 0
	for _, band := range frame.Bands {
		if band.Species == gameapi.HomoSapiens && band.Population > 0 {
			total += uint64(band.Population)
			living++
		}
	}
	population := t.rowOf(6)
	population.AddChild(t.label(fmt.Sprintf("Homo sapiens %d", total), 14, colorGold))
	population.AddChild(t.label(fmt.Sprintf("· %d bands · Regions %d", living, len(frame.SapiensEstablishedRegions)), 10.5, colorDim))
	column.AddChild(population)
	return column
}

func formatThousands(value int) string {
	digits := fmt.Sprintf("%d", value)
	for index := len(digits) - 3; index > 0; index -= 3 {
		digits = digits[:index] + "," + digits[index:]
	}
	return digits
}
