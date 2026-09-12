package render

import (
	"image/color"
	"testing"

	"github.com/adsouza/africa2ice/pkg/gameapi"
	"github.com/hajimehoshi/ebiten/v2"
)

func TestLakeShapesRenderAtCameraAndDisplayScaleWithoutFogLeak(t *testing.T) {
	for _, cell := range []float32{8, 24} {
		for _, scale := range []float64{1, 2} {
			size := int(4 * cell * float32(scale))
			img := ebiten.NewImage(size, size)
			background := color.RGBA{R: 25, G: 30, B: 20, A: 255}
			img.Fill(background)
			shape := gameapi.LakeShape{Name: "Test lake", Points: []gameapi.LakePoint{{X: 0.25, Y: 0.25}, {X: 0.75, Y: 0.25}, {X: 0.75, Y: 0.75}, {X: 0.25, Y: 0.75}}}
			frame := &gameapi.Frame{Tiles: []gameapi.Tile{
				{X: 1, Y: 1, Land: true, Explored: true, Lakes: []gameapi.LakeShape{shape}},
				{X: 2, Y: 1, Land: true, Explored: false, Lakes: []gameapi.LakeShape{shape}},
			}}
			scene := &MapScene{}
			scene.drawLakes(newLogicalCanvas(img, scale), MapGeometry{Cell: cell}, frame)
			at := func(x, y float32) color.RGBA {
				return color.RGBAModel.Convert(img.At(int(x*cell*float32(scale)), int(y*cell*float32(scale)))).(color.RGBA)
			}
			if at(1.5, 1.5) == background {
				t.Fatalf("lake invisible at cell=%g scale=%g", cell, scale)
			}
			if at(1, 1) != background {
				t.Fatal("lake filled surrounding land")
			}
			if at(2.5, 1.5) != background {
				t.Fatal("lake revealed unexplored tile")
			}
			img.Deallocate()
		}
	}
}
