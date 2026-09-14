package domain

import (
	"errors"
	"reflect"
	"testing"
)

func TestSplitEligibilityMatchesCommandWithoutMutation(t *testing.T) {
	for _, tc := range []struct {
		name   string
		change func(*World, *BandID, *TileID)
		want   error
	}{
		{name: "allowed"},
		{name: "finished", change: func(w *World, _ *BandID, _ *TileID) { w.result = CampaignVictory }, want: ErrCampaignComplete},
		{name: "missing band", change: func(_ *World, id *BandID, _ *TileID) { *id = 999 }, want: ErrBandNotFound},
		{name: "computer band", change: func(w *World, _ *BandID, _ *TileID) { w.bands[0].Species = ArchaicHominin }, want: ErrComputerControlledBand},
		{name: "spent action", change: func(w *World, _ *BandID, _ *TileID) { w.bands[0].SpatialActionUsed = true }, want: ErrSpatialActionUsed},
		{name: "band limit", change: func(w *World, _ *BandID, _ *TileID) {
			for len(w.bands) < MaxBands {
				w.bands = append(w.bands, Band{ID: BandID(len(w.bands) + 1)})
			}
		}, want: ErrBandLimitReached},
		{name: "id exhaustion", change: func(w *World, _ *BandID, _ *TileID) { w.nextBandID = ^BandID(0) }, want: ErrBandIDExhausted},
		{name: "stress", change: func(w *World, _ *BandID, _ *TileID) { w.bands[0].Population = 1 }, want: ErrSplitStressTooLow},
		{name: "not adjacent", change: func(w *World, _ *BandID, dest *TileID) { *dest = w.bands[0].TileID }, want: ErrSplitDestinationNotAdjacent},
		{name: "unexplored", change: func(w *World, _ *BandID, dest *TileID) { w.exploredTiles[*dest/64] &^= uint64(1) << (*dest % 64) }, want: ErrSplitDestinationUnexplored},
		{name: "uninhabitable", change: func(w *World, _ *BandID, dest *TileID) { h := *w.habitat; h[*dest].BaselineK = 0; w.habitat = &h }, want: ErrSplitDestinationUninhabitable},
		{name: "normal population", change: func(w *World, _ *BandID, _ *TileID) {
			w.bands[1].TileID = w.bands[0].TileID
			w.bands[1].Population = 10000
			w.bands[0].Population = 39
		}, want: ErrSplitPopulationTooLow},
		{name: "easy allowed", change: func(w *World, _ *BandID, _ *TileID) { w.easyMode = true; w.bands[0].Population = 20 }},
		{name: "easy population", change: func(w *World, _ *BandID, _ *TileID) { w.easyMode = true; w.bands[0].Population = 19 }, want: ErrSplitPopulationTooLow},
	} {
		t.Run(tc.name, func(t *testing.T) {
			w, err := NewWorld(2)
			if err != nil {
				t.Fatal(err)
			}
			w.bands[0].Population = 10000
			id, dest := w.bands[0].ID, w.MigrationCandidates(1)[0].TileID
			if tc.change != nil {
				tc.change(w, &id, &dest)
			}
			before := *w
			before.bands = append([]Band(nil), w.bands...)
			before.events = append([]Event(nil), w.events...)
			rngBefore, _ := w.rng.MarshalBinary()
			if got := w.ValidateSplit(id, dest, true); !errors.Is(got, tc.want) {
				t.Fatalf("eligibility = %v, want %v", got, tc.want)
			}
			rngAfter, _ := w.rng.MarshalBinary()
			if !reflect.DeepEqual(before, *w) || !reflect.DeepEqual(rngBefore, rngAfter) {
				t.Fatal("eligibility mutated world")
			}
			if got := w.Split(id, dest, true); !errors.Is(got, tc.want) {
				t.Fatalf("command = %v, want %v", got, tc.want)
			}
		})
	}
}
