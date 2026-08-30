package domain

import "testing"

func TestResolveStartingAnchors(t *testing.T) {
	grid, err := (WorldGenerator{}).Generate()
	if err != nil {
		t.Fatal(err)
	}
	habitat, _, err := BuildHabitat(grid, 0, 0)
	if err != nil {
		t.Fatal(err)
	}
	ids, err := ResolveStartingTiles(grid, habitat)
	if err != nil {
		t.Fatal(err)
	}
	if ids != StartingTileIDs {
		t.Fatalf("resolved starting tile IDs changed: %v != %v", ids, StartingTileIDs)
	}
	seen := map[TileID]bool{}
	for index, id := range ids {
		if seen[id] {
			t.Fatalf("duplicate starting tile %d", id)
		}
		seen[id] = true
		geography, _ := grid.Tile(id)
		if geography.Region != StartingAnchors[index].Region || habitat[id].BaselineK <= 0 {
			t.Fatalf("anchor %d invalid tile %d", index, id)
		}
	}
}
