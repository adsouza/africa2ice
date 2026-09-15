//go:build !js

package main

import (
	"bytes"
	"errors"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/adsouza/africa2ice/internal/adapters/logging"
	"github.com/adsouza/africa2ice/pkg/app"
	"github.com/hajimehoshi/ebiten/v2"
)

func TestInitialDesktopWindowSize(t *testing.T) {
	tests := []struct {
		name                  string
		monitorWidth          int
		monitorHeight         int
		wantWidth, wantHeight int
	}{
		{name: "full HD", monitorWidth: 1920, monitorHeight: 1080, wantWidth: 1728, wantHeight: 972},
		{name: "4K", monitorWidth: 3840, monitorHeight: 2160, wantWidth: 3456, wantHeight: 1944},
		{name: "ultrawide is height limited", monitorWidth: 3440, monitorHeight: 1440, wantWidth: 2304, wantHeight: 1296},
		{name: "small monitor bottoms out", monitorWidth: 1024, monitorHeight: 768, wantWidth: app.LogicalWidth, wantHeight: app.LogicalHeight},
		{name: "invalid monitor falls back", monitorWidth: 0, monitorHeight: 0, wantWidth: app.LogicalWidth, wantHeight: app.LogicalHeight},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			width, height := initialDesktopWindowSize(test.monitorWidth, test.monitorHeight)
			if width != test.wantWidth || height != test.wantHeight {
				t.Fatalf("initialDesktopWindowSize(%d, %d) = %dx%d, want %dx%d", test.monitorWidth, test.monitorHeight, width, height, test.wantWidth, test.wantHeight)
			}
		})
	}
}

func TestParseDesktopOptions(t *testing.T) {
	var stderr bytes.Buffer
	options, err := parseDesktopOptions([]string{"-headless", "-turns", "100", "-seed", "0x2a", "-policy", "toward-sahul", "-checkpoint-json", "result.json"}, &stderr)
	if err != nil {
		t.Fatal(err)
	}
	if !options.headless || options.turns != 100 || options.seed != 42 || options.policy != "toward-sahul" || options.checkpointJSON != "result.json" {
		t.Fatalf("options = %#v", options)
	}
}

func TestParseDesktopOptionsRejectsInvalidVerificationInputs(t *testing.T) {
	for _, args := range [][]string{{"-turns", "401"}, {"-policy", "wandering"}, {"extra"}} {
		var stderr bytes.Buffer
		if _, err := parseDesktopOptions(args, &stderr); err == nil {
			t.Fatalf("parseDesktopOptions(%q) accepted invalid input", strings.Join(args, " "))
		}
	}
	for _, args := range [][]string{{"-terrain-detail", "low"}, {"-dumpmap", "-headless"}, {"-screenshot", "x.png", "-checkpoint-json", "x.json"}} {
		var stderr bytes.Buffer
		if _, err := parseDesktopOptions(args, &stderr); err == nil {
			t.Fatalf("parseDesktopOptions(%q) accepted conflicting mode", strings.Join(args, " "))
		}
	}
}

func TestRunReferenceWritesTheSameCanonicalBytesItPrints(t *testing.T) {
	path := filepath.Join(t.TempDir(), "reference-checkpoints.json")
	options := desktopOptions{headless: true, turns: 0, seed: 42, policy: "reference", checkpointJSON: path}
	var stdout, stderr bytes.Buffer
	if code := runReference(options, &stdout, &stderr); code != 0 {
		t.Fatalf("runReference code = %d, stderr = %s", code, stderr.String())
	}
	written, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(written, stdout.Bytes()) || !bytes.HasSuffix(written, []byte("\n")) {
		t.Fatalf("file/stdout differ:\nfile %q\nout %q", written, stdout.Bytes())
	}
}

func TestScreenshotFramesDefaultsToOne(t *testing.T) {
	options, err := parseDesktopOptions([]string{"-screenshot", "shot.png"}, &bytes.Buffer{})
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	if options.screenshotFrames != 1 {
		t.Fatalf("screenshotFrames = %d, want 1", options.screenshotFrames)
	}
}

func TestScreenshotFramesRejectsNonPositiveCounts(t *testing.T) {
	if _, err := parseDesktopOptions([]string{"-screenshot", "shot.png", "-screenshot-frames", "0"}, &bytes.Buffer{}); err == nil {
		t.Fatal("expected an error for a zero frame count")
	}
}

