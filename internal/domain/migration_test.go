package domain

import (
	"math"
	"reflect"
	"testing"
)

func TestMigrationCandidatesAreBoundedRankedAndFinite(t *testing.T) {
	world, err := NewWorld(17)
	if err != nil {
		t.Fatal(err)
	}
	candidates := world.MigrationCandidates(1)
	if len(candidates) == 0 || len(candidates) > MaxGridNeighbors+MaxPassageEdgesPerTile {
		t.Fatalf("candidate count = %d", len(candidates))
	}
	for index, candidate := range candidates {
		values := []float64{candidate.Cost, candidate.Attraction, candidate.EcologicalK, candidate.UsableFoodEquivalent, candidate.WaterSurvivalEquivalent, candidate.WarningSuitability}
		for _, value := range values {
			if math.IsNaN(value) || math.IsInf(value, 0) || value < 0 {
				t.Fatalf("invalid candidate value: %#v", candidate)
			}
		}
		if candidate.Cost <= 0 || candidate.WarningSuitability > 1 {
			t.Fatalf("invalid candidate bounds: %#v", candidate)
		}
		if index > 0 {
			previous := candidates[index-1]
			if previous.Attraction < candidate.Attraction || previous.Attraction == candidate.Attraction && previous.TileID > candidate.TileID {
				t.Fatalf("candidates are not stably ranked: %#v then %#v", previous, candidate)
			}
		}
		if !world.IsExplored(candidate.TileID) {
			t.Fatalf("sapiens candidate %d leaks unexplored geography", candidate.TileID)
		}
	}
}

func TestMigrationPreviewDoesNotMutateWorldOrRNG(t *testing.T) {
	world, _ := NewWorld(18)
	before, _ := world.ExportState()
	_ = world.MigrationCandidates(1)
	after, _ := world.ExportState()
	if !reflect.DeepEqual(before, after) {
		t.Fatal("migration preview changed durable state")
	}
}

// A migration candidate must project what the crowding term will cost the band
// on arrival. Seasonal and chronic rates are already previewed and are the two
// smallest contributors to population loss; crowding is by far the largest when
// a band steps onto a tile too small for it, and a player choosing a destination
// has no way to anticipate it from capacity alone.
func TestMigrationCandidatesProjectTheCrowdingDeclineOnArrival(t *testing.T) {
	world, err := NewWorld(17)
	if err != nil {
		t.Fatal(err)
	}
	for _, candidate := range world.MigrationCandidates(1) {
		if math.IsNaN(candidate.CrowdingDecline) || math.IsInf(candidate.CrowdingDecline, 0) || candidate.CrowdingDecline < 0 {
			t.Fatalf("invalid projected decline: %#v", candidate)
		}
	}

	// The projection has to answer "what will taking *this* band there cost",
	// so it must scale with the band. An explicit 100-person probe draws a small
	// warning on marginal neighbours; a band forty times larger must be warned
	// about every reachable tile, and far more sharply. Keeping the probe local
	// to this unit test prevents scenario-balance changes from weakening the
	// projection contract.
	bandID := world.bands[0].ID
	world.bands[0].Population = 100
	probe := world.MigrationCandidates(bandID)
	if len(probe) == 0 {
		t.Fatal("no candidates to test")
	}
	probeTotal, probeWarned := 0.0, 0
	for _, candidate := range probe {
		probeTotal += candidate.CrowdingDecline
		if candidate.CrowdingDecline > 0 {
			probeWarned++
		}
	}
	if probeWarned == len(probe) {
		t.Fatal("every tile warned the 100-person probe, so the projection is not discriminating")
	}

	world.bands[0].Population = 4000
	crowded := world.MigrationCandidates(bandID)
	projected := 0
	for _, candidate := range crowded {
		if candidate.CrowdingDecline > 0 {
			projected++
		}
		if limit := float64(4000 * MaxCrowdingDeclineFraction); candidate.CrowdingDecline > limit {
			t.Fatalf("projected decline %v exceeds the per-turn bound %v", candidate.CrowdingDecline, limit)
		}
	}
	if projected != len(crowded) {
		t.Fatalf("only %d of %d candidates warned a 4000-person band, want every one", projected, len(crowded))
	}
	crowdedTotal := 0.0
	for _, candidate := range crowded {
		crowdedTotal += candidate.CrowdingDecline
	}
	if crowdedTotal <= probeTotal {
		t.Fatalf("a 4000-person band projected %v against the 100-person probe's %v; the warning must scale with the band", crowdedTotal, probeTotal)
	}
}
