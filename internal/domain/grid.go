package domain

import "math"

type Grid struct {
	tiles              [TileCount]TileGeography
	biomes             []Biome
	lastHabitableTurn  [TileCount]int16
	escarpments        [MaxEscarpmentEdges]EscarpmentEdge
	escarpmentCount    int
	escarpmentEdgeMask [TileCount]uint8
}

// LastHabitableTurn is the final campaign turn on which a tile has any capacity,
// or -1 for open water and for land the climate never supports. It is derived
// alongside the biome history because BaselineK depends on nothing the seed
// touches: the vegetation index is built from the habitat temperature and the
// effective moisture, and only LocalTemperatureC carries the per-seed noise.
// Presentation needs this to distinguish terrain closed for one cold snap from
// terrain the campaign has finished with.
func (g *Grid) LastHabitableTurn(id TileID) int {
	if g == nil || id >= TileCount {
		return -1
	}
	return int(g.lastHabitableTurn[id])
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

func (g *Grid) biomeAt(turn int, id TileID) (Biome, bool) {
	if g == nil || turn < 0 || turn > MaxCampaignTurn || id >= TileCount || len(g.biomes) != (MaxCampaignTurn+1)*TileCount {
		return 0, false
	}
	return g.biomes[turn*TileCount+int(id)], true
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
	return g.AppendOrdinaryEdges(make([]GridEdge, 0, MaxGridNeighbors), from)
}

// AppendOrdinaryEdges appends from's legal one-step moves to buffer and returns
// the extended slice. The turn pipeline walks the grid inside loops quadratic in
// the band count and, when planning routes, once per land tile per turn; those
// callers pass a reused buffer so walking it costs nothing to allocate.
func (g *Grid) AppendOrdinaryEdges(buffer []GridEdge, from TileID) []GridEdge {
	if g == nil || from >= TileCount || !g.tiles[from].Land {
		return buffer
	}
	x, y, _ := TileXY(from)
	for _, delta := range neighborOrder {
		if edge, ok := g.ordinaryEdge(from, x, y, delta.dx, delta.dy); ok {
			buffer = append(buffer, edge)
		}
	}
	return buffer
}

// OrdinaryNeighbors reports whether one ordinary land move connects the tiles.
// The contact predicates only ever asked the grid this question, and answering
// it directly avoids materializing an edge list per pair of bands.
func (g *Grid) OrdinaryNeighbors(from, to TileID) bool {
	if g == nil || from >= TileCount || to >= TileCount || !g.tiles[from].Land {
		return false
	}
	x, y, _ := TileXY(from)
	for _, delta := range neighborOrder {
		if edge, ok := g.ordinaryEdge(from, x, y, delta.dx, delta.dy); ok && edge.To == to {
			return true
		}
	}
	return false
}

// ordinaryEdge is the single authority on whether one step is legal, so the
// list, append, and predicate forms cannot drift apart. (x, y) are from's
// coordinates, already resolved by the caller.
func (g *Grid) ordinaryEdge(from TileID, x, y, dx, dy int) (GridEdge, bool) {
	to, err := TileIDAt(x+dx, y+dy)
	if err != nil || !g.tiles[to].Land {
		return GridEdge{}, false
	}
	diagonal := dx != 0 && dy != 0
	if diagonal {
		horizontal, hErr := TileIDAt(x+dx, y)
		vertical, vErr := TileIDAt(x, y+dy)
		if hErr != nil || vErr != nil || !g.tiles[horizontal].Land || !g.tiles[vertical].Land {
			return GridEdge{}, false
		}
	}
	if g.EscarpmentBlocks(from, to) {
		return GridEdge{}, false
	}
	length := 1.0
	if diagonal {
		length = math.Sqrt2
	}
	return GridEdge{To: to, StepLength: length}, true
}
