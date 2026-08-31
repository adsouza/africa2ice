package domain

import "testing"

func TestStartingPopulationCatalog(t *testing.T) {
	want := [...]Population{120, 120, 120, 120, 120, 60, 60, 150}
	if len(StartingAnchors) != len(want) {
		t.Fatalf("starting anchors = %d, want %d", len(StartingAnchors), len(want))
	}
	for index, population := range want {
		if StartingAnchors[index].Population != population {
			t.Fatalf("anchor %d population = %d, want %d", index, StartingAnchors[index].Population, population)
		}
	}
}

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

func TestBaishiyaAnchorRepresentsAHighAltitudeDenisovanBand(t *testing.T) {
	const wantName = "Baishiya Karst Cave (Denisovan)"
	index := len(StartingAnchors) - 1
	anchor := StartingAnchors[index]
	if anchor.Name != wantName || anchor.Species != ArchaicHominin || anchor.Region != YellowRiverBasin {
		t.Fatalf("Baishiya anchor = %#v", anchor)
	}

	grid, err := (WorldGenerator{}).Generate()
	if err != nil {
		t.Fatal(err)
	}
	geography, ok := grid.Tile(StartingTileIDs[index])
	if !ok {
		t.Fatalf("missing Baishiya tile %d", StartingTileIDs[index])
	}
	if geography.ElevationKm <= HighlandElevationKm {
		t.Fatalf("Baishiya elevation = %.2f km, want > %.2f km", geography.ElevationKm, HighlandElevationKm)
	}

	traits, ok := StartingHeritableState(anchor.Species, anchor.Region)
	if !ok {
		t.Fatal("Baishiya band has no starting heritable-state profile")
	}
	if traits[HighAltitudeAdaptation] != archaicYellowRiverTraits[HighAltitudeAdaptation] || traits[HighAltitudeAdaptation] <= archaicFrangistanTraits[HighAltitudeAdaptation] {
		t.Fatalf("Baishiya high-altitude adaptation = %.2f", traits[HighAltitudeAdaptation])
	}
}
