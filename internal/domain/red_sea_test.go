package domain

import "testing"

func TestRedSeaSeparatesAfricaAndArabia(t *testing.T) {
	grid := generatedGrid(t)
	for _, point := range []GeoPoint{g(43.3, 12.5), g(42.5, 11), g(45, 11), g(49, 13), g(54, 12)} {
		tile, _ := grid.Tile(nearestTile(t, point))
		if tile.Land {
			t.Errorf("Bab-el-Mandeb/Gulf of Aden point %+v is walkable tile %d", point, tile.ID)
		}
	}
	for _, point := range []GeoPoint{g(40, 24), g(45, 19), g(49, 17)} {
		tile, _ := grid.Tile(nearestTile(t, point))
		if !tile.Land || tile.Region != Arabia {
			t.Errorf("Arabian shore %+v classified as %+v", point, tile)
		}
	}
	// Search actual movement edges, including the diagonal corner rule. No
	// route may cross south of Sinai, but the northern land route must survive.
	canReachArabia := func(northernRoute bool) bool {
		start := StartingTileIDs[0]
		seen := [TileCount]bool{}
		seen[start] = true
		queue := []TileID{start}
		for len(queue) > 0 {
			id := queue[0]
			queue = queue[1:]
			tile, _ := grid.Tile(id)
			if tile.Region == Arabia {
				return true
			}
			for _, edge := range grid.OrdinaryEdges(id) {
				next, _ := grid.Tile(edge.To)
				if seen[edge.To] || (!northernRoute && next.Latitude >= 28) {
					continue
				}
				seen[edge.To] = true
				queue = append(queue, edge.To)
			}
		}
		return false
	}
	if canReachArabia(false) {
		t.Fatal("Africa can still reach Arabia on foot south of Sinai")
	}
	if !canReachArabia(true) {
		t.Fatal("correction disconnected the northern land route into Arabia")
	}
}
