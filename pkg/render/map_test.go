package render

import (
	"image"
	"image/color"
	"reflect"
	"strings"
	"testing"

	"github.com/adsouza/africa2ice/pkg/gameapi"
	"github.com/hajimehoshi/ebiten/v2"
)

func TestMapTileAt(t *testing.T) {
	tests := []struct {
		name   string
		x      int
		y      int
		wantID gameapi.TileID
		wantOK bool
	}{
		{name: "top-left pixel", x: mapOriginX, y: mapOriginY, wantID: 0, wantOK: true},
		{name: "inside second tile", x: mapOriginX + mapTileSize + 4, y: mapOriginY + 3, wantID: 1, wantOK: true},
		{name: "bottom-right pixel", x: mapOriginX + 96*mapTileSize - 1, y: mapOriginY + 64*mapTileSize - 1, wantID: 6_143, wantOK: true},
		{name: "left of map", x: mapOriginX - 1, y: mapOriginY, wantOK: false},
		{name: "above map", x: mapOriginX, y: mapOriginY - 1, wantOK: false},
		{name: "right half-open edge", x: mapOriginX + 96*mapTileSize, y: mapOriginY, wantOK: false},
		{name: "bottom half-open edge", x: mapOriginX, y: mapOriginY + 64*mapTileSize, wantOK: false},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			gotID, gotOK := MapTileAt(test.x, test.y)
			if gotID != test.wantID || gotOK != test.wantOK {
				t.Fatalf("MapTileAt(%d, %d) = (%d, %t), want (%d, %t)", test.x, test.y, gotID, gotOK, test.wantID, test.wantOK)
			}
		})
	}
}

func TestFormatHealthDelta(t *testing.T) {
	tests := []struct {
		name   string
		points float64
		want   string
	}{
		{name: "zero keeps precision", points: 0, want: "+0.00pp"},
		{name: "small gain", points: 0.054, want: "+0.05pp"},
		{name: "small loss", points: -0.054, want: "-0.05pp"},
		{name: "threshold", points: 0.1, want: "+0.1pp"},
		{name: "large loss", points: -2.36, want: "-2.4pp"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if got := formatHealthDelta(test.points); got != test.want {
				t.Fatalf("formatHealthDelta(%v) = %q, want %q", test.points, got, test.want)
			}
		})
	}
}

func TestSelectedBandInFrame(t *testing.T) {
	frame := &gameapi.Frame{Bands: []gameapi.Band{{ID: 3, Population: 30}, {ID: 8, Population: 80}}}
	tests := []struct {
		name       string
		id         gameapi.BandID
		wantFound  bool
		wantPeople uint32
	}{
		{name: "first", id: 3, wantFound: true, wantPeople: 30},
		{name: "later", id: 8, wantFound: true, wantPeople: 80},
		{name: "missing", id: 99, wantFound: false},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got := selectedBandInFrame(frame, test.id)
			if (got != nil) != test.wantFound {
				t.Fatalf("selectedBandInFrame(%d) found = %t, want %t", test.id, got != nil, test.wantFound)
			}
			if got != nil && got.Population != test.wantPeople {
				t.Fatalf("selectedBandInFrame(%d).Population = %d, want %d", test.id, got.Population, test.wantPeople)
			}
		})
	}

	selectedBandInFrame(frame, 8).Population = 81
	if frame.Bands[1].Population != 81 {
		t.Fatal("selected band pointer does not refer to the frame's band")
	}
}

func TestClampRender(t *testing.T) {
	tests := []struct {
		name  string
		value float64
		want  float64
	}{
		{name: "below", value: -0.2, want: 0},
		{name: "lower boundary", value: 0, want: 0},
		{name: "interior", value: 0.45, want: 0.45},
		{name: "upper boundary", value: 1, want: 1},
		{name: "above", value: 1.2, want: 1},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if got := clampRender(test.value); got != test.want {
				t.Fatalf("clampRender(%v) = %v, want %v", test.value, got, test.want)
			}
		})
	}
}

