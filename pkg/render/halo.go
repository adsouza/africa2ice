package render

import (
	"image/color"

	"github.com/adsouza/africa2ice/pkg/gameapi"
)

// haloRingCount is how many rings of unexplored tiles around the explored set
// show a trace of their real terrain. Beyond it the fog stays flat.
const haloRingCount = 3

// haloSolveIterations bisects the blend fraction to well under a quantisation
// step of an 8-bit channel; more iterations cannot change the rounded colour.
const haloSolveIterations = 40

// haloRing is one ring's slice of the lightness range between the fog and the
// darkest explored colour. Land spans landFloor..landFloor+landSpan, mapped
// order-preservingly from the explored biome ladder; water sits at its own
// lower lightness so a coastline reads as a lightness edge rather than a hue
// change, which is not available this far down. jitter is how far the shimmer
// may pull a tile below its target, subtractively, so the ring's ceiling holds
// structurally instead of by a clamp a later edit could drop.
type haloRing struct {
	landFloor float64
	landSpan  float64
	water     float64
	jitter    float64
}

var haloRings = [haloRingCount]haloRing{
	{landFloor: 12.0, landSpan: 12.0, water: 8.0, jitter: 1.5},
	{landFloor: 8.0, landSpan: 8.0, water: 6.0, jitter: 1.0},
	{landFloor: 5.0, landSpan: 4.5, water: 4.2, jitter: 0.6},
}

// haloBlendPair is the blend fraction along fog -> terrain at the jittered
// floor and at the ring's target, in that order.
type haloBlendPair [2]float64

var (
	darkestBiomeLightness, brightestBiomeLightness = biomeLightnessRange()
	haloLandBlend                                  = solveHaloLandBlend()
)

func biomeLightnessRange() (darkest, brightest float64) {
	darkest, brightest = cieLightness(climateBiomeColor(0, 0)), cieLightness(climateBiomeColor(0, 0))
	for biome := range gameapi.Biome(gameapi.BiomeCount) {
		lightness := cieLightness(climateBiomeColor(biome, 0))
		darkest, brightest = min(darkest, lightness), max(brightest, lightness)
	}
	return darkest, brightest
}

func solveHaloLandBlend() [gameapi.BiomeCount][haloRingCount]haloBlendPair {
	var table [gameapi.BiomeCount][haloRingCount]haloBlendPair
	for biome := range gameapi.Biome(gameapi.BiomeCount) {
		swatch := climateBiomeColor(biome, 0)
		position := (cieLightness(swatch) - darkestBiomeLightness) / (brightestBiomeLightness - darkestBiomeLightness)
		for ring := range haloRingCount {
			target := haloRings[ring].landFloor + position*haloRings[ring].landSpan
			table[biome][ring] = haloBlendPair{
				haloBlendFor(swatch, target-haloRings[ring].jitter),
				haloBlendFor(swatch, target),
			}
		}
	}
	return table
}

// haloWaterBlend solves the water ring targets for one frame's grade. Water
// cannot join the init-time table because EpochGrade varies continuously with
// the aridity index; the caller solves it once per terrain rebuild, not per
// tile and not per frame.
func haloWaterBlend(water color.RGBA) [haloRingCount]haloBlendPair {
	var pairs [haloRingCount]haloBlendPair
	for ring := range haloRingCount {
		pairs[ring] = haloBlendPair{
			haloBlendFor(water, haloRings[ring].water-haloRings[ring].jitter),
			haloBlendFor(water, haloRings[ring].water),
		}
	}
	return pairs
}

// haloBlendFor finds the blend fraction along fog -> target whose result has
// the requested lightness. Every channel of every terrain colour exceeds the
// corresponding fog channel, so L* is strictly monotonic in the fraction and
// bisection needs no guard against a non-monotonic segment.
func haloBlendFor(target color.RGBA, lightness float64) float64 {
	low, high := 0.0, 1.0
	for range haloSolveIterations {
		middle := (low + high) / 2
		if cieLightness(interpolateRGBA(unexploredTileColor, target, middle)) < lightness {
			low = middle
		} else {
			high = middle
		}
	}
	return (low + high) / 2
}

// haloColor places a tile between its ring's jittered floor and its target.
// Interpolating the two solved fractions rather than re-solving per sample
// costs under 0.05 L* of nonlinearity across a band at most 1.5 L* wide.
func haloColor(base color.RGBA, blend haloBlendPair, noise float64) color.RGBA {
	return interpolateRGBA(unexploredTileColor, base, lerp(blend[0], blend[1], noise))
}
