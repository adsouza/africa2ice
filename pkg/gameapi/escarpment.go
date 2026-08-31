package gameapi

// EscarpmentBlocks reports whether an explored escarpment prevents an
// ordinary one-tile move. Diagonal moves require all four sides of the
// enclosing square to be clear, matching the authoritative domain rule.
func EscarpmentBlocks(frame *Frame, from, to TileID) bool {
	if frame == nil || int(from) >= len(frame.Tiles) || int(to) >= len(frame.Tiles) {
		return false
	}
	fromTile, toTile := frame.Tiles[from], frame.Tiles[to]
	dx, dy := toTile.X-fromTile.X, toTile.Y-fromTile.Y
	if absolute(dx) > 1 || absolute(dy) > 1 || dx == 0 && dy == 0 {
		return false
	}
	if dx == 0 || dy == 0 {
		return hasEscarpment(frame, from, to)
	}
	horizontal, horizontalOK := tileAtCoordinates(frame, toTile.X, fromTile.Y)
	vertical, verticalOK := tileAtCoordinates(frame, fromTile.X, toTile.Y)
	if !horizontalOK || !verticalOK {
		return false
	}
	return hasEscarpment(frame, from, horizontal) ||
		hasEscarpment(frame, from, vertical) ||
		hasEscarpment(frame, horizontal, to) ||
		hasEscarpment(frame, vertical, to)
}

func hasEscarpment(frame *Frame, first, second TileID) bool {
	for _, edge := range frame.Escarpments {
		if edge.First == first && edge.Second == second || edge.First == second && edge.Second == first {
			return true
		}
	}
	return false
}

func tileAtCoordinates(frame *Frame, x, y int) (TileID, bool) {
	for index := range frame.Tiles {
		if frame.Tiles[index].X == x && frame.Tiles[index].Y == y {
			return TileID(index), true
		}
	}
	return 0, false
}

func absolute(value int) int {
	if value < 0 {
		return -value
	}
	return value
}