func TestOverlayAndFieldNotesHitTargets(t *testing.T) {
	for row, y := range []int{228, 264, 300} {
		if got := MenuOverlayRowAt(500, y, 3); got != row {
			t.Fatalf("overlay row at y=%d = %d, want %d", y, got, row)
		}
	}
	if MenuOverlayRowAt(100, 228, 3) != -1 || MenuOverlayRowAt(500, 336, 3) != -1 {
		t.Fatal("overlay accepted a point outside its rows")
	}
	if !FieldNotesToggleContains(1200, 90) || FieldNotesToggleContains(900, 90) {
		t.Fatal("Field Notes top-bar hit target is inconsistent")
	}
	if !FieldNotesPanelContains(1000, 500) || FieldNotesPanelContains(1000, 620) {
		t.Fatal("Field Notes scroll hit target is inconsistent")
	}
}

func TestFieldNoteWrappingPreservesWordsAndBoundsLines(t *testing.T) {
	value := "Historical context uses several words that need wrapping.\nHint remains separate."
	lines := wrapTextLines(value, 24)
	if len(lines) < 3 || strings.Join(lines, " ") != strings.ReplaceAll(value, "\n", " ") {
		t.Fatalf("wrapped lines = %#v", lines)
	}
	for _, line := range lines {
		if len([]rune(line)) > 24 {
			t.Fatalf("overlong wrapped line %q", line)
		}
	}
}

func TestMapSceneDrawsTerrainVisibilityBandsAndPassagesOffscreen(t *testing.T) {
	frame := representativeRenderFrame()
	wantFrame := cloneRenderFrame(frame)
	screen := renderMapOffscreen(t, frame, 7, MigrationPreview{}, FieldNote{}, false, EndScene{}, "")
	defer screen.Deallocate()
	if !reflect.DeepEqual(frame, wantFrame) {
		t.Fatal("drawing mutated the accepted frame")
	}

	if got := tileColorForRender(frame.Tiles[1], frame.Climate.AridityIndex); got != EpochGrade(frame.Climate.AridityIndex).Water {
		t.Fatalf("water render color = %v", got)
	}
	if got := tileColorForRender(frame.Tiles[2], frame.Climate.AridityIndex); got != unexploredTileColor {
		t.Fatalf("unexplored render color = %v", got)
	}
	if got := tileColorForRender(frame.Tiles[0], frame.Climate.AridityIndex); got != climateBiomeColor(gameapi.Savanna, 0.4) {
		t.Fatalf("savanna render color = %v", got)
	}
	if got, visible := bandColorForRender(frame, frame.Bands[0]); !visible || got != (color.RGBA{R: 245, G: 202, B: 92, A: 255}) {
		t.Fatalf("sapiens marker = (%v, %t)", got, visible)
	}
	if got, visible := bandColorForRender(frame, frame.Bands[1]); !visible || got != archaicBandMarkerColor {
		t.Fatalf("visible archaic marker = (%v, %t)", got, visible)
	}
	if got, visible := bandColorForRender(frame, frame.Bands[2]); visible || got != (color.RGBA{}) {
		t.Fatalf("hidden archaic marker = (%v, %t)", got, visible)
	}
	if got, visible := passageColorForRender(frame, frame.Passages[0]); !visible || got != (color.RGBA{R: 203, G: 172, B: 104, A: 210}) {
		t.Fatalf("open passage = (%v, %t)", got, visible)
	}
	if got, visible := passageColorForRender(frame, frame.Passages[1]); !visible || got != (color.RGBA{R: 124, G: 111, B: 101, A: 180}) {
		t.Fatalf("locked passage = (%v, %t)", got, visible)
	}
	if _, visible := passageColorForRender(frame, frame.Passages[2]); visible {
		t.Fatal("out-of-range passage rendered")
	}
	if first, later := reachableTileColor(0), reachableTileColor(1); first == later || first != (color.RGBA{R: 245, G: 202, B: 92, A: 255}) || later != (color.RGBA{R: 87, G: 211, B: 211, A: 255}) {
		t.Fatalf("reachable colors = first %v, later %v", first, later)
	}
}

