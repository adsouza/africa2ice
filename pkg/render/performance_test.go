package render

import (
	"testing"

	"github.com/adsouza/africa2ice/pkg/gameapi"
	"github.com/hajimehoshi/ebiten/v2"
)

// haloBenchmarkSpacing tiles the grid with anchors exactly far enough apart
// that their ring-3 neighbourhoods meet without overlapping: a tile within
// haloRingCount Chebyshev steps of an anchor forms a (2*haloRingCount+1)-square,
// and squares of that size laid on that pitch cover the plane with no gap. A
// wider pitch leaves rows that escape every ring -- spacing 8 stranded every
// fourth row and column, reaching only 68% of the grid while claiming to
// maximise the fringe.
const haloBenchmarkSpacing = 2*haloRingCount + 1

// worstCaseHaloFrame maximises fringe area: a diagonal lattice of explored
// tiles spaced, via haloBenchmarkSpacing, so their ring-3 neighbourhoods tile
// the grid edge to edge -- 5,940 of the 6,144 tiles (96.7%) land inside some
// ring. The shipped performance fixture explores all 6,144 tiles and
// therefore has no fringe at all, so it cannot measure this feature in
// either direction.
func worstCaseHaloFrame() *gameapi.Frame {
	frame := &gameapi.Frame{TerrainRevision: 1, Tiles: make([]gameapi.Tile, TerrainGridWidth*TerrainGridHeight)}
	for y := range TerrainGridHeight {
		for x := range TerrainGridWidth {
			id := gameapi.TileID(y*TerrainGridWidth + x)
			frame.Tiles[id] = gameapi.Tile{
				ID: id, X: x, Y: y, Land: x%7 != 0, Biome: gameapi.Biome(int(id) % int(gameapi.BiomeCount)),
				Explored: x%haloBenchmarkSpacing == 0 && y%haloBenchmarkSpacing == 0,
			}
		}
	}
	return frame
}

func BenchmarkMapDrawWithHalo(b *testing.B) {
	frame := worstCaseHaloFrame()
	fringe := 0
	for _, distance := range haloDistances(frame) {
		if distance >= 1 && distance <= haloRingCount {
			fringe++
		}
	}
	b.Logf("fringe tiles: %d of %d", fringe, len(frame.Tiles))
	screen := ebiten.NewImage(1280, 720)
	scene := NewMapScene()
	scene.Update()
	scene.Draw(screen, frame, 0, MigrationPreview{}, "", EndScene{}, false)
	b.ResetTimer()
	for b.Loop() {
		for range shimmerTickStride {
			scene.Update()
		}
		if !scene.Draw(screen, frame, 0, MigrationPreview{}, "", EndScene{}, false) {
			b.Fatal("Draw skipped a frame the shimmer should have advanced")
		}
	}
}
