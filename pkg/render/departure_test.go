package render

import (
	"bytes"
	"encoding/json"
	"image"
	"testing"

	"github.com/adsouza/africa2ice/pkg/gameapi"
	"github.com/hajimehoshi/ebiten/v2"
)

func departureFrame() *gameapi.Frame {
	frame := cameraFrame()
	frame.Turn = 4
	origin := gameapi.TileID(55*TerrainGridWidth + 44)
	frame.Tiles[origin].Biome = gameapi.Savanna
	frame.Tiles[origin].LocalTemperatureC = 24
	frame.Tiles[origin].VegetationIndex = .5
	frame.Bands = []gameapi.Band{{ID: 7, TileID: origin, Population: 120, Health: .8, Species: gameapi.HomoSapiens, HasQueuedMigration: true, QueuedMigration: origin + 1}}
	return frame
}

func departurePixels(screen *ebiten.Image) []byte {
	pixels := make([]byte, screen.Bounds().Dx()*screen.Bounds().Dy()*4)
	screen.ReadPixels(pixels)
	return pixels
}

func TestDepartureCrowdMatchesCampPopulationAndStaysInTheStrip(t *testing.T) {
	for _, population := range []uint32{0, 1, 8, 12, 13, 20, 21, 1000} {
		for _, seconds := range []float64{0, 1, 3.95} {
			people := departureCrowd(population, seconds)
			if len(people) != int(min(population, 20)) {
				t.Fatalf("population %d produced %d figures", population, len(people))
			}
			poses := map[int]bool{}
			for _, person := range people {
				if poses[person.pose] {
					t.Fatal("duplicate member in the procession")
				}
				poses[person.pose] = true
				x, y, scale := person.position[0], person.position[1], person.position[2]
				if x-35*scale < 24 || x+50*scale > departureWidth-24 || y-136*scale < 24 || y+7 > departureHeight-16 {
					t.Fatalf("figure %d escapes the soft-edged strip at %.2fs: %+v", person.pose, seconds, person)
				}
			}
		}
	}
}

func TestDepartureMotionRestoresMapAndKeepsTheCache(t *testing.T) {
	for _, size := range []image.Point{{1280, 720}, {2560, 1600}, {1601, 1000}} {
		frame := departureFrame()
		before, _ := json.Marshal(frame)
		scene := NewMapScene()
		screen := ebiten.NewImage(size.X, size.Y)
		defer screen.Deallocate()
		draw := func() bool { return scene.Draw(screen, frame, 7, MigrationPreview{}, "", EndScene{}, false) }
		draw()
		baseline := departurePixels(screen)
		scene.StartDeparture(frame.Bands[0], frame.Tiles[frame.Bands[0].TileID], frame.Turn)
		draw()
		initial := departurePixels(screen)
		if bytes.Equal(initial, baseline) {
			t.Fatal("departure did not appear immediately")
		}
		paints, cachedMap := scene.Paints, scene.frameImage
		scene.Update()
		draw()
		if !bytes.Equal(initial, departurePixels(screen)) {
			t.Fatal("departure repainted before its 20 Hz phase changed")
		}
		for range 11 {
			scene.Update()
		}
		if draw() || scene.Paints != paints || scene.frameImage != cachedMap {
			t.Fatal("walking invalidated the underlying map cache")
		}
		animated := departurePixels(screen)
		if bytes.Equal(initial, animated) {
			t.Fatal("walking figures did not move")
		}
		// A complete repaint at the same tick must match the bounded one,
		// including fractional crop edges on a letterboxed high-DPI target.
		scene.frameCached = false
		draw()
		if !bytes.Equal(animated, departurePixels(screen)) {
			t.Fatal("partial departure repaint differs from a full repaint")
		}
		for range departureTicks - 13 {
			scene.Update()
		}
		if !scene.DepartureActive() || scene.departure.opacity(false) >= .01 {
			t.Fatal("departure did not soften before its four-second expiry")
		}
		scene.Update()
		draw()
		if scene.DepartureActive() || !bytes.Equal(baseline, departurePixels(screen)) {
			t.Fatal("expiry left translucent art or trails on the map")
		}
		after, _ := json.Marshal(frame)
		if !bytes.Equal(before, after) {
			t.Fatal("departure animation mutated the accepted frame")
		}
	}
}

