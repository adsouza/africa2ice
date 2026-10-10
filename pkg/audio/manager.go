package audio

import (
	"bytes"
	"io"
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
	context   *oto.Context
	players   []soundPlayer
	newPlayer func(io.Reader) soundPlayer
	music     soundPlayer
	active    bool
	stopped   bool
	volume    float64
	muted     bool

	// opened is written once by the goroutine watching oto's ready channel
	// and read from the game loop, so it needs the mutex.
	mutex  sync.Mutex
	opened bool

	// panicGuard is deferred by the device-watching goroutine; never nil.
	panicGuard func()
}

// A narrow seam keeps gain, focus, and lifetime checks independent of hardware.
type soundPlayer interface {
	Play()
	Pause()
	IsPlaying() bool
	SetVolume(float64)
}

const musicGain = 0.7
const musicSeed = 20261010

// NewManager opens the sound device. The open runs on oto's own goroutine, so
// a device that cannot be opened surfaces through Err() a few frames later
// rather than from this call: waiting on oto's ready channel here would stall
// the game loop, and on wasm would deadlock the event loop the browser needs
// in order to resume audio in the first place.
//
// panicGuard is the session's panic hook for that watching goroutine; nil
// means none.
func NewManager(panicGuard func()) (*Manager, error) {
	context, ready, err := oto.NewContext(&oto.NewContextOptions{
		SampleRate:   sampleRate,
		ChannelCount: channelCount,
		Format:       oto.FormatFloat32LE,
	})
	if err != nil {
		return nil, err
	}
	manager := &Manager{context: context, volume: 0.5, active: true, panicGuard: orNoGuard(panicGuard),
		newPlayer: func(reader io.Reader) soundPlayer { return context.NewPlayer(reader) }}
	// oto stores a setup error before it closes ready, so the error is already
	// final here: a clean ready means the device genuinely came up, and any
	// error after this point is a working device that later stopped.
	go manager.awaitDevice(ready, context.Err)
	return manager, nil
}

// awaitDevice records that the device opened once oto reports it ready
// without an error. It runs on its own goroutine, so it defers the session's
// panic hook.
func (manager *Manager) awaitDevice(ready <-chan struct{}, deviceErr func() error) {
	defer manager.panicGuard()
	<-ready
	if deviceErr() != nil {
		return
	}
	manager.mutex.Lock()
	manager.opened = true
	manager.mutex.Unlock()
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
	if manager == nil || manager.stopped || sound >= soundCount {
		return
	}
	manager.prunePlayers()
	manager.StartMusic()
	player := manager.newPlayer(bytes.NewReader(synthPCM(sound)))
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
	if manager.music != nil {
		manager.music.SetVolume(musicGain * manager.effectiveVolume())
	}
}

// StartMusic is idempotent. LazyManager calls it only after a user gesture and
// preference settlement; this long-lived player is separate from effects.
func (manager *Manager) StartMusic() {
	if manager == nil || manager.stopped || manager.music != nil || manager.Err() != nil {
		return
	}
	manager.music = manager.newPlayer(NewAmbient(musicSeed))
	manager.music.SetVolume(musicGain * manager.effectiveVolume())
	if manager.active {
		manager.music.Play()
	}
}

func (manager *Manager) SetActive(active bool) {
	if manager == nil || manager.stopped || manager.active == active {
		return
	}
	manager.active = active
	if manager.music != nil {
		if active {
			manager.music.Play()
		} else {
			manager.music.Pause()
		}
	}
}

