package domain

import (
	"math"
	"reflect"
	"testing"
)

// shuffledOrder returns a deterministic permutation of 0..n-1. It draws from
// SplitMix64 rather than math/rand, which the domain forbids outside rng.go.
func shuffledOrder(n int, seed uint64) []int {
	order := make([]int, n)
	for index := range order {
		order[index] = index
	}
	state := seed
	for index := n - 1; index > 0; index-- {
		var value uint64
		state, value = SplitMix64(state)
		swap := int(value % uint64(index+1))
		order[index], order[swap] = order[swap], order[index]
	}
	return order
}

// ulpsApart is how many representable float64 values separate a and b.
func ulpsApart(a, b float64) uint64 {
	if a == b {
		return 0
	}
	ai, bi := int64(math.Float64bits(a)), int64(math.Float64bits(b))
	if ai < 0 {
		ai = math.MinInt64 - ai
	}
	if bi < 0 {
		bi = math.MinInt64 - bi
	}
	if ai > bi {
		return uint64(ai - bi)
	}
	return uint64(bi - ai)
}

// Step 5b: shared-stock allocation must not reward a band for where it sits in
// the evaluation order. Each permutation of the same demands must give every
// demand the same share, up to float rounding, and the total must never exceed
// the stock.
//
// The rounding is not zero. ProportionalAllocate sums the scaled demands in
// slice order and charges any overshoot of the stock to the last positive
// demand, so moving a band to the end can cost it a few rounding errors (63
// ulps of a 1.34 FU share, about 1.4e-14 FU, was observed). Making the
// allocator bit-exact under permutation would change every simulation result
// and the shared-stock algorithm versions, for an effect far below one
// person's food. The bound is therefore what that mechanism can produce: n
// additions' worth of rounding of the stock, n·2⁻⁵²·available. A real ordering
// bias, such as serving demands first come first served, exceeds it by many
// orders of magnitude.
func TestProportionalAllocateIsPermutationInvariant(t *testing.T) {
	cases := map[string]struct {
		available float64
		demands   []float64
	}{
		"scarce, uneven":            {60, []float64{200, 100, 0, 37.5, 12.25, 200, 1e-9}},
		"abundant":                  {1e6, []float64{200, 100, 37.5, 0, 12.25}},
		"equal demands under limit": {10, []float64{7, 7, 7, 7, 7, 7, 7}},
		"products near overflow":    {5e307, []float64{1e308, 1.5e308, 3e307, 1, 0}},
		"one positive demand":       {3, []float64{0, 0, 9, 0}},
	}
	for name, c := range cases {
		t.Run(name, func(t *testing.T) {
			reference := ProportionalAllocate(c.available, c.demands)
			for seed := uint64(1); seed <= 64; seed++ {
				order := shuffledOrder(len(c.demands), seed)
				permuted := make([]float64, len(c.demands))
				for position, original := range order {
					permuted[position] = c.demands[original]
				}
				allocated := ProportionalAllocate(c.available, permuted)
				bound := float64(float64(float64(len(c.demands))*0x1p-52) * c.available)
				total := 0.0
				for position, original := range order {
					total += allocated[position]
					if difference := math.Abs(allocated[position] - reference[original]); difference > bound {
						t.Fatalf("seed %d: demand %d (%v) got %v in position %d, %v first; %v apart, bound %v",
							seed, original, c.demands[original], allocated[position], position, reference[original], difference, bound)
					}
					if allocated[position] < 0 || math.IsNaN(allocated[position]) || math.IsInf(allocated[position], 0) {
						t.Fatalf("seed %d: allocation %v is not finite and non-negative", seed, allocated[position])
					}
				}
				if total > c.available {
					t.Fatalf("seed %d: allocated %v of %v", seed, total, c.available)
				}
			}
		})
	}
}

