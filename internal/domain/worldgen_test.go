package domain

import (
	"math"
	"testing"
)

func nearestTile(t *testing.T, point GeoPoint) TileID {
	t.Helper()
	projected, err := ProjectGeo(point)
	if err != nil {
		t.Fatal(err)
	}
	best := InvalidTileID
	bestDistance := math.MaxFloat64
	for id := range TileCount {
		x, y, _ := TileXY(TileID(id))
		dx, dy := float64(x)-projected.X, float64(y)-projected.Y
		distance := dx*dx + dy*dy
		if distance < bestDistance {
			best, bestDistance = TileID(id), distance
		}
	}
	return best
}

func TestWorldGeneratorLandmarksAndBounds(t *testing.T) {
	grid, err := (WorldGenerator{}).Generate()
	if err != nil {
		t.Fatal(err)
	}
	landmarks := []GeoPoint{g(35, 5), g(44, 20), g(80, 20), g(145, -25), g(195, 62)}
	for _, point := range landmarks {
		tile, _ := grid.Tile(nearestTile(t, point))
		if !tile.Land {
			t.Errorf("landmark %#v resolved to water", point)
		}
	}
	Atlantic, _ := grid.Tile(nearestTile(t, g(-15, 0)))
	if Atlantic.Land {
		t.Fatal("mid-Atlantic resolved to land")
	}
	for _, tile := range grid.Tiles() {
		if tile.ElevationKm < 0 || tile.ElevationKm > 3 || tile.BaseMoisture < 0 || tile.BaseMoisture > 1 || tile.NaturalShelter < 0 || tile.NaturalShelter > 1 {
			t.Fatalf("tile %d has out-of-range geography: %#v", tile.ID, tile)
		}
		if !tile.Land && (tile.ElevationKm != 0 || tile.NaturalShelter != 0 || tile.BaseMoisture != 0) {
			t.Fatalf("water tile %d has land geography", tile.ID)
		}
	}
}

func TestWorldGenerationDeterministic(t *testing.T) {
	a, err := (WorldGenerator{}).Generate()
	if err != nil {
		t.Fatal(err)
	}
	b, err := (WorldGenerator{}).Generate()
	if err != nil {
		t.Fatal(err)
	}
	if a.Tiles() != b.Tiles() {
		t.Fatal("geography generation is not deterministic")
	}
}

func TestProjectionRoundTripAtCorners(t *testing.T) {
	for _, coordinate := range [][2]int{{0, 0}, {95, 0}, {0, 63}, {95, 63}, {24, 35}} {
		point, err := GeoAt(coordinate[0], coordinate[1])
		if err != nil {
			t.Fatal(err)
		}
		projected, err := ProjectGeo(point)
		if err != nil {
			t.Fatal(err)
		}
		if math.Abs(projected.X-float64(coordinate[0])) > 1e-12 || math.Abs(projected.Y-float64(coordinate[1])) > 1e-12 {
			t.Fatalf("round trip %#v -> %#v", coordinate, projected)
		}
	}
}
