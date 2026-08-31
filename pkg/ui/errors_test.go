package ui

import (
	"errors"
	"strings"
	"testing"

	"github.com/adsouza/africa2ice/pkg/gameapi"
)

func TestPlayerErrorCopyCoversEveryPublicErrorCode(t *testing.T) {
	codes := []gameapi.ErrorCode{
		gameapi.ErrInvalidCommand, gameapi.ErrInvalidAssignment, gameapi.ErrBandNotFound,
		gameapi.ErrComputerControlledBand, gameapi.ErrSpatialActionUsed, gameapi.ErrInvalidMigration,
		gameapi.ErrSplitStressTooLow, gameapi.ErrSplitPopulationTooLow, gameapi.ErrSplitDestinationNotAdjacent,
		gameapi.ErrSplitDestinationUninhabitable, gameapi.ErrBandLimitReached, gameapi.ErrBandIDExhausted,
		gameapi.ErrMissingTechnologyPrerequisite, gameapi.ErrTechnologyAlreadyAcquired, gameapi.ErrInvalidInterbreedTarget,
		gameapi.ErrCampaignComplete, gameapi.ErrStoragePending, gameapi.ErrStorageReadOnly,
		gameapi.ErrStorageFailure, gameapi.ErrInvalidSave, gameapi.ErrIncompatibleSave,
	}
	if len(playerErrorMessages) != len(codes) {
		t.Fatalf("player error messages = %d, public codes = %d", len(playerErrorMessages), len(codes))
	}
	for _, code := range codes {
		message := ErrorMessage(&gameapi.GameError{Code: code, Message: "internal wording"})
		if message == "" || message == string(code) || message == "internal wording" {
			t.Fatalf("code %q has non-player copy %q", code, message)
		}
	}
}

func TestRequiredPlayerErrorCopyNamesTheActionableConstraint(t *testing.T) {
	tests := []struct {
		code gameapi.ErrorCode
		want string
	}{
		{code: gameapi.ErrComputerControlledBand, want: "computer-controlled"},
		{code: gameapi.ErrBandLimitReached, want: "256-band limit"},
		{code: gameapi.ErrMissingTechnologyPrerequisite, want: "prerequisite"},
	}
	for _, test := range tests {
		if message := ErrorMessage(&gameapi.GameError{Code: test.code}); !strings.Contains(message, test.want) {
			t.Fatalf("code %q message %q does not contain %q", test.code, message, test.want)
		}
	}
}

func TestErrorMessagePreservesUnknownErrorsAndHandlesNil(t *testing.T) {
	if got := ErrorMessage(nil); got != "" {
		t.Fatalf("nil error = %q", got)
	}
	if got := ErrorMessage(errors.New("graphics device lost")); got != "graphics device lost" {
		t.Fatalf("unknown error = %q", got)
	}
	if got := ErrorMessage(&gameapi.GameError{Code: "future_error", Message: "future detail"}); got != "future detail" {
		t.Fatalf("future game error = %q", got)
	}
}
