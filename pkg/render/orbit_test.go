package render

import "testing"

func TestOrbitCameraClampsElevationZoomAndPan(t *testing.T) {
	orbit := NewOrbitCamera()
	if !orbit.Rotate(0.2, 99) || orbit.elevation != maxOrbitElevation || orbit.revision != 1 {
		t.Fatalf("clamped rotation = %#v", orbit)
	}
	if !orbit.Zoom(999) || orbit.orthoScale != minOrbitScale || orbit.revision != 2 {
		t.Fatalf("clamped zoom = %#v", orbit)
	}
	if orbit.Zoom(1) {
		t.Fatal("zoom advanced at its clamp")
	}
	if !orbit.Pan(99, -99) || orbit.focusX != 30 || orbit.focusZ != -20 || orbit.revision != 3 {
		t.Fatalf("clamped pan = %#v", orbit)
	}
	if orbit.Pan(1, -1) {
		t.Fatal("pan advanced at both clamps")
	}
}
