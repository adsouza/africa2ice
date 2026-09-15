//go:build js

package main

import (
	"fmt"
	"os"
	"syscall/js"

	"github.com/adsouza/africa2ice/internal/adapters/logging"
	"github.com/adsouza/africa2ice/internal/application"
	"github.com/adsouza/africa2ice/internal/verification"
	"github.com/adsouza/africa2ice/pkg/app"
	"github.com/hajimehoshi/ebiten/v2"
)

func main() {
	if runCheckpointMode() {
		return
	}
	session, err := logging.NewDefaultSession(os.Stdout)
	if err != nil {
		reportBootError(err)
		_, _ = os.Stderr.WriteString(err.Error() + "\n")
		return
	}
	defer func() { _ = session.Close() }()
	defer func() {
		if value := recover(); value != nil {
			reportBootError(fmt.Errorf("panic: %v", value))
			panic(value)
		}
	}()
	defer logging.GuardPanic(session)
	game, err := app.NewGame(0x9e3779b97f4a7c15, session)
	if err == nil {
		defer func() { _ = game.Close() }()
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
		reportBootError(err)
		_, _ = os.Stderr.WriteString(err.Error() + "\n")
	}
}

// runCheckpointMode lets the browser determinism gate drive the exact
// optimized module that is deployed. It deliberately exits before composing
// Ebitengine or storage, and is reachable only through the gate's query flag.
func runCheckpointMode() bool {
	location := js.Global().Get("location")
	parameters := js.Global().Get("URLSearchParams").New(location.Get("search"))
	if parameters.Call("get", "checkpoint").String() != "1" {
		return false
	}
	records, err := verification.ReferenceRun(verification.ReferenceSeed, verification.MaxTurns, "reference")
	payload := ""
	if err == nil {
		var encoded []byte
		encoded, err = verification.CanonicalJSON(records)
		payload = string(encoded)
	}
	errorText := ""
	if err != nil {
		errorText = err.Error()
	}
	callback := js.Global().Get("africa2iceCheckpointReady")
	if callback.Type() == js.TypeFunction {
		callback.Invoke(payload, errorText)
	} else if err != nil {
		js.Global().Get("console").Call("error", errorText)
	} else {
		js.Global().Get("console").Call("log", payload)
	}
	return true
}

func reportBootError(err error) {
	reporter := js.Global().Get("africa2iceShowBootError")
	if reporter.Type() != js.TypeFunction {
		return
	}
	reporter.Invoke(
		"Africa 2 Ice could not start",
		"Check the browser console for details, rebuild with ./build_web.sh --dev, and reload.",
		err.Error(),
	)
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
