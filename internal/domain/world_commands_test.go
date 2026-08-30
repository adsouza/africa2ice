package domain

import (
	"errors"
	"testing"
)

func TestPlayerAuthorityAndAssignment(t *testing.T) {
	world, _ := NewWorld(1)
	allocation := [AssignmentCount]AssignmentBP{2000, 2000, 2000, 2000, 2000}
	if err := world.SetAssignment(1, allocation, true); err != nil {
		t.Fatal(err)
	}
	if world.Bands()[0].Allocation != allocation {
		t.Fatal("assignment not applied")
	}
	if err := world.SetAssignment(5, allocation, true); err != ErrComputerControlledBand {
		t.Fatalf("archaic player assignment = %v", err)
	}
}

func TestMigrationUsesOneSpatialActionAndExploredDestination(t *testing.T) {
	world, _ := NewWorld(1)
	candidates := world.MigrationCandidates(1)
	if len(candidates) == 0 {
		t.Fatal("initial band has no migration candidate")
	}
	if err := world.QueueMigration(1, candidates[0].TileID, true); err != nil {
		t.Fatal(err)
	}
	if err := world.QueueMigration(1, candidates[0].TileID, true); err != ErrSpatialActionUsed {
		t.Fatalf("second action = %v", err)
	}
}

func TestSplitPlacesDescendantAtChosenOrdinaryDestination(t *testing.T) {
	world, _ := NewWorld(2)
	world.bands[0].Population = 10_000
	candidate := world.MigrationCandidates(1)[0]
	if candidate.RequiresPassage {
		t.Fatal("initial recommended candidate unexpectedly requires a passage")
	}
	if err := world.Split(1, candidate.TileID, true); err != nil {
		t.Fatal(err)
	}
	bands := world.Bands()
	if bands[0].TileID == bands[len(bands)-1].TileID || bands[len(bands)-1].TileID != candidate.TileID {
		t.Fatalf("split did not use destination: origin=%d descendant=%d want=%d", bands[0].TileID, bands[len(bands)-1].TileID, candidate.TileID)
	}
}

func TestInterbreedRequiresColocationAndOppositeSpecies(t *testing.T) {
	world, _ := NewWorld(1)
	if err := world.Interbreed(1, 5, true); err != ErrInvalidInterbreedTarget {
		t.Fatalf("distant interbreed = %v", err)
	}
	world.bands[4].TileID = world.bands[0].TileID
	if err := world.Interbreed(1, 5, true); err != nil {
		t.Fatal(err)
	}
	if !world.bands[0].SpatialActionUsed || !world.bands[0].HasInterbreedTarget {
		t.Fatal("interbreed did not spend action")
	}
}

func TestTerminalWorldRejectsEveryPlanningCommand(t *testing.T) {
	world, _ := NewWorld(1)
	world.result = CampaignVictory
	allocation := world.bands[0].Allocation
	tests := []struct {
		name string
		run  func() error
	}{
		{name: "assignment", run: func() error { return world.SetAssignment(1, allocation, true) }},
		{name: "research", run: func() error { return world.Research(1, Firecraft, true) }},
		{name: "migration", run: func() error { return world.QueueMigration(1, StartingTileIDs[0], true) }},
		{name: "split", run: func() error { return world.Split(1, StartingTileIDs[0], true) }},
		{name: "interbreed", run: func() error { return world.Interbreed(1, 5, true) }},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if err := test.run(); !errors.Is(err, ErrCampaignComplete) {
				t.Fatalf("terminal command error = %v, want %v", err, ErrCampaignComplete)
			}
		})
	}
}
