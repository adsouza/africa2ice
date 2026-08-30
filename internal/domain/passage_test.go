package domain

import (
	"bytes"
	"testing"
)

func TestPassageCatalogAndBeringiaTrajectory(t *testing.T) {
	grid, err := (WorldGenerator{}).Generate()
	if err != nil {
		t.Fatal(err)
	}
	if err := ValidatePassages(grid); err != nil {
		t.Fatal(err)
	}
	want := map[int]bool{177: false, 178: true, 194: true, 195: false, 230: false, 231: true, 400: true}
	for turn, expected := range want {
		climate, err := ClimateAt(1, turn)
		if err != nil {
			t.Fatal(err)
		}
		if got := BeringiaOpen(climate.LongTermTempOffset); got != expected {
			t.Fatalf("BeringiaOpen(turn %d) = %t, want %t", turn, got, expected)
		}
	}
}

func TestNamedPassageRequiresTechnologyAndQueuesWithoutRNG(t *testing.T) {
	world, err := NewWorld(42)
	if err != nil {
		t.Fatal(err)
	}
	band := &world.bands[0]
	band.TileID = passageCatalog[NorthWallacea].From
	world.markExplored(passageCatalog[NorthWallacea].To)
	before, _ := world.rng.MarshalBinary()
	if err := world.QueueMigration(band.ID, passageCatalog[NorthWallacea].To, true); err != ErrInvalidMigration {
		t.Fatalf("locked queue error = %v", err)
	}
	band.Technology.Acquired |= 1 << CoastalNavigation
	band.Technology.Progress[CoastalNavigation] = ResearchCost[CoastalNavigation]
	if err := world.QueueMigration(band.ID, passageCatalog[NorthWallacea].To, true); err != nil {
		t.Fatal(err)
	}
	after, _ := world.rng.MarshalBinary()
	if !band.QueueUsesPassage || band.QueuedPassage != NorthWallacea || !bytes.Equal(before, after) {
		t.Fatalf("queued passage = %#v", band)
	}
}