func TestDepartureReducedMotionFreezesThenExpires(t *testing.T) {
	frame := departureFrame()
	scene := NewMapScene()
	scene.SetReducedMotion(true)
	screen := ebiten.NewImage(1280, 720)
	defer screen.Deallocate()
	draw := func() { scene.Draw(screen, frame, 7, MigrationPreview{}, "", EndScene{}, false) }
	draw()
	baseline := departurePixels(screen)
	scene.StartDeparture(frame.Bands[0], frame.Tiles[frame.Bands[0].TileID], frame.Turn)
	draw()
	frozen := departurePixels(screen)
	if bytes.Equal(frozen, baseline) {
		t.Fatal("reduced motion suppressed the departure still")
	}
	for range departureTicks - 1 {
		scene.Update()
		draw()
	}
	if !bytes.Equal(frozen, departurePixels(screen)) {
		t.Fatal("reduced-motion departure walked or faded")
	}
	scene.Update()
	draw()
	if !bytes.Equal(baseline, departurePixels(screen)) {
		t.Fatal("reduced-motion still did not clear after four seconds")
	}
}

func TestDeparturePlacementRoutesAndObscuredLayers(t *testing.T) {
	frame := departureFrame()
	scene := NewMapScene()
	screen := ebiten.NewImage(1280, 720)
	defer screen.Deallocate()
	draw := func() { scene.Draw(screen, frame, 7, MigrationPreview{}, "", EndScene{}, false) }
	draw()
	baseline := departurePixels(screen)
	scene.StartDeparture(frame.Bands[0], frame.Tiles[frame.Bands[0].TileID], frame.Turn)
	draw()
	// This arrow crosses the vignette. Its centre stays exactly the route's red.
	x, y := scene.geometry(frame).TilePoint(frame.Tiles[frame.Bands[0].TileID+1])
	r, g, b, _ := screen.At(int(x-3), int(y)).RGBA()
	if r < 180*257 || g > 100*257 || b > 100*257 {
		t.Fatalf("vignette obscured the queued route: %d %d %d", r, g, b)
	}
	scene.SetDepartureVisible(false)
	draw()
	if !bytes.Equal(baseline, departurePixels(screen)) {
		t.Fatal("hiding departure beneath a HUD overlay left pixels behind")
	}
	scene.SetDepartureVisible(true)
	draw()
	if bytes.Equal(baseline, departurePixels(screen)) {
		t.Fatal("uncovering the map did not restore a live departure")
	}
	for _, height := range []float64{286, 524, 606} {
		scene.SetCamera(Camera{}, height)
		draw()
		bounds := scene.departure.paintKey.bounds
		if bounds.Max.Y > mapOriginY+int(height)-18 || bounds.Min.Y < mapOriginY || bounds.Max.X > mapOriginX+mapAreaWidth {
			t.Fatalf("vignette escaped the map above the drawer: %v", bounds)
		}
	}
	scene.ClearDeparture()
	draw()
	scene.StartDeparture(frame.Bands[0], frame.Tiles[frame.Bands[0].TileID], frame.Turn)
	scene.Draw(screen, frame, 999, MigrationPreview{}, "", EndScene{}, false)
	if scene.DepartureActive() {
		t.Fatal("changing bands kept the old departure")
	}
	scene.StartDeparture(frame.Bands[0], frame.Tiles[frame.Bands[0].TileID], frame.Turn)
	copyFrame := *frame
	copyFrame.Turn++
	scene.Draw(screen, &copyFrame, 7, MigrationPreview{}, "", EndScene{}, false)
	if scene.DepartureActive() {
		t.Fatal("departure survived turn resolution")
	}
}

func TestDepartureLandscapeIsDaylitAndSoftEdged(t *testing.T) {
	var previous []byte
	for biome := gameapi.Biome(0); biome < gameapi.BiomeCount; biome++ {
		scene := departureScene{tile: gameapi.Tile{Biome: biome, LocalTemperatureC: 24, VegetationIndex: .5}}
		scene.prepare(0)
		pixels := departurePixels(scene.background)
		if bytes.Equal(previous, pixels) {
			t.Fatal("changing departure biome did not change scenery")
		}
		previous = pixels
		r, g, b, a := scene.background.At(360, 30).RGBA()
		if r < 140*257 || g < 190*257 || b < 210*257 || a != 65535 {
			t.Fatalf("departure sky is not daylight: %d %d %d %d", r, g, b, a)
		}
		for _, point := range []image.Point{{0, 80}, {719, 80}, {360, 0}, {360, 175}} {
			_, _, _, alpha := scene.background.At(point.X, point.Y).RGBA()
			if alpha != 0 {
				t.Fatal("departure scenery has a hard opaque edge")
			}
		}
	}
}
