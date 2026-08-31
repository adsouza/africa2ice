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
	fishing, _ := TechnologyFieldNote(gameapi.CordageAndNets, 7, 1)
	combined := fishing.Context + " " + fishing.Hint
	for _, required := range []string{"~90 ka", "~42 ka", "23–16 ka", "post-campaign"} {
		if !strings.Contains(combined, required) {
			t.Fatalf("fishing chronology omits %q: %#v", required, fishing)
		}
	}
}

func TestCampaignOverviewIdentifiesTheDenisovanBand(t *testing.T) {
	if note := CampaignOverviewFieldNote(); !strings.Contains(note.Hint, "Denisovan") {
		t.Fatalf("campaign overview does not identify the Denisovan band: %#v", note)
	}
}

func TestTraitAndRegionFieldNotesCoverTheirClosedCatalogs(t *testing.T) {
	for trait := gameapi.HeritableTrait(0); trait < gameapi.HeritableTraitCount; trait++ {
		note, ok := TraitFieldNote(trait, 0.5)
		if !ok || note.Topic == "" || note.Introduction == "" || note.Context == "" || note.GameEffect == "" || note.Hint == "" {
			t.Fatalf("trait %d has incomplete Field Notes: %#v", trait, note)
		}
	}
	for region := gameapi.Region(0); region < gameapi.RegionCount; region++ {
		note, ok := RegionEstablishedFieldNote(region)
		if !ok || note.Topic == "" || note.Context == "" || note.GameEffect == "" || note.Hint == "" {
			t.Fatalf("region %d has incomplete Field Notes: %#v", region, note)
		}
	}
	if _, ok := TraitFieldNote(gameapi.HeritableTraitCount, 0.5); ok {
		t.Fatal("out-of-range trait has Field Notes")
	}
	if _, ok := RegionEstablishedFieldNote(gameapi.RegionCount); ok {
		t.Fatal("out-of-range region has Field Notes")
	}
}

func TestCampanianFieldNoteLabelsTheSimulationEnvelope(t *testing.T) {
	note, ok := MacroEpisodeFieldNote(gameapi.MacroEpisodeSummary{Episode: gameapi.CampanianIgnimbrite, Warned: true})
	if !ok || !strings.Contains(note.Context, "39,850") || !strings.Contains(note.GameEffect, "envelope") || !strings.Contains(note.Introduction, "warning") {
		t.Fatalf("Campanian note = %#v", note)
	}
}

func TestClimateAndTobaNotesSeparateContextFromGameplay(t *testing.T) {
	for epoch := gameapi.ClimateEpoch(0); epoch < gameapi.ClimateEpochCount; epoch++ {
		note, ok := ClimateEpochFieldNote(epoch)
		if !ok || !strings.Contains(note.Context, "Lisiecki") || !strings.Contains(note.GameEffect, "palette") {
			t.Fatalf("climate epoch %d note = %#v", epoch, note)
		}
	}
	if _, ok := ClimateEpochFieldNote(gameapi.ClimateEpochCount); ok {
		t.Fatal("out-of-range climate epoch has Field Notes")
	}
	toba := TobaFieldNote()
	if !strings.Contains(toba.Context+toba.Hint, "Storey") || !strings.Contains(toba.GameEffect, "no effect") {
		t.Fatalf("Toba note does not separate evidence and gameplay: %#v", toba)
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
