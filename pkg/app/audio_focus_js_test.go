//go:build js && wasm

package app

import (
	"syscall/js"
	"testing"

	gameaudio "github.com/adsouza/africa2ice/pkg/audio"
)

type focusSound struct{ active []bool }

func (*focusSound) Play(gameaudio.Sound)    {}
func (*focusSound) SetMaster(float64, bool) {}
func (s *focusSound) SetActive(active bool) { s.active = append(s.active, active) }

func TestVisibilityEventPausesAudioWithoutAnUpdateAndDisposesListener(t *testing.T) {
	document := js.Global().Get("document")
	// Override only this document instance, restoring its prototype getter.
	js.Global().Get("Object").Call("defineProperty", document, "hidden", map[string]any{"value": true, "configurable": true})
	defer js.Global().Get("Reflect").Call("deleteProperty", document, "hidden")
	sound := &focusSound{}
	game := &Game{sound: sound}
	dispose := game.observeAudioFocus()
	event := js.Global().Get("Event").New("visibilitychange")
	document.Call("dispatchEvent", event)
	if len(sound.active) != 1 || sound.active[0] {
		t.Fatal("hidden tab did not pause without Game.Update")
	}
	dispose()
	document.Call("dispatchEvent", event)
	if len(sound.active) != 1 {
		t.Fatal("disposed listener still called the audio port")
	}
}
