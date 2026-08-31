package render

import (
	"fmt"
	"image/color"
	"math"

	"github.com/adsouza/africa2ice/pkg/gameapi"
	"github.com/hajimehoshi/ebiten/v2"
	"github.com/solarlune/tetra3d"
)

const (
	terrain3DWidth  = mapPixelWidth
	terrain3DHeight = mapPixelHeight
)

type terrainScreenPoint struct {
	x, y, depth float32
}

type terrainScreenTile struct {
	id       gameapi.TileID
	explored bool
	corners  [4]terrainScreenPoint
	center   terrainScreenPoint
}

// TerrainScene3D owns the retained Tetra3D terrain graph. The simulation only
// supplies immutable projected tiles; camera, meshes, triangle lookup, and
// screen-space picking remain presentation state.
type TerrainScene3D struct {
	scene            *tetra3d.Scene
	camera           *tetra3d.Camera
	models           []*tetra3d.Model
	material         *tetra3d.Material
	directionalLight *tetra3d.DirectionalLight
	plans            []TerrainChunkPlan
	tiles            []gameapi.Tile
	screenTiles      []terrainScreenTile
	triangleTileIDs  [][]gameapi.TileID
	detail           TerrainDetailMode
	terrainRevision  uint64
	rebuilds         uint64
	orbit            OrbitCamera
}

func NewTerrainScene3D() *TerrainScene3D { return &TerrainScene3D{} }

