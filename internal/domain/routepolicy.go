package domain

import (
	"fmt"
	"sync"
)

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

// RoutePolicy steers one route-leading sapiens band toward a destination. The
// reference policy uses the nearest destination while the directed policies
// name their destination explicitly.
type RoutePolicy struct {
	Name   string
	Target Region
}

// ReferenceRoutePolicy is the route-neutral proof that outward dispersal and
// turn-400 survival can coexist; the directed policies prove each destination.
var ReferenceRoutePolicy = RoutePolicy{Name: "reference", Target: RegionCount}

var survivalResearchPriority = [TechCount]Technology{
	Firecraft, HaftedTools, Campcraft, PlantKnowledge, MedicinalKnowledge,
	TailoredClothing, CordageAndNets, Trapping, CoastalNavigation,
}

var sahulResearchPriority = [TechCount]Technology{
	Firecraft, HaftedTools, CordageAndNets, CoastalNavigation, Campcraft,
	PlantKnowledge, MedicinalKnowledge, TailoredClothing, Trapping,
}

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
	RegionsVisited     uint16
	TargetPeakBand     Population
	RoutedGrowth       float64
	RoutedMortality    MortalityReport
	RoutedDeficitTurns int
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

const unreachableRouteCost = ^uint32(0)

type temporalRoute struct {
	target    Region
	costs     []uint32
	stepCosts []uint8
}

func (route *temporalRoute) cost(turn int, tile TileID) uint32 {
	if route == nil || turn < 0 || turn > MaxCampaignTurn || tile >= TileCount {
		return unreachableRouteCost
	}
	return route.costs[turn*TileCount+int(tile)]
}

func (route *temporalRoute) stepCost(turn int, tile TileID) uint32 {
	if route == nil || turn < 0 || turn > MaxCampaignTurn || tile >= TileCount {
		return unreachableRouteCost
	}
	cost := route.stepCosts[turn*TileCount+int(tile)]
	if cost == 0 {
		return unreachableRouteCost
	}
	return uint32(cost)
}

type temporalRouteCache struct {
	once  sync.Once
	route *temporalRoute
	err   error
}

var temporalRoutes [len(DestinationRegions)]temporalRouteCache

func destinationIndex(target Region) (int, bool) {
	for index, region := range DestinationRegions {
		if region == target {
			return index, true
		}
	}
	return 0, false
}

func temporalRouteFor(grid *Grid, target Region) (*temporalRoute, error) {
	index, ok := destinationIndex(target)
	if !ok {
		return nil, fmt.Errorf("route target %v is not a destination", target)
	}
	cache := &temporalRoutes[index]
	cache.once.Do(func() {
		cache.route, cache.err = buildTemporalRoute(grid, target)
	})
	return cache.route, cache.err
}

