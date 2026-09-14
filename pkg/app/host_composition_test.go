//go:build !js

package app

import (
	"errors"
	"testing"
	"time"

	"github.com/adsouza/africa2ice/internal/adapters/storage"
	"github.com/adsouza/africa2ice/internal/application"
	gameaudio "github.com/adsouza/africa2ice/pkg/audio"
	"github.com/adsouza/africa2ice/pkg/ui"
)

func TestHostCompositionPropagatesRepositoryFailure(t *testing.T) {
	want := errors.New("cannot open campaign storage")
	calls := 0
	game, err := composeHostedGame(1, nil, false, func() (application.CampaignRepository, error) { calls++; return nil, want }, func() (ui.UISettingsStore, error) {
		t.Fatal("preferences opened after repository failure")
		return nil, nil
	}, true)
	if game != nil || err != want || calls != 1 {
		t.Fatalf("host=%v error=%v calls=%d", game, err, calls)
	}
}

func TestHostCompositionResumesThroughIsolatedRepository(t *testing.T) {
	for _, resume := range []bool{false, true} {
		t.Run(map[bool]string{false: "fresh", true: "resume"}[resume], func(t *testing.T) {
			repository, err := storage.NewFileRepository(t.TempDir())
			if err != nil {
				t.Fatal(err)
			}
			defer repository.Close()
			settings := &settingsStoreStub{}
			repositoryCalls, settingsCalls := 0, 0
			game, err := composeHostedGame(17, nil, false,
				func() (application.CampaignRepository, error) { repositoryCalls++; return repository, nil },
				func() (ui.UISettingsStore, error) { settingsCalls++; return settings, nil }, resume)
			if err != nil {
				t.Fatal(err)
			}
			if repositoryCalls != 1 || settingsCalls != 1 || len(settings.reads) != 1 || !game.settingsLoading || game.frame == nil || len(game.frame.Bands) == 0 || game.scenes.Current() != ui.SceneTitle {
				t.Fatalf("incomplete startup: repository=%d settings=%d loading=%t scene=%v", repositoryCalls, settingsCalls, game.settingsLoading, game.scenes.Current())
			}
			if _, ok := game.sound.(gameaudio.NoopManager); !ok {
				t.Fatalf("silent startup constructed %T", game.sound)
			}
			if game.startupRestorePending != resume {
				t.Fatalf("resume pending=%t want %t", game.startupRestorePending, resume)
			}
			deadline := time.Now().Add(3 * time.Second)
			for game.startupRestorePending && time.Now().Before(deadline) {
				game.pollStorage()
				time.Sleep(time.Millisecond)
			}
			if game.startupRestorePending || game.scenes.Current() != ui.SceneTitle {
				t.Fatal("empty repository did not settle on title")
			}
			settings.completions = []ui.UISettingsCompletion{{Operation: ui.UISettingsRead, Revision: 1, Settings: ui.DefaultUISettings()}}
			game.pollUISettings()
			if game.settingsLoading {
				t.Fatal("host did not consume preference completion")
			}
			if _, err := game.port.StateHash(); err != nil {
				t.Fatalf("composed application is unusable: %v", err)
			}
		})
	}
}

type failedSettingsRead struct{ settingsStoreStub }

func (*failedSettingsRead) BeginRead(uint64) error { return errors.New("preferences unreadable") }

func TestHostPreferenceFailuresUseDefaults(t *testing.T) {
	for _, openFailure := range []bool{false, true} {
		t.Run(map[bool]string{false: "read", true: "open"}[openFailure], func(t *testing.T) {
			repository, err := storage.NewFileRepository(t.TempDir())
			if err != nil {
				t.Fatal(err)
			}
			defer repository.Close()
			game, err := composeHostedGame(17, nil, true,
				func() (application.CampaignRepository, error) { return repository, nil },
				func() (ui.UISettingsStore, error) {
					if openFailure {
						return nil, errors.New("preferences unavailable")
					}
					return &failedSettingsRead{}, nil
				}, false)
			if err != nil {
				t.Fatal(err)
			}
			if game.settingsLoading || game.settings != ui.DefaultUISettings() || game.frame.EasyMode != game.settings.EasyMode {
				t.Fatal("preference failure blocked startup or skipped defaults")
			}
			if _, ok := game.sound.(*gameaudio.LazyManager); !ok {
				t.Fatalf("audio startup should remain lazy, got %T", game.sound)
			}
			want := "Preferences could not be loaded; using defaults"
			if openFailure {
				want = "Preferences are unavailable; using defaults"
			}
			if game.notice != want {
				t.Fatalf("notice=%q", game.notice)
			}
		})
	}
}
