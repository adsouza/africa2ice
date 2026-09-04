package hud

import (
	"fmt"
	"image"
	"strings"
	"testing"

	"github.com/adsouza/africa2ice/pkg/gameapi"
	"github.com/adsouza/africa2ice/pkg/render"
	"github.com/adsouza/africa2ice/pkg/ui"
	"github.com/ebitenui/ebitenui/input"
	"github.com/ebitenui/ebitenui/widget"
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
	if panel.handles.drawerTab.Text().Label != "▲ notes · F  ·  T12 · Migration · Band 1 migrated again" {
		t.Fatalf("hidden tab = %q, want newest event retained", panel.handles.drawerTab.Text().Label)
	}
	if panel.handles.drawerMore != nil || len(panel.handles.events) != 0 {
		t.Fatal("hidden drawer still shows body controls")
	}
}

func TestHiddenTabTruncatesLongEventsAndShowsBreakthrough(t *testing.T) {
	frame := testFrame(1)
	frame.Events = []gameapi.Event{{Turn: 12, Kind: gameapi.EventMigration, Summary: strings.Repeat("x", 70)}}
	panel := New()
	state := testState(frame, 1)
	state.NotesMode = NotesHidden
	panel.Update(state)
	label := panel.handles.drawerTab.Text().Label
	parts := strings.SplitN(label, "  ·  ", 2)
	if len(parts) != 2 || !strings.HasSuffix(parts[1], "…") || len([]rune(parts[1])) != 42 {
		t.Fatalf("hidden tab with long event = %q, want a 42-rune truncated event part ending in an ellipsis", label)
	}
	state.Note.Celebration = true
	panel.Update(state)
	if got := panel.handles.drawerTab.Text().Label; !strings.HasPrefix(got, "BREAKTHROUGH · ") {
		t.Fatalf("celebration hidden tab = %q, want the BREAKTHROUGH prefix", got)
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
	state.Camera = CameraState{FocusAvailable: true, Focused: true}
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
		t.Fatal("camera button shown when focus is unavailable")
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
// the Field Notes drawer's TextArea (and the reader's scroll position with
// it) on every tick.
func TestHoverRefreshesTargetWithoutRebuilding(t *testing.T) {
	frame := testFrame(1)
	panel := New()
	state := testState(frame, 1)
	panel.Update(state)

	buildsBefore := panel.builds
	notesAreaBefore := panel.handles.notesArea
	if notesAreaBefore == nil {
		t.Fatal("drawer TextArea handle missing before the hover change")
	}
	if got := panel.handles.moveTargetHeader.Label; got != "TARGET" {
		t.Fatalf("TARGET header before hover = %q, want plain TARGET", got)
	}

	state.Hover = render.TileHover{TileID: 1, Visible: true}
	panel.Update(state)

	if panel.builds != buildsBefore {
		t.Fatalf("hover change rebuilt the tree: builds %d -> %d", buildsBefore, panel.builds)
	}
	if panel.handles.notesArea != notesAreaBefore {
		t.Fatal("hover change replaced the drawer's TextArea, losing the reader's scroll position")
	}
	if got := panel.handles.moveTargetHeader.Label; got != "TARGET · hover" {
		t.Fatalf("TARGET header while hovering = %q, want TARGET · hover", got)
	}
	if got := panel.handles.moveTargetValues[0].Label; got != frame.Tiles[1].Biome.String() {
		t.Fatalf("TARGET biome cell = %q, want the hovered tile's biome %q", got, frame.Tiles[1].Biome.String())
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
