package verification

import (
	"testing"

	"github.com/adsouza/africa2ice/pkg/gameapi"
)

// frameWithTwoDestinations puts one established band in Frangistan and a larger
// one in Sahul, both newly established on the same turn.
func frameWithTwoDestinations(turn int) *gameapi.Frame {
	frame := &gameapi.Frame{
		Turn:                      turn,
		SapiensEstablishedRegions: []gameapi.Region{gameapi.Frangistan, gameapi.Sahul},
		Tiles: []gameapi.Tile{
			{ID: 0, Region: gameapi.Frangistan, Land: true},
			{ID: 1, Region: gameapi.Sahul, Land: true},
			{ID: 2, Region: gameapi.Sahul, Land: true},
		},
		Bands: []gameapi.Band{
			{ID: 1, Species: gameapi.HomoSapiens, TileID: 0, Population: 30},
			{ID: 2, Species: gameapi.HomoSapiens, TileID: 1, Population: 70},
			// Below MinEstablishedBand: present in a destination but not established.
			{ID: 3, Species: gameapi.HomoSapiens, TileID: 2, Population: 200},
		},
	}
	frame.Bands[2].Population = gameapi.MinEstablishedBand - 1
	return frame
}

// Two destinations establishing on one turn used to be resolved by ranging over
// a map, so which one supplied firstDestinationPopulation was random. That value
// reaches the checkpoint record the cross-target job compares byte for byte.
func TestFirstDestinationMarginIsOrderIndependent(t *testing.T) {
	var initial destinationSet
	for attempt := range 200 {
		margins := runMargins{firstDestinationTurn: -1}
		margins.observeDestinations(frameWithTwoDestinations(7), initial)
		if margins.firstDestinationTurn != 7 {
			t.Fatalf("attempt %d: turn = %d, want 7", attempt, margins.firstDestinationTurn)
		}
		if margins.firstDestinationPopulation != 70 {
			t.Fatalf("attempt %d: population = %d, want the largest established band (70)",
				attempt, margins.firstDestinationPopulation)
		}
	}
}

func TestFirstDestinationIgnoresAlreadyEstablishedRegions(t *testing.T) {
	frame := frameWithTwoDestinations(7)
	initial := establishedDestinations(frame)

	margins := runMargins{firstDestinationTurn: -1}
	margins.observeDestinations(frame, initial)
	if margins.firstDestinationTurn != -1 || margins.firstDestinationPopulation != 0 {
		t.Fatalf("no new destination should record a margin: %+v", margins)
	}

	// Only Sahul is new relative to a start that already held Frangistan.
	var frangistanOnly destinationSet
	for index, region := range destinationRegions {
		frangistanOnly[index] = region == gameapi.Frangistan
	}
	margins = runMargins{firstDestinationTurn: -1}
	margins.observeDestinations(frame, frangistanOnly)
	if margins.firstDestinationPopulation != 70 {
		t.Fatalf("population = %d, want the Sahul band (70)", margins.firstDestinationPopulation)
	}
}