func (terrain *TerrainScene3D) Rebuild(frame *gameapi.Frame, detail TerrainDetailMode) error {
	if frame == nil {
		return fmt.Errorf("terrain frame is nil")
	}
	plans, err := BuildTerrainChunkPlans(frame.Tiles, detail)
	if err != nil {
		return err
	}
	grid := make([]*gameapi.Tile, len(frame.Tiles))
	for index := range frame.Tiles {
		tile := &frame.Tiles[index]
		grid[tile.Y*TerrainGridWidth+tile.X] = tile
	}

	scene := tetra3d.NewScene("Africa 2 Ice terrain")
	scene.World.LightingOn = true
	material := tetra3d.NewMaterial("terrain")
	material.UseTexture = false
	material.Shadeless = detail == TerrainDetailLow
	material.BackfaceCulling = false
	var directionalLight *tetra3d.DirectionalLight
	if detail == TerrainDetailNormal {
		grade := EpochGrade(frame.Climate.AridityIndex)
		directionalLight = tetra3d.NewDirectionalLight(
			"terrain-sun",
			float32(grade.DirectionalLight.R)/255,
			float32(grade.DirectionalLight.G)/255,
			float32(grade.DirectionalLight.B)/255,
			float32(grade.DirectionalLightIntensity),
		)
		directionalLight.Rotate(1, 0, 0, -0.9)
		directionalLight.Rotate(0, 1, 0, -0.55)
		scene.World.AmbientLight.SetEnergy(float32(grade.AmbientLevel))
		scene.Root.AddChildren(directionalLight)
	}
	models := make([]*tetra3d.Model, 0, len(plans))
	triangleTileIDs := make([][]gameapi.TileID, len(plans))
	for _, plan := range plans {
		mesh := tetra3d.NewMesh(fmt.Sprintf("terrain-chunk-%d", plan.Index))
		part := mesh.AddMeshPart(material)
		vertices := make([]tetra3d.VertexInfo, 0, plan.TriangleCount()*3)
		indices := make([]int, 0, plan.TriangleCount()*3)
		tileIDs := make([]gameapi.TileID, 0, plan.TriangleCount())
		addQuad := func(points [4]tetra3d.Vector3, rgba color.RGBA, tileID gameapi.TileID, shade float32) {
			base := len(vertices)
			vertexColor := tetra3d.NewColor4(
				float32(rgba.R)/255*shade,
				float32(rgba.G)/255*shade,
				float32(rgba.B)/255*shade,
				float32(rgba.A)/255,
			)
			for _, point := range points {
				vertex := tetra3d.NewVertex(point.X, point.Y, point.Z, 0, 0)
				vertex.Colors = append(vertex.Colors, vertexColor)
				vertices = append(vertices, vertex)
			}
			indices = append(indices, base, base+1, base+2, base+3, base+1, base)
			tileIDs = append(tileIDs, tileID, tileID)
		}
		for y := plan.MinY; y < plan.MaxY; y++ {
			for x := plan.MinX; x < plan.MaxX; x++ {
				tile := grid[y*TerrainGridWidth+x]
				height := terrainTileHeight(*tile)
				rgba := tileColorForRender(*tile, frame.Climate.AridityIndex)
				worldX, worldZ := terrainWorldXZ(x, y)
				addQuad([4]tetra3d.Vector3{
					tetra3d.NewVector3(worldX, height, worldZ),
					tetra3d.NewVector3(worldX+1, height, worldZ+1),
					tetra3d.NewVector3(worldX+1, height, worldZ),
					tetra3d.NewVector3(worldX, height, worldZ+1),
				}, rgba, tile.ID, 1)
				if detail == TerrainDetailLow {
					continue
				}
				for _, direction := range [...]struct{ dx, dy int }{{dx: -1}, {dx: 1}, {dy: -1}, {dy: 1}} {
					neighborX, neighborY := x+direction.dx, y+direction.dy
					if neighborX < 0 || neighborX >= TerrainGridWidth || neighborY < 0 || neighborY >= TerrainGridHeight {
						continue
					}
					neighbor := grid[neighborY*TerrainGridWidth+neighborX]
					neighborHeight := terrainTileHeight(*neighbor)
					if tile.ElevationKm <= neighbor.ElevationKm {
						continue
					}
					addQuad(terrainWallQuad(worldX, worldZ, height, neighborHeight, direction.dx, direction.dy), rgba, tile.ID, 0.68)
				}
			}
		}
		if len(tileIDs) != len(plan.TriangleTileIDs) {
			return fmt.Errorf("terrain chunk %d built %d triangle IDs, planned %d", plan.Index, len(tileIDs), len(plan.TriangleTileIDs))
		}
		mesh.AddVertices(vertices...)
		part.AddTriangles(indices...)
		mesh.VertexActiveColorChannel = 0
		mesh.UpdateBounds()
		mesh.AutoNormal()
		model := tetra3d.NewModel(fmt.Sprintf("terrain-chunk-%d", plan.Index), mesh)
		scene.Root.AddChildren(model)
		models = append(models, model)
		triangleTileIDs[plan.Index] = tileIDs
	}

	camera := tetra3d.NewCamera("terrain-camera", terrain3DWidth, terrain3DHeight)
	camera.SetPerspective(false)
	terrain.orbit.ensureInitialized()
	terrain.orbit.Apply(camera)
	scene.Root.AddChildren(camera)

	terrain.scene = scene
	terrain.camera = camera
	terrain.models = models
	terrain.material = material
	terrain.directionalLight = directionalLight
	terrain.plans = plans
	terrain.tiles = frame.Tiles
	terrain.triangleTileIDs = triangleTileIDs
	terrain.detail = detail
	terrain.terrainRevision = frame.TerrainRevision
	terrain.rebuilds++
	terrain.screenTiles = projectTerrainTiles(camera, frame.Tiles)
	return nil
}

func (terrain *TerrainScene3D) AdjustCamera(deltaAzimuth, deltaElevation, zoom, panX, panZ float32) bool {
	if terrain == nil || terrain.camera == nil {
		return false
	}
	changed := terrain.orbit.Rotate(deltaAzimuth, deltaElevation)
	changed = terrain.orbit.Zoom(zoom) || changed
	changed = terrain.orbit.Pan(panX, panZ) || changed
	if !changed {
		return false
	}
	terrain.orbit.Apply(terrain.camera)
	terrain.screenTiles = projectTerrainTiles(terrain.camera, terrain.tiles)
	return true
}

func (terrain *TerrainScene3D) Draw(target *ebiten.Image) {
	if terrain == nil || terrain.scene == nil || terrain.camera == nil || target == nil {
		return
	}
	terrain.camera.Clear()
	terrain.camera.RenderScene(terrain.scene)
	target.DrawImage(terrain.camera.ColorTexture(), nil)
}

func (terrain *TerrainScene3D) TilePoint(tile gameapi.TileID) (float32, float32, bool) {
	if terrain == nil || int(tile) >= len(terrain.screenTiles) {
		return 0, 0, false
	}
	point := terrain.screenTiles[tile].center
	return mapOriginX + point.x, mapOriginY + point.y, true
}

