package audio

import (
	"errors"
	"io"
	"math"
	"sync"
	"testing"
)

type recordedPlayer struct {
	playing      bool
	volume       float64
	plays        int
	pauses       int
	volumeAtPlay float64
	source       io.Reader
}

func (p *recordedPlayer) Play()                   { p.plays++; p.playing = true; p.volumeAtPlay = p.volume }
func (p *recordedPlayer) Pause()                  { p.pauses++; p.playing = false }
func (p *recordedPlayer) IsPlaying() bool         { return p.playing }
func (p *recordedPlayer) SetVolume(value float64) { p.volume = value }

func recordingDevice() (*Manager, *[]*recordedPlayer) {
	var players []*recordedPlayer
	m := &Manager{active: true, newPlayer: func(reader io.Reader) soundPlayer {
		p := &recordedPlayer{source: reader}
		players = append(players, p)
		return p
	}}
	return m, &players
}

func TestMusicSharesMasterAndSurvivesEffectPruning(t *testing.T) {
	m, players := recordingDevice()
	m.SetMaster(0.8, false)
	m.Play(SFXChoiceClick)
	if len(*players) != 2 {
		t.Fatal("expected music and one effect")
	}
	music, effect := (*players)[0], (*players)[1]
	if _, ok := music.source.(*Ambient); !ok {
		t.Fatal("music is not the streaming score")
	}
	if math.Abs(music.volumeAtPlay-musicGain*0.8) > 1e-12 || effect.volumeAtPlay != 0.8 {
		t.Fatal("initial gain was applied after playback")
	}
	m.SetMaster(0.2, true)
	if music.volume != 0 || effect.volume != 0 || !music.playing {
		t.Fatal("mute did not silence both channels while preserving position")
	}
	m.SetMaster(0.2, false)
	if math.Abs(music.volume-musicGain*0.2) > 1e-12 || effect.volume != 0.2 {
		t.Fatal("unmute did not restore the selected gain")
	}
	effect.playing = false
	m.Play(SFXSaveComplete)
	if len(*players) != 3 || len(m.players) != 1 || m.music != music {
		t.Fatal("pruning effects restarted or discarded the music")
	}
	m.SetActive(false)
	m.SetActive(false)
	if music.playing || music.pauses != 1 {
		t.Fatal("background music did not pause once")
	}
	m.SetMaster(0.4, false)
	if music.playing {
		t.Fatal("volume change resumed background music")
	}
	m.SetActive(true)
	if music.plays != 2 || m.music != music {
		t.Fatal("focus reset the music instead of resuming")
	}
	m.Stop()
	m.Stop()
	m.StartMusic()
	m.Play(SFXEventTrigger)
	if music.playing || (*players)[2].playing || m.music != nil || m.players != nil || len(*players) != 3 {
		t.Fatal("stopped players leaked or restarted")
	}
}

func TestLazyMusicRequiresGestureAndSettledUnmutedPreferences(t *testing.T) {
	for _, muted := range []bool{false, true} {
		m, players := recordingDevice()
		creates := 0
		lazy := newLazyManager(func() (SoundManager, error) { creates++; return m, nil }, nil)
		lazy.Poll()
		lazy.SetActive(false)
		if creates != 0 {
			t.Fatal("startup or focus opened the device")
		}
		lazy.Unlock()
		lazy.Unlock()
		if creates != 1 || len(*players) != 0 || m.effectiveVolume() != 0 {
			t.Fatal("gesture bypassed preference settlement")
		}
		lazy.SetMaster(0.6, muted)
		if muted {
			if len(*players) != 0 {
				t.Fatal("persisted mute started music")
			}
			lazy.SetMaster(0.6, false)
		}
		if len(*players) != 1 || (*players)[0].playing || math.Abs((*players)[0].volume-musicGain*0.6) > 1e-12 {
			t.Fatal("music did not honour focus and settled gain")
		}
		lazy.SetActive(true)
		lazy.Unlock()
		if len(*players) != 1 || !(*players)[0].playing {
			t.Fatal("music duplicated or failed to resume")
		}
		lazy.Stop()
		lazy.Unlock()
		lazy.SetMaster(1, false)
		if (*players)[0].playing || creates != 1 {
			t.Fatal("closed music restarted")
		}
	}
	m, _ := recordingDevice()
	creates := 0
	lazy := newLazyManager(func() (SoundManager, error) { creates++; return m, nil }, nil)
	lazy.SetMaster(0.8, false)
	lazy.Poll()
	lazy.SetActive(true)
	if creates != 0 {
		t.Fatal("settled preferences opened the device before a gesture")
	}
}

type failingMusicBackend struct {
	fakeManager
	stops int
}

func (m *failingMusicBackend) StartMusic() {}
func (m *failingMusicBackend) Stop()       { m.stops++ }

func TestMusicDeviceFailureStopsBeforeDiscardingBackend(t *testing.T) {
	backend := &failingMusicBackend{}
	reports := 0
	lazy := newLazyManager(func() (SoundManager, error) { return backend, nil }, func(string, error) { reports++ })
	lazy.SetMaster(0.5, false)
	lazy.Unlock()
	backend.health = errors.New("device lost")
	backend.opened = true
	lazy.Poll()
	lazy.Poll()
	lazy.Stop()
	lazy.Unlock()
	if backend.stops != 1 || reports != 1 || lazy.manager != nil {
		t.Fatal("failed music was not stopped and reported exactly once")
	}
}

func TestLazyConcurrentFocusAndControlsKeepOneMusicPlayer(t *testing.T) {
	m, players := recordingDevice()
	creates := 0
	lazy := newLazyManager(func() (SoundManager, error) { creates++; return m, nil }, nil)
	lazy.SetMaster(0.5, false)
	var done sync.WaitGroup
	for worker := range 4 {
		done.Go(func() {
			for i := range 50 {
				lazy.Unlock()
				lazy.SetActive((worker+i)%2 == 0)
				lazy.SetMaster(0.3, (worker+i)%3 == 0)
				lazy.Poll()
			}
		})
	}
	done.Wait()
	lazy.SetMaster(0.5, false)
	lazy.SetActive(true)
	if creates != 1 || len(*players) != 1 || !(*players)[0].playing {
		t.Fatal("overlapping controls duplicated or stalled music")
	}
	lazy.Stop()
}
