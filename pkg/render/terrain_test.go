package render

import (
	"math"
	"testing"

	"github.com/adsouza/africa2ice/pkg/gameapi"
)

func TestTerrainChunkPlansPartitionTheFullGridWithStableTopLookup(t *testing.T) {
	tiles := terrainFixture(func(_, _ int) float64 { return 0 })
	chunks, err := BuildTerrainChunkPlans(tiles, TerrainDetailNormal)
	if err != nil {
		t.Fatal(err)
	}
	if len(chunks) != TerrainChunkCount {
		t.Fatalf("chunks = %d, want %d", len(chunks), TerrainChunkCount)
	}
	for index, chunk := range chunks {
		if chunk.Index != index || chunk.MaxX-chunk.MinX != TerrainChunkSize || chunk.MaxY-chunk.MinY != TerrainChunkSize {
			t.Fatalf("chunk %d bounds = %#v", index, chunk)
		}
		if chunk.TopTriangles != 2*TerrainChunkSize*TerrainChunkSize || chunk.WallTriangles != 0 || chunk.TriangleCount() != len(chunk.TriangleTileIDs) {
			t.Fatalf("flat chunk %d topology = %#v", index, chunk)
		}
		firstTile := gameapi.TileID(chunk.MinY*TerrainGridWidth + chunk.MinX)
		lastTile := gameapi.TileID((chunk.MaxY-1)*TerrainGridWidth + chunk.MaxX - 1)
		if got, ok := chunk.TileForTriangle(0); !ok || got != firstTile {
			t.Fatalf("chunk %d first triangle = (%d,%t), want tile %d", index, got, ok, firstTile)
		}
		if got, ok := chunk.TileForTriangle(chunk.TopTriangles - 1); !ok || got != lastTile {
			t.Fatalf("chunk %d last top triangle = (%d,%t), want tile %d", index, got, ok, lastTile)
		}
		if _, ok := chunk.TileForTriangle(-1); ok {
			t.Fatalf("chunk %d accepted negative triangle", index)
		}
		if _, ok := chunk.TileForTriangle(chunk.TriangleCount()); ok {
			t.Fatalf("chunk %d accepted past-end triangle", index)
		}
	}
}

func TestTerrainNormalDetailStaysBelowTheWorstCaseChunkBound(t *testing.T) {
	tiles := terrainFixture(func(x, y int) float64 {
		if (x+y)%2 == 0 {
			return 1
		}
		return 0
	})
	chunks, err := BuildTerrainChunkPlans(tiles, TerrainDetailNormal)
	if err != nil {
		t.Fatal(err)
	}
	for _, chunk := range chunks {
		if chunk.TriangleCount() > 6_144 || chunk.TriangleCount() >= MaxTrianglesPerPart {
			t.Fatalf("checkerboard chunk %d has %d triangles", chunk.Index, chunk.TriangleCount())
		}
		if chunk.WallTriangles == 0 || len(chunk.TriangleTileIDs) != chunk.TriangleCount() {
			t.Fatalf("checkerboard chunk %d lost wall topology", chunk.Index)
		}
	}
}

func TestTerrainWallsBelongToTheHigherTileAcrossChunkBoundaries(t *testing.T) {
	tiles := terrainFixture(func(x, y int) float64 {
		if x == 31 && y == 31 {
			return 1
		}
		return 0
	})
	chunks, err := BuildTerrainChunkPlans(tiles, TerrainDetailNormal)
	if err != nil {
		t.Fatal(err)
	}
	raisedTile := gameapi.TileID(31*TerrainGridWidth + 31)
	owner := chunks[0]
	if owner.WallTriangles != 8 {
		t.Fatalf("raised boundary tile walls = %d, want 8", owner.WallTriangles)
	}
	for triangle := owner.TopTriangles; triangle < owner.TriangleCount(); triangle++ {
		if tile, ok := owner.TileForTriangle(triangle); !ok || tile != raisedTile {
			t.Fatalf("wall triangle %d maps to (%d,%t), want tile %d", triangle, tile, ok, raisedTile)
		}
	}
	if chunks[1].WallTriangles != 0 || chunks[3].WallTriangles != 0 {
		t.Fatal("lower neighboring chunks claimed the higher tile's boundary walls")
	}
}

