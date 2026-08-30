package render

import (
	"image/color"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/solarlune/tetra3d"
)

// SkeletonScene is the retained cross-platform graphics seam used while the
// production terrain renderer is built around immutable frames.
type SkeletonScene struct {
	scene  *tetra3d.Scene
	camera *tetra3d.Camera
	model  *tetra3d.Model
}

func NewSkeletonScene(width, height int) *SkeletonScene {
	scene := tetra3d.NewScene("Africa 2 Ice walking skeleton")
	scene.World.LightingOn = false
	model := tetra3d.NewModel("World", tetra3d.NewCubeMesh(5, 0.35, 3))
	model.Color = tetra3d.NewColor4(0.45, 0.57, 0.30, 1)
	scene.Root.AddChildren(model)
	camera := tetra3d.NewCamera("Camera", width, height)
	camera.Move(0, 4, 7)
	camera.Rotate(1, 0, 0, -0.42)
	scene.Root.AddChildren(camera)
	return &SkeletonScene{scene: scene, camera: camera, model: model}
}

func (s *SkeletonScene) Update() {
	if s == nil || s.model == nil {
		return
	}
	s.model.Rotate(0, 1, 0, 0.002)
}

func (s *SkeletonScene) Draw(screen *ebiten.Image) {
	if s == nil || screen == nil {
		return
	}
	screenColor := color.RGBA{R: 20, G: 29, B: 38, A: 255}
	screen.Fill(screenColor)
	s.camera.Clear()
	s.camera.RenderScene(s.scene)
	screen.DrawImage(s.camera.ColorTexture(), nil)
}
