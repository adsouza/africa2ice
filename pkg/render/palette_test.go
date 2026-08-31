package render

import (
	"image/color"
	"math"
	"reflect"
	"testing"
)

func TestGradeColorsHasExactlyTheTwoSpecifiedOutputs(t *testing.T) {
	if fields := reflect.TypeOf(GradeColors{}).NumField(); fields != 2 {
		t.Fatalf("GradeColors fields = %d, want 2", fields)
	}
}

func TestEpochGradeLockedAnchorsAndMidpoints(t *testing.T) {
	tests := []struct {
		name  string
		index float64
		want  GradeColors
	}{
		{name: "humid anchor", index: 0, want: gradeFixture(0x206c9cff, 0x9e9e48ff)},
		{name: "humid-transition midpoint", index: 0.25, want: gradeFixture(0x2c6a90ff, 0xa28f45ff)},
		{name: "transition anchor", index: 0.5, want: gradeFixture(0x386884ff, 0xa68042ff)},
		{name: "transition-glacial midpoint", index: 0.75, want: gradeFixture(0x365e7cff, 0x8b9587ff)},
		{name: "glacial anchor", index: 1, want: gradeFixture(0x345474ff, 0x70aaccff)},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if got := EpochGrade(test.index); !gradeEqual(got, test.want) {
				t.Fatalf("EpochGrade(%v) = %#v, want %#v", test.index, got, test.want)
			}
		})
	}
}

func gradeEqual(got, want GradeColors) bool {
	return got.Water == want.Water && got.HUDChromeAccent == want.HUDChromeAccent
}

func TestEpochGradeClampsMalformedOrOutOfRangePresentationInput(t *testing.T) {
	humid, glacial := EpochGrade(0), EpochGrade(1)
	for _, input := range []float64{math.NaN(), math.Inf(-1), -1} {
		if got := EpochGrade(input); got != humid {
			t.Fatalf("EpochGrade(%v) = %#v, want humid endpoint", input, got)
		}
	}
	for _, input := range []float64{math.Inf(1), 2} {
		if got := EpochGrade(input); got != glacial {
			t.Fatalf("EpochGrade(%v) = %#v, want glacial endpoint", input, got)
		}
	}
}

func TestEpochGradeIsContinuousAndBoundedAcrossCampaignSamples(t *testing.T) {
	previous := EpochGrade(0)
	for turn := 0; turn <= 400; turn++ {
		grade := EpochGrade(float64(turn) / 400)
		if turn > 0 && (channelDistance(previous.Water, grade.Water) > 2 || channelDistance(previous.HUDChromeAccent, grade.HUDChromeAccent) > 2) {
			t.Fatalf("turn %d has a discontinuous grade jump: %#v -> %#v", turn, previous, grade)
		}
		previous = grade
	}
}

func gradeFixture(water, accent uint32) GradeColors {
	return GradeColors{Water: rgbaHex(water), HUDChromeAccent: rgbaHex(accent)}
}

func rgbaHex(value uint32) color.RGBA {
	return color.RGBA{R: uint8(value >> 24), G: uint8(value >> 16), B: uint8(value >> 8), A: uint8(value)}
}

func channelDistance(first, second color.RGBA) int {
	distance := func(a, b uint8) int {
		if a > b {
			return int(a - b)
		}
		return int(b - a)
	}
	return max(distance(first.R, second.R), distance(first.G, second.G), distance(first.B, second.B), distance(first.A, second.A))
}