func TestScreenshotFramesRequiresAScreenshotPath(t *testing.T) {
	if _, err := parseDesktopOptions([]string{"-screenshot-frames", "8"}, &bytes.Buffer{}); err == nil {
		t.Fatal("expected an error when frames are requested with no screenshot path")
	}
}

func TestNumberedScreenshotPaths(t *testing.T) {
	tests := []struct {
		path  string
		index int
		want  string
	}{
		{path: "shot.png", index: 0, want: "shot.png"},
		{path: "shot.png", index: 1, want: "shot-002.png"},
		{path: "out/dir.d/shot.png", index: 11, want: "out/dir.d/shot-012.png"},
		{path: "noext", index: 2, want: "noext-003"},
	}
	for _, test := range tests {
		if got := numberedScreenshotPath(test.path, test.index); got != test.want {
			t.Errorf("numberedScreenshotPath(%q, %d) = %q, want %q", test.path, test.index, got, test.want)
		}
	}
}

// TestParseDesktopOptionsAcceptsNoSound covers the AerynOS report: a machine
// whose ALSA device cannot be opened aborted RunGame on the first click, so a
// player needs a way to start the game without ever constructing a device.
func TestParseDesktopOptionsAcceptsNoSound(t *testing.T) {
	var stderr bytes.Buffer
	options, err := parseDesktopOptions([]string{"-no-sound"}, &stderr)
	if err != nil {
		t.Fatal(err)
	}
	if !options.noSound {
		t.Fatalf("-no-sound did not set noSound: %#v", options)
	}
	defaults, err := parseDesktopOptions(nil, &stderr)
	if err != nil {
		t.Fatal(err)
	}
	if defaults.noSound {
		t.Fatalf("noSound defaulted to true: %#v", defaults)
	}
}

type closeRecorder struct {
	closes int
	err    error
}

func (c *closeRecorder) Close() error { c.closes++; return c.err }

// TestRunRoutesTheGameThroughTheLifetimeSeam asserts the wiring, not the helper.
// app.Game.Close has unit tests that construct a game and close it directly; those
// stay green if run() stops closing at all, which is how a log decorator went
// unwired two commits ago. Stubbing the seam proves run() reaches the one path
// that closes.
func TestRunRoutesTheGameThroughTheLifetimeSeam(t *testing.T) {
	previousNew, previousLoop, previousRun := newGameWithSound, runGameLoop, runGame
	t.Cleanup(func() { newGameWithSound, runGameLoop, runGame = previousNew, previousLoop, previousRun })

	// The entry-point wiring test needs identity, not persisted campaign state.
	built := app.NewWalkingSkeleton()
	newGameWithSound = func(uint64, *logging.Session) (*app.Game, error) { return built, nil }
	runGame = func(ebiten.Game) error { t.Error("run() reached the real loop instead of the seam"); return nil }
	routed, looped := 0, 0
	runGameLoop = func(game closable, loop func() error) error {
		routed++
		if game != closable(built) {
			t.Error("seam received a different game than run() constructed")
		}
		looped++
		return nil
	}
	if code := run(nil, io.Discard, io.Discard); code != 0 {
		t.Fatalf("run exited %d", code)
	}
	if routed != 1 || looped != 1 {
		t.Fatalf("run() invoked the lifetime seam %d times", routed)
	}
}

// TestDefaultRunGameLoopAlwaysCloses covers the seam's own contract: the host is
// released on the ordinary path and while a panicking loop unwinds, and neither
// the loop's error nor the close's error is dropped.
func TestDefaultRunGameLoopAlwaysCloses(t *testing.T) {
	loopErr, closeErr := errors.New("loop failed"), errors.New("close failed")

	recorder := &closeRecorder{err: closeErr}
	err := defaultRunGameLoop(recorder, func() error { return loopErr })
	if !errors.Is(err, loopErr) || !errors.Is(err, closeErr) {
		t.Fatalf("joined error = %v, want both loop and close failures", err)
	}
	if recorder.closes == 0 {
		t.Fatal("ordinary return did not close the host")
	}

	panicking := &closeRecorder{}
	boom := errors.New("loop panicked")
	recovered := func() (value any) {
		defer func() { value = recover() }()
		_ = defaultRunGameLoop(panicking, func() error { panic(boom) })
		return nil
	}()
	if recovered != any(boom) {
		t.Fatalf("panic = %v, want it to propagate", recovered)
	}
	if panicking.closes == 0 {
		t.Fatal("panicking loop leaked the host")
	}
}
