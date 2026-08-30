package domain

import "errors"

var (
	ErrInvalidValue                  = errors.New("invalid domain value")
	ErrInvalidTurn                   = errors.New("turn outside campaign")
	ErrInvalidCoordinate             = errors.New("coordinate outside world")
	ErrBandNotFound                  = errors.New("band not found")
	ErrComputerControlledBand        = errors.New("band is computer controlled")
	ErrInvalidAssignment             = errors.New("assignment must total 10000 basis points")
	ErrSpatialActionUsed             = errors.New("spatial action already used")
	ErrInvalidMigration              = errors.New("invalid migration")
	ErrSplitStressTooLow             = errors.New("stress is too low to split")
	ErrSplitPopulationTooLow         = errors.New("population is too low to split")
	ErrSplitDestinationNotAdjacent   = errors.New("split destination is not adjacent land")
	ErrSplitDestinationUninhabitable = errors.New("split destination is uninhabitable")
	ErrBandLimitReached              = errors.New("band limit reached")
	ErrBandIDExhausted               = errors.New("band identifier exhausted")
	ErrMissingTechnologyPrerequisite = errors.New("missing technology prerequisite")
	ErrTechnologyAlreadyAcquired     = errors.New("technology already acquired")
	ErrInvalidInterbreedTarget       = errors.New("invalid interbreeding target")
	ErrCampaignComplete              = errors.New("campaign is complete")
)