func TestTerrainLowDetailRetainsOnlyTopTriangles(t *testing.T) {
	tiles := terrainFixture(func(x, y int) float64 { return float64((x + y) % 3) })
	chunks, err := BuildTerrainChunkPlans(tiles, TerrainDetailLow)
	if err != nil {
		t.Fatal(err)
	}
	for _, chunk := range chunks {
		if chunk.WallTriangles != 0 || chunk.TopTriangles != 2*TerrainChunkSize*TerrainChunkSize || len(chunk.TriangleTileIDs) != chunk.TopTriangles {
			t.Fatalf("low-detail chunk %d = %#v", chunk.Index, chunk)
		}
	}
}

func TestTerrainChunkPlansRejectMalformedProjectedGrids(t *testing.T) {
	tiles := terrainFixture(func(_, _ int) float64 { return 0 })
	tests := []struct {
		name   string
		mutate func([]gameapi.Tile) []gameapi.Tile
	}{
		{name: "wrong length", mutate: func(values []gameapi.Tile) []gameapi.Tile { return values[:len(values)-1] }},
		{name: "duplicate coordinate", mutate: func(values []gameapi.Tile) []gameapi.Tile {
			values[1].X = values[0].X
			values[1].Y = values[0].Y
			return values
		}},
		{name: "out of range", mutate: func(values []gameapi.Tile) []gameapi.Tile { values[1].X = TerrainGridWidth; return values }},
		{name: "nan elevation", mutate: func(values []gameapi.Tile) []gameapi.Tile { values[1].ElevationKm = math.NaN(); return values }},
		{name: "over-height", mutate: func(values []gameapi.Tile) []gameapi.Tile { values[1].ElevationKm = 3.01; return values }},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			copyOfTiles := append([]gameapi.Tile(nil), tiles...)
			if _, err := BuildTerrainChunkPlans(test.mutate(copyOfTiles), TerrainDetailNormal); err == nil {
				t.Fatal("malformed grid was accepted")
			}
		})
	}
	if _, err := BuildTerrainChunkPlans(tiles, TerrainDetailMode(99)); err == nil {
		t.Fatal("invalid detail mode was accepted")
	}
}

func TestTerrainScene3DBuildsSixChunksAndPicksOnlyExploredTops(t *testing.T) {
	tiles := terrainFixture(func(x, y int) float64 {
		if x >= 40 && x < 48 && y >= 24 && y < 32 {
			return 1
		}
		return 0
	})
	for index := range tiles {
		tiles[index].Land = true
		tiles[index].Explored = true
		tiles[index].Biome = gameapi.Savanna
	}
	frame := &gameapi.Frame{TerrainRevision: 7, Tiles: tiles, Climate: gameapi.ClimateSummary{AridityIndex: 0.4}}
	terrain := NewTerrainScene3D()
	if err := terrain.Rebuild(frame, TerrainDetailNormal); err != nil {
		t.Fatal(err)
	}
	if len(terrain.models) != TerrainChunkCount || len(terrain.plans) != TerrainChunkCount || terrain.rebuilds != 1 {
		t.Fatalf("terrain graph = models %d plans %d rebuilds %d", len(terrain.models), len(terrain.plans), terrain.rebuilds)
	}
	if len(terrain.colliders) != TerrainChunkCount || len(terrain.triangleTiles) == 0 {
		t.Fatalf("terrain colliders = %d, triangle lookup = %d", len(terrain.colliders), len(terrain.triangleTiles))
	}
	if terrain.material == nil || terrain.material.Shadeless || terrain.directionalLight == nil {
		t.Fatal("normal detail did not retain the lit shared material and one directional light")
	}
	for index, plan := range terrain.plans {
		if len(terrain.triangleTileIDs[index]) != plan.TriangleCount() {
			t.Fatalf("chunk %d lookup = %d, want %d", index, len(terrain.triangleTileIDs[index]), plan.TriangleCount())
		}
	}

	tileID := gameapi.TileID(30*TerrainGridWidth + 44)
	x, y, ok := terrain.TilePoint(tileID)
	if !ok {
		t.Fatal("explored highland has no screen position")
	}
	picked, ok := terrain.PickTile(int(x), int(y))
	if !ok || picked != tileID {
		t.Fatalf("picked tile = (%d,%t), want %d", picked, ok, tileID)
	}
	terrain.exploredTiles[tileID] = false
	if picked, ok := terrain.PickTile(int(x), int(y)); ok && picked == tileID {
		t.Fatal("hidden tile remained pickable")
	}
}

