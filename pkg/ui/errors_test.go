package ui

import (
	"errors"
	"strings"
	"testing"

	"github.com/adsouza/africa2ice/pkg/gameapi"
)

// The contract is enumerated from gameapi rather than retyped here. A retyped
// list is one a new error code can be added without, leaving the omission to
// surface as a raw code string in front of a player.
func TestPlayerErrorCopyCoversEveryPublicErrorCode(t *testing.T) {
	codes := gameapi.ErrorCodes()
	if len(codes) == 0 {
		t.Fatal("gameapi reports no error codes")
	}
	for _, code := range codes {
		message := ErrorMessage(&gameapi.GameError{Code: code, Message: "internal wording"})
		if message == "" || message == string(code) || message == "internal wording" {
			t.Fatalf("code %q has non-player copy %q", code, message)
		}
	}
	// Copy for a code gameapi no longer publishes is dead weight that will
	// outlive whatever removed it.
	published := make(map[gameapi.ErrorCode]bool, len(codes))
	for _, code := range codes {
		published[code] = true
	}
	for code := range playerErrorMessages {
		if !published[code] {
			t.Fatalf("player copy for %q, which is not a public error code", code)
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
