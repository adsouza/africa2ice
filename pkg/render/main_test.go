package render

import (
	"os"
	"testing"

	"github.com/hajimehoshi/ebiten/v2"
)

// testMainGame drives *testing.M inside a running Ebiten game loop so that
// (*ebiten.Image).At is available to tests — ReadPixels panics unless the
// game has actually started. See ebiten's own internal/testing package for
// the same pattern; it is not importable from outside the ebiten module.
type testMainGame struct {
	m    *testing.M
	code int
}

func (g *testMainGame) Update() error {
	g.code = g.m.Run()
	return ebiten.Termination
}

func (*testMainGame) Draw(*ebiten.Image) {}

func (*testMainGame) Layout(int, int) (int, int) {
	return 320, 240
}

func TestMain(m *testing.M) {
	g := &testMainGame{m: m, code: 1}
	if err := ebiten.RunGame(g); err != nil {
		panic(err)
	}
	os.Exit(g.code)
}
