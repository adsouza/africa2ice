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

func TestBatchedMigrationCandidatesMatchIndividualProjection(t *testing.T) {
	world, err := NewWorld(19)
	if err != nil {
		t.Fatal(err)
	}
	batched := world.MigrationCandidatesByBand()
	if len(batched) != len(world.bands) {
		t.Fatalf("batch contains %d bands, want %d", len(batched), len(world.bands))
	}
	for index, band := range world.bands {
		if batched[index].BandID != band.ID {
			t.Fatalf("batch %d identifies band %d, want %d", index, batched[index].BandID, band.ID)
		}
		if individual := world.MigrationCandidates(band.ID); !reflect.DeepEqual(batched[index].Candidates, individual) {
			t.Fatalf("band %d candidates differ between batch and individual projection", band.ID)
		}
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

// The attraction score must account for who is doing the moving. Every other
// input describes the destination or the band's technology and traits, so before
// the crowding safety factor a tile ranked identically for a band of twenty and
// one of four thousand, and the HUD paints the top-ranked candidate gold as a
// recommendation.
//
// Population is the only input changed here, and it feeds nothing else in the
// score: the destination's own population, movement cost, capacity, food and
// water previews are all independent of the size of the band considering the
// move. Any change in attraction is therefore the safety factor alone, which
// makes the exact ratio assertable.
func TestAttractionFallsForBandsTooLargeForTheDestination(t *testing.T) {
	world, err := NewWorld(17)
	if err != nil {
		t.Fatal(err)
	}
	band := world.bands[0]
	foundingPopulation := float64(band.Population)
	before := map[TileID]MigrationCandidate{}
	for _, candidate := range world.MigrationCandidates(band.ID) {
		before[candidate.TileID] = candidate
	}

	const crowdedPopulation = 4000
	world.bands[0].Population = crowdedPopulation
	after := world.MigrationCandidates(band.ID)
	if len(after) == 0 {
		t.Fatal("no candidates to test")
	}
	penalized := 0
	for _, candidate := range after {
		original, ok := before[candidate.TileID]
		if !ok {
			t.Fatalf("candidate %d appeared only for the larger band", candidate.TileID)
		}
		// Both scores already carry a safety factor, since even a founding band
		// draws a small penalty on marginal ground, so the assertable quantity is
		// the ratio between the two survival fractions. The factor is the share of
		// the arriving population the destination can actually support.
		arrivalK := float64(candidate.EcologicalK * band.Technology.CapacityMultiplier())
		survival := supportedShare(arrivalK, float64(candidate.DestinationPopulation)+float64(crowdedPopulation))
		foundingSurvival := supportedShare(arrivalK, float64(original.DestinationPopulation)+foundingPopulation)
		want := float64(original.Attraction*survival) / foundingSurvival
		if math.Abs(candidate.Attraction-want) > 1e-9 {
			t.Fatalf("tile %d attraction = %v, want %v (original %v, survival %v vs founding %v)",
				candidate.TileID, candidate.Attraction, want, original.Attraction, survival, foundingSurvival)
		}
		if survival < foundingSurvival {
			penalized++
			if candidate.Attraction >= original.Attraction {
				t.Fatalf("tile %d would cost the larger band %v people yet did not rank lower",
					candidate.TileID, candidate.CrowdingDecline)
			}
		}
	}
	if penalized == 0 {
		t.Fatal("no candidate cost the larger band more than the founding one, so the scaling above proves nothing")
	}
}

func supportedShare(capacity, arriving float64) float64 {
	if arriving <= 0 {
		return 1
	}
	if capacity <= 0 {
		return 0
	}
	if share := capacity / arriving; share < 1 {
		return share
	}
	return 1
}
