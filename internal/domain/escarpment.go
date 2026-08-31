package domain

import "fmt"

const MaxEscarpmentEdges = 24

const (
	escarpmentNorth uint8 = 1 << iota
	escarpmentEast
	escarpmentSouth
	escarpmentWest
)

// EscarpmentEdge is an authored, impassable boundary between two cardinally
// adjacent land tiles. The boundary is symmetric: a sheer descent is no safer
// for a migrating band than the corresponding ascent.
type EscarpmentEdge struct {
	Name          string
	First, Second TileID
}

type authoredEscarpment struct {
	name    string
	firstX  int
	firstY  int
	secondX int
	secondY int
}

// The gaps in these fronts are deliberate passes. Escarpments are authored
// instead of inferred from the grid's representative highland elevations: a
// whole plateau tile being high does not establish which of its approaches is
// a sheer cliff.
var authoredEscarpments = [...]authoredEscarpment{
	// Ethiopian Highlands: gaps at the central western/eastern, northern, and
	// southern approaches keep every surrounding route connected.
	{"Ethiopian Escarpment", 22, 31, 23, 31},
	{"Ethiopian Escarpment", 22, 32, 23, 32},
	{"Ethiopian Escarpment", 22, 34, 23, 34},
	{"Ethiopian Escarpment", 22, 35, 23, 35},
	{"Ethiopian Escarpment", 27, 33, 28, 33},
	{"Ethiopian Escarpment", 27, 34, 28, 34},
	{"Ethiopian Escarpment", 27, 35, 28, 35},
	{"Ethiopian Escarpment", 24, 29, 24, 30},
	{"Ethiopian Escarpment", 24, 35, 24, 36},
	{"Ethiopian Escarpment", 26, 35, 26, 36},

	// The Himalayan southern face is discontinuous; the omitted columns are
	// traversable passes, including routes into and around the Tibetan plateau.
	{"Himalayan Escarpment", 42, 24, 42, 25},
	{"Himalayan Escarpment", 43, 24, 43, 25},
	{"Himalayan Escarpment", 44, 24, 44, 25},
	{"Himalayan Escarpment", 46, 24, 46, 25},
	{"Himalayan Escarpment", 48, 24, 48, 25},
	{"Himalayan Escarpment", 49, 24, 49, 25},
	{"Himalayan Escarpment", 50, 24, 50, 25},
}

func (g *Grid) installEscarpments() error {
	if g == nil || len(authoredEscarpments) > MaxEscarpmentEdges {
		return fmt.Errorf("%w: escarpment catalog size", ErrInvalidValue)
	}
	for index, authored := range authoredEscarpments {
		first, firstErr := TileIDAt(authored.firstX, authored.firstY)
		second, secondErr := TileIDAt(authored.secondX, authored.secondY)
		if firstErr != nil || secondErr != nil || authored.name == "" {
			return fmt.Errorf("%w: escarpment %d endpoint", ErrInvalidValue, index)
		}
		dx, dy := authored.secondX-authored.firstX, authored.secondY-authored.firstY
		if absInt(dx)+absInt(dy) != 1 || !g.tiles[first].Land || !g.tiles[second].Land {
			return fmt.Errorf("%w: escarpment %d (%d,%d)-(%d,%d) must join cardinal land tiles (land %t/%t)", ErrInvalidValue, index, authored.firstX, authored.firstY, authored.secondX, authored.secondY, g.tiles[first].Land, g.tiles[second].Land)
		}
		if g.tiles[first].ElevationKm == g.tiles[second].ElevationKm {
			return fmt.Errorf("%w: escarpment %d does not bound an elevation change", ErrInvalidValue, index)
		}
		firstBit, secondBit := escarpmentDirectionBits(dx, dy)
		if g.escarpmentEdgeMask[first]&firstBit != 0 || g.escarpmentEdgeMask[second]&secondBit != 0 {
			return fmt.Errorf("%w: duplicate escarpment %d", ErrInvalidValue, index)
		}
		g.escarpments[index] = EscarpmentEdge{Name: authored.name, First: first, Second: second}
		g.escarpmentEdgeMask[first] |= firstBit
		g.escarpmentEdgeMask[second] |= secondBit
	}
	g.escarpmentCount = len(authoredEscarpments)
	return nil
}

func escarpmentDirectionBits(dx, dy int) (uint8, uint8) {
	switch {
	case dx == 1:
		return escarpmentEast, escarpmentWest
	case dx == -1:
		return escarpmentWest, escarpmentEast
	case dy == 1:
		return escarpmentSouth, escarpmentNorth
	default:
		return escarpmentNorth, escarpmentSouth
	}
}

// Escarpments returns the stable authored catalog without exposing the grid's
// backing storage.
func (g *Grid) Escarpments() []EscarpmentEdge {
	if g == nil || g.escarpmentCount == 0 {
		return nil
	}
	result := make([]EscarpmentEdge, g.escarpmentCount)
	copy(result, g.escarpments[:g.escarpmentCount])
	return result
}

// HasEscarpment reports whether a cardinal tile boundary is impassable.
func (g *Grid) HasEscarpment(first, second TileID) bool {
	if g == nil || first >= TileCount || second >= TileCount {
		return false
	}
	firstX, firstY, _ := TileXY(first)
	secondX, secondY, _ := TileXY(second)
	dx, dy := secondX-firstX, secondY-firstY
	if absInt(dx)+absInt(dy) != 1 {
		return false
	}
	firstBit, _ := escarpmentDirectionBits(dx, dy)
	return g.escarpmentEdgeMask[first]&firstBit != 0
}

// EscarpmentBlocks applies the cardinal barriers to an ordinary move. A
// diagonal must have all four sides of its enclosing square clear, preventing
// a band from jumping around the end of a one-tile cliff segment.
func (g *Grid) EscarpmentBlocks(from, to TileID) bool {
	if g == nil || from >= TileCount || to >= TileCount {
		return false
	}
	fromX, fromY, _ := TileXY(from)
	toX, toY, _ := TileXY(to)
	dx, dy := toX-fromX, toY-fromY
	if absInt(dx) > 1 || absInt(dy) > 1 || dx == 0 && dy == 0 {
		return false
	}
	if dx == 0 || dy == 0 {
		return g.HasEscarpment(from, to)
	}
	horizontal, hErr := TileIDAt(toX, fromY)
	vertical, vErr := TileIDAt(fromX, toY)
	if hErr != nil || vErr != nil {
		return false
	}
	return g.HasEscarpment(from, horizontal) ||
		g.HasEscarpment(from, vertical) ||
		g.HasEscarpment(horizontal, to) ||
		g.HasEscarpment(vertical, to)
}
