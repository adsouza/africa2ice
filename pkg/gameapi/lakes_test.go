package gameapi

import "testing"

func TestLakeStageTransitions(t *testing.T) {
	for _, tt := range []struct {
		name          string
		year          int
		before, after LakeStage
	}{
		{"Lake Malawi / Nyasa", 60000, MalawiEarlyLow, MalawiRecovered},
		{"Lake Malawi / Nyasa", 35000, MalawiRecovered, MalawiGlacialLow},
		{"Lake Lisan", 70000, LakeUnstaged, LisanInitial},
		{"Lake Lisan", 27000, LisanInitial, LisanHigh},
		{"Lake Lisan", 23000, LisanHigh, LisanDeclining},
	} {
		if LakeStageAt(tt.name, tt.year+1) != tt.before || LakeStageAt(tt.name, tt.year) != tt.after || LakeStageAt(tt.name, tt.year-1) != tt.after {
			t.Fatalf("wrong boundary at %s %d", tt.name, tt.year)
		}
		if LakeStageStart(tt.after) != tt.year {
			t.Fatal("notes and map stage clocks disagree")
		}
	}
	if LakeStageAt("Lake Baikal", 20000) != LakeUnstaged {
		t.Fatal("static lake received a changing stage")
	}
}
