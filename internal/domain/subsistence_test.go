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

// The crowding term is a brake, not the model's largest killer. §7 describes a
// tile at capacity as stopping growth for everyone standing on it; without a
// bound the same expression removes most of a band in a single turn, and it does
// so through the growth term, so no cause appears in the persisted
// MortalityReport and nothing in the UI can explain it.
func TestCrowdingDeclineIsBounded(t *testing.T) {
	// The measured case: a band of 112 on a tile whose effective capacity has
	// collapsed to roughly 1/30th of the people standing on it.
	//
	// The bound is computed here exactly as the production code computes it, from
	// a float64 population rather than an untyped constant. Written as
	// -112 * MaxCrowdingDeclineFraction both operands fold to an exact rational
	// and the result differs from the runtime product by one ulp, which is enough
	// to fail an otherwise correct implementation.
	population := 112.0
	got := LogisticGrowth(population, 112, 3.67, 0)
	if limit := -float64(population * MaxCrowdingDeclineFraction); got < limit {
		t.Errorf("crowding removed %v of a 112-person band in one turn, the bound is %v", -got, -limit)
	}
	if got >= 0 {
		t.Errorf("a band 30x over capacity should still decline, got %v", got)
	}

	// A tile that cannot support anyone declines by the same bound rather than
	// annihilating the band, so a climate shift under a settled band is
	// survivable long enough to be seen and answered.
	population = 50.0
	got = LogisticGrowth(population, 50, 0, 0)
	if limit := -float64(population * MaxCrowdingDeclineFraction); got < limit {
		t.Errorf("zero capacity removed %v of a 50-person band, the bound is %v", -got, -limit)
	}
	if got >= 0 {
		t.Errorf("zero capacity should still decline, got %v", got)
	}
}

// The bound must not become a floor that all crowding decline snaps to: a mild
// overshoot has to stay proportional, or crowding stops carrying information.
func TestMildCrowdingDeclineIsNotClamped(t *testing.T) {
	population := 100.0
	got := LogisticGrowth(population, 110, 100, 0)
	want := float64(float64(PopulationGrowthRate*population) * (1 - 110.0/100.0))
	if math.Abs(got-want) > 1e-9 {
		t.Errorf("mild overshoot = %v, want the unbounded %v", got, want)
	}
	if bound := -float64(population * MaxCrowdingDeclineFraction); got <= bound {
		t.Fatalf("test is vacuous: %v already exceeds the bound %v", got, bound)
	}
}
