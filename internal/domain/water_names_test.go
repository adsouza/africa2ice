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
		{49, 13, "Gulf of Aden"},
		{35, 43, "Black Sea"},
		{50, 42, "Caspian Sea"},
		{-15, -30, "Atlantic Ocean"},
		{15, -40, "Atlantic Ocean"},
		{25, -40, "Indian Ocean"},
		{70, -20, "Indian Ocean"},
		{110, -20, "Indian Ocean"},
		{140, -48, "Indian Ocean"},
		{150, -48, "Pacific Ocean"},
		{170, 0, "Pacific Ocean"},
		{195, 40, "Pacific Ocean"},
		{80, 72, "Arctic Ocean"},
		{-15, 68, "Atlantic Ocean"},
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
