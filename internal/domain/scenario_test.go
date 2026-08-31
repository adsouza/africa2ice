package domain

import "testing"

func TestStartingPopulationCatalog(t *testing.T) {
	want := [...]Population{120, 120, 120, 120, 120, 60, 60, 12, 12, 90}
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

// The three Denisovan-representative anchors share the lineage's high-altitude
// adaptation, which is what distinguishes them from the western archaic
// populations, while differing on the pressures their own ground applies.
func TestDenisovanAnchorsShareTheLineageAltitudeMarker(t *testing.T) {
	wanted := map[string]Region{
		"Denisova Cave (Altai)": Siberia,
		"Tam Pa Ling":           SoutheastAsia,
		"Harbin (Denisovan)":    EastAsia,
	}
	found := 0
	for _, anchor := range StartingAnchors {
		region, ok := wanted[anchor.Name]
		if !ok {
			continue
		}
		found++
		if anchor.Species != ArchaicHominin || anchor.Region != region {
			t.Fatalf("%s anchor = %#v", anchor.Name, anchor)
		}
		traits, ok := StartingHeritableState(anchor.Species, anchor.Region)
		if !ok {
			t.Fatalf("%s has no starting heritable-state profile", anchor.Name)
		}
		if traits[HighAltitudeAdaptation] <= archaicFrangistanTraits[HighAltitudeAdaptation] {
			t.Fatalf("%s high-altitude adaptation = %.2f, want above the western archaic value %.2f",
				anchor.Name, traits[HighAltitudeAdaptation], archaicFrangistanTraits[HighAltitudeAdaptation])
		}
	}
	if found != len(wanted) {
		t.Fatalf("found %d of the %d Denisovan anchors", found, len(wanted))
	}
}

// No anchor may start in maximal crowding collapse.
//
// ResolveStartingTiles only requires BaselineK > 0, which reads as a viability
// check and is not one: a tile with K = 4.1 satisfies it while supporting nobody.
// A Denisovan anchor placed at the 3,280 m Baishiya Karst Cave resolved to
// exactly such a tile and began 36x over its capacity, losing 90% of its people
// within five turns — a placement no gate in this package objected to.
//
// The bound is derived rather than chosen. The crowding term is pinned at
// MaxCrowdingDeclineFraction once r · (P_total/K_eff − 1) reaches it, so a band
// starting at or beyond that ratio takes the maximum loss every turn from turn
// one. Anchors are allowed to start stressed, and several deliberately do; what
// they may not do is start already in free fall.
func TestNoStartingAnchorBeginsInMaximalCrowdingCollapse(t *testing.T) {
	grid, err := (WorldGenerator{}).Generate()
	if err != nil {
		t.Fatal(err)
	}
	habitat, _, err := BuildHabitat(grid, 0, 0)
	if err != nil {
		t.Fatal(err)
	}
	limit := 1 + MaxCrowdingDeclineFraction/PopulationGrowthRate
	for index, anchor := range StartingAnchors {
		capacity := float64(habitat[StartingTileIDs[index]].BaselineK)
		if capacity <= 0 {
			t.Fatalf("%s resolved to tile %d with no capacity", anchor.Name, StartingTileIDs[index])
		}
		if ratio := float64(anchor.Population) / capacity; ratio >= limit {
			t.Errorf("%s starts %.1fx over its tile's capacity (%d people on K %.1f); the crowding term is pinned at its bound from %.1fx",
				anchor.Name, ratio, anchor.Population, capacity, limit)
		}
	}
}
