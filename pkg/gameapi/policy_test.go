package gameapi

import "testing"

func TestSharedRoutePolicyConfiguration(t *testing.T) {
	if MinEstablishedBand != 20 || MinSplitSourcePopulation != 40 ||
		SplitStressThreshold != 0.67 || ReferenceRouteDeparturePopulation != 50 {
		t.Fatalf("shared policy values = %v/%v/%v/%v",
			MinEstablishedBand, MinSplitSourcePopulation, SplitStressThreshold, ReferenceRouteDeparturePopulation)
	}
	tests := []struct {
		capacity float64
		want     int
	}{
		{capacity: 100, want: 1},
		{capacity: 99.9, want: 2},
		{capacity: 75, want: 2},
		{capacity: 50, want: 4},
		{capacity: 25, want: 8},
		{capacity: 10, want: 16},
		{capacity: 9.9, want: 32},
	}
	for _, test := range tests {
		if got := ReferenceRouteStepCost(test.capacity); got != test.want {
			t.Errorf("ReferenceRouteStepCost(%v) = %d, want %d", test.capacity, got, test.want)
		}
	}
}
