package audio

import (
	"bytes"
	"sync"

	"github.com/ebitengine/oto/v3"
)

// SoundManager is the presentation port used by the host. Verification modes
// receive NoopManager and therefore never construct a sound-device context.
type SoundManager interface {
	Play(Sound)
	SetMaster(volume float64, muted bool)
}

// Reporter receives audio-subsystem failures. pkg/audio is a leaf and cannot
// import the observability adapter, so the host injects one. stage is "init"
// for a device that never opened and "play" for one that stopped working, a
// distinction a session log needs to tell a missing device from a lost one.
type Reporter func(stage string, err error)

// deviceHealth is implemented by managers backed by a real device, whose
// errors surface asynchronously rather than from the Play call itself.
// Opened separates a device that never came up from one that worked and then
// stopped, because the two ask a player for different remedies and both arrive
// through the same Err() after the same first sound request.
type deviceHealth interface {
	Err() error
	Opened() bool
}

type NoopManager struct{}

func (NoopManager) Play(Sound)              {}
func (NoopManager) SetMaster(float64, bool) {}

// Manager owns synthesized players and applies one master gain to both
// current and future sounds.
//
// It drives oto directly rather than through Ebitengine's audio package, and
// that is the whole point: Ebitengine reports a device error from a per-tick
// hook, where a non-nil result ends RunGame and takes the campaign with it.
// A sound device is presentation-only, so its failure is reported here and
// degrades to silence instead.
type Manager struct {
	context *oto.Context
	players []*oto.Player
	volume  float64
	muted   bool

	// opened is written once by the goroutine watching oto's ready channel
	// and read from the game loop, so it needs the mutex.
	mutex  sync.Mutex
	opened bool
}

// NewManager opens the sound device. The open runs on oto's own goroutine, so
// a device that cannot be opened surfaces through Err() a few frames later
// rather than from this call: waiting on oto's ready channel here would stall
// the game loop, and on wasm would deadlock the event loop the browser needs
// in order to resume audio in the first place.
func NewManager() (*Manager, error) {
	context, ready, err := oto.NewContext(&oto.NewContextOptions{
		SampleRate:   sampleRate,
		ChannelCount: channelCount,
		Format:       oto.FormatFloat32LE,
	})
	if err != nil {
		return nil, err
	}
	manager := &Manager{context: context, volume: 0.5}
	// oto stores a setup error before it closes ready, so the error is already
	// final here: a clean ready means the device genuinely came up, and any
	// error after this point is a working device that later stopped.
	go func() {
		<-ready
		if context.Err() != nil {
			return
		}
		manager.mutex.Lock()
		manager.opened = true
		manager.mutex.Unlock()
	}()
	return manager, nil
}

// Err reports a device that never opened or has stopped working.
func (manager *Manager) Err() error {
	if manager == nil || manager.context == nil {
		return nil
	}
	return manager.context.Err()
}

// Opened reports whether the device ever reached a usable state. oto's ALSA
// driver asks for SND_PCM_FORMAT_FLOAT_LE unconditionally, so a device can open
// and still fail setup; that is a device which never played, not one that broke.
func (manager *Manager) Opened() bool {
	if manager == nil {
		return false
	}
	manager.mutex.Lock()
	defer manager.mutex.Unlock()
	return manager.opened
}

func (manager *Manager) Play(sound Sound) {
	if manager == nil || sound >= soundCount {
		return
	}
	manager.prunePlayers()
	player := manager.context.NewPlayer(bytes.NewReader(synthPCM(sound)))
	player.SetVolume(manager.effectiveVolume())
	player.Play()
	manager.players = append(manager.players, player)
}

func (manager *Manager) SetMaster(volume float64, muted bool) {
	if manager == nil {
		return
	}
	manager.volume = clampVolume(volume)
	manager.muted = muted
	for _, player := range manager.players {
		player.SetVolume(manager.effectiveVolume())
	}
}

func (manager *Manager) effectiveVolume() float64 {
	if manager.muted {
		return 0
	}
	return manager.volume
}

func (manager *Manager) prunePlayers() {
	live := manager.players[:0]
	for _, player := range manager.players {
		if player.IsPlaying() {
			live = append(live, player)
		}
	}
	// oto releases a finished player through a runtime cleanup rather than
	// Close, which is a deprecated no-op as of v3.4. A player therefore has to
	// become genuinely unreachable: leaving stale pointers in the backing
	// array past len would pin every player the session ever created.
	for index := len(live); index < len(manager.players); index++ {
		manager.players[index] = nil
	}
	manager.players = live
}

func clampVolume(value float64) float64 {
	if value < 0 {
		return 0
	}
	if value > 1 {
		return 1
	}
	return value
}

// LazyManager delays audio-context construction until the first accepted
// sound request. This keeps browser autoplay policy tied to a user gesture and
// leaves startup, headless, map-dump, and screenshot paths device-independent.
type LazyManager struct {
	manager SoundManager
	create  func() (SoundManager, error)
	report  Reporter
	volume  float64
	muted   bool
	settled bool
	failed  bool
	pending *Sound
}

func NewLazyManager(report Reporter) *LazyManager {
	return newLazyManager(func() (SoundManager, error) { return NewManager() }, report)
}

func newLazyManager(create func() (SoundManager, error), report Reporter) *LazyManager {
	return &LazyManager{create: create, report: report, volume: 0.5}
}

func (manager *LazyManager) Play(sound Sound) {
	if manager == nil || manager.failed {
		return
	}
	if manager.manager == nil {
		created, err := manager.create()
		if err != nil {
			manager.fail("init", err)
			return
		}
		manager.manager = created
		if manager.settled {
			manager.manager.SetMaster(manager.volume, manager.muted)
		} else {
			manager.manager.SetMaster(0, true)
		}
	}
	if !manager.settled {
		if manager.pending == nil {
			pending := sound
			manager.pending = &pending
		}
		return
	}
	if manager.muted {
		return
	}
	manager.manager.Play(sound)
	manager.checkHealth()
}

// fail silences audio permanently for this session and reports once. A device
// that could not be opened will not open on the next click, so retrying per
// click would only repeat the log record.
func (manager *LazyManager) fail(stage string, err error) {
	manager.failed = true
	manager.manager = nil
	manager.pending = nil
	if manager.report != nil {
		manager.report(stage, err)
	}
}

// Poll surfaces a device that failed after it was opened. oto reports an ALSA
// failure asynchronously, several frames after the sound that triggered it, so
// the host ticks this rather than waiting for the next sound request.
func (manager *LazyManager) Poll() {
	if manager == nil || manager.failed || manager.manager == nil {
		return
	}
	manager.checkHealth()
}

func (manager *LazyManager) checkHealth() {
	health, ok := manager.manager.(deviceHealth)
	if !ok {
		return
	}
	if err := health.Err(); err != nil {
		stage := "init"
		if health.Opened() {
			stage = "play"
		}
		manager.fail(stage, err)
	}
}

func (manager *LazyManager) SetMaster(volume float64, muted bool) {
	if manager == nil {
		return
	}
	manager.volume = clampVolume(volume)
	manager.muted = muted
	manager.settled = true
	if manager.failed {
		return
	}
	if manager.manager != nil {
		manager.manager.SetMaster(manager.volume, muted)
		if manager.pending != nil && !muted {
			manager.manager.Play(*manager.pending)
			manager.pending = nil
			manager.checkHealth()
			return
		}
	}
	manager.pending = nil
}