func (terrain *TerrainScene3D) PickTile(x, y int) (gameapi.TileID, bool) {
	if terrain == nil || terrain.camera == nil {
		return 0, false
	}
	localX, localY := float32(x-mapOriginX), float32(y-mapOriginY)
	var selected gameapi.TileID
	selectedDepth := float32(math.Inf(1))
	found := false
	for _, tile := range terrain.screenTiles {
		if !tile.explored || !screenQuadContains(tile.corners, localX, localY) {
			continue
		}
		if !found || tile.center.depth < selectedDepth {
			selected, selectedDepth, found = tile.id, tile.center.depth, true
		}
	}
	return selected, found
}

func terrainTileHeight(tile gameapi.Tile) float32 {
	if !tile.Land {
		return -0.18
	}
	return 0.08 + float32(tile.ElevationKm)*2.4
}

func terrainWorldXZ(x, y int) (float32, float32) {
	return float32(x) - TerrainGridWidth/2, float32(y) - TerrainGridHeight/2
}

func terrainWallQuad(x, z, high, low float32, dx, dy int) [4]tetra3d.Vector3 {
	switch {
	case dx < 0:
		return [4]tetra3d.Vector3{tetra3d.NewVector3(x, high, z), tetra3d.NewVector3(x, low, z+1), tetra3d.NewVector3(x, low, z), tetra3d.NewVector3(x, high, z+1)}
	case dx > 0:
		return [4]tetra3d.Vector3{tetra3d.NewVector3(x+1, high, z+1), tetra3d.NewVector3(x+1, low, z), tetra3d.NewVector3(x+1, low, z+1), tetra3d.NewVector3(x+1, high, z)}
	case dy < 0:
		return [4]tetra3d.Vector3{tetra3d.NewVector3(x+1, high, z), tetra3d.NewVector3(x, low, z), tetra3d.NewVector3(x+1, low, z), tetra3d.NewVector3(x, high, z)}
	default:
		return [4]tetra3d.Vector3{tetra3d.NewVector3(x, high, z+1), tetra3d.NewVector3(x+1, low, z+1), tetra3d.NewVector3(x, low, z+1), tetra3d.NewVector3(x+1, high, z+1)}
	}
}

func projectTerrainTiles(camera *tetra3d.Camera, tiles []gameapi.Tile) []terrainScreenTile {
	projected := make([]terrainScreenTile, len(tiles))
	for _, tile := range tiles {
		height := terrainTileHeight(tile) + 0.02
		x, z := terrainWorldXZ(tile.X, tile.Y)
		world := [4]tetra3d.Vector3{
			tetra3d.NewVector3(x, height, z), tetra3d.NewVector3(x+1, height, z),
			tetra3d.NewVector3(x+1, height, z+1), tetra3d.NewVector3(x, height, z+1),
		}
		entry := terrainScreenTile{id: tile.ID, explored: tile.Explored}
		for index, point := range world {
			screen := camera.WorldToScreenPixels(point)
			entry.corners[index] = terrainScreenPoint{x: screen.X, y: screen.Y, depth: screen.Z}
		}
		center := camera.WorldToScreenPixels(tetra3d.NewVector3(x+0.5, height, z+0.5))
		entry.center = terrainScreenPoint{x: center.X, y: center.Y, depth: center.Z}
		projected[tile.ID] = entry
	}
	return projected
}

func screenQuadContains(corners [4]terrainScreenPoint, x, y float32) bool {
	return screenTriangleContains(corners[0], corners[1], corners[2], x, y) || screenTriangleContains(corners[0], corners[2], corners[3], x, y)
}

func screenTriangleContains(a, b, c terrainScreenPoint, x, y float32) bool {
	sign := func(first, second terrainScreenPoint) float32 {
		return (x-second.x)*(first.y-second.y) - (first.x-second.x)*(y-second.y)
	}
	d1, d2, d3 := sign(a, b), sign(b, c), sign(c, a)
	hasNegative := d1 < 0 || d2 < 0 || d3 < 0
	hasPositive := d1 > 0 || d2 > 0 || d3 > 0
	return !hasNegative || !hasPositive
}
