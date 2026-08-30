//go:build !js

package main

import (
	"os"

	"github.com/adsouza/africa2ice/internal/adapters/logging"
	"github.com/adsouza/africa2ice/pkg/app"
	"github.com/hajimehoshi/ebiten/v2"
)

const desktopWindowFillPercent = 90

func main() {
	os.Exit(run())
}

func run() int {
	session, err := logging.NewDefaultSession(os.Stdout)
	if err != nil {
		_, _ = os.Stderr.WriteString(err.Error() + "\n")
		return 1
	}
	defer func() { _ = session.Close() }()
	windowWidth, windowHeight := initialDesktopWindowSize(0, 0)
	if monitor := ebiten.Monitor(); monitor != nil {
		windowWidth, windowHeight = initialDesktopWindowSize(monitor.Size())
	}
	ebiten.SetWindowSize(windowWidth, windowHeight)
	ebiten.SetWindowResizingMode(ebiten.WindowResizingModeEnabled)
	ebiten.SetWindowClosingHandled(true)
	ebiten.SetWindowTitle("Africa 2 Ice: Paleolithic Dispersal")
	game, err := app.NewGame(0x9e3779b97f4a7c15, session)
	if err == nil {
		err = ebiten.RunGame(game)
	}
	if err != nil {
		_, _ = os.Stderr.WriteString(err.Error() + "\n")
		return 1
	}
	return 0
}

func initialDesktopWindowSize(monitorWidth, monitorHeight int) (int, int) {
	if monitorWidth <= 0 || monitorHeight <= 0 {
		return app.LogicalWidth, app.LogicalHeight
	}

	availableWidth := monitorWidth * desktopWindowFillPercent / 100
	availableHeight := monitorHeight * desktopWindowFillPercent / 100
	windowWidth := availableWidth
	windowHeight := windowWidth * app.LogicalHeight / app.LogicalWidth
	if windowHeight > availableHeight {
		windowHeight = availableHeight
		windowWidth = windowHeight * app.LogicalWidth / app.LogicalHeight
	}
	if windowWidth <= 0 || windowHeight <= 0 {
		return app.LogicalWidth, app.LogicalHeight
	}
	return windowWidth, windowHeight
}
