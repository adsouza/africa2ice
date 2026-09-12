package application

import (
	"math"
	"reflect"
	"testing"

	"github.com/adsouza/africa2ice/internal/domain"
	"github.com/adsouza/africa2ice/pkg/gameapi"
)

func TestLakeCoverageAndChronology(t *testing.T) {
	service, err := NewGameService(1)
	if err != nil {
		t.Fatal(err)
	}
	base, err := service.Snapshot()
	if err != nil {
		t.Fatal(err)
	}
	for _, year := range []int{80000, 70001, 70000, 20000} {
		frame := &gameapi.Frame{YearBP: year, Tiles: append([]gameapi.Tile(nil), base.Tiles...)}
		for i := range frame.Tiles {
			frame.Tiles[i].Explored = true
			frame.Tiles[i].Lakes = nil
			frame.Tiles[i].NearbyLake = ""
		}
		before := append([]gameapi.Tile(nil), frame.Tiles...)
		projectLakes(frame)
		counts := map[string]int{}
		for i, tile := range frame.Tiles {
			for _, lake := range tile.Lakes {
				counts[lake.Name]++
				if tile.NearbyLake != lake.Name {
					t.Fatalf("missing inspector name for %s", lake.Name)
				}
				if !tile.Land {
					t.Fatalf("lake patch on full water tile: %s", lake.Name)
				}
				for _, p := range lake.Points {
					if math.IsNaN(p.X) || math.IsNaN(p.Y) || p.X < 0 || p.X > 1 || p.Y < 0 || p.Y > 1 {
						t.Fatalf("fragment escapes tile: %+v", p)
					}
				}
			}
			// Projection cannot change terrain, resource capacity or movement costs.
			tile.Lakes = nil
			tile.NearbyLake = ""
			if !reflect.DeepEqual(tile, before[i]) {
				t.Fatalf("lake projection changed tile rules at %d", i)
			}
		}
		for _, name := range []string{"Lake Baikal", "Lake Tanganyika", "Lake Malawi / Nyasa", "Lake Victoria", "Lake Turkana", "Issyk-Kul", "Lake Qinghai", "Lake Van"} {
			if counts[name] == 0 {
				t.Errorf("%s missing at %d BP", name, year)
			}
		}
		if (counts["Lake Lisan"] > 0) != (year <= 70000) {
			t.Errorf("Lisan chronology at %d: %v", year, counts)
		}
	}
}

func TestLakeProjectionHidesOtherShoreAndOwnsItsPoints(t *testing.T) {
	frame := &gameapi.Frame{YearBP: 80000, Tiles: make([]gameapi.Tile, domain.TileCount)}
	// Reveal just one fragment of the long Baikal shoreline.
	id := lakePatches[0].tile
	frame.Tiles[id].Land = true
	frame.Tiles[id].Explored = true
	projectLakes(frame)
	if len(frame.Tiles[id].Lakes) == 0 {
		t.Fatal("revealed shore missing")
	}
	for i, tile := range frame.Tiles {
		if i != int(id) && (len(tile.Lakes) != 0 || tile.NearbyLake != "") {
			t.Fatal("lake leaked into fog")
		}
	}
	old := frame.Tiles[id].Lakes[0].Points[0]
	frame.Tiles[id].Lakes[0].Points[0].X = 999
	frame.Tiles[id].Lakes = nil
	frame.Tiles[id].NearbyLake = ""
	projectLakes(frame)
	if frame.Tiles[id].Lakes[0].Points[0] != old {
		t.Fatal("published frame aliases the lake catalog")
	}
}

func TestLakeClippingPreservesAreaAcrossCells(t *testing.T) {
	// This square straddles four tiles. Clipping must conserve its area while
	// leaving every fragment within the tile that controls its fog visibility.
	ring := []gameapi.LakePoint{{X: 0.5, Y: 0.5}, {X: 1.5, Y: 0.5}, {X: 1.5, Y: 1.5}, {X: 0.5, Y: 1.5}}
	total := 0.0
	for y := 0; y < 2; y++ {
		for x := 0; x < 2; x++ {
			clipped := clipLakeRing(ring, float64(x), float64(y))
			area := 0.0
			for i, p := range clipped {
				q := clipped[(i+1)%len(clipped)]
				area += p.X*q.Y - q.X*p.Y
			}
			total += math.Abs(area) / 2
		}
	}
	if math.Abs(total-1) > 1e-12 {
		t.Fatalf("clipped area = %g, want 1", total)
	}
}
