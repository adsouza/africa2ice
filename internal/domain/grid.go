package domain

import "math"

type Grid struct {
	tiles [TileCount]TileGeography
}

func (g *Grid) Tile(id TileID) (TileGeography, bool) {
	if g == nil || id >= TileCount {
		return TileGeography{}, false
	}
	return g.tiles[id], true
}

func (g *Grid) Tiles() [TileCount]TileGeography {
	if g == nil {
		return [TileCount]TileGeography{}
	}
	return g.tiles
}

type GridEdge struct {
	To         TileID
	StepLength float64
}

var neighborOrder = [...]struct{ dx, dy int }{
	{0, -1}, {1, -1}, {1, 0}, {1, 1}, {0, 1}, {-1, 1}, {-1, 0}, {-1, -1},
}

func (g *Grid) OrdinaryEdges(from TileID) []GridEdge {
	if g == nil || from >= TileCount || !g.tiles[from].Land {
		return nil
	}
	x, y, _ := TileXY(from)
	result := make([]GridEdge, 0, 8)
	for _, delta := range neighborOrder {
		to, err := TileIDAt(x+delta.dx, y+delta.dy)
		if err != nil || !g.tiles[to].Land {
			continue
		}
		diagonal := delta.dx != 0 && delta.dy != 0
		if diagonal {
			horizontal, hErr := TileIDAt(x+delta.dx, y)
			vertical, vErr := TileIDAt(x, y+delta.dy)
			if hErr != nil || vErr != nil || !g.tiles[horizontal].Land || !g.tiles[vertical].Land {
				continue
			}
		}
		length := 1.0
		if diagonal {
			length = math.Sqrt2
		}
		result = append(result, GridEdge{To: to, StepLength: length})
	}
	return result
}
