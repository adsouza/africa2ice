package hud

import (
	"image"
	"testing"

	"github.com/ebitenui/ebitenui/widget"
	"github.com/hajimehoshi/ebiten/v2"
)

func TestFixedLayoutPlacesChildrenAtTheirOwnRectangles(t *testing.T) {
	root := widget.NewContainer(widget.ContainerOpts.Layout(fixedLayout{}))
	first := widget.NewContainer(widget.ContainerOpts.WidgetOpts(widget.WidgetOpts.LayoutData(fixedRect(image.Rect(10, 20, 110, 220)))))
	second := widget.NewContainer(widget.ContainerOpts.WidgetOpts(widget.WidgetOpts.LayoutData(fixedRect(image.Rect(500, 0, 900, 50)))))
	root.AddChild(first, second)
	root.SetLocation(image.Rect(0, 0, 1280, 720))
	root.RequestRelayout()
	root.Render(ebiten.NewImage(1280, 720)) // performs the pending relayout
	if got := first.GetWidget().Rect; got != image.Rect(10, 20, 110, 220) {
		t.Fatalf("first child rect = %v", got)
	}
	if got := second.GetWidget().Rect; got != image.Rect(500, 0, 900, 50) {
		t.Fatalf("second child rect = %v", got)
	}
}
