package domain

import (
	"math"
	"testing"
)

// Property tests for invariants DESIGN.md states outright. Each is a Go fuzz
// target: `go test` replays the seeds below, and `go test -fuzz` explores
// arbitrary inputs, which these map into the function's valid domain.

// unit maps any float to [0, 1) monotonically in |value|; NaN and infinities
// become 0. (math.Mod is outside the domain's closed math allowlist.)
func unit(value float64) float64 {
	if math.IsNaN(value) || math.IsInf(value, 0) {
		return 0
	}
	magnitude := math.Abs(value)
	return magnitude / (1 + magnitude)
}

// scaled maps any float to [0, limit].
func scaled(value, limit float64) float64 { return float64(unit(value) * limit) }

// DESIGN.md §5: "allocations are non-negative, their sum never exceeds the
// available stock", and a stock that covers every demand meets each in full.
func FuzzProportionalAllocateNeverOverspends(f *testing.F) {
	f.Add(0.6, 0.2, 0.1, 0.0, 0.0)
	f.Add(0.0, 0.5, 0.5, 0.5, 0.5)
	f.Add(0.999, 0.001, 0.000001, 0.9, 0.3)
	f.Add(0.5, 1e-300, 0.9, 0.0, 0.7)
	f.Fuzz(func(t *testing.T, available, first, second, third, fourth float64) {
		stock := scaled(available, 1e6)
		demands := []float64{scaled(first, 1e6), scaled(second, 1e6), scaled(third, 1e6), scaled(fourth, 1e6)}
		allocations := ProportionalAllocate(stock, demands)
		total, demanded := 0.0, 0.0
		for index, allocation := range allocations {
			if allocation < 0 {
				t.Fatalf("allocation %d = %v is negative (stock %v, demands %v)", index, allocation, stock, demands)
			}
			if allocation > float64(demands[index]*(1+1e-12)) {
				t.Fatalf("allocation %d = %v exceeds its demand %v", index, allocation, demands[index])
			}
			total += allocation
			demanded += demands[index]
		}
		if total > stock {
			t.Fatalf("allocations sum to %v, more than the stock %v (demands %v)", total, stock, demands)
		}
		if demanded <= float64(stock*(1-1e-9)) {
			for index, allocation := range allocations {
				if allocation != demands[index] {
					t.Fatalf("stock %v covers total demand %v, yet allocation %d = %v of %v", stock, demanded, index, allocation, demands[index])
				}
			}
		}
	})
}

// DESIGN.md: "Assert 0 <= Growth <= BaseGrowth for positive bases", and the
// crowding decline never removes more than MaxCrowdingDeclineFraction.
func FuzzLogisticGrowthStaysWithinItsBounds(f *testing.F) {
	f.Add(0.1, 0.0, 0.2, 0.0)
	f.Add(0.1, 0.5, 0.05, 0.3)
	f.Add(0.9, 0.9, 0.0, 1.0)
	f.Add(0.5, 0.1, 1.0, -2.0)
	f.Fuzz(func(t *testing.T, ownShare, othersShare, capacityShare, deficit float64) {
		population := scaled(ownShare, 1e5)
		total := population + scaled(othersShare, 1e5)
		capacity := scaled(capacityShare, 2e5)
		growth := LogisticGrowth(population, total, capacity, deficit)
		if population == 0 {
			if growth != 0 {
				t.Fatalf("an empty band grew by %v", growth)
			}
			return
		}
		if floor := -float64(population * MaxCrowdingDeclineFraction); growth < floor {
			t.Fatalf("growth %v is below the crowding bound %v (P=%v total=%v K=%v)", growth, floor, population, total, capacity)
		}
		if capacity <= 0 {
			return
		}
		base := float64(float64(PopulationGrowthRate*population) * (1 - total/capacity))
		if base > 0 && (growth < 0 || growth > base) {
			t.Fatalf("growth %v outside [0, base %v] (P=%v total=%v K=%v deficit=%v)", growth, base, population, total, capacity, deficit)
		}
	})
}

