package application

import (
	"reflect"
	"testing"

	"github.com/adsouza/africa2ice/internal/domain"
)

func TestStructuredEventTextAndSaveRoundTrip(t *testing.T) {
	tests := []struct {
		kind    domain.EventKind
		details domain.EventDetails
		want    string
	}{
		{domain.EventMigration, domain.EventDetails{}, "Band 2 migrated."},
		{domain.EventSplit, domain.EventDetails{ParentBandID: 1}, "Band 2 split from band 1."},
		{domain.EventTechnology, domain.EventDetails{Technology: domain.HaftedTools}, "Band 2 learned Hafted Tools."},
		{domain.EventInterbreeding, domain.EventDetails{}, "Band 2 interbred with an archaic band."},
		{domain.EventAcuteIncident, domain.EventDetails{AcuteKind: domain.AcuteExposureFall}, "Band 2 suffered an exposure or fall incident."},
		{domain.EventMacroEpisode, domain.EventDetails{}, "Band 2 was affected by the Campanian eruption."},
		{domain.EventAchievement, domain.EventDetails{}, "Sapiens established East Africa."},
		{domain.EventExtinction, domain.EventDetails{Species: domain.ArchaicHominin, MortalityCause: domain.MortalityStarvation}, "Archaic band 2 died out after starvation."},
		{domain.EventExtinction, domain.EventDetails{MortalityCause: domain.MortalityAcute}, "Band 2 died out after an acute incident."},
		{domain.EventExtinction, domain.EventDetails{}, "Band 2 died out."},
	}
	service, err := NewGameService(2)
	if err != nil {
		t.Fatal(err)
	}
	state, err := service.world.ExportState()
	if err != nil {
		t.Fatal(err)
	}
	for _, tc := range tests {
		event := domain.Event{Kind: tc.kind, BandID: 2, Region: domain.EastAfrica, Details: tc.details}
		if got := eventSummary(event); got != tc.want {
			t.Errorf("event %v: %q, want %q", tc.kind, got, tc.want)
		}
		state.Events = append(state.Events, event)
	}
	world, err := domain.RestoreWorld(state)
	if err != nil {
		t.Fatal(err)
	}
	save, err := SaveStateFromWorld(world, 1)
	if err != nil {
		t.Fatal(err)
	}
	for _, event := range save.Events {
		if event.Summary != "" || event.Details == nil {
			t.Fatalf("new event persisted text instead of facts: %+v", event)
		}
	}
	payload, err := EncodeSaveState(save)
	if err != nil {
		t.Fatal(err)
	}
	decoded, err := DecodeSaveState(payload)
	if err != nil {
		t.Fatal(err)
	}
	restored, err := decoded.RestoreWorld()
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(world.Events(), restored.Events()) {
		t.Fatal("event facts lost across save/load")
	}
	frame, err := ProjectSaveState(decoded)
	if err != nil {
		t.Fatal(err)
	}
	for i, event := range frame.Events {
		if event.Summary != tests[i].want {
			t.Fatalf("loaded event %d = %q", i, event.Summary)
		}
	}
}

func TestAcuteIncidentPhraseChoosesItsArticle(t *testing.T) {
	for _, tc := range []struct {
		kind domain.AcuteKind
		want string
	}{
		{domain.AcutePredation, "a predation incident"},
		{domain.AcuteDiseaseOutbreak, "a disease outbreak incident"},
		{domain.AcuteFloodStorm, "a flood or storm incident"},
		{domain.AcuteExposureFall, "an exposure or fall incident"},
		{domain.AcuteCrossingMishap, "a crossing mishap incident"},
		{domain.AcuteKindCount, "an acute incident"},
	} {
		if got := acuteIncidentPhrase(tc.kind); got != tc.want {
			t.Errorf("%v: %q, want %q", tc.kind, got, tc.want)
		}
	}
}

func TestLegacyEventsRetainTextWhenUpgradedAndContinued(t *testing.T) {
	service, err := NewGameService(3)
	if err != nil {
		t.Fatal(err)
	}
	legacy, err := service.ExportSaveState()
	if err != nil {
		t.Fatal(err)
	}
	legacy.SchemaVersion = 1
	legacy.Events = []EventSave{{Kind: uint8(domain.EventAchievement), Region: uint8(domain.EastAfrica), Summary: "Original wording from an old save."}}
	world, err := legacy.RestoreWorld()
	if err != nil {
		t.Fatal(err)
	}
	world.SetEasyMode(true)
	if err := world.AdvanceTurn(); err != nil {
		t.Fatal(err)
	}
	upgraded, err := SaveStateFromWorld(world, 7)
	if err != nil {
		t.Fatal(err)
	}
	if upgraded.SchemaVersion != 2 || upgraded.Events[0].Summary != legacy.Events[0].Summary || upgraded.Events[0].Details != nil {
		t.Fatalf("bad upgrade: %+v", upgraded.Events[0])
	}
	frame, err := ProjectSaveState(upgraded)
	if err != nil {
		t.Fatal(err)
	}
	if frame.Events[0].Summary != legacy.Events[0].Summary {
		t.Fatal("legacy event was reworded")
	}
}

func TestSaveRejectsMalformedEventFacts(t *testing.T) {
	for _, tc := range []struct {
		name  string
		event EventSave
	}{
		{"missing facts", EventSave{Kind: uint8(domain.EventMigration), BandID: 1}},
		{"facts and summary", EventSave{Kind: uint8(domain.EventMigration), BandID: 1, Summary: "ambiguous", Details: &EventDetailsSave{}}},
		{"unknown technology", EventSave{Kind: uint8(domain.EventTechnology), BandID: 1, Details: &EventDetailsSave{Technology: 255}}},
		{"unknown hazard", EventSave{Kind: uint8(domain.EventAcuteIncident), BandID: 1, Details: &EventDetailsSave{AcuteKind: 255}}},
		{"unknown species", EventSave{Kind: uint8(domain.EventExtinction), BandID: 1, Details: &EventDetailsSave{Species: 255}}},
		{"unknown cause", EventSave{Kind: uint8(domain.EventExtinction), BandID: 1, Details: &EventDetailsSave{MortalityCause: 255}}},
		{"missing parent", EventSave{Kind: uint8(domain.EventSplit), BandID: 1, Details: &EventDetailsSave{}}},
		{"irrelevant facts", EventSave{Kind: uint8(domain.EventMigration), BandID: 1, Details: &EventDetailsSave{ParentBandID: 2}}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			service, err := NewGameService(4)
			if err != nil {
				t.Fatal(err)
			}
			save, err := service.ExportSaveState()
			if err != nil {
				t.Fatal(err)
			}
			save.Events = []EventSave{tc.event}
			if _, err := save.RestoreWorld(); err == nil {
				t.Fatal("malformed facts accepted")
			}
		})
	}
}
