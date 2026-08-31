package domain

import "testing"

// The list, append, and predicate forms all read the same ordinaryEdge rule.
// This pins them together: OrdinaryNeighbors is what the O(bands^2) contact
// loops call, and a divergence there would silently change gene flow and kin
// support rather than failing anywhere visible.
func TestOrdinaryEdgeFormsAgree(t *testing.T) {
	grid, err := canonicalGrid()
	if err != nil {
		t.Fatal(err)
	}
	buffer := make([]GridEdge, 0, MaxGridNeighbors)
	checked := 0
	for id := range TileCount {
		from := TileID(id)
		listed := grid.OrdinaryEdges(from)
		appended := grid.AppendOrdinaryEdges(buffer[:0], from)
		if len(listed) != len(appended) {
			t.Fatalf("tile %d: OrdinaryEdges gave %d edges, AppendOrdinaryEdges gave %d", id, len(listed), len(appended))
		}
		for index := range listed {
			if listed[index] != appended[index] {
				t.Fatalf("tile %d edge %d: %+v != %+v", id, index, listed[index], appended[index])
			}
		}
		neighbors := map[TileID]bool{}
		for _, edge := range listed {
			neighbors[edge.To] = true
			if !grid.OrdinaryNeighbors(from, edge.To) {
				t.Fatalf("tile %d: OrdinaryNeighbors denies listed edge to %d", id, edge.To)
			}
			checked++
		}
		// Every tile in the 3x3 block that is not a listed edge must be denied,
		// which covers water, blocked diagonals, and escarpments alike.
		x, y, _ := TileXY(from)
		for dy := -1; dy <= 1; dy++ {
			for dx := -1; dx <= 1; dx++ {
				to, err := TileIDAt(x+dx, y+dy)
				if err != nil || neighbors[to] {
					continue
				}
				if grid.OrdinaryNeighbors(from, to) {
					t.Fatalf("tile %d: OrdinaryNeighbors allows unlisted move to %d", id, to)
				}
			}
		}
	}
	if checked == 0 {
		t.Fatal("no edges were compared")
	}
}

func BenchmarkOrdinaryEdgeForms(b *testing.B) {
	grid, err := canonicalGrid()
	if err != nil {
		b.Fatal(err)
	}
	from := StartingTileIDs[0]
	target := grid.OrdinaryEdges(from)[0].To
	b.Run("OrdinaryEdges", func(b *testing.B) {
		b.ReportAllocs()
		for range b.N {
			_ = grid.OrdinaryEdges(from)
		}
	})
	b.Run("AppendOrdinaryEdges", func(b *testing.B) {
		buffer := make([]GridEdge, 0, MaxGridNeighbors)
		b.ReportAllocs()
		for range b.N {
			_ = grid.AppendOrdinaryEdges(buffer[:0], from)
		}
	})
	b.Run("OrdinaryNeighbors", func(b *testing.B) {
		b.ReportAllocs()
		for range b.N {
			_ = grid.OrdinaryNeighbors(from, target)
		}
	})
}
