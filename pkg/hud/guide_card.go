package hud

import (
	"github.com/adsouza/africa2ice/pkg/ui"
	"github.com/ebitenui/ebitenui/widget"
)

// buildGuideCard is the dismissible first-turn guide (spec §7). It sits in the
// column's slack above the checklist and never blocks input.
func (p *Panel) buildGuideCard(state State) widget.PreferredSizeLocateableWidget {
	if !state.Guide.Visible() {
		return nil
	}
	t := p.theme
	card := t.column(5, t.insets(8, 12, 12, 8), bordered(colorGuide, colorGold, t.px(1)), stretch())
	head := t.rowOf(6, stretch())
	head.AddChild(t.label(state.Guide.Title(), 9, colorGold))
	current, total := state.Guide.Progress()
	bar := t.rowOf(2)
	for step := 1; step <= total; step++ {
		fill := colorPanelEdge
		if step <= current {
			fill = colorGold
		}
		bar.AddChild(widget.NewContainer(widget.ContainerOpts.BackgroundImage(solid(fill)), widget.ContainerOpts.WidgetOpts(widget.WidgetOpts.MinSize(t.px(18), t.px(4)))))
	}
	head.AddChild(bar)
	x := t.button("×", 11, colorGold, colorGold, func() { p.emit(Intent{Kind: IntentGuideDismiss}) })
	x.GetWidget().LayoutData = widget.RowLayoutData{Position: widget.RowLayoutPositionEnd}
	p.handles.guideX = x
	head.AddChild(x)
	card.AddChild(head)
	card.AddChild(t.wrapped(state.Guide.Body(), 10, colorTitle, panelWidth-2*panelPadding-30))
	if state.Guide.Step < ui.GuideClosing {
		// U+25B8 (▸) is not in the bundled Go Regular font; U+25BA (►) is.
		next := t.button("Next ►", 10, colorGold, colorGold, func() { p.emit(Intent{Kind: IntentGuideNext}) })
		p.handles.guideNext = next
		card.AddChild(next)
	}
	return card
}
