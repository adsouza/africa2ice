package app

import (
	"encoding/json"
	"testing"

	"github.com/adsouza/africa2ice/pkg/gameapi"
)

func TestE2ESummaryUsesOnlyThePublishedFrame(t *testing.T) {
	frame := &gameapi.Frame{
		WorldRevision: 7, Turn: 3, YearBP: 79_100, Era: gameapi.EraEarly, CalendarProgress: 0.1,
		Tiles: []gameapi.Tile{{ID: 0, Explored: true}, {ID: 1}, {ID: 2, Explored: true}},
		Bands: []gameapi.Band{
			{ID: 9, Species: gameapi.HomoSapiens, Population: 40},
			{ID: 2, Species: gameapi.ArchaicHominin, Population: 80},
			{ID: 4, Species: gameapi.HomoSapiens, TileID: 2, Population: 60, Health: 0.75, StoredFood: 12,
				LastFoodReport:    gameapi.FoodTurnReport{Turn: 3, RequiredFU: 60, DeficitFU: 2},
				LastOutcomeReport: gameapi.OutcomeReport{Turn: 3, StartingPopulation: 61, EndingPopulation: 60}},
		},
	}
	encoded := e2eSummaryJSON(frame)
	var summary E2ESummary
	if err := json.Unmarshal([]byte(encoded), &summary); err != nil {
		t.Fatal(err)
	}
	if summary.Turn != 3 || summary.WorldRevision != 7 || summary.SapiensPopulation != 100 || !summary.FirstSapiensBand.Present || summary.FirstSapiensBand.ID != 4 {
		t.Fatalf("summary = %#v", summary)
	}
	if len(summary.ExploredTileIDs) != 2 || summary.ExploredTileIDs[0] != 0 || summary.ExploredTileIDs[1] != 2 || len(summary.ExploredHash) != 64 {
		t.Fatalf("exploration summary = %#v", summary)
	}
}

func TestFrameObserverPublishesWithoutCallingTheGamePort(t *testing.T) {
	frame := migrationPreviewFrame()
	stub := &gameStub{frame: frame}
	game := New(stub)
	var encoded string
	game.SetFramePublishedCallback(func(summary string) { encoded = summary })
	if encoded == "" || stub.appliedCommand != nil || stub.nextStorageID != 0 {
		t.Fatalf("observer result = %q, stub = %#v", encoded, stub)
	}
}