func TestMapSceneDrawsEscarpmentBoundaryOffscreen(t *testing.T) {
	frame := representativeRenderFrame()
	frame.Tiles[1] = gameapi.Tile{ID: 1, X: 1, Y: 0, Land: true, Explored: true, Biome: gameapi.MountainousHighlands}
	frame.Escarpments = []gameapi.Escarpment{{Name: "Test Front", First: 0, Second: 1}}
	wantFrame := cloneRenderFrame(frame)
	image := renderMapOffscreen(t, frame, 7, MigrationPreview{}, FieldNote{}, false, EndScene{}, "")
	defer image.Deallocate()
	if !reflect.DeepEqual(frame, wantFrame) {
		t.Fatal("drawing the escarpment mutated the accepted frame")
	}
	if escarpmentColor.R < 160 || escarpmentColor.G < 80 || escarpmentColor.B > 130 {
		t.Fatalf("escarpment color = %v, want visible ochre", escarpmentColor)
	}

	boundaryX := mapOriginX + mapTileSize
	fromX, fromY, toX, toY, ok := escarpmentLine(frame.Tiles[0], frame.Tiles[1])
	if !ok || fromX != float32(boundaryX) || toX != float32(boundaryX) || fromY != mapOriginY || toY != mapOriginY+mapTileSize {
		t.Fatalf("escarpment line = (%.0f,%.0f)-(%.0f,%.0f), %t", fromX, fromY, toX, toY, ok)
	}
}

func TestMapSceneCachesTerrainByTerrainRevisionAndAridity(t *testing.T) {
	frame := representativeRenderFrame()
	screen := ebiten.NewImage(1280, 720)
	defer screen.Deallocate()
	scene := NewMapScene()

	scene.Draw(screen, frame, 7, MigrationPreview{}, "", FieldNote{}, false, EndScene{}, false)
	scene.Draw(screen, frame, 7, MigrationPreview{}, "", FieldNote{}, false, EndScene{}, false)
	if scene.terrainRebuilds != 1 {
		t.Fatalf("terrain rebuilds for unchanged accepted frame = %d, want 1", scene.terrainRebuilds)
	}
	if !scene.frameCached || scene.frameKey.frame != frame {
		t.Fatal("complete immutable presentation frame was not cached")
	}

	nextFrame := cloneRenderFrame(frame)
	nextFrame.WorldRevision++
	scene.Draw(screen, nextFrame, 7, MigrationPreview{}, "", FieldNote{}, false, EndScene{}, false)
	if scene.terrainRebuilds != 1 {
		t.Fatalf("terrain rebuilds after planning-only frame replacement = %d, want 1", scene.terrainRebuilds)
	}

	nextFrame = cloneRenderFrame(nextFrame)
	nextFrame.TerrainRevision++
	scene.Draw(screen, nextFrame, 7, MigrationPreview{}, "", FieldNote{}, false, EndScene{}, false)
	if scene.terrainRebuilds != 2 {
		t.Fatalf("terrain rebuilds after terrain revision = %d, want 2", scene.terrainRebuilds)
	}

	nextFrame = cloneRenderFrame(nextFrame)
	nextFrame.Climate.AridityIndex += 0.1
	scene.Draw(screen, nextFrame, 7, MigrationPreview{}, "", FieldNote{}, false, EndScene{}, false)
	if scene.terrainRebuilds != 3 {
		t.Fatalf("terrain rebuilds after water-grade change = %d, want 3", scene.terrainRebuilds)
	}
}

func TestTopDownTerrainKeepsColorPickingAndMarkersOnTheSameGrid(t *testing.T) {
	frame := representativeRenderFrame()
	frame.Tiles[5].Biome = gameapi.MountainousHighlands
	scene := NewMapScene()

	for _, test := range []struct {
		name string
		tile gameapi.Tile
		want color.RGBA
	}{
		{name: "revealed highland", tile: frame.Tiles[5], want: climateBiomeColor(gameapi.MountainousHighlands, frame.Climate.AridityIndex)},
		{name: "unexplored", tile: frame.Tiles[2], want: unexploredTileColor},
	} {
		t.Run(test.name, func(t *testing.T) {
			localX := test.tile.X*mapTileSize + mapTileSize/2
			localY := test.tile.Y*mapTileSize + mapTileSize/2
			got := tileColorForRender(test.tile, frame.Climate.AridityIndex)
			if got != test.want {
				t.Fatalf("tile %d color = %v, want %v", test.tile.ID, got, test.want)
			}
			pointX, pointY := scene.tilePoint(test.tile)
			if pointX != float32(mapOriginX+localX) || pointY != float32(mapOriginY+localY) {
				t.Fatalf("tile %d marker point = (%v,%v), want (%d,%d)", test.tile.ID, pointX, pointY, mapOriginX+localX, mapOriginY+localY)
			}
			picked, ok := scene.PickTile(mapOriginX+localX, mapOriginY+localY)
			wantID := gameapi.TileID(test.tile.Y*TerrainGridWidth + test.tile.X)
			if !ok || picked != wantID {
				t.Fatalf("tile %d pick = (%d,%t), want (%d,true)", test.tile.ID, picked, ok, wantID)
			}
		})
	}
}

