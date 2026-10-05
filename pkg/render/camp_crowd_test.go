package render

import (
	"bytes"
	"testing"

	"github.com/adsouza/africa2ice/pkg/gameapi"
	"github.com/hajimehoshi/ebiten/v2"
)

func TestCampCrowdScalesWithPopulationAndBoundsExtremeBands(t *testing.T) {
	for _, test := range []struct {
		population uint32
		figures    int
	}{{0, 0}, {1, 1}, {2, 2}, {7, 7}, {8, 8}, {9, 9}, {10, 10}, {12, 12}, {19, 19}, {20, 20}, {21, 20}, {120, 20}, {^uint32(0), 20}} {
		people := campCrowd(test.population)
		if len(people) != test.figures {
			t.Fatalf("population %d: got %d figures, want %d", test.population, len(people), test.figures)
		}
		poses := map[int]bool{}
		for i, person := range people {
			if poses[person.pose] || (i > 0 && people[i-1].position[1] > person.position[1]) {
				t.Fatal("crowd duplicated a seat or painted out of depth order")
			}
			poses[person.pose] = true
		}
	}
}

func TestCampPopulationChangesReplaceBothIllustrationSizes(t *testing.T) {
	for _, width := range []int{316, 1280} {
		var scene CampScene
		var previous []byte
		for _, population := range []uint32{1, 4, 8, 12, 20, 7, 0} {
			frame := campTerrainFrame(gameapi.Tile{Biome: gameapi.Savanna, LocalTemperatureC: 20})
			frame.Bands[0].Population = population
			art, changed := scene.Illustration(frame, 7, true, width)
			pixels := make([]byte, art.Bounds().Dx()*art.Bounds().Dy()*4)
			art.ReadPixels(pixels)
			if !changed || bytes.Equal(pixels, previous) {
				t.Fatalf("width %d: population %d did not replace the gathering", width, population)
			}
			previous = pixels
		}
	}
}

func TestCampLargestCrowdAnimationStaysInsideItsRepaintAtBothDPIs(t *testing.T) {
	frame := campTerrainFrame(gameapi.Tile{Biome: gameapi.RiverineWoodland, LocalTemperatureC: 20})
	frame.Bands[0].Population = ^uint32(0)
	for _, scale := range []int{1, 2} {
		var scene CampScene
		screen := ebiten.NewImage(1280*scale, 720*scale)
		scene.Draw(screen, frame, 7, false, 0)
		for range 90 {
			scene.Update(false)
		}
		scene.Draw(screen, frame, 7, false, 0)
		partial := make([]byte, screen.Bounds().Dx()*screen.Bounds().Dy()*4)
		screen.ReadPixels(partial)
		scene.Invalidate()
		scene.Draw(screen, frame, 7, false, 0)
		full := make([]byte, len(partial))
		screen.ReadPixels(full)
		if !bytes.Equal(partial, full) {
			t.Fatalf("DPR %d: crowd animation left stale pixels outside its repaint", scale)
		}
		screen.Deallocate()
	}
}
