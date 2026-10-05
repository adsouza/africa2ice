package hud

import (
	"bytes"
	"image"
	"image/color"
	"testing"

	"github.com/adsouza/africa2ice/pkg/gameapi"
	"github.com/adsouza/africa2ice/pkg/ui"
	"github.com/hajimehoshi/ebiten/v2"
)

func TestWorkforceCampAnimationIsClippedAndPreservesControls(t *testing.T) {
	for _, scale := range []float64{1, 2} {
		panel := New()
		state := testState(testFrame(2), scale)
		state.OpenRow = ui.RowWorkforce
		panel.Update(state)
		screen := ebiten.NewImage(int(1280*scale), int(720*scale))
		screen.Fill(color.NRGBA{R: 10, G: 20, B: 30, A: 255})
		panel.Draw(screen)
		panel.handles.panelMiddle.ScrollTop = 1
		panel.Draw(screen)
		clip := panel.workforceCampClip()
		if clip.Empty() || clip.Min.Y < panel.handles.workforce.apply.GetWidget().Rect.Max.Y || clip.Max.X > panel.theme.px(panelX+panelWidth-panelPadding) {
			t.Fatalf("scale %g: camp did not fit below workforce controls: %v", scale, clip)
		}
		slider, builds, key := panel.handles.workforce.sliders[0], panel.builds, panel.PresentationKey()
		before := make([]byte, screen.Bounds().Dx()*screen.Bounds().Dy()*4)
		screen.ReadPixels(before)
		for range 12 {
			if intents := panel.Update(state); len(intents) != 0 {
				t.Fatalf("animation emitted intents: %+v", intents)
			}
			panel.DrawAnimations(screen)
		}
		after := make([]byte, len(before))
		screen.ReadPixels(after)
		changed := false
		campRect := panel.handles.workforceCamp.GetWidget().Rect
		staticSky := image.Rect(campRect.Min.X, campRect.Min.Y, campRect.Max.X, campRect.Min.Y+panel.theme.px(52)).Intersect(clip)
		for y := range screen.Bounds().Dy() {
			for x := range screen.Bounds().Dx() {
				offset := (y*screen.Bounds().Dx() + x) * 4
				if !bytes.Equal(before[offset:offset+4], after[offset:offset+4]) {
					if image.Pt(x, y).In(staticSky) {
						t.Fatalf("scale %g: animation changed the miniature's static sky at (%d,%d)", scale, x, y)
					}
					changed = true
					if !image.Pt(x, y).In(clip) {
						t.Fatalf("scale %g: animation overwrote chrome/map pixel (%d,%d)", scale, x, y)
					}
				}
			}
		}
		if !changed || panel.PresentationKey() != key || panel.builds != builds || panel.handles.workforce.sliders[0] != slider {
			t.Fatal("camp did not animate independently of map cache and workforce widgets")
		}
		state.Workforce.AllocationBP[0] += 100
		state.Workforce.Dirty, state.Workforce.Valid = true, false
		panel.Update(state)
		if panel.handles.workforce.sliders[0] != slider || slider.Current != 36 || !panel.handles.workforce.apply.GetWidget().Disabled {
			t.Fatal("miniature interfered with an in-place workforce edit")
		}
		state.Overlay.ReducedMotion = true
		panel.Update(state)
		panel.Draw(screen)
		panel.handles.panelMiddle.ScrollTop = 1
		panel.Draw(screen)
		screen.ReadPixels(before)
		for range 12 {
			panel.Update(state)
			panel.DrawAnimations(screen)
		}
		screen.ReadPixels(after)
		if !bytes.Equal(before, after) {
			t.Fatal("reduced motion did not freeze the miniature")
		}
		screen.Deallocate()
	}
}

func TestWorkforceCampHidesAndYieldsToOverlays(t *testing.T) {
	panel := New()
	state := testState(testFrame(2), 1)
	panel.Update(state)
	if panel.handles.workforceCamp != nil {
		t.Fatal("miniature appeared while Workforce was closed")
	}
	state.OpenRow = ui.RowWorkforce
	panel.Update(state)
	screen := ebiten.NewImage(1280, 720)
	panel.Draw(screen)
	panel.handles.panelMiddle.ScrollTop = 1
	panel.Draw(screen)
	if !panel.workforceCampVisible() {
		t.Fatal("miniature missing from open Workforce row")
	}
	state.SelectedBand = 2
	state.Frame.Bands[1].Species = gameapi.ArchaicHominin
	state.Workforce.Visible = false
	panel.Update(state)
	panel.Draw(screen)
	if panel.handles.workforceCamp == nil {
		t.Fatal("read-only Workforce row did not show its band's camp")
	}
	for _, scene := range []ui.SceneID{ui.SceneSettings, ui.SceneCamp} {
		state.Overlay.Scene = scene
		panel.Update(state)
		panel.Draw(screen)
		if panel.workforceCampVisible() {
			t.Fatal("miniature animation did not yield to an overlay")
		}
	}
	state.Overlay.Scene, state.OpenRow = ui.SceneGameplay, ui.RowResearch
	panel.Update(state)
	if panel.handles.workforceCamp != nil {
		t.Fatal("miniature remained after closing Workforce")
	}
}
