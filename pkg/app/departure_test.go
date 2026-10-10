package app

import (
	"testing"

	"github.com/adsouza/africa2ice/pkg/gameapi"
	"github.com/adsouza/africa2ice/pkg/hud"
	"github.com/adsouza/africa2ice/pkg/ui"
	"github.com/hajimehoshi/ebiten/v2"
)

func TestDepartureStartsOnlyAfterAnAcceptedDestination(t *testing.T) {
	for _, kind := range []string{"keyboard", "move-here", "best-tile", "map-destination"} {
		t.Run(kind, func(t *testing.T) {
			stub := &gameStub{frame: migrationPreviewFrame()}
			game := New(stub)
			game.disclosure.choose(ui.RowMove)
			game.handleDirectionalMigration(0, -1)
			if game.scene.DepartureActive() || len(stub.appliedCommands) != 0 {
				t.Fatal("previewing an ineligible tile started departure")
			}
			game.handleDirectionalMigration(-1, 0)
			if game.preview.TileID != 2 || !game.preview.Visible || game.scene.DepartureActive() || len(stub.appliedCommands) != 0 {
				t.Fatal("previewing a valid destination started departure before confirmation")
			}
			switch kind {
			case "keyboard":
				game.handleRowKey(ebiten.KeyEnter, false)
			case "move-here":
				game.handleIntent(hud.Intent{Kind: hud.IntentMoveTo, Tile: 2})
			case "best-tile":
				game.handleIntent(hud.Intent{Kind: hud.IntentMoveToBest})
			case "map-destination":
				game.tryQueueMigration(game.selected(), 2)
			}
			if !game.scene.DepartureActive() || len(stub.appliedCommands) != 1 || stub.endTurns != 0 || game.frame.Turn != 0 || game.frame.Bands[0].TileID != 0 {
				t.Fatal("accepted destination did not start presentation without resolving movement")
			}
			// The vignette must not block the next planning command.
			game.handleIntent(hud.Intent{Kind: hud.IntentChooseResearch, Tech: gameapi.Firecraft})
			if len(stub.appliedCommands) != 2 || !game.scene.DepartureActive() {
				t.Fatal("departure blocked research or a planning refresh cleared it")
			}
		})
	}
	for _, rejectedByPort := range []bool{false, true} {
		stub := &gameStub{frame: migrationPreviewFrame()}
		game := New(stub)
		target := gameapi.TileID(1)
		if rejectedByPort {
			target, stub.applyErrorAt = 2, 1
		}
		game.handleIntent(hud.Intent{Kind: hud.IntentMoveTo, Tile: target})
		if game.scene.DepartureActive() {
			t.Fatal("a rejected migration started departure")
		}
	}
}

func TestDepartureClearsOnSelectionTurnAndCampaignReplacement(t *testing.T) {
	for _, change := range []string{"chip", "tab", "map-band", "end-turn", "load", "new-campaign"} {
		t.Run(change, func(t *testing.T) {
			frame := migrationPreviewFrame()
			frame.Bands = append(frame.Bands, gameapi.Band{ID: 8, TileID: 1, Species: gameapi.HomoSapiens, Population: 20})
			game := New(&gameStub{frame: frame})
			game.selectedBand = 7
			if !game.tryQueueMigration(game.selected(), 2) {
				t.Fatal("fixture could not queue departure")
			}
			switch change {
			case "chip":
				game.selectBandByID(8)
			case "tab":
				game.selectNextSapiens()
			case "map-band":
				game.selectBandAtTile(1)
			case "end-turn":
				game.endTurn(true)
			case "load":
				// Even a save from the same turn/band must clear transient art.
				game.installStoredFrame(frame, false)
			case "new-campaign":
				game.startNewCampaign()
			}
			if game.scene.DepartureActive() {
				t.Fatalf("departure survived %s", change)
			}
		})
	}
}
