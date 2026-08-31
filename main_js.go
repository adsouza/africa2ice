//go:build js

package main

import (
	"os"
	"syscall/js"

	"github.com/adsouza/africa2ice/internal/adapters/logging"
	"github.com/adsouza/africa2ice/internal/application"
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
	defer logging.GuardPanic(session)
	game, err := app.NewGame(0x9e3779b97f4a7c15, session)
	if err == nil {
		installPresentationOptions(game)
		game.SetFirstDrawCallback(func() {
			document := js.Global().Get("document")
			document.Get("documentElement").Get("dataset").Set("africa2iceReady", "true")
			if loader := document.Call("getElementById", "loader"); !loader.IsNull() {
				loader.Get("style").Set("display", "none")
			}
		})
		ebiten.SetScreenClearedEveryFrame(false)
		err = ebiten.RunGameWithOptions(game, &ebiten.RunGameOptions{DisableHiDPI: false})
	}
	if err != nil {
		_, _ = os.Stderr.WriteString(err.Error() + "\n")
	}
}

func installPresentationOptions(game *app.Game) {
	location := js.Global().Get("location")
	parameters := js.Global().Get("URLSearchParams").New(location.Get("search"))
	if parameters.Call("get", "profile").String() == "1" {
		payload := js.Global().Get("africa2iceRenderProfileSave")
		if payload.Type() == js.TypeString {
			state, err := application.DecodeSaveState([]byte(payload.String()))
			if err != nil {
				panic(err)
			}
			frame, err := application.ProjectSaveState(state)
			if err != nil {
				panic(err)
			}
			game.SetRenderProfileFrame(frame)
		}
	}
	if parameters.Call("get", "e2e").String() != "1" {
		return
	}
	observer := js.Global().Get("africa2iceE2EObserver")
	if observer.Type() != js.TypeFunction {
		return
	}
	game.SetFramePublishedCallback(func(summary string) {
		document := js.Global().Get("document")
		document.Get("documentElement").Get("dataset").Set("africa2iceSummary", summary)
		observer.Invoke(summary)
	})
}