func TestMapSceneBuildsPhysicalPresentationForAHighDPITarget(t *testing.T) {
	frame := representativeRenderFrame()
	highDPI := ebiten.NewImage(2560, 1440)
	defer highDPI.Deallocate()
	scene := NewMapScene()
	scene.Draw(highDPI, frame, 7, MigrationPreview{}, "", FieldNote{}, false, EndScene{}, false)
	if scene.frameWidth != 2560 || scene.frameHeight != 1440 {
		t.Fatalf("cached target dimensions = %dx%d", scene.frameWidth, scene.frameHeight)
	}
	if scene.frameImage == nil || scene.frameImage.Bounds() != image.Rect(0, 0, 2560, 1440) || scene.frameScale != 2 {
		t.Fatalf("physical presentation image = %v at scale %v", scene.frameImage, scene.frameScale)
	}
	if scene.terrainImage == nil || scene.terrainImage.Bounds() != image.Rect(0, 0, mapPixelWidth*2, mapPixelHeight*2) || scene.terrainScale != 2 {
		t.Fatalf("physical terrain image = %v at scale %v", scene.terrainImage, scene.terrainScale)
	}
}

func TestMapSceneRebuildsTerrainOnlyWhenPhysicalScaleChanges(t *testing.T) {
	frame := representativeRenderFrame()
	scene := NewMapScene()
	standard := ebiten.NewImage(1280, 720)
	defer standard.Deallocate()
	highDPI := ebiten.NewImage(2560, 1440)
	defer highDPI.Deallocate()

	scene.Draw(standard, frame, 7, MigrationPreview{}, "", FieldNote{}, false, EndScene{}, false)
	if scene.terrainRebuilds != 1 || scene.terrainScale != 1 {
		t.Fatalf("standard terrain cache = %d rebuilds at scale %v", scene.terrainRebuilds, scene.terrainScale)
	}
	scene.Draw(highDPI, frame, 7, MigrationPreview{}, "", FieldNote{}, false, EndScene{}, false)
	if scene.terrainRebuilds != 2 || scene.terrainScale != 2 {
		t.Fatalf("high-DPI terrain cache = %d rebuilds at scale %v", scene.terrainRebuilds, scene.terrainScale)
	}
	scene.Draw(highDPI, frame, 7, MigrationPreview{}, "", FieldNote{}, false, EndScene{}, false)
	if scene.terrainRebuilds != 2 {
		t.Fatalf("unchanged high-DPI draw rebuilt terrain %d times", scene.terrainRebuilds)
	}
}

func TestMapSceneDrawsQueuedAndPreviewMigrationsOffscreen(t *testing.T) {
	baseFrame := representativeRenderFrame()
	base := renderMapOffscreen(t, baseFrame, 7, MigrationPreview{}, FieldNote{}, false, EndScene{}, "")
	defer base.Deallocate()

	queuedFrame := cloneRenderFrame(baseFrame)
	queuedFrame.Bands[0].HasQueuedMigration = true
	queuedFrame.Bands[0].QueuedMigration = 3
	queued := renderMapOffscreen(t, queuedFrame, 7, MigrationPreview{}, FieldNote{}, false, EndScene{}, "")
	defer queued.Deallocate()
	if summary := targetTileSummary(queuedFrame, &queuedFrame.Bands[0], MigrationPreview{}); !strings.Contains(summary.status, "queued") {
		t.Fatalf("queued target status = %q", summary.status)
	}
	if !queuedFrame.Bands[0].HasQueuedMigration || queuedFrame.Bands[0].QueuedMigration != 3 {
		t.Fatalf("draw changed queued migration state: %#v", queuedFrame.Bands[0])
	}

	preview := MigrationPreview{BandID: 7, TileID: 4, Visible: true}
	previewed := renderMapOffscreen(t, baseFrame, 7, preview, FieldNote{}, false, EndScene{}, "")
	defer previewed.Deallocate()
	if summary := targetTileSummary(baseFrame, &baseFrame.Bands[0], preview); !strings.Contains(summary.status, "arrow cursor") || !strings.Contains(summary.status, "reachable") {
		t.Fatalf("preview target status = %q", summary.status)
	}
	if baseFrame.Bands[0].HasQueuedMigration {
		t.Fatal("drawing a preview queued it in authoritative state")
	}

	blank := ebiten.NewImage(32, 32)
	defer blank.Deallocate()
	drawMigrationArrow(newLogicalCanvas(blank, 1), 12, 12, 12, 12, color.White)
	if blank.Bounds() != image.Rect(0, 0, 32, 32) {
		t.Fatalf("zero-length arrow changed image bounds to %v", blank.Bounds())
	}
}

