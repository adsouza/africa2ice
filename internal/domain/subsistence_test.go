package domain

import (
	"math"
	"testing"
)

func TestForagingAndHuntingRates(t *testing.T) {
	band := Band{}
	if got := ForagingRate(band, Savanna); got != 2.5 {
		t.Fatalf("foraging = %v", got)
	}
	profile, _ := FaunaFor(EastAfrica, Savanna, true)
	rates := HuntingRates(band, EastAfrica, profile)
	if math.Abs(rates.Terrestrial-1.75) > 1e-12 || math.Abs(rates.Aquatic-0.25) > 1e-12 || math.Abs(rates.Total-2) > 1e-12 {
		t.Fatalf("hunting rates = %#v", rates)
	}
	mega := MegafaunaRate(band, profile, rates.Terrestrial)
	if mega <= rates.Terrestrial {
		t.Fatalf("megafauna rate %v <= terrestrial %v", mega, rates.Terrestrial)
	}
}

func TestProportionalAllocationExamples(t *testing.T) {
	got := ProportionalAllocate(60, []float64{200, 100})
	if math.Abs(got[0]-40) > 1e-12 || math.Abs(got[1]-20) > 1e-12 {
		t.Fatalf("allocation = %v", got)
	}
	got = ProportionalAllocate(60, []float64{100, 100, 100})
	if got[0] != 20 || got[1] != 20 || got[2] != 20 {
		t.Fatalf("role allocation = %v", got)
	}
	got = ProportionalAllocate(math.MaxFloat64, []float64{math.MaxFloat64, math.MaxFloat64 / 2})
	if math.IsInf(got[0], 0) || math.IsNaN(got[0]) || got[0]+got[1] > math.MaxFloat64 {
		t.Fatalf("overflow allocation = %v", got)
	}
}

func TestWaterDemandCurveAndAdaptation(t *testing.T) {
	if WaterDemandMultiplier(20, 0) != 1 || WaterDemandMultiplier(35, 0) != 1.3 || WaterDemandMultiplier(50, 0) != 1.3 {
		t.Fatal("water anchors changed")
	}
	if got := WaterDemandMultiplier(35, 1); math.Abs(got-1.18) > 1e-12 {
		t.Fatalf("adapted demand = %v", got)
	}
}

func TestFoodHealthStarvationAndGrowth(t *testing.T) {
	consumed, remaining, deficit, fraction := FoodDeficit(50, 40)
	if consumed != 40 || remaining != 0 || deficit != 10 || fraction != 0.2 {
		t.Fatalf("food accounting = %v/%v/%v/%v", consumed, remaining, deficit, fraction)
	}
	if NutritionHealthDelta(0) != 0.05 || math.Abs(NutritionHealthDelta(0.1)-(-0.02)) > 1e-15 {
		t.Fatal("nutrition delta changed")
	}
	for fraction, want := range map[float64]float64{0: 0, 0.1: 0.1, 0.2: 0.4, 0.5: 2.5, 1: 10} {
		if got := StarvationLoss(100, fraction); math.Abs(got-want) > 1e-12 {
			t.Fatalf("starvation(%v) = %v", fraction, got)
		}
	}
	if LogisticGrowth(50, 100, 100, 0) != 0 {
		t.Fatal("co-located population did not stop growth at capacity")
	}
	if LogisticGrowth(100, 50, 1000, 0.5) <= 0 {
		t.Fatal("fed fraction removed all positive growth")
	}
}
