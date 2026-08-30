package domain

import (
	"math"
	"testing"
)

func TestFaunaProfilesCompleteAndNormalized(t *testing.T) {
	seen := [FaunaArchetypeCount]bool{}
	for region := Region(0); region < RegionCount; region++ {
		for biome := Biome(0); biome < BiomeCount; biome++ {
			profile, ok := FaunaFor(region, biome, true)
			if !ok {
				t.Fatalf("missing %v/%v", region, biome)
			}
			seen[profile.Archetype] = true
			total := 0.0
			for _, weight := range profile.Weights {
				total += weight
			}
			if math.Abs(total-1) > 1e-12 {
				t.Fatalf("profile %d sums to %.17g", profile.Archetype, total)
			}
			if profile.MegafaunaSupported != (profile.Weights[Megafauna] > 0) {
				t.Fatalf("profile %d megafauna mismatch", profile.Archetype)
			}
			assignment, ok := ArchaicAssignment(region, biome)
			if !ok || ValidateAssignments(assignment) != nil {
				t.Fatalf("invalid archaic assignment %v/%v", region, biome)
			}
		}
	}
	for archetype, reached := range seen {
		if !reached {
			t.Fatalf("archetype %d is unreachable", archetype)
		}
	}
}

func TestFaunaExceptions(t *testing.T) {
	tests := []struct {
		region Region
		biome  Biome
		want   FaunaArchetype
	}{
		{Arabia, Savanna, AridSmallGame}, {SoutheastAsia, CoastalShrubland, TropicalIsland},
		{Sahul, CoastalShrubland, TropicalIsland}, {Beringia, CoastalShrubland, BeringianCoast}, {Beringia, GlacialTundra, BeringianCoast},
	}
	for _, test := range tests {
		profile, _ := FaunaFor(test.region, test.biome, true)
		if profile.Archetype != test.want {
			t.Fatalf("%v/%v = %v", test.region, test.biome, profile.Archetype)
		}
	}
}