// diffusionScenario is five bands sharing contacts: two sapiens and one
// archaic on one tile, one sapiens on an adjacent tile, and an isolated
// archaic band. Every trait is non-zero, so no band is eligible for a
// mutation draw and the whole knowledge-and-genetics step must consume no
// randomness at all.
func diffusionScenario(t *testing.T, grid *Grid) []Band {
	t.Helper()
	origin := StartingTileIDs[0]
	x, y, _ := TileXY(origin)
	neighbour := InvalidTileID
	for _, edge := range grid.OrdinaryEdges(origin) {
		ex, ey, _ := TileXY(edge.To)
		if ex == x || ey == y {
			neighbour = edge.To
			break
		}
	}
	far := StartingTileIDs[len(StartingTileIDs)-1]
	if neighbour == InvalidTileID || ordinaryContact(grid, origin, far) || origin == far {
		t.Fatal("scenario needs an ordinary neighbour and a distant tile")
	}
	knows := func(technologies ...Technology) TechnologyState {
		state := TechnologyState{}
		for _, technology := range technologies {
			state.Acquired |= 1 << technology
			state.Progress[technology] = ResearchCost[technology]
		}
		return state
	}
	learning := knows(HaftedTools)
	learning.Target = Some(CordageAndNets)
	return []Band{
		{ID: 1, Species: HomoSapiens, TileID: origin, Population: 120, Heritable: uniformTraits(0.2), Technology: knows(Firecraft), InterbreedTarget: Some(BandID(3))},
		{ID: 2, Species: HomoSapiens, TileID: origin, Population: 45, Heritable: uniformTraits(0.6), Technology: learning, InterbreedTarget: Some(BandID(3))},
		{ID: 3, Species: ArchaicHominin, TileID: origin, Population: 80, Heritable: uniformTraits(0.9), Technology: knows(Firecraft, PlantKnowledge)},
		{ID: 4, Species: HomoSapiens, TileID: neighbour, Population: 70, Heritable: uniformTraits(0.4), Technology: knows(HaftedTools, PlantKnowledge)},
		{ID: 5, Species: ArchaicHominin, TileID: far, Population: 60, Heritable: uniformTraits(0.7), Technology: knows(Firecraft)},
	}
}

// Step 5d: technology contact pairs are discovered in band order, and the
// result must not depend on it. Permuting the band slice permutes discovery
// order for diffusion, gene-flow partners, and the two sapiens bands
// interbreeding with one archaic band. Technology state is integer counts
// times fixed costs and must match exactly; trait values are float sums over
// partners and must match within a few ulps. No permutation may draw from the
// world RNG.
func TestKnowledgeAndGeneticsAreInvariantToDiscoveryOrder(t *testing.T) {
	grid := generatedGrid(t)
	research := map[BandID]float64{2: 7.5, 4: 3}
	selection := map[BandID]HeritableState{1: {0.01}, 4: {0, 0.02}}

	reference := diffusionScenario(t, grid)
	rng := NewWorldRNG(9)
	before, _ := rng.MarshalBinary()
	referenceInterbred := applyKnowledgeAndGenetics(reference, grid, research, selection, rng)
	if after, _ := rng.MarshalBinary(); string(after) != string(before) {
		t.Fatal("knowledge and genetics drew from the world RNG with no mutation-eligible trait")
	}
	byID := map[BandID]Band{}
	for _, band := range reference {
		byID[band.ID] = band
	}
	if byID[2].Technology.Progress[CordageAndNets] == 0 || byID[4].Technology.Progress[Firecraft] == 0 {
		t.Fatal("scenario produced no diffusion; the invariance check would be vacuous")
	}

	for seed := uint64(1); seed <= 32; seed++ {
		bands := diffusionScenario(t, grid)
		order := shuffledOrder(len(bands), seed)
		permuted := make([]Band, len(bands))
		for position, original := range order {
			permuted[position] = bands[original]
		}
		rng := NewWorldRNG(9)
		interbred := applyKnowledgeAndGenetics(permuted, grid, research, selection, rng)
		if after, _ := rng.MarshalBinary(); string(after) != string(before) {
			t.Fatalf("seed %d: permuted order drew from the world RNG", seed)
		}
		for id, done := range referenceInterbred {
			if interbred[id] != done {
				t.Fatalf("seed %d: band %d interbreeding completion %t, %t first", seed, id, interbred[id], done)
			}
		}
		for _, band := range permuted {
			want := byID[band.ID]
			if band.Technology != want.Technology {
				t.Fatalf("seed %d: band %d technology %+v, %+v first", seed, band.ID, band.Technology, want.Technology)
			}
			for trait := HeritableTrait(0); trait < HeritableTraitCount; trait++ {
				if apart := ulpsApart(float64(band.Heritable[trait]), float64(want.Heritable[trait])); apart > 4 {
					t.Fatalf("seed %d: band %d trait %d %v, %v first; %d ulps apart",
						seed, band.ID, trait, band.Heritable[trait], want.Heritable[trait], apart)
				}
			}
		}
	}
}

