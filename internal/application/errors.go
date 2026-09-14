package application

import (
	"errors"

	"github.com/adsouza/africa2ice/internal/domain"
	"github.com/adsouza/africa2ice/pkg/gameapi"
)

// domainErrorCode classifies a domain error without building a GameError.
// Frame projection asks this question once per band per frame and keeps
// only the code, so the error value and its message must not be allocated.
func domainErrorCode(err error) gameapi.ErrorCode {
	code := gameapi.ErrInvalidCommand
	switch {
	case errors.Is(err, domain.ErrBandNotFound):
		code = gameapi.ErrBandNotFound
	case errors.Is(err, domain.ErrComputerControlledBand):
		code = gameapi.ErrComputerControlledBand
	case errors.Is(err, domain.ErrInvalidAssignment):
		code = gameapi.ErrInvalidAssignment
	case errors.Is(err, domain.ErrSpatialActionUsed):
		code = gameapi.ErrSpatialActionUsed
	case errors.Is(err, domain.ErrInvalidMigration):
		code = gameapi.ErrInvalidMigration
	case errors.Is(err, domain.ErrSplitStressTooLow):
		code = gameapi.ErrSplitStressTooLow
	case errors.Is(err, domain.ErrSplitPopulationTooLow):
		code = gameapi.ErrSplitPopulationTooLow
	case errors.Is(err, domain.ErrSplitDestinationNotAdjacent):
		code = gameapi.ErrSplitDestinationNotAdjacent
	case errors.Is(err, domain.ErrSplitDestinationUninhabitable):
		code = gameapi.ErrSplitDestinationUninhabitable
	case errors.Is(err, domain.ErrSplitDestinationUnexplored):
		code = gameapi.ErrSplitDestinationUnexplored
	case errors.Is(err, domain.ErrBandLimitReached):
		code = gameapi.ErrBandLimitReached
	case errors.Is(err, domain.ErrBandIDExhausted):
		code = gameapi.ErrBandIDExhausted
	case errors.Is(err, domain.ErrMissingTechnologyPrerequisite):
		code = gameapi.ErrMissingTechnologyPrerequisite
	case errors.Is(err, domain.ErrTechnologyAlreadyAcquired):
		code = gameapi.ErrTechnologyAlreadyAcquired
	case errors.Is(err, domain.ErrInvalidInterbreedTarget):
		code = gameapi.ErrInvalidInterbreedTarget
	case errors.Is(err, domain.ErrCampaignComplete):
		code = gameapi.ErrCampaignComplete
	}
	return code
}

func mapDomainError(err error) *gameapi.GameError {
	return &gameapi.GameError{Code: domainErrorCode(err), Message: err.Error()}
}
