package app

import (
	"bytes"
	"os"
	"strings"
	"testing"

	"github.com/adsouza/africa2ice/internal/adapters/logging"
	"github.com/adsouza/africa2ice/pkg/hud"
	"github.com/adsouza/africa2ice/pkg/render"
	"github.com/ebitenui/ebitenui/input"
	"github.com/hajimehoshi/ebiten/v2"
)

// openLoggedSession creates a real logging.Session backed by a temp JSONL
// file — the same seam production uses via logging.NewDefaultSession — and
// returns it plus a function that closes it and returns the file's content.
func openLoggedSession(t *testing.T) (*logging.Session, func() string) {
	t.Helper()
	var announce bytes.Buffer
	session, err := logging.NewDefaultSession(&announce)
	if err != nil {
		t.Fatalf("NewDefaultSession: %v", err)
	}
	_, path, ok := strings.Cut(strings.TrimSpace(announce.String()), ": ")
	if !ok {
		t.Fatalf("could not parse session log path from announcement: %q", announce.String())
	}
	return session, func() string {
		_ = session.Close()
		content, err := os.ReadFile(path)
		if err != nil {
			t.Fatalf("read session log: %v", err)
		}
		_ = os.Remove(path)
		return string(content)
	}
}

// TestUIIntentClickWritesUIIntentRecord covers Wave I item I0: a synthetic
// click that produces an intent (the same New Campaign click I1 drives)
// must leave a ui.intent record naming that kind in the session log — the
// gap that made the New Campaign bug undiagnosable from a log.
func TestUIIntentClickWritesUIIntentRecord(t *testing.T) {
	t.Cleanup(func() { input.SetCursorUpdater(nil) })

	session, closeAndRead := openLoggedSession(t)
	frame := terminalCampaignFrame(t)
	stub := &gameStub{frame: frame}
	game := New(stub)
	game.logSession = session

	left := render.EndSceneX + (render.EndSceneWidth-300)/2
	centerX, centerY := int(left+150), int(494+23)
	cursor := &clickCursor{x: centerX, y: centerY}
	input.SetCursorUpdater(cursor)
	screen := ebiten.NewImage(1280, 720)
	defer screen.Deallocate()

	if err := game.Update(); err != nil {
		t.Fatalf("initial Update: %v", err)
	}
	game.Draw(screen)

	cursor.pressed, cursor.justPressed = true, true
	if err := game.Update(); err != nil {
		t.Fatalf("press Update: %v", err)
	}
	game.Draw(screen)

	cursor.pressed, cursor.justPressed = false, false
	cursor.justReleased = true
	if err := game.Update(); err != nil {
		t.Fatalf("release Update: %v", err)
	}
	game.Draw(screen)

	logged := closeAndRead()
	for _, required := range []string{`"msg":"ui.intent"`, `"kind":"new-campaign"`} {
		if !strings.Contains(logged, required) {
			t.Fatalf("session log missing %q after the New Campaign click: %s", required, logged)
		}
	}
	if stub.newCampaigns != 1 {
		t.Fatalf("New Campaign requests = %d, want 1", stub.newCampaigns)
	}
}

// TestUIIntentRefusedOnTerminalCampaignWritesRefusalRecord covers Wave I
// item I0: a band action intent arriving while the campaign is over must be
// visible in the log as a deliberate refusal, not a silent no-op. I1 makes
// the terminal dialog a full-screen modal window, so a band action can no
// longer reach handleIntent via a real click once the campaign has ended —
// this drives handleIntent directly, exercising the same guard clause a
// stray or racing intent would hit.
func TestUIIntentRefusedOnTerminalCampaignWritesRefusalRecord(t *testing.T) {
	session, closeAndRead := openLoggedSession(t)
	frame := terminalCampaignFrame(t)
	stub := &gameStub{frame: frame}
	game := New(stub)
	game.logSession = session

	game.handleIntent(hud.Intent{Kind: hud.IntentSplit})

	logged := closeAndRead()
	for _, required := range []string{`"msg":"ui.intent_refused"`, `"kind":"split"`, `"reason":"campaign-over"`} {
		if !strings.Contains(logged, required) {
			t.Fatalf("session log missing %q after a refused band action: %s", required, logged)
		}
	}
}

// TestUIIntentLoggingWithNilSessionDoesNotPanic covers the session-less path
// pkg/app's own tests construct games without (New never installs a
// logSession): every internal/adapters/logging.Session method begins with a
// session == nil guard, and handleIntents/handleIntent must stay safe
// calling through a nil *logging.Session exactly like the existing
// LogActionDispatch/LogActionRejected call sites do.
func TestUIIntentLoggingWithNilSessionDoesNotPanic(t *testing.T) {
	frame := terminalCampaignFrame(t)
	stub := &gameStub{frame: frame}
	game := New(stub)
	if game.logSession != nil {
		t.Fatal("expected a nil logSession for a game constructed with New")
	}

	game.handleIntents([]hud.Intent{{Kind: hud.IntentCameraToggle}})
	game.handleIntent(hud.Intent{Kind: hud.IntentSplit})
}
