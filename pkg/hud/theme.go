package hud

import (
	"bytes"
	"image/color"

	"github.com/ebitenui/ebitenui/image"
	"github.com/ebitenui/ebitenui/widget"
	"github.com/hajimehoshi/ebiten/v2/text/v2"
	"golang.org/x/image/font/gofont/goregular"
)

// Chrome palette. Map colors stay in pkg/render; these are panel-only. Only
// the colors and theme helpers a builder actually calls live here today:
// Tasks 8-13 add the rest of the spec §3 palette and the button/wrapped/
// rowOf/stretch helpers alongside the first widget that needs each one, so
// the unused linter never has to flag scaffolding with no caller.
var (
	colorPanel = color.RGBA{R: 25, G: 35, B: 42, A: 238}
	colorTitle = color.RGBA{R: 239, G: 220, B: 178, A: 255}
)

// theme owns the font source and the current presentation scale. Every size
// and inset the panel uses goes through px() or face(), so a viewport change
// only needs a rebuild with a new scale.
type theme struct {
	source *text.GoTextFaceSource
	scale  float64
	faces  map[float64]*text.Face
}

func newTheme() *theme {
	source, err := text.NewGoTextFaceSource(bytes.NewReader(goregular.TTF))
	if err != nil {
		panic(err)
	}
	return &theme{source: source, scale: 1, faces: map[float64]*text.Face{}}
}

func (t *theme) setScale(scale float64) {
	if scale <= 0 {
		scale = 1
	}
	if scale != t.scale {
		t.scale = scale
		t.faces = map[float64]*text.Face{}
	}
}

// px converts a DIP length to render pixels, rounding to the nearest pixel.
func (t *theme) px(dip float64) int { return int(dip*t.scale + 0.5) }

// face returns a cached ebitenui font face for a DIP size at the current scale.
func (t *theme) face(sizeDIP float64) *text.Face {
	if face, ok := t.faces[sizeDIP]; ok {
		return face
	}
	var face text.Face = &text.GoTextFace{Source: t.source, Size: sizeDIP * t.scale}
	t.faces[sizeDIP] = &face
	return &face
}

func (t *theme) insets(top, left, right, bottom float64) *widget.Insets {
	return &widget.Insets{Top: t.px(top), Left: t.px(left), Right: t.px(right), Bottom: t.px(bottom)}
}

func solid(c color.Color) *image.NineSlice { return image.NewNineSliceColor(c) }

// label builds static text.
func (t *theme) label(value string, sizeDIP float64, textColor color.Color) *widget.Text {
	return widget.NewText(widget.TextOpts.Text(value, t.face(sizeDIP), textColor))
}

// column is a vertical row layout container with DIP spacing and padding.
func (t *theme) column(spacingDIP float64, padding *widget.Insets, background *image.NineSlice, opts ...widget.WidgetOpt) *widget.Container {
	containerOpts := []widget.ContainerOpt{
		widget.ContainerOpts.Layout(widget.NewRowLayout(
			widget.RowLayoutOpts.Direction(widget.DirectionVertical),
			widget.RowLayoutOpts.Spacing(t.px(spacingDIP)),
			widget.RowLayoutOpts.Padding(padding),
		)),
		widget.ContainerOpts.WidgetOpts(opts...),
	}
	if background != nil {
		containerOpts = append(containerOpts, widget.ContainerOpts.BackgroundImage(background))
	}
	return widget.NewContainer(containerOpts...)
}
