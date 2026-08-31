package domain

import (
	"reflect"
	"testing"
)

func TestNewWorldInitializer(t *testing.T) {
	world, err := NewWorld(17)
	if err != nil {
		t.Fatal(err)
	}
	if world.Turn() != 0 || world.Result() != CampaignOngoing {
		t.Fatalf("new world state: turn=%d result=%d", world.Turn(), world.Result())
	}
	bands := world.Bands()
	if len(bands) != len(StartingAnchors) {
		t.Fatalf("bands = %d", len(bands))
	}
	var sapiens, archaic uint64
	for index, band := range bands {
		if band.ID != BandID(index+1) || band.TileID != StartingTileIDs[index] || band.Population != StartingAnchors[index].Population || band.Health != 1 || band.StoredFood != 0 {
			t.Fatalf("band %d: %#v", index, band)
		}
		if band.Species == HomoSapiens {
			sapiens += uint64(band.Population)
		} else {
			archaic += uint64(band.Population)
		}
	}
	if sapiens != 480 || archaic != 390 || world.nextBandID != BandID(len(StartingAnchors)+1) {
		t.Fatalf("scenario totals sapiens=%v archaic=%v next=%d", sapiens, archaic, world.nextBandID)
	}
}

func TestStateRoundTripIsolatedAndContinuesRNG(t *testing.T) {
	world, _ := NewWorld(42)
	state, err := world.ExportState()
	if err != nil {
		t.Fatal(err)
	}
	restored, err := RestoreWorld(state)
	if err != nil {
		t.Fatal(err)
	}
	restoredState, err := restored.ExportState()
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(state, restoredState) {
		t.Fatal("state changed across restore")
	}
	state.Bands[0].Population = 999
	state.RNGState[0] ^= 0xff
	if world.Bands()[0].Population == 999 {
		t.Fatal("export aliases live bands")
	}
	nextOriginal, nextRestored := world.rng.Uint64(), restored.rng.Uint64()
	if nextOriginal != nextRestored {
		t.Fatalf("RNG continuation differs: %d != %d", nextOriginal, nextRestored)
	}
}

func TestInitialExplorationIsSapiensOnly(t *testing.T) {
	world, _ := NewWorld(0)
	for _, band := range world.Bands() {
		if band.Species == ArchaicHominin && world.IsExplored(band.TileID) {
			t.Fatalf("archaic tile %d was revealed", band.TileID)
		}
		if band.Species == HomoSapiens && !world.IsExplored(band.TileID) {
			t.Fatalf("sapiens tile %d is hidden", band.TileID)
		}
	}
}
