package domain

import "testing"

func TestReferenceRouteHoldsFinalApproachForPopulationReserve(t *testing.T) {
	current, destination := TileID(1), TileID(2)
	route := &temporalRoute{
		target:    Frangistan,
		costs:     make([]uint32, 2*TileCount),
		stepCosts: make([]uint8, 2*TileCount),
	}
	for index := range route.costs {
		route.costs[index] = unreachableRouteCost
	}
	route.costs[int(current)] = 2
	route.costs[TileCount+int(current)] = 2
	route.costs[TileCount+int(destination)] = 0
	route.stepCosts[TileCount+int(current)] = 1
	route.stepCosts[TileCount+int(destination)] = 1
	candidates := []MigrationCandidate{{TileID: destination, EcologicalK: 100, Attraction: 1}}

	belowReserve := Band{TileID: current, Population: referenceRouteDeparturePopulation - 1}
	if choice, ok := ReferenceRoutePolicy.choose(candidates, belowReserve, route, 0, true); ok {
		t.Fatalf("reference policy chose %d below its final-approach reserve", choice)
	}
	atReserve := Band{TileID: current, Population: referenceRouteDeparturePopulation}
	if choice, ok := ReferenceRoutePolicy.choose(candidates, atReserve, route, 0, true); !ok || choice != destination {
		t.Fatalf("reference policy at reserve chose (%d, %t), want (%d, true)", choice, ok, destination)
	}
	if choice, ok := ReferenceRoutePolicy.choose(candidates, belowReserve, route, 0, false); !ok || choice != destination {
		t.Fatalf("reference policy after first establishment chose (%d, %t), want (%d, true)", choice, ok, destination)
	}

	directed := DirectedRoutePolicies[0]
	if choice, ok := directed.choose(candidates, belowReserve, route, 0, false); !ok || choice != destination {
		t.Fatalf("directed policy chose (%d, %t), want (%d, true)", choice, ok, destination)
	}
}
