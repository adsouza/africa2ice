package ui

import (
	"strings"
	"testing"

	"github.com/adsouza/africa2ice/pkg/gameapi"
)

func TestTechnologyFieldNotesCoverTheClosedCatalog(t *testing.T) {
	for technology := gameapi.Tech(0); technology < gameapi.TechCount; technology++ {
		note, ok := TechnologyFieldNote(technology, 7, 1)
		if !ok || note.Topic == "" || note.Introduction == "" || note.Context == "" || note.GameEffect == "" || note.Hint == "" {
			t.Fatalf("technology %d has incomplete Field Notes: %#v", technology, note)
		}
		for _, block := range []string{note.Introduction, note.Context, note.GameEffect, note.Hint} {
			for _, line := range strings.Split(block, "\n") {
				if len([]rune(line)) > 48 {
					t.Fatalf("technology %s has an overlong Field Notes line %q", technology, line)
				}
			}
		}
	}
	if _, ok := TechnologyFieldNote(gameapi.TechCount, 7, 1); ok {
		t.Fatal("out-of-range technology has Field Notes")
	}
}

func TestCampaignOverviewIdentifiesTheDenisovanBand(t *testing.T) {
	if note := CampaignOverviewFieldNote(); !strings.Contains(note.Hint, "Denisovan") {
		t.Fatalf("campaign overview does not identify the Denisovan band: %#v", note)
	}
}

func TestBandAndEventContextNotesUseAcceptedFrameValues(t *testing.T) {
	frame := &gameapi.Frame{Tiles: []gameapi.Tile{{ID: 0, Region: gameapi.EastAfrica, Biome: gameapi.Savanna, FloraStock: 20, FloraCap: 40, FaunaStock: 30, FaunaCap: 50, WaterStock: 10, WaterCap: 15, EcologicalK: 80, NaturalShelter: 0.25}}}
	band := &gameapi.Band{ID: 7, TileID: 0, Population: 120, Health: 0.9, StoredFood: 6.5}
	note := BandContextFieldNote(frame, band)
	if !strings.Contains(note.Topic, "BAND 7") || !strings.Contains(note.Introduction, "120 people") || !strings.Contains(note.GameEffect, "Food 50/90") {
		t.Fatalf("band context note = %#v", note)
	}
	event := EventFieldNote(gameapi.Event{Turn: 4, Kind: gameapi.EventAcuteIncident, BandID: 7, Summary: "A predator attacked."})
	if event.Topic != gameapi.EventAcuteIncident.String() || event.Introduction != "A predator attacked." || !strings.Contains(event.Context, "turn 4") {
		t.Fatalf("event note = %#v", event)
	}
}
