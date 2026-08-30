package domain

import (
	"reflect"
	"testing"
)

func TestSplitMixCorpus(t *testing.T) {
	tests := []struct{ seed, first, second uint64 }{
		{0, 0xe220a8397b1dcdaf, 0x6e789e6aa1b965f4},
		{1, 0x910a2dec89025cc1, 0xbeeb8da1658eec67},
		{2, 0x975835de1c9756ce, 0xbfc846100bfc1e42},
		{3, 0x1d0b14e4db018fed, 0xb3466f8a7b81a989},
		{0x9e3779b97f4a7c15, 0x6e789e6aa1b965f4, 0x06c45d188009454f},
		{0xd1b54a32d192ed03, 0x2d0f28c7e7e786b2, 0x75856f745165f252},
		{0x94d049bb133111eb, 0xbb1fd59c964c1554, 0xee4b135308a7ae87},
		{0xffffffffffffffff, 0xe4d971771b652c20, 0xe99ff867dbf682c9},
	}
	for _, test := range tests {
		state, first := SplitMix64(test.seed)
		_, second := SplitMix64(state)
		if first != test.first || second != test.second {
			t.Fatalf("seed %#x -> %#x/%#x", test.seed, first, second)
		}
	}
}

func TestWorldRNGRoundTripContinues(t *testing.T) {
	rng := NewWorldRNG(42)
	_ = rng.Float64()
	state, err := rng.MarshalBinary()
	if err != nil {
		t.Fatal(err)
	}
	restored, err := RestoreWorldRNG(state)
	if err != nil {
		t.Fatal(err)
	}
	got, want := make([]uint64, 8), make([]uint64, 8)
	for i := range got {
		got[i], want[i] = restored.Uint64(), rng.Uint64()
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("restored sequence differs: %v != %v", got, want)
	}
}

func TestCounterNoiseBoundsAndOrderIndependence(t *testing.T) {
	for turn := 0; turn <= MaxCampaignTurn; turn++ {
		value := UnitNoiseV1(12, turn)
		if value < -1 || value > 1 {
			t.Fatalf("noise %d = %v", turn, value)
		}
	}
	first := UnitResourceAbundanceV1(2, ResourceTile, 9, WaterStock)
	_ = UnitResourceAbundanceV1(2, ResourceRegion, 4, FloraStock)
	if second := UnitResourceAbundanceV1(2, ResourceTile, 9, WaterStock); first != second {
		t.Fatal("counter hash depends on call order")
	}
}
