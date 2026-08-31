package domain

import (
	"math"
	"testing"
)

func uniformTraits(value TraitValue) HeritableState {
	return HeritableState{value, value, value, value, value, value}
}

func TestSameSpeciesGeneFlowIsReciprocalAndPopulationWeighted(t *testing.T) {
	grid, _ := (WorldGenerator{}).Generate()
	bands := []Band{
		{ID: 1, Species: HomoSapiens, TileID: StartingTileIDs[0], Population: 100, Heritable: uniformTraits(0.2)},
		{ID: 2, Species: HomoSapiens, TileID: StartingTileIDs[0], Population: 100, Heritable: uniformTraits(0.8)},
	}
	applyKnowledgeAndGenetics(bands, grid, nil, nil, NewWorldRNG(1))
	mix := SameSpeciesGeneFlowRate / (1 + SameSpeciesGeneFlowRate)
	if got, want := float64(bands[0].Heritable[ColdAdaptation]), 0.2+float64(mix*0.6); math.Abs(got-want) > 1e-12 {
		t.Fatalf("recipient trait = %.15f, want %.15f", got, want)
	}
	if got, want := float64(bands[1].Heritable[ColdAdaptation]), 0.8-float64(mix*0.6); math.Abs(got-want) > 1e-12 {
		t.Fatalf("partner trait = %.15f, want %.15f", got, want)
	}
}

func TestActiveInterbreedingCreatesCrossSpeciesGeneFlow(t *testing.T) {
	grid, _ := (WorldGenerator{}).Generate()
	bands := []Band{
		{ID: 1, Species: HomoSapiens, TileID: StartingTileIDs[0], Population: 100, Heritable: uniformTraits(0.1), HasInterbreedTarget: true, InterbreedTarget: 2},
		{ID: 2, Species: ArchaicHominin, TileID: StartingTileIDs[0], Population: 100, Heritable: uniformTraits(0.9)},
	}
	applyKnowledgeAndGenetics(bands, grid, nil, nil, NewWorldRNG(2))
	mix := InterbreedGeneFlowRate / (1 + InterbreedGeneFlowRate)
	if got, want := float64(bands[0].Heritable[ColdAdaptation]), 0.1+float64(mix*0.8); math.Abs(got-want) > 1e-12 {
		t.Fatalf("introgressed trait = %.15f, want %.15f", got, want)
	}
}

func TestKnowledgeDiffusionDoesNotRelayInSameTurn(t *testing.T) {
	grid, _ := (WorldGenerator{}).Generate()
	var line [3]TileID
	found := false
	for id := range TileCount {
		x, y, _ := TileXY(TileID(id))
		left, leftErr := TileIDAt(x-1, y)
		right, rightErr := TileIDAt(x+1, y)
		if leftErr != nil || rightErr != nil {
			continue
		}
		leftTile, _ := grid.Tile(left)
		centerTile, _ := grid.Tile(TileID(id))
		rightTile, _ := grid.Tile(right)
		if leftTile.Land && centerTile.Land && rightTile.Land {
			line, found = [3]TileID{left, TileID(id), right}, true
			break
		}
	}
	if !found {
		t.Fatal("no three-tile land line")
	}
	bands := []Band{
		{ID: 1, Species: HomoSapiens, TileID: line[0], Population: 100, Heritable: uniformTraits(0.5), Technology: TechnologyState{Acquired: 1 << Firecraft}},
		{ID: 2, Species: HomoSapiens, TileID: line[1], Population: 100, Heritable: uniformTraits(0.5)},
		{ID: 3, Species: HomoSapiens, TileID: line[2], Population: 100, Heritable: uniformTraits(0.5)},
	}
	bands[0].Technology.Progress[Firecraft] = ResearchCost[Firecraft]
	applyKnowledgeAndGenetics(bands, grid, nil, nil, NewWorldRNG(3))
	if got := bands[1].Technology.Progress[Firecraft]; got != float64(DiffusionRate*ResearchCost[Firecraft]) {
		t.Fatalf("direct diffusion = %v", got)
	}
	if got := bands[2].Technology.Progress[Firecraft]; got != 0 {
		t.Fatalf("same-turn relay = %v", got)
	}
}

func TestKnowledgeDiffusionCannotUsePrerequisiteLearnedInSameTurn(t *testing.T) {
	grid, _ := (WorldGenerator{}).Generate()
	tile := StartingTileIDs[0]
	sourceTech := TechnologyState{Acquired: 1<<HaftedTools | 1<<TailoredClothing}
	sourceTech.Progress[HaftedTools] = ResearchCost[HaftedTools]
	sourceTech.Progress[TailoredClothing] = ResearchCost[TailoredClothing]
	recipientTech := TechnologyState{}
	recipientTech.Progress[HaftedTools] = float64(ResearchCost[HaftedTools] * (1 - DiffusionRate))
	bands := []Band{
		{ID: 1, Species: HomoSapiens, TileID: tile, Population: 100, Heritable: uniformTraits(0.5), Technology: sourceTech},
		{ID: 2, Species: HomoSapiens, TileID: tile, Population: 100, Heritable: uniformTraits(0.5), Technology: recipientTech},
	}

	applyKnowledgeAndGenetics(bands, grid, nil, nil, NewWorldRNG(4))

	if !bands[1].Technology.Has(HaftedTools) {
		t.Fatal("eligible prerequisite did not complete")
	}
	if got := bands[1].Technology.Progress[TailoredClothing]; got != 0 {
		t.Fatalf("dependent gained %v research from a prerequisite learned in the same turn", got)
	}
}
