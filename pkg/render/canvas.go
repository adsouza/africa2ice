package render

import (
	"image/color"

	"github.com/hajimehoshi/ebiten/v2"
	ebitenvector "github.com/hajimehoshi/ebiten/v2/vector"
)

// logicalCanvas draws DIP-authored presentation geometry directly into a
// physical-resolution image. Keeping the transform here prevents individual
// widgets from mixing logical positions with render pixels.
type logicalCanvas struct {
	image *ebiten.Image
	scale float32
}

func newLogicalCanvas(image *ebiten.Image, scale float64) logicalCanvas {
	if scale <= 0 {
		scale = 1
	}
	return logicalCanvas{image: image, scale: float32(scale)}
}

// scaledVector mirrors the small vector API used by the renderer, multiplying
// coordinates and stroke widths before Ebitengine rasterizes them.
type scaledVector struct{}

var vector scaledVector

func (scaledVector) FillRect(destination logicalCanvas, x, y, width, height float32, fillColor color.Color, antialias bool) {
	scale := destination.scale
	ebitenvector.FillRect(destination.image, x*scale, y*scale, width*scale, height*scale, fillColor, antialias)
}

func (scaledVector) StrokeRect(destination logicalCanvas, x, y, width, height, strokeWidth float32, strokeColor color.Color, antialias bool) {
	scale := destination.scale
	ebitenvector.StrokeRect(destination.image, x*scale, y*scale, width*scale, height*scale, strokeWidth*scale, strokeColor, antialias)
}

func (scaledVector) FillCircle(destination logicalCanvas, cx, cy, radius float32, fillColor color.Color, antialias bool) {
	scale := destination.scale
	ebitenvector.FillCircle(destination.image, cx*scale, cy*scale, radius*scale, fillColor, antialias)
}

func (scaledVector) StrokeCircle(destination logicalCanvas, cx, cy, radius, strokeWidth float32, strokeColor color.Color, antialias bool) {
	scale := destination.scale
	ebitenvector.StrokeCircle(destination.image, cx*scale, cy*scale, radius*scale, strokeWidth*scale, strokeColor, antialias)
}

func (scaledVector) StrokeLine(destination logicalCanvas, x1, y1, x2, y2, strokeWidth float32, strokeColor color.Color, antialias bool) {
	scale := destination.scale
	ebitenvector.StrokeLine(destination.image, x1*scale, y1*scale, x2*scale, y2*scale, strokeWidth*scale, strokeColor, antialias)
}
