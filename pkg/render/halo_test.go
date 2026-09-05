package render

import (
	"math"
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
//
// This compares water's brightest attainable colour (noise = 1, its ring
// target) against each biome's darkest attainable colour (noise = 0, its
// jittered floor): the worst case that can appear on screen at once. Three
// anchor aridities are not enough to trust this — water is continuous in
// AridityIndex, and an intermediate blend can land nearer a biome than either
// endpoint does, exactly as biome_contrast_test.go's waterGradeSamples comment
// explains — so this sweeps aridity at that same density instead. haloColor
// returns a color.RGBA, whose channels are already rounded to 8-bit sRGB by
// interpolateChannel, so cieLightness of its output is already a quantised
// comparison; no separate rounding step is needed here.
func TestHaloWaterIsDarkerThanEveryLandBiomeInTheSameRing(t *testing.T) {
	worstGap, worstRing, worstAridity, worstBiome := math.Inf(1), 0, 0.0, ""
	for ring := range haloRingCount {
		for sample := range waterGradeSamples {
			aridity := float64(sample) / float64(waterGradeSamples-1)
			water := EpochGrade(aridity).Water
			wet := cieLightness(haloColor(water, haloWaterBlend(water)[ring], 1))
			for biome := range gameapi.Biome(gameapi.BiomeCount) {
				swatch := climateBiomeColor(biome, 0)
				dry := cieLightness(haloColor(swatch, haloLandBlend[biome][ring], 0))
				if gap := dry - wet; gap < worstGap {
					worstGap, worstRing, worstAridity, worstBiome = gap, ring, aridity, biome.String()
				}
			}
		}
	}
	t.Logf("worst water-vs-land gap: ring %d aridity %.4f biome %s, water below land by %.4f L*",
		worstRing, worstAridity, worstBiome, worstGap)
	if worstGap <= 0 {
		t.Fatalf("ring %d aridity %.4f: water is not darker than %s (gap %.4f L*)", worstRing, worstAridity, worstBiome, worstGap)
	}
}

// TestHaloWaterRingsDimWithDistance sweeps the same cross-ring ordering that
// TestHaloRingsDimWithDistance checks for biomes, but for water: at every
// aridity, a ring's darkest attainable water colour (noise = 0, its jittered
// floor) must stay above the next ring's brightest attainable water colour
// (noise = 1, its target) -- the worst case across independent shimmer phases
// for two different rings' tiles. Nothing else in this test file swept water
// across rings; TestHaloRingsDimWithDistance only loops biomes.
func TestHaloWaterRingsDimWithDistance(t *testing.T) {
	worstGap, worstRing, worstAridity := math.Inf(1), 0, 0.0
	for sample := range waterGradeSamples {
		aridity := float64(sample) / float64(waterGradeSamples-1)
		water := EpochGrade(aridity).Water
		blend := haloWaterBlend(water)
		for ring := 0; ring < haloRingCount-1; ring++ {
			nearFloor := cieLightness(haloColor(water, blend[ring], 0))
			farTarget := cieLightness(haloColor(water, blend[ring+1], 1))
			if gap := nearFloor - farTarget; gap < worstGap {
				worstGap, worstRing, worstAridity = gap, ring, aridity
			}
		}
	}
	t.Logf("worst cross-ring water gap: ring %d floor vs ring %d target at aridity %.4f, margin %.4f L*",
		worstRing, worstRing+1, worstAridity, worstGap)
	if worstGap <= 0 {
		t.Fatalf("ring %d's darkest water is not above ring %d's brightest water at aridity %.4f (margin %.4f L*)",
			worstRing, worstRing+1, worstAridity, worstGap)
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

// haloTestFrame builds a full grid with a single explored tile at (centreX,
// centreY), which is the cleanest way to read a distance field back out.
func haloTestFrame(centreX, centreY int) *gameapi.Frame {
	frame := &gameapi.Frame{Tiles: make([]gameapi.Tile, TerrainGridWidth*TerrainGridHeight)}
	for y := range TerrainGridHeight {
		for x := range TerrainGridWidth {
			id := gameapi.TileID(y*TerrainGridWidth + x)
			frame.Tiles[id] = gameapi.Tile{ID: id, X: x, Y: y, Land: true}
		}
	}
	frame.Tiles[centreY*TerrainGridWidth+centreX].Explored = true
	return frame
}

func TestHaloDistancesAreChebyshev(t *testing.T) {
	const centreX, centreY = 40, 30
	distances := haloDistances(haloTestFrame(centreX, centreY))
	tests := []struct {
		name string
		x, y int
		want uint8
	}{
		{name: "the explored tile itself", x: centreX, y: centreY, want: 0},
		{name: "orthogonal neighbour", x: centreX + 1, y: centreY, want: 1},
		{name: "diagonal neighbour is also ring 1", x: centreX + 1, y: centreY + 1, want: 1},
		{name: "knight-ish offset is ring 2 under Chebyshev", x: centreX + 2, y: centreY + 1, want: 2},
		{name: "far diagonal corner of ring 3", x: centreX + 3, y: centreY + 3, want: 3},
		{name: "one past the last ring", x: centreX + 4, y: centreY, want: haloFar},
		{name: "far away", x: centreX + 20, y: centreY + 20, want: haloFar},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if got := distances[test.y*TerrainGridWidth+test.x]; got != test.want {
				t.Fatalf("distance at (%d, %d) = %d, want %d", test.x, test.y, got, test.want)
			}
		})
	}
}

func TestHaloDistancesTakeTheNearestExploredTile(t *testing.T) {
	frame := haloTestFrame(40, 30)
	frame.Tiles[30*TerrainGridWidth+45].Explored = true
	distances := haloDistances(frame)
	// (43, 30) is 3 from the left anchor and 2 from the right one.
	if got := distances[30*TerrainGridWidth+43]; got != 2 {
		t.Fatalf("distance at (43, 30) = %d, want 2 from the nearer anchor", got)
	}
}

func TestHaloDistancesAreZeroForEveryExploredTile(t *testing.T) {
	frame := haloTestFrame(40, 30)
	for index := range frame.Tiles {
		frame.Tiles[index].Explored = true
	}
	for index, distance := range haloDistances(frame) {
		if distance != 0 {
			t.Fatalf("tile %d of a fully explored frame has distance %d", index, distance)
		}
	}
}

func TestHaloDistancesAreAllFarWithNothingExplored(t *testing.T) {
	frame := haloTestFrame(40, 30)
	frame.Tiles[30*TerrainGridWidth+40].Explored = false
	for index, distance := range haloDistances(frame) {
		if distance != haloFar {
			t.Fatalf("tile %d of an unexplored frame has distance %d", index, distance)
		}
	}
}
