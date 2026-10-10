//go:build js && wasm

package app

import "syscall/js"

func audioFocused() bool {
	document := js.Global().Get("document")
	return !document.Get("hidden").Bool() && document.Call("hasFocus").Bool()
}

// Hidden tabs can stop requestAnimationFrame entirely. Pause on the DOM event
// rather than waiting for another Game.Update that may never arrive.
func (g *Game) observeAudioFocus() func() {
	window := js.Global()
	document := window.Get("document")
	panicGuard := g.logSession.PanicGuard
	callback := js.FuncOf(func(js.Value, []js.Value) any {
		defer panicGuard()
		g.setAudioActive(audioFocused())
		return nil
	})
	document.Call("addEventListener", "visibilitychange", callback)
	window.Call("addEventListener", "blur", callback)
	window.Call("addEventListener", "focus", callback)
	return func() {
		document.Call("removeEventListener", "visibilitychange", callback)
		window.Call("removeEventListener", "blur", callback)
		window.Call("removeEventListener", "focus", callback)
		callback.Release()
	}
}
