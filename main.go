//go:build !js

package main

import (
	"flag"
	"fmt"
	"image"
	"image/png"
	"io"
	"os"
	"strings"

	"github.com/adsouza/africa2ice/internal/adapters/logging"
	"github.com/adsouza/africa2ice/internal/verification"
	"github.com/adsouza/africa2ice/pkg/app"
	"github.com/hajimehoshi/ebiten/v2"
)

const desktopWindowFillPercent = 90

func main() {
	os.Exit(run(os.Args[1:], os.Stdout, os.Stderr))
}

type desktopOptions struct {
	dumpMap        bool
	headless       bool
	turns          int
	seed           uint64
	policy         string
	checkpointJSON string
	screenshot     string
}

func run(args []string, stdout, stderr io.Writer) int {
	options, err := parseDesktopOptions(args, stderr)
	if err != nil {
		return 2
	}
	session, err := logging.NewDefaultSession(stdout)
	if err != nil {
		_, _ = fmt.Fprintln(stderr, err)
		return 1
	}
	defer func() { _ = session.Close() }()
	defer logging.GuardPanic(session)
	if options.dumpMap {
		return runDumpMap(options.seed, stdout, stderr)
	}
	if options.headless || options.checkpointJSON != "" {
		return runReference(options, stdout, stderr)
	}
	if options.screenshot != "" {
		return runScreenshot(options, session, stderr)
	}
	windowWidth, windowHeight := initialDesktopWindowSize(0, 0)
	if monitor := ebiten.Monitor(); monitor != nil {
		windowWidth, windowHeight = initialDesktopWindowSize(monitor.Size())
	}
	ebiten.SetWindowSize(windowWidth, windowHeight)
	ebiten.SetWindowResizingMode(ebiten.WindowResizingModeEnabled)
	ebiten.SetWindowClosingHandled(true)
	ebiten.SetScreenClearedEveryFrame(false)
	ebiten.SetWindowTitle("Africa 2 Ice: Paleolithic Dispersal")
	game, err := app.NewGame(0x9e3779b97f4a7c15, session)
	if err == nil {
		err = ebiten.RunGame(game)
	}
	if err != nil {
		_, _ = fmt.Fprintln(stderr, err)
		return 1
	}
	return 0
}

func parseDesktopOptions(args []string, stderr io.Writer) (desktopOptions, error) {
	options := desktopOptions{turns: verification.MaxTurns, seed: 0x9e3779b97f4a7c15, policy: "reference"}
	flags := flag.NewFlagSet("africa2ice", flag.ContinueOnError)
	flags.SetOutput(stderr)
	flags.BoolVar(&options.dumpMap, "dumpmap", false, "print biome and region map layers without opening a window")
	flags.BoolVar(&options.headless, "headless", false, "run a deterministic reference campaign without opening a window")
	flags.IntVar(&options.turns, "turns", options.turns, "number of turns for a verification run")
	flags.Uint64Var(&options.seed, "seed", options.seed, "campaign seed (decimal or 0x-prefixed)")
	flags.StringVar(&options.policy, "policy", options.policy, "route policy: "+strings.Join(verification.PolicyNames(), ", "))
	flags.StringVar(&options.checkpointJSON, "checkpoint-json", "", "write canonical checkpoint JSON to this path")
	flags.StringVar(&options.screenshot, "screenshot", "", "write one rendered PNG to this path and exit")
	if err := flags.Parse(args); err != nil {
		return desktopOptions{}, err
	}
	if flags.NArg() != 0 {
		return desktopOptions{}, fmt.Errorf("unexpected arguments: %s", strings.Join(flags.Args(), " "))
	}
	if options.turns < 0 || options.turns > verification.MaxTurns {
		return desktopOptions{}, fmt.Errorf("turns must be in [0,%d]", verification.MaxTurns)
	}
	if _, err := verification.ParsePolicy(options.policy); err != nil {
		return desktopOptions{}, err
	}
	modeCount := 0
	for _, enabled := range []bool{options.dumpMap, options.headless || options.checkpointJSON != "", options.screenshot != ""} {
		if enabled {
			modeCount++
		}
	}
	if modeCount > 1 {
		return desktopOptions{}, fmt.Errorf("dumpmap, headless/checkpoint, and screenshot modes are mutually exclusive")
	}
	return options, nil
}

func runScreenshot(options desktopOptions, session *logging.Session, stderr io.Writer) int {
	port, _, err := verification.PrepareGame(options.seed, options.turns, options.policy)
	if err != nil {
		_, _ = fmt.Fprintln(stderr, err)
		return 1
	}
	game := app.New(logging.DecorateGame(session, port))
	runner := &screenshotRunner{game: game, path: options.screenshot}
	ebiten.SetScreenClearedEveryFrame(false)
	ebiten.SetWindowSize(app.LogicalWidth, app.LogicalHeight)
	ebiten.SetWindowTitle("Africa 2 Ice: screenshot verification")
	if err := ebiten.RunGame(runner); err != nil {
		_, _ = fmt.Fprintln(stderr, err)
		return 1
	}
	return 0
}

type screenshotRunner struct {
	game     *app.Game
	path     string
	captured bool
	err      error
}

func (runner *screenshotRunner) Update() error {
	if runner.err != nil {
		return runner.err
	}
	if runner.captured {
		return ebiten.Termination
	}
	return nil
}

func (runner *screenshotRunner) Draw(screen *ebiten.Image) {
	runner.game.Draw(screen)
	if runner.captured || runner.err != nil {
		return
	}
	bounds := screen.Bounds()
	pixels := make([]byte, 4*bounds.Dx()*bounds.Dy())
	screen.ReadPixels(pixels)
	frame := image.NewRGBA(bounds)
	copy(frame.Pix, pixels)
	file, err := os.Create(runner.path)
	if err == nil {
		err = png.Encode(file, frame)
		closeErr := file.Close()
		if err == nil {
			err = closeErr
		}
	}
	runner.err = err
	runner.captured = err == nil
}

func (runner *screenshotRunner) LayoutF(_, _ float64) (float64, float64) {
	return app.LogicalWidth, app.LogicalHeight
}

func (runner *screenshotRunner) Layout(_, _ int) (int, int) {
	return app.LogicalWidth, app.LogicalHeight
}

var _ ebiten.LayoutFer = (*screenshotRunner)(nil)
var _ ebiten.Game = (*screenshotRunner)(nil)

func runDumpMap(seed uint64, stdout, stderr io.Writer) int {
	encoded, err := verification.DumpMap(seed)
	if err != nil {
		_, _ = fmt.Fprintln(stderr, err)
		return 1
	}
	_, _ = io.WriteString(stdout, encoded)
	return 0
}

func runReference(options desktopOptions, stdout, stderr io.Writer) int {
	records, err := verification.ReferenceRun(options.seed, options.turns, options.policy)
	if err != nil {
		_, _ = fmt.Fprintln(stderr, err)
		return 1
	}
	payload, err := verification.CanonicalJSON(records)
	if err != nil {
		_, _ = fmt.Fprintln(stderr, err)
		return 1
	}
	if options.checkpointJSON != "" {
		if err := os.WriteFile(options.checkpointJSON, append(payload, '\n'), 0o644); err != nil {
			_, _ = fmt.Fprintln(stderr, err)
			return 1
		}
	}
	_, _ = stdout.Write(append(payload, '\n'))
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
