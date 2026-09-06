package domain

import (
	"fmt"
	"testing"
)

// TestAcuteIncidentPhraseChoosesItsArticle covers the event feed's grammar:
// the summary hardcoded "a", so the one vowel-initial hazard read "Band 9
// suffered a exposure or fall incident."
func TestAcuteIncidentPhraseChoosesItsArticle(t *testing.T) {
	tests := []struct {
		kind AcuteKind
		want string
	}{
		{kind: AcutePredation, want: "a predation incident"},
		{kind: AcuteDiseaseOutbreak, want: "a disease outbreak incident"},
		{kind: AcuteFloodStorm, want: "a flood or storm incident"},
		{kind: AcuteExposureFall, want: "an exposure or fall incident"},
		{kind: AcuteCrossingMishap, want: "a crossing mishap incident"},
		{kind: AcuteKindCount, want: "an acute incident"},
	}
	for _, test := range tests {
		if got := acuteIncidentPhrase(test.kind); got != test.want {
			t.Errorf("acuteIncidentPhrase(%d) = %q, want %q", test.kind, got, test.want)
		}
	}
}

// TestMortalityCauseNamesTheLargestComponent pins how an extinction line
// explains itself. Ties resolve in field order so the same report always
// produces the same sentence.
func TestMortalityCauseNamesTheLargestComponent(t *testing.T) {
	tests := []struct {
		name   string
		report MortalityReport
		want   string
	}{
		{name: "nothing recorded", report: MortalityReport{}, want: ""},
		{name: "starvation", report: MortalityReport{Starvation: 5, Seasonal: 1}, want: "starvation"},
		{name: "seasonal", report: MortalityReport{Seasonal: 4, Chronic: 3}, want: "seasonal mortality"},
		{name: "chronic", report: MortalityReport{Chronic: 9, Acute: 8}, want: "chronic mortality"},
		{name: "macro", report: MortalityReport{Macro: 7, Starvation: 2}, want: "a macro episode"},
		{name: "acute", report: MortalityReport{Acute: 6, Seasonal: 5}, want: "an acute incident"},
		{name: "tie takes the earlier field", report: MortalityReport{Starvation: 3, Acute: 3}, want: "starvation"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if got := mortalityCause(test.report); got != test.want {
				t.Errorf("mortalityCause(%+v) = %q, want %q", test.report, got, test.want)
			}
		})
	}
}

// TestExtinctionSummaryNamesTheBandAndItsCause covers the line the Recent
// Events feed shows when a band is filtered out of the world. Archaic bands
// carry a prefix so background attrition is not mistaken for the player
// losing bands of their own.
func TestExtinctionSummaryNamesTheBandAndItsCause(t *testing.T) {
	tests := []struct {
		name string
		band Band
		want string
	}{
		{
			name: "sapiens with a cause",
			band: Band{ID: 31, Species: HomoSapiens, LastMortality: MortalityReport{Acute: 4}},
			want: "Band 31 died out after an acute incident.",
		},
		{
			name: "archaic with a cause",
			band: Band{ID: 12, Species: ArchaicHominin, LastMortality: MortalityReport{Starvation: 9}},
			want: "Archaic band 12 died out after starvation.",
		},
		{
			name: "nothing recorded drops the clause",
			band: Band{ID: 4, Species: HomoSapiens},
			want: "Band 4 died out.",
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if got := extinctionSummary(test.band); got != test.want {
				t.Errorf("extinctionSummary() = %q, want %q", got, test.want)
			}
		})
	}
}

// TestAdvanceTurnRecordsBandsThatDieOut covers the gap the feed had: the turn
// filtered zero-population bands out of the world silently, so a band the
// player was watching simply stopped existing with no line explaining it.
func TestAdvanceTurnRecordsBandsThatDieOut(t *testing.T) {
	world, err := NewWorld(5)
	if err != nil {
		t.Fatal(err)
	}
	doomed := world.bands[0]
	world.bands[0].Population = 0
	world.bands[0].LastMortality = MortalityReport{Starvation: 12}
	if err := world.AdvanceTurn(); err != nil {
		t.Fatal(err)
	}
	// Other bands may split during the same turn, so the doomed band's absence
	// is the fact to assert rather than a total count.
	for _, band := range world.Bands() {
		if band.ID == doomed.ID {
			t.Fatalf("band %d survived with population %d", band.ID, band.Population)
		}
	}
	var found []Event
	for _, event := range world.Events() {
		if event.Kind == EventExtinction {
			found = append(found, event)
		}
	}
	if len(found) != 1 {
		t.Fatalf("extinction events = %d, want exactly 1: %+v", len(found), found)
	}
	event := found[0]
	if event.BandID != doomed.ID || event.TileID != doomed.TileID || event.Turn != world.Turn() {
		t.Fatalf("extinction event = %+v, want band %d on tile %d at turn %d", event, doomed.ID, doomed.TileID, world.Turn())
	}
	doomed.Population = 0
	doomed.LastMortality = MortalityReport{Starvation: 12}
	if want := extinctionSummary(doomed); event.Summary != want {
		t.Fatalf("summary = %q, want %q", event.Summary, want)
	}
}

