package domain

import "testing"

// ordinaryComponent is every tile a chain of ordinary land moves reaches from
// start, ignoring habitability.
func ordinaryComponent(grid *Grid, start TileID) map[TileID]bool {
	seen := map[TileID]bool{start: true}
	queue := []TileID{start}
	for len(queue) > 0 {
		id := queue[0]
		queue = queue[1:]
		for _, edge := range grid.OrdinaryEdges(id) {
			if !seen[edge.To] {
				seen[edge.To] = true
				queue = append(queue, edge.To)
			}
		}
	}
	return seen
}

// Sahul and Beringia are achievements that must prove their passage was
// crossed (§6). Endpoint non-adjacency cannot show that: in dispersal-map-v5
// both gaps could be walked around, Wallacea in 21 steps down a Sahul strip
// under Java and Bering in 11 across the overlapping Siberian and Alaskan
// rings. Each destination must be its own ordinary-move component.
func TestPassageDestinationsAreSealedLandComponents(t *testing.T) {
	grid := generatedGrid(t)
	for _, region := range []Region{Sahul, Beringia} {
		start, members := InvalidTileID, 0
		for id := range TileCount {
			tile, _ := grid.Tile(TileID(id))
			if tile.Land && tile.Region == region {
				members++
				if start == InvalidTileID {
					start = TileID(id)
				}
			}
		}
		if members == 0 {
			t.Fatalf("%v has no land", region)
		}
		component := ordinaryComponent(grid, start)
		for id := range component {
			if tile, _ := grid.Tile(id); tile.Region != region {
				t.Fatalf("%v reaches %v tile %d on foot", region, tile.Region, id)
			}
		}
		if len(component) != members {
			t.Fatalf("%v is %d disconnected pieces of %d tiles; component from %d holds %d", region, members, members, start, len(component))
		}
	}
	for _, passage := range Passages() {
		if grid.ordinaryRouteExists(passage.From, passage.To) {
			t.Errorf("passage %d endpoints %d and %d are joined on foot", passage.ID, passage.From, passage.To)
		}
	}
}

// §6 resolves each frozen endpoint as the nearest land tile on its named side
// of the gap, ties broken by tile ID. The side is the region the crossing
// leaves or enters.
func TestPassageEndpointsResolveFromAnchors(t *testing.T) {
	grid := generatedGrid(t)
	resolve := func(anchor GeoPoint, side Region) TileID {
		projected, err := ProjectGeo(anchor)
		if err != nil {
			t.Fatal(err)
		}
		best, bestDistance := InvalidTileID, 0.0
		for id := range TileCount {
			tile, _ := grid.Tile(TileID(id))
			if !tile.Land || tile.Region != side {
				continue
			}
			dx, dy := float64(tile.X)-projected.X, float64(tile.Y)-projected.Y
			if distance := float64(dx*dx) + float64(dy*dy); best == InvalidTileID || distance < bestDistance {
				best, bestDistance = TileID(id), distance
			}
		}
		return best
	}
	cases := []struct {
		id       PassageID
		from, to GeoPoint
		fromSide Region
		toSide   Region
	}{
		{NorthWallacea, g(118, -3), g(138, -3), SoutheastAsia, Sahul},
		{SouthWallacea, g(120, -11), g(140, -14), SoutheastAsia, Sahul},
		{BeringStrait, g(182, 55), g(191, 55), Siberia, Beringia},
	}
	for _, c := range cases {
		passage := Passages()[c.id]
		if from := resolve(c.from, c.fromSide); from != passage.From {
			t.Errorf("passage %d origin resolves to %d, catalog has %d", c.id, from, passage.From)
		}
		if to := resolve(c.to, c.toSide); to != passage.To {
			t.Errorf("passage %d destination resolves to %d, catalog has %d", c.id, to, passage.To)
		}
	}
}

// The Campi Flegrei epicenter must be Frangistan land (step 3). In v5 the
// Mediterranean ring covered all of Italy, so the direct and proximal tiers
// were entirely sea and the "cannot sterilize the direct zone" fixture tested
// a water tile.
func TestCampanianZonesStrikeLand(t *testing.T) {
	grid := generatedGrid(t)
	land := map[MacroZone]int{}
	for id := range TileCount {
		tile, _ := grid.Tile(TileID(id))
		if !tile.Land {
			continue
		}
		zone := CampanianZone(TileID(id))
		land[zone]++
		if zone == MacroDirect && tile.Region != Frangistan {
			t.Errorf("direct-zone tile %d is %v, want Frangistan", id, tile.Region)
		}
	}
	for _, zone := range []MacroZone{MacroDirect, MacroProximal, MacroWide} {
		if land[zone] == 0 {
			t.Errorf("Campanian zone %d contains no land", zone)
		}
	}
	// The peninsula hangs off the mainland: a band can walk to the epicenter.
	epicenter := InvalidTileID
	for id := range TileCount {
		if CampanianZone(TileID(id)) == MacroDirect {
			epicenter = TileID(id)
		}
	}
	if !grid.ordinaryRouteExists(StartingTileIDs[0], epicenter) {
		t.Error("the Campanian epicenter is not reachable on foot from East Africa")
	}
}

// Beringia's shores fall below the vegetation cold cutoff on every turn the
// land bridge is open, so without the refugium floor no band could hold an
// established population on the Alaska side. The floor must keep every
// refugium tile habitable on every turn, leave other tiles untouched, and make
// the crossing usable: both endpoints habitable on consecutive open turns.
func TestBeringianRefugiumKeepsTheCrossingUsable(t *testing.T) {
	grid := generatedGrid(t)
	bering := Passages()[BeringStrait]
	usable := false
	previousOpen, previousEndpoints := false, false
	for turn := 0; turn <= MaxCampaignTurn; turn++ {
		habitat, climate, err := BuildHabitat(grid, 0, turn)
		if err != nil {
			t.Fatal(err)
		}
		for id := range TileCount {
			tile, _ := grid.Tile(TileID(id))
			raw := BaselineK(habitat[id].VegetationIndex, habitat[id].Biome)
			switch {
			case !tile.Land:
			case InBeringianRefugium(tile):
				if habitat[id].BaselineK < BeringianRefugiumK {
					t.Fatalf("turn %d refugium tile %d BaselineK = %v, floor %v", turn, id, habitat[id].BaselineK, BeringianRefugiumK)
				}
			case habitat[id].BaselineK != raw:
				t.Fatalf("turn %d tile %d outside the refugium has BaselineK %v, curve gives %v", turn, id, habitat[id].BaselineK, raw)
			}
		}
		open := BeringiaOpen(climate.LongTermTempOffset)
		endpoints := habitat[bering.From].BaselineK > 0 && habitat[bering.To].BaselineK > 0
		if open && endpoints && previousOpen && previousEndpoints {
			usable = true
		}
		previousOpen, previousEndpoints = open, endpoints
	}
	if !usable {
		t.Fatal("the Bering crossing is never open with both endpoints habitable")
	}
	for _, id := range []TileID{bering.From, bering.To} {
		if tile, _ := grid.Tile(id); !InBeringianRefugium(tile) {
			t.Errorf("Bering endpoint %d lies outside the refugium", id)
		}
	}
}
