package render

import (
	"fmt"
	"math"

	"github.com/adsouza/africa2ice/pkg/gameapi"
)

const (
	TerrainGridWidth    = 96
	TerrainGridHeight   = 64
	TerrainChunkSize    = 32
	TerrainChunkColumns = TerrainGridWidth / TerrainChunkSize
	TerrainChunkRows    = TerrainGridHeight / TerrainChunkSize
	TerrainChunkCount   = TerrainChunkColumns * TerrainChunkRows
	MaxTrianglesPerPart = 21_000
)

type TerrainDetailMode uint8

const (
	TerrainDetailNormal TerrainDetailMode = iota
	TerrainDetailLow
)

// TerrainChunkPlan is the bounded, renderer-independent topology consumed by
// the eventual Tetra3D mesh builder. Every triangle carries its owning tile so
// top, wall, and chunk-boundary picking share one lookup.
type TerrainChunkPlan struct {
	Index           int
	MinX            int
	MinY            int
	MaxX            int
	MaxY            int
	TopTriangles    int
	WallTriangles   int
	TriangleTileIDs []gameapi.TileID
}

func (chunk TerrainChunkPlan) TriangleCount() int {
	return chunk.TopTriangles + chunk.WallTriangles
}

func (chunk TerrainChunkPlan) TileForTriangle(triangle int) (gameapi.TileID, bool) {
	if triangle < 0 || triangle >= len(chunk.TriangleTileIDs) {
		return 0, false
	}
	return chunk.TriangleTileIDs[triangle], true
}

// BuildTerrainChunkPlans partitions the full projected grid into the locked
// six 32×32 chunks. Normal detail assigns every lower-neighbour wall to the
// higher tile's chunk, including walls that cross a chunk boundary. Low detail
// retains the same top lookup and omits every wall.
func BuildTerrainChunkPlans(tiles []gameapi.Tile, detail TerrainDetailMode) ([]TerrainChunkPlan, error) {
	if detail != TerrainDetailNormal && detail != TerrainDetailLow {
		return nil, fmt.Errorf("invalid terrain detail mode %d", detail)
	}
	if len(tiles) != TerrainGridWidth*TerrainGridHeight {
		return nil, fmt.Errorf("terrain requires %d tiles, got %d", TerrainGridWidth*TerrainGridHeight, len(tiles))
	}
	grid := make([]*gameapi.Tile, len(tiles))
	for index := range tiles {
		tile := &tiles[index]
		if tile.X < 0 || tile.X >= TerrainGridWidth || tile.Y < 0 || tile.Y >= TerrainGridHeight {
			return nil, fmt.Errorf("tile %d has out-of-range coordinate (%d,%d)", tile.ID, tile.X, tile.Y)
		}
		if math.IsNaN(tile.ElevationKm) || math.IsInf(tile.ElevationKm, 0) || tile.ElevationKm < 0 || tile.ElevationKm > 3 {
			return nil, fmt.Errorf("tile %d has invalid elevation %v", tile.ID, tile.ElevationKm)
		}
		gridIndex := tile.Y*TerrainGridWidth + tile.X
		if grid[gridIndex] != nil {
			return nil, fmt.Errorf("duplicate terrain coordinate (%d,%d)", tile.X, tile.Y)
		}
		grid[gridIndex] = tile
	}

	chunks := make([]TerrainChunkPlan, TerrainChunkCount)
	for row := 0; row < TerrainChunkRows; row++ {
		for column := 0; column < TerrainChunkColumns; column++ {
			index := row*TerrainChunkColumns + column
			chunks[index] = TerrainChunkPlan{
				Index: index,
				MinX:  column * TerrainChunkSize, MinY: row * TerrainChunkSize,
				MaxX: (column + 1) * TerrainChunkSize, MaxY: (row + 1) * TerrainChunkSize,
				TriangleTileIDs: make([]gameapi.TileID, 0, 6_500),
			}
		}
	}

	directions := [...]struct{ dx, dy int }{{dx: -1}, {dx: 1}, {dy: -1}, {dy: 1}}
	for y := 0; y < TerrainGridHeight; y++ {
		for x := 0; x < TerrainGridWidth; x++ {
			tile := grid[y*TerrainGridWidth+x]
			if tile == nil {
				return nil, fmt.Errorf("missing terrain coordinate (%d,%d)", x, y)
			}
			chunkIndex := (y/TerrainChunkSize)*TerrainChunkColumns + x/TerrainChunkSize
			chunk := &chunks[chunkIndex]
			chunk.TopTriangles += 2
			chunk.TriangleTileIDs = append(chunk.TriangleTileIDs, tile.ID, tile.ID)
			if detail == TerrainDetailLow {
				continue
			}
			for _, direction := range directions {
				neighborX, neighborY := x+direction.dx, y+direction.dy
				if neighborX < 0 || neighborX >= TerrainGridWidth || neighborY < 0 || neighborY >= TerrainGridHeight {
					continue
				}
				neighbor := grid[neighborY*TerrainGridWidth+neighborX]
				if neighbor == nil {
					return nil, fmt.Errorf("missing terrain coordinate (%d,%d)", neighborX, neighborY)
				}
				if tile.ElevationKm <= neighbor.ElevationKm {
					continue
				}
				chunk.WallTriangles += 2
				chunk.TriangleTileIDs = append(chunk.TriangleTileIDs, tile.ID, tile.ID)
			}
		}
	}
	for _, chunk := range chunks {
		if chunk.TriangleCount() > MaxTrianglesPerPart {
			return nil, fmt.Errorf("terrain chunk %d has %d triangles, limit %d", chunk.Index, chunk.TriangleCount(), MaxTrianglesPerPart)
		}
	}
	return chunks, nil
}