// Stop pauses players before releasing them: oto.Player.Close is a no-op.
func (manager *Manager) Stop() {
	if manager == nil || manager.stopped {
		return
	}
	manager.stopped = true
	if manager.music != nil {
		manager.music.Pause()
		manager.music = nil
	}
	for _, player := range manager.players {
		player.Pause()
	}
	manager.players = nil
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

// LazyManager delays audio-context construction until the first user gesture
// or accepted sound request. This keeps browser autoplay tied to that gesture and
// leaves startup, headless, map-dump, and screenshot paths device-independent.
type LazyManager struct {
	// Browser focus callbacks can overlap game-loop controls when a player
	// operation yields. Serialize the complete lifecycle, including construction.
	controlMutex sync.Mutex
	manager      SoundManager
	create       func() (SoundManager, error)
	report       Reporter
	volume       float64
	muted        bool
	settled      bool
	failed       bool
	pending      *Sound
	active       bool
	stopped      bool
}

// NewLazyManager defers opening the device to the first gesture. panicGuard is
// the session's panic hook, passed on to the device's goroutine; nil means none.
func NewLazyManager(report Reporter, panicGuard func()) *LazyManager {
	return newLazyManager(func() (SoundManager, error) { return NewManager(panicGuard) }, report)
}

func newLazyManager(create func() (SoundManager, error), report Reporter) *LazyManager {
	return &LazyManager{create: create, report: report, volume: 0.5, active: true}
}

// Unlock is called only for a user gesture. Opening silently before settings
// settle permits browser autoplay without letting music bypass persisted mute.
func (manager *LazyManager) Unlock() {
	if manager == nil {
		return
	}
	manager.controlMutex.Lock()
	defer manager.controlMutex.Unlock()
	manager.unlock()
}

func (manager *LazyManager) unlock() {
	if manager.failed || manager.stopped {
		return
	}
	if manager.manager == nil {
		created, err := manager.create()
		if err != nil {
			manager.fail("init", err)
			return
		}
		manager.manager = created
		if focus, ok := created.(interface{ SetActive(bool) }); ok {
			focus.SetActive(manager.active)
		}
		if manager.settled {
			manager.manager.SetMaster(manager.volume, manager.muted)
		} else {
			manager.manager.SetMaster(0, true)
		}
	}
	manager.startMusic()
}

func (manager *LazyManager) startMusic() {
	if !manager.settled || manager.muted {
		return
	}
	if music, ok := manager.manager.(interface{ StartMusic() }); ok {
		music.StartMusic()
	}
}

func (manager *LazyManager) Play(sound Sound) {
	if manager == nil {
		return
	}
	manager.controlMutex.Lock()
	defer manager.controlMutex.Unlock()
	if manager.failed || manager.stopped || sound >= soundCount {
		return
	}
	manager.unlock()
	if manager.failed {
		return
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
	manager.stopBackend()
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
	if manager == nil {
		return
	}
	manager.controlMutex.Lock()
	defer manager.controlMutex.Unlock()
	if manager.failed || manager.stopped || manager.manager == nil {
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
	manager.controlMutex.Lock()
	defer manager.controlMutex.Unlock()
	manager.volume = clampVolume(volume)
	manager.muted = muted
	manager.settled = true
	if manager.failed || manager.stopped {
		return
	}
	if manager.manager != nil {
		manager.manager.SetMaster(manager.volume, muted)
		manager.startMusic()
		if manager.pending != nil && !muted {
			manager.manager.Play(*manager.pending)
			manager.pending = nil
			manager.checkHealth()
			return
		}
	}
	manager.pending = nil
}

func (manager *LazyManager) SetActive(active bool) {
	if manager == nil {
		return
	}
	manager.controlMutex.Lock()
	defer manager.controlMutex.Unlock()
	if manager.failed || manager.stopped {
		return
	}
	manager.active = active
	if focus, ok := manager.manager.(interface{ SetActive(bool) }); ok {
		focus.SetActive(active)
	}
}

func (manager *LazyManager) stopBackend() {
	if backend, ok := manager.manager.(interface{ Stop() }); ok {
		backend.Stop()
	}
}

func (manager *LazyManager) Stop() {
	if manager == nil {
		return
	}
	manager.controlMutex.Lock()
	defer manager.controlMutex.Unlock()
	if manager.stopped {
		return
	}
	manager.stopBackend()
	manager.stopped = true
	manager.manager = nil
	manager.pending = nil
}
