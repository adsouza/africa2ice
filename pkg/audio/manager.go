package audio

import ebitenaudio "github.com/hajimehoshi/ebiten/v2/audio"

// SoundManager is the presentation port used by the host. Verification modes
// receive NoopManager and therefore never construct a sound-device context.
type SoundManager interface {
	Play(Sound)
	SetMaster(volume float64, muted bool)
}

type NoopManager struct{}

func (NoopManager) Play(Sound)              {}
func (NoopManager) SetMaster(float64, bool) {}

// Manager owns synthesized players and applies one master gain to both
// current and future sounds.
type Manager struct {
	context *ebitenaudio.Context
	players []*ebitenaudio.Player
	volume  float64
	muted   bool
}

func NewManager() *Manager {
	return &Manager{context: ebitenaudio.NewContext(sampleRate), volume: 0.5}
}

func (manager *Manager) Play(sound Sound) {
	if manager == nil || sound >= soundCount {
		return
	}
	manager.prunePlayers()
	player := manager.context.NewPlayerF32FromBytes(synthPCM(sound))
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
			continue
		}
		_ = player.Close()
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
	create  func() SoundManager
	volume  float64
	muted   bool
	settled bool
	pending *Sound
}

func NewLazyManager() *LazyManager {
	return newLazyManager(func() SoundManager { return NewManager() })
}

func newLazyManager(create func() SoundManager) *LazyManager {
	return &LazyManager{create: create, volume: 0.5}
}

func (manager *LazyManager) Play(sound Sound) {
	if manager == nil {
		return
	}
	if manager.manager == nil {
		manager.manager = manager.create()
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
}

func (manager *LazyManager) SetMaster(volume float64, muted bool) {
	if manager == nil {
		return
	}
	manager.volume = clampVolume(volume)
	manager.muted = muted
	manager.settled = true
	if manager.manager != nil {
		manager.manager.SetMaster(manager.volume, muted)
		if manager.pending != nil && !muted {
			manager.manager.Play(*manager.pending)
		}
	}
	manager.pending = nil
}
