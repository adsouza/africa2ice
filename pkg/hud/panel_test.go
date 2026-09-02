package hud

import (
	"testing"

	"github.com/adsouza/africa2ice/pkg/gameapi"
	"github.com/adsouza/africa2ice/pkg/render"
	"github.com/adsouza/africa2ice/pkg/ui"
	"github.com/hajimehoshi/ebiten/v2"
)

func testFrame(bands int) *gameapi.Frame {
	frame := &gameapi.Frame{CampaignResult: gameapi.Ongoing, YearBP: 76_400, Turn: 12,
		Tiles: []gameapi.Tile{
			{ID: 0, X: 5, Y: 5, Land: true, Explored: true, Biome: gameapi.RiverineWoodland, BaselineK: 150, EcologicalK: 145, FloraStock: 212, FloraCap: 810, WaterStock: 315, WaterCap: 500, NaturalShelter: 0.5, MovementCost: 1.2},
			{ID: 1, X: 6, Y: 4, Land: true, Explored: true, Biome: gameapi.Savanna, BaselineK: 150, EcologicalK: 150, FloraStock: 640, FloraCap: 810, WaterStock: 480, WaterCap: 500, NaturalShelter: 0.4, MovementCost: 1.2},
		}}
	for index := 0; index < bands; index++ {
		band := gameapi.Band{ID: gameapi.BandID(index + 1), Species: gameapi.HomoSapiens, Population: 60, Health: 1, TileID: 0,
			AllocationBP:        [gameapi.AssignmentCount]uint16{3_500, 3_000, 1_500, 500, 1_500},
			MigrationCandidates: []gameapi.MigrationCandidate{{TileID: 1}}}
		band.ResearchOptions[gameapi.Firecraft] = gameapi.ResearchOption{Available: true, Cost: 80}
		frame.Bands = append(frame.Bands, band)
	}
	return frame
}

func testState(frame *gameapi.Frame, scale float64) State {
	viewport := render.NextViewport(render.Viewport{}, 1280*scale, 720*scale, 1)
	return State{
		Frame: frame, SelectedBand: 1, OpenRow: ui.RowMove, NotesMode: NotesCompact,
		Guide:    ui.NewGuideState(true),
		EndTurn:  ui.EndTurnGateFor(frame, false, false, false),
		Viewport: viewport, Transform: render.FitPresentation(viewport.RenderWidthPx, viewport.RenderHeightPx),
		Workforce: WorkforceDraft{Visible: true, Population: 60, AllocationBP: frame.Bands[0].AllocationBP, Valid: true},
	}
}

func TestPanelBuildsAtEveryScaleWithoutIntents(t *testing.T) {
	for _, scale := range []float64{1, 1.5, 2} {
		for _, bands := range []int{1, 9, 17, 256} {
			panel := New()
			state := testState(testFrame(bands), scale)
			if intents := panel.Update(state); len(intents) != 0 {
				t.Fatalf("scale %.1f bands %d: unsolicited intents %v", scale, bands, intents)
			}
			screen := ebiten.NewImage(int(1280*scale), int(720*scale))
			panel.Draw(screen)
			screen.Deallocate()
		}
	}
}

func TestPanelRebuildsOnlyWhenStateChanges(t *testing.T) {
	panel := New()
	state := testState(testFrame(2), 1)
	panel.Update(state)
	before := panel.root
	panel.Update(state)
	if panel.root != before || !panel.built {
		t.Fatal("panel root was replaced without a state change")
	}
	first := panel.root.Children()
	panel.Update(state)
	if len(panel.root.Children()) != len(first) || panel.root.Children()[0] != first[0] {
		t.Fatal("unchanged state rebuilt the widget tree")
	}
	state.OpenRow = ui.RowResearch
	panel.Update(state)
	if panel.root.Children()[0] == first[0] {
		t.Fatal("changed state did not rebuild the widget tree")
	}
}

func TestChipsCarryProgressColorMarkerAndSelection(t *testing.T) {
	frame := testFrame(3)
	frame.Bands[1].HasQueuedMigration = true
	frame.Bands[2].Health = 0.4
	panel := New()
	panel.Update(testState(frame, 1))
	if len(panel.handles.chips) != 3 {
		t.Fatalf("chip count = %d", len(panel.handles.chips))
	}
	if got := panel.handles.chips[1].Text().Label; got != "B1" {
		t.Fatalf("selected chip label = %q", got)
	}
	if got := panel.handles.chips[3].Text().Label; got != "!! B3" {
		t.Fatalf("suffering chip label = %q, want the !! prefix", got)
	}
	panel.handles.chips[2].Click()
	intents := panel.Update(testState(frame, 1))
	if len(intents) != 1 || intents[0].Kind != IntentSelectBand || intents[0].Band != 2 {
		t.Fatalf("chip click intents = %+v", intents)
	}
}

func TestChipRowOverflowsIntoAPlusChip(t *testing.T) {
	panel := New()
	state := testState(testFrame(12), 1)
	panel.Update(state)
	if len(panel.handles.chips) != 7 {
		t.Fatalf("visible chips = %d, want 7 alongside the +N chip", len(panel.handles.chips))
	}
	if panel.handles.more == nil || panel.handles.more.Text().Label != "+5" {
		t.Fatal("overflow chip missing or mislabelled")
	}
	panel.handles.more.Click()
	if intents := panel.Update(state); len(intents) != 1 || intents[0].Kind != IntentToggleBandList {
		t.Fatalf("+N click intents = %+v", intents)
	}
	state.BandListOpen = true
	panel.Update(state)
	if panel.handles.bandList == nil {
		t.Fatal("band list window not shown when BandListOpen")
	}
}

func TestDetailsDisclosureListsTraitsAsFocusButtons(t *testing.T) {
	panel := New()
	state := testState(testFrame(1), 1)
	panel.Update(state)
	if panel.handles.details == nil || len(panel.handles.traits) != 0 {
		t.Fatal("collapsed details should have a toggle and no trait cells")
	}
	panel.handles.details.Click()
	if intents := panel.Update(state); len(intents) != 1 || intents[0].Kind != IntentToggleDetails {
		t.Fatalf("details click = %+v", intents)
	}
	state.DetailsOpen = true
	panel.Update(state)
	if len(panel.handles.traits) != int(gameapi.HeritableTraitCount) {
		t.Fatalf("trait cells = %d", len(panel.handles.traits))
	}
	panel.handles.traits[gameapi.PigmentationLevel].Click()
	if intents := panel.Update(state); len(intents) != 1 || intents[0].Kind != IntentFocusTrait || intents[0].Trait != gameapi.PigmentationLevel {
		t.Fatalf("trait click = %+v", intents)
	}
}

func TestChipColorsKeepMoveStatusBorderWhenSelected(t *testing.T) {
	if border, fill, borderPx := chipColors(true, true); border != colorGreen || fill != colorButtonHover || borderPx != 2 {
		t.Fatalf("chipColors(done, selected) = %v, %v, %v", border, fill, borderPx)
	}
	if border, fill, borderPx := chipColors(false, false); border != colorGold || fill != colorButtonIdle || borderPx != 1 {
		t.Fatalf("chipColors(open, unselected) = %v, %v, %v", border, fill, borderPx)
	}
	if border, _, _ := chipColors(true, false); border != colorGreen {
		t.Fatalf("chipColors(done, unselected) border = %v, want colorGreen", border)
	}
}
