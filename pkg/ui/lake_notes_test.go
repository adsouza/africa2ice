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
	return before, after
}

func TestLakeNotesRequireNearbyLivingSapiensAndDateCrossing(t *testing.T) {
	tests := []struct {
		name   string
		change func(*gameapi.Frame, *gameapi.Frame)
		want   int
	}{
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
		before.YearBP = entry.yearBP + 1
		after.YearBP = entry.yearBP
		after.Tiles[1].NearbyLake = entry.name
		notes := NearbyLakeHistoryNotes(before, after)
		if len(notes) != 1 {
			t.Fatalf("missing milestone %s %d", entry.name, entry.yearBP)
		}
		note := notes[0]
		assertFieldNoteHasNoManualLineBreaks(t, note)
		if !strings.Contains(FieldNoteReferenceMarkup(note.References), "[link=https://doi.org/") {
			t.Fatal("unlinked research")
		}
		if entry.yearBP != 70000 && !strings.Contains(note.GameEffect, "not simulated") {
			t.Fatal("historical change presented as simulation")
		}
	}
}
