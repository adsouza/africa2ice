package render

import (
	"image"
	"image/color"
	"math"
	"reflect"
	"strings"
	"testing"

	"github.com/adsouza/africa2ice/pkg/gameapi"
	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/text/v2"
)

func TestMapTileAt(t *testing.T) {
	frame := cameraFrame()
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
			gotID, gotOK := MapTileAt(Camera{}, frame, 626, test.x, test.y)
			if gotID != test.wantID || gotOK != test.wantOK {
				t.Fatalf("MapTileAt(%d, %d) = (%d, %t), want (%d, %t)", test.x, test.y, gotID, gotOK, test.wantID, test.wantOK)
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

// TestTileMarkersForRenderSplitsCoLocatedSpecies covers the disc a tile draws
// for the bands standing on it. Each band used to fill its own circle at its
// tile's centre, so a sapiens band and an archaic band sharing a tile painted
// the same disc and whichever sorted later in the slice simply won
// (user-reported). Grouping by tile first is what lets the marker show both.
func TestTileMarkersForRenderSplitsCoLocatedSpecies(t *testing.T) {
	tiles := make([]gameapi.Tile, 5)
	for id := range tiles {
		tiles[id] = gameapi.Tile{ID: gameapi.TileID(id), Land: true, Explored: true}
	}
	tiles[3].Explored = false
	frame := &gameapi.Frame{
		Tiles: tiles,
		Bands: []gameapi.Band{
			{ID: 1, Species: gameapi.HomoSapiens, TileID: 2},
			{ID: 2, Species: gameapi.ArchaicHominin, TileID: 2},
			{ID: 3, Species: gameapi.HomoSapiens, TileID: 0},
			{ID: 4, Species: gameapi.HomoSapiens, TileID: 0},
			{ID: 5, Species: gameapi.ArchaicHominin, TileID: 3},
			{ID: 6, Species: gameapi.ArchaicHominin, TileID: 1},
			{ID: 7, Species: gameapi.HomoSapiens, TileID: 99},
		},
	}
	// Tile order follows first appearance so the map does not reshuffle discs
	// between frames; tile 3's archaic band stays fogged and tile 99 does not
	// exist, so neither reaches the map.
	want := []tileMarker{
		{TileID: 2, Counts: [gameapi.SpeciesCount]int{1, 1}},
		{TileID: 0, Counts: [gameapi.SpeciesCount]int{2, 0}},
		{TileID: 1, Counts: [gameapi.SpeciesCount]int{0, 1}},
	}
	if got := tileMarkersForRender(frame); !reflect.DeepEqual(got, want) {
		t.Fatalf("tileMarkersForRender() = %+v, want %+v", got, want)
	}
	if got := tileMarkersForRender(nil); got != nil {
		t.Fatalf("tileMarkersForRender(nil) = %+v, want nil", got)
	}
}

// TestMarkerWedgesSplitsTheDiscByBandCount pins the proportions of a tile's
// disc. Wedges run clockwise from twelve o'clock in species order so a tile
// keeps its slice arrangement as bands are founded and die.
func TestMarkerWedgesSplitsTheDiscByBandCount(t *testing.T) {
	const up = -math.Pi / 2
	tests := []struct {
		name   string
		counts [gameapi.SpeciesCount]int
		want   []markerWedge
	}{
		{name: "empty", counts: [gameapi.SpeciesCount]int{0, 0}, want: []markerWedge{}},
		{
			name: "sapiens alone", counts: [gameapi.SpeciesCount]int{1, 0},
			want: []markerWedge{{Color: sapiensBandMarkerColor, StartAngle: up, SweepAngle: 2 * math.Pi}},
		},
		{
			name: "archaic alone", counts: [gameapi.SpeciesCount]int{0, 3},
			want: []markerWedge{{Color: archaicBandMarkerColor, StartAngle: up, SweepAngle: 2 * math.Pi}},
		},
		{
			name: "one each", counts: [gameapi.SpeciesCount]int{1, 1},
			want: []markerWedge{
				{Color: sapiensBandMarkerColor, StartAngle: up, SweepAngle: math.Pi},
				{Color: archaicBandMarkerColor, StartAngle: up + math.Pi, SweepAngle: math.Pi},
			},
		},
		{
			name: "three sapiens to one archaic", counts: [gameapi.SpeciesCount]int{3, 1},
			want: []markerWedge{
				{Color: sapiensBandMarkerColor, StartAngle: up, SweepAngle: 3 * math.Pi / 2},
				{Color: archaicBandMarkerColor, StartAngle: up + 3*math.Pi/2, SweepAngle: math.Pi / 2},
			},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			wedges, count := markerWedges(tileMarker{Counts: test.counts})
			if count != len(test.want) {
				t.Fatalf("markerWedges(%v) count = %d, want %d", test.counts, count, len(test.want))
			}
			if got := wedges[:count]; !reflect.DeepEqual(got, test.want) {
				t.Fatalf("markerWedges(%v) = %+v, want %+v", test.counts, got, test.want)
			}
		})
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

func TestMapSceneDrawsTerrainVisibilityBandsAndPassagesOffscreen(t *testing.T) {
	frame := representativeRenderFrame()
	wantFrame := cloneRenderFrame(frame)
	screen := renderMapOffscreen(t, frame, 7, MigrationPreview{}, EndScene{}, "")
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

// TestMapSceneSplitsACoLocatedTileMarkerBetweenSpecies is the regression for
// the marker overdraw: with a band per circle, a sapiens and an archaic band
// on one tile filled the same disc and the later one in the slice simply won,
// hiding the other entirely (user-reported).
func TestMapSceneSplitsACoLocatedTileMarkerBetweenSpecies(t *testing.T) {
	frame := representativeRenderFrame()
	frame.Bands = append(frame.Bands, gameapi.Band{ID: 11, Species: gameapi.ArchaicHominin, TileID: 0, Population: 40})
	screen := renderMapOffscreen(t, frame, 0, MigrationPreview{}, EndScene{}, "")
	defer screen.Deallocate()

	x, y := NewMapScene().geometry(frame).TilePoint(frame.Tiles[0])
	// Wedges run clockwise from twelve o'clock, so an even split puts the
	// sapiens slice in the right half of the disc and the archaic in the left.
	if got := screen.At(int(x)+2, int(y)); got != sapiensBandMarkerColor {
		t.Errorf("right half of the shared marker = %v, want sapiens %v", got, sapiensBandMarkerColor)
	}
	if got := screen.At(int(x)-2, int(y)); got != archaicBandMarkerColor {
		t.Errorf("left half of the shared marker = %v, want archaic %v", got, archaicBandMarkerColor)
	}
}

func TestMapSceneDrawsEscarpmentBoundaryOffscreen(t *testing.T) {
	frame := representativeRenderFrame()
	frame.Tiles[1] = gameapi.Tile{ID: 1, X: 1, Y: 0, Land: true, Explored: true, Biome: gameapi.MountainousHighlands}
	frame.Escarpments = []gameapi.Escarpment{{Name: "Test Front", First: 0, Second: 1}}
	wantFrame := cloneRenderFrame(frame)
	image := renderMapOffscreen(t, frame, 7, MigrationPreview{}, EndScene{}, "")
	defer image.Deallocate()
	if !reflect.DeepEqual(frame, wantFrame) {
		t.Fatal("drawing the escarpment mutated the accepted frame")
	}
	if escarpmentColor.R < 160 || escarpmentColor.G < 80 || escarpmentColor.B > 130 {
		t.Fatalf("escarpment color = %v, want visible ochre", escarpmentColor)
	}

	boundaryX := mapOriginX + mapTileSize
	overview := MapGeometry{OriginX: mapOriginX, OriginY: mapOriginY, Cell: mapTileSize}
	fromX, fromY, toX, toY, ok := escarpmentLine(overview, frame.Tiles[0], frame.Tiles[1])
	if !ok || fromX != float32(boundaryX) || toX != float32(boundaryX) || fromY != mapOriginY || toY != mapOriginY+mapTileSize {
		t.Fatalf("escarpment line = (%.0f,%.0f)-(%.0f,%.0f), %t", fromX, fromY, toX, toY, ok)
	}
}

func TestMapSceneCachesTerrainByTerrainRevisionAndAridity(t *testing.T) {
	frame := representativeRenderFrame()
	screen := ebiten.NewImage(1280, 720)
	defer screen.Deallocate()
	scene := NewMapScene()

	scene.Draw(screen, frame, 7, MigrationPreview{}, "", EndScene{}, false)
	scene.Draw(screen, frame, 7, MigrationPreview{}, "", EndScene{}, false)
	if scene.terrainRebuilds != 1 {
		t.Fatalf("terrain rebuilds for unchanged accepted frame = %d, want 1", scene.terrainRebuilds)
	}
	if !scene.frameCached || scene.frameKey.frame != frame {
		t.Fatal("complete immutable presentation frame was not cached")
	}

	nextFrame := cloneRenderFrame(frame)
	nextFrame.WorldRevision++
	scene.Draw(screen, nextFrame, 7, MigrationPreview{}, "", EndScene{}, false)
	if scene.terrainRebuilds != 1 {
		t.Fatalf("terrain rebuilds after planning-only frame replacement = %d, want 1", scene.terrainRebuilds)
	}

	nextFrame = cloneRenderFrame(nextFrame)
	nextFrame.TerrainRevision++
	scene.Draw(screen, nextFrame, 7, MigrationPreview{}, "", EndScene{}, false)
	if scene.terrainRebuilds != 2 {
		t.Fatalf("terrain rebuilds after terrain revision = %d, want 2", scene.terrainRebuilds)
	}

	nextFrame = cloneRenderFrame(nextFrame)
	nextFrame.Climate.AridityIndex += 0.1
	scene.Draw(screen, nextFrame, 7, MigrationPreview{}, "", EndScene{}, false)
	if scene.terrainRebuilds != 3 {
		t.Fatalf("terrain rebuilds after water-grade change = %d, want 3", scene.terrainRebuilds)
	}
}

// TestDrawSkipsOnlyWhenNeitherFrameNorChromeChanged covers the performance
// floor (DESIGN.md §8): production disables Ebitengine's automatic screen
// clear, so an idle frame must do nothing. It also covers the bug that an
// always-repaint fix was papering over: when only the chrome (pkg/hud's
// panel) changed — details collapsing, the drawer shrinking, a settings
// window closing — the map's own key is unchanged, but the vacated chrome
// pixels still need this frame's pixels blitted back over them. Both must
// hold: skip when neither the frame key nor the chrome revision changed;
// repaint when either did.
func TestDrawSkipsOnlyWhenNeitherFrameNorChromeChanged(t *testing.T) {
	frame := representativeRenderFrame()
	screen := ebiten.NewImage(1280, 720)
	defer screen.Deallocate()
	scene := NewMapScene()

	scene.Draw(screen, frame, 7, MigrationPreview{}, "", EndScene{}, false)

	screen.Fill(color.RGBA{R: 255, G: 0, B: 255, A: 255})
	if painted := scene.Draw(screen, frame, 7, MigrationPreview{}, "", EndScene{}, false); painted {
		t.Fatal("Draw repainted on an unchanged key; idle frames must be skipped")
	}
	if got := screen.At(2, 2); got != (color.RGBA{R: 255, G: 0, B: 255, A: 255}) {
		t.Fatalf("screen.At(2,2) = %v, want the untouched magenta fill from a skipped Draw", got)
	}

	scene.SetChromeRevision(1)
	if painted := scene.Draw(screen, frame, 7, MigrationPreview{}, "", EndScene{}, false); !painted {
		t.Fatal("Draw skipped after a chrome-only change; the vacated chrome pixels would stay stale")
	}
	if got := screen.At(2, 2); got == (color.RGBA{R: 255, G: 0, B: 255, A: 255}) {
		t.Fatalf("screen.At(2,2) = %v, want the magenta repainted over after a chrome-only change", got)
	}
}

func TestTopDownTerrainKeepsColorPickingAndMarkersOnTheSameGrid(t *testing.T) {
	frame := representativeRenderFrame()
	frame.Tiles[5].Biome = gameapi.MountainousHighlands
	scene := NewMapScene()
	scene.SetCamera(Camera{}, 626)

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
			pointX, pointY := scene.geometry(frame).TilePoint(test.tile)
			if pointX != float32(mapOriginX+localX) || pointY != float32(mapOriginY+localY) {
				t.Fatalf("tile %d marker point = (%v,%v), want (%d,%d)", test.tile.ID, pointX, pointY, mapOriginX+localX, mapOriginY+localY)
			}
			picked, ok := scene.geometry(frame).TileAt(mapOriginX+localX, mapOriginY+localY)
			wantID := gameapi.TileID(test.tile.Y*TerrainGridWidth + test.tile.X)
			if !ok || picked != wantID {
				t.Fatalf("tile %d pick = (%d,%t), want (%d,true)", test.tile.ID, picked, ok, wantID)
			}
		})
	}
}

func TestTopDownTerrainPicksTheSameGridInFocus(t *testing.T) {
	frame := cameraFrame()
	center := gameapi.TileID(30*TerrainGridWidth + 40)
	camera := Camera{Mode: CameraFocus, CenterTile: center, Progress: 1}

	for _, id := range []gameapi.TileID{center, center + 1, center + TerrainGridWidth} {
		tile := frame.Tiles[id]
		x, y := CameraGeometry(camera, frame, 626).TilePoint(tile)
		picked, ok := MapTileAt(camera, frame, 626, int(x), int(y))
		if !ok || picked != id {
			t.Fatalf("focused tile %d pick = (%d,%t), want (%d,true)", id, picked, ok, id)
		}
	}
}

func TestMapSceneBuildsPhysicalPresentationForAHighDPITarget(t *testing.T) {
	frame := representativeRenderFrame()
	highDPI := ebiten.NewImage(2560, 1440)
	defer highDPI.Deallocate()
	scene := NewMapScene()
	scene.Draw(highDPI, frame, 7, MigrationPreview{}, "", EndScene{}, false)
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

	scene.Draw(standard, frame, 7, MigrationPreview{}, "", EndScene{}, false)
	if scene.terrainRebuilds != 1 || scene.terrainScale != 1 {
		t.Fatalf("standard terrain cache = %d rebuilds at scale %v", scene.terrainRebuilds, scene.terrainScale)
	}
	scene.Draw(highDPI, frame, 7, MigrationPreview{}, "", EndScene{}, false)
	if scene.terrainRebuilds != 2 || scene.terrainScale != 2 {
		t.Fatalf("high-DPI terrain cache = %d rebuilds at scale %v", scene.terrainRebuilds, scene.terrainScale)
	}
	scene.Draw(highDPI, frame, 7, MigrationPreview{}, "", EndScene{}, false)
	if scene.terrainRebuilds != 2 {
		t.Fatalf("unchanged high-DPI draw rebuilt terrain %d times", scene.terrainRebuilds)
	}
}

func TestMapSceneDrawsQueuedAndPreviewMigrationsOffscreen(t *testing.T) {
	baseFrame := representativeRenderFrame()
	base := renderMapOffscreen(t, baseFrame, 7, MigrationPreview{}, EndScene{}, "")
	defer base.Deallocate()

	queuedFrame := cloneRenderFrame(baseFrame)
	queuedFrame.Bands[0].HasQueuedMigration = true
	queuedFrame.Bands[0].QueuedMigration = 3
	queued := renderMapOffscreen(t, queuedFrame, 7, MigrationPreview{}, EndScene{}, "")
	defer queued.Deallocate()
	if !queuedFrame.Bands[0].HasQueuedMigration || queuedFrame.Bands[0].QueuedMigration != 3 {
		t.Fatalf("draw changed queued migration state: %#v", queuedFrame.Bands[0])
	}

	preview := MigrationPreview{BandID: 7, TileID: 4, Visible: true}
	previewed := renderMapOffscreen(t, baseFrame, 7, preview, EndScene{}, "")
	defer previewed.Deallocate()
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

func TestMapSceneDrawsTerminalVariantsOffscreen(t *testing.T) {
	frame := representativeRenderFrame()
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
			image := renderMapOffscreen(t, frame, 7, MigrationPreview{}, ending, "")
			defer image.Deallocate()
			if got := endSceneAccent(test.result); got != test.accent {
				t.Fatalf("end-scene accent = %v, want %v", got, test.accent)
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

func renderMapOffscreen(t *testing.T, frame *gameapi.Frame, selected gameapi.BandID, preview MigrationPreview, ending EndScene, notice string) *ebiten.Image {
	t.Helper()
	screen := ebiten.NewImage(1280, 720)
	scene := NewMapScene()
	scene.Update()
	scene.Draw(screen, frame, selected, preview, notice, ending, false)
	if screen.Bounds() != image.Rect(0, 0, 1280, 720) {
		t.Fatalf("offscreen render bounds = %v", screen.Bounds())
	}
	return screen
}

func TestPassageOverlayShowsOnlyLocalGlyphUntilBothEndpointsExplored(t *testing.T) {
	frame := representativeRenderFrame()
	// Tile 2 is the fixture's only unexplored tile; the projection sets
	// Passage.Explored whenever either endpoint is known, so that bit alone
	// cannot say which shore the player has actually reached.
	passage := gameapi.Passage{ID: gameapi.BeringStrait, From: 0, To: 2, Status: gameapi.PassageLocked, Explored: true}
	if kind, anchor := passageOverlayForRender(frame, passage); kind != passageOverlayGlyph || anchor != 0 {
		t.Fatalf("one explored endpoint = (%v, %d), want glyph at tile 0", kind, anchor)
	}
	passage.From, passage.To = 2, 0
	if kind, anchor := passageOverlayForRender(frame, passage); kind != passageOverlayGlyph || anchor != 0 {
		t.Fatalf("reversed endpoints = (%v, %d), want glyph at the explored tile 0", kind, anchor)
	}
	passage.From, passage.To = 0, 3
	if kind, _ := passageOverlayForRender(frame, passage); kind != passageOverlayLine {
		t.Fatalf("both endpoints explored = %v, want full line", kind)
	}
	frame.Tiles[0].Explored = false
	passage.From, passage.To = 0, 2
	if kind, _ := passageOverlayForRender(frame, passage); kind != passageOverlayHidden {
		t.Fatalf("stale Explored bit with both tiles hidden = %v, want hidden", kind)
	}

	frame = representativeRenderFrame()
	frame.Passages = append(frame.Passages, gameapi.Passage{ID: gameapi.BeringStrait, From: 0, To: 2, Status: gameapi.PassageLocked, Explored: true})
	wantFrame := cloneRenderFrame(frame)
	screen := renderMapOffscreen(t, frame, 7, MigrationPreview{}, EndScene{}, "")
	defer screen.Deallocate()
	if !reflect.DeepEqual(frame, wantFrame) {
		t.Fatal("drawing a one-endpoint passage mutated the accepted frame")
	}
}

func TestNoticeWrapsToBoxWidthByMeasuredPixels(t *testing.T) {
	scene := NewMapScene()
	long := "Bands cannot occupy open water; South Wallacea crosses it from here: select its highlighted far endpoint. Keep using arrows, or press Esc to clear."
	lines := scene.wrapTextToWidth(long, noticeFontSize, noticeTextMaxWidth)
	if len(lines) < 2 || strings.Join(lines, " ") != long {
		t.Fatalf("long notice wrapped to %#v", lines)
	}
	face := &text.GoTextFace{Source: scene.faceSource, Size: noticeFontSize}
	for _, line := range lines {
		if width, _ := text.Measure(line, face, 0); width > noticeTextMaxWidth {
			t.Fatalf("line %q measures %.1f px, wider than the %d px notice box", line, width, noticeTextMaxWidth)
		}
	}
	if short := scene.wrapTextToWidth("Workforce allocation applied", noticeFontSize, noticeTextMaxWidth); len(short) != 1 {
		t.Fatalf("short notice wrapped to %#v", short)
	}
	if got := noticeBoxHeight(len(lines)); got <= noticeBoxHeight(1) {
		t.Fatalf("box height for %d lines = %.1f, not taller than one line %.1f", len(lines), got, noticeBoxHeight(1))
	}
}

// haloRenderFrame is a full grid with one explored land tile, so the halo has
// clean rings to read and nothing else is drawn over them.
func haloRenderFrame() *gameapi.Frame {
	frame := &gameapi.Frame{TerrainRevision: 1, Tiles: make([]gameapi.Tile, TerrainGridWidth*TerrainGridHeight)}
	for y := range TerrainGridHeight {
		for x := range TerrainGridWidth {
			id := gameapi.TileID(y*TerrainGridWidth + x)
			frame.Tiles[id] = gameapi.Tile{ID: id, X: x, Y: y, Land: true, Biome: gameapi.GlacialTundra}
		}
	}
	frame.Tiles[30*TerrainGridWidth+40].Explored = true
	return frame
}

// tileCentrePixel reads the screen pixel at the centre of a tile's cell in
// overview, where the cell is mapTileSize and the origin is fixed.
func tileCentrePixel(screen *ebiten.Image, x, y int) color.RGBA {
	at := screen.At(mapOriginX+x*mapTileSize+mapTileSize/2, mapOriginY+y*mapTileSize+mapTileSize/2)
	red, green, blue, alpha := at.RGBA()
	return color.RGBA{R: uint8(red >> 8), G: uint8(green >> 8), B: uint8(blue >> 8), A: uint8(alpha >> 8)}
}

func TestHaloLightensNearbyFogAndLeavesDistantFogAlone(t *testing.T) {
	screen := renderMapOffscreen(t, haloRenderFrame(), 0, MigrationPreview{}, EndScene{}, "")
	fog := cieLightness(unexploredTileColor)
	for ring := 1; ring <= haloRingCount; ring++ {
		got := cieLightness(tileCentrePixel(screen, 40+ring, 30))
		if got <= fog {
			t.Errorf("ring %d tile L* %.2f is not above the flat fog L* %.2f", ring, got, fog)
		}
	}
	if got := cieLightness(tileCentrePixel(screen, 40+haloRingCount+1, 30)); math.Abs(got-fog) > 0.5 {
		t.Errorf("tile past the last ring has L* %.2f, want flat fog L* %.2f", got, fog)
	}
}

func TestHaloNeverReachesAnExploredTilesLightness(t *testing.T) {
	screen := renderMapOffscreen(t, haloRenderFrame(), 0, MigrationPreview{}, EndScene{}, "")
	floor := darkestExploredLightness()
	for ring := 1; ring <= haloRingCount; ring++ {
		for offset := -ring; offset <= ring; offset++ {
			if got := cieLightness(tileCentrePixel(screen, 40+ring, 30+offset)); got >= floor {
				t.Fatalf("halo tile at ring %d offset %d has L* %.2f, at or above the explored floor %.2f", ring, offset, got, floor)
			}
		}
	}
}

func TestDrawRepaintsWhenTheShimmerStepAdvancesAndSkipsWhenItDoesNot(t *testing.T) {
	frame := haloRenderFrame()
	screen := ebiten.NewImage(1280, 720)
	scene := NewMapScene()
	scene.Update()
	if !scene.Draw(screen, frame, 0, MigrationPreview{}, "", EndScene{}, false) {
		t.Fatal("the first Draw did not paint")
	}
	if scene.Draw(screen, frame, 0, MigrationPreview{}, "", EndScene{}, false) {
		t.Fatal("Draw repainted with nothing changed")
	}
	for range shimmerTickStride {
		scene.Update()
	}
	if !scene.Draw(screen, frame, 0, MigrationPreview{}, "", EndScene{}, false) {
		t.Fatal("Draw did not repaint after the shimmer phase advanced")
	}
}

// A frame with everything explored has no fringe, so the shimmer has nothing
// to animate and must not cost a repaint. This is also what keeps the
// all-explored performance fixture on its existing idle path.
func TestShimmerDoesNotRepaintWithNoFringe(t *testing.T) {
	frame := haloRenderFrame()
	for index := range frame.Tiles {
		frame.Tiles[index].Explored = true
	}
	screen := ebiten.NewImage(1280, 720)
	scene := NewMapScene()
	scene.Update()
	scene.Draw(screen, frame, 0, MigrationPreview{}, "", EndScene{}, false)
	for range 4 * shimmerTickStride {
		scene.Update()
	}
	if scene.Draw(screen, frame, 0, MigrationPreview{}, "", EndScene{}, false) {
		t.Fatal("the shimmer forced a repaint on a fully explored frame")
	}
}

// The halo is a paint change and must not become an input change. Picking
// already resolves a tile ID for any in-bounds pixel; what matters is that a
// haloed tile still reports as unexplored, which is the bit every caller in
// pkg/app gates selection and migration on.
func TestHaloTilesStayUnexploredForPicking(t *testing.T) {
	frame := haloRenderFrame()
	for ring := 1; ring <= haloRingCount; ring++ {
		x := mapOriginX + (40+ring)*mapTileSize + mapTileSize/2
		y := mapOriginY + 30*mapTileSize + mapTileSize/2
		id, ok := MapTileAt(Camera{}, frame, 626, x, y)
		if !ok {
			t.Fatalf("ring %d tile did not resolve for picking", ring)
		}
		if frame.Tiles[id].Explored {
			t.Fatalf("ring %d tile %d reports as explored; the halo has leaked into input", ring, id)
		}
	}
}

// An unexplored tile must not gain a hover tint just because it is haloed.
// drawFrame's hover branch already tests Explored; this pins that it keeps
// doing so now that the tile is no longer flat fog.
func TestHoverDoesNotTintAHaloTile(t *testing.T) {
	frame := haloRenderFrame()
	screen := ebiten.NewImage(1280, 720)
	scene := NewMapScene()
	scene.Update()
	scene.Draw(screen, frame, 0, MigrationPreview{}, "", EndScene{}, false)
	plain := tileCentrePixel(screen, 41, 30)

	hovered := NewMapScene()
	hovered.SetTileHover(TileHover{TileID: gameapi.TileID(30*TerrainGridWidth + 41), Visible: true})
	hovered.Update()
	hovered.Draw(screen, frame, 0, MigrationPreview{}, "", EndScene{}, false)
	if got := tileCentrePixel(screen, 41, 30); got != plain {
		t.Fatalf("hovering an unexplored halo tile changed it: %v then %v", plain, got)
	}
}

// The shimmer step stored in the frame key must be the one drawHalo actually
// rendered with. haloActive used to be set inside drawTerrain's cache-miss
// block, which runs after the key is built, so on the tick the halo first
// became active the key held a stale step and forced a needless repaint.
// Ticking past a stride boundary before the first Draw is what separates the
// two values; with the cache refreshed ahead of the key, the second Draw is
// correctly idle.
func TestShimmerStepInTheKeyMatchesTheOneDrawn(t *testing.T) {
	frame := haloRenderFrame()
	screen := ebiten.NewImage(1280, 720)
	scene := NewMapScene()
	for range shimmerTickStride {
		scene.Update()
	}
	if !scene.Draw(screen, frame, 0, MigrationPreview{}, "", EndScene{}, false) {
		t.Fatal("the first Draw did not paint")
	}
	if scene.Draw(screen, frame, 0, MigrationPreview{}, "", EndScene{}, false) {
		t.Fatal("the second Draw repainted: the key's shimmer step disagreed with the one drawHalo used")
	}
}

func TestReducedMotionFreezesTheShimmer(t *testing.T) {
	frame := haloRenderFrame()
	screen := ebiten.NewImage(1280, 720)
	scene := NewMapScene()
	scene.SetReducedMotion(true)
	scene.Update()
	scene.Draw(screen, frame, 0, MigrationPreview{}, "", EndScene{}, false)
	before := tileCentrePixel(screen, 41, 30)
	for range 4 * shimmerTickStride {
		scene.Update()
	}
	if scene.Draw(screen, frame, 0, MigrationPreview{}, "", EndScene{}, false) {
		t.Fatal("reduced motion still forced a repaint")
	}
	if after := tileCentrePixel(screen, 41, 30); after != before {
		t.Fatalf("halo colour changed under reduced motion: %v then %v", before, after)
	}
	// The reveal itself must survive; only the motion is dropped.
	if cieLightness(before) <= cieLightness(unexploredTileColor) {
		t.Fatal("reduced motion removed the halo instead of freezing it")
	}
}

// Glyphs must be a separate pass from the terrain cache, which is a fixed
// 8px-per-tile image that drawTerrain scales up with FilterNearest. Counting
// draws mirrors the terrainRebuilds instrumentation and is robust where pixel
// matching against an antialiased glyph would not be.
func TestBiomeGlyphsDrawOnlyAtFocusZoom(t *testing.T) {
	frame := representativeRenderFrame()
	explored := 0
	for _, tile := range frame.Tiles {
		if tile.Explored && tile.Land {
			explored++
		}
	}
	if explored == 0 {
		t.Fatal("fixture has no explored land tiles")
	}
	screen := ebiten.NewImage(1280, 720)
	defer screen.Deallocate()
	scene := NewMapScene()

	scene.SetCamera(Camera{Mode: CameraOverview}, mapAreaHeight)
	scene.Draw(screen, frame, 7, MigrationPreview{}, "", EndScene{}, false)
	if scene.glyphDraws != 0 {
		t.Fatalf("overview drew %d biome glyphs, want 0", scene.glyphDraws)
	}

	scene.SetCamera(Camera{Mode: CameraFocus, Progress: 1}, mapAreaHeight)
	scene.Draw(screen, frame, 7, MigrationPreview{}, "", EndScene{}, false)
	// Pin the exact count rather than bounding a range: "not zero" and "not
	// more than the tiles that exist" both still pass if a regression
	// under-draws (skips one qualifying tile) -- 0 < glyphDraws <= 7 is
	// satisfied by 5 as much as by 7. Only an exact match catches that.
	if scene.glyphDraws != uint64(explored) {
		t.Fatalf("focus drew %d biome glyphs, want %d (one per explored land tile in the fixture)",
			scene.glyphDraws, explored)
	}
}

// Water, unexplored and halo tiles assert nothing a pictograph could restate,
// so a glyph there would be new information rather than a redundant channel.
// gameapi.Tile spells this as Land, not Water -- there is no Water field.
func TestBiomeGlyphsSkipUnexploredAndWaterTiles(t *testing.T) {
	frame := representativeRenderFrame()
	explored := 0
	for _, tile := range frame.Tiles {
		if tile.Explored && tile.Land {
			explored++
		}
	}
	if explored == 0 {
		t.Fatal("fixture has no explored land tiles")
	}
	screen := ebiten.NewImage(1280, 720)
	defer screen.Deallocate()
	scene := NewMapScene()
	scene.SetCamera(Camera{Mode: CameraFocus, Progress: 1}, mapAreaHeight)
	scene.Draw(screen, frame, 7, MigrationPreview{}, "", EndScene{}, false)
	if scene.glyphDraws > uint64(explored) {
		t.Fatalf("drew %d glyphs for %d explored land tiles", scene.glyphDraws, explored)
	}
}

// logicalCanvas scales coordinates at draw time, so a glyph sized in DIP would
// be upscaled and blurry on a high-DPI target. runeGlyph.paint multiplies size
// by canvas scale, mirroring drawText; this pins that it actually does.
func TestBiomeGlyphsRasterizeAtPhysicalScale(t *testing.T) {
	painters, _, err := newBiomeGlyphs()
	if err != nil {
		t.Fatalf("newBiomeGlyphs: %v", err)
	}
	background := climateBiomeColor(gameapi.RiverineWoodland, 0)
	ink := glyphInk(background)

	inkPixels := func(scale float64, size int) int {
		target := ebiten.NewImage(size, size)
		defer target.Deallocate()
		target.Fill(background)
		centre := float32(size) / 2 / float32(scale)
		painters[gameapi.RiverineWoodland].paint(newLogicalCanvas(target, scale), centre, centre, 20, ink, 1)
		count := 0
		for y := 0; y < size; y++ {
			for x := 0; x < size; x++ {
				if r, g, b, _ := target.At(x, y).RGBA(); r>>8 != uint32(background.R) || g>>8 != uint32(background.G) || b>>8 != uint32(background.B) {
					count++
				}
			}
		}
		return count
	}

	standard, highDPI := inkPixels(1, 24), inkPixels(2, 48)
	// A glyph rasterized at DIP and blitted up would cover the same fraction
	// but from 4x fewer source pixels. Rasterizing at physical size means the
	// 2x target carries close to 4x the ink pixels.
	if float64(highDPI) < 3*float64(standard) {
		t.Errorf("high-DPI glyph covered %d px against %d at 1x; expected near 4x", highDPI, standard)
	}
}
