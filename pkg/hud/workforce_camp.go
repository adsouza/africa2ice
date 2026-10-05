package hud

import (
	"image"

	"github.com/adsouza/africa2ice/pkg/ui"
	"github.com/ebitenui/ebitenui/widget"
	"github.com/hajimehoshi/ebiten/v2"
)

func (p *Panel) buildWorkforceCamp(state State) *widget.Graphic {
	art, _ := p.workforceCamp.Illustration(state.Frame, state.SelectedBand, state.Overlay.ReducedMotion, p.theme.px(panelWidth-2*panelPadding))
	graphic := widget.NewGraphic(widget.GraphicOpts.Image(art), widget.GraphicOpts.WidgetOpts(stretch()))
	p.handles.workforceCamp = graphic
	return graphic
}

func (p *Panel) workforceCampClip() image.Rectangle {
	if p.handles.workforceCamp == nil || p.handles.panelMiddle == nil {
		return image.Rectangle{}
	}
	return p.handles.workforceCamp.GetWidget().Rect.Intersect(p.handles.panelMiddle.ViewRect())
}

func (p *Panel) workforceCampVisible() bool {
	return p.last.Overlay.Scene == ui.SceneGameplay && !p.last.ShortcutsOpen && !p.last.BandListOpen && !p.last.Ending.Visible && !p.tooltipShown && !p.workforceCampClip().Empty()
}

func (p *Panel) refreshWorkforceCamp() bool {
	if p.handles.workforceCamp == nil {
		return false
	}
	art, changed := p.workforceCamp.Illustration(p.last.Frame, p.last.SelectedBand, p.last.Overlay.ReducedMotion, p.theme.px(panelWidth-2*panelPadding))
	p.handles.workforceCamp.Image = art
	return changed
}

// DrawAnimations updates only the visible part of the opaque camp image on a
// map cache hit. Full UI draws already prepare it in Draw. Modals and tooltips
// pause it so an image refresh never paints over a window or hover explanation.
func (p *Panel) DrawAnimations(screen *ebiten.Image) {
	if !p.workforceCampVisible() || !p.refreshWorkforceCamp() {
		return
	}
	clip := p.workforceCampClip().Intersect(screen.Bounds())
	if !clip.Empty() {
		p.handles.workforceCamp.Render(screen.SubImage(clip).(*ebiten.Image))
	}
}
