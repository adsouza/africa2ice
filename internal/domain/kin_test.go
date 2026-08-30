package domain

import "testing"

// Kin support reuses the same "same tile or adjacent" predicate that same-species
// gene flow already uses, so the model has one notion of a neighbouring band.
// It attaches to acute events, the only term that moves an integer population
// under this calibration; fertility is revisited once growth can register.

func TestKinContactShareIsZeroWithoutContactAndSaturates(t *testing.T) {
	if got := kinContactShare(0); got != 0 {
		t.Fatalf("isolated band share = %v, want exactly 0", got)
	}
	if got := kinContactShare(-3); got != 0 {
		t.Fatalf("negative contact count share = %v, want exactly 0", got)
	}
	previous := 0.0
	for contacts := 1; contacts <= 64; contacts++ {
		got := kinContactShare(contacts)
		if got <= previous || got >= 1 {
			t.Fatalf("share at %d contacts = %v, want strictly between %v and 1", contacts, got, previous)
		}
		previous = got
	}
}

// The acute factor follows the codebase's remaining-risk idiom: 1 means the
// hazard is untouched, and kin support can only ever reduce it.
func TestKinSupportRemainingRiskIsOneAloneAndBoundedBelow(t *testing.T) {
	if got := KinSupportRemainingRisk(0); got != 1 {
		t.Fatalf("isolated band remaining risk = %v, want exactly 1", got)
	}
	floor := 1 - MaxKinAcuteReduction
	previous := 1.0
	for contacts := 1; contacts <= 64; contacts++ {
		got := KinSupportRemainingRisk(contacts)
		if got >= previous {
			t.Fatalf("remaining risk at %d contacts = %v, not below %v", contacts, got, previous)
		}
		if got < floor {
			t.Fatalf("remaining risk at %d contacts = %v, below the floor %v", contacts, got, floor)
		}
		previous = got
	}
}

func acuteFixture(t *testing.T) (Band, TileGeography, HabitatTile, Season, [AcuteKindCount]float64) {
	t.Helper()
	world, err := NewWorld(BalanceSeedCorpus[0])
	if err != nil {
		t.Fatal(err)
	}
	band := world.Bands()[0]
	geography, _ := world.Grid().Tile(band.TileID)
	work := [AcuteKindCount]float64{}
	work[AcutePredation] = 0.05
	season, _ := SeasonForTurn(0)
	return band, geography, world.Habitat()[band.TileID], season, work
}

func TestKinSupportLowersEveryAcuteProbability(t *testing.T) {
	band, geography, habitat, season, work := acuteFixture(t)
	alone := AcuteProbabilities(band, geography, habitat, season, work, false, PassageCount, 0)
	supported := AcuteProbabilities(band, geography, habitat, season, work, false, PassageCount, 4)
	loweredSomething := false
	for kind := range alone {
		if supported[kind] > alone[kind] {
			t.Fatalf("kin support raised acute risk for kind %d: %v vs %v", kind, supported[kind], alone[kind])
		}
		if supported[kind] < alone[kind] {
			loweredSomething = true
		}
	}
	if !loweredSomething {
		t.Fatal("kin support left every acute probability unchanged")
	}
}

func TestAcuteProbabilitiesStayBoundedAndDeterministic(t *testing.T) {
	band, geography, habitat, season, work := acuteFixture(t)
	before := AcuteProbabilities(band, geography, habitat, season, work, false, PassageCount, 3)
	after := AcuteProbabilities(band, geography, habitat, season, work, false, PassageCount, 3)
	if before != after {
		t.Fatal("AcuteProbabilities is not deterministic")
	}
	for kind, probability := range before {
		if probability < 0 || probability > 1 {
			t.Fatalf("acute probability %d = %v, outside [0, 1]", kind, probability)
		}
	}
}

