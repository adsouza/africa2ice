package hud

import (
	"github.com/adsouza/africa2ice/pkg/ui"
	"github.com/ebitenui/ebitenui/widget"
)

// buildEndTurn is the button whose label explains its own state (spec §5.3).
func (p *Panel) buildEndTurn(state State) widget.PreferredSizeLocateableWidget {
	t := p.theme
	gate := state.EndTurn
	fill, textColor := colorGoldDeep, colorBlack
	if gate.Soft {
		fill = colorAmber
	}
	force := gate.Enabled && !gate.Soft && state.Frame != nil && ui.BandsNeedingMove(state.Frame.Bands) > 0
	button := widget.NewButton(
		widget.ButtonOpts.Image(&widget.ButtonImage{Idle: t.solid(fill), Hover: t.solid(colorGold), Pressed: t.solid(colorGoldDeep), Disabled: t.bordered(colorRow, colorDisabled, t.px(1))}),
		widget.ButtonOpts.Text(gate.Label, t.face(13), &widget.ButtonTextColor{Idle: textColor, Hover: textColor, Pressed: textColor, Disabled: colorDisabled}),
		widget.ButtonOpts.TextPadding(t.insets(8, 12, 12, 8)),
		widget.ButtonOpts.ClickedHandler(func(*widget.ButtonClickedEventArgs) { p.emit(Intent{Kind: IntentEndTurn, Force: force}) }),
		widget.ButtonOpts.WidgetOpts(stretch(), widget.WidgetOpts.CursorHovered("pointer")),
	)
	button.GetWidget().Disabled = !gate.Enabled
	p.handles.endTurn = button
	return button
}
