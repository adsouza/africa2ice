package render

import (
	"bytes"
	"encoding/json"
	"testing"

	"github.com/adsouza/africa2ice/pkg/gameapi"
	"github.com/hajimehoshi/ebiten/v2"
)

func TestCampAnimationCacheReducedMotionAndReentry(t *testing.T) {
	frame := &gameapi.Frame{Bands: []gameapi.Band{{ID: 7, Population: 40, Health: .8}}, Tiles: []gameapi.Tile{{Biome: gameapi.Savanna, LocalTemperatureC: 25}}}
	before, _ := json.Marshal(frame)
	screen := ebiten.NewImage(1280, 720)
	var scene CampScene
	if scene.Draw(screen, frame, 7, false, 0) != CampFull || scene.Draw(screen, frame, 7, false, 0) != CampSkipped {
		t.Fatal("initial camp did not paint exactly once")
	}
	first := make([]byte, 1280*720*4)
	screen.ReadPixels(first)
	background := scene.background
	for range 12 {
		scene.Update(false)
	}
	if scene.Draw(screen, frame, 7, false, 0) != CampMotion || scene.background != background {
		t.Fatal("animation did not repaint with cached scenery")
	}
	animated := make([]byte, len(first))
	screen.ReadPixels(animated)
	if bytes.Equal(first, animated) {
		t.Fatal("the camp illustration did not animate")
	}
	scene.Invalidate()
	scene.Draw(screen, frame, 7, false, 0)
	full := make([]byte, len(first))
	screen.ReadPixels(full)
	if !bytes.Equal(animated, full) {
		t.Fatal("partial animation repaint did not match a complete repaint")
	}
	if scene.Draw(screen, frame, 7, true, 0) == CampSkipped {
		t.Fatal("enabling reduced motion did not restore the fixed pose")
	}
	frozen := make([]byte, len(first))
	screen.ReadPixels(frozen)
	if !bytes.Equal(first, frozen) {
		t.Fatal("reduced motion did not reproduce the original pose")
	}
	for range 60 {
		scene.Update(true)
	}
	if scene.Draw(screen, frame, 7, true, 0) != CampSkipped {
		t.Fatal("reduced motion repainted an unchanged camp")
	}
	if scene.Draw(screen, frame, 7, true, 1) != CampFull {
		t.Fatal("return button hover did not repaint a frozen camp")
	}
	scene.Invalidate()
	if scene.Draw(screen, frame, 7, true, 1) != CampFull {
		t.Fatal("reentering camp left the map on screen")
	}
	after, _ := json.Marshal(frame)
	if !bytes.Equal(before, after) {
		t.Fatal("camp renderer mutated the accepted frame")
	}
}

func TestCampResizesAndReplacesTheSelectedBandsLandscape(t *testing.T) {
	frame := &gameapi.Frame{
		Bands: []gameapi.Band{{ID: 1, Population: 6}, {ID: 2, TileID: 1, Population: 50}},
		Tiles: []gameapi.Tile{{Biome: gameapi.Savanna, LocalTemperatureC: 25}, {Biome: gameapi.GlacialTundra, LocalTemperatureC: -10}},
	}
	var scene CampScene
	screen := ebiten.NewImage(1280, 720)
	scene.Draw(screen, frame, 1, true, 0)
	first := make([]byte, 1280*720*4)
	scene.background.ReadPixels(first)
	if scene.Draw(screen, frame, 2, true, 0) != CampFull {
		t.Fatal("selecting another band did not repaint")
	}
	second := make([]byte, len(first))
	scene.background.ReadPixels(second)
	if bytes.Equal(first, second) {
		t.Fatal("the cold band's landscape matched the warm camp")
	}
	large := ebiten.NewImage(2560, 1600)
	if scene.Draw(large, frame, 2, true, 0) != CampFull || scene.image.Bounds().Dx() != 1280 || scene.image.Bounds().Dy() != 720 {
		t.Fatal("camp did not reuse its illustration at high DPI")
	}
	if r, g, b, _ := large.At(10, 10).RGBA(); r != 5*257 || g != 9*257 || b != 14*257 {
		t.Fatal("camp art escaped its letterbox")
	}
	if large.At(1280, 1000) == large.At(10, 10) {
		t.Fatal("camp did not scale into the high DPI content area")
	}
}

func TestCampFirelightVisiblyFlickersOnStationaryGround(t *testing.T) {
	frame := &gameapi.Frame{Bands: []gameapi.Band{{ID: 7, Population: 40, Health: .8}}, Tiles: []gameapi.Tile{{Biome: gameapi.Savanna, LocalTemperatureC: 25}}}
	screen := ebiten.NewImage(1280, 720)
	var scene CampScene
	dimmest, brightest := uint32(255), uint32(0)
	for range 41 {
		scene.Draw(screen, frame, 7, false, 0)
		// This patch is below the fire and clear of stones, people and smoke.
		// Its brightness must change even without any moving shapes crossing it.
		red, _, _, _ := screen.At(640, 501).RGBA()
		red >>= 8
		dimmest, brightest = min(dimmest, red), max(brightest, red)
		for range 3 {
			scene.Update(false)
		}
	}
	if brightest-dimmest < 25 {
		t.Fatalf("firelight brightness barely changed: red ranged from %d to %d", dimmest, brightest)
	}
}
