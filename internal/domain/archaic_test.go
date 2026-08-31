package domain

import "testing"

func TestArchaicPlanningUsesClosedAssignmentResearchAndRankedSplit(t *testing.T) {
	world, _ := NewWorld(19)
	world.bands[4].Population = 10_000
	candidates := world.MigrationCandidates(5)
	if len(candidates) == 0 || candidates[0].RequiresPassage {
		t.Fatal("fixture needs an ordinary migration candidate")
	}
	beforeRNG, _ := world.rng.MarshalBinary()
	planning, err := world.planArchaic()
	if err != nil {
		t.Fatal(err)
	}
	afterRNG, _ := world.rng.MarshalBinary()
	if string(beforeRNG) != string(afterRNG) {
		t.Fatal("archaic planning consumed randomness")
	}
	// The child takes the next free ID, which follows the anchor catalog. Deriving
	// it keeps this fixture correct when anchors are added: hard-coded, it silently
	// began matching a starting band instead of the split descendant.
	const sourceID = BandID(5)
	childID := BandID(len(StartingAnchors) + 1)
	var source, child *Band
	for index := range planning.bands {
		switch planning.bands[index].ID {
		case sourceID:
			source = &planning.bands[index]
		case childID:
			child = &planning.bands[index]
		}
	}
	if source == nil || child == nil {
		t.Fatalf("ranked pressure did not split first archaic band; next ID=%d", planning.nextBandID)
	}
	if child.TileID != candidates[0].TileID || source.TileID == child.TileID || !source.SpatialActionUsed || !child.SpatialActionUsed {
		t.Fatalf("incorrect ranked split: source=%#v child=%#v", source, child)
	}
	if !source.Technology.HasTarget || source.Technology.Target != PlantKnowledge || child.Technology != source.Technology || child.Allocation != source.Allocation {
		t.Fatalf("planning state was not applied/copied: source=%#v child=%#v", source, child)
	}
}
