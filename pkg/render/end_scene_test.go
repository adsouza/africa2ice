package render

import "testing"

// TestEndScenePanelStaysInsideTheMapArea covers Wave H item H1: the terminal
// panel used to run 240-1040 DIP, spilling under the HUD column that starts
// at x 908. It must instead sit fully inside the map area (mapOriginX to
// mapOriginX+mapAreaWidth) and stay centred on that area's own centre, so a
// future map-geometry change is what moves it, not a second hand-tuned
// constant.
func TestEndScenePanelStaysInsideTheMapArea(t *testing.T) {
	if EndSceneX < mapOriginX {
		t.Fatalf("panel left edge %v is left of the map area's origin %v", EndSceneX, mapOriginX)
	}
	if EndSceneX+EndSceneWidth > mapOriginX+mapAreaWidth {
		t.Fatalf("panel right edge %v spills past the map area's right edge %v", EndSceneX+EndSceneWidth, mapOriginX+mapAreaWidth)
	}
	panelCentre := EndSceneX + EndSceneWidth/2
	mapCentre := float64(mapOriginX) + float64(mapAreaWidth)/2
	if diff := panelCentre - mapCentre; diff > 1 || diff < -1 {
		t.Fatalf("panel centre %v is not within a pixel of the map area's centre %v", panelCentre, mapCentre)
	}
}