// DESIGN.md: degradation stays in [0, MaxDegradation], rises under pressure
// above capacity, falls below it, and more people never degrade a tile less.
func FuzzNextDegradationIsBoundedAndMonotonic(f *testing.F) {
	f.Add(0.0, 0.5, 0.1, 0.5)
	f.Add(0.75, 0.9, 0.95, 0.001)
	f.Add(0.3, 0.25, 0.25, 0.25)
	f.Add(0.5, 0.0, 1.0, 1e-9)
	// Near the cap under moderate overcrowding: the damage step lands just
	// past MaxDegradation, where only the clamp keeps it in bounds.
	f.Add(74.0, 0.1, 0.401, 1.0)
	f.Fuzz(func(t *testing.T, currentShare, fewerShare, moreShare, capacityShare float64) {
		current := scaled(currentShare, MaxDegradation)
		capacity := scaled(capacityShare, 1e4)
		fewer, more := scaled(fewerShare, 4e4), scaled(moreShare, 4e4)
		if fewer > more {
			fewer, more = more, fewer
		}
		next := NextDegradation(current, more, capacity)
		if next < 0 || next > MaxDegradation {
			t.Fatalf("degradation %v left [0, %v]", next, MaxDegradation)
		}
		if capacity <= 0 {
			return
		}
		switch pressure := more / capacity; {
		case pressure > 1 && next < current:
			t.Fatalf("pressure %v lowered degradation from %v to %v", pressure, current, next)
		case pressure < 1 && next > current:
			t.Fatalf("pressure %v raised degradation from %v to %v", pressure, current, next)
		}
		if lighter := NextDegradation(current, fewer, capacity); lighter > next {
			t.Fatalf("%v people degrade the tile to %v but %v people only to %v", fewer, lighter, more, next)
		}
	})
}

// EcologicalK = BaselineK · (1 − Degradation) · MacroHabitatFactor: never
// above BaselineK, never negative, and never higher on a more degraded tile.
func FuzzEcologicalKNeverExceedsBaselineAndFallsWithDegradation(f *testing.F) {
	f.Add(0.5, 0.0, 0.75, 1.0)
	f.Add(0.1, 0.3, 0.3, 0.5)
	f.Add(1.0, 0.75, 0.0, 0.0)
	f.Fuzz(func(t *testing.T, baselineShare, firstDegradation, secondDegradation, habitatShare float64) {
		baseline := scaled(baselineShare, 1e4)
		less, more := scaled(firstDegradation, MaxDegradation), scaled(secondDegradation, MaxDegradation)
		if less > more {
			less, more = more, less
		}
		macro := MacroImpact{HabitatFactor: unit(habitatShare)}
		healthy, degraded := EcologicalK(baseline, less, macro), EcologicalK(baseline, more, macro)
		if healthy < 0 || degraded < 0 || healthy > baseline {
			t.Fatalf("EcologicalK %v / %v outside [0, BaselineK %v]", healthy, degraded, baseline)
		}
		if degraded > healthy {
			t.Fatalf("degradation %v gives K %v, above %v at degradation %v", more, degraded, healthy, less)
		}
	})
}

// A split conserves people exactly and food to rounding, halves the band
// within one person, keeps each half within its own storage capacity, and
// leaves neither half holding an order the parent had queued.
func FuzzSplitConservesPopulationAndFood(f *testing.F) {
	f.Add(uint32(40), 0.5)
	f.Add(uint32(41), 1.0)
	f.Add(uint32(1_000_000), 0.999)
	f.Add(uint32(41), 0.0)
	f.Fuzz(func(t *testing.T, rawPopulation uint32, foodShare float64) {
		population := Population(uint32(MinSplitSourcePopulation) + rawPopulation%2_000_000)
		food := float64(unit(foodShare) * FoodStorageCapacity(population))
		parent := Band{ID: 7, Population: population, StoredFood: FU(food), QueuedMigration: Some(MigrationOrder{Destination: 3}), InterbreedTarget: Some(BandID(9))}
		left, right, err := splitBand(parent, 8)
		if err != nil {
			t.Fatalf("split of %d people holding %v FU failed: %v", population, food, err)
		}
		if left.Population+right.Population != population || left.Population < right.Population || left.Population-right.Population > 1 {
			t.Fatalf("%d people split into %d and %d", population, left.Population, right.Population)
		}
		if sum := float64(left.StoredFood) + float64(right.StoredFood); math.Abs(sum-food) > float64(1e-9*math.Max(1, food)) {
			t.Fatalf("%v FU split into %v and %v, summing to %v", food, left.StoredFood, right.StoredFood, sum)
		}
		if float64(left.StoredFood) > FoodStorageCapacity(left.Population) || float64(right.StoredFood) > FoodStorageCapacity(right.Population) {
			t.Fatalf("a half holds more than it can store: %v/%v and %v/%v", left.StoredFood, FoodStorageCapacity(left.Population), right.StoredFood, FoodStorageCapacity(right.Population))
		}
		if left.ID != 7 || right.ID != 8 || left.QueuedMigration.Present() || right.QueuedMigration.Present() || left.InterbreedTarget.Present() || right.InterbreedTarget.Present() {
			t.Fatalf("split halves kept identity or orders wrongly: %+v / %+v", left, right)
		}
	})
}