func TestMapSceneDrawsFieldNotesAndTerminalVariantsOffscreen(t *testing.T) {
	frame := representativeRenderFrame()
	note := FieldNote{
		Topic: "Cold adaptation", Introduction: "A visible field note.", Context: "Scientific context.",
		GameEffect: "A modeled effect.", Hint: "A useful hint.", Celebration: true,
	}

	visible := renderMapOffscreen(t, frame, 7, MigrationPreview{}, note, true, EndScene{}, "Technology learned")
	defer visible.Deallocate()
	if panel, heading := fieldNotePanelColors(note.Celebration); panel != (color.RGBA{R: 45, G: 39, B: 24, A: 255}) || heading != (color.RGBA{R: 255, G: 213, B: 92, A: 255}) {
		t.Fatalf("celebration colors = panel %v, heading %v", panel, heading)
	}

	hidden := renderMapOffscreen(t, frame, 7, MigrationPreview{}, note, false, EndScene{}, "")
	defer hidden.Deallocate()
	if panel, heading := fieldNotePanelColors(false); panel != (color.RGBA{R: 19, G: 28, B: 34, A: 255}) || heading != (color.RGBA{R: 203, G: 172, B: 104, A: 255}) {
		t.Fatalf("ordinary Field Notes colors = panel %v, heading %v", panel, heading)
	}

	endings := []struct {
		name   string
		result gameapi.CampaignResult
		accent color.RGBA
	}{
		{name: "victory", result: gameapi.Victory, accent: color.RGBA{R: 121, G: 195, B: 137, A: 255}},
		{name: "extinction", result: gameapi.Extinction, accent: color.RGBA{R: 232, G: 112, B: 92, A: 255}},
		{name: "dispersal failed", result: gameapi.DispersalFailed, accent: color.RGBA{R: 203, G: 172, B: 104, A: 255}},
	}
	for _, test := range endings {
		t.Run(test.name, func(t *testing.T) {
			ending := EndScene{
				Visible: true, Result: test.result, Title: test.result.String(), Subtitle: "Campaign complete",
				Epilogue: "The journey is recorded.", Destinations: "Levant · South Asia", Turn: 400, YearBP: 20_000,
				SapiensPopulation: 840, ArchaicPopulation: 190, SapiensBands: 8, ArchaicBands: 2, RegionsEstablished: 7, DestinationCount: 2,
			}
			image := renderMapOffscreen(t, frame, 7, MigrationPreview{}, note, true, ending, "")
			defer image.Deallocate()
			if got := endSceneAccent(test.result); got != test.accent {
				t.Fatalf("end-scene accent = %v, want %v", got, test.accent)
			}
			if !NewCampaignButtonContains(newCampaignButtonX+5, newCampaignButtonY+5) {
				t.Fatal("rendered new-campaign button has no active interior")
			}
		})
	}
}