// TestArchaicBackgroundNoiseIsDroppedFromTheFeed covers the event log's worst
// problem: the computer moves, researches, splits and suffers hazards every
// turn with dozens of bands, which buried the player's own campaign. Measured
// at turn 200 of a reference run, all 128 retained events were archaic and 125
// of those were migrations — not one sapiens event survived the window.
// Milestones and anything a sapiens band is party to still surface.
func TestArchaicBackgroundNoiseIsDroppedFromTheFeed(t *testing.T) {
	tests := []struct {
		kind     EventKind
		archaic  bool
		wantKept bool
	}{
		{kind: EventMigration, archaic: true, wantKept: false},
		{kind: EventTechnology, archaic: true, wantKept: false},
		{kind: EventSplit, archaic: true, wantKept: false},
		{kind: EventAcuteIncident, archaic: true, wantKept: false},
		{kind: EventExtinction, archaic: true, wantKept: true},
		{kind: EventInterbreeding, archaic: true, wantKept: true},
		{kind: EventMacroEpisode, archaic: true, wantKept: true},
		{kind: EventMigration, archaic: false, wantKept: true},
		{kind: EventTechnology, archaic: false, wantKept: true},
		{kind: EventSplit, archaic: false, wantKept: true},
		{kind: EventAcuteIncident, archaic: false, wantKept: true},
		{kind: EventExtinction, archaic: false, wantKept: true},
	}
	for _, test := range tests {
		species, name := HomoSapiens, "sapiens"
		if test.archaic {
			species, name = ArchaicHominin, "archaic"
		}
		t.Run(fmt.Sprintf("%s/kind%d", name, test.kind), func(t *testing.T) {
			world, err := NewWorld(5)
			if err != nil {
				t.Fatal(err)
			}
			before := len(world.Events())
			world.appendBandEvent(Band{ID: 1, Species: species}, Event{Turn: 1, Kind: test.kind, BandID: 1})
			if kept := len(world.Events()) > before; kept != test.wantKept {
				t.Fatalf("%s kind %d kept = %t, want %t", name, test.kind, kept, test.wantKept)
			}
		})
	}
}

// TestAdvanceTurnKeepsArchaicRoutineActivityOutOfTheFeed is the end-to-end
// guard for the policy: it is not enough that appendBandEvent filters, every
// band-scoped emission has to actually route through it.
func TestAdvanceTurnKeepsArchaicRoutineActivityOutOfTheFeed(t *testing.T) {
	world, err := NewWorld(5)
	if err != nil {
		t.Fatal(err)
	}
	species := map[BandID]Species{}
	record := func() {
		for _, band := range world.Bands() {
			species[band.ID] = band.Species
		}
	}
	record()
	for turn := 0; turn < 50; turn++ {
		if err := world.AdvanceTurn(); err != nil {
			t.Fatalf("turn %d: %v", turn+1, err)
		}
		record()
	}
	noise, sapiens := 0, 0
	for _, event := range world.Events() {
		switch {
		case event.BandID == 0:
		case species[event.BandID] != ArchaicHominin:
			sapiens++
		case archaicBackgroundNoise(event.Kind):
			noise++
			t.Errorf("archaic band %d reached the feed with kind %d: %q", event.BandID, event.Kind, event.Summary)
		}
	}
	if noise > 0 {
		t.Fatalf("%d archaic background events reached the feed", noise)
	}
	// Without this the test would also pass if the filter swallowed the whole
	// feed, which is the failure it is least likely to notice.
	if sapiens == 0 {
		t.Fatal("no sapiens events survived; the filter is dropping too much")
	}
}
