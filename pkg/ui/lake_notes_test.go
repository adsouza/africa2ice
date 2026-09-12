package ui

import (
	"strings"
	"testing"

	"github.com/adsouza/africa2ice/pkg/gameapi"
)

func lakeNoteFrames() (*gameapi.Frame, *gameapi.Frame) {
	before := &gameapi.Frame{Turn: 10, YearBP: 60100}
	after := &gameapi.Frame{Turn: 11, YearBP: 59900, Tiles: []gameapi.Tile{
		{X: 20, Y: 20, Explored: true},
		{X: 21, Y: 21, Explored: true, NearbyLake: "Lake Malawi / Nyasa"},
	}, Bands: []gameapi.Band{{Species: gameapi.HomoSapiens, Population: 100, TileID: 0}}}
	after.Tiles[1].Lakes = []gameapi.LakeShape{{Name: "Lake Malawi / Nyasa", Points: []gameapi.LakePoint{{X: 0, Y: 0}, {X: 1, Y: 0}, {X: 1, Y: 1}}}}
	before.Tiles = append([]gameapi.Tile(nil), after.Tiles...)
	before.Tiles[1].Lakes = []gameapi.LakeShape{{Name: "Lake Malawi / Nyasa", Points: []gameapi.LakePoint{{X: 0, Y: 0}, {X: 0.5, Y: 0}, {X: 0.5, Y: 1}}}}
	return before, after
}

func TestLakeNotesRequireNearbyLivingSapiensAndDateCrossing(t *testing.T) {
	tests := []struct {
		name   string
		change func(*gameapi.Frame, *gameapi.Frame)
		want   int
	}{
		{"no actual outline change", func(b, a *gameapi.Frame) { b.Tiles[1].Lakes = a.Tiles[1].Lakes }, 0},
		{"newly explored only", func(b, a *gameapi.Frame) { b.Tiles[1].Explored = false }, 0},
		{"retreated from old shore", func(b, a *gameapi.Frame) { a.Tiles[1].Lakes = nil; a.Tiles[1].NearbyLake = "" }, 1},
		{"nearby", func(b, a *gameapi.Frame) {}, 1},
		{"distant", func(b, a *gameapi.Frame) { a.Tiles[0].X = 10 }, 0},
		{"archaic", func(b, a *gameapi.Frame) { a.Bands[0].Species = gameapi.ArchaicHominin }, 0},
		{"dead", func(b, a *gameapi.Frame) { a.Bands[0].Population = 0 }, 0},
		{"fog", func(b, a *gameapi.Frame) { a.Tiles[1].Explored = false }, 0},
		{"same turn", func(b, a *gameapi.Frame) { a.Turn = b.Turn }, 0},
		{"past marker", func(b, a *gameapi.Frame) { b.YearBP = 60000 }, 0},
		{"before marker", func(b, a *gameapi.Frame) { a.YearBP = 60050 }, 0},
		{"two nearby bands", func(b, a *gameapi.Frame) { a.Bands = append(a.Bands, a.Bands[0]) }, 1},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			before, after := lakeNoteFrames()
			tt.change(before, after)
			if got := NearbyLakeHistoryNotes(before, after); len(got) != tt.want {
				t.Fatalf("notes=%d, want %d", len(got), tt.want)
			}
		})
	}
	if len(NearbyLakeHistoryNotes(nil, &gameapi.Frame{})) != 0 {
		t.Fatal("load generated history")
	}
}

func TestLakeNotesCoverMilestonesWithLinkedResearchAndExplicitAbstraction(t *testing.T) {
	for _, entry := range lakeHistory {
		before, after := lakeNoteFrames()
		before.YearBP = gameapi.LakeStageStart(entry.stage) + 1
		after.YearBP = gameapi.LakeStageStart(entry.stage)
		after.Tiles[1].NearbyLake = entry.name
		before.Tiles[1].NearbyLake = entry.name
		after.Tiles[1].Lakes[0].Name = entry.name
		before.Tiles[1].Lakes[0].Name = entry.name
		notes := NearbyLakeHistoryNotes(before, after)
		if len(notes) != 1 {
			t.Fatalf("missing milestone %s %d", entry.name, gameapi.LakeStageStart(entry.stage))
		}
		note := notes[0]
		assertFieldNoteHasNoManualLineBreaks(t, note)
		if !strings.Contains(FieldNoteReferenceMarkup(note.References), "[link=https://doi.org/") {
			t.Fatal("unlinked research")
		}
		if !strings.Contains(note.GameEffect, "different schematic shoreline") {
			t.Fatal("missing explanation of schematic map change")
		}
	}
}
