package app

import (
	"fmt"
	"io"

	"github.com/adsouza/africa2ice/internal/adapters/logging"
	gameaudio "github.com/adsouza/africa2ice/pkg/audio"
	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
)

// audioUnavailableNotice matches the wording of the preferences fallback: a
// presentation-only subsystem degraded, and the campaign continues.
const audioUnavailableNotice = "Sound is unavailable; continuing without audio"

// audioReporter fans one audio-device failure out to the three readers that
// can each be the only one a given player has: the HUD notice for someone
// playing, the console for someone running from a terminal, and the session
// log for someone reporting the problem afterwards. Before this existed the
// device error reached stderr only, and then only because it had already
// ended the Ebitengine loop.
func audioReporter(session *logging.Session, console io.Writer, notify func(string)) gameaudio.Reporter {
	return func(stage string, err error) {
		session.LogAudioFailure(stage, err)
		if console != nil {
			_, _ = fmt.Fprintf(console, "Africa 2 Ice: sound is unavailable (%s): %v\n", stage, err)
		}
		if notify != nil {
			notify(audioUnavailableNotice)
		}
	}
}

// pollAudio ticks a sound port that reports device failures asynchronously.
// SoundManager stays a two-method port so real .wav assets can still replace
// the synth without an API change, so this is an optional interface rather
// than a third method every implementation would have to carry.
func (g *Game) pollAudio() {
	g.setAudioActive(audioFocused())
	if poller, ok := g.sound.(interface{ Poll() }); ok {
		poller.Poll()
	}
	if inpututil.IsMouseButtonJustPressed(ebiten.MouseButtonLeft) ||
		len(inpututil.AppendJustPressedKeys(nil)) > 0 || len(inpututil.AppendJustPressedTouchIDs(nil)) > 0 {
		if unlocker, ok := g.sound.(interface{ Unlock() }); ok {
			unlocker.Unlock()
		}
	}
}

func (g *Game) setAudioActive(active bool) {
	if focus, ok := g.sound.(interface{ SetActive(bool) }); ok {
		focus.SetActive(active)
	}
}
