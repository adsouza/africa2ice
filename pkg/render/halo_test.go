package render

import (
	"testing"

	"github.com/adsouza/africa2ice/pkg/gameapi"
)

// darkestExploredLightness is the floor the whole halo must stay under: an
// explored riverine woodland tile. If a fogged tile can be brighter than this,
// the explored/unexplored distinction inverts at exactly the boundary that
// matters, which is the defect that killed the original alpha-blend design.
func darkestExploredLightness() float64 {
	darkest := cieLightness(climateBiomeColor(0, 0))
	for biome := range gameapi.Biome(gameapi.BiomeCount) {
		if l := cieLightness(climateBiomeColor(biome, 0)); l < darkest {
			darkest = l
		}
	}
	return darkest
}

func TestHaloStaysBelowTheDarkestExploredColor(t *testing.T) {
	floor := darkestExploredLightness()
	brightest, where := 0.0, ""
	for ring := range haloRingCount {
		for biome := range gameapi.Biome(gameapi.BiomeCount) {
			for _, noise := range []float64{0, 1} {
				got := cieLightness(haloColor(climateBiomeColor(biome, 0), haloLandBlend[biome][ring], noise))
				if got > brightest {
					brightest, where = got, biome.String()
				}
			}
		}
		for _, aridity := range []float64{0, 0.5, 1} {
			water := EpochGrade(aridity).Water
			for _, noise := range []float64{0, 1} {
				got := cieLightness(haloColor(water, haloWaterBlend(water)[ring], noise))
				if got > brightest {
					brightest, where = got, "water"
				}
			}
		}
	}
	if brightest >= floor {
		t.Fatalf("brightest halo colour L* %.2f (%s) reaches the explored floor L* %.2f", brightest, where, floor)
	}
	t.Logf("brightest halo L* %.2f (%s) against floor %.2f", brightest, where, floor)
}

func TestHaloPreservesTheBiomeLadderWithinARing(t *testing.T) {
	for ring := range haloRingCount {
		type entry struct {
			explored float64
			halo     float64
			name     string
		}
		entries := make([]entry, 0, gameapi.BiomeCount)
		for biome := range gameapi.Biome(gameapi.BiomeCount) {
			swatch := climateBiomeColor(biome, 0)
			entries = append(entries, entry{
				explored: cieLightness(swatch),
				halo:     cieLightness(haloColor(swatch, haloLandBlend[biome][ring], 1)),
				name:     biome.String(),
			})
		}
		for i := range entries {
			for j := range entries {
				if entries[i].explored < entries[j].explored && entries[i].halo >= entries[j].halo {
					t.Errorf("ring %d: %s is darker than %s when explored but not in the halo", ring, entries[i].name, entries[j].name)
				}
			}
		}
	}
}

// At halo lightnesses chroma discrimination has collapsed, so land and sea
// cannot be told apart by hue. Water therefore sits below every land biome in
// its own ring, making the coastline a lightness edge.
func TestHaloWaterIsDarkerThanEveryLandBiomeInTheSameRing(t *testing.T) {
	for ring := range haloRingCount {
		for _, aridity := range []float64{0, 0.5, 1} {
			water := EpochGrade(aridity).Water
			wet := cieLightness(haloColor(water, haloWaterBlend(water)[ring], 1))
			for biome := range gameapi.Biome(gameapi.BiomeCount) {
				swatch := climateBiomeColor(biome, 0)
				dry := cieLightness(haloColor(swatch, haloLandBlend[biome][ring], 0))
				if wet >= dry {
					t.Errorf("ring %d aridity %.1f: water L* %.2f is not below %s L* %.2f", ring, aridity, wet, biome.String(), dry)
				}
			}
		}
	}
}

func TestHaloRingsDimWithDistance(t *testing.T) {
	for biome := range gameapi.Biome(gameapi.BiomeCount) {
		swatch := climateBiomeColor(biome, 0)
		previous := cieLightness(swatch)
		for ring := range haloRingCount {
			got := cieLightness(haloColor(swatch, haloLandBlend[biome][ring], 1))
			if got >= previous {
				t.Fatalf("%s ring %d L* %.2f did not dim below %.2f", biome.String(), ring, got, previous)
			}
			previous = got
		}
	}
}
