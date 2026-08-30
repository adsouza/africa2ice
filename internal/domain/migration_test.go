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
