package render

// The shimmer advances on a frame count, never a wall clock: -screenshot ticks
// a fixed number of times, and a clock would make its output irreproducible.
// Ebitengine runs at its default 60 TPS (nothing in this repo calls SetTPS).
const (
	// shimmerTickStride is how many Update ticks pass between phase steps.
	// Four gives 15 steps a second. This is the axis to turn if the motion
	// reads as steppy: stride 2 doubles the substeps at the same roll rate,
	// at double the paint rate.
	shimmerTickStride = 4
	// shimmerStepsPerRoll is how many phase steps a tile takes to reach its
	// next target, so tiles re-roll five times a second and the easing curve
	// is sampled at three points per transition.
	shimmerStepsPerRoll = 3
)

// shimmerNoise is a tile's position between its ring's jittered floor and its
// target at one phase step, in [0, 1). It is a pure hash rather than a stateful
// generator: pkg/render cannot reach the simulation RNG (internal/archtest
// pins its imports), and a hash keeps -screenshot reproducible for free.
func shimmerNoise(x, y, step int) float64 {
	cycle := shimmerCycle(x, y, step)
	from, to := unitNoise(hashTile(x, y, cycle+1)), unitNoise(hashTile(x, y, cycle+2))
	return lerp(from, to, smoothstep(shimmerFraction(x, y, step)))
}

// shimmerOffset staggers when a tile re-rolls, so the field twinkles tile by
// tile instead of stepping as one sheet. Cycle 0 is reserved for it, which is
// why shimmerNoise reads cycles from 1.
func shimmerOffset(x, y int) int {
	return int(hashTile(x, y, 0) % shimmerStepsPerRoll)
}

func shimmerCycle(x, y, step int) int {
	return (step + shimmerOffset(x, y)) / shimmerStepsPerRoll
}

func shimmerFraction(x, y, step int) float64 {
	return float64((step+shimmerOffset(x, y))%shimmerStepsPerRoll) / shimmerStepsPerRoll
}

func smoothstep(fraction float64) float64 {
	return fraction * fraction * (3 - 2*fraction)
}

// hashTile is murmur3's fmix32 over the mixed coordinates and cycle.
func hashTile(x, y, cycle int) uint32 {
	value := uint32(x)*0x9e3779b1 ^ uint32(y)*0x85ebca77 ^ uint32(cycle)*0xc2b2ae3d
	value ^= value >> 16
	value *= 0x7feb352d
	value ^= value >> 15
	value *= 0x846ca68b
	value ^= value >> 16
	return value
}

// unitNoise keeps the top 24 bits, which a float64 represents exactly.
func unitNoise(value uint32) float64 {
	return float64(value>>8) / float64(1<<24)
}
