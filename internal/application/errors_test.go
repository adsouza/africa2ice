package application

import (
	"errors"
	"fmt"
	"testing"

	"github.com/adsouza/africa2ice/internal/domain"
	"github.com/adsouza/africa2ice/pkg/gameapi"
)

// The application boundary must preserve refusal categories even when domain
// operations add context, so clients can select the correct recovery action.
func TestDomainErrorContract(t *testing.T) {
	cases := []struct {
		err  error
		code gameapi.ErrorCode
	}{
		{domain.ErrBandNotFound, gameapi.ErrBandNotFound},
		{domain.ErrComputerControlledBand, gameapi.ErrComputerControlledBand},
		{domain.ErrInvalidAssignment, gameapi.ErrInvalidAssignment},
		{domain.ErrSpatialActionUsed, gameapi.ErrSpatialActionUsed},
		{domain.ErrInvalidMigration, gameapi.ErrInvalidMigration},
		{domain.ErrSplitStressTooLow, gameapi.ErrSplitStressTooLow},
		{domain.ErrSplitPopulationTooLow, gameapi.ErrSplitPopulationTooLow},
		{domain.ErrSplitDestinationNotAdjacent, gameapi.ErrSplitDestinationNotAdjacent},
		{domain.ErrSplitDestinationUninhabitable, gameapi.ErrSplitDestinationUninhabitable},
		{domain.ErrSplitDestinationUnexplored, gameapi.ErrSplitDestinationUnexplored},
		{domain.ErrBandLimitReached, gameapi.ErrBandLimitReached},
		{domain.ErrBandIDExhausted, gameapi.ErrBandIDExhausted},
		{domain.ErrMissingTechnologyPrerequisite, gameapi.ErrMissingTechnologyPrerequisite},
		{domain.ErrTechnologyAlreadyAcquired, gameapi.ErrTechnologyAlreadyAcquired},
		{domain.ErrInvalidInterbreedTarget, gameapi.ErrInvalidInterbreedTarget},
		{domain.ErrCampaignComplete, gameapi.ErrCampaignComplete},
		{errors.New("unexpected failure"), gameapi.ErrInvalidCommand},
	}
	for _, tc := range cases {
		t.Run(tc.err.Error(), func(t *testing.T) {
			for _, err := range []error{tc.err, fmt.Errorf("command refused: %w", tc.err)} {
				got := mapDomainError(err)
				if got.Code != tc.code || got.Message != err.Error() {
					t.Fatalf("mapped error = %#v, want code %v and message %q", got, tc.code, err.Error())
				}
				if domainErrorCode(err) != got.Code {
					t.Fatal("projection and command error classification differ")
				}
			}
		})
	}
}
