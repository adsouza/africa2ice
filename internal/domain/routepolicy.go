package domain

// Deterministic sapiens route policies. DESIGN.md §12 step 5e requires the
// domain viability gate and the step-12 balance pass to drive complete campaigns
// through the *same* policies, so that "the campaign is viable" and "the
// campaign is balanced" are claims about one reproducible thing. They live in
// production code rather than in a test file for exactly that reason: the
// balance pass and the release-readiness job run outside this package.
//
// A policy is not an AI and is not a difficulty setting. It is a fixed decision
// rule that consumes no randomness of its own, so a campaign's entire history is
// a function of (seed, policy) alone.

// DestinationRegions are the five regions whose establishment ends the campaign
// in victory. The turn loop's own terminal check uses the same set.
var DestinationRegions = [5]Region{Frangistan, SouthAsia, YellowRiverBasin, Sahul, Beringia}

// RoutePolicy steers every sapiens band toward one destination region. The
// reference policy sets Target to RegionCount, meaning "take the most
// attractive move available and let dispersal fall where it may" — the
// pessimistic case, since nothing is steering it at all.
type RoutePolicy struct {
	Name   string
	Target Region
}

// ReferenceRoutePolicy is the unsteered survival case.
var ReferenceRoutePolicy = RoutePolicy{Name: "reference", Target: RegionCount}

// DirectedRoutePolicies are the five policies §12 step 13 names, one per
// destination region.
var DirectedRoutePolicies = [len(DestinationRegions)]RoutePolicy{
	{Name: "toward-frangistan", Target: Frangistan},
	{Name: "toward-south-asia", Target: SouthAsia},
	{Name: "toward-yellow-river", Target: YellowRiverBasin},
	{Name: "toward-sahul", Target: Sahul},
	{Name: "toward-beringia", Target: Beringia},
}

func (policy RoutePolicy) directed() bool { return policy.Target < RegionCount }

// DirectedSurvivalFloor is the fraction of the best available tile's
// attraction that a directed policy insists on before stepping toward its
// target. It is a property of the harness, not of the game: it encodes
// "populations disperse along habitable ground", which is what makes a
// reachability result a statement about the map rather than about how
// suicidally the policy was willing to march.
const DirectedSurvivalFloor = 0.50

// CampaignOutcome is what one (seed, policy) campaign produced. Extinction and
// dispersal failure are outcomes recorded here, not harness errors: §12 step 5e
// is explicit that they are expected possible results.
type CampaignOutcome struct {
	Seed               uint64
	Policy             RoutePolicy
	Result             CampaignResult
	TurnsCompleted     int
	EstablishedRegions uint16
	PeakBands          int
	PeakSapiens        uint64
	FinalSapiens       uint64
	// FirstDestinationTurn is the turn a destination region first held an
	// established sapiens band, or -1 if none ever did.
	FirstDestinationTurn int
	// TargetReachedTurn is the same for the policy's own target, or -1.
	TargetReachedTurn int
}

func destinationMask() uint16 {
	var mask uint16
	for _, region := range DestinationRegions {
		mask |= 1 << region
	}
	return mask
}

// regionCentroid returns the mean grid position of a region's land tiles. It is
// computed from the authored geography, so it is identical on every target and
// for every seed.
func regionCentroid(grid *Grid, region Region) (float64, float64, bool) {
	var sumX, sumY, count float64
	for id := range TileCount {
		tile, ok := grid.Tile(TileID(id))
		if !ok || !tile.Land || tile.Region != region {
			continue
		}
		sumX += float64(tile.X)
		sumY += float64(tile.Y)
		count++
	}
	if count == 0 {
		return 0, 0, false
	}
	return sumX / count, sumY / count, true
}

func squaredDistanceTo(grid *Grid, id TileID, centroidX, centroidY float64) float64 {
	tile, ok := grid.Tile(id)
	if !ok {
		return 0
	}
	dx := float64(tile.X) - centroidX
	dy := float64(tile.Y) - centroidY
	return float64(dx*dx) + float64(dy*dy)
}

// cheapestAvailableResearch picks a deterministic next technology: the lowest
// research cost among those whose prerequisites are met, ties broken by the
// enum's own order.
func cheapestAvailableResearch(state TechnologyState, prefer Technology) (Technology, bool) {
	if state.HasTarget {
		return state.Target, false
	}
	if prefer < TechCount && !state.Has(prefer) && state.PrerequisitesMet(prefer) {
		return prefer, true
	}
	best, found := Technology(0), false
	for technology := Technology(0); technology < TechCount; technology++ {
		if state.Has(technology) || !state.PrerequisitesMet(technology) {
			continue
		}
		if !found || ResearchCost[technology] < ResearchCost[best] {
			best, found = technology, true
		}
	}
	return best, found
}

