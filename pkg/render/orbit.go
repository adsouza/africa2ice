package render

import (
	"math"

	"github.com/solarlune/tetra3d"
)

const (
	defaultOrbitElevation = 1.19
	defaultOrbitScale     = 112
	minOrbitElevation     = 0.82
	maxOrbitElevation     = 1.42
	minOrbitScale         = 76
	maxOrbitScale         = 150
)

// OrbitCamera is session-local presentation state. Its revision invalidates
// only camera-sized render caches and never terrain topology or game state.
type OrbitCamera struct {
	focusX, focusZ float32
	azimuth        float32
	elevation      float32
	orthoScale     float32
	revision       uint64
	initialized    bool
}

func NewOrbitCamera() OrbitCamera {
	return OrbitCamera{elevation: defaultOrbitElevation, orthoScale: defaultOrbitScale, initialized: true}
}

func (orbit *OrbitCamera) Rotate(deltaAzimuth, deltaElevation float32) bool {
	if orbit == nil || deltaAzimuth == 0 && deltaElevation == 0 {
		return false
	}
	orbit.ensureInitialized()
	nextElevation := min(max(orbit.elevation+deltaElevation, minOrbitElevation), maxOrbitElevation)
	nextAzimuth := orbit.azimuth + deltaAzimuth
	if nextElevation == orbit.elevation && nextAzimuth == orbit.azimuth {
		return false
	}
	orbit.elevation, orbit.azimuth = nextElevation, nextAzimuth
	orbit.revision++
	return true
}

func (orbit *OrbitCamera) Zoom(delta float32) bool {
	if orbit == nil || delta == 0 {
		return false
	}
	orbit.ensureInitialized()
	next := min(max(orbit.orthoScale-delta, minOrbitScale), maxOrbitScale)
	if next == orbit.orthoScale {
		return false
	}
	orbit.orthoScale = next
	orbit.revision++
	return true
}

func (orbit *OrbitCamera) Pan(deltaX, deltaZ float32) bool {
	if orbit == nil || deltaX == 0 && deltaZ == 0 {
		return false
	}
	orbit.ensureInitialized()
	nextX := min(max(orbit.focusX+deltaX, -30), 30)
	nextZ := min(max(orbit.focusZ+deltaZ, -20), 20)
	if nextX == orbit.focusX && nextZ == orbit.focusZ {
		return false
	}
	orbit.focusX, orbit.focusZ = nextX, nextZ
	orbit.revision++
	return true
}

func (orbit *OrbitCamera) Apply(camera *tetra3d.Camera) {
	if orbit == nil || camera == nil {
		return
	}
	orbit.ensureInitialized()
	const distance = float32(108)
	horizontal := distance * float32(math.Cos(float64(orbit.elevation)))
	position := tetra3d.NewVector3(
		orbit.focusX+horizontal*float32(math.Sin(float64(orbit.azimuth))),
		distance*float32(math.Sin(float64(orbit.elevation))),
		orbit.focusZ+horizontal*float32(math.Cos(float64(orbit.azimuth))),
	)
	camera.SetWorldPositionVec(position)
	rotation := tetra3d.NewMatrix4Rotate(0, 1, 0, orbit.azimuth).Rotated(1, 0, 0, -orbit.elevation)
	camera.SetLocalRotation(rotation)
	camera.SetOrthoScale(orbit.orthoScale)
}

func (orbit *OrbitCamera) ensureInitialized() {
	if orbit.initialized {
		return
	}
	*orbit = NewOrbitCamera()
}
