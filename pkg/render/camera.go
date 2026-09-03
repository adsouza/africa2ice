package render

import "github.com/adsouza/africa2ice/pkg/gameapi"

// CameraMode is one of the two map views (spec §6). There is no free pan.
type CameraMode uint8

const (
	CameraOverview CameraMode = iota
	CameraFocus
)

const (
	FocusScale            = 3.0   // 24 px cells
	CameraTransitionTicks = 15    // 250 ms at 60 TPS
	mapAreaWidth          = 864.0 // map rectangle x 20-884
	mapAreaHeight         = 626.0 // map rectangle y 74-700
)

// Camera is UI-local presentation state owned by the application. Progress is
// the blend toward Focus so a mode change animates from wherever it was.
type Camera struct {
	Mode       CameraMode
	CenterTile gameapi.TileID
	Progress   float64
}

func (camera Camera) target() float64 {
	if camera.Mode == CameraFocus {
		return 1
	}
	return 0
}

// Step advances the blend one tick toward the mode. The comparisons snap to
// the target within a small epsilon rather than using a plain min/max clamp:
// accumulating 1.0/CameraTransitionTicks in float64 lands one ULP short of
// the target on the final tick (0.9999999999999999, not 1), which a plain
// min/max never catches and Settled would then never report true.
func (camera Camera) Step() Camera {
	const epsilon = 1e-9
	step := 1.0 / CameraTransitionTicks
	target := camera.target()
	switch {
	case camera.Progress < target:
		camera.Progress += step
		if target-camera.Progress < epsilon {
			camera.Progress = target
		}
	case camera.Progress > target:
		camera.Progress -= step
		if camera.Progress-target < epsilon {
			camera.Progress = target
		}
	}
	return camera
}

func (camera Camera) Settled() bool { return camera.Progress == camera.target() }

// MapGeometry is the resolved cell size and grid origin for one frame.
type MapGeometry struct {
	OriginX       float32
	OriginY       float32
	Cell          float32
	visibleHeight float32 // TileAt's bottom bound; 0 means the full map area
}

func (geometry MapGeometry) visibleBottom() float32 {
	if geometry.visibleHeight <= 0 {
		return mapOriginY + float32(mapAreaHeight)
	}
	return mapOriginY + geometry.visibleHeight
}

// CameraGeometry blends the overview grid with a 3x view centred on the
// camera's tile, clamped so the grid never shows space beyond its edges inside
// the visible map area (x 20-884, y 74 to 74+visibleHeight).
func CameraGeometry(camera Camera, frame *gameapi.Frame, visibleHeight float64) MapGeometry {
	overview := MapGeometry{OriginX: mapOriginX, OriginY: mapOriginY, Cell: mapTileSize, visibleHeight: float32(visibleHeight)}
	if camera.Progress <= 0 || frame == nil || int(camera.CenterTile) >= len(frame.Tiles) {
		return overview
	}
	cell := mapTileSize * FocusScale
	tile := frame.Tiles[camera.CenterTile]
	centerX := (float64(tile.X) + 0.5) * cell
	centerY := (float64(tile.Y) + 0.5) * cell
	originX := mapOriginX + mapAreaWidth/2 - centerX
	originY := mapOriginY + visibleHeight/2 - centerY
	originX = max(min(originX, mapOriginX), mapOriginX+mapAreaWidth-float64(TerrainGridWidth)*cell)
	originY = max(min(originY, mapOriginY), mapOriginY+visibleHeight-float64(TerrainGridHeight)*cell)
	blend := camera.Progress
	return MapGeometry{
		OriginX: float32(lerp(float64(overview.OriginX), originX, blend)),
		OriginY: float32(lerp(float64(overview.OriginY), originY, blend)),
		Cell:    float32(lerp(float64(overview.Cell), cell, blend)),

		visibleHeight: float32(visibleHeight),
	}
}

// TilePoint is the centre of a tile's cell under this geometry.
func (geometry MapGeometry) TilePoint(tile gameapi.Tile) (float32, float32) {
	return geometry.OriginX + (float32(tile.X)+0.5)*geometry.Cell, geometry.OriginY + (float32(tile.Y)+0.5)*geometry.Cell
}

// TileAt inverts TilePoint for a logical point inside the map area.
func (geometry MapGeometry) TileAt(x, y int) (gameapi.TileID, bool) {
	fx, fy := float32(x), float32(y)
	if fx < mapOriginX || fy < mapOriginY || fx >= mapOriginX+float32(mapAreaWidth) || fy >= geometry.visibleBottom() {
		return 0, false
	}
	gridX := int((fx - geometry.OriginX) / geometry.Cell)
	gridY := int((fy - geometry.OriginY) / geometry.Cell)
	if fx < geometry.OriginX || fy < geometry.OriginY || gridX < 0 || gridX >= TerrainGridWidth || gridY < 0 || gridY >= TerrainGridHeight {
		return 0, false
	}
	return gameapi.TileID(gridY*TerrainGridWidth + gridX), true
}

// MapTileAt is the camera-aware pick used by hover and clicks.
func MapTileAt(camera Camera, frame *gameapi.Frame, visibleHeight float64, x, y int) (gameapi.TileID, bool) {
	return CameraGeometry(camera, frame, visibleHeight).TileAt(x, y)
}
