package gameapi

// Shared deterministic policy values live at the dependency-free boundary so
// the simulation and the release-verification driver cannot silently drift.
const (
	MinEstablishedBand                = 20
	MinSplitSourcePopulation          = 2 * MinEstablishedBand
	SplitStressThreshold              = 0.67
	ReferenceRouteDeparturePopulation = 5 * MinEstablishedBand / 2
)

// ReferenceRouteStepCost turns destination capacity into the integer weight
// used by both deterministic route solvers.
func ReferenceRouteStepCost(capacity float64) int {
	switch {
	case capacity >= 100:
		return 1
	case capacity >= 75:
		return 2
	case capacity >= 50:
		return 4
	case capacity >= 25:
		return 8
	case capacity >= 10:
		return 16
	default:
		return 32
	}
}
