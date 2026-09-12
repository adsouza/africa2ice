package domain

import (
	"testing"
)

func TestWaterBodyNamesFollowAuthoredGeography(t *testing.T) {
	grid, err := (WorldGenerator{}).Generate()
	if err != nil {
		t.Fatal(err)
	}
	for _, test := range []struct {
		lon, lat int
		want     string
	}{
		{38, 20, "Red Sea"},
		{35, 43, "Black Sea"},
		{50, 42, "Caspian Sea"},
		{-15, -30, "Open ocean"},
		{33, -1, ""},
	} {
		// Round the authored integer-degree coordinates to the nearest cell
		// with integer arithmetic, as required by the domain's numeric rules.
		x := ((test.lon+20)*95 + 110) / 220
		y := ((72-test.lat)*63 + 60) / 120
		id, _ := TileIDAt(x, y)
		if got := grid.WaterBodyName(id); got != test.want {
			t.Errorf("(%d, %d): got %q, want %q", test.lon, test.lat, got, test.want)
		}
	}
	if grid.WaterBodyName(InvalidTileID) != "" {
		t.Fatal("invalid tile has a water name")
	}
}
