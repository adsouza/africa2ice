package verification

import (
	"errors"
	"fmt"

	"github.com/adsouza/africa2ice/internal/application"
	"github.com/adsouza/africa2ice/pkg/gameapi"
)

const MaxTurns = 400

var destinationRegions = [...]gameapi.Region{
	gameapi.Frangistan, gameapi.SouthAsia, gameapi.YellowRiverBasin, gameapi.Sahul, gameapi.Beringia,
}

// ReferenceRun performs no I/O, reads no wall clock, and reaches the aggregate
// only through gameapi.Game. Records are emitted at turn 0 and every 100 turns,
// including the requested terminal turn when it is a multiple of 100.
func ReferenceRun(seed uint64, turns int, policyName string) ([]CheckpointRecord, error) {
	_, records, err := PrepareGame(seed, turns, policyName)
	return records, err
}

// PrepareGame drives the same deterministic harness and returns the advanced
// port for display-required verification such as the desktop screenshot mode.
func PrepareGame(seed uint64, turns int, policyName string) (gameapi.Game, []CheckpointRecord, error) {
	if turns < 0 || turns > MaxTurns {
		return nil, nil, fmt.Errorf("turns must be in [0,%d]", MaxTurns)
	}
	policy, err := ParsePolicy(policyName)
	if err != nil {
		return nil, nil, err
	}
	service, err := application.NewGameService(seed)
	if err != nil {
		return nil, nil, err
	}
	records, err := Run(service, seed, turns, policy)
	if err != nil {
		return nil, nil, err
	}
	return service, records, nil
}

// Run is the injected-port form used by architecture and behavior fixtures.
func Run(game gameapi.Game, seed uint64, turns int, policy Policy) ([]CheckpointRecord, error) {
	frame, err := game.Snapshot()
	if err != nil {
		return nil, err
	}
	margins := runMargins{firstDestinationTurn: -1}
	initialDestinations := establishedDestinations(frame)
	records := make([]CheckpointRecord, 0, turns/100+1)
	if record, checkpointErr := checkpoint(game, frame, seed, policy, margins); checkpointErr != nil {
		return nil, checkpointErr
	} else {
		records = append(records, record)
	}
	for frame.Turn < turns && frame.CampaignResult == gameapi.Ongoing {
		for _, command := range researchCommands(frame, policy) {
			frame, err = game.Apply(command)
			if err != nil {
				return nil, fmt.Errorf("turn %d policy %s command %T: %w", frameTurn(frame), policy.Name, command, err)
			}
		}
		for _, command := range spatialCommands(frame, policy) {
			next, applyErr := game.Apply(command)
			if applyErr != nil {
				if optionalSplitRejection(command, applyErr) {
					continue
				}
				return nil, fmt.Errorf("turn %d policy %s command %T: %w", frameTurn(frame), policy.Name, command, applyErr)
			}
			frame = next
		}
		frame, err = game.EndTurn()
		if err != nil {
			return nil, fmt.Errorf("turn %d policy %s end: %w", frameTurn(frame), policy.Name, err)
		}
		if margins.firstDestinationTurn < 0 {
			margins.observeDestinations(frame, initialDestinations)
		}
		if frame.Turn%100 == 0 || frame.Turn == turns || frame.CampaignResult != gameapi.Ongoing {
			record, checkpointErr := checkpoint(game, frame, seed, policy, margins)
			if checkpointErr != nil {
				return nil, checkpointErr
			}
			records = append(records, record)
		}
	}
	return records, nil
}

func optionalSplitRejection(command gameapi.Command, err error) bool {
	if _, ok := command.(gameapi.SplitBand); !ok {
		return false
	}
	var gameErr *gameapi.GameError
	if !errors.As(err, &gameErr) {
		return false
	}
	switch gameErr.Code {
	case gameapi.ErrSplitStressTooLow, gameapi.ErrSplitPopulationTooLow, gameapi.ErrBandLimitReached:
		return true
	default:
		return false
	}
}

func frameTurn(frame *gameapi.Frame) int {
	if frame == nil {
		return -1
	}
	return frame.Turn
}

// destinationSet reports which destinations hold an established sapiens band,
// indexed by position in destinationRegions. It is a fixed-size array rather
// than a map so that every consumer walks the destinations in one declared
// order: firstDestinationPopulation reaches the cross-target checkpoint record,
// and Go randomizes map iteration.
type destinationSet [len(destinationRegions)]bool

func establishedDestinations(frame *gameapi.Frame) destinationSet {
	var result destinationSet
	if frame == nil {
		return result
	}
	for _, established := range frame.SapiensEstablishedRegions {
		for index, destination := range destinationRegions {
			if established == destination {
				result[index] = true
			}
		}
	}
	return result
}

// observeDestinations records the turn on which any destination region first
// became established, and the largest established sapiens band standing in one
// of the regions newly established on that turn. It takes the maximum over
// every newly established destination and applies the same MinEstablishedBand
// floor as domain.RunPolicyCampaign, so both drivers report one number.
func (margins *runMargins) observeDestinations(frame *gameapi.Frame, initial destinationSet) {
	current := establishedDestinations(frame)
	var newly destinationSet
	found := false
	for index := range current {
		if current[index] && !initial[index] {
			newly[index], found = true, true
		}
	}
	if !found {
		return
	}
	margins.firstDestinationTurn = frame.Turn
	for _, band := range frame.Bands {
		if band.Species != gameapi.HomoSapiens || band.Population < gameapi.MinEstablishedBand {
			continue
		}
		region := tileRegion(frame, band.TileID)
		for index, destination := range destinationRegions {
			if newly[index] && region == destination && band.Population > margins.firstDestinationPopulation {
				margins.firstDestinationPopulation = band.Population
			}
		}
	}
}

// DumpMap returns the fixed grid as a biome layer followed by a region layer.
// It is intentionally derived from the same public frame the renderer uses.
func DumpMap(seed uint64) (string, error) {
	service, err := application.NewGameService(seed)
	if err != nil {
		return "", err
	}
	frame, err := service.Snapshot()
	if err != nil {
		return "", err
	}
	return EncodeMap(frame), nil
}
