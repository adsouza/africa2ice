package render

import (
	"math"
	"testing"
)

func TestNextViewportUsesDPIAwareBoundedPhysicalDimensions(t *testing.T) {
	tests := []struct {
		name       string
		scale      float64
		wantScale  float64
		wantWidth  int
		wantHeight int
	}{
		{name: "one", scale: 1, wantScale: 1, wantWidth: 1281, wantHeight: 721},
		{name: "one and a quarter", scale: 1.25, wantScale: 1.25, wantWidth: 1601, wantHeight: 901},
		{name: "one and a half", scale: 1.5, wantScale: 1.5, wantWidth: 1921, wantHeight: 1081},
		{name: "two", scale: 2, wantScale: 2, wantWidth: 2561, wantHeight: 1441},
		{name: "three caps at two", scale: 3, wantScale: 2, wantWidth: 2561, wantHeight: 1441},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got := NextViewport(Viewport{}, 1280.25, 720.25, test.scale)
			if got.RenderScale != test.wantScale || got.RenderWidthPx != test.wantWidth || got.RenderHeightPx != test.wantHeight || got.ViewportRevision != 1 {
				t.Fatalf("viewport = %#v", got)
			}
			if !got.SupportsGameplay() {
				t.Fatal("valid minimum-size viewport disabled gameplay")
			}
		})
	}
}

func TestNextViewportScaleFallbackMinimizationAndRevision(t *testing.T) {
	for _, scale := range []float64{0, -1, math.NaN(), math.Inf(1), math.Inf(-1)} {
		got := NextViewport(Viewport{}, 1280, 720, scale)
		if got.RenderScale != 1 || got.RenderWidthPx != 1280 || got.RenderHeightPx != 720 {
			t.Fatalf("scale %v produced %#v", scale, got)
		}
	}

	initial := NextViewport(Viewport{}, 1280, 720, 2)
	if identical := NextViewport(initial, 1280, 720, 2); identical != initial {
		t.Fatalf("identical layout changed viewport: %#v -> %#v", initial, identical)
	}
	if minimized := NextViewport(initial, 0, 720, 2); minimized != initial {
		t.Fatalf("minimization changed viewport: %#v -> %#v", initial, minimized)
	}
	moved := NextViewport(initial, 1280, 720, 1.5)
	if moved.ViewportRevision != initial.ViewportRevision+1 || moved.RenderScale != 1.5 {
		t.Fatalf("monitor move = %#v", moved)
	}
	if fallback := NextViewport(Viewport{}, 0, 0, 2); fallback.RenderWidthPx != 1 || fallback.RenderHeightPx != 1 || fallback.ViewportRevision != 0 {
		t.Fatalf("pre-layout fallback = %#v", fallback)
	}
}

func TestViewportCoordinateRoundTripsAndMinimumGate(t *testing.T) {
	viewport := NextViewport(Viewport{}, 1280.25, 720.25, 1.25)
	for _, point := range [][2]float64{{0, 0}, {1, 1}, {377.125, 244.5}, {1280.25, 720.25}} {
		xPx, yPx := viewport.DIPToRender(point[0], point[1])
		xDIP, yDIP := viewport.RenderToDIP(xPx, yPx)
		if math.Abs(xDIP-point[0])*viewport.ScaleX > 1 || math.Abs(yDIP-point[1])*viewport.ScaleY > 1 {
			t.Fatalf("point %v round-tripped through (%v,%v) to (%v,%v)", point, xPx, yPx, xDIP, yDIP)
		}
	}
	if NextViewport(Viewport{}, MinViewportWidthDIP-1, MinViewportHeightDIP, 1).SupportsGameplay() {
		t.Fatal("one DIP below minimum width enabled gameplay")
	}
	if NextViewport(Viewport{}, MinViewportWidthDIP, MinViewportHeightDIP-1, 1).SupportsGameplay() {
		t.Fatal("one DIP below minimum height enabled gameplay")
	}
	if !NextViewport(Viewport{}, MinViewportWidthDIP, MinViewportHeightDIP, 1).SupportsGameplay() {
		t.Fatal("exact minimum disabled gameplay")
	}
	if x, y := (Viewport{}).RenderToDIP(10, 10); x != 0 || y != 0 {
		t.Fatalf("uninitialized inverse transform = (%v,%v)", x, y)
	}
}

func TestPresentationTransformAspectFitsAndRoundTrips(t *testing.T) {
	for _, dimensions := range [][2]int{{1280, 720}, {2560, 1440}, {1600, 1200}, {1200, 720}} {
		transform := FitPresentation(dimensions[0], dimensions[1])
		for _, point := range [][2]float64{{0, 0}, {640, 360}, {1279, 719}} {
			x, y := transform.LogicalToRender(point[0], point[1])
			logicalX, logicalY, inside := transform.RenderToLogical(x, y)
			if !inside || math.Abs(logicalX-point[0]) > 1e-9 || math.Abs(logicalY-point[1]) > 1e-9 {
				t.Fatalf("%v in %v round-tripped to (%v,%v,%v)", point, dimensions, logicalX, logicalY, inside)
			}
		}
	}
	pillarboxed := FitPresentation(1600, 1200)
	if pillarboxed.OffsetX != 0 || pillarboxed.OffsetY != 150 || pillarboxed.Scale != 1.25 {
		t.Fatalf("4:3 transform = %#v", pillarboxed)
	}
	if _, _, inside := pillarboxed.RenderToLogical(800, 10); inside {
		t.Fatal("letterbox point reported inside presentation")
	}
}
