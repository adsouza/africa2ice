package hud

import (
	"image"

	"github.com/adsouza/africa2ice/pkg/render"
	"github.com/ebitenui/ebitenui/widget"
)

// buildEndScene adds the New Campaign button over the render-drawn terminal
// scene; the text itself stays in pkg/render. Its rect derives from the
// dialog's own exported geometry (render.EndSceneX/Width) rather than a
// screen-centred literal, so it stays centred on the dialog if that
// rectangle ever moves again.
//
// drawEndScene dims the entire screen, so the dialog is conceptually modal;
// this builds it as an actual modal widget.Window, the way buildOverlay
// builds the title/menu/storage/settings scenes, rather than a plain button
// added to the root column. A user-reported bug (reproduced in
// TestEndSceneClickReachesNewCampaignOverTheDrawer) showed why a plain
// button is not enough: with the Field Notes drawer expanded, its
// ScrollContainer elevates its own content to an input layer covering x
// 20–884, y 382–700 with BlockLower set — which swallows clicks at the
// button's rect (302–602, 494–540) regardless of widget add-order, since
// ebitenui's input layers are resolved by the layer stack, not by tree
// position. A modal window's own input layer is FullScreen with BlockLower
// set, so it always wins over any other elevated layer beneath it — which
// also means it correctly stops clicks from reaching the chips, checklist
// rows and drawer behind a finished campaign.
func (p *Panel) buildEndScene(state State) {
	if !state.Ending.Visible {
		return
	}
	t := p.theme
	rect := image.Rectangle(p.rect(render.EndSceneX+(render.EndSceneWidth-300)/2, 494, 300, 46))
	holder := widget.NewContainer(widget.ContainerOpts.Layout(widget.NewAnchorLayout()))
	// Same gold-filled-primary-button treatment as End turn (end_turn.go):
	// black text over solid gold reads as the dialog's one action, instead
	// of the near-invisible black-on-dark-grey the border style produced.
	button := widget.NewButton(
		widget.ButtonOpts.Image(&widget.ButtonImage{Idle: t.solid(colorGoldDeep), Hover: t.solid(colorGold), Pressed: t.solid(colorGoldDeep)}),
		widget.ButtonOpts.Text("New Campaign · N", t.face(15), t.buttonText(colorBlack)),
		widget.ButtonOpts.TextPadding(t.insets(3, 8, 8, 3)),
		widget.ButtonOpts.ClickedHandler(func(*widget.ButtonClickedEventArgs) { p.emit(Intent{Kind: IntentNewCampaign}) }),
		widget.ButtonOpts.WidgetOpts(widget.WidgetOpts.CursorHovered("pointer")),
	)
	button.GetWidget().LayoutData = widget.AnchorLayoutData{StretchHorizontal: true, StretchVertical: true}
	p.handles.newCampaign = button
	holder.AddChild(button)
	window := widget.NewWindow(widget.WindowOpts.Contents(holder), widget.WindowOpts.Modal(), widget.WindowOpts.CloseMode(widget.NONE), widget.WindowOpts.Location(rect))
	p.ui.AddWindow(window)
	p.handles.endScene = window
}
