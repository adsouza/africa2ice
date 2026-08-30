//go:build js

package main

import (
	"os"
	"syscall/js"

	"github.com/adsouza/africa2ice/internal/adapters/logging"
	"github.com/adsouza/africa2ice/pkg/app"
	"github.com/hajimehoshi/ebiten/v2"
)

func main() {
	session, err := logging.NewDefaultSession(os.Stdout)
	if err != nil {
		_, _ = os.Stderr.WriteString(err.Error() + "\n")
		return
	}
	defer func() { _ = session.Close() }()
	game, err := app.NewGame(0x9e3779b97f4a7c15, session)
	if err == nil {
		game.SetFirstDrawCallback(func() {
			document := js.Global().Get("document")
			document.Get("documentElement").Get("dataset").Set("africa2iceReady", "true")
			if loader := document.Call("getElementById", "loader"); !loader.IsNull() {
				loader.Get("style").Set("display", "none")
			}
		})
		err = ebiten.RunGameWithOptions(game, &ebiten.RunGameOptions{DisableHiDPI: false})
	}
	if err != nil {
		_, _ = os.Stderr.WriteString(err.Error() + "\n")
	}
}
