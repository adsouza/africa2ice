package ui

import (
	"errors"
	"fmt"

	"github.com/adsouza/africa2ice/pkg/gameapi"
)

var playerErrorMessages = map[gameapi.ErrorCode]string{
	gameapi.ErrInvalidCommand:                "That action is not available right now.",
	gameapi.ErrInvalidAssignment:             "Work assignments must include all five roles and total exactly 100%.",
	gameapi.ErrBandNotFound:                  "That band no longer exists in the current campaign.",
	gameapi.ErrComputerControlledBand:        "That archaic band is computer-controlled; select a Homo sapiens band instead.",
	gameapi.ErrSpatialActionUsed:             "This band has already used its migration, split, or interbreeding action this turn.",
	gameapi.ErrInvalidMigration:              "That destination is not currently reachable by this band.",
	gameapi.ErrSplitStressTooLow:             "This band is not under enough pressure to split.",
	gameapi.ErrSplitPopulationTooLow:         fmt.Sprintf("This band needs at least %d people to split into two viable bands.", gameapi.MinSplitSourcePopulation),
	gameapi.ErrSplitDestinationNotAdjacent:   "A new band can only establish on an eligible neighboring tile.",
	gameapi.ErrSplitDestinationUninhabitable: "The proposed new band cannot survive on that destination tile.",
	gameapi.ErrSplitDestinationUnexplored:    "That area is unexplored; scout it before settling a new band there.",
	gameapi.ErrBandLimitReached:              "The campaign has reached the 256-band limit; a band must disappear before another can split.",
	gameapi.ErrBandIDExhausted:               "No additional band identifiers remain in this campaign.",
	gameapi.ErrMissingTechnologyPrerequisite: "That research is locked until its prerequisite technology has been learned.",
	gameapi.ErrTechnologyAlreadyAcquired:     "This band has already learned that technology.",
	gameapi.ErrInvalidInterbreedTarget:       "That archaic band is not an eligible interbreeding partner on the current tile.",
	gameapi.ErrCampaignComplete:              "The campaign has ended; start a new campaign to continue playing.",
	gameapi.ErrStoragePending:                "Please wait for the current save or load operation to finish.",
	gameapi.ErrStorageReadOnly:               "Saving is unavailable because this storage session is read-only.",
	gameapi.ErrStorageFailure:                "Storage failed; the current in-memory campaign is unchanged.",
	gameapi.ErrInvalidSave:                   "That save is incomplete or damaged and cannot be loaded.",
	gameapi.ErrIncompatibleSave:              "That save was created by an incompatible game version.",
}

// ErrorMessage turns the public typed error contract into actionable player
// copy without importing the domain. Unknown adapter errors retain their text
// so diagnostics are not replaced by a generic failure.
func ErrorMessage(err error) string {
	if err == nil {
		return ""
	}
	var gameError *gameapi.GameError
	if errors.As(err, &gameError) {
		if message, ok := playerErrorMessages[gameError.Code]; ok {
			return message
		}
	}
	return err.Error()
}
