package hud

import (
	"image"

	"github.com/ebitenui/ebitenui/widget"
)

// fixedRect is the layout data a direct child of a fixedLayout container
// carries: its absolute rectangle in render pixels. The chrome has a handful
// of regions at spec-fixed DIP positions; a layout that just honours them is
// simpler than expressing the panel as nested anchors.
type fixedRect image.Rectangle

type fixedLayout struct{}

func (fixedLayout) PreferredSize(widgets []widget.PreferredSizeLocateableWidget) (int, int) {
	width, height := 0, 0
	for _, child := range widgets {
		if rect, ok := child.GetWidget().LayoutData.(fixedRect); ok {
			width = max(width, rect.Max.X)
			height = max(height, rect.Max.Y)
		}
	}
	return width, height
}

func (fixedLayout) Layout(widgets []widget.PreferredSizeLocateableWidget, _ image.Rectangle) {
	for _, child := range widgets {
		if rect, ok := child.GetWidget().LayoutData.(fixedRect); ok {
			child.SetLocation(image.Rectangle(rect))
		}
	}
}

var _ widget.Layouter = fixedLayout{}
