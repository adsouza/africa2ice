package render

import (
	"math"
	"testing"

	"github.com/adsouza/africa2ice/pkg/gameapi"
)

func cameraFrame() *gameapi.Frame {
	frame := &gameapi.Frame{Tiles: make([]gameapi.Tile, TerrainGridWidth*TerrainGridHeight)}
	for y := 0; y < TerrainGridHeight; y++ {
		for x := 0; x < TerrainGridWidth; x++ {
			id := gameapi.TileID(y*TerrainGridWidth + x)
			frame.Tiles[id] = gameapi.Tile{ID: id, X: x, Y: y, Land: true, Explored: true}
		}
	}
	return frame
}

func TestOverviewGeometryMatchesTheLockedGrid(t *testing.T) {
	geometry := CameraGeometry(Camera{}, cameraFrame(), 626)
	if geometry.OriginX != mapOriginX || geometry.OriginY != mapOriginY || geometry.Cell != mapTileSize {
		t.Fatalf("overview geometry = %+v", geometry)
	}
	x, y := geometry.TilePoint(gameapi.Tile{X: 3, Y: 2})
	if x != mapOriginX+3*mapTileSize+mapTileSize/2 || y != mapOriginY+2*mapTileSize+mapTileSize/2 {
		t.Fatalf("overview tile point = (%.1f, %.1f)", x, y)
	}
}

func TestFocusGeometryCentersTheTileAndClampsToTheGrid(t *testing.T) {
	frame := cameraFrame()
	center := gameapi.TileID(30*TerrainGridWidth + 40)
	geometry := CameraGeometry(Camera{Mode: CameraFocus, CenterTile: center, Progress: 1}, frame, 626)
	if geometry.Cell != mapTileSize*FocusScale {
		t.Fatalf("focus cell = %.1f", geometry.Cell)
	}
	x, y := geometry.TilePoint(frame.Tiles[center])
	if math.Abs(float64(x)-(mapOriginX+864/2)) > 0.6 || math.Abs(float64(y)-(mapOriginY+626/2)) > 0.6 {
		t.Fatalf("focused tile center = (%.1f, %.1f), want the middle of the visible map area", x, y)
	}
	corner := CameraGeometry(Camera{Mode: CameraFocus, CenterTile: 0, Progress: 1}, frame, 626)
	if corner.OriginX != mapOriginX || corner.OriginY != mapOriginY {
		t.Fatalf("top-left focus should clamp to the map origin, got %+v", corner)
	}
	last := gameapi.TileID(len(frame.Tiles) - 1)
	farCorner := CameraGeometry(Camera{Mode: CameraFocus, CenterTile: last, Progress: 1}, frame, 626)
	if right := farCorner.OriginX + float32(TerrainGridWidth)*farCorner.Cell; math.Abs(float64(right)-(mapOriginX+864)) > 0.6 {
		t.Fatalf("bottom-right focus should clamp so the grid's right edge meets the map edge, got right %.1f", right)
	}
	if bottom := farCorner.OriginY + float32(TerrainGridHeight)*farCorner.Cell; math.Abs(float64(bottom)-(mapOriginY+626)) > 0.6 {
		t.Fatalf("bottom-right focus should clamp to the visible height, got bottom %.1f", bottom)
	}
}

func TestTileAtRoundTripsTilePointInBothModesAndMidTransition(t *testing.T) {
	frame := cameraFrame()
	for _, camera := range []Camera{
		{},
		{Mode: CameraFocus, CenterTile: 2000, Progress: 1},
		{Mode: CameraFocus, CenterTile: 2000, Progress: 0.4},
	} {
		geometry := CameraGeometry(camera, frame, 500)
		for _, id := range []gameapi.TileID{2000, 2001, 2000 + TerrainGridWidth} {
			x, y := geometry.TilePoint(frame.Tiles[id])
			if got, ok := geometry.TileAt(int(x), int(y)); !ok || got != id {
				t.Fatalf("camera %+v: TileAt(TilePoint(%d)) = (%d, %t)", camera, id, got, ok)
			}
		}
		if _, ok := geometry.TileAt(mapOriginX-1, mapOriginY); ok {
			t.Fatalf("camera %+v: point left of the map picked a tile", camera)
		}
		if _, ok := geometry.TileAt(mapOriginX+10, mapOriginY+501); ok {
			t.Fatalf("camera %+v: point below the visible area picked a tile", camera)
		}
	}
}

func TestCameraStepsTowardItsModeAndSettles(t *testing.T) {
	camera := Camera{Mode: CameraFocus}
	for tick := 0; tick < CameraTransitionTicks; tick++ {
		if camera.Settled() {
			t.Fatalf("settled after %d ticks", tick)
		}
		camera = camera.Step()
	}
	if !camera.Settled() || camera.Progress != 1 {
		t.Fatalf("after %d ticks = %+v", CameraTransitionTicks, camera)
	}
	camera.Mode = CameraOverview
	camera = camera.Step()
	if camera.Progress >= 1 || camera.Settled() {
		t.Fatalf("switching back did not start moving: %+v", camera)
	}
}
