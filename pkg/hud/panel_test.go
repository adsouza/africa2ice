package hud

import (
	"fmt"
	"image"
	"strings"
	"testing"
	"time"

	"github.com/adsouza/africa2ice/pkg/gameapi"
	"github.com/adsouza/africa2ice/pkg/render"
	"github.com/adsouza/africa2ice/pkg/ui"
	"github.com/ebitenui/ebitenui/input"
	"github.com/ebitenui/ebitenui/widget"
	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/text/v2"
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

// testFrameWithIDs is testFrame's tile setup with caller-chosen band IDs, so
// a test can exercise IDs that are not simply 1..N (three-digit IDs, for
// instance).
func testFrameWithIDs(ids []gameapi.BandID) *gameapi.Frame {
	frame := &gameapi.Frame{CampaignResult: gameapi.Ongoing, YearBP: 76_400, Turn: 12,
		Tiles: []gameapi.Tile{
			{ID: 0, X: 5, Y: 5, Land: true, Explored: true, Biome: gameapi.RiverineWoodland, BaselineK: 150, EcologicalK: 145, FloraStock: 212, FloraCap: 810, WaterStock: 315, WaterCap: 500, NaturalShelter: 0.5, MovementCost: 1.2},
			{ID: 1, X: 6, Y: 4, Land: true, Explored: true, Biome: gameapi.Savanna, BaselineK: 150, EcologicalK: 150, FloraStock: 640, FloraCap: 810, WaterStock: 480, WaterCap: 500, NaturalShelter: 0.4, MovementCost: 1.2},
		}}
	for _, id := range ids {
		band := gameapi.Band{ID: id, Species: gameapi.HomoSapiens, Population: 60, Health: 1, TileID: 0,
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

// TestHeaderRegionFitsItsContent covers D3: panelHeaderH used to be a fixed
// constant sized for the case with a macro warning line present, which left
// roughly 17 DIP of dead space between the population line and the chips
// when there was no warning. The header region now sizes itself from its
// own measured content instead of a constant tuned for the busier case.
func TestHeaderRegionFitsItsContent(t *testing.T) {
	frame := testFrame(1)
	panel := New()
	state := testState(frame, 1)
	panel.Update(state)
	screen := ebiten.NewImage(1280, 720)
	panel.Draw(screen)
	screen.Deallocate()

	headerBottom := panel.handles.headerContent.GetWidget().Rect.Max.Y
	chipTop := panel.handles.chips[uint32(state.SelectedBand)].GetWidget().Rect.Min.Y
	if gap := chipTop - headerBottom; gap < 0 || gap >= panel.theme.px(16) {
		t.Fatalf("header-to-chip gap = %d render px, want [0, %d) now that the header fits its content", gap, panel.theme.px(16))
	}

	warned := testFrame(1)
	warned.MacroEpisodes = []gameapi.MacroEpisodeSummary{{Episode: gameapi.CampanianIgnimbrite, Warned: true}}
	warnedPanel := New()
	warnedState := testState(warned, 1)
	warnedPanel.Update(warnedState)
	warnedScreen := ebiten.NewImage(1280, 720)
	warnedPanel.Draw(warnedScreen)
	warnedScreen.Deallocate()

	if warnedPanel.handles.macroWarning == nil {
		t.Fatal("macro warning line missing from the header")
	}
	warningBottom := warnedPanel.handles.macroWarning.GetWidget().Rect.Max.Y
	scrollTop := warnedPanel.handles.panelMiddle.GetWidget().Rect.Min.Y
	if warningBottom > scrollTop {
		t.Fatalf("macro warning bottom %d render px is below the scroll container's top %d; it no longer fits inside the header region", warningBottom, scrollTop)
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

// TestChipRowOverflowsIntoAPlusChip covers the +N chip's wiring: past two
// full rows of chips, a +N cell appears, clicking it emits
// IntentToggleBandList, and BandListOpen shows the modal window. The row 1
// pinned / row 2 windowed split itself (chipRows) has its own exhaustive
// table test, TestChipRowsPinUrgentBandsAndWindowTheRest, so this test
// derives the expected chip count and +N label from the same production
// helpers (chipColumns, chipRows) rather than a hardcoded column count —
// the widest label, and therefore the column count, shifts as band IDs
// gain digits.
func TestChipRowOverflowsIntoAPlusChip(t *testing.T) {
	panel := New()
	frame := testFrame(20)
	state := testState(frame, 1)
	panel.Update(state)

	ordered := ui.SapiensBandIDsByAttention(frame.Bands)
	byID := make(map[gameapi.BandID]gameapi.Band, len(frame.Bands))
	for _, band := range frame.Bands {
		byID[band.ID] = band
	}
	measureLabels := make([]string, 0, len(ordered)+1)
	for _, id := range ordered {
		measureLabels = append(measureLabels, chipLabel(byID[id]))
	}
	measureLabels = append(measureLabels, fmt.Sprintf("+%d", len(ordered)))
	columns := panel.chipColumns(measureLabels)
	first, second, hidden := chipRows(ordered, state.SelectedBand, columns)
	if hidden == 0 {
		t.Fatalf("test setup: 20 bands at %d columns produced no overflow to exercise the +N chip", columns)
	}

	if want := len(first) + len(second); len(panel.handles.chips) != want {
		t.Fatalf("visible chips = %d, want %d", len(panel.handles.chips), want)
	}
	wantLabel := fmt.Sprintf("+%d", hidden)
	if panel.handles.more == nil || panel.handles.more.Text().Label != wantLabel {
		t.Fatalf("overflow chip missing or mislabelled, want %q", wantLabel)
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

// TestChipRowsPinUrgentBandsAndWindowTheRest drives chipRows directly
// (brief: pkg/hud/chips.go chipRows) across the row-1/row-2/+N split rules.
func TestChipRowsPinUrgentBandsAndWindowTheRest(t *testing.T) {
	ordered := func(n int) []gameapi.BandID {
		out := make([]gameapi.BandID, n)
		for i := range out {
			out[i] = gameapi.BandID(i)
		}
		return out
	}
	containsBand := func(ids []gameapi.BandID, id gameapi.BandID) bool {
		for _, got := range ids {
			if got == id {
				return true
			}
		}
		return false
	}
	equalBands := func(a, b []gameapi.BandID) bool {
		if len(a) != len(b) {
			return false
		}
		for i := range a {
			if a[i] != b[i] {
				return false
			}
		}
		return true
	}

	tests := []struct {
		name     string
		bands    int
		columns  int
		selected gameapi.BandID
		check    func(t *testing.T, first, second []gameapi.BandID, hidden int)
	}{
		{
			name: "fits in one row", bands: 5, columns: 8, selected: 0,
			check: func(t *testing.T, first, second []gameapi.BandID, hidden int) {
				if !equalBands(first, ordered(5)) || second != nil || hidden != 0 {
					t.Fatalf("first=%v second=%v hidden=%d", first, second, hidden)
				}
			},
		},
		{
			name: "two full rows, no overflow", bands: 12, columns: 8, selected: 0,
			check: func(t *testing.T, first, second []gameapi.BandID, hidden int) {
				full := ordered(12)
				if !equalBands(first, full[:8]) || !equalBands(second, full[8:12]) || hidden != 0 {
					t.Fatalf("first=%v second=%v hidden=%d", first, second, hidden)
				}
			},
		},
		{
			name: "selection in row 1 windows at the start", bands: 30, columns: 8, selected: 2,
			check: func(t *testing.T, first, second []gameapi.BandID, hidden int) {
				full := ordered(30)
				if !equalBands(first, full[:8]) || !equalBands(second, full[8:15]) || hidden != 15 {
					t.Fatalf("first=%v second=%v hidden=%d", first, second, hidden)
				}
			},
		},
		{
			name: "selection deep in rest keeps row 1 and centers the window", bands: 30, columns: 8, selected: 20,
			check: func(t *testing.T, first, second []gameapi.BandID, hidden int) {
				full := ordered(30)
				if !equalBands(first, full[:8]) {
					t.Fatalf("first changed: %v", first)
				}
				if len(second) != 7 {
					t.Fatalf("len(second) = %d, want 7", len(second))
				}
				if !containsBand(second, 20) {
					t.Fatalf("second %v does not contain selected band 20", second)
				}
				if hidden != 15 {
					t.Fatalf("hidden = %d, want 15", hidden)
				}
			},
		},
		{
			name: "selection at the tail clamps the window", bands: 30, columns: 8, selected: 29,
			check: func(t *testing.T, first, second []gameapi.BandID, hidden int) {
				if len(second) == 0 || second[len(second)-1] != 29 {
					t.Fatalf("second = %v, want to end at band 29", second)
				}
			},
		},
		{
			name: "narrower columns still window correctly", bands: 30, columns: 4, selected: 10,
			check: func(t *testing.T, first, second []gameapi.BandID, hidden int) {
				full := ordered(30)
				if !equalBands(first, full[:4]) {
					t.Fatalf("first = %v, want %v", first, full[:4])
				}
				if len(second) != 3 {
					t.Fatalf("len(second) = %d, want 3", len(second))
				}
				if !containsBand(second, 10) {
					t.Fatalf("second %v does not contain selected band 10", second)
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			first, second, hidden := chipRows(ordered(tt.bands), tt.selected, tt.columns)
			tt.check(t, first, second, hidden)
		})
	}
}

// TestChipRowsKeepTheSelectedBandOnScreen is the render-level companion to
// TestChipRowsPinUrgentBandsAndWindowTheRest: with 30 bands and a selection
// late in attention order, row 2 must still bring the selected band's chip
// onto the panel, row 1's pinned bands must remain, and no chip's rect may
// spill past the panel's right edge at any UI scale.
func TestChipRowsKeepTheSelectedBandOnScreen(t *testing.T) {
	frame := testFrame(30)
	selected := gameapi.BandID(25)
	for _, scale := range []float64{1, 2} {
		panel := New()
		state := testState(frame, scale)
		state.SelectedBand = selected
		state.BandListOpen = false
		panel.Update(state)
		screen := ebiten.NewImage(int(1280*scale), int(720*scale))
		panel.Draw(screen)
		screen.Deallocate()

		if _, ok := panel.handles.chips[uint32(selected)]; !ok {
			t.Fatalf("scale %.1f: selected band %d not rendered as a chip", scale, selected)
		}
		if _, _, borderPx := chipColors(false, true); borderPx != 2 {
			t.Fatalf("scale %.1f: selected chip's ring border = %v, want 2px", scale, borderPx)
		}
		for id := gameapi.BandID(1); id <= 3; id++ {
			if _, ok := panel.handles.chips[uint32(id)]; !ok {
				t.Fatalf("scale %.1f: pinned row-1 band %d missing", scale, id)
			}
		}

		rightEdge := image.Rectangle(panel.rect(panelX, panelY, panelWidth, panelHeight)).Max.X
		for id, chip := range panel.handles.chips {
			if got := chip.GetWidget().Rect.Max.X; got > rightEdge {
				t.Fatalf("scale %.1f: chip %d right edge = %d, want <= panel right edge %d", scale, id, got, rightEdge)
			}
		}
	}
}

// TestChipGridUsesTheAvailableWidth covers the reviewer-found waste: buildChips
// hardcoded a 4-column grid, so eight single-digit chips always wrapped to two
// rows of four while roughly half the column's width sat empty. The column
// count must instead come from the widest label actually being rendered —
// and must not simply become 8, since three-digit, suffering band IDs
// ("!! B256") are far wider than "B3" and would overflow the panel at 8
// columns.
func TestChipGridUsesTheAvailableWidth(t *testing.T) {
	panel := New()

	var singleDigitLabels []string
	for _, band := range testFrame(8).Bands {
		singleDigitLabels = append(singleDigitLabels, chipLabel(band))
	}
	if got := panel.chipColumns(singleDigitLabels); got != 8 {
		t.Fatalf("single-digit chip columns = %d, want 8", got)
	}

	wideFrame := testFrameWithIDs([]gameapi.BandID{100, 101, 102, 103, 104, 105, 106, 107})
	wideFrame.Bands[0].Health = 0.3
	wideFrame.Bands[1].Health = 0.3
	var wideLabels []string
	for _, band := range wideFrame.Bands {
		wideLabels = append(wideLabels, chipLabel(band))
	}
	columns := panel.chipColumns(wideLabels)
	if columns != 4 && columns != 5 {
		t.Fatalf("three-digit suffering chip columns = %d, want 4 or 5", columns)
	}

	state := testState(wideFrame, 1)
	panel.Update(state)
	screen := ebiten.NewImage(1280, 720)
	defer screen.Deallocate()
	panel.Draw(screen)

	rightEdge := image.Rectangle(panel.rect(panelX, panelY, panelWidth, panelHeight)).Max.X
	for id, chip := range panel.handles.chips {
		if got := chip.GetWidget().Rect.Max.X; got > rightEdge {
			t.Fatalf("chip %d right edge = %d, want <= panel right edge %d", id, got, rightEdge)
		}
	}
}

// TestBandListStaysOnScreenWithManyBands covers the reviewer-found overflow:
// openBandList sized its window at 40+16*len(bands) DIP with no scroll
// container, so past roughly 35 bands its rows and Close button fall below
// the 720 DIP presentation and become unreachable.
func TestBandListStaysOnScreenWithManyBands(t *testing.T) {
	frame := testFrame(60)
	panel := New()
	state := testState(frame, 1)
	state.BandListOpen = true
	panel.Update(state)
	screen := ebiten.NewImage(1280, 720)
	defer screen.Deallocate()
	panel.Draw(screen)

	if panel.handles.bandList == nil {
		t.Fatal("band list window not shown when BandListOpen")
	}
	bottom := panel.handles.bandList.GetContainer().GetWidget().Rect.Max.Y
	presentationBottom := int(state.Transform.OffsetY + render.PresentationHeight*state.Transform.Scale + 0.5)
	if bottom > presentationBottom {
		t.Fatalf("band list bottom = %d, want <= presentation bottom %d", bottom, presentationBottom)
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

// TestTraitCellHighlightsTheDisplayedNote covers the Wave G highlight: the
// details grid marks whichever variant's Field Note is currently displayed,
// found from state.Note itself (not pkg/app's traitFocus cursor, which means
// "next" on one path and "current" on the other).
func TestTraitCellHighlightsTheDisplayedNote(t *testing.T) {
	panel := New()
	state := testState(testFrame(1), 1)
	state.DetailsOpen = true
	note, ok := ui.TraitFieldNote(gameapi.PigmentationLevel, 0.4)
	if !ok {
		t.Fatal("TraitFieldNote(PigmentationLevel) not ok")
	}
	state.Note = note
	panel.Update(state)
	if panel.handles.traitFocused == nil || panel.handles.traitFocused != panel.handles.traits[gameapi.PigmentationLevel] {
		t.Fatalf("traitFocused = %v, want the Pigmentation Level cell %v", panel.handles.traitFocused, panel.handles.traits[gameapi.PigmentationLevel])
	}
	for trait, cell := range panel.handles.traits {
		if trait != gameapi.PigmentationLevel && cell == panel.handles.traitFocused {
			t.Fatalf("trait %v cell wrongly shares the focused handle", trait)
		}
	}
	state.Note = render.FieldNote{Topic: "FIRECRAFT"}
	panel.Update(state)
	if panel.handles.traitFocused != nil {
		t.Fatalf("traitFocused = %v, want nil once the drawer shows a non-trait note", panel.handles.traitFocused)
	}
}

func TestTraitCellColors(t *testing.T) {
	if border, text, borderPx := traitCellColors(true); border != colorGold || text != colorGold || borderPx != 2 {
		t.Fatalf("traitCellColors(focused) = %v, %v, %v", border, text, borderPx)
	}
	if border, text, borderPx := traitCellColors(false); border != colorPanelEdge || text != colorText || borderPx != 1 {
		t.Fatalf("traitCellColors(unfocused) = %v, %v, %v", border, text, borderPx)
	}
}

// walkDescendants visits w and, if it is a container, every descendant
// beneath it.
func walkDescendants(w widget.PreferredSizeLocateableWidget, visit func(widget.PreferredSizeLocateableWidget)) {
	visit(w)
	if container, ok := w.(*widget.Container); ok {
		for _, child := range container.Children() {
			walkDescendants(child, visit)
		}
	}
}

// TestDetailsLinesStayInsideThePanel covers the reviewer-found overflow: the
// food, deaths, stored-food, and interbreeding lines used unbounded labels
// that spilled past the 352 DIP column (spec §4 item 4).
func TestDetailsLinesStayInsideThePanel(t *testing.T) {
	for _, scale := range []float64{1, 2} {
		frame := testFrame(2)
		frame.Bands[0].LastFoodReport = gameapi.FoodTurnReport{Turn: 12, RequiredFU: 300, DeficitFU: 12}
		frame.Bands[0].LastOutcomeReport = gameapi.OutcomeReport{Turn: 12, StartingPopulation: 60, EndingPopulation: 60}
		frame.Bands[0].LastMortality = gameapi.MortalityReport{Starvation: 0, Seasonal: 0.12, Chronic: 0.31, Macro: 0, Acute: 0}
		frame.Bands[0].StoredFood = 1_234.5
		frame.Bands[0].InterbreedCandidateIDs = []gameapi.BandID{2, 3, 4, 5, 6, 7, 8, 9}

		panel := New()
		state := testState(frame, scale)
		state.DetailsOpen = true
		panel.Update(state)
		screen := ebiten.NewImage(int(1280*scale), int(720*scale))
		panel.Draw(screen)
		screen.Deallocate()

		if panel.handles.detailsBody == nil {
			t.Fatalf("scale %.1f: details body handle missing", scale)
		}
		rightEdge := image.Rectangle(panel.rect(panelX, panelY, panelWidth, panelHeight)).Max.X
		walkDescendants(panel.handles.detailsBody, func(w widget.PreferredSizeLocateableWidget) {
			if got := w.GetWidget().Rect.Max.X; got > rightEdge {
				t.Fatalf("scale %.1f: widget %T right edge = %d, want <= panel right edge %d", scale, w, got, rightEdge)
			}
		})
	}
}

// TestMoveRowButtonsFitThePanel covers a pre-existing, user-reported defect:
// the four-button row (Move here / Best tile / Split / Interbreed) is wider
// than the 352 DIP column, so Interbreed is cut off at the panel edge.
func TestMoveRowButtonsFitThePanel(t *testing.T) {
	for _, scale := range []float64{1, 2} {
		panel := New()
		state := testState(testFrame(1), scale)
		panel.Update(state)
		screen := ebiten.NewImage(int(1280*scale), int(720*scale))
		panel.Draw(screen)
		screen.Deallocate()

		rightEdge := image.Rectangle(panel.rect(panelX, panelY, panelWidth, panelHeight)).Max.X
		buttons := map[string]*widget.Button{
			"Move here":  panel.handles.moveHere,
			"Best tile":  panel.handles.best,
			"Split":      panel.handles.split,
			"Interbreed": panel.handles.interbreed,
		}
		for name, button := range buttons {
			if button == nil {
				t.Fatalf("scale %.1f: %s button handle is missing", scale, name)
			}
			if got := button.GetWidget().Rect.Max.X; got > rightEdge {
				t.Fatalf("scale %.1f: %s button right edge = %d, want <= panel right edge %d", scale, name, got, rightEdge)
			}
		}
	}
}

// TestPartnerPickerFocusesWithoutCommitting covers the picker's original
// defect: every chip emitted IntentInterbreed, the same intent as the button
// that spends the band's spatial action, so the only way to see a second
// candidate's genetics was to breed with it. The chips choose what the
// comparison describes; the button remains the sole commit.
func TestPartnerPickerFocusesWithoutCommitting(t *testing.T) {
	frame := testFrame(1)
	frame.Bands[0].InterbreedCandidateIDs = []gameapi.BandID{2, 3}
	frame.Bands = append(frame.Bands,
		gameapi.Band{ID: 2, Species: gameapi.ArchaicHominin, Population: 40, TileID: 0},
		gameapi.Band{ID: 3, Species: gameapi.ArchaicHominin, Population: 25, TileID: 0},
	)

	panel := New()
	state := testState(frame, 1)
	panel.Update(state)

	if len(panel.handles.partnerPicker) != 2 {
		t.Fatalf("picker chips = %d, want one per candidate", len(panel.handles.partnerPicker))
	}
	panel.handles.partnerPicker[1].Click()
	intents := panel.Update(state)
	if len(intents) != 1 || intents[0].Kind != IntentFocusInterbreedPartner || intents[0].Band != 3 {
		t.Fatalf("picker click = %+v, want a focus intent naming B3", intents)
	}

	if panel.handles.interbreed == nil {
		t.Fatal("interbreed button missing")
	}
	panel.handles.interbreed.Click()
	if intents := panel.Update(state); len(intents) != 1 || intents[0].Kind != IntentInterbreed {
		t.Fatalf("interbreed button = %+v, want the commit intent", intents)
	}
}

// TestPartnerGeneticsShowTheFocusedCandidate covers F2: interbreeding moves
// the band's heritable traits toward the partner's, but the panel never
// showed the partner's own values, so choosing among more than one
// candidate was blind. The block must name the focused partner, show all
// six traits, colour a cell green when the partner is higher there, and
// update when State.InterbreedFocus moves to a different candidate.
func TestPartnerGeneticsShowTheFocusedCandidate(t *testing.T) {
	frame := testFrame(1)
	frame.Bands[0].HeritableState = [gameapi.HeritableTraitCount]float64{0.1, 0.2, 0.3, 0.4, 0.5, 0.6}
	frame.Bands[0].InterbreedCandidateIDs = []gameapi.BandID{2, 3}
	frame.Bands = append(frame.Bands,
		gameapi.Band{ID: 2, Species: gameapi.ArchaicHominin, Population: 40, TileID: 0,
			HeritableState: [gameapi.HeritableTraitCount]float64{0.5, 0.1, 0.3, 0.4, 0.5, 0.6}},
		gameapi.Band{ID: 3, Species: gameapi.ArchaicHominin, Population: 25, TileID: 0,
			HeritableState: [gameapi.HeritableTraitCount]float64{0.9, 0.9, 0.9, 0.9, 0.9, 0.9}},
	)

	panel := New()
	state := testState(frame, 1)
	panel.Update(state)
	screen := ebiten.NewImage(1280, 720)
	defer screen.Deallocate()
	panel.Draw(screen)

	if panel.handles.partnerGeneticsHeading == nil {
		t.Fatal("partner genetics heading missing")
	}
	if got := panel.handles.partnerGeneticsHeading.Label; !strings.Contains(got, "B2") || !strings.Contains(got, "40") {
		t.Fatalf("partner heading = %q, want it to name B2 and its population 40", got)
	}
	for trait, value := range panel.handles.partnerGeneticsValues {
		if value == nil {
			t.Fatalf("trait %d value cell missing", trait)
		}
	}
	// widget.Text keeps its resolved color unexported with no getter, so the
	// colour rule itself is verified directly against the pure function the
	// cell construction uses (partnerTraitColor), rather than by reading the
	// built widget back.
	if got := partnerTraitColor(0.1, 0.5); got != colorGreen {
		t.Fatalf("partnerTraitColor(0.1, 0.5) = %v, want colorGreen (partner higher)", got)
	}
	if got := partnerTraitColor(0.5, 0.1); got != colorDim {
		t.Fatalf("partnerTraitColor(0.5, 0.1) = %v, want colorDim (partner lower)", got)
	}
	if got := partnerTraitColor(0.4, 0.4); got != colorText {
		t.Fatalf("partnerTraitColor(0.4, 0.4) = %v, want colorText (equal)", got)
	}

	rightEdge := image.Rectangle(panel.rect(panelX, panelY, panelWidth, panelHeight)).Max.X
	walkDescendants(panel.handles.partnerGenetics, func(w widget.PreferredSizeLocateableWidget) {
		if got := w.GetWidget().Rect.Max.X; got > rightEdge {
			t.Fatalf("partner genetics widget %T right edge = %d, want <= panel right edge %d", w, got, rightEdge)
		}
	})

	before := panel.handles.partnerGeneticsValues[gameapi.ColdAdaptation].Label
	state.InterbreedFocus = 3
	panel.Update(state)
	panel.Draw(screen)
	if got := panel.handles.partnerGeneticsHeading.Label; !strings.Contains(got, "B3") {
		t.Fatalf("partner heading after focus change = %q, want it to name B3", got)
	}
	if after := panel.handles.partnerGeneticsValues[gameapi.ColdAdaptation].Label; after == before {
		t.Fatalf("changing InterbreedFocus to a different candidate did not change the displayed values (%q both times)", before)
	}
}

// TestEndTurnStaysVisibleWithGuideAndDetailsOpen covers the reviewer-found
// overflow: header + chips + band line + details + guide + three row headers
// + an open row body + End turn + footer exceeds the 632 DIP column, and
// RowLayout neither shrinks nor scrolls (spec §4). The middle content must
// scroll so End turn and the footer stay inside the panel.
func TestEndTurnStaysVisibleWithGuideAndDetailsOpen(t *testing.T) {
	frame := testFrame(7)
	panel := New()
	state := testState(frame, 1)
	state.Guide = ui.GuideState{Step: ui.GuideResearch}
	state.DetailsOpen = true
	state.OpenRow = ui.RowMove

	panel.Update(state)
	screen := ebiten.NewImage(1280, 720)
	defer screen.Deallocate()
	panel.Draw(screen)

	if panel.handles.endTurn == nil {
		t.Fatal("End turn button missing")
	}
	panelRect := image.Rectangle(panel.rect(panelX, panelY, panelWidth, panelHeight))
	endTurnRect := panel.handles.endTurn.GetWidget().Rect
	if endTurnRect.Empty() {
		t.Fatal("End turn button has no rectangle")
	}
	if !endTurnRect.In(panelRect) {
		t.Fatalf("End turn rect = %v, want fully inside panel rect %v", endTurnRect, panelRect)
	}
}

func TestBandLineShowsLastTurnPopulationDelta(t *testing.T) {
	panel := New()
	frame := testFrame(1)
	frame.Bands[0].Population = 61
	frame.Bands[0].LastOutcomeReport = gameapi.OutcomeReport{Turn: 3, StartingPopulation: 60, EndingPopulation: 61}
	state := testState(frame, 1)
	panel.Update(state)
	if panel.handles.bandDetail == nil {
		t.Fatal("band detail label not wired up")
	}
	if label := panel.handles.bandDetail.Label; !strings.Contains(label, "Pop 61 (+1)") {
		t.Fatalf("band detail = %q, want it to contain %q", label, "Pop 61 (+1)")
	}

	freshFrame := testFrame(1)
	panel = New()
	state = testState(freshFrame, 1)
	panel.Update(state)
	if label := panel.handles.bandDetail.Label; !strings.Contains(label, "Pop 60") || strings.Contains(label, "(") {
		t.Fatalf("fresh band detail = %q, want Pop 60 with no delta", label)
	}

	declineFrame := testFrame(1)
	declineFrame.Bands[0].Population = 56
	declineFrame.Bands[0].LastOutcomeReport = gameapi.OutcomeReport{Turn: 4, StartingPopulation: 60, EndingPopulation: 56}
	panel = New()
	state = testState(declineFrame, 1)
	panel.Update(state)
	if label := panel.handles.bandDetail.Label; !strings.Contains(label, "Pop 56 (-4)") {
		t.Fatalf("decline band detail = %q, want it to contain %q", label, "Pop 56 (-4)")
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

func TestRowHeadersOpenRowsAndMoveButtonsEmitIntents(t *testing.T) {
	frame := testFrame(1)
	panel := New()
	state := testState(frame, 1)
	state.Hover = render.TileHover{TileID: 1, Visible: true}
	panel.Update(state)
	if panel.handles.moveHere == nil || panel.handles.moveHere.GetWidget().Disabled {
		t.Fatal("Move here should be enabled while hovering a reachable tile")
	}
	panel.handles.moveHere.Click()
	if intents := panel.Update(state); len(intents) != 1 || intents[0].Kind != IntentMoveTo || intents[0].Tile != 1 {
		t.Fatalf("Move here = %+v", intents)
	}
	panel.handles.best.Click()
	if intents := panel.Update(state); len(intents) != 1 || intents[0].Kind != IntentMoveToBest {
		t.Fatalf("Best tile = %+v", intents)
	}
	if !panel.handles.interbreed.GetWidget().Disabled {
		t.Fatal("Interbreed enabled without a co-located archaic band")
	}
	panel.handles.rowHeader[ui.RowResearch].Click()
	if intents := panel.Update(state); len(intents) != 1 || intents[0].Kind != IntentOpenRow || intents[0].Row != ui.RowResearch {
		t.Fatalf("row header = %+v", intents)
	}
	state.Hover = render.TileHover{}
	panel.Update(state)
	if !panel.handles.moveHere.GetWidget().Disabled {
		t.Fatal("Move here should be disabled with no target")
	}
}

func TestResearchRowListsAvailableTechnologiesAsButtons(t *testing.T) {
	panel := New()
	state := testState(testFrame(1), 1)
	state.OpenRow = ui.RowResearch
	panel.Update(state)
	if panel.handles.research[gameapi.Firecraft] == nil {
		t.Fatal("available technology has no button")
	}
	if panel.handles.research[gameapi.Campcraft] != nil && !panel.handles.research[gameapi.Campcraft].GetWidget().Disabled {
		t.Fatal("locked technology is clickable")
	}
	panel.handles.research[gameapi.Firecraft].Click()
	if intents := panel.Update(state); len(intents) != 1 || intents[0].Kind != IntentChooseResearch || intents[0].Tech != gameapi.Firecraft {
		t.Fatalf("research click = %+v", intents)
	}
}

func TestWorkforceRowRefreshesWithoutRebuildingAndGuardsApply(t *testing.T) {
	panel := New()
	frame := testFrame(1)
	state := testState(frame, 1)
	state.OpenRow = ui.RowWorkforce
	panel.Update(state)
	builds := panel.builds
	sliders := panel.handles.workforce.sliders
	if sliders[0] == nil || !panel.handles.workforce.apply.GetWidget().Disabled {
		t.Fatal("clean draft should render sliders and a disabled Apply")
	}
	state.Workforce.AllocationBP[0] = 3_600
	state.Workforce.Dirty, state.Workforce.Valid = true, false
	panel.Update(state)
	if panel.builds != builds {
		t.Fatal("a workforce-only change rebuilt the tree and would break a slider drag")
	}
	if panel.handles.workforce.sliders[0] != sliders[0] || panel.handles.workforce.sliders[0].Current != 36 {
		t.Fatalf("slider not refreshed in place: %v", panel.handles.workforce.sliders[0].Current)
	}
	if !panel.handles.workforce.apply.GetWidget().Disabled || panel.handles.workforce.total.Label != "Total 101% · reduce 1% to apply" {
		t.Fatalf("invalid total not reflected: %q", panel.handles.workforce.total.Label)
	}
	if label := panel.handles.rowHeader[ui.RowWorkforce].Text().Label; !strings.Contains(label, "Unapplied changes") {
		t.Fatalf("row header not refreshed for the dirty draft: %q", label)
	}
	panel.handles.workforce.plus[1].Click()
	if intents := panel.Update(state); len(intents) != 1 || intents[0].Kind != IntentAdjustRole || intents[0].Role != gameapi.HuntingAndFishing || intents[0].Delta != 100 {
		t.Fatalf("plus click = %+v", intents)
	}
	state.Workforce.AllocationBP[1] = 2_900
	state.Workforce.Valid = true
	panel.Update(state)
	if panel.handles.workforce.apply.GetWidget().Disabled {
		t.Fatal("valid dirty draft left Apply disabled")
	}
	if label := panel.handles.rowHeader[ui.RowWorkforce].Text().Label; !strings.Contains(label, "Unapplied changes") {
		t.Fatalf("row header lost the dirty state once the draft became valid: %q", label)
	}
	panel.handles.workforce.apply.Click()
	if intents := panel.Update(state); len(intents) != 1 || intents[0].Kind != IntentApplyWorkforce {
		t.Fatalf("apply click = %+v", intents)
	}
	state.Workforce.AllocationBP = frame.Bands[0].AllocationBP
	state.Workforce.Dirty = false
	panel.Update(state)
	if panel.builds != builds {
		t.Fatal("clearing Dirty rebuilt the tree instead of refreshing in place")
	}
	if label := panel.handles.rowHeader[ui.RowWorkforce].Text().Label; !strings.Contains(label, "F 35") {
		t.Fatalf("row header did not return to the applied summary: %q", label)
	}
}

// TestWorkforceMarkerFollowsTheSelectedRole covers the reviewer-found defect:
// SelectedRole changes take the refresh-in-place path (Workforce is excluded
// from the structural key), but refreshWorkforce updated only percentages and
// totals, so the › marker stayed on the old role while Left/Right edited a
// different one.
func TestWorkforceMarkerFollowsTheSelectedRole(t *testing.T) {
	panel := New()
	frame := testFrame(1)
	state := testState(frame, 1)
	state.OpenRow = ui.RowWorkforce
	panel.Update(state)
	builds := panel.builds

	state.Workforce.SelectedRole = gameapi.Toolcraft
	panel.Update(state)

	if panel.builds != builds {
		t.Fatal("a SelectedRole-only change rebuilt the tree")
	}
	if label := panel.handles.workforce.roleLabels[gameapi.Toolcraft].Label; !strings.HasPrefix(label, "› ") {
		t.Fatalf("Toolcraft label = %q, want the › marker", label)
	}
	if label := panel.handles.workforce.roleLabels[gameapi.Foraging].Label; strings.HasPrefix(label, "› ") {
		t.Fatalf("Foraging label = %q, should have lost the › marker", label)
	}
}

// TestWorkforceRowShowsWorkerCounts covers the old HUD behavior the redesign
// dropped: WorkforceDraft.Population was carried but never shown, so a player
// could not see how a percentage allocation translated into actual people.
func TestWorkforceRowShowsWorkerCounts(t *testing.T) {
	panel := New()
	frame := testFrame(1)
	state := testState(frame, 1)
	state.OpenRow = ui.RowWorkforce
	state.Workforce.Population = 60
	state.Workforce.AllocationBP[gameapi.Foraging] = 3_500
	panel.Update(state)

	label := panel.handles.workforce.values[gameapi.Foraging].Label
	if !strings.Contains(label, "35%") || !strings.Contains(label, "21") {
		t.Fatalf("Foraging value label = %q, want it to contain 35%% and 21", label)
	}
}

// TestWorkforceSlidersShareOneColumn covers the reviewer-found misalignment:
// each row's role label was "%-14s"-padded, but that padding does nothing in
// a proportional font, so the five role labels have different pixel widths
// and every row's −, slider, + and value start at a different x. The row
// must instead be a fixed-label-column grid so all five sliders share one
// left edge and one right edge, and all five value labels share one left
// edge.
func TestWorkforceSlidersShareOneColumn(t *testing.T) {
	panel := New()
	state := testState(testFrame(1), 1)
	state.OpenRow = ui.RowWorkforce
	panel.Update(state)
	screen := ebiten.NewImage(1280, 720)
	defer screen.Deallocate()
	panel.Draw(screen)

	h := panel.handles.workforce
	minX, maxX, valueMinX := -1, -1, -1
	for role := range h.sliders {
		if h.sliders[role] == nil {
			t.Fatalf("role %d: slider handle missing", role)
		}
		rect := h.sliders[role].GetWidget().Rect
		if minX == -1 {
			minX, maxX = rect.Min.X, rect.Max.X
		} else if rect.Min.X != minX || rect.Max.X != maxX {
			t.Fatalf("role %d: slider rect = %v, want Min.X %d and Max.X %d to match every other role", role, rect, minX, maxX)
		}
		valueRect := h.values[role].GetWidget().Rect
		if valueMinX == -1 {
			valueMinX = valueRect.Min.X
		} else if valueRect.Min.X != valueMinX {
			t.Fatalf("role %d: value label Min.X = %d, want %d to match every other role", role, valueRect.Min.X, valueMinX)
		}
	}
}

func TestDrawerHasThreeStatesAndClickableEvents(t *testing.T) {
	frame := testFrame(1)
	frame.Events = []gameapi.Event{{Turn: 11, Kind: gameapi.EventMigration, Summary: "Band 1 migrated"}, {Turn: 12, Kind: gameapi.EventMigration, Summary: "Band 1 migrated again"}}
	panel := New()
	state := testState(frame, 1)
	state.Note = render.FieldNote{Topic: "FIRECRAFT", Introduction: "Controlled fire.", Context: "Attested early.", GameEffect: "Raises survival."}
	panel.Update(state)
	if panel.handles.drawerTab == nil || panel.handles.drawerTab.Text().Label != "hide notes · F" {
		t.Fatal("compact drawer tab missing")
	}
	if panel.handles.drawerMore == nil || panel.handles.drawerMore.Text().Label != "▲ more" {
		t.Fatal("compact drawer lacks the expand control")
	}
	if len(panel.handles.events) != 2 || panel.handles.events[0].Text().Label != "T12 · Migration · Band 1 migrated again" {
		t.Fatalf("event lines = %d, first %q", len(panel.handles.events), panel.handles.events[0].Text().Label)
	}
	panel.handles.events[0].Click()
	if intents := panel.Update(state); len(intents) != 1 || intents[0].Kind != IntentFocusEvent || intents[0].Event != gameapi.EventMigration {
		t.Fatalf("event click = %+v", intents)
	}
	panel.handles.drawerMore.Click()
	if intents := panel.Update(state); len(intents) != 1 || intents[0].Kind != IntentSetNotesMode || intents[0].Notes != NotesExpanded {
		t.Fatalf("more click = %+v", intents)
	}
	state.NotesMode = NotesExpanded
	panel.Update(state)
	if panel.handles.drawerMore.Text().Label != "▼ less" {
		t.Fatal("expanded drawer lacks the collapse control")
	}
	state.NotesMode = NotesHidden
	panel.Update(state)
	if panel.handles.drawerTab.Text().Label != "▲ notes · F" {
		t.Fatalf("hidden bar control = %q, want the plain control label now that the event moved to its own button", panel.handles.drawerTab.Text().Label)
	}
	if panel.handles.drawerBarEvent == nil || panel.handles.drawerBarEvent.Text().Label != "T12 · Migration · Band 1 migrated again" {
		t.Fatalf("hidden bar event button = %+v, want the newest event retained", panel.handles.drawerBarEvent)
	}
	if panel.handles.drawerMore != nil || len(panel.handles.events) != 0 {
		t.Fatal("hidden drawer still shows body controls")
	}
}

// manyEvents builds count distinct events, oldest first, so a test can ask
// for more lines than any drawer mode will actually show.
func manyEvents(count int) []gameapi.Event {
	events := make([]gameapi.Event, 0, count)
	for index := 0; index < count; index++ {
		events = append(events, gameapi.Event{Turn: index + 1, Kind: gameapi.EventMigration, Summary: fmt.Sprintf("Band %d migrated.", index+1)})
	}
	return events
}

// drawEverything builds and draws the panel so every widget has been through
// a layout pass and carries a real Rect. Several drawer assertions below
// compare rectangles, which are zero until a Draw has run.
func drawEverything(t *testing.T, panel *Panel, state State) {
	t.Helper()
	panel.Update(state)
	screen := ebiten.NewImage(1280, 720)
	panel.Draw(screen)
	screen.Deallocate()
}

// TestDrawerColumnsSitSideBySide covers the two-column drawer body: Field
// Notes on the left, RECENT EVENTS in its own right-hand column rather than
// stacked underneath. The stacked layout spent 42 of compact mode's 102 DIP
// on the events heading and its rows, leaving the note body a single line;
// side by side both columns get the drawer's full inner height.
func TestDrawerColumnsSitSideBySide(t *testing.T) {
	for _, mode := range []NotesMode{NotesCompact, NotesExpanded} {
		frame := testFrame(1)
		frame.Events = manyEvents(30)
		panel := New()
		state := testState(frame, 1)
		state.NotesMode = mode
		state.Note = render.FieldNote{Topic: "FIRECRAFT", Introduction: "Controlled fire."}
		drawEverything(t, panel, state)

		notes, events := panel.handles.notesColumn, panel.handles.eventsColumn
		if notes == nil || events == nil {
			t.Fatalf("%v: drawer is missing a column (notes %v, events %v)", mode, notes, events)
		}
		notesRect, eventsRect := notes.GetWidget().Rect, events.GetWidget().Rect
		if notesRect.Max.X > eventsRect.Min.X {
			t.Fatalf("%v: notes column ends at x=%d but events column starts at x=%d, want them side by side without overlap", mode, notesRect.Max.X, eventsRect.Min.X)
		}
		if notesRect.Min.Y != eventsRect.Min.Y {
			t.Fatalf("%v: column tops = %d and %d, want both columns to start at the same y", mode, notesRect.Min.Y, eventsRect.Min.Y)
		}
		body := panel.handles.drawerBody.GetWidget().Rect
		if !notesRect.In(body) || !eventsRect.In(body) {
			t.Fatalf("%v: columns %v and %v escape the drawer body %v", mode, notesRect, eventsRect, body)
		}
	}
}

// TestDrawerEventColumnFillsItsHeight covers the point of giving events their
// own column: it holds as many events as fit, so expanding the drawer shows
// more history rather than just more note. The 30-event frame is deeper than
// either mode can show, so the counts here are the layout's, not the data's.
func TestDrawerEventColumnFillsItsHeight(t *testing.T) {
	counts := map[NotesMode]int{}
	for _, mode := range []NotesMode{NotesCompact, NotesExpanded} {
		frame := testFrame(1)
		frame.Events = manyEvents(30)
		panel := New()
		state := testState(frame, 1)
		state.NotesMode = mode
		drawEverything(t, panel, state)
		counts[mode] = len(panel.handles.events)

		if counts[mode] == 0 {
			t.Fatalf("%v: event column is empty with 30 events available", mode)
		}
		body := panel.handles.drawerBody.GetWidget().Rect
		for index, line := range panel.handles.events {
			if got := line.GetWidget().Rect; !got.In(body) {
				t.Fatalf("%v: event row %d at %v escapes the drawer body %v — the column is showing more rows than fit", mode, index, got, body)
			}
		}
		if got, want := panel.handles.events[0].Text().Label, eventLine(frame.Events[len(frame.Events)-1]); got != want {
			t.Fatalf("%v: first event row = %q, want the newest event %q", mode, got, want)
		}
	}
	if counts[NotesExpanded] <= counts[NotesCompact] {
		t.Fatalf("event rows: compact %d, expanded %d — want expanding the drawer to show strictly more history", counts[NotesCompact], counts[NotesExpanded])
	}
}

// TestDrawerNoteBodyGainsTheEventRowsHeight covers the reason for the
// rearrangement: with the events beside the note rather than below it, the
// note body's scroll viewport is taller than the whole events block used to
// let it be. Compact mode is where the old formula hurt most.
func TestDrawerNoteBodyGainsTheEventRowsHeight(t *testing.T) {
	frame := testFrame(1)
	frame.Events = manyEvents(30)
	panel := New()
	state := testState(frame, 1)
	state.NotesMode = NotesCompact
	state.Note = render.FieldNote{Topic: "FIRECRAFT", Introduction: strings.Repeat("Controlled fire. ", 40)}
	drawEverything(t, panel, state)

	// The stacked layout gave the note height-24-14*3-12 = 24 DIP at compact.
	// Anything at or below that means the events are still eating the column.
	const stackedNoteHeightDIP = 24.0
	got := panel.handles.notesScroll.ViewRect().Dy()
	if want := panel.theme.px(stackedNoteHeightDIP); got <= want {
		t.Fatalf("compact note viewport = %d render px, want more than the %d px the stacked layout left it", got, want)
	}
}

// TestDrawerEventColumnTruncatesLongSummaries mirrors the hidden bar's
// truncation: the events column has a fixed width, so a summary that would
// overflow it is shortened rather than drawn over the note column.
func TestDrawerEventColumnTruncatesLongSummaries(t *testing.T) {
	frame := testFrame(1)
	frame.Events = []gameapi.Event{{Turn: 12, Kind: gameapi.EventMigration, Summary: strings.Repeat("x", 300)}}
	panel := New()
	state := testState(frame, 1)
	state.NotesMode = NotesExpanded
	drawEverything(t, panel, state)

	label := panel.handles.events[0].Text().Label
	if full := eventLine(frame.Events[0]); !strings.HasSuffix(label, "…") || len([]rune(label)) >= len([]rune(full)) {
		t.Fatalf("event column line with a long summary = %q, want it shortened and ending in an ellipsis", label)
	}
	budget := panel.theme.eventColumnTextBudgetPx()
	if width, _ := text.Measure(label, *panel.theme.face(8.5), 0); width > budget {
		t.Fatalf("event column line width = %.1f render px, want at most the %.1f px column budget", width, budget)
	}
}

// TestHiddenDrawerSpansTheMapWidth covers D1: the hidden drawer is a
// full-width bar along the bottom of the map, not a small corner tab, so the
// collapsed state shares the drawer's left edge and can show a whole event
// line. Both the event text and the notes control open the drawer.
func TestHiddenDrawerSpansTheMapWidth(t *testing.T) {
	frame := testFrame(1)
	frame.Events = []gameapi.Event{{Turn: 12, Kind: gameapi.EventMigration, Summary: "Band 1 migrated"}}
	panel := New()
	state := testState(frame, 1)
	state.NotesMode = NotesHidden
	panel.Update(state)
	screen := ebiten.NewImage(1280, 720)
	panel.Draw(screen)
	screen.Deallocate()

	want := image.Rectangle(panel.rect(mapLeft, mapBottom-drawerHiddenH, mapRight-mapLeft, drawerHiddenH))
	if got := panel.handles.drawerBar.GetWidget().Rect; got != want {
		t.Fatalf("hidden bar rect = %v, want %v (spanning the full map width)", got, want)
	}
	if got, want := panel.handles.drawerBar.GetWidget().Rect.Dy(), panel.theme.px(drawerHiddenH); got != want {
		t.Fatalf("hidden bar height = %d render px, want %d", got, want)
	}
	if panel.handles.drawerBarEvent == nil {
		t.Fatal("hidden bar has no clickable event text")
	}
	panel.handles.drawerBarEvent.Click()
	if intents := panel.Update(state); len(intents) != 1 || intents[0].Kind != IntentSetNotesMode || intents[0].Notes != NotesCompact {
		t.Fatalf("clicking the hidden bar's event text = %+v, want IntentSetNotesMode(NotesCompact)", intents)
	}
	panel.handles.drawerTab.Click()
	if intents := panel.Update(state); len(intents) != 1 || intents[0].Kind != IntentSetNotesMode || intents[0].Notes != NotesCompact {
		t.Fatalf("clicking the hidden bar's control = %+v, want IntentSetNotesMode(NotesCompact)", intents)
	}
}

func TestHiddenTabTruncatesLongEventsAndShowsBreakthrough(t *testing.T) {
	frame := testFrame(1)
	frame.Events = []gameapi.Event{{Turn: 12, Kind: gameapi.EventMigration, Summary: strings.Repeat("x", 300)}}
	panel := New()
	state := testState(frame, 1)
	state.NotesMode = NotesHidden
	panel.Update(state)
	label := panel.handles.drawerBarEvent.Text().Label
	full := eventLine(frame.Events[0])
	if !strings.HasSuffix(label, "…") || len([]rune(label)) >= len([]rune(full)) {
		t.Fatalf("hidden bar event with a long summary = %q, want it shortened and ending in an ellipsis", label)
	}
	budget := panel.hiddenBarEventBudgetPx(panel.theme)
	if width, _ := text.Measure(label, *panel.theme.face(9), 0); width > budget {
		t.Fatalf("hidden bar event width = %.1f, want at most the %.1f px budget left after the notes control", width, budget)
	}
	state.Note.Celebration = true
	panel.Update(state)
	if got := panel.handles.drawerBarEvent.Text().Label; !strings.HasPrefix(got, "BREAKTHROUGH · ") {
		t.Fatalf("celebration hidden bar event = %q, want the BREAKTHROUGH prefix", got)
	}
}

func TestEndTurnButtonReflectsTheGate(t *testing.T) {
	panel := New()
	frame := testFrame(2)
	state := testState(frame, 1)
	panel.Update(state)
	if panel.handles.endTurn.GetWidget().Disabled || panel.handles.endTurn.Text().Label != "End turn · 2 bands still need a move" {
		t.Fatalf("soft gate button = %q", panel.handles.endTurn.Text().Label)
	}
	panel.handles.endTurn.Click()
	if intents := panel.Update(state); len(intents) != 1 || intents[0].Kind != IntentEndTurn || intents[0].Force {
		t.Fatalf("soft click = %+v", intents)
	}
	state.EndTurn = ui.EndTurnGateFor(frame, true, false, false)
	panel.Update(state)
	if !panel.handles.endTurn.GetWidget().Disabled {
		t.Fatal("hard block did not disable the button")
	}
	state.EndTurn = ui.EndTurnGateFor(frame, false, false, true)
	panel.Update(state)
	panel.handles.endTurn.Click()
	if intents := panel.Update(state); len(intents) != 1 || !intents[0].Force {
		t.Fatalf("armed click = %+v", intents)
	}
}

func TestCameraButtonReflectsFocusAndEmitsToggle(t *testing.T) {
	panel := New()
	state := testState(testFrame(1), 1)
	state.Camera = CameraState{ToggleAvailable: true, Focused: true}
	panel.Update(state)
	if panel.handles.camera == nil || panel.handles.camera.Text().Label != "Overview · Z" {
		t.Fatal("focused camera should offer Overview")
	}
	panel.handles.camera.Click()
	if intents := panel.Update(state); len(intents) != 1 || intents[0].Kind != IntentCameraToggle {
		t.Fatalf("camera click = %+v", intents)
	}
	state.Camera = CameraState{}
	panel.Update(state)
	if panel.handles.camera != nil {
		t.Fatal("camera button shown with no selection to toggle")
	}
}

// TestEndSceneHidesTheCameraButtonAndCentersNewCampaign covers Wave H item
// H2: the map-corner Overview/Focus button used to draw over the terminal
// dialog, and New Campaign was centred on the screen rather than on the
// (now off-centre) dialog. The camera button must yield while the dialog is
// up, and New Campaign must track render.EndSceneX/Width so it recentres
// itself if the dialog's rect ever moves again.
func TestEndSceneHidesTheCameraButtonAndCentersNewCampaign(t *testing.T) {
	panel := New()
	state := testState(testFrame(1), 1)
	state.Camera = CameraState{ToggleAvailable: true}
	state.Ending = render.EndScene{Visible: true}
	panel.Update(state)
	if panel.handles.camera != nil {
		t.Fatal("camera button shown while the end scene is up")
	}
	if panel.handles.newCampaign == nil {
		t.Fatal("no New Campaign button while the end scene is up")
	}

	screen := ebiten.NewImage(1280, 720)
	panel.Draw(screen)

	dialog := panel.rect(render.EndSceneX, render.EndSceneY, render.EndSceneWidth, render.EndSceneHeight)
	dialogCentreX := (dialog.Min.X + dialog.Max.X) / 2
	button := panel.handles.newCampaign.GetWidget().Rect
	buttonCentreX := (button.Min.X + button.Max.X) / 2
	if buttonCentreX != dialogCentreX {
		t.Fatalf("New Campaign centre x = %d, want the dialog's centre %d", buttonCentreX, dialogCentreX)
	}
	if button.Max.X > dialog.Max.X {
		t.Fatalf("New Campaign right edge %d spills past the dialog's right edge %d", button.Max.X, dialog.Max.X)
	}

	state.Camera = CameraState{ToggleAvailable: true}
	state.Ending = render.EndScene{}
	panel.Update(state)
	if panel.handles.camera == nil {
		t.Fatal("camera button stayed hidden once the end scene closed")
	}
}

func TestGuideCardShowsStepProgressAndOnlyXDismisses(t *testing.T) {
	panel := New()
	state := testState(testFrame(1), 1)
	state.Guide = ui.NewGuideState(false)
	panel.Update(state)
	if panel.handles.guideNext == nil || panel.handles.guideX == nil {
		t.Fatal("guide card controls missing on a fresh campaign")
	}
	panel.handles.guideNext.Click()
	if intents := panel.Update(state); len(intents) != 1 || intents[0].Kind != IntentGuideNext {
		t.Fatalf("Next = %+v", intents)
	}
	panel.handles.guideX.Click()
	if intents := panel.Update(state); len(intents) != 1 || intents[0].Kind != IntentGuideDismiss {
		t.Fatalf("× = %+v", intents)
	}
	state.Guide = ui.GuideState{Step: ui.GuideClosing}
	panel.Update(state)
	if panel.handles.guideNext != nil || panel.handles.guideX == nil {
		t.Fatal("closing card should offer only the ×")
	}
	state.Guide = ui.NewGuideState(true)
	panel.Update(state)
	if panel.handles.guideX != nil {
		t.Fatal("dismissed guide still rendered")
	}
}

// TestArchaicSelectionDisablesTheMoveRowButtons covers the read-only rule for
// a computer-controlled selection: the Move row still renders its comparison,
// but none of its four actions may be sent (spec §4.1).
func TestArchaicSelectionDisablesTheMoveRowButtons(t *testing.T) {
	frame := testFrame(1)
	frame.Bands[0].Species = gameapi.ArchaicHominin
	frame.Bands[0].InterbreedCandidateIDs = []gameapi.BandID{2}
	panel := New()
	state := testState(frame, 1)
	state.Hover = render.TileHover{TileID: 1, Visible: true}
	panel.Update(state)
	for name, button := range map[string]*widget.Button{
		"Move here": panel.handles.moveHere, "Best tile": panel.handles.best,
		"Split": panel.handles.split, "Interbreed": panel.handles.interbreed,
	} {
		if button == nil {
			t.Fatalf("%s button missing", name)
		}
		if !button.GetWidget().Disabled {
			t.Fatalf("%s is enabled for a computer-controlled band", name)
		}
	}
	if label := panel.handles.bandDetail.Label; !strings.Contains(label, "Computer controlled") {
		t.Fatalf("band line = %q, want it to say the band is computer controlled", label)
	}
}

// testCursor places ebitenui's pointer at a fixed render-pixel position.
// Everything except CursorPosition is an inert stub: ebiten's mouse-button
// state cannot be injected from a test, so this fixture can only exercise
// hit-testing (Panel.Hovered), never clicks. Button clicks are covered by
// calling Button.Click directly elsewhere in this file.
type testCursor struct{ x, y int }

func (c *testCursor) Update()                                         {}
func (c *testCursor) AfterUpdate()                                    {}
func (c *testCursor) Draw(*ebiten.Image)                              {}
func (c *testCursor) AfterDraw(*ebiten.Image)                         {}
func (c *testCursor) MouseButtonPressed(ebiten.MouseButton) bool      { return false }
func (c *testCursor) MouseButtonJustPressed(ebiten.MouseButton) bool  { return false }
func (c *testCursor) MouseButtonJustReleased(ebiten.MouseButton) bool { return false }
func (c *testCursor) CursorPosition() (int, int)                      { return c.x, c.y }
func (c *testCursor) GetCursorImage(string) *ebiten.Image             { return nil }
func (c *testCursor) GetCursorOffset(string) image.Point              { return image.Point{} }

// TestChromeHitTestingFollowsThePresentationTransform is the DIP contract for
// spec §4: ebitenui lays out and hit-tests in render pixels, so a chrome
// widget must be hovered at its render-pixel rectangle — including the
// letterbox offset — and the map area must never register as chrome, whatever
// the presentation scale.
func TestChromeHitTestingFollowsThePresentationTransform(t *testing.T) {
	t.Cleanup(func() { input.SetCursorUpdater(nil) })
	for _, viewport := range []render.Viewport{
		render.NextViewport(render.Viewport{}, 1280, 720, 1),
		render.NextViewport(render.Viewport{}, 1280*2, 900*2, 1),
	} {
		transform := render.FitPresentation(viewport.RenderWidthPx, viewport.RenderHeightPx)
		panel := New()
		state := testState(testFrame(1), 1)
		state.Viewport, state.Transform = viewport, transform
		screen := ebiten.NewImage(viewport.RenderWidthPx, viewport.RenderHeightPx)
		defer screen.Deallocate()
		// The first Update/Draw pair lays the tree out; widget rectangles are
		// assigned during Draw, and input.UIHovered is published there too.
		panel.Update(state)
		panel.Draw(screen)
		rect := panel.handles.endTurn.GetWidget().Rect
		if rect.Empty() {
			t.Fatalf("scale %.1f: End turn button has no rectangle", transform.Scale)
		}
		// The panel itself must sit at its DIP geometry mapped through the
		// presentation transform, letterbox offset included; a cursor derived
		// from a widget rectangle alone would follow a transform mistake
		// instead of catching it.
		left, top := transform.LogicalToRender(panelX, panelY)
		want := image.Rect(int(left+0.5), int(top+0.5), int(left+0.5)+int(panelWidth*transform.Scale+0.5), int(top+0.5)+int(panelHeight*transform.Scale+0.5))
		if got := panel.root.Children()[0].GetWidget().Rect; got != want {
			t.Fatalf("scale %.1f: panel rect = %v, want %v", transform.Scale, got, want)
		}

		cursor := &testCursor{x: (rect.Min.X + rect.Max.X) / 2, y: (rect.Min.Y + rect.Max.Y) / 2}
		input.SetCursorUpdater(cursor)
		panel.Update(state)
		panel.Draw(screen)
		if !panel.Hovered() {
			t.Fatalf("scale %.1f: pointer at the End turn button centre %v is not over chrome", transform.Scale, *cursor)
		}

		if transform.Scale != 1 || transform.OffsetY != 0 {
			// Same button, addressed in DIP instead of render pixels: the
			// chrome must not answer there, or the panel would be hit-testing
			// in the wrong coordinate space.
			cursor.x = int(float64(cursor.x-int(transform.OffsetX)) / transform.Scale)
			cursor.y = int(float64(cursor.y-int(transform.OffsetY)) / transform.Scale)
			panel.Update(state)
			panel.Draw(screen)
			if panel.Hovered() {
				t.Fatalf("scale %.1f: the button's DIP centre %v registered as chrome", transform.Scale, *cursor)
			}
		}

		mapX, mapY := transform.LogicalToRender(400, 400)
		cursor.x, cursor.y = int(mapX), int(mapY)
		panel.Update(state)
		panel.Draw(screen)
		if panel.Hovered() {
			t.Fatalf("scale %.1f: pointer inside the map at %v registered as chrome", transform.Scale, *cursor)
		}
	}
}

// TestEndTurnDisappearsOnceTheCampaignIsOver covers spec §5.3: after victory,
// extinction, or dispersal failure there is no turn left to end, so the
// button is omitted rather than rendered as an empty disabled bar.
func TestEndTurnDisappearsOnceTheCampaignIsOver(t *testing.T) {
	panel := New()
	frame := testFrame(1)
	state := testState(frame, 1)
	panel.Update(state)
	if panel.handles.endTurn == nil {
		t.Fatal("an ongoing campaign has no End turn button")
	}
	frame.CampaignResult = gameapi.Extinction
	over := testState(frame, 1)
	over.EndTurn = ui.EndTurnGate{}
	over.Ending = ui.CampaignEndScene(frame)
	panel.Update(over)
	if panel.handles.endTurn != nil {
		t.Fatal("End turn is still rendered after the campaign ended")
	}
}

// TestResearchCursorRowIsMarked covers the reviewer-found defect: Up/Down
// moved g.researchCursor but it never reached hud.State, and buildResearchBody
// ignored its state argument, so pressing Enter committed an invisible
// selection.
func TestResearchCursorRowIsMarked(t *testing.T) {
	frame := testFrame(1)
	panel := New()
	state := testState(frame, 1)
	state.OpenRow = ui.RowResearch
	state.ResearchCursor = gameapi.HaftedTools
	panel.Update(state)

	cursorLabel := panel.handles.research[gameapi.HaftedTools].Text().Label
	if !strings.HasPrefix(cursorLabel, "› ") {
		t.Fatalf("cursor row label = %q, want a › prefix", cursorLabel)
	}
	for tech := gameapi.Tech(0); tech < gameapi.TechCount; tech++ {
		if tech == gameapi.HaftedTools {
			continue
		}
		if label := panel.handles.research[tech].Text().Label; strings.HasPrefix(label, "› ") {
			t.Fatalf("non-cursor row %v label = %q, should not carry the › prefix", tech, label)
		}
	}
}

// TestHoverRefreshesTargetWithoutRebuilding covers the reviewer-found defect:
// State.Hover was part of the structural comparison, so sweeping the pointer
// across map tiles rebuilt the whole widget tree at up to 60 Hz, discarding
// the Field Notes drawer's scroll container (and the reader's scroll
// position with it) on every tick.
func TestHoverRefreshesTargetWithoutRebuilding(t *testing.T) {
	frame := testFrame(1)
	frame.Tiles[1].NearbyLake = "Lake Victoria"
	panel := New()
	state := testState(frame, 1)
	panel.Update(state)

	buildsBefore := panel.builds
	notesScrollBefore := panel.handles.notesScroll
	if notesScrollBefore == nil {
		t.Fatal("drawer scroll container handle missing before the hover change")
	}
	if got := panel.handles.moveTargetHeader.Label; got != "TARGET" {
		t.Fatalf("TARGET header before hover = %q, want plain TARGET", got)
	}

	state.Hover = render.TileHover{TileID: 1, Visible: true}
	panel.Update(state)

	if panel.builds != buildsBefore {
		t.Fatalf("hover change rebuilt the tree: builds %d -> %d", buildsBefore, panel.builds)
	}
	if panel.handles.notesScroll != notesScrollBefore {
		t.Fatal("hover change replaced the drawer's scroll container, losing the reader's scroll position")
	}
	if got := panel.handles.moveTargetHeader.Label; got != "TARGET · hover" {
		t.Fatalf("TARGET header while hovering = %q, want TARGET · hover", got)
	}
	if got := panel.handles.moveTargetValues[0].Label; got != frame.Tiles[1].Biome.String() {
		t.Fatalf("TARGET biome cell = %q, want the hovered tile's biome %q", got, frame.Tiles[1].Biome.String())
	}
	if got := panel.handles.moveTargetValues[1].Label; got != frame.Tiles[1].Region.String() {
		t.Fatalf("TARGET region cell = %q, want %q", got, frame.Tiles[1].Region.String())
	}
	if got := panel.handles.moveTargetValues[2].Label; got != "Lake Victoria" {
		t.Fatalf("TARGET nearby lake = %q", got)
	}
	if panel.handles.moveHere.GetWidget().Disabled {
		t.Fatal("Move here should be enabled once a reachable tile is hovered")
	}

	state.Hover = render.TileHover{}
	panel.Update(state)
	if panel.builds != buildsBefore {
		t.Fatalf("clearing hover rebuilt the tree: builds %d -> %d", buildsBefore, panel.builds)
	}
	if got := panel.handles.moveTargetHeader.Label; got != "TARGET" {
		t.Fatalf("TARGET header after clearing hover = %q, want plain TARGET", got)
	}
	if panel.handles.moveHere.GetWidget().Disabled != true {
		t.Fatal("Move here should be disabled again once there is no target")
	}
}

func TestHoverShowsWaterGeographyWithoutEnablingMovement(t *testing.T) {
	frame := testFrame(1)
	frame.Tiles[1].Land = false
	frame.Tiles[1].WaterBody = "Red Sea"
	frame.Bands[0].MigrationCandidates = nil
	panel := New()
	state := testState(frame, 1)
	panel.Update(state)
	state.Hover = render.TileHover{TileID: 1, Visible: true}
	panel.Update(state)
	if panel.handles.moveTargetValues[0].Label != "Open water" || panel.handles.moveTargetValues[1].Label != "Red Sea" {
		t.Fatal("hovered water lost its biome or region")
	}
	if !panel.handles.moveHere.GetWidget().Disabled {
		t.Fatal("water became a valid movement destination")
	}
}

// TestSimultaneousExcludedFieldChangesAllRefresh covers a structural defect
// in Update's dispatch: it was one switch over mutually exclusive cases
// (rebuild / Workforce / MasterVolume / Hover), so if two of the
// switch-excluded fields (Workforce, Overlay.MasterVolume, Hover) changed in
// the same tick, only the first matching case would run and p.last = state
// would still absorb every field's new value — silently dropping the other
// field's refresh with no way for a later tick to detect it, since p.last
// already agrees with state.
//
// The Move row is opened (not the Workforce row) so the target half is
// directly observable: with the Workforce row open instead, buildMoveBody
// never runs and moveTargetHeader stays nil, so the TARGET-header assertion
// below would be impossible to make. With the Move row open, the workforce
// body (sliders, values, role labels) instead never runs and stays nil, so
// the workforce half is asserted the other way: no panic when refreshWorkforce
// runs against a closed row, and p.last correctly carries the new
// SelectedRole rather than staying stale.
func TestSimultaneousExcludedFieldChangesAllRefresh(t *testing.T) {
	frame := testFrame(1)
	panel := New()
	state := testState(frame, 1)
	panel.Update(state)
	builds := panel.builds

	state.Workforce.SelectedRole = gameapi.Toolcraft
	state.Hover = render.TileHover{TileID: 1, Visible: true}
	panel.Update(state)

	if panel.builds != builds {
		t.Fatalf("simultaneous Workforce and Hover changes rebuilt the tree: builds %d -> %d", builds, panel.builds)
	}
	if got := panel.handles.moveTargetHeader.Label; got != "TARGET · hover" {
		t.Fatalf("TARGET header = %q, want TARGET . hover: the Hover-driven refresh was dropped because Workforce also changed this tick", got)
	}
	if panel.last.Workforce.SelectedRole != gameapi.Toolcraft {
		t.Fatalf("p.last.Workforce.SelectedRole = %v, want %v: p.last went stale for the field whose refresh did not run", panel.last.Workforce.SelectedRole, gameapi.Toolcraft)
	}
	if panel.last.Hover != state.Hover {
		t.Fatalf("p.last.Hover = %+v, want %+v", panel.last.Hover, state.Hover)
	}
}

// TestPresentationKeyChangesWithRebuildsAndRefreshes covers pkg/app's use of
// PresentationKey to decide whether the map needs to repaint chrome it
// otherwise wouldn't know changed: a rebuild (a structural State change) and
// each in-place refresh path (Workforce, Hover) must all change the key, and
// two identical Update calls with no cursor movement must not.
func TestPresentationKeyChangesWithRebuildsAndRefreshes(t *testing.T) {
	panel := New()
	frame := testFrame(1)
	state := testState(frame, 1) // OpenRow: ui.RowMove
	panel.Update(state)
	key := panel.PresentationKey()

	panel.Update(state)
	if panel.PresentationKey() != key {
		t.Fatal("PresentationKey changed across two identical Update calls with no cursor movement")
	}

	state.OpenRow = ui.RowResearch
	panel.Update(state)
	if panel.PresentationKey() == key {
		t.Fatal("PresentationKey did not change after a rebuild (OpenRow changed)")
	}

	state.OpenRow = ui.RowMove
	panel.Update(state)
	key = panel.PresentationKey()

	state.Workforce.SelectedRole = gameapi.Toolcraft
	panel.Update(state)
	if panel.PresentationKey() == key {
		t.Fatal("PresentationKey did not change after a Workforce.SelectedRole-only refresh")
	}
	key = panel.PresentationKey()

	state.Hover = render.TileHover{TileID: 1, Visible: true}
	panel.Update(state)
	if panel.PresentationKey() == key {
		t.Fatal("PresentationKey did not change after a Hover-only refresh")
	}
}

// TestPresentationKeyChangesWithWheelScroll covers a gap none of
// PresentationKey's other fields close: wireScrollWheel mutates a
// *widget.ScrollContainer's ScrollTop directly from the mouse wheel
// (pkg/hud/panel.go, pkg/hud/chips.go), with no other observable side
// effect — not a rebuild, not one of the three refresh paths, and no change
// to the cursor position, mouse buttons, or input.UIHovered. Without reading
// ScrollTop directly, a wheel-scroll-only tick would report the same key as
// the tick before it, and the map would wrongly skip repainting over newly
// scrolled chrome.
func TestPresentationKeyChangesWithWheelScroll(t *testing.T) {
	panel := New()
	state := testState(testFrame(40), 1)
	panel.Update(state)
	if panel.handles.panelMiddle == nil {
		t.Fatal("panel's scroll container handle missing")
	}

	key := panel.PresentationKey()
	panel.handles.panelMiddle.ScrollTop = 0.5
	if panel.PresentationKey() == key {
		t.Fatal("PresentationKey did not change when the panel's own ScrollTop changed")
	}

	panel.handles.panelMiddle.ScrollTop = 0
	state.BandListOpen = true
	panel.Update(state)
	if panel.handles.bandListScroll == nil {
		t.Fatal("band list scroll container handle missing once the list is open")
	}

	key = panel.PresentationKey()
	panel.handles.bandListScroll.ScrollTop = 0.5
	if panel.PresentationKey() == key {
		t.Fatal("PresentationKey did not change when the band list's own ScrollTop changed")
	}
}

// TestNotesBodyScrollsAndCountsInThePresentationKey covers F3: DESIGN.md
// says the drawer scrolls with the wheel, but the old widget.TextArea was
// built without ShowVerticalScrollbar, and ebitenui only wires its wheel
// handler in that case — so a note longer than the drawer could not be
// scrolled into view at all. The notes body is now a pkg/hud-owned
// ScrollContainer, so its ScrollTop must be reachable for PresentationKey
// exactly like panelMiddle and bandListScroll already are.
func TestNotesBodyScrollsAndCountsInThePresentationKey(t *testing.T) {
	frame := testFrame(1)
	panel := New()
	state := testState(frame, 1)
	state.NotesMode = NotesExpanded
	state.Note = render.FieldNote{Topic: "LONG NOTE", Introduction: strings.Repeat("word ", 400)}
	panel.Update(state)
	screen := ebiten.NewImage(1280, 720)
	defer screen.Deallocate()
	panel.Draw(screen)

	if panel.handles.notesScroll == nil {
		t.Fatal("notes scroll container handle missing")
	}

	key := panel.PresentationKey()
	panel.handles.notesScroll.ScrollTop = 0.5
	if panel.PresentationKey() == key {
		t.Fatal("PresentationKey did not change when the notes body's own ScrollTop changed")
	}

	drawerRect := image.Rectangle(panel.rect(mapLeft, mapBottom-drawerExpandedH-drawerTabH, mapRight-mapLeft, drawerExpandedH+drawerTabH))
	bodyRect := panel.handles.notesScroll.GetWidget().Rect
	if !bodyRect.In(drawerRect) {
		t.Fatalf("notes body rect %v not inside drawer rect %v", bodyRect, drawerRect)
	}
}

// TestHiddenBarControlMatchesTheBreakthroughAccent covers a reviewer-found
// cosmetic (F4): the hidden bar's "▲ notes · F" control hardcoded
// colorGoldDeep, so it stayed dim during a breakthrough while the bar's
// background and event text turned gold. Both buttons must share the bar's
// own accent. Button colors have no public getter, so this checks the
// theme's memoized border cache instead: with a fresh theme, only the
// colors buildHiddenDrawerBar actually asked for appear as border keys.
func TestHiddenBarControlMatchesTheBreakthroughAccent(t *testing.T) {
	panel := New()
	state := State{Note: render.FieldNote{Celebration: true}}
	panel.buildHiddenDrawerBar(state, nil)

	sawGoldDeepBorder, sawGoldBorder := false, false
	for key := range panel.theme.borders {
		switch key.border {
		case colorGoldDeep:
			sawGoldDeepBorder = true
		case colorGold:
			sawGoldBorder = true
		}
	}
	if sawGoldDeepBorder {
		t.Fatal("hidden bar control still uses colorGoldDeep during a breakthrough; it should match the bar's own gold accent")
	}
	if !sawGoldBorder {
		t.Fatal("hidden bar control does not use the breakthrough accent (colorGold)")
	}
}

// TestShortcutSheetDocumentsGlobalDAndWorkforceOverride covers a
// reviewer-found gap (F4): the ? shortcut sheet listed D only as the
// Workforce row's discard, never mentioning D's global meaning (toggle band
// details) or that A and D are row-owned instead while Workforce is open.
// The Workforce guidance now spans two lines (fix round 1: the combined
// line overflowed the window, see TestShortcutSheetLinesFitTheWindow), so
// this checks for the header line and the override note independently
// rather than assuming both live on one line.
func TestShortcutSheetDocumentsGlobalDAndWorkforceOverride(t *testing.T) {
	panel := New()
	var lines []string
	walkDescendants(panel.shortcutSheet(), func(w widget.PreferredSizeLocateableWidget) {
		if text, ok := w.(*widget.Text); ok {
			lines = append(lines, text.Label)
		}
	})

	var globalDLine string
	sawWorkforceHeader, sawWorkforceOverrideNote := false, false
	for _, line := range lines {
		if strings.Contains(line, "toggle band details") {
			globalDLine = line
		}
		if strings.HasPrefix(line, "Workforce row:") {
			sawWorkforceHeader = true
		}
		lower := strings.ToLower(line)
		if strings.Contains(line, "A") && strings.Contains(line, "D") && strings.Contains(lower, "belong") {
			sawWorkforceOverrideNote = true
		}
	}
	if globalDLine == "" {
		t.Fatalf("shortcut sheet does not document D's global meaning (toggle band details); lines = %v", lines)
	}
	if !sawWorkforceHeader {
		t.Fatalf("shortcut sheet lost its Workforce row line; lines = %v", lines)
	}
	if !sawWorkforceOverrideNote {
		t.Fatalf("shortcut sheet does not mention that A and D belong to the Workforce row while open; lines = %v", lines)
	}
}

// TestShortcutSheetLinesFitTheWindow covers a reviewer-found Critical
// defect: shortcutSheet's overlayFrame window is neither Dynamic nor
// Resizeable, and its body labels carry no Stretch/MaxWidth layout data, so
// ebitenui's RowLayout lays each line out at its own natural width — a line
// wider than the window's usable interior spills straight past its right
// edge. The Workforce row line this fix splits measured 897.2 DIP against a
// 544 DIP budget before the re-lay. This measures every line the sheet
// actually renders, with the same face shortcutSheet builds them at, against
// the same overlayWidth/insets constants the window is built from (not a
// duplicated literal), so it tracks any future change to either.
func TestShortcutSheetLinesFitTheWindow(t *testing.T) {
	panel := New()
	face := panel.theme.face(shortcutLineFontSizeDIP)
	for _, line := range shortcutSheetLines {
		width, _ := text.Measure(line, *face, 0)
		if width > overlayUsableWidthDIP {
			t.Fatalf("shortcut sheet line measures %.1f DIP, want <= %.1f DIP (overlayWidth %.0f minus insets %.0f+%.0f): %q",
				width, overlayUsableWidthDIP, float64(overlayWidth), float64(overlayInsetLeftDIP), float64(overlayInsetRightDIP), line)
		}
	}
}

// campaignControlTestFrame builds a one-band frame where every action
// control this test walks would be enabled absent CampaignOver: a
// non-passage migration candidate (Best/Split), an interbreed candidate
// (Interbreed), and one available, unacquired research option (Firecraft) —
// testFrame already sets the last of these. Every other research slot stays
// unavailable, so it is disabled either way; recorded separately as
// wantDisabledWithoutCampaignOver so the false-case assertion does not
// confuse "disabled because unavailable" with "disabled because the
// campaign ended".
func campaignControlTestFrame() *gameapi.Frame {
	frame := testFrame(1)
	frame.Bands[0].InterbreedCandidateIDs = []gameapi.BandID{2}
	// Nothing blocks the split, so that CampaignOver is the only thing this
	// fixture leaves disabling the Split button. Eligibility is the projected
	// Band.SplitBlock alone; Stress and Population no longer decide it.
	frame.Bands[0].SplitBlock = ""
	return frame
}

// TestCampaignOverDisablesEveryActionControl covers Wave I item I3: with the
// campaign over, the Move row's four buttons, every research technology
// button, and the Workforce row's sliders/−/+/Apply/Discard must all go
// dead — controls that look live and do nothing are worse than disabled
// ones. With the campaign ongoing, the same controls must be enabled
// wherever they would normally be (i.e. unaffected by CampaignOver).
func TestCampaignOverDisablesEveryActionControl(t *testing.T) {
	for _, campaignOver := range []bool{true, false} {
		frame := campaignControlTestFrame()

		// --- Move row ---
		panel := New()
		state := testState(frame, 1)
		state.OpenRow = ui.RowMove
		state.CampaignOver = campaignOver
		panel.Update(state)
		moveButtons := map[string]*widget.Button{
			"best": panel.handles.best, "split": panel.handles.split, "interbreed": panel.handles.interbreed,
		}
		for name, button := range moveButtons {
			if button == nil {
				t.Fatalf("campaignOver=%v: Move row button %q was not built", campaignOver, name)
			}
			got := button.GetWidget().Disabled
			want := true // done=false, readOnly=species-sapiens||campaignOver, candidates present: enabled only when campaign is not over
			if !campaignOver {
				want = false
			}
			if got != want {
				t.Fatalf("campaignOver=%v: Move row button %q disabled = %v, want %v", campaignOver, name, got, want)
			}
		}
		if moveHere := panel.handles.moveHere; moveHere == nil || !moveHere.GetWidget().Disabled {
			t.Fatalf("campaignOver=%v: Move here should stay disabled (no target chosen) regardless of CampaignOver", campaignOver)
		}

		// --- Research row ---
		panel = New()
		state = testState(frame, 1)
		state.OpenRow = ui.RowResearch
		state.CampaignOver = campaignOver
		panel.Update(state)
		for technology := gameapi.Tech(0); technology < gameapi.TechCount; technology++ {
			button := panel.handles.research[technology]
			if button == nil {
				t.Fatalf("campaignOver=%v: research button %d was not built", campaignOver, technology)
			}
			option := frame.Bands[0].ResearchOptions[technology]
			wantDisabledWithoutCampaignOver := option.Acquired || !option.Available
			want := true
			if !campaignOver {
				want = wantDisabledWithoutCampaignOver
			}
			if got := button.GetWidget().Disabled; got != want {
				t.Fatalf("campaignOver=%v: research button %d disabled = %v, want %v", campaignOver, technology, got, want)
			}
		}

		// --- Workforce row ---
		panel = New()
		state = testState(frame, 1)
		state.OpenRow = ui.RowWorkforce
		state.CampaignOver = campaignOver
		state.Workforce = WorkforceDraft{Visible: true, Population: 60, AllocationBP: frame.Bands[0].AllocationBP, Dirty: true, Valid: true}
		panel.Update(state)
		h := panel.handles.workforce
		for role := gameapi.WorkforceRole(0); role < gameapi.AssignmentCount; role++ {
			if h.sliders[role] == nil || h.minus[role] == nil || h.plus[role] == nil {
				t.Fatalf("campaignOver=%v: workforce role %d controls were not built", campaignOver, role)
			}
			if got := h.sliders[role].GetWidget().Disabled; got != campaignOver {
				t.Fatalf("campaignOver=%v: workforce slider %d disabled = %v, want %v", campaignOver, role, got, campaignOver)
			}
			if got := h.minus[role].GetWidget().Disabled; got != campaignOver {
				t.Fatalf("campaignOver=%v: workforce − %d disabled = %v, want %v", campaignOver, role, got, campaignOver)
			}
			if got := h.plus[role].GetWidget().Disabled; got != campaignOver {
				t.Fatalf("campaignOver=%v: workforce + %d disabled = %v, want %v", campaignOver, role, got, campaignOver)
			}
		}
		if h.apply == nil || h.discard == nil {
			t.Fatalf("campaignOver=%v: Apply/Discard were not built", campaignOver)
		}
		if got := h.apply.GetWidget().Disabled; got != campaignOver {
			t.Fatalf("campaignOver=%v: Apply disabled = %v, want %v", campaignOver, got, campaignOver)
		}
		if got := h.discard.GetWidget().Disabled; got != campaignOver {
			t.Fatalf("campaignOver=%v: Discard disabled = %v, want %v", campaignOver, got, campaignOver)
		}
	}
}

// TestMoveGridValuesFitTheirColumns guards the widened HERE / TARGET values:
// Capacity now carries capacity, occupancy and degradation on one line, and
// the worst case lands within a couple of DIP of the column. A widget.Text
// reports its measured text as its PreferredSize, so comparing that with the
// laid-out rect catches an overflow at any scale without this test needing to
// know the font or redo the column arithmetic.
func TestMoveGridValuesFitTheirColumns(t *testing.T) {
	for _, scale := range []float64{1, 2} {
		frame := testFrame(1)
		// Worst case for every row at once: a capacity-100 target that is both
		// half degraded and over-full, with four-digit crowding and two-digit
		// mortality percentages.
		frame.Tiles[1].EcologicalK, frame.Tiles[1].Degradation = 100, 0.5
		frame.Tiles[1].NearbyLake = "Lake Victoria"
		frame.Tiles[1].FloraStock, frame.Tiles[1].FloraCap = 6_408, 8_100
		frame.Tiles[1].WaterStock, frame.Tiles[1].WaterCap = 4_800, 5_000
		frame.Bands[0].LastFoodReport = gameapi.FoodTurnReport{Turn: 12, RequiredFU: 300}
		frame.Bands[0].MigrationCandidates = []gameapi.MigrationCandidate{
			{TileID: 1, SeasonalMortalityRate: 0.0123, ChronicMortalityRate: 0.0456, CrowdingDecline: 120},
		}
		frame.Bands = append(frame.Bands, gameapi.Band{ID: 7, Species: gameapi.ArchaicHominin, Population: 95, TileID: 1})

		panel := New()
		state := testState(frame, scale)
		state.Hover = render.TileHover{TileID: 1, Visible: true}
		panel.Update(state)
		screen := ebiten.NewImage(int(1280*scale), int(720*scale))
		panel.Draw(screen)
		screen.Deallocate()

		// The cell container, not the labels inside it, is what the grid
		// stretches to the column width: measuring a label against its own
		// rect compares a number with itself and can never fail.
		for index, cell := range panel.handles.moveTargetCells {
			if cell == nil {
				continue
			}
			wanted, _ := cell.PreferredSize()
			if available := cell.GetWidget().Rect.Dx(); wanted > available {
				t.Fatalf("scale %.1f: target cell %d %q%q needs %d px in a %d px column",
					scale, index, panel.handles.moveTargetValues[index].Label, panel.handles.moveTargetMarks[index].Label, wanted, available)
			}
		}
	}
}

// A ▲▼ mark earns a warning colour only when the difference it reports has
// consequences. Between two comfortable tiles the mark stays — it still ranks
// them — but drops to the dim colour, because colouring an inconsequential
// difference red cried wolf (user-reported).
func TestDeltaMarkDimsAnInconsequentialDifference(t *testing.T) {
	if mark, got := deltaMark(ui.LiveabilityRow{Delta: 1, DeltaMaterial: true}); mark != " ▲" || got != colorGreen {
		t.Fatalf("material improvement = %q, %v, want \" ▲\" green", mark, got)
	}
	if mark, got := deltaMark(ui.LiveabilityRow{Delta: -1, DeltaMaterial: true}); mark != " ▼" || got != colorRed {
		t.Fatalf("material regression = %q, %v, want \" ▼\" red", mark, got)
	}
	if mark, got := deltaMark(ui.LiveabilityRow{Delta: -1}); mark != " ▼" || got != colorDim {
		t.Fatalf("inconsequential regression = %q, %v, want \" ▼\" dim", mark, got)
	}
	if mark, got := deltaMark(ui.LiveabilityRow{Delta: 1}); mark != " ▲" || got != colorDim {
		t.Fatalf("inconsequential improvement = %q, %v, want \" ▲\" dim", mark, got)
	}
	if mark, _ := deltaMark(ui.LiveabilityRow{}); mark != "" {
		t.Fatalf("no difference = %q, want no mark", mark)
	}
}

// The comparison mark lives in its own label beside the value so the two can
// carry different colours (see moveTargetCell). buildMoveBody and
// refreshTarget populate those labels through separate code, so both paths are
// exercised here: a divergence would show up as a mark appended to the value's
// own text, or as a stale mark left behind when the target changes.
func TestTargetValueAndMarkAreSeparateLabels(t *testing.T) {
	// Tile 1 holds more food and water than tile 0, so some rows earn a mark.
	// Neither tile is short of anything, so those marks are inconsequential —
	// exactly the case that must still produce a mark, just a dim one.
	assertSplit := func(t *testing.T, panel *Panel, path string) {
		t.Helper()
		marks := 0
		for index, value := range panel.handles.moveTargetValues {
			if value == nil {
				continue
			}
			if strings.ContainsAny(value.Label, "▲▼") {
				t.Fatalf("%s: cell %d value %q carries the mark; it belongs in its own label", path, index, value.Label)
			}
			if mark := panel.handles.moveTargetMarks[index].Label; mark != "" {
				if mark != " ▲" && mark != " ▼" {
					t.Fatalf("%s: cell %d mark = %q", path, index, mark)
				}
				marks++
			}
		}
		if marks == 0 {
			t.Fatalf("%s: no row earned a mark, so this test proves nothing", path)
		}
	}

	panel := New()
	state := testState(testFrame(1), 1)
	state.Hover = render.TileHover{TileID: 1, Visible: true}
	panel.Update(state) // the build path: the hover is present before the first build
	assertSplit(t, panel, "build")

	// The Food row compares 640 against 212 with neither tile short: a mark,
	// but a dim one, and the value beside it stays untouched.
	if value, mark := panel.handles.moveTargetValues[3], panel.handles.moveTargetMarks[3]; value.Label != "640 / 810" || mark.Label != " ▲" {
		t.Fatalf("food cell = %q + %q, want \"640 / 810\" + \" ▲\"", value.Label, mark.Label)
	}

	builds := panel.builds
	state.Hover = render.TileHover{}
	panel.Update(state) // the refresh path
	if panel.builds != builds {
		t.Fatalf("clearing the hover rebuilt the tree: builds %d -> %d", builds, panel.builds)
	}
	for index, mark := range panel.handles.moveTargetMarks {
		if mark != nil && mark.Label != "" {
			t.Fatalf("cell %d kept the mark %q after the target went away", index, mark.Label)
		}
	}

	state.Hover = render.TileHover{TileID: 1, Visible: true}
	panel.Update(state)
	if panel.builds != builds {
		t.Fatalf("restoring the hover rebuilt the tree: builds %d -> %d", builds, panel.builds)
	}
	assertSplit(t, panel, "refresh")
}

// The Split button used to be disabled only by a spent action, so a band that
// the domain would refuse — not crowded enough, too small, nowhere adjacent to
// settle — still offered the action and answered a click with a notice
// (user-reported). The button now follows the projected domain rejection.
func TestSplitButtonFollowsSplitEligibility(t *testing.T) {
	build := func(t *testing.T, mutate func(*gameapi.Frame)) *Panel {
		t.Helper()
		frame := testFrame(1)
		mutate(frame)
		panel := New()
		panel.Update(testState(frame, 1))
		return panel
	}
	// Stated rather than left to the zero value, so a future non-empty default
	// for SplitBlock cannot silently keep enabling Split here.
	eligible := func(f *gameapi.Frame) { f.Bands[0].SplitBlock = "" }
	if panel := build(t, eligible); panel.handles.split.GetWidget().Disabled {
		t.Fatal("Split is disabled for a band the domain would allow to split")
	}
	for name, mutate := range map[string]func(*gameapi.Frame){
		"not crowded enough": func(f *gameapi.Frame) {
			eligible(f)
			f.Bands[0].SplitBlock = gameapi.ErrSplitStressTooLow
		},
		"too few people": func(f *gameapi.Frame) {
			eligible(f)
			f.Bands[0].SplitBlock = gameapi.ErrSplitPopulationTooLow
		},
		"nowhere adjacent to settle": func(f *gameapi.Frame) {
			eligible(f)
			f.Bands[0].MigrationCandidates = []gameapi.MigrationCandidate{{TileID: 1, RequiresPassage: true}}
			f.Bands[0].SplitBlock = gameapi.ErrSplitDestinationNotAdjacent
		},
	} {
		t.Run(name, func(t *testing.T) {
			if !build(t, mutate).handles.split.GetWidget().Disabled {
				t.Fatal("Split is offered for a band the domain would refuse")
			}
		})
	}
}

// Best tile carries the same defect one button over: moveToBestTile takes the
// first candidate that needs no passage, but the button only checked that the
// candidate list was non-empty, so a band whose only routes are passage
// crossings was offered the action and answered with a notice.
func TestBestTileButtonRequiresAnOrdinaryLandCandidate(t *testing.T) {
	build := func(candidates []gameapi.MigrationCandidate) *Panel {
		frame := testFrame(1)
		frame.Bands[0].MigrationCandidates = candidates
		panel := New()
		panel.Update(testState(frame, 1))
		return panel
	}
	if !build([]gameapi.MigrationCandidate{{TileID: 1, RequiresPassage: true}}).handles.best.GetWidget().Disabled {
		t.Fatal("Best tile is offered when every candidate requires a passage")
	}
	if build([]gameapi.MigrationCandidate{{TileID: 1}}).handles.best.GetWidget().Disabled {
		t.Fatal("Best tile is disabled despite an ordinary-land candidate")
	}
}

// Spec §4.1 asks disabled buttons to explain themselves on hover, which was
// never built: a control went dead with no way to find out why. The reason
// shown is the same string that disabled the button, so the two cannot
// disagree.
func TestMoveRowButtonsExplainWhyTheyAreDisabled(t *testing.T) {
	frame := testFrame(1) // Stress 0, no interbreed candidate, no target chosen
	panel := New()
	panel.Update(testState(frame, 1))
	band := &frame.Bands[0]
	want := ui.DiagnoseMoveActions(frame, band, ui.TileLiveability{}, ui.TargetNone, 0)

	buttons := [moveActionCount]*widget.Button{panel.handles.moveHere, panel.handles.best, panel.handles.split, panel.handles.interbreed}
	reasons := [moveActionCount]string{want.MoveHere, want.BestTile, want.Split, want.Interbreed}
	blocked := 0
	for action, button := range buttons {
		reason, tooltips := reasons[action], button.GetWidget().ToolTips
		if reason == "" {
			if len(tooltips) != 0 {
				t.Fatalf("action %d is available but carries %d tooltips", action, len(tooltips))
			}
			if button.GetWidget().Disabled {
				t.Fatalf("action %d is available but its button is disabled", action)
			}
			continue
		}
		blocked++
		if !button.GetWidget().Disabled {
			t.Fatalf("action %d is blocked (%q) but its button is live", action, reason)
		}
		if len(tooltips) != 1 {
			t.Fatalf("action %d is blocked (%q) but carries %d tooltips, want 1", action, reason, len(tooltips))
		}
		if got := panel.handles.moveTooltips[action]; got == nil || got.Label != reason {
			t.Fatalf("action %d tooltip = %v, want %q", action, got, reason)
		}
	}
	if blocked == 0 {
		t.Fatal("no action was blocked, so this test proves nothing")
	}
}

// A tooltip becomes visible precisely when nothing else about the frame has
// changed: it waits for the cursor to hold still, so at the moment it appears
// the cursor position, button state and hover flag in PresentationKey are all
// unchanged from the previous frame. Production disables Ebitengine's
// automatic screen clear and repaints only when the map key or this key
// changes, so a tooltip missing from the key is a tooltip that never gets
// drawn — the same defect shape as the stale chrome regions found by playing.
func TestPresentationKeyChangesWhenATooltipAppears(t *testing.T) {
	panel, _, _, armed := showSplitTooltip(t)
	if shown := panel.PresentationKey(); shown == armed {
		t.Fatalf("PresentationKey is unchanged (%+v) after a tooltip appeared, so the frame would not be repainted", shown)
	}
}

// showSplitTooltip hovers the disabled Split button until its explanation is
// on screen, and returns the panel, its screen, and the key from the last
// frame before the tooltip appeared. Real time has to pass: ebitenui arms the
// tooltip with a time.AfterFunc, so nothing but the wall clock advances it.
func showSplitTooltip(t *testing.T) (*Panel, *ebiten.Image, State, PresentationKey) {
	t.Helper()
	t.Cleanup(func() { input.SetCursorUpdater(nil) })
	panel := New()
	state := testState(testFrame(1), 1)
	state.Frame.Bands[0].SplitBlock = gameapi.ErrSplitStressTooLow
	screen := ebiten.NewImage(1280, 720)
	t.Cleanup(screen.Deallocate)
	panel.Update(state)
	panel.Draw(screen)
	if len(panel.handles.split.GetWidget().ToolTips) != 1 {
		t.Fatal("Split has no tooltip, so this test cannot show one")
	}

	rect := panel.handles.split.GetWidget().Rect
	input.SetCursorUpdater(&testCursor{x: (rect.Min.X + rect.Max.X) / 2, y: (rect.Min.Y + rect.Max.Y) / 2})
	// Two ticks to reach the armed state and start its delay timer.
	for range 2 {
		panel.Update(state)
		panel.Draw(screen)
	}
	armed := panel.PresentationKey()

	// ebitenui applies tooltipDelay against the wall clock, but only an Update
	// tick can notice it has elapsed and announce the tooltip. Sleeping out the
	// delay and then ticking a fixed number of times asserts the tooltip
	// appears inside one exact window, and a loaded machine misses it -- that
	// cost a Windows CI job, which then passed on a re-run of the same commit.
	// Tick until it actually shows instead. On an unloaded machine this returns
	// sooner than the old sleep did, because it stops the moment the tooltip
	// appears rather than always waiting out the delay plus a margin.
	deadline := time.Now().Add(3 * time.Second)
	for time.Now().Before(deadline) {
		panel.Update(state)
		panel.Draw(screen)
		if panel.tooltipShown {
			return panel, screen, state, armed
		}
		time.Sleep(time.Millisecond)
	}
	t.Fatal("the tooltip never became visible")
	return nil, nil, State{}, PresentationKey{}
}

// The Move row sits against the right edge of the presentation, so a tooltip
// that opened at the cursor (ebitenui's default) or extended rightward would
// run off screen. It opens below its button and right-aligned to it instead.
func TestTooltipStaysOnScreen(t *testing.T) {
	panel, screen, _, _ := showSplitTooltip(t)
	label := panel.handles.moveTooltips[moveActionSplit]
	if label == nil {
		t.Fatal("no tooltip label handle for the disabled Split button")
	}
	rect := label.GetWidget().Rect
	if rect.Empty() {
		t.Fatal("the tooltip label was never laid out")
	}
	if bounds := screen.Bounds(); !rect.In(bounds) {
		t.Fatalf("tooltip text at %v is not inside the screen %v", rect, bounds)
	}
}

// Hiding it must repaint too: the tooltip covers map and panel pixels, and
// with the automatic screen clear disabled they would otherwise stay behind
// as a ghost after the pointer moved away.
func TestPresentationKeyChangesWhenATooltipDisappears(t *testing.T) {
	panel, screen, state, _ := showSplitTooltip(t)
	shown := panel.PresentationKey()

	// Move the pointer off the button; the tooltip hides on the next tick.
	input.SetCursorUpdater(&testCursor{x: 1, y: 1})
	panel.Update(state)
	panel.Draw(screen)
	if panel.tooltipShown {
		t.Fatal("the tooltip is still shown after the pointer left the button")
	}
	if hidden := panel.PresentationKey(); hidden == shown {
		t.Fatalf("PresentationKey is unchanged (%+v) after the tooltip disappeared, so its pixels would stay on screen", hidden)
	}
}

// A rebuild replaces the buttons that would have reported their tooltips
// hiding, so a visible one has to be forgotten along with them. Left set, the
// next tooltip to appear would leave PresentationKey unchanged and never be
// drawn — the defect TestPresentationKeyChangesWhenATooltipAppears covers,
// returning by a different route.
func TestRebuildForgetsAVisibleTooltip(t *testing.T) {
	panel, screen, _, _ := showSplitTooltip(t)
	// A different Frame pointer is a structural change, so the tree rebuilds.
	panel.Update(testState(testFrame(2), 1))
	panel.Draw(screen)
	if panel.tooltipShown {
		t.Fatal("a rebuild kept a visible tooltip that its widgets can no longer hide")
	}
}
