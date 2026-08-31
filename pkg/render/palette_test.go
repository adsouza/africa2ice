package render

import (
	"image/color"
	"math"
	"reflect"
	"testing"
)

func TestGradeColorsHasExactlyTheSixSpecifiedOutputs(t *testing.T) {
	if fields := reflect.TypeOf(GradeColors{}).NumField(); fields != 6 {
		t.Fatalf("GradeColors fields = %d, want 6", fields)
	}
}

func TestEpochGradeLockedAnchorsAndMidpoints(t *testing.T) {
	tests := []struct {
		name  string
		index float64
		want  GradeColors
	}{
		{name: "humid anchor", index: 0, want: gradeFixture(0xffe8bcff, 0.94, 0.72, 0x485860ff, 0x206c9cff, 0x9e9e48ff)},
		{name: "humid-transition midpoint", index: 0.25, want: gradeFixture(0xf8e1b9ff, 0.88, 0.66, 0x4c5960ff, 0x2c6a90ff, 0xa28f45ff)},
		{name: "transition anchor", index: 0.5, want: gradeFixture(0xf1dab6ff, 0.82, 0.60, 0x505a60ff, 0x386884ff, 0xa68042ff)},
		{name: "transition-glacial midpoint", index: 0.75, want: gradeFixture(0xe0ddd3ff, 0.76, 0.54, 0x4e5964ff, 0x365e7cff, 0x8b9587ff)},
		{name: "glacial anchor", index: 1, want: gradeFixture(0xcfe0f0ff, 0.70, 0.48, 0x4c5868ff, 0x345474ff, 0x70aaccff)},
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
	return got.DirectionalLight == want.DirectionalLight &&
		math.Abs(got.DirectionalLightIntensity-want.DirectionalLightIntensity) <= 1e-12 &&
		math.Abs(got.AmbientLevel-want.AmbientLevel) <= 1e-12 &&
		got.Veil == want.Veil && got.Water == want.Water && got.HUDChromeAccent == want.HUDChromeAccent
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
		if !finiteUnitInterval(grade.DirectionalLightIntensity) || !finiteUnitInterval(grade.AmbientLevel) {
			t.Fatalf("turn %d has invalid scalar grade %#v", turn, grade)
		}
		if turn > 0 && channelDistance(previous.DirectionalLight, grade.DirectionalLight) > 2 {
			t.Fatalf("turn %d has a discontinuous directional-light jump: %v -> %v", turn, previous.DirectionalLight, grade.DirectionalLight)
		}
		previous = grade
	}
}

func gradeFixture(light uint32, intensity, ambient float64, veil, water, accent uint32) GradeColors {
	return GradeColors{
		DirectionalLight: rgbaHex(light), DirectionalLightIntensity: intensity, AmbientLevel: ambient,
		Veil: rgbaHex(veil), Water: rgbaHex(water), HUDChromeAccent: rgbaHex(accent),
	}
}

func rgbaHex(value uint32) color.RGBA {
	return color.RGBA{R: uint8(value >> 24), G: uint8(value >> 16), B: uint8(value >> 8), A: uint8(value)}
}

func finiteUnitInterval(value float64) bool {
	return !math.IsNaN(value) && !math.IsInf(value, 0) && value >= 0 && value <= 1
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
