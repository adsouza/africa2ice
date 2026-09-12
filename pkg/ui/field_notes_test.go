package ui

import (
	"strings"
	"testing"

	"github.com/adsouza/africa2ice/pkg/gameapi"
	"github.com/adsouza/africa2ice/pkg/render"
)

func TestTechnologyFieldNotesCoverTheClosedCatalog(t *testing.T) {
	for technology := gameapi.Tech(0); technology < gameapi.TechCount; technology++ {
		note, ok := TechnologyFieldNote(technology, 7, 1)
		if !ok || note.Topic == "" || note.Introduction == "" || note.Context == "" || note.GameEffect == "" || note.Hint == "" || note.References == "" {
			t.Fatalf("technology %d has incomplete Field Notes: %#v", technology, note)
		}
		assertFieldNoteHasNoManualLineBreaks(t, note)
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
	multiple, _ := TechnologyFieldNote(gameapi.Firecraft, 7, 2)
	assertFieldNoteHasNoManualLineBreaks(t, multiple)
}

func TestFieldNotesCoverClosedContextCatalogs(t *testing.T) {
	assertComplete := func(name string, note render.FieldNote, ok bool) {
		t.Helper()
		if !ok || note.Topic == "" || note.Introduction == "" || note.Context == "" || note.GameEffect == "" || note.Hint == "" || note.References == "" {
			t.Fatalf("%s has incomplete Field Notes: %#v", name, note)
		}
		assertFieldNoteHasNoManualLineBreaks(t, note)
	}
	for biome := gameapi.Biome(0); biome < gameapi.BiomeCount; biome++ {
		note, ok := BiomeFieldNote(biome)
		assertComplete(biome.String(), note, ok)
	}
	for passage := gameapi.PassageID(0); passage < gameapi.PassageCount; passage++ {
		note, ok := PassageFieldNote(passage, gameapi.PassageLocked)
		assertComplete(passage.String(), note, ok)
	}
	for species := gameapi.Species(0); species < gameapi.SpeciesCount; species++ {
		note, ok := SpeciesFieldNote(species)
		assertComplete(species.String(), note, ok)
	}
	for kind := gameapi.EventKind(0); kind < gameapi.EventKindCount; kind++ {
		note, ok := EventKindFieldNote(kind)
		assertComplete(kind.String(), note, ok)
	}
	for kind := AcuteContext(0); kind < AcuteContextCount; kind++ {
		note, ok := AcuteIncidentFieldNote(kind)
		assertComplete(note.Topic, note, ok)
	}
	for role := gameapi.WorkforceRole(0); role < gameapi.AssignmentCount; role++ {
		note, ok := WorkforceRoleFieldNote(role)
		assertComplete(role.String(), note, ok)
	}
	note, ok := AbruptClimateFieldNote(gameapi.EastAfrica, 0.1)
	assertComplete("abrupt climate", note, ok)
	assertComplete("interbreeding", InterbreedingFieldNote(2), true)
}

func TestCampaignOverviewIncludesDenisovanHistoricalContext(t *testing.T) {
	note := CampaignOverviewFieldNote()
	assertFieldNoteHasNoManualLineBreaks(t, note)
	if !strings.Contains(note.Context, "Denisovan") {
		t.Fatalf("campaign overview lacks Denisovan historical context: %#v", note)
	}
}

func TestTraitAndRegionFieldNotesCoverTheirClosedCatalogs(t *testing.T) {
	for trait := gameapi.HeritableTrait(0); trait < gameapi.HeritableTraitCount; trait++ {
		note, ok := TraitFieldNote(trait, 0.5)
		if !ok || note.Topic == "" || note.Introduction == "" || note.Context == "" || note.GameEffect == "" || note.Hint == "" {
			t.Fatalf("trait %d has incomplete Field Notes: %#v", trait, note)
		}
		assertFieldNoteHasNoManualLineBreaks(t, note)
	}
	for region := gameapi.Region(0); region < gameapi.RegionCount; region++ {
		note, ok := RegionEstablishedFieldNote(region)
		if !ok || note.Topic == "" || note.Context == "" || note.GameEffect == "" || note.Hint == "" {
			t.Fatalf("region %d has incomplete Field Notes: %#v", region, note)
		}
		assertFieldNoteHasNoManualLineBreaks(t, note)
	}
	if _, ok := TraitFieldNote(gameapi.HeritableTraitCount, 0.5); ok {
		t.Fatal("out-of-range trait has Field Notes")
	}
	if _, ok := RegionEstablishedFieldNote(gameapi.RegionCount); ok {
		t.Fatal("out-of-range region has Field Notes")
	}
}

// TestTraitFieldNoteCarriesItsTraitForTheHighlight covers the details grid's
// highlight (pkg/hud's traitCellColors): TraitFieldNote is the only
// constructor that identifies a heritable-variant cell, so every other note
// must leave HasTrait false by construction rather than by an explicit
// reset at each call site.
func TestTraitFieldNoteCarriesItsTraitForTheHighlight(t *testing.T) {
	note, ok := TraitFieldNote(gameapi.PigmentationLevel, 0.4)
	if !ok || !note.HasTrait || note.Trait != gameapi.PigmentationLevel {
		t.Fatalf("TraitFieldNote(PigmentationLevel) = %#v, ok %t", note, ok)
	}
	technology, ok := TechnologyFieldNote(gameapi.Firecraft, 7, 1)
	if !ok || technology.HasTrait {
		t.Fatalf("technology note should not carry a trait highlight: %#v", technology)
	}
	if overview := CampaignOverviewFieldNote(); overview.HasTrait {
		t.Fatalf("campaign overview should not carry a trait highlight: %#v", overview)
	}
}

func TestCampanianFieldNoteLabelsTheSimulationEnvelope(t *testing.T) {
	note, ok := MacroEpisodeFieldNote(gameapi.MacroEpisodeSummary{Episode: gameapi.CampanianIgnimbrite, Warned: true})
	assertFieldNoteHasNoManualLineBreaks(t, note)
	if !ok || !strings.Contains(note.Context, "39,850") || !strings.Contains(note.GameEffect, "envelope") || !strings.Contains(note.Introduction, "warning") {
		t.Fatalf("Campanian note = %#v", note)
	}
}

func TestClimateAndTobaNotesSeparateContextFromGameplay(t *testing.T) {
	for epoch := gameapi.ClimateEpoch(0); epoch < gameapi.ClimateEpochCount; epoch++ {
		note, ok := ClimateEpochFieldNote(epoch)
		assertFieldNoteHasNoManualLineBreaks(t, note)
		if !ok || !strings.Contains(note.Context, "Lisiecki") || !strings.Contains(note.GameEffect, "palette") {
			t.Fatalf("climate epoch %d note = %#v", epoch, note)
		}
	}
	if _, ok := ClimateEpochFieldNote(gameapi.ClimateEpochCount); ok {
		t.Fatal("out-of-range climate epoch has Field Notes")
	}
	toba := TobaFieldNote()
	assertFieldNoteHasNoManualLineBreaks(t, toba)
	if !strings.Contains(toba.Context+toba.Hint, "Storey") || !strings.Contains(toba.GameEffect, "no effect") {
		t.Fatalf("Toba note does not separate evidence and gameplay: %#v", toba)
	}
}

func assertFieldNoteHasNoManualLineBreaks(t *testing.T, note render.FieldNote) {
	t.Helper()
	blocks := [...]struct {
		name  string
		value string
	}{
		{name: "topic", value: note.Topic},
		{name: "introduction", value: note.Introduction},
		{name: "context", value: note.Context},
		{name: "game effect", value: note.GameEffect},
		{name: "hint", value: note.Hint},
		{name: "references", value: note.References},
	}
	for _, block := range blocks {
		if strings.ContainsAny(block.value, "\r\n") {
			t.Fatalf("Field Note %s contains a manual line break: %q", block.name, block.value)
		}
	}
}

func TestBandAndEventContextNotesUseAcceptedFrameValues(t *testing.T) {
	frame := &gameapi.Frame{Tiles: []gameapi.Tile{{ID: 0, Region: gameapi.EastAfrica, Biome: gameapi.Savanna, FloraStock: 20, FloraCap: 40, FaunaStock: 30, FaunaCap: 50, WaterStock: 10, WaterCap: 15, EcologicalK: 80, NaturalShelter: 0.25}}}
	band := &gameapi.Band{ID: 7, TileID: 0, Population: 120, Health: 0.9, StoredFood: 6.5}
	note := BandContextFieldNote(frame, band)
	assertFieldNoteHasNoManualLineBreaks(t, note)
	if !strings.Contains(note.Topic, "BAND 7") || !strings.Contains(note.Introduction, "120 people") || !strings.Contains(note.GameEffect, "Food 50/90") {
		t.Fatalf("band context note = %#v", note)
	}
	event := EventFieldNote(gameapi.Event{Turn: 4, Kind: gameapi.EventAcuteIncident, BandID: 7, Summary: "A predator attacked."})
	assertFieldNoteHasNoManualLineBreaks(t, event)
	if event.Topic != gameapi.EventAcuteIncident.String() || event.Introduction != "A predator attacked." || !strings.Contains(event.Context, "turn 4") {
		t.Fatalf("event note = %#v", event)
	}
}

func TestAcuteEventClassificationUsesStablePrecedence(t *testing.T) {
	event := EventFieldNote(gameapi.Event{
		Kind:    gameapi.EventAcuteIncident,
		Summary: "A crossing mishap followed a predation incident.",
	})
	if event.Topic != "ACUTE EVENT · PREDATION" {
		t.Fatalf("ambiguous acute event topic = %q, want deterministic predation precedence", event.Topic)
	}
}
