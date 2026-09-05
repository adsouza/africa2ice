package render

import (
	"math"
	"testing"

	"github.com/hajimehoshi/ebiten/v2"
)

func TestShimmerNoiseIsBoundedAndDeterministic(t *testing.T) {
	for step := range 200 {
		for x := range 17 {
			for y := range 13 {
				got := shimmerNoise(x, y, step)
				if got < 0 || got >= 1 {
					t.Fatalf("shimmerNoise(%d, %d, %d) = %v, want [0, 1)", x, y, step, got)
				}
				if again := shimmerNoise(x, y, step); again != got {
					t.Fatalf("shimmerNoise(%d, %d, %d) is not deterministic: %v then %v", x, y, step, got, again)
				}
			}
		}
	}
}

// The halo must twinkle tile by tile rather than pulse as one sheet, so
// neighbouring tiles must not share a value at the same step.
func TestShimmerNoiseDecorrelatesNeighbouringTiles(t *testing.T) {
	identical := 0
	for x := range 40 {
		for y := range 40 {
			if math.Abs(shimmerNoise(x, y, 7)-shimmerNoise(x+1, y, 7)) < 1e-12 {
				identical++
			}
		}
	}
	if identical > 0 {
		t.Fatalf("%d horizontally adjacent tile pairs shimmer identically", identical)
	}
}

// Tiles must also re-roll on different steps, or the field steps in lockstep
// even though the values differ.
func TestShimmerRollBoundariesAreSpreadAcrossTiles(t *testing.T) {
	seen := map[int]bool{}
	for x := range 64 {
		for y := range 64 {
			// The offset a tile uses is what staggers its rolls; recover it by
			// finding the step at which its interpolation fraction returns to 0.
			for step := range shimmerStepsPerRoll {
				if shimmerFraction(x, y, step) == 0 {
					seen[step] = true
				}
			}
		}
	}
	if len(seen) != shimmerStepsPerRoll {
		t.Fatalf("tiles restart on %d of %d possible steps; the field is not staggered", len(seen), shimmerStepsPerRoll)
	}
}

func TestShimmerNoiseChangesOverTime(t *testing.T) {
	first := shimmerNoise(5, 5, 0)
	changed := false
	for step := 1; step <= 4*shimmerStepsPerRoll; step++ {
		if shimmerNoise(5, 5, step) != first {
			changed = true
			break
		}
	}
	if !changed {
		t.Fatal("shimmerNoise never varied over four roll periods")
	}
}

// The shimmer's documented rates -- 15 phase steps a second, five per-tile
// re-rolls a second -- are quoted in this package's comments, in the design
// spec, and in DESIGN.md. They are consequences of these two constants and
// Ebitengine's tick rate, so a constant edited without updating the prose
// would leave three documents lying. Pin the derivation, not the prose.
func TestShimmerRatesMatchTheirDocumentedValues(t *testing.T) {
	const wantPhaseStepsPerSecond, wantRollsPerSecond = 15, 5
	if ebiten.DefaultTPS%shimmerTickStride != 0 {
		t.Fatalf("shimmerTickStride %d does not divide ebiten.DefaultTPS %d evenly, so there is no whole phase rate to document",
			shimmerTickStride, ebiten.DefaultTPS)
	}
	if steps := ebiten.DefaultTPS / shimmerTickStride; steps != wantPhaseStepsPerSecond {
		t.Errorf("phase steps per second = %d, want %d (ebiten.DefaultTPS %d / shimmerTickStride %d)",
			steps, wantPhaseStepsPerSecond, ebiten.DefaultTPS, shimmerTickStride)
	}
	if rolls := ebiten.DefaultTPS / shimmerTickStride / shimmerStepsPerRoll; rolls != wantRollsPerSecond {
		t.Errorf("re-rolls per second = %d, want %d (that rate / shimmerStepsPerRoll %d)",
			rolls, wantRollsPerSecond, shimmerStepsPerRoll)
	}
}
