package domain

import "testing"

func TestEscarpmentCatalogIsStableCardinalLandGeography(t *testing.T) {
	grid, err := (WorldGenerator{}).Generate()
	if err != nil {
		t.Fatal(err)
	}
	edges := grid.Escarpments()
	if len(edges) != len(authoredEscarpments) {
		t.Fatalf("escarpment count = %d, catalog has %d", len(edges), len(authoredEscarpments))
	}
	if len(edges) != 17 {
		t.Fatalf("escarpment count = %d, want 17", len(edges))
	}
	for index, edge := range edges {
		first, firstOK := grid.Tile(edge.First)
		second, secondOK := grid.Tile(edge.Second)
		if !firstOK || !secondOK || !first.Land || !second.Land {
			t.Fatalf("edge %d does not join land: %+v", index, edge)
		}
		if absInt(first.X-second.X)+absInt(first.Y-second.Y) != 1 {
			t.Fatalf("edge %d is not cardinal: %+v", index, edge)
		}
		if first.ElevationKm == second.ElevationKm {
			t.Fatalf("edge %d has no elevation change: %+v", index, edge)
		}
		if !grid.HasEscarpment(edge.First, edge.Second) || !grid.HasEscarpment(edge.Second, edge.First) {
			t.Fatalf("edge %d is not symmetric: %+v", index, edge)
		}
	}
	edges[0] = EscarpmentEdge{}
	if grid.Escarpments()[0].Name == "" {
		t.Fatal("Escarpments exposed mutable backing storage")
	}
}

func TestOrdinaryEdgesRespectEscarpmentsAndAuthoredPasses(t *testing.T) {
	grid, err := (WorldGenerator{}).Generate()
	if err != nil {
		t.Fatal(err)
	}
	blockedFirst, _ := TileIDAt(22, 31)
	blockedSecond, _ := TileIDAt(23, 31)
	if ordinaryEdgeExists(grid, blockedFirst, blockedSecond) || ordinaryEdgeExists(grid, blockedSecond, blockedFirst) {
		t.Fatal("ordinary movement crossed the Ethiopian escarpment")
	}
	passFirst, _ := TileIDAt(22, 33)
	passSecond, _ := TileIDAt(23, 33)
	if !ordinaryEdgeExists(grid, passFirst, passSecond) || !ordinaryEdgeExists(grid, passSecond, passFirst) {
		t.Fatal("authored Ethiopian pass is not traversable")
	}
}

func TestDiagonalCannotCutAcrossEscarpmentCorner(t *testing.T) {
	grid, err := (WorldGenerator{}).Generate()
	if err != nil {
		t.Fatal(err)
	}
	from, _ := TileIDAt(22, 30)
	to, _ := TileIDAt(23, 31)
	if !grid.EscarpmentBlocks(from, to) {
		t.Fatal("diagonal did not inherit the enclosing escarpment")
	}
	if ordinaryEdgeExists(grid, from, to) {
		t.Fatal("ordinary movement cut diagonally around an escarpment")
	}
}

func ordinaryEdgeExists(grid *Grid, from, to TileID) bool {
	for _, edge := range grid.OrdinaryEdges(from) {
		if edge.To == to {
			return true
		}
	}
	return false
}
