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

func TestCompletedResearchAutomaticallyContinues(t *testing.T) {
	grid, err := (WorldGenerator{}).Generate()
	if err != nil {
		t.Fatal(err)
	}
	for _, tc := range []struct {
		name      string
		target    Technology
		learned   []Technology
		want      Technology
		finished  bool
		diffusion bool
		archaic   bool
	}{
		{name: "next in sequence", target: Firecraft, want: HaftedTools},
		{name: "skip learned", target: Firecraft, learned: []Technology{HaftedTools}, want: PlantKnowledge},
		{name: "skip locked and wrap", target: PlantKnowledge, want: Firecraft},
		{name: "newly unlocked", target: HaftedTools, learned: []Technology{PlantKnowledge}, want: TailoredClothing},
		{name: "diffusion completion", target: Firecraft, want: HaftedTools, diffusion: true},
		{name: "skip simultaneous diffusion completion", target: Firecraft, want: PlantKnowledge, diffusion: true, learned: nil},
		{name: "all learned", target: CoastalNavigation, learned: []Technology{Firecraft, HaftedTools, PlantKnowledge, TailoredClothing, CordageAndNets, Campcraft, MedicinalKnowledge, Trapping}, finished: true},
		{name: "archaic keeps own policy", target: Firecraft, finished: true, archaic: true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			state := TechnologyState{Target: tc.target, HasTarget: true}
			for _, tech := range tc.learned {
				state.Acquired |= 1 << tech
				state.Progress[tech] = ResearchCost[tech]
			}
			state.Progress[tc.target] = ResearchCost[tc.target] - 1
			species := HomoSapiens
			if tc.archaic {
				species = ArchaicHominin
			}
			bands := []Band{{ID: 1, Species: species, TileID: StartingTileIDs[0], Population: 100, Technology: state}}
			gains := map[BandID]float64{1: 1}
			if tc.diffusion {
				gains = nil
				source := TechnologyState{}
				source.AdvanceResearch(tc.target, ResearchCost[tc.target])
				if tc.name == "skip simultaneous diffusion completion" {
					source.AdvanceResearch(HaftedTools, ResearchCost[HaftedTools])
					bands[0].Technology.Progress[HaftedTools] = ResearchCost[HaftedTools] - 1
				}
				bands = append(bands, Band{ID: 2, Species: species, TileID: StartingTileIDs[0], Population: 100, Technology: source})
			}
			applyKnowledgeAndGenetics(bands, grid, gains, nil, NewWorldRNG(1))
			got := bands[0].Technology
			if !got.Has(tc.target) || got.HasTarget == tc.finished || (!tc.finished && got.Target != tc.want) {
				t.Fatalf("unexpected research state: %+v", got)
			}
			if !tc.finished && got.Progress[tc.want] != 0 {
				t.Fatalf("new target received old target's production: %v", got.Progress[tc.want])
			}
			if err := got.Validate(); err != nil {
				t.Fatal(err)
			}
		})
	}
}

func TestAutomaticResearchPreservesProgressAndManualChoice(t *testing.T) {
	grid, err := (WorldGenerator{}).Generate()
	if err != nil {
		t.Fatal(err)
	}
	bands := []Band{{ID: 1, Species: HomoSapiens, TileID: StartingTileIDs[0], Population: 100}}
	state := &bands[0].Technology
	if err := state.Select(Firecraft); err != nil {
		t.Fatal(err)
	}
	state.Progress[Firecraft] = ResearchCost[Firecraft] - 1
	state.Progress[HaftedTools] = 12
	applyKnowledgeAndGenetics(bands, grid, map[BandID]float64{1: 1}, nil, NewWorldRNG(1))
	if state.Target != HaftedTools || state.Progress[HaftedTools] != 12 {
		t.Fatalf("next target lost its saved progress: %+v", state)
	}
	if err := state.Select(PlantKnowledge); err != nil {
		t.Fatal(err)
	}
	applyKnowledgeAndGenetics(bands, grid, map[BandID]float64{1: 1}, nil, NewWorldRNG(1))
	if state.Target != PlantKnowledge || state.Progress[PlantKnowledge] != 1 || state.Progress[HaftedTools] != 12 {
		t.Fatalf("manual override was not preserved: %+v", state)
	}
}
