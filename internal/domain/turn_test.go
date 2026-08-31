package domain

import (
	"bytes"
	"math"
	"reflect"
	"testing"
)

func TestAdvanceTurnPublishesOneCompleteTransition(t *testing.T) {
	world, _ := NewWorld(5)
	if err := world.AdvanceTurn(); err != nil {
		t.Fatal(err)
	}
	if world.Turn() != 1 || world.Result() != CampaignOngoing {
		t.Fatalf("turn/result = %d/%d", world.Turn(), world.Result())
	}
	for _, band := range world.Bands() {
		if band.LastFoodReport.Turn != 1 || band.LastOutcomeReport.Turn != 1 || band.SpatialActionUsed || band.HasQueuedMigration || band.HasInterbreedTarget {
			t.Fatalf("band not finalized: %#v", band)
		}
		if band.LastOutcomeReport.StartingPopulation == 0 || band.LastOutcomeReport.EndingPopulation != band.Population || band.LastOutcomeReport.StartingHealth != 1 || band.LastOutcomeReport.EndingHealth != band.Health {
			t.Fatalf("band outcome report does not bracket the turn: %#v", band.LastOutcomeReport)
		}
		if float64(band.StoredFood) > FoodStorageCapacity(band.Population) {
			t.Fatal("food cap not enforced")
		}
	}
}

func TestAdvanceTurnRejectsCandidateWithoutCommittingAggregateOrRNG(t *testing.T) {
	world, err := NewWorld(6)
	if err != nil {
		t.Fatal(err)
	}
	world.bands[0].Health = Health(math.Float64frombits(0x7ff8000000000000))
	beforeRNG, err := world.rng.MarshalBinary()
	if err != nil {
		t.Fatal(err)
	}
	beforeTiles := world.tiles
	beforeExplored := world.exploredTiles
	beforePopulation := world.bands[0].Population

	if err := world.AdvanceTurn(); err == nil {
		t.Fatal("invalid candidate transition was accepted")
	}
	afterRNG, err := world.rng.MarshalBinary()
	if err != nil {
		t.Fatal(err)
	}
	if world.turn != 0 || world.tiles != beforeTiles || world.exploredTiles != beforeExplored || world.bands[0].Population != beforePopulation || !math.IsNaN(float64(world.bands[0].Health)) {
		t.Fatal("failed transition changed the live aggregate")
	}
	if !bytes.Equal(beforeRNG, afterRNG) {
		t.Fatal("failed transition advanced the live RNG")
	}
}

func TestQueuedMigrationResolvesOnce(t *testing.T) {
	world, _ := NewWorld(8)
	candidate := world.MigrationCandidates(1)[0]
	if err := world.QueueMigration(1, candidate.TileID, true); err != nil {
		t.Fatal(err)
	}
	if err := world.AdvanceTurn(); err != nil {
		t.Fatal(err)
	}
	if world.Bands()[0].TileID != candidate.TileID || world.Bands()[0].SpatialActionUsed {
		t.Fatalf("migration did not finalize: %#v", world.Bands()[0])
	}
}

func TestAdvanceTurnContinuesIdenticallyAfterMemento(t *testing.T) {
	original, _ := NewWorld(12)
	state, _ := original.ExportState()
	restored, _ := RestoreWorld(state)
	if err := original.AdvanceTurn(); err != nil {
		t.Fatal(err)
	}
	if err := restored.AdvanceTurn(); err != nil {
		t.Fatal(err)
	}
	a, _ := original.ExportState()
	b, _ := restored.ExportState()
	if !reflect.DeepEqual(a, b) {
		t.Fatal("turn differs after memento round trip")
	}
}

func TestUnattendedCampaignAdvancesDeterministicallyToTerminalState(t *testing.T) {
	left, _ := NewWorld(0x9e3779b97f4a7c15)
	right, _ := NewWorld(0x9e3779b97f4a7c15)
	for left.Result() == CampaignOngoing {
		if err := left.AdvanceTurn(); err != nil {
			t.Fatalf("left turn %d: %v", left.Turn(), err)
		}
		if err := right.AdvanceTurn(); err != nil {
			t.Fatalf("right turn %d: %v", right.Turn(), err)
		}
	}
	if left.Turn() <= 0 || left.Turn() > MaxCampaignTurn || left.Result() == CampaignOngoing {
		t.Fatalf("invalid terminal state: turn=%d result=%d", left.Turn(), left.Result())
	}
	t.Logf("unattended campaign reached result %d on turn %d", left.Result(), left.Turn())
	a, _ := left.ExportState()
	b, _ := right.ExportState()
	if !reflect.DeepEqual(a, b) {
		t.Fatal("same-seed unattended campaigns diverged")
	}
}
