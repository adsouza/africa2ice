package domain

import "testing"

func TestCampanianMaskCardinalityAndActivation(t *testing.T) {
	counts := [4]int{}
	for id := range TileCount {
		counts[CampanianZone(TileID(id))]++
	}
	if counts[MacroDirect] != 1 || counts[MacroProximal] != 16 || counts[MacroWide] != 72 {
		t.Fatalf("zone counts = %#v", counts)
	}
	activeTurn := -1
	for turn := 1; turn <= MaxCampaignTurn; turn++ {
		if MacroEpisodeActive(turn) {
			if activeTurn != -1 {
				t.Fatalf("episode active more than once: %d and %d", activeTurn, turn)
			}
			activeTurn = turn
		}
	}
	if activeTurn < 1 || !MacroEpisodeWarned(activeTurn-1) {
		t.Fatalf("active/warning turn = %d", activeTurn)
	}
}

func TestMacroImpactCannotSterilizeDirectZone(t *testing.T) {
	grid, _ := (WorldGenerator{}).Generate()
	activeTurn := 1
	for !MacroEpisodeActive(activeTurn) {
		activeTurn++
	}
	for id := range TileCount {
		if CampanianZone(TileID(id)) != MacroDirect {
			continue
		}
		tile, _ := grid.Tile(TileID(id))
		impact := MacroImpactAt(tile, activeTurn)
		if impact.LossFraction >= 1 || impact.HabitatFactor <= 0 || impact.FloraFactor <= 0 || impact.FaunaFactor <= 0 || impact.WaterFactor <= 0 {
			t.Fatalf("direct impact = %#v", impact)
		}
		return
	}
	t.Fatal("direct zone missing")
}
