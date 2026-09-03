package hud

import "github.com/ebitenui/ebitenui/widget"

// buildEndScene adds the New Campaign button over the render-drawn terminal
// scene; the text itself stays in pkg/render.
func (p *Panel) buildEndScene(state State) widget.PreferredSizeLocateableWidget {
	if !state.Ending.Visible {
		return nil
	}
	t := p.theme
	holder := widget.NewContainer(widget.ContainerOpts.Layout(widget.NewAnchorLayout()),
		widget.ContainerOpts.WidgetOpts(widget.WidgetOpts.LayoutData(p.rect(490, 494, 300, 46))))
	button := t.button("New Campaign · N", 15, colorGoldDeep, colorBlack, func() { p.emit(Intent{Kind: IntentNewCampaign}) })
	button.GetWidget().LayoutData = widget.AnchorLayoutData{StretchHorizontal: true, StretchVertical: true}
	p.handles.newCampaign = button
	holder.AddChild(button)
	return holder
}
