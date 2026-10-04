package app

import (
	"testing"

	"github.com/adsouza/africa2ice/internal/adapters/storage"
	"github.com/adsouza/africa2ice/pkg/ui"
)

// A blocked upgrade or a lost connection makes every later save fail with a
// generic storage error. The availability notice is the only place the player
// learns why and what to do, so each transition must raise exactly its own.
func TestStorageAvailabilityTransitionsRaiseTheirNotices(t *testing.T) {
	game := newGameWithPresentation(&gameStub{frame: migrationPreviewFrame()}, &soundRecorder{}, &settingsStoreStub{})
	availability := storage.AvailabilityReady
	game.storageAvailability = func() storage.Availability { return availability }
	steps := []struct {
		name    string
		to      storage.Availability
		notice  string
		isError bool
	}{
		{"ready at startup says nothing", storage.AvailabilityReady, "", false},
		{"blocked upgrade", storage.AvailabilityUpgradeBlocked, storageUpgradeBlockedNotice, true},
		{"unchanged says nothing again", storage.AvailabilityUpgradeBlocked, "", false},
		{"blocked upgrade recovers", storage.AvailabilityReady, storageAvailableAgainNotice, false},
		{"connection taken away", storage.AvailabilityReloadRequired, storageReloadRequiredNotice, true},
	}
	for _, step := range steps {
		game.toasts = ui.ToastManager{}
		availability = step.to
		game.pollStorageAvailability()
		toast, shown := game.toasts.Current()
		if step.notice == "" {
			if shown {
				t.Fatalf("%s: raised %q", step.name, toast.Message)
			}
			continue
		}
		if !shown || toast.Message != step.notice || toast.Error != step.isError {
			t.Fatalf("%s: toast %#v (shown %t), want %q error=%t", step.name, toast, shown, step.notice, step.isError)
		}
	}
}

// Stores with no standing condition, the desktop file store among them,
// leave the hook nil, and polling must not touch the toast queue.
func TestStorageAvailabilityIsSilentWithoutAReporter(t *testing.T) {
	game := newGameWithPresentation(&gameStub{frame: migrationPreviewFrame()}, &soundRecorder{}, &settingsStoreStub{})
	game.toasts = ui.ToastManager{}
	game.pollStorageAvailability()
	if toast, shown := game.toasts.Current(); shown {
		t.Fatalf("no reporter raised %q", toast.Message)
	}
}