// permutations calls visit with every ordering of 0..n-1 (Heap's algorithm).
func permutations(n int, visit func([]int)) {
	order := make([]int, n)
	for index := range order {
		order[index] = index
	}
	var generate func(k int)
	generate = func(k int) {
		if k <= 1 {
			visit(append([]int(nil), order...))
			return
		}
		for index := 0; index < k-1; index++ {
			generate(k - 1)
			if k%2 == 0 {
				order[index], order[k-1] = order[k-1], order[index]
			} else {
				order[0], order[k-1] = order[k-1], order[0]
			}
		}
		generate(k - 1)
	}
	generate(n)
}

// Step 5a and §7: player commands for different bands commute. Two sapiens
// bands interbreed with one archaic band, a third changes its assignment and
// queues a migration, and a fourth selects research; every ordering of those
// five commands must leave an identical world after the turn resolves,
// archaic batch included. Band storage is ID-ordered by construction
// (RestoreWorld rejects anything else), so the order a player issues commands
// in is the ordering that can actually vary.
func TestPlayerCommandOrderDoesNotChangeTheResolvedTurn(t *testing.T) {
	setup := func(t *testing.T) *World {
		t.Helper()
		world, err := NewWorld(7)
		if err != nil {
			t.Fatal(err)
		}
		shared := world.bands[0].TileID
		world.bands[1].TileID = shared // sapiens band 2 joins band 1
		world.bands[4].TileID = shared // archaic band 5 joins them
		if world.bands[1].Species != HomoSapiens || world.bands[4].Species != ArchaicHominin {
			t.Fatal("starting catalog no longer puts sapiens at index 1 and archaic at index 4")
		}
		return world
	}
	probe := setup(t)
	candidates := probe.MigrationCandidates(3)
	if len(candidates) == 0 {
		t.Fatal("band 3 has no migration candidate")
	}
	destination := candidates[0].TileID
	commands := []struct {
		name string
		run  func(*World) error
	}{
		{"band 1 interbreeds with 5", func(w *World) error { return w.Interbreed(1, 5, true) }},
		{"band 2 interbreeds with 5", func(w *World) error { return w.Interbreed(2, 5, true) }},
		{"band 3 reassigns", func(w *World) error {
			return w.SetAssignment(3, [AssignmentCount]AssignmentBP{2000, 4000, 1000, 1000, 2000}, true)
		}},
		{"band 3 migrates", func(w *World) error { return w.QueueMigration(3, destination, true) }},
		{"band 4 researches", func(w *World) error { return w.Research(4, PlantKnowledge, true) }},
	}

	var reference State
	referenceSet := false
	checked := 0
	permutations(len(commands), func(order []int) {
		world := setup(t)
		for _, index := range order {
			if err := commands[index].run(world); err != nil {
				t.Fatalf("order %v: %s: %v", order, commands[index].name, err)
			}
		}
		if err := world.AdvanceTurn(); err != nil {
			t.Fatalf("order %v: %v", order, err)
		}
		state, err := world.ExportState()
		if err != nil {
			t.Fatal(err)
		}
		checked++
		if !referenceSet {
			reference, referenceSet = state, true
			return
		}
		if !reflect.DeepEqual(state, reference) {
			t.Fatalf("order %v resolved differently from the first ordering", order)
		}
	})
	if checked != 120 {
		t.Fatalf("checked %d orderings, want 5! = 120", checked)
	}
	// Both interbreedings must actually have resolved, or the check above
	// compared worlds in which nothing order-sensitive happened.
	for _, band := range reference.Bands {
		if band.ID <= 2 && band.InterbreedTarget.Present() {
			t.Fatalf("band %d still holds an interbreeding intent after the turn", band.ID)
		}
	}
	if reference.Bands[2].TileID != destination {
		t.Fatalf("band 3 ended on %d, not its queued destination %d", reference.Bands[2].TileID, destination)
	}
}
