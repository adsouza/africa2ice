package hud

import (
	"github.com/adsouza/africa2ice/pkg/render"
	"github.com/ebitenui/ebitenui/widget"
)

// buildEndScene adds the New Campaign button over the render-drawn terminal
// scene; the text itself stays in pkg/render. Its rect derives from the
// dialog's own exported geometry (render.EndSceneX/Width) rather than a
// screen-centred literal, so it stays centred on the dialog if that
// rectangle ever moves again.
func (p *Panel) buildEndScene(state State) widget.PreferredSizeLocateableWidget {
	if !state.Ending.Visible {
		return nil
	}
	t := p.theme
	holder := widget.NewContainer(widget.ContainerOpts.Layout(widget.NewAnchorLayout()),
		widget.ContainerOpts.WidgetOpts(widget.WidgetOpts.LayoutData(p.rect(render.EndSceneX+(render.EndSceneWidth-300)/2, 494, 300, 46))))
	button := t.button("New Campaign · N", 15, colorGoldDeep, colorBlack, func() { p.emit(Intent{Kind: IntentNewCampaign}) })
	button.GetWidget().LayoutData = widget.AnchorLayoutData{StretchHorizontal: true, StretchVertical: true}
	p.handles.newCampaign = button
	holder.AddChild(button)
	return holder
}
