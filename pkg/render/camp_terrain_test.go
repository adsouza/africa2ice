package render

import (
	"bytes"
	"crypto/sha256"
	"encoding/json"
	"testing"

	"github.com/adsouza/africa2ice/pkg/gameapi"
	"github.com/hajimehoshi/ebiten/v2"
)

func TestCampTerrainBiomesAreDistinctAtBothIllustrationSizes(t *testing.T) {
	for _, width := range []int{316, 1280} {
		seen := map[[32]byte]gameapi.Biome{}
		var scene CampScene
		for biome := gameapi.Biome(0); biome < gameapi.BiomeCount; biome++ {
			frame := campTerrainFrame(gameapi.Tile{Biome: biome, LocalTemperatureC: 18, ElevationKm: .5, VegetationIndex: .5})
			before, _ := json.Marshal(frame)
			art, changed := scene.Illustration(frame, 7, true, width)
			if !changed || art.Bounds().Dx() != width {
				t.Fatalf("width %d biome %s: new terrain was not composed", width, biome)
			}
			pixels := make([]byte, art.Bounds().Dx()*art.Bounds().Dy()*4)
			scene.background.ReadPixels(pixels)
			digest := sha256.Sum256(pixels)
			if previous, duplicate := seen[digest]; duplicate {
				t.Fatalf("width %d: %s and %s have identical scenery", width, previous, biome)
			}
			seen[digest] = biome
			if _, changed := scene.Illustration(frame, 7, true, width); changed {
				t.Fatal("unchanged terrain bypassed the illustration cache")
			}
			after, _ := json.Marshal(frame)
			if !bytes.Equal(before, after) {
				t.Fatal("drawing terrain mutated the accepted frame")
			}
		}
	}
}

func TestCampTerrainReadsLocalConditionsFromNewFrames(t *testing.T) {
	cases := []struct {
		name     string
		from, to gameapi.Tile
	}{
		{"freezing", gameapi.Tile{Biome: gameapi.GlacialTundra, LocalTemperatureC: 8}, gameapi.Tile{Biome: gameapi.GlacialTundra, LocalTemperatureC: -8}},
		{"elevation", gameapi.Tile{Biome: gameapi.MountainousHighlands, LocalTemperatureC: 12, ElevationKm: .5}, gameapi.Tile{Biome: gameapi.MountainousHighlands, LocalTemperatureC: 12, ElevationKm: 3}},
		{"vegetation", gameapi.Tile{Biome: gameapi.RiverineWoodland, LocalTemperatureC: 20, VegetationIndex: .1}, gameapi.Tile{Biome: gameapi.RiverineWoodland, LocalTemperatureC: 20, VegetationIndex: .9}},
		{"nearby lake", gameapi.Tile{Biome: gameapi.Savanna, LocalTemperatureC: 20}, gameapi.Tile{Biome: gameapi.Savanna, LocalTemperatureC: 20, NearbyLake: "Lake Victoria"}},
	}
	for _, test := range cases {
		t.Run(test.name, func(t *testing.T) {
			var scene CampScene
			art, _ := scene.Illustration(campTerrainFrame(test.from), 7, true, 316)
			before := make([]byte, art.Bounds().Dx()*art.Bounds().Dy()*4)
			scene.background.ReadPixels(before)
			_, changed := scene.Illustration(campTerrainFrame(test.to), 7, true, 316)
			after := make([]byte, len(before))
			scene.background.ReadPixels(after)
			if !changed || bytes.Equal(before, after) {
				t.Fatal("updated local conditions did not replace the cached landscape")
			}
		})
	}
}

func TestCampTerrainPartialAnimationMatchesFullPaintInEveryBiome(t *testing.T) {
	screen := ebiten.NewImage(1280, 720)
	for biome := gameapi.Biome(0); biome < gameapi.BiomeCount; biome++ {
		frame := campTerrainFrame(gameapi.Tile{Biome: biome, LocalTemperatureC: -2, ElevationKm: 2, VegetationIndex: .6, NaturalShelter: .5, NearbyLake: "Lake"})
		var scene CampScene
		scene.Draw(screen, frame, 7, false, 0)
		for range 12 {
			scene.Update(false)
		}
		if scene.Draw(screen, frame, 7, false, 0) != CampMotion {
			t.Fatalf("%s did not animate", biome)
		}
		partial := make([]byte, 1280*720*4)
		screen.ReadPixels(partial)
		scene.Invalidate()
		scene.Draw(screen, frame, 7, false, 0)
		full := make([]byte, len(partial))
		screen.ReadPixels(full)
		if !bytes.Equal(partial, full) {
			t.Fatalf("%s partial repaint left stale scenery", biome)
		}
	}
}

func campTerrainFrame(tile gameapi.Tile) *gameapi.Frame {
	return &gameapi.Frame{Bands: []gameapi.Band{{ID: 7, Population: 40, Health: .8}}, Tiles: []gameapi.Tile{tile}}
}
