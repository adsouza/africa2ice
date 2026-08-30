package domain

import "testing"

func TestBandSplitConservesState(t *testing.T) {
	parent := Band{
		ID: 1, Population: 100, StoredFood: 240, Allocation: [AssignmentCount]AssignmentBP{3500, 3000, 1500, 500, 1500}, Heritable: sapiensEastAfricaTraits,
		LastFoodReport: FoodTurnReport{1, 100, 10}, LastMortality: MortalityReport{Seasonal: 1},
		LastOutcomeReport: OutcomeReport{Turn: 1, StartingPopulation: 101, EndingPopulation: 100, StartingHealth: 1, EndingHealth: 1},
	}
	left, right, err := splitBand(parent, 2)
	if err != nil {
		t.Fatal(err)
	}
	if left.Population+right.Population != parent.Population || left.StoredFood+right.StoredFood != parent.StoredFood {
		t.Fatal("split does not conserve population and food")
	}
	if left.StoredFood != 120 || right.StoredFood != 120 || FoodStorageCapacity(left.Population) != 150 {
		t.Fatalf("split values: %#v %#v", left, right)
	}
	if left.Heritable != parent.Heritable || right.Heritable != parent.Heritable || left.Allocation != parent.Allocation || right.Allocation != parent.Allocation {
		t.Fatal("split did not copy proportional state")
	}
	if left.LastFoodReport != (FoodTurnReport{}) || right.LastFoodReport != (FoodTurnReport{}) || left.LastMortality != (MortalityReport{}) || right.LastMortality != (MortalityReport{}) || left.LastOutcomeReport != (OutcomeReport{}) || right.LastOutcomeReport != (OutcomeReport{}) || !left.SpatialActionUsed || !right.SpatialActionUsed {
		t.Fatal("split lifecycle state incorrect")
	}
}

func TestBandSplitConservesOddWholePersonPopulation(t *testing.T) {
	parent := Band{ID: 1, Population: 101, Allocation: [AssignmentCount]AssignmentBP{3500, 3000, 1500, 500, 1500}}
	left, right, err := splitBand(parent, 2)
	if err != nil {
		t.Fatal(err)
	}
	if left.Population != 51 || right.Population != 50 || left.Population+right.Population != parent.Population {
		t.Fatalf("odd split populations = %v/%v", left.Population, right.Population)
	}
}

func TestAssignmentUsesWideSum(t *testing.T) {
	if ValidateAssignments([AssignmentCount]AssignmentBP{65_535, 65_535, 65_535, 65_535, 65_535}) == nil {
		t.Fatal("wrapped assignment accepted")
	}
	band := Band{Population: 100, Allocation: [AssignmentCount]AssignmentBP{3500, 3000, 1500, 500, 1500}}
	total := 0.0
	for role := WorkforceRole(0); role < AssignmentCount; role++ {
		total += band.Workers(role)
	}
	if total != 100 {
		t.Fatalf("workers sum to %v", total)
	}
}