func TestTerrainScene3DLowDetailOmitsEveryWall(t *testing.T) {
	tiles := terrainFixture(func(x, y int) float64 { return float64((x + y) % 2) })
	for index := range tiles {
		tiles[index].Land = true
		tiles[index].Explored = true
	}
	terrain := NewTerrainScene3D()
	if err := terrain.Rebuild(&gameapi.Frame{Tiles: tiles}, TerrainDetailLow); err != nil {
		t.Fatal(err)
	}
	if terrain.material == nil || !terrain.material.Shadeless || terrain.directionalLight != nil {
		t.Fatal("low detail retained lighting or lost its shadeless material")
	}
	for _, plan := range terrain.plans {
		if plan.WallTriangles != 0 || len(terrain.triangleTileIDs[plan.Index]) != plan.TopTriangles {
			t.Fatalf("low detail chunk %d retained walls", plan.Index)
		}
	}
}

func TestTerrainCameraMovementReprojectsWithoutRebuildingChunks(t *testing.T) {
	tiles := terrainFixture(func(_, _ int) float64 { return 0 })
	for index := range tiles {
		tiles[index].Land = true
		tiles[index].Explored = true
	}
	terrain := NewTerrainScene3D()
	if err := terrain.Rebuild(&gameapi.Frame{TerrainRevision: 4, Tiles: tiles}, TerrainDetailNormal); err != nil {
		t.Fatal(err)
	}
	tile := gameapi.TileID(20*TerrainGridWidth + 70)
	beforeX, beforeY, _ := terrain.TilePoint(tile)
	if !terrain.AdjustCamera(0.15, 0.04, 5, 2, -1) {
		t.Fatal("non-zero camera input reported no change")
	}
	afterX, afterY, _ := terrain.TilePoint(tile)
	if beforeX == afterX && beforeY == afterY {
		t.Fatal("camera movement did not reproject tile positions")
	}
	if terrain.rebuilds != 1 || terrain.orbit.revision != 3 {
		t.Fatalf("camera movement rebuilt chunks or lost revision: rebuilds=%d orbit=%d", terrain.rebuilds, terrain.orbit.revision)
	}
	picked, ok := terrain.PickTile(int(afterX), int(afterY))
	if !ok || picked != tile {
		t.Fatalf("camera-adjusted pick = (%d,%t), want %d", picked, ok, tile)
	}
}

func terrainFixture(elevation func(x, y int) float64) []gameapi.Tile {
	tiles := make([]gameapi.Tile, 0, TerrainGridWidth*TerrainGridHeight)
	for y := 0; y < TerrainGridHeight; y++ {
		for x := 0; x < TerrainGridWidth; x++ {
			id := gameapi.TileID(y*TerrainGridWidth + x)
			tiles = append(tiles, gameapi.Tile{ID: id, X: x, Y: y, ElevationKm: elevation(x, y)})
		}
	}
	return tiles
}
