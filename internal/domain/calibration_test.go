package domain

import (
	"go/ast"
	"go/parser"
	"go/token"
	"math"
	"path/filepath"
	"strings"
	"testing"
)

// The step 4c calibration assertions need the whole map over the whole
// campaign. Turn 0's East Africa savanna and woodland is pinned by
// TestTurnZeroEastAfricaHasSavannaAndWoodland; these cover the rest.

// southernmostTundraLatitude is the lowest northern-hemisphere latitude at
// which the published biome on a land tile is Glacial Tundra.
func southernmostTundraLatitude(t *testing.T, grid *Grid, turn int) float64 {
	t.Helper()
	southernmost, found := 0.0, false
	for id := range TileCount {
		tile, _ := grid.Tile(TileID(id))
		if !tile.Land || tile.Latitude <= 0 {
			continue
		}
		if biome, _ := grid.biomeAt(turn, TileID(id)); biome == GlacialTundra && (!found || tile.Latitude < southernmost) {
			southernmost, found = tile.Latitude, true
		}
	}
	if !found {
		t.Fatalf("turn %d has no northern Glacial Tundra", turn)
	}
	return southernmost
}

func TestTurn400PushesTheTundraBoundarySouth(t *testing.T) {
	grid := generatedGrid(t)
	start, end := southernmostTundraLatitude(t, grid, 0), southernmostTundraLatitude(t, grid, MaxCampaignTurn)
	if end >= start {
		t.Fatalf("tundra boundary at turn 400 = %.1f°N, not strictly south of turn 0's %.1f°N", end, start)
	}
}

// inSaharanArabianCorridor is the desert belt §6's moisture model dries: all of
// Arabia, and African land between 15°N and 35°N.
func inSaharanArabianCorridor(tile TileGeography) bool {
	if !tile.Land {
		return false
	}
	african := tile.Region == RestOfAfrica || tile.Region == EastAfrica
	return tile.Region == Arabia || (african && tile.Latitude >= 15 && tile.Latitude <= 35)
}

// The corridor must desertify across the campaign, and its habitable count
// must at some turn rise above the preceding minimum. Otherwise the long-term
// drying trend swamps the precession term entirely, and the cyclic climate
// the design promises would be invisible in the one region built to show it.
func TestSaharanArabianCorridorDesertifiesWithAPrecessionRebound(t *testing.T) {
	grid := generatedGrid(t)
	desert := [MaxCampaignTurn + 1]int{}
	rebound := -1
	lowest := TileCount + 1
	for turn := 0; turn <= MaxCampaignTurn; turn++ {
		habitat, _, err := BuildHabitat(grid, 0, turn)
		if err != nil {
			t.Fatal(err)
		}
		habitable := 0
		for id := range TileCount {
			tile, _ := grid.Tile(TileID(id))
			if !inSaharanArabianCorridor(tile) {
				continue
			}
			if habitat[id].Biome == SemiAridDesert {
				desert[turn]++
			}
			if habitat[id].BaselineK > 0 {
				habitable++
			}
		}
		if habitable > lowest && rebound < 0 {
			rebound = turn
		}
		lowest = min(lowest, habitable)
	}
	if desert[MaxCampaignTurn] <= desert[0] {
		t.Errorf("corridor desert tiles: turn 0 %d, turn 400 %d; want net expansion", desert[0], desert[MaxCampaignTurn])
	}
	if rebound < 0 {
		t.Error("corridor habitable count never rises above a preceding minimum; precession is swamped by the trend")
	}
}

func TestEpochThresholdTableRequiresStrictHeadroom(t *testing.T) {
	if err := ValidateEpochThresholds(EpochLowerThreshold, EpochUpperThreshold, EpochHysteresis); err != nil {
		t.Fatalf("configured epoch thresholds rejected: %v", err)
	}
	for name, table := range map[string][3]float64{
		// The step 4c example: the trip point equals the clamped maximum.
		"upper 0.98 band 0.02 has no entry headroom": {0.25, 0.98, 0.02},
		"lower exit at zero":                         {0.02, 0.96, 0.02},
		"overlapping bands":                          {0.50, 0.53, 0.02},
		"zero hysteresis":                            {0.25, 0.96, 0},
		// +Inf from its bit pattern: math.Inf is off the domain allowlist.
		"non-finite threshold": {0.25, math.Float64frombits(0x7ff0000000000000), 0.02},
	} {
		if err := ValidateEpochThresholds(table[0], table[1], table[2]); err == nil {
			t.Errorf("%s: accepted %v", name, table)
		}
	}
}

// ClimateEpoch is a presentation label (§7): no domain rule may branch on it.
// Only climate.go, which derives it, may name the type, its values, or the
// state field that carries it.
func TestNoDomainRuleReadsClimateEpoch(t *testing.T) {
	files, err := filepath.Glob("*.go")
	if err != nil {
		t.Fatal(err)
	}
	forbidden := map[string]bool{"ClimateEpoch": true, "ClimateEpochForTurn": true, "HumidOptimum": true, "AridTransition": true, "GlacialMaximum": true, "ClimateEpochCount": true}
	fileSet := token.NewFileSet()
	checked := 0
	for _, path := range files {
		if strings.HasSuffix(path, "_test.go") || path == "climate.go" {
			continue
		}
		file, err := parser.ParseFile(fileSet, path, nil, parser.SkipObjectResolution)
		if err != nil {
			t.Fatal(err)
		}
		checked++
		ast.Inspect(file, func(node ast.Node) bool {
			switch value := node.(type) {
			case *ast.Ident:
				if forbidden[value.Name] {
					t.Errorf("%s reads %s", fileSet.Position(value.Pos()), value.Name)
				}
			case *ast.SelectorExpr:
				if value.Sel.Name == "Epoch" {
					t.Errorf("%s reads .Epoch", fileSet.Position(value.Pos()))
				}
			}
			return true
		})
	}
	if checked == 0 {
		t.Fatal("no domain source files were checked")
	}
}
