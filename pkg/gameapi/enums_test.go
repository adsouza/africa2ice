package gameapi

import "testing"

func TestClosedCatalogCountsAndStrings(t *testing.T) {
	if FaunaGroupCount != 6 || TechCount != 9 || HeritableTraitCount != 6 || AssignmentCount != 5 || RegionCount != 13 || PassageCount != 3 {
		t.Fatal("closed catalog count changed")
	}
	for value, limit := range map[int]int{int(BiomeCount): 6, int(SpeciesCount): 2, int(SeasonCount): 4} {
		if value != limit {
			t.Fatalf("catalog count = %d, want %d", value, limit)
		}
	}
}

func TestFoodTurnReportDerivations(t *testing.T) {
	r := FoodTurnReport{Turn: 1, RequiredFU: 50, DeficitFU: 10}
	if r.ConsumedFU() != 40 || r.DeficitFraction() != 0.2 {
		t.Fatalf("unexpected food derivation: %#v", r)
	}
	if (FoodTurnReport{}).DeficitFraction() != 0 {
		t.Fatal("zero requirement must have zero deficit fraction")
	}
}
