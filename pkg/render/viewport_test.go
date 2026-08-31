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
		{name: "one", scale: 1, wantScale: 1, wantWidth: 1001, wantHeight: 601},
		{name: "one and a quarter", scale: 1.25, wantScale: 1.25, wantWidth: 1251, wantHeight: 751},
		{name: "one and a half", scale: 1.5, wantScale: 1.5, wantWidth: 1501, wantHeight: 901},
		{name: "two", scale: 2, wantScale: 2, wantWidth: 2001, wantHeight: 1201},
		{name: "three caps at two", scale: 3, wantScale: 2, wantWidth: 2001, wantHeight: 1201},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got := NextViewport(Viewport{}, 1000.25, 600.25, test.scale)
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
	viewport := NextViewport(Viewport{}, 1000.25, 600.25, 1.25)
	for _, point := range [][2]float64{{0, 0}, {1, 1}, {377.125, 244.5}, {1000.25, 600.25}} {
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
	if x, y := (Viewport{}).RenderToDIP(10, 10); x != 0 || y != 0 {
		t.Fatalf("uninitialized inverse transform = (%v,%v)", x, y)
	}
}