// buildTemporalRoute solves the authored campaign as a time-expanded graph.
// A state can wait, take an ordinary land edge, or take a named passage on the
// next turn only when the destination will be habitable then. The plan includes
// technology-gated passages so it can approach their endpoints; the live
// MigrationCandidates check remains authoritative until the technology exists.
func buildTemporalRoute(grid *Grid, target Region) (*temporalRoute, error) {
	habitable := make([][ExplorationWordCount]uint64, MaxCampaignTurn+1)
	beringiaOpen := make([]bool, MaxCampaignTurn+1)
	stepCosts := make([]uint8, (MaxCampaignTurn+1)*TileCount)
	for turn := 0; turn <= MaxCampaignTurn; turn++ {
		habitat, climate, err := BuildHabitat(grid, 0, turn)
		if err != nil {
			return nil, err
		}
		beringiaOpen[turn] = BeringiaOpen(climate.LongTermTempOffset)
		for id := range TileCount {
			if habitat[id].BaselineK > 0 {
				habitable[turn][id/64] |= uint64(1) << (id % 64)
				switch capacity := habitat[id].BaselineK; {
				case capacity >= 100:
					stepCosts[turn*TileCount+id] = 1
				case capacity >= 75:
					stepCosts[turn*TileCount+id] = 2
				case capacity >= 50:
					stepCosts[turn*TileCount+id] = 4
				case capacity >= 25:
					stepCosts[turn*TileCount+id] = 8
				case capacity >= 10:
					stepCosts[turn*TileCount+id] = 16
				default:
					stepCosts[turn*TileCount+id] = 32
				}
			}
		}
	}
	isHabitable := func(turn int, tile TileID) bool {
		return habitable[turn][tile/64]&(uint64(1)<<(tile%64)) != 0
	}
	costs := make([]uint32, (MaxCampaignTurn+1)*TileCount)
	for index := range costs {
		costs[index] = unreachableRouteCost
	}
	route := &temporalRoute{target: target, costs: costs, stepCosts: stepCosts}
	for turn := MaxCampaignTurn; turn >= 0; turn-- {
		for id := range TileCount {
			tileID := TileID(id)
			geography, ok := grid.Tile(tileID)
			if !ok || !geography.Land || !isHabitable(turn, tileID) {
				continue
			}
			if geography.Region == target {
				route.costs[turn*TileCount+id] = 0
				continue
			}
			if turn == MaxCampaignTurn {
				continue
			}
			best := unreachableRouteCost
			consider := func(destination TileID) {
				// QueueMigration validates against the planning frame and turn
				// resolution validates again after phase-1 climate, so a planned
				// destination must be habitable on both sides of the boundary.
				if !isHabitable(turn, destination) || !isHabitable(turn+1, destination) {
					return
				}
				next := route.cost(turn+1, destination)
				step := route.stepCost(turn+1, destination)
				if next == unreachableRouteCost || step == unreachableRouteCost {
					return
				}
				candidate := step + next
				if candidate < best {
					best = candidate
				}
			}
			consider(tileID)
			for _, edge := range grid.OrdinaryEdges(tileID) {
				consider(edge.To)
			}
			for _, passage := range passageCatalog {
				destination, atEndpoint := passageDestination(passage, tileID)
				if atEndpoint && (!passage.ClimateGated || beringiaOpen[turn+1]) {
					consider(destination)
				}
			}
			if best != unreachableRouteCost {
				route.costs[turn*TileCount+id] = best
			}
		}
	}
	return route, nil
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

func routeResearch(state TechnologyState, target Region) (Technology, bool) {
	if state.HasTarget {
		return state.Target, false
	}
	priority := survivalResearchPriority
	if target == Sahul {
		priority = sahulResearchPriority
	}
	for _, technology := range priority {
		if !state.Has(technology) && state.PrerequisitesMet(technology) {
			return technology, true
		}
	}
	return 0, false
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
	routeTarget := policy.Target
	if !policy.directed() {
		// The reference campaign takes the nearest destination as its route-neutral
		// proof that outward dispersal is possible; the five directed policies
		// separately prove every named destination.
		routeTarget = Frangistan
	}
	route, err := temporalRouteFor(grid, routeTarget)
	if err != nil {
		return CampaignOutcome{}, err
	}
	destinations := destinationMask()
	// The tile each non-route band occupied on its previous turn. Without this,
	// local attraction can make those bands oscillate between adjacent tiles.
	previous := map[BandID]TileID{}

	for world.Result() == CampaignOngoing {
		bands := world.Bands()
		routedBandID := BandID(0)
		if route != nil {
			bestCost := unreachableRouteCost
			bestPopulation := Population(0)
			for _, band := range bands {
				if band.Species != HomoSapiens {
					continue
				}
				cost := route.cost(world.Turn(), band.TileID)
				if cost < bestCost || cost == bestCost && (band.Population > bestPopulation || band.Population == bestPopulation && (routedBandID == 0 || band.ID < routedBandID)) {
					routedBandID, bestCost, bestPopulation = band.ID, cost, band.Population
				}
			}
		}
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
			technology, selectResearch := cheapestAvailableResearch(band.Technology, TechCount)
			if band.ID == routedBandID {
				technology, selectResearch = routeResearch(band.Technology, routeTarget)
			}
			if selectResearch {
				if err := world.Research(band.ID, technology, true); err != nil {
					return outcome, fmt.Errorf("policy %s could not select research for band %d: %w", policy.Name, band.ID, err)
				}
			}
			if band.ID != routedBandID && world.BandStress(band.ID) <= SplitStressThreshold {
				continue
			}
			candidates := world.MigrationCandidates(band.ID)
			if prior, ok := previous[band.ID]; ok && band.ID != routedBandID {
				candidates = withoutBacktrack(candidates, prior)
			}
			if len(candidates) == 0 {
				continue
			}
			// The reference policy may split its non-route bands under pressure;
			// the route band remains whole so reaching a destination is not
			// manufactured by accepting a below-margin descendant.
			if !policy.directed() && band.ID != routedBandID && len(world.Bands()) < MaxBands && world.BandStress(band.ID) > SplitStressThreshold {
				if destination, ok := bestOrdinary(candidates); ok {
					if err := world.Split(band.ID, destination, true); err == nil {
						continue
					}
				}
			}
			if policy.directed() && band.ID != routedBandID {
				continue
			}
			bandRoute := route
			if band.ID != routedBandID {
				bandRoute = nil
			}
			choice, ok := policy.choose(candidates, band, bandRoute, world.Turn())
			if ok {
				previous[band.ID] = band.TileID
				if err := world.QueueMigration(band.ID, choice, true); err != nil {
					return outcome, fmt.Errorf("policy %s could not migrate band %d to tile %d: %w", policy.Name, band.ID, choice, err)
				}
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
		for _, band := range world.Bands() {
			if band.Species != HomoSapiens {
				continue
			}
			if band.ID == routedBandID {
				outcome.RoutedGrowth += band.LastOutcomeReport.Growth
				outcome.RoutedMortality.Starvation += band.LastMortality.Starvation
				outcome.RoutedMortality.Seasonal += band.LastMortality.Seasonal
				outcome.RoutedMortality.Chronic += band.LastMortality.Chronic
				outcome.RoutedMortality.Macro += band.LastMortality.Macro
				outcome.RoutedMortality.Acute += band.LastMortality.Acute
				if band.LastFoodReport.DeficitFU > 0 {
					outcome.RoutedDeficitTurns++
				}
			}
			geography, _ := grid.Tile(band.TileID)
			outcome.RegionsVisited |= 1 << geography.Region
			if policy.directed() && geography.Region == policy.Target && band.Population > outcome.TargetPeakBand {
				outcome.TargetPeakBand = band.Population
			}
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

// choose applies the policy's rule. A routed band takes the most attractive
// move among the habitable candidates that minimize its time-expanded route.
// If waiting is at least as fast, it holds position until climate or a named
// passage opens rather than wandering away from the route.
func (policy RoutePolicy) choose(candidates []MigrationCandidate, band Band, route *temporalRoute, turn int) (TileID, bool) {
	reference, referenceFound := MigrationCandidate{}, false
	for _, candidate := range candidates {
		if candidate.EcologicalK <= 0 {
			continue
		}
		if !referenceFound || better(candidate, reference) {
			reference, referenceFound = candidate, true
		}
	}
	if route == nil {
		return reference.TileID, referenceFound
	}
	geographyTarget := false
	if route.target < RegionCount {
		// distance zero is sufficient, but checking the authored region makes
		// the terminal behavior explicit even at turn 400.
		geographyTarget = route.cost(turn, band.TileID) == 0
	}
	if geographyTarget || turn >= MaxCampaignTurn {
		return 0, false
	}
	closer, closerFound := MigrationCandidate{}, false
	closerCost := unreachableRouteCost
	for _, candidate := range candidates {
		if candidate.EcologicalK <= 0 {
			continue
		}
		next := route.cost(turn+1, candidate.TileID)
		step := route.stepCost(turn+1, candidate.TileID)
		if next == unreachableRouteCost || step == unreachableRouteCost {
			continue
		}
		cost := step + next
		if !closerFound || cost < closerCost ||
			(cost == closerCost && better(candidate, closer)) {
			closer, closerCost, closerFound = candidate, cost, true
		}
	}
	waitNext := route.cost(turn+1, band.TileID)
	waitStep := route.stepCost(turn+1, band.TileID)
	waitCost := unreachableRouteCost
	if waitNext != unreachableRouteCost && waitStep != unreachableRouteCost {
		waitCost = waitStep + waitNext
	}
	if closerFound && closerCost <= waitCost {
		return closer.TileID, true
	}
	return 0, false
}
