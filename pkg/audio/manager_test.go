package audio

import (
	"errors"
	"testing"
)

// fakeManager stands in for a device-backed Manager. health is what Err()
// reports after each Play, which is how a real oto context surfaces a device
// that opened and then failed.
type fakeManager struct {
	played []Sound
	health error
	opened bool
}

func (manager *fakeManager) Play(sound Sound)        { manager.played = append(manager.played, sound) }
func (manager *fakeManager) SetMaster(float64, bool) {}
func (manager *fakeManager) Err() error              { return manager.health }
func (manager *fakeManager) Opened() bool            { return manager.opened }

// TestLazyManagerDegradesWhenTheDeviceCannotBeCreated covers the AerynOS
// crash: oto could not open any ALSA PCM device, ebiten surfaced that through
// its before-update hook, and RunGame returned it, ending the session on the
// first click. A sound device is presentation-only, so failing to get one must
// silence audio and report once, never end the campaign.
func TestLazyManagerDegradesWhenTheDeviceCannotBeCreated(t *testing.T) {
	deviceErr := errors.New("oto: ALSA error at snd_pcm_open")
	creates := 0
	var stages []string
	var reported error
	manager := newLazyManager(
		func() (SoundManager, error) {
			creates++
			return nil, deviceErr
		},
		func(stage string, err error) {
			stages = append(stages, stage)
			reported = err
		},
	)

	manager.SetMaster(0.8, false)
	manager.Play(SFXChoiceClick)
	manager.Play(SFXSaveComplete)
	manager.SetMaster(0.4, false)

	if creates != 1 {
		t.Fatalf("create attempts = %d, want 1: a dead device must not be retried on every click", creates)
	}
	if len(stages) != 1 || stages[0] != "init" {
		t.Fatalf("reported stages = %v, want exactly one \"init\"", stages)
	}
	if !errors.Is(reported, deviceErr) {
		t.Fatalf("reported error = %v, want %v", reported, deviceErr)
	}
}

// TestLazyManagerReportsADeviceThatFailsAfterOpening covers a device that
// opens and later stops working. The stage must distinguish this from a device
// that was never available, so a session log says which one happened.
func TestLazyManagerReportsADeviceThatFailsAfterOpening(t *testing.T) {
	playErr := errors.New("oto: device disappeared")
	backing := &fakeManager{health: playErr, opened: true}
	var stages []string
	var reported error
	manager := newLazyManager(
		func() (SoundManager, error) { return backing, nil },
		func(stage string, err error) {
			stages = append(stages, stage)
			reported = err
		},
	)

	manager.SetMaster(0.8, false)
	manager.Play(SFXChoiceClick)
	manager.Play(SFXSaveComplete)

	if len(stages) != 1 || stages[0] != "play" {
		t.Fatalf("reported stages = %v, want exactly one \"play\"", stages)
	}
	if !errors.Is(reported, playErr) {
		t.Fatalf("reported error = %v, want %v", reported, playErr)
	}
	if len(backing.played) != 1 {
		t.Fatalf("played %d sounds, want 1: a failed device must not be driven again", len(backing.played))
	}
}

// TestLazyManagerStaysSilentWithoutAReporter covers pkg/app's session-less
// paths, which construct a manager without a log to report into.
func TestLazyManagerStaysSilentWithoutAReporter(t *testing.T) {
	manager := newLazyManager(
		func() (SoundManager, error) { return nil, errors.New("no device") },
		nil,
	)
	manager.SetMaster(0.8, false)
	manager.Play(SFXChoiceClick)
}

// TestLazyManagerPlaysThroughAHealthyDevice pins the success path, so the
// degradation tests above cannot pass by silencing audio unconditionally.
func TestLazyManagerPlaysThroughAHealthyDevice(t *testing.T) {
	backing := &fakeManager{}
	var stages []string
	manager := newLazyManager(
		func() (SoundManager, error) { return backing, nil },
		func(stage string, err error) { stages = append(stages, stage) },
	)

	manager.SetMaster(0.8, false)
	manager.Play(SFXChoiceClick)
	manager.Play(SFXSaveComplete)

	if len(stages) != 0 {
		t.Fatalf("reported stages = %v, want none for a healthy device", stages)
	}
	if len(backing.played) != 2 {
		t.Fatalf("played = %v, want both sounds", backing.played)
	}
}

// TestLazyManagerPollReportsADeviceThatFailedAfterTheFirstSound covers the
// asynchronous shape of a real ALSA failure. oto opens the device on its own
// goroutine, so the error appears a few frames after the click that triggered
// it. Waiting for the *next* sound to notice would leave a player who stops
// clicking with silence and no diagnostic at all.
func TestLazyManagerPollReportsADeviceThatFailedAfterTheFirstSound(t *testing.T) {
	backing := &fakeManager{}
	var stages []string
	manager := newLazyManager(
		func() (SoundManager, error) { return backing, nil },
		func(stage string, err error) { stages = append(stages, stage) },
	)
	manager.SetMaster(0.8, false)
	manager.Play(SFXChoiceClick)
	if len(stages) != 0 {
		t.Fatalf("reported %v before the device settled", stages)
	}

	backing.opened = true
	backing.health = errors.New("oto: ALSA error at snd_pcm_writei")
	manager.Poll()
	manager.Poll()

	if len(stages) != 1 || stages[0] != "play" {
		t.Fatalf("reported stages = %v, want exactly one \"play\"", stages)
	}
}

// TestLazyManagerPollIsSafeBeforeAnySound covers the ticks between launch and
// the first click, when no device has been constructed.
func TestLazyManagerPollIsSafeBeforeAnySound(t *testing.T) {
	var stages []string
	manager := newLazyManager(
		func() (SoundManager, error) { return &fakeManager{}, nil },
		func(stage string, err error) { stages = append(stages, stage) },
	)
	manager.Poll()
	if len(stages) != 0 {
		t.Fatalf("reported %v with no device constructed", stages)
	}
}

// TestLazyManagerCallsAnUnopenedDeviceAnInitFailure covers the AerynOS
// follow-up. oto opens the device on its own goroutine, so a setup failure
// (there: snd_pcm_hw_params_set_format rejecting FLOAT_LE) surfaces only after
// the first sound was requested. Reporting that as "play" claims a device
// played and then broke, when it never opened at all, and the two call for
// different fixes.
func TestLazyManagerCallsAnUnopenedDeviceAnInitFailure(t *testing.T) {
	setupErr := errors.New("oto: ALSA error at snd_pcm_hw_params_set_format: Invalid argument")
	backing := &fakeManager{}
	var stages []string
	manager := newLazyManager(
		func() (SoundManager, error) { return backing, nil },
		func(stage string, err error) { stages = append(stages, stage) },
	)
	manager.SetMaster(0.8, false)
	manager.Play(SFXChoiceClick)

	// The device setup fails asynchronously, after the sound was requested.
	backing.health = setupErr

	manager.Poll()

	if len(stages) != 1 || stages[0] != "init" {
		t.Fatalf("reported stages = %v, want exactly one \"init\": the device never opened", stages)
	}
}
