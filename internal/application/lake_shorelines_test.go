package application

import (
	"math"
	"testing"

	"github.com/adsouza/africa2ice/pkg/gameapi"
)

func TestProjectedShorelinesChangeAtStagesAndRemainStableBetween(t *testing.T) {
	service, err := NewGameService(1)
	if err != nil {
		t.Fatal(err)
	}
	base, err := service.Snapshot()
	if err != nil {
		t.Fatal(err)
	}
	measure := func(year int, name string) (float64, float64, float64) {
		frame := &gameapi.Frame{YearBP: year, Tiles: append([]gameapi.Tile(nil), base.Tiles...)}
		for i := range frame.Tiles {
			frame.Tiles[i].Explored = true
			frame.Tiles[i].Lakes = nil
			frame.Tiles[i].NearbyLake = ""
		}
		projectLakes(frame)
		area, minY, maxY := 0.0, math.Inf(1), math.Inf(-1)
		for _, tile := range frame.Tiles {
			for _, shape := range tile.Lakes {
				if shape.Name != name {
					continue
				}
				if shape.Stage != gameapi.LakeStageAt(name, year) {
					t.Fatal("stale stage projected")
				}
				sum := 0.0
				for i, p := range shape.Points {
					q := shape.Points[(i+1)%len(shape.Points)]
					sum += p.X*q.Y - q.X*p.Y
					minY = min(minY, float64(tile.Y)+p.Y)
					maxY = max(maxY, float64(tile.Y)+p.Y)
				}
				area += math.Abs(sum) / 2
			}
		}
		return area, minY, maxY
	}
	for _, name := range []string{"Lake Malawi / Nyasa", "Lake Lisan"} {
		years := []int{80000, 60000, 35000}
		if name == "Lake Lisan" {
			years = []int{70000, 27000, 23000}
		}
		low, lowN, lowS := measure(years[0], name)
		high, highN, highS := measure(years[1], name)
		late, _, _ := measure(years[2], name)
		t.Logf("%s visible areas %.3f -> %.3f -> %.3f tiles; N/S shifts %.2f/%.2f tiles", name, low, high, late, lowN-highN, highS-lowS)
		if high <= low*1.2 || late >= high*0.9 || late <= 0 {
			t.Fatalf("shoreline states not visibly distinct: %s", name)
		}
		for _, year := range years {
			a, _, _ := measure(year, name)
			b, _, _ := measure(year-1, name)
			if a != b {
				t.Fatal("shoreline flickers within stage")
			}
		}
	}
}
