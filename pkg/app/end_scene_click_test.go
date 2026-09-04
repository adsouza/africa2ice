package app

import (
	"image"
	"os"
	"testing"

	"github.com/adsouza/africa2ice/internal/application"
	"github.com/adsouza/africa2ice/pkg/gameapi"
	"github.com/adsouza/africa2ice/pkg/hud"
	"github.com/adsouza/africa2ice/pkg/render"
	"github.com/ebitenui/ebitenui/input"
	"github.com/hajimehoshi/ebiten/v2"
)

// clickCursor is a synthetic input.CursorUpdater that can drive a real press
// and release, unlike pkg/hud's testCursor (which only ever reports the
// cursor position). ebitenui's own widgets read press state through
// MouseButtonPressed/MouseButtonJustPressed/MouseButtonJustReleased rather
// than any lower-level ebiten API, so a fixture that answers those three
// methods can inject presses and releases exactly like real input would — an
// earlier wave concluded this was impossible; it is not.
type clickCursor struct {
	x, y                               int
	pressed, justPressed, justReleased bool
}

func (c *clickCursor) Update()                             {}
func (c *clickCursor) AfterUpdate()                        {}
func (c *clickCursor) Draw(*ebiten.Image)                  {}
func (c *clickCursor) AfterDraw(*ebiten.Image)             {}
func (c *clickCursor) CursorPosition() (int, int)          { return c.x, c.y }
func (c *clickCursor) GetCursorImage(string) *ebiten.Image { return nil }
func (c *clickCursor) GetCursorOffset(string) image.Point  { return image.Point{} }

func (c *clickCursor) MouseButtonPressed(b ebiten.MouseButton) bool {
	return b == ebiten.MouseButtonLeft && c.pressed
}
func (c *clickCursor) MouseButtonJustPressed(b ebiten.MouseButton) bool {
	return b == ebiten.MouseButtonLeft && c.justPressed
}
func (c *clickCursor) MouseButtonJustReleased(b ebiten.MouseButton) bool {
	return b == ebiten.MouseButtonLeft && c.justReleased
}

// terminalCampaignFrame loads the 256-band performance fixture and marks it
// as a failed dispersal, so the test drives a genuine terminal frame rather
// than a hand-rolled one-band stub.
func terminalCampaignFrame(t *testing.T) *gameapi.Frame {
	t.Helper()
	payload, err := os.ReadFile("../../testdata/performance_profile_save.json")
	if err != nil {
		t.Fatal(err)
	}
	state, err := application.DecodeSaveState(payload)
	if err != nil {
		t.Fatal(err)
	}
	frame, err := application.ProjectSaveState(state)
	if err != nil {
		t.Fatal(err)
	}
	frame.CampaignResult = gameapi.DispersalFailed
	return frame
}

// TestEndSceneClickReachesNewCampaignOverTheDrawer reproduces the
// user-reported bug (Wave I item I1): clicking New Campaign on the game-over
// dialog did nothing while N worked. A synthetic cursor driving Game.Update
// showed the click never reached the button when the Field Notes drawer was
// expanded (its ScrollContainer elevates its own content to an input layer
// covering x 20–884, y 382–700 with BlockLower set, which swallows the
// button's clicks at (302–602, 494–540) even though the end scene is added
// to the root after the drawer) but did reach it when the drawer was
// compact. The fix makes the end scene a modal widget.Window exactly like
// buildOverlay's title/menu/storage/settings windows, so it always wins.
func TestEndSceneClickReachesNewCampaignOverTheDrawer(t *testing.T) {
	t.Cleanup(func() { input.SetCursorUpdater(nil) })

	frame := terminalCampaignFrame(t)
	// The button's own rect (spec, and hud.buildEndScene): centred under
	// render.EndSceneX/Width, at y 494 with height 46. The viewport is left
	// at its zero value below, so render.FitPresentation reports Scale 1 and
	// zero offsets and this DIP rect is also the render-pixel rect ebitenui
	// hit-tests against.
	left := render.EndSceneX + (render.EndSceneWidth-300)/2
	centerX, centerY := int(left+150), int(494+23)

	for _, mode := range []hud.NotesMode{hud.NotesCompact, hud.NotesExpanded} {
		stub := &gameStub{frame: frame}
		game := New(stub)
		game.notesMode = mode

		cursor := &clickCursor{x: centerX, y: centerY}
		input.SetCursorUpdater(cursor)
		screen := ebiten.NewImage(1280, 720)

		// First Update/Draw pair lays the tree out (widget rects are
		// published during Draw, not Update) with the cursor already
		// positioned but not yet pressed.
		if err := game.Update(); err != nil {
			t.Fatalf("notes mode %v: initial Update: %v", mode, err)
		}
		game.Draw(screen)

		// Press tick: MouseButtonJustPressed fires the widget's pressed
		// event while the cursor is inside its rect.
		cursor.pressed, cursor.justPressed = true, true
		if err := game.Update(); err != nil {
			t.Fatalf("notes mode %v: press Update: %v", mode, err)
		}
		game.Draw(screen)

		// Release tick: MouseButtonPressed going false while the widget
		// still thinks it is pressed fires the released+clicked events.
		cursor.pressed, cursor.justPressed = false, false
		cursor.justReleased = true
		if err := game.Update(); err != nil {
			t.Fatalf("notes mode %v: release Update: %v", mode, err)
		}
		game.Draw(screen)

		screen.Deallocate()

		if stub.newCampaigns != 1 {
			t.Fatalf("notes mode %v: New Campaign requests = %d, want 1 — the click at %v did not reach the button", mode, stub.newCampaigns, image.Pt(centerX, centerY))
		}
	}
}