func representativeRenderFrame() *gameapi.Frame {
	tile := func(id gameapi.TileID, x, y int, biome gameapi.Biome) gameapi.Tile {
		return gameapi.Tile{
			ID: id, X: x, Y: y, Land: true, Explored: true, Biome: biome, Region: gameapi.EastAfrica,
			BaselineK: 160, EcologicalK: 150, FloraStock: 80, FloraCap: 120, FaunaStock: 60, FaunaCap: 100,
			WaterStock: 90, WaterCap: 110, LocalTemperatureC: 24, NaturalShelter: 0.25, MovementCost: 1.1,
		}
	}
	tiles := []gameapi.Tile{
		tile(0, 0, 0, gameapi.Savanna),
		{ID: 1, X: 1, Y: 0, Land: false, Explored: true},
		tile(2, 2, 0, gameapi.GlacialTundra),
		tile(3, 3, 0, gameapi.RiverineWoodland),
		tile(4, 4, 0, gameapi.CoastalShrubland),
		tile(5, 0, 2, gameapi.Savanna),
		tile(6, 4, 2, gameapi.Savanna),
		tile(7, 0, 4, gameapi.SemiAridDesert),
		tile(8, 4, 4, gameapi.SemiAridDesert),
	}
	tiles[2].Explored = false
	options := [gameapi.TechCount]gameapi.ResearchOption{}
	options[0] = gameapi.ResearchOption{Current: true, Available: true}
	options[1] = gameapi.ResearchOption{Acquired: true}
	options[2] = gameapi.ResearchOption{Available: true}
	return &gameapi.Frame{
		WorldRevision: 12, TerrainRevision: 4, Turn: 135, YearBP: 43_250, Era: gameapi.EraMiddle,
		CalendarProgress: 0.6125, Season: gameapi.SeasonCooling,
		Climate:       gameapi.ClimateSummary{AridityIndex: 0.4, Epoch: gameapi.AridTransition},
		MacroEpisodes: []gameapi.MacroEpisodeSummary{{Episode: gameapi.CampanianIgnimbrite, Warned: true}},
		Passages: []gameapi.Passage{
			{ID: gameapi.NorthWallacea, From: 5, To: 6, Status: gameapi.PassageOpen, Explored: true},
			{ID: gameapi.SouthWallacea, From: 7, To: 8, Status: gameapi.PassageLocked, Explored: true},
			{ID: gameapi.BeringStrait, From: 0, To: 99, Status: gameapi.PassageOpen, Explored: true},
		},
		SapiensEstablishedRegions: []gameapi.Region{gameapi.EastAfrica, gameapi.Levant},
		Tiles:                     tiles,
		Bands: []gameapi.Band{
			{
				ID: 7, Species: gameapi.HomoSapiens, TileID: 0, Population: 120, Health: 0.94,
				ResearchOptions: options, InterbreedCandidateIDs: []gameapi.BandID{9},
				MigrationCandidates:   []gameapi.MigrationCandidate{{TileID: 3}, {TileID: 4}},
				SeasonalMortalityRate: 0.001, ChronicMortalityRate: 0.002,
				LastMortality:     gameapi.MortalityReport{Seasonal: 2, Chronic: 1},
				LastOutcomeReport: gameapi.OutcomeReport{Turn: 135, StartingPopulation: 124, EndingPopulation: 120, StartingHealth: 0.96, EndingHealth: 0.94, DiseaseHealthLoss: 0.02},
			},
			{ID: 9, Species: gameapi.ArchaicHominin, TileID: 4, Population: 90, Health: 0.9},
			{ID: 10, Species: gameapi.ArchaicHominin, TileID: 2, Population: 80, Health: 0.85},
		},
		CampaignResult: gameapi.Ongoing,
	}
}

func cloneRenderFrame(frame *gameapi.Frame) *gameapi.Frame {
	clone := *frame
	clone.Tiles = append([]gameapi.Tile(nil), frame.Tiles...)
	clone.Bands = append([]gameapi.Band(nil), frame.Bands...)
	clone.Passages = append([]gameapi.Passage(nil), frame.Passages...)
	clone.Escarpments = append([]gameapi.Escarpment(nil), frame.Escarpments...)
	clone.MacroEpisodes = append([]gameapi.MacroEpisodeSummary(nil), frame.MacroEpisodes...)
	clone.SapiensEstablishedRegions = append([]gameapi.Region(nil), frame.SapiensEstablishedRegions...)
	for index := range clone.Bands {
		clone.Bands[index].MigrationCandidates = append([]gameapi.MigrationCandidate(nil), frame.Bands[index].MigrationCandidates...)
		clone.Bands[index].InterbreedCandidateIDs = append([]gameapi.BandID(nil), frame.Bands[index].InterbreedCandidateIDs...)
	}
	return &clone
}

func renderMapOffscreen(t *testing.T, frame *gameapi.Frame, selected gameapi.BandID, preview MigrationPreview, note FieldNote, notesVisible bool, ending EndScene, notice string) *ebiten.Image {
	t.Helper()
	screen := ebiten.NewImage(1280, 720)
	scene := NewMapScene()
	scene.Update()
	scene.Draw(screen, frame, selected, preview, notice, note, notesVisible, ending, false)
	if screen.Bounds() != image.Rect(0, 0, 1280, 720) {
		t.Fatalf("offscreen render bounds = %v", screen.Bounds())
	}
	return screen
}
