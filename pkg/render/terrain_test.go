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
