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
	initialDestinations := destinationSet(frame)
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
			newDestinations := destinationSet(frame)
			for region := range newDestinations {
				if initialDestinations[region] {
					continue
				}
				margins.firstDestinationTurn = frame.Turn
				for _, band := range frame.Bands {
					if band.Species == gameapi.HomoSapiens && tileRegion(frame, band.TileID) == region && band.Population > margins.firstDestinationPopulation {
						margins.firstDestinationPopulation = band.Population
					}
				}
				break
			}
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

func destinationSet(frame *gameapi.Frame) map[gameapi.Region]bool {
	result := make(map[gameapi.Region]bool, len(destinationRegions))
	if frame == nil {
		return result
	}
	for _, established := range frame.SapiensEstablishedRegions {
		for _, destination := range destinationRegions {
			if established == destination {
				result[established] = true
			}
		}
	}
	return result
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
