package app

import (
	"bytes"
	"errors"
	"os"
	"strings"
	"testing"

	"github.com/adsouza/africa2ice/internal/adapters/logging"
	gameaudio "github.com/adsouza/africa2ice/pkg/audio"
)

// TestAudioReporterReachesTheLogTheConsoleAndTheHUD covers the AerynOS
// report's blind spot. The ALSA error ended the process through RunGame and
// was printed to stderr by main, so the JSON log a player can actually send
// showed an orderly session.end and no cause at all. Every one of the three
// readers must see the failure, because each is the only one some player has:
// the HUD notice for someone playing, the console for someone in a terminal,
// and the log for someone reporting it afterwards.
func TestAudioReporterReachesTheLogTheConsoleAndTheHUD(t *testing.T) {
	var announcement bytes.Buffer
	session, err := logging.NewDefaultSession(&announcement)
	if err != nil {
		t.Fatal(err)
	}
	path, ok := strings.CutPrefix(strings.TrimSpace(announcement.String()), "Africa 2 Ice session log: ")
	if !ok {
		t.Skipf("session log did not land in a readable file: %q", announcement.String())
	}
	t.Cleanup(func() { _ = os.Remove(path) })

	var console bytes.Buffer
	var notices []string
	report := audioReporter(session, &console, func(notice string) { notices = append(notices, notice) })

	report("init", errors.New("oto: ALSA error at snd_pcm_open"))
	_ = session.Close()

	if !strings.Contains(console.String(), "snd_pcm_open") {
		t.Fatalf("console output = %q, want the device error", console.String())
	}
	if len(notices) != 1 {
		t.Fatalf("HUD notices = %v, want exactly one", notices)
	}
	logged, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	for _, required := range []string{"audio.failure", "init", "snd_pcm_open"} {
		if !strings.Contains(string(logged), required) {
			t.Fatalf("session log missing %q: %s", required, logged)
		}
	}
}

// TestAudioReporterSurvivesAMissingSession covers pkg/app's session-less
// construction paths, which tests and the screenshot runner both use.
func TestAudioReporterSurvivesAMissingSession(t *testing.T) {
	var console bytes.Buffer
	report := audioReporter(nil, &console, nil)
	report("play", errors.New("device disappeared"))
	if !strings.Contains(console.String(), "device disappeared") {
		t.Fatalf("console output = %q, want the device error", console.String())
	}
}

// pollingManager records whether the host ticks the audio port.
type pollingManager struct{ polls int }

func (manager *pollingManager) Play(gameaudio.Sound)    {}
func (manager *pollingManager) SetMaster(float64, bool) {}
func (manager *pollingManager) Poll()                   { manager.polls++ }

// TestUpdatePollsTheSoundPort covers the feedback path for an asynchronous
// device failure: pkg/audio can only report a device that died between clicks
// if the host actually ticks it, so a missing call here would silently undo
// the reporting the tests in pkg/audio prove.
func TestUpdatePollsTheSoundPort(t *testing.T) {
	sound := &pollingManager{}
	game := NewWithSound(&gameStub{frame: migrationPreviewFrame()}, sound)
	if err := game.Update(); err != nil {
		t.Fatal(err)
	}
	if sound.polls == 0 {
		t.Fatal("Update did not poll the sound port")
	}
}
