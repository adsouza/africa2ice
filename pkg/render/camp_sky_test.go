package render

import (
	"bytes"
	"testing"

	"github.com/adsouza/africa2ice/pkg/gameapi"
	"github.com/hajimehoshi/ebiten/v2"
)

func TestCampStarsTwinkleOnlyInTheFullSizeView(t *testing.T) {
	frame := campTerrainFrame(gameapi.Tile{Biome: gameapi.Savanna, LocalTemperatureC: 20})
	var fullScene, miniature CampScene
	screen := ebiten.NewImage(1280, 720)
	fullScene.Draw(screen, frame, 7, false, 0)
	first := make([]byte, 1280*720*4)
	screen.ReadPixels(first)
	art, _ := miniature.Illustration(frame, 7, false, 316)
	miniFirst := make([]byte, art.Bounds().Dx()*art.Bounds().Dy()*4)
	art.ReadPixels(miniFirst)
	for range 90 {
		fullScene.Update(false)
		miniature.Update(false)
	}
	fullScene.Draw(screen, frame, 7, false, 0)
	later := make([]byte, len(first))
	screen.ReadPixels(later)
	// This strip contains stars and moon, above all campfire smoke and figures.
	if bytes.Equal(first[120*1280*4:210*1280*4], later[120*1280*4:210*1280*4]) {
		t.Fatal("full-size sky did not twinkle")
	}
	art, _ = miniature.Illustration(frame, 7, false, 316)
	miniLater := make([]byte, len(miniFirst))
	art.ReadPixels(miniLater)
	if !bytes.Equal(miniFirst[:52*316*4], miniLater[:52*316*4]) || bytes.Equal(miniFirst, miniLater) {
		t.Fatal("miniature did not keep steady stars while its camp animated")
	}
	fullScene.Draw(screen, frame, 7, true, 0)
	screen.ReadPixels(later)
	if !bytes.Equal(first, later) {
		t.Fatal("reduced motion did not restore the original star field and camp pose")
	}
}

func TestCampStarsStayBehindTheMoonAndMountainPeaks(t *testing.T) {
	frame := campTerrainFrame(gameapi.Tile{Biome: gameapi.MountainousHighlands, ElevationKm: 4, LocalTemperatureC: -5})
	var scene CampScene
	screen := ebiten.NewImage(1280, 720)
	for range 90 {
		scene.Update(false)
	}
	scene.Draw(screen, frame, 7, false, 0)
	foreground, composite := make([]byte, 1280*720*4), make([]byte, 1280*720*4)
	scene.background.ReadPixels(foreground)
	screen.ReadPixels(composite)
	covered := 0
	for y := 120; y < 210; y++ {
		for x := range 1280 {
			offset := (y*1280 + x) * 4
			if foreground[offset+3] == 255 {
				covered++
				if !bytes.Equal(foreground[offset:offset+4], composite[offset:offset+4]) {
					t.Fatalf("twinkling sky painted over foreground at (%d,%d)", x, y)
				}
			}
		}
	}
	if covered < 100 {
		t.Fatal("fixture did not exercise opaque sky occlusion")
	}
}

func TestCampTwinklePartialRepaintMatchesFullAtHighDPI(t *testing.T) {
	frame := campTerrainFrame(gameapi.Tile{Biome: gameapi.RiverineWoodland, VegetationIndex: .9, LocalTemperatureC: 20})
	var scene CampScene
	screen := ebiten.NewImage(2560, 1600)
	scene.Draw(screen, frame, 7, false, 0)
	for range 90 {
		scene.Update(false)
	}
	scene.Draw(screen, frame, 7, false, 0)
	partial := make([]byte, 2560*1600*4)
	screen.ReadPixels(partial)
	scene.Invalidate()
	scene.Draw(screen, frame, 7, false, 0)
	full := make([]byte, len(partial))
	screen.ReadPixels(full)
	if !bytes.Equal(partial, full) {
		t.Fatal("high-DPI star and camp crops left stale pixels")
	}
}
