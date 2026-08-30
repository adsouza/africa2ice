package gameapi

type ErrorCode string

const (
	ErrInvalidCommand                ErrorCode = "invalid_command"
	ErrInvalidAssignment             ErrorCode = "invalid_assignment"
	ErrBandNotFound                  ErrorCode = "band_not_found"
	ErrComputerControlledBand        ErrorCode = "computer_controlled_band"
	ErrSpatialActionUsed             ErrorCode = "spatial_action_used"
	ErrInvalidMigration              ErrorCode = "invalid_migration"
	ErrSplitStressTooLow             ErrorCode = "split_stress_too_low"
	ErrSplitPopulationTooLow         ErrorCode = "split_population_too_low"
	ErrSplitDestinationNotAdjacent   ErrorCode = "split_destination_not_adjacent"
	ErrSplitDestinationUninhabitable ErrorCode = "split_destination_uninhabitable"
	ErrBandLimitReached              ErrorCode = "band_limit_reached"
	ErrBandIDExhausted               ErrorCode = "band_id_exhausted"
	ErrMissingTechnologyPrerequisite ErrorCode = "missing_technology_prerequisite"
	ErrTechnologyAlreadyAcquired     ErrorCode = "technology_already_acquired"
	ErrInvalidInterbreedTarget       ErrorCode = "invalid_interbreed_target"
	ErrCampaignComplete              ErrorCode = "campaign_complete"
	ErrStoragePending                ErrorCode = "storage_pending"
	ErrStorageReadOnly               ErrorCode = "storage_read_only"
	ErrStorageFailure                ErrorCode = "storage_failure"
	ErrInvalidSave                   ErrorCode = "invalid_save"
	ErrIncompatibleSave              ErrorCode = "incompatible_save"
)

type GameError struct {
	Code    ErrorCode
	Message string
}

func (e *GameError) Error() string {
	if e == nil {
		return ""
	}
	if e.Message != "" {
		return e.Message
	}
	return string(e.Code)
}
