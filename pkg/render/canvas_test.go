package render

import (
	"image/color"
	"math"
	"testing"

	"github.com/hajimehoshi/ebiten/v2"
)

// TestFillPieDrawsOnlyItsOwnWedge is the guard that a pie actually splits.
// Filling one wedge and then the other must leave both visible: a FillPie that
// ignored its angles and filled the whole disc would pass a single-colour
// check, because the second call would simply paint over the first.
func TestFillPieDrawsOnlyItsOwnWedge(t *testing.T) {
	image := ebiten.NewImage(64, 64)
	defer image.Deallocate()
	canvas := newLogicalCanvas(image, 1)
	red := color.RGBA{R: 255, A: 255}
	green := color.RGBA{G: 255, A: 255}
	// Both wedges start on the vertical axis, so the divider is the line
	// x = 32 and sample points 10px either side of it are far from any
	// antialiased edge.
	vector.FillPie(canvas, 32, 32, 20, -math.Pi/2, math.Pi, red, true)
	vector.FillPie(canvas, 32, 32, 20, math.Pi/2, math.Pi, green, true)

	if got := image.At(42, 32); got != color.RGBA(red) {
		t.Errorf("right wedge at (42, 32) = %v, want %v", got, red)
	}
	if got := image.At(22, 32); got != color.RGBA(green) {
		t.Errorf("left wedge at (22, 32) = %v, want %v", got, green)
	}
	if _, _, _, alpha := image.At(32, 2).RGBA(); alpha != 0 {
		t.Errorf("point beyond the radius has alpha %d, want 0", alpha)
	}
}

// TestFillPieScalesLogicalGeometry keeps the pie inside logicalCanvas's
// contract: callers pass DIP-authored geometry and the canvas is the only
// place that multiplies by the physical scale.
func TestFillPieScalesLogicalGeometry(t *testing.T) {
	image := ebiten.NewImage(128, 128)
	defer image.Deallocate()
	canvas := newLogicalCanvas(image, 2)
	red := color.RGBA{R: 255, A: 255}
	vector.FillPie(canvas, 32, 32, 20, -math.Pi/2, math.Pi, red, true)

	// Logical centre (32, 32) radius 20 becomes physical centre (64, 64)
	// radius 40, so a point 20px right of the physical centre is inside the
	// wedge while the same offset from the logical centre is not.
	if got := image.At(84, 64); got != color.RGBA(red) {
		t.Errorf("scaled wedge at (84, 64) = %v, want %v", got, red)
	}
	if _, _, _, alpha := image.At(42, 32).RGBA(); alpha != 0 {
		t.Errorf("unscaled centre offset has alpha %d, want 0", alpha)
	}
}
