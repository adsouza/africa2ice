package verification

import (
	"testing"

	"github.com/adsouza/africa2ice/pkg/gameapi"
)

func TestRouteDistancesDoNotCrossDiscoveredEscarpments(t *testing.T) {
	frame := &gameapi.Frame{
		Tiles: []gameapi.Tile{
			{ID: 0, X: 0, Y: 0, Land: true, EcologicalK: 1, Region: gameapi.EastAfrica},
			{ID: 1, X: 1, Y: 0, Land: true, EcologicalK: 1, Region: gameapi.SouthAsia},
		},
		Escarpments: []gameapi.Escarpment{{First: 0, Second: 1}},
	}
	distances := routeDistances(frame, gameapi.SouthAsia, false)
	if distances[0] != unreachableDistance || distances[1] != 0 {
		t.Fatalf("distances across escarpment = %v", distances)
	}
}