// RunPolicyCampaign plays one complete campaign from turn 0 and returns what
// happened. It consumes no randomness beyond the world's own owned RNG, so the
// same (seed, policy) always produces the same CampaignOutcome.
func RunPolicyCampaign(seed uint64, policy RoutePolicy) (CampaignOutcome, error) {
	world, err := NewWorld(seed)
	if err != nil {
		return CampaignOutcome{}, err
	}
	outcome := CampaignOutcome{
		Seed: seed, Policy: policy,
		FirstDestinationTurn: -1, TargetReachedTurn: -1,
	}
	grid := world.Grid()
	var centroidX, centroidY float64
	haveCentroid := false
	if policy.directed() {
		centroidX, centroidY, haveCentroid = regionCentroid(grid, policy.Target)
	}
	prefer := TechCount
	if policy.Target == Sahul {
		// Wallacea is the only way into Sahul, and it is gated on this one
		// technology; without steering research the policy cannot arrive.
		prefer = CoastalNavigation
	}
	destinations := destinationMask()
	// The tile each band occupied on its previous turn. Without this a policy
	// that always takes the most attractive neighbour oscillates between two
	// adjacent tiles forever: the tile it just vacated is empty again, so it
	// scores best again. A memoryless hill-climber does not disperse, and the
	// resulting "the model cannot leave Africa" reading is an artifact of the
	// harness rather than a finding about the model.
	previous := map[BandID]TileID{}

	for world.Result() == CampaignOngoing {
		bands := world.Bands()
		if len(bands) > outcome.PeakBands {
			outcome.PeakBands = len(bands)
		}
		var sapiens uint64
		for _, band := range bands {
			if band.Species == HomoSapiens {
				sapiens += uint64(band.Population)
			}
		}
		if sapiens > outcome.PeakSapiens {
			outcome.PeakSapiens = sapiens
		}

		for _, band := range bands {
			if band.Species != HomoSapiens {
				continue
			}
			if technology, ok := cheapestAvailableResearch(band.Technology, prefer); ok {
				_ = world.Research(band.ID, technology, true)
			}
			candidates := withoutBacktrack(world.MigrationCandidates(band.ID), previous[band.ID])
			if len(candidates) == 0 {
				continue
			}
			// Split first when the band is under enough pressure to be allowed
			// to: two bands cover two regions, and establishment is per region.
			if len(world.Bands()) < MaxBands && world.BandStress(band.ID) > SplitStressThreshold {
				if destination, ok := bestOrdinary(candidates); ok {
					if err := world.Split(band.ID, destination, true); err == nil {
						continue
					}
				}
			}
			choice, ok := policy.choose(grid, candidates, band, centroidX, centroidY, haveCentroid)
			if ok {
				previous[band.ID] = band.TileID
				_ = world.QueueMigration(band.ID, choice, true)
			}
		}

		if err := world.AdvanceTurn(); err != nil {
			return outcome, err
		}
		outcome.TurnsCompleted = world.Turn()
		established := world.EstablishedRegions()
		outcome.EstablishedRegions = established
		if outcome.FirstDestinationTurn < 0 && established&destinations != 0 {
			outcome.FirstDestinationTurn = world.Turn()
		}
		if outcome.TargetReachedTurn < 0 && policy.directed() && established&(1<<policy.Target) != 0 {
			outcome.TargetReachedTurn = world.Turn()
		}
	}

	outcome.Result = world.Result()
	for _, band := range world.Bands() {
		if band.Species == HomoSapiens {
			outcome.FinalSapiens += uint64(band.Population)
		}
	}
	return outcome, nil
}

// withoutBacktrack drops the tile a band occupied last turn, unless doing so
// would leave it nowhere to go.
func withoutBacktrack(candidates []MigrationCandidate, previous TileID) []MigrationCandidate {
	kept := make([]MigrationCandidate, 0, len(candidates))
	for _, candidate := range candidates {
		if candidate.TileID != previous {
			kept = append(kept, candidate)
		}
	}
	if len(kept) == 0 {
		return candidates
	}
	return kept
}

// bestOrdinary returns the most attractive non-passage destination, which is
// where a split descendant is placed.
func bestOrdinary(candidates []MigrationCandidate) (TileID, bool) {
	best, found := MigrationCandidate{}, false
	for _, candidate := range candidates {
		if candidate.RequiresPassage || candidate.EcologicalK <= 0 {
			continue
		}
		if !found || better(candidate, best) {
			best, found = candidate, true
		}
	}
	return best.TileID, found
}

// better orders candidates by attraction, breaking ties on tile ID so that the
// choice never depends on slice order.
func better(candidate, incumbent MigrationCandidate) bool {
	if candidate.Attraction != incumbent.Attraction {
		return candidate.Attraction > incumbent.Attraction
	}
	return candidate.TileID < incumbent.TileID
}

// choose applies the policy's rule. A directed policy takes the most attractive
// move among those that strictly reduce distance to its target; if no move does,
// it falls back to the reference rule and survives in place rather than walking
// into a worse tile for the sake of the heading.
func (policy RoutePolicy) choose(grid *Grid, candidates []MigrationCandidate, band Band, centroidX, centroidY float64, haveCentroid bool) (TileID, bool) {
	reference, referenceFound := MigrationCandidate{}, false
	for _, candidate := range candidates {
		if candidate.EcologicalK <= 0 {
			continue
		}
		if !referenceFound || better(candidate, reference) {
			reference, referenceFound = candidate, true
		}
	}
	if !policy.directed() || !haveCentroid {
		return reference.TileID, referenceFound
	}
	current := squaredDistanceTo(grid, band.TileID, centroidX, centroidY)
	closer, closerFound := MigrationCandidate{}, false
	var closerDistance float64
	// A heading is not a reason to walk into a tile that will kill the band. A
	// directed policy advances only into ground at least half as attractive as
	// the best available; below that it holds position and survives instead.
	// Without this floor every directed policy marches its bands into desert
	// and goes extinct, which says nothing about whether the route exists.
	survivalFloor := float64(DirectedSurvivalFloor * reference.Attraction)
	for _, candidate := range candidates {
		if candidate.EcologicalK <= 0 || candidate.Attraction < survivalFloor {
			continue
		}
		distance := squaredDistanceTo(grid, candidate.TileID, centroidX, centroidY)
		if distance >= current {
			continue
		}
		if !closerFound || distance < closerDistance ||
			(distance == closerDistance && better(candidate, closer)) {
			closer, closerDistance, closerFound = candidate, distance, true
		}
	}
	if closerFound {
		return closer.TileID, true
	}
	return reference.TileID, referenceFound
}