func contactFixture(t *testing.T) (*Grid, TileID, TileID, TileID) {
	t.Helper()
	grid, err := (WorldGenerator{}).Generate()
	if err != nil {
		t.Fatal(err)
	}
	home := StartingTileIDs[0]
	edges := grid.OrdinaryEdges(home)
	if len(edges) == 0 {
		t.Fatal("starting tile has no ordinary neighbours")
	}
	neighbour := edges[0].To
	far, found := TileID(0), false
	for id := range TileCount {
		candidate := TileID(id)
		if candidate != home && candidate != neighbour && !ordinaryContact(grid, home, candidate) {
			far, found = candidate, true
			break
		}
	}
	if !found {
		t.Fatal("no distant tile found")
	}
	return grid, home, neighbour, far
}

func TestKinContactCountsSameTileAndAdjacentButNotDistant(t *testing.T) {
	grid, home, neighbour, far := contactFixture(t)
	snapshot := []Band{
		{ID: 1, Species: HomoSapiens, TileID: home, Population: 50},
		{ID: 2, Species: HomoSapiens, TileID: home, Population: 50},
		{ID: 3, Species: HomoSapiens, TileID: neighbour, Population: 50},
		{ID: 4, Species: HomoSapiens, TileID: far, Population: 50},
	}
	counts := kinContactCounts(snapshot, grid)
	if counts[0] != 2 {
		t.Fatalf("band on a shared tile beside a neighbour counted %d kin, want 2", counts[0])
	}
	if counts[3] != 0 {
		t.Fatalf("distant band counted %d kin, want 0", counts[3])
	}
}

// Archaic bands are a population too, and their own gene-flow rule already
// treats same-species contact symmetrically.
func TestKinContactIsSymmetricAcrossSpecies(t *testing.T) {
	grid, home, neighbour, _ := contactFixture(t)
	snapshot := []Band{
		{ID: 1, Species: ArchaicHominin, TileID: home, Population: 50},
		{ID: 2, Species: ArchaicHominin, TileID: neighbour, Population: 50},
		{ID: 3, Species: HomoSapiens, TileID: home, Population: 50},
	}
	counts := kinContactCounts(snapshot, grid)
	if counts[0] != 1 || counts[1] != 1 {
		t.Fatalf("archaic pair counted %d and %d kin, want 1 each", counts[0], counts[1])
	}
	if counts[2] != 0 {
		t.Fatalf("a lone sapiens counted %d kin beside archaics, want 0", counts[2])
	}
}

func TestKinContactIgnoresExtinctBandsAndSelf(t *testing.T) {
	grid, home, neighbour, _ := contactFixture(t)
	snapshot := []Band{
		{ID: 1, Species: HomoSapiens, TileID: home, Population: 50},
		{ID: 2, Species: HomoSapiens, TileID: neighbour, Population: 0},
	}
	counts := kinContactCounts(snapshot, grid)
	if counts[0] != 0 {
		t.Fatalf("an extinct neighbour counted as kin: %d", counts[0])
	}
	if counts[1] != 0 {
		t.Fatalf("an extinct band was credited with kin: %d", counts[1])
	}
}

// Counting from a snapshot is what keeps the result independent of the order
// bands happen to sit in, which phase 3 mutates as it runs.
func TestKinContactCountsAreOrderIndependent(t *testing.T) {
	grid, home, neighbour, far := contactFixture(t)
	forward := []Band{
		{ID: 1, Species: HomoSapiens, TileID: home, Population: 50},
		{ID: 2, Species: HomoSapiens, TileID: neighbour, Population: 50},
		{ID: 3, Species: HomoSapiens, TileID: far, Population: 50},
	}
	reversed := []Band{forward[2], forward[1], forward[0]}
	forwardCounts, reversedCounts := kinContactCounts(forward, grid), kinContactCounts(reversed, grid)
	if forwardCounts[0] != reversedCounts[2] || forwardCounts[2] != reversedCounts[0] {
		t.Fatalf("counts depend on band order: %v vs %v", forwardCounts, reversedCounts)
	}
}
