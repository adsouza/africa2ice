package app

import (
	"strings"
	"testing"

	"github.com/adsouza/africa2ice/pkg/gameapi"
)

func TestLakeNoteQueuesBehindRegionAndDoesNotRepeat(t *testing.T) {
	before := migrationPreviewFrame()
	before.YearBP = 60100
	before.Tiles[before.Bands[0].TileID].NearbyLake = "Lake Malawi / Nyasa"
	before.Tiles[before.Bands[0].TileID].Explored = true
	before.Bands[0].Species = gameapi.HomoSapiens
	before.Bands[0].Population = 100
	after := cloneAppFrame(before)
	after.Turn++
	after.YearBP = 59900
	after.SapiensEstablishedRegions = append(after.SapiensEstablishedRegions, gameapi.Arabia)
	game := New(&gameStub{frame: before})
	game.acceptCompletedTurn(after)
	if len(game.pendingLakeNotes) != 1 {
		t.Fatal("nearby lake context was lost behind region discovery")
	}
	next := cloneAppFrame(after)
	next.Turn++
	next.YearBP -= 200
	game.acceptCompletedTurn(next)
	if !strings.Contains(game.fieldNote.Topic, "Lake Malawi") || len(game.pendingLakeNotes) != 0 {
		t.Fatal("deferred lake note was not surfaced")
	}
	next2 := cloneAppFrame(next)
	next2.Turn++
	next2.YearBP -= 200
	game.acceptCompletedTurn(next2)
	if len(game.pendingLakeNotes) != 0 {
		t.Fatal("lake milestone repeated")
	}
}
