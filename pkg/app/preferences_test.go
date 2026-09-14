package app

import (
	"errors"
	"testing"

	"github.com/adsouza/africa2ice/pkg/ui"
)

type preferenceStoreStub struct {
	settingsStoreStub
	writeErr error
}

func (s *preferenceStoreStub) BeginWrite(revision uint64, settings ui.UISettings) error {
	s.writes = append(s.writes, ui.UISettingsCompletion{Operation: ui.UISettingsWrite, Revision: revision, Settings: settings})
	return s.writeErr
}

func TestPreferencesIgnoreStaleCompletionsAndKeepLatestEdit(t *testing.T) {
	store := &preferenceStoreStub{}
	controller, err := newPreferenceController(store)
	if err != nil {
		t.Fatal(err)
	}
	loaded := ui.DefaultUISettings()
	loaded.MasterVolume = 0.7
	store.completions = []ui.UISettingsCompletion{{Operation: ui.UISettingsRead, Revision: 99, Settings: loaded}}
	if len(controller.poll()) != 0 || !controller.loading {
		t.Fatal("stale read installed preferences")
	}
	store.completions = []ui.UISettingsCompletion{{Operation: ui.UISettingsRead, Revision: 1, Settings: loaded}}
	if results := controller.poll(); len(results) != 1 || !results[0].installed || controller.value != loaded {
		t.Fatal("accepted read was not installed")
	}
	first := loaded
	first.MasterVolume = 0.4
	latest := first
	latest.MasterVolume = 0.2
	if err := controller.update(first); err != nil {
		t.Fatal(err)
	}
	if err := controller.update(latest); err != nil {
		t.Fatal(err)
	}
	store.completions = []ui.UISettingsCompletion{{Operation: ui.UISettingsWrite, Revision: 99, Settings: first}}
	controller.poll()
	if len(store.writes) != 1 || !controller.writeActive || controller.pending == nil {
		t.Fatal("stale completion released active write")
	}
	failed := store.writes[0]
	failed.Err = errors.New("disk unavailable")
	store.completions = []ui.UISettingsCompletion{failed}
	results := controller.poll()
	if len(results) != 1 || results[0].notice == "" || results[0].installed || len(store.writes) != 2 || store.writes[1].Settings != latest || controller.value != latest {
		t.Fatal("failed write lost latest queued preference or reinstalled old state")
	}
	store.completions = []ui.UISettingsCompletion{store.writes[1], failed}
	if len(controller.poll()) != 0 || controller.writeActive || controller.pending != nil || controller.value != latest {
		t.Fatal("stale failure changed settled preferences")
	}
}

func TestPreferencesRecoverAfterWriteCannotStart(t *testing.T) {
	store := &preferenceStoreStub{writeErr: errors.New("unavailable")}
	controller, err := newPreferenceController(store)
	if err != nil {
		t.Fatal(err)
	}
	store.completions = []ui.UISettingsCompletion{{Operation: ui.UISettingsRead, Revision: 1, Settings: ui.DefaultUISettings()}}
	controller.poll()
	settings := controller.value
	settings.Muted = true
	if err := controller.update(settings); err != store.writeErr || controller.writeActive || controller.value != settings {
		t.Fatal("failed write blocked or rolled back live edit")
	}
	store.writeErr = nil
	settings.MasterVolume = 0.3
	if err := controller.update(settings); err != nil || !controller.writeActive || len(store.writes) != 2 || store.writes[1].Revision <= store.writes[0].Revision {
		t.Fatal("next edit did not recover with a fresh revision")
	}
}

// TestPreferencesWriteDuringLoadDoesNotStrandTheRead covers the one ordering the
// other two skip: both let the initial read settle before editing. A write that
// begins while the read is in flight must not consume the read's correlation --
// Game.Update() returns early while preferences.loading is set, so a dropped read
// completion kills every gameplay input for the rest of the session.
func TestPreferencesWriteDuringLoadDoesNotStrandTheRead(t *testing.T) {
	store := &preferenceStoreStub{}
	controller, err := newPreferenceController(store)
	if err != nil {
		t.Fatal(err)
	}
	edit := ui.DefaultUISettings()
	edit.Muted = true
	if err := controller.update(edit); err != nil {
		t.Fatal(err)
	}
	if !controller.loading || len(store.writes) != 1 {
		t.Fatalf("edit during load: loading=%t writes=%d", controller.loading, len(store.writes))
	}
	loaded := ui.DefaultUISettings()
	loaded.MasterVolume = 0.7
	store.completions = []ui.UISettingsCompletion{{Operation: ui.UISettingsRead, Revision: 1, Settings: loaded}}
	results := controller.poll()
	if controller.loading || len(results) != 1 || !results[0].installed || controller.value != loaded {
		t.Fatalf("read stranded by a concurrent write: loading=%t results=%d value=%+v", controller.loading, len(results), controller.value)
	}
}

// TestPreferencesFailedReadKeepsSeededDefaults pins that a store is not required
// to carry usable settings alongside a read error. A zero record would otherwise
// install silent audio with easy mode off while announcing "using defaults".
func TestPreferencesFailedReadKeepsSeededDefaults(t *testing.T) {
	store := &preferenceStoreStub{}
	controller, err := newPreferenceController(store)
	if err != nil {
		t.Fatal(err)
	}
	store.completions = []ui.UISettingsCompletion{{Operation: ui.UISettingsRead, Revision: 1, Err: errors.New("unreadable")}}
	results := controller.poll()
	if len(results) != 1 || !results[0].installed || results[0].notice == "" {
		t.Fatalf("failed read = %+v", results)
	}
	if controller.loading || controller.value != ui.DefaultUISettings() {
		t.Fatalf("failed read installed %+v, want defaults", controller.value)
	}
}
