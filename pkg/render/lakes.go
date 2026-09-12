package render

import (
	"image/color"

	"github.com/adsouza/africa2ice/pkg/gameapi"
	ebitenvector "github.com/hajimehoshi/ebiten/v2/vector"
)

// Paint at the camera's resolution, not into the 8px terrain cache: thin lake
// shapes must stay crisp when focusing on a band or using a high-DPI display.
func (scene *MapScene) drawLakes(canvas logicalCanvas, geometry MapGeometry, frame *gameapi.Frame) {
	for _, tile := range frame.Tiles {
		if !tile.Explored || !tile.Land || len(tile.Lakes) == 0 {
			continue
		}
		left := geometry.OriginX + float32(tile.X)*geometry.Cell
		top := geometry.OriginY + float32(tile.Y)*geometry.Cell
		for _, lake := range tile.Lakes {
			if len(lake.Points) < 3 {
				continue
			}
			var path ebitenvector.Path
			for i, p := range lake.Points {
				x := (left + float32(p.X)*geometry.Cell) * canvas.scale
				y := (top + float32(p.Y)*geometry.Cell) * canvas.scale
				if i == 0 {
					path.MoveTo(x, y)
				} else {
					path.LineTo(x, y)
				}
			}
			path.Close()
			options := &ebitenvector.DrawPathOptions{AntiAlias: true}
			// Brighter than offshore water so a lake remains legible inside a small
			// land cell, even against the mountain and woodland palette.
			options.ColorScale.ScaleWithColor(color.RGBA{R: 66, G: 160, B: 192, A: 255})
			ebitenvector.FillPath(canvas.image, &path, nil, options)
		}
	}
}
