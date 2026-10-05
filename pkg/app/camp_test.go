package app

import (
	"testing"

	"github.com/adsouza/africa2ice/pkg/gameapi"
	"github.com/adsouza/africa2ice/pkg/hud"
	"github.com/adsouza/africa2ice/pkg/render"
	"github.com/adsouza/africa2ice/pkg/ui"
	"github.com/hajimehoshi/ebiten/v2"
)

func TestCampVisitPreservesDraftAndBlocksCampaignInput(t *testing.T) {
	stub := &gameStub{frame: migrationPreviewFrame()}
	game := New(stub)
	game.editAssignmentDraft(100)
	allocation, selected, frame := game.workforce.Allocation(), game.selectedBand, game.frame
	game.handleIntent(hud.Intent{Kind: hud.IntentViewCamp})
	if game.scenes.Current() != ui.SceneCamp || game.hudState().Overlay.Scene != ui.SceneCamp {
		t.Fatal("camp button did not open camp")
	}
	// A simultaneous or stale map intent must not escape the camp scene.
	game.handleIntent(hud.Intent{Kind: hud.IntentEndTurn, Force: true})
	game.handleIntent(hud.Intent{Kind: hud.IntentMoveTo, Tile: 2})
	game.handleSceneKeyState(keysDown(), keysDown(ebiten.KeySpace, ebiten.KeyTab, ebiten.KeyN))
	game.handleGameplayKeyState(keysDown(), keysDown(ebiten.KeySpace, ebiten.KeyTab))
	game.handleSceneKeyState(keysDown(), keysDown(ebiten.KeyEscape, ebiten.KeySpace))
	if game.scenes.Current() != ui.SceneGameplay || game.selectedBand != selected || game.frame != frame || game.workforce.Allocation() != allocation || !game.workforce.Dirty() || stub.endTurns != 0 || len(stub.appliedCommands) != 0 {
		t.Fatal("visiting camp changed the draft, selection, or campaign")
	}
	game.handleGameplayKeyState(keysDown(), keysDown(ebiten.KeyC))
	if game.scenes.Current() != ui.SceneCamp {
		t.Fatal("C did not reopen camp")
	}
	game.handleIntent(hud.Intent{Kind: hud.IntentBack})
	if game.scenes.Current() != ui.SceneGameplay {
		t.Fatal("return button did not restore gameplay")
	}
}

func TestCampRequiresASelectedBandAndAnOngoingCampaign(t *testing.T) {
	frame := migrationPreviewFrame()
	game := New(&gameStub{frame: frame})
	game.selectedBand = 999
	game.openCamp()
	if game.scenes.Current() != ui.SceneGameplay {
		t.Fatal("opened camp without a band")
	}
	game.selectedBand = frame.Bands[0].ID
	frame.CampaignResult = gameapi.DispersalFailed
	game.openCamp()
	if game.scenes.Current() != ui.SceneGameplay {
		t.Fatal("camp hid the campaign ending")
	}
}

func TestWorkforceCampKeepsTheMapCacheAndDirtyDraft(t *testing.T) {
	stub := &gameStub{frame: migrationPreviewFrame()}
	game := New(stub)
	game.disclosure.openRow = ui.RowWorkforce
	game.editAssignmentDraft(100)
	allocation, frame := game.workforce.Allocation(), game.frame
	// Freeze map shimmer independently, so only the miniature's clock advances.
	game.scene.SetReducedMotion(true)
	for range render.CameraTransitionTicks + 2 {
		game.Update()
	}
	screen := ebiten.NewImage(1280, 720)
	defer screen.Deallocate()
	game.Draw(screen)
	game.Update()
	game.Draw(screen)
	paints := game.scene.Paints
	for range 12 {
		game.Update()
		game.Draw(screen)
	}
	if game.scene.Paints != paints {
		t.Fatal("miniature animation invalidated the map cache")
	}
	if game.workforce.Allocation() != allocation || !game.workforce.Dirty() || game.frame != frame || len(stub.appliedCommands) != 0 {
		t.Fatal("miniature changed the workforce draft or campaign")
	}
}
