package hud

import (
	"fmt"

	"github.com/ebitenui/ebitenui/widget"
)

// The camp occupies the entire presentation. Only its caption and return
// control are widgets; the animated illustration belongs to pkg/render.
func (p *Panel) buildCampView(state State) {
	t := p.theme
	header := t.column(7, t.insets(0, 0, 0, 0), nil,
		widget.WidgetOpts.LayoutData(p.rect(80, 42, 1120, 100)))
	header.AddChild(t.label("A moment by the fire", 30, colorTitle))
	if band := state.selectedBand(); band != nil {
		place := ""
		if int(band.TileID) < len(state.Frame.Tiles) {
			place = " · " + state.Frame.Tiles[band.TileID].Biome.String()
		}
		header.AddChild(t.label(fmt.Sprintf("Band %d · %s · %d people · health %.0f%%%s", band.ID, band.Species, band.Population, band.Health*100, place), 12, colorDim))
	}
	p.root.AddChild(header)
	footer := t.column(10, t.insets(0, 0, 0, 0), nil,
		widget.WidgetOpts.LayoutData(p.rect(80, 620, 1120, 85)))
	footer.AddChild(t.label("Beyond the journey, there is the warmth of company.", 13, colorTitle))
	back := t.button("Return to the map · Esc / C", 12, colorGoldDeep, colorGoldDeep, func() { p.emit(Intent{Kind: IntentBack}) })
	p.handles.overlayButtons = append(p.handles.overlayButtons, back)
	footer.AddChild(back)
	p.root.AddChild(footer)
}
