package render

import (
	"math"
	"testing"

	"github.com/adsouza/africa2ice/pkg/gameapi"
)

// The halo ladder in halo.go is derived from these exact lightnesses, and the
// bands it allocates sit just below the darkest of them. Pinning the values
// here means a palette edit fails with a number rather than silently sliding
// the halo up against the explored floor.
func TestBiomeLightnessesArePinned(t *testing.T) {
	want := map[gameapi.Biome]float64{
		gameapi.RiverineWoodland:     29.4,
		gameapi.MountainousHighlands: 39.0,
		gameapi.CoastalShrubland:     51.1,
		gameapi.Savanna:              59.5,
		gameapi.SemiAridDesert:       71.6,
		gameapi.GlacialTundra:        79.9,
	}
	for biome, expected := range want {
		got := cieLightness(climateBiomeColor(biome, 0))
		if math.Abs(got-expected) > 0.05 {
			t.Errorf("cieLightness(%v) = %.2f, want %.2f", biome, got, expected)
		}
	}
}

func TestFogLightnessIsPinned(t *testing.T) {
	if got := cieLightness(unexploredTileColor); math.Abs(got-2.82) > 0.05 {
		t.Fatalf("cieLightness(unexploredTileColor) = %.2f, want 2.82", got)
	}
}
