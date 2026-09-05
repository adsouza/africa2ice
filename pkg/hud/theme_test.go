package hud

import "testing"

// TestNineSlicesAreMemoizedPerScale covers the rebuild cost of the chrome:
// every builder asks the theme for the same handful of backgrounds, so equal
// arguments must hand back the one image rather than allocating a fresh
// nine-slice per widget per rebuild. A scale change invalidates them all,
// exactly like the font faces.
func TestNineSlicesAreMemoizedPerScale(t *testing.T) {
	theme := newTheme()
	first, second := theme.solid(colorPanel), theme.solid(colorPanel)
	if first != second {
		t.Fatal("solid allocated a second nine-slice for the same color")
	}
	if theme.solid(colorRow) == first {
		t.Fatal("solid returned the same nine-slice for two different colors")
	}
	edge, edgeAgain := theme.bordered(colorPanel, colorGoldDeep, 2), theme.bordered(colorPanel, colorGoldDeep, 2)
	if edge != edgeAgain {
		t.Fatal("bordered allocated a second nine-slice for the same arguments")
	}
	if theme.bordered(colorPanel, colorGoldDeep, 1) == edge {
		t.Fatal("bordered ignored the border width when caching")
	}
	if theme.bordered(colorPanel, colorGold, 2) == edge {
		t.Fatal("bordered ignored the border color when caching")
	}

	theme.setScale(2)
	if len(theme.solids) != 0 || len(theme.borders) != 0 {
		t.Fatalf("a scale change left %d solid and %d bordered entries cached", len(theme.solids), len(theme.borders))
	}
	// Border widths arrive in render pixels, so the same DIP border is a
	// different image after a scale change; ebitenui interns plain color
	// nine-slices globally, so only the bordered pointer is observably new.
	if theme.bordered(colorPanel, colorGoldDeep, 2) == edge {
		t.Fatal("a scale change reused the pre-scale bordered nine-slice")
	}
}
