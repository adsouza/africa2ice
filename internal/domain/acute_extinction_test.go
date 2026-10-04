package domain

import "testing"

// severityMitigated reports whether band's technology softens kind, the only
// mitigation ResolveAcute applies after interpolating the loss bounds.
func severityMitigated(band Band, kind AcuteKind) bool {
	switch kind {
	case AcutePredation:
		return band.Technology.Has(HaftedTools)
	case AcuteDiseaseOutbreak:
		return band.Technology.Has(MedicinalKnowledge)
	case AcuteCrossingMishap:
		return band.Technology.Has(CoastalNavigation)
	}
	return false
}

// Step 5's dedicated acute fixture: with every kind's loss bounds at the valid
// boundary MinAcuteLoss = MaxAcuteLoss = 1, an incident against a band with no
// severity mitigation must leave no survivor, and the turn's extinction filter
// must then report and remove that band like any other, freeing its slot.
// The real bounds top out at 0.15, so without pinning them an acute death is a
// rounding accident on a tiny band rather than something a test can demand.
func TestAcuteIncidentAtFullSeverityIsExactExtinction(t *testing.T) {
	savedMin, savedMax := minimumAcuteLoss, maximumAcuteLoss
	defer func() { minimumAcuteLoss, maximumAcuteLoss = savedMin, savedMax }()
	for kind := range minimumAcuteLoss {
		minimumAcuteLoss[kind], maximumAcuteLoss[kind] = 1, 1
	}

	exact := 0
	for seed := uint64(1); seed <= 16; seed++ {
		world, err := NewWorld(seed)
		if err != nil {
			t.Fatal(err)
		}
		for turn := 0; turn < 8; turn++ {
			before := map[BandID]Band{}
			for _, band := range world.Bands() {
				before[band.ID] = band
			}
			seen := len(world.Events())
			if err := world.AdvanceTurn(); err != nil {
				t.Fatal(err)
			}
			events := world.Events()
			if seen > len(events) {
				seen = 0 // the feed evicted; this turn's events are all that remain
			}
			extinct := map[BandID]Event{}
			var incidents []Event
			for _, event := range events[seen:] {
				switch event.Kind {
				case EventExtinction:
					extinct[event.BandID] = event
				case EventAcuteIncident:
					incidents = append(incidents, event)
				}
			}
			after := map[BandID]bool{}
			for _, band := range world.Bands() {
				if band.Population == 0 {
					t.Fatalf("seed %d turn %d: band %d survived the turn with no population", seed, turn, band.ID)
				}
				after[band.ID] = true
			}
			for _, incident := range incidents {
				band, existed := before[incident.BandID]
				if !existed || severityMitigated(band, incident.Details.AcuteKind) {
					continue
				}
				if _, gone := extinct[incident.BandID]; !gone {
					t.Fatalf("seed %d turn %d: unmitigated full-severity %v left band %d alive", seed, turn, incident.Details.AcuteKind, incident.BandID)
				}
				if after[incident.BandID] {
					t.Fatalf("seed %d turn %d: band %d reported extinct but still holds a slot", seed, turn, incident.BandID)
				}
				exact++
			}
		}
	}
	if exact == 0 {
		t.Fatal("no seed produced an unmitigated acute incident, so the fixture exercised nothing")
	}
	t.Logf("%d unmitigated full-severity incidents, each an exact extinction", exact)
}
