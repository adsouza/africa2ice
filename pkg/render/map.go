package render

import (
	"bytes"
	"fmt"
	"image/color"
	"math"

	"github.com/adsouza/africa2ice/pkg/gameapi"
	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/text/v2"
	"golang.org/x/image/font/gofont/goregular"
)

const (
	TerrainGridWidth       = 96
	TerrainGridHeight      = 64
	mapOriginX             = 20
	mapOriginY             = 74
	mapTileSize            = 9
	mapPixelWidth          = TerrainGridWidth * mapTileSize
	mapPixelHeight         = TerrainGridHeight * mapTileSize
	mapLegendOriginY       = 48
	mapLegendHeight        = 25
	bandListOriginY        = 230
	bandRowHeight          = 16
	bandOutcomeOriginY     = 310
	tileInspectorOriginY   = 340
	tileInspectorHeight    = 116
	tileInspectorTextSize  = 7.4
	tileInspectorRowGap    = 8.8
	interbreedPanelLineY   = 446
	fieldNotesPanelOriginY = 462
	fieldNotesPanelHeight  = 138
	workforcePanelOriginY  = 604
)

var (
	unexploredTileColor    = color.RGBA{R: 6, G: 11, B: 15, A: 255}
	archaicBandMarkerColor = color.RGBA{R: 201, G: 103, B: 82, A: 255}
	escarpmentColor        = color.RGBA{R: 220, G: 142, B: 88, A: 255}
)

type MapScene struct {
	faceSource      *text.GoTextFaceSource
	terrainImage    *ebiten.Image
	terrainRevision uint64
	terrainAridity  float64
	terrainScale    float32
	terrainCached   bool
	terrainRebuilds uint64
	frameImage      *ebiten.Image
	frameKey        mapFrameKey
	frameWidth      int
	frameHeight     int
	frameScale      float64
	frameCached     bool
	workforce       WorkforceDraft
	overlay         MenuOverlay
}

type mapFrameKey struct {
	frame             *gameapi.Frame
	selectedBand      gameapi.BandID
	preview           MigrationPreview
	notice            string
	fieldNote         FieldNote
	fieldNotesVisible bool
	ending            EndScene
	workforce         WorkforceDraft
	resizeRequired    bool
	overlay           MenuOverlay
}

type MigrationPreview struct {
	BandID  gameapi.BandID
	TileID  gameapi.TileID
	Visible bool
}

// WorkforceDraft is UI-local editor state. Accepted allocations continue to
// come from the frame until the complete draft is explicitly applied.
type WorkforceDraft struct {
	Visible      bool
	BandID       gameapi.BandID
	AllocationBP [gameapi.AssignmentCount]uint16
	SelectedRole gameapi.WorkforceRole
	Dirty        bool
	Valid        bool
}

type MenuOverlay struct {
	Visible   bool
	Heading   string
	Help      string
	Lines     [8]string
	LineCount int
	Selected  int
}

// FieldNote is UI-local presentation content. It is never simulation or save
// state; the renderer only lays out the already-selected entry.
type FieldNote struct {
	Topic        string
	Introduction string
	Context      string
	GameEffect   string
	Hint         string
	Celebration  bool
}

func NewMapScene() *MapScene {
	source, err := text.NewGoTextFaceSource(bytes.NewReader(goregular.TTF))
	if err != nil {
		panic(err)
	}
	return &MapScene{faceSource: source}
}

func (scene *MapScene) Update() {}

func (scene *MapScene) SetWorkforceDraft(draft WorkforceDraft) { scene.workforce = draft }
func (scene *MapScene) SetMenuOverlay(overlay MenuOverlay)     { scene.overlay = overlay }

func (scene *MapScene) Draw(screen *ebiten.Image, frame *gameapi.Frame, selectedBand gameapi.BandID, preview MigrationPreview, notice string, fieldNote FieldNote, fieldNotesVisible bool, ending EndScene, resizeRequired bool) {
	if frame == nil {
		screen.Fill(color.RGBA{R: 15, G: 22, B: 29, A: 255})
		return
	}
	key := mapFrameKey{
		frame: frame, selectedBand: selectedBand, preview: preview, notice: notice,
		fieldNote: fieldNote, fieldNotesVisible: fieldNotesVisible, ending: ending,
		workforce: scene.workforce, resizeRequired: resizeRequired, overlay: scene.overlay,
	}
	width, height := screen.Bounds().Dx(), screen.Bounds().Dy()
	if scene.frameCached && scene.frameKey == key && scene.frameWidth == width && scene.frameHeight == height {
		// Production disables Ebitengine's automatic screen clear. Leaving an
		// unchanged screen untouched lets the engine skip GPU work entirely for
		// this turn-based presentation.
		return
	}
	if scene.frameImage != nil {
		scene.frameImage.Deallocate()
	}
	transform := FitPresentation(width, height)
	contentWidth := max(1, int(math.Ceil(PresentationWidth*transform.Scale)))
	contentHeight := max(1, int(math.Ceil(PresentationHeight*transform.Scale)))
	scene.frameImage = ebiten.NewImage(contentWidth, contentHeight)
	canvas := newLogicalCanvas(scene.frameImage, transform.Scale)
	scene.drawFrame(canvas, frame, selectedBand, preview, notice, fieldNote, fieldNotesVisible, ending)
	if resizeRequired {
		scene.drawResizeOverlay(canvas)
	}
	scene.frameKey = key
	scene.frameWidth = width
	scene.frameHeight = height
	scene.frameScale = transform.Scale
	scene.frameCached = true
	screen.Fill(color.RGBA{R: 6, G: 11, B: 15, A: 255})
	op := &ebiten.DrawImageOptions{}
	op.GeoM.Translate(transform.OffsetX, transform.OffsetY)
	screen.DrawImage(scene.frameImage, op)
}

func (scene *MapScene) drawResizeOverlay(screen logicalCanvas) {
	vector.FillRect(screen, 0, 0, PresentationWidth, PresentationHeight, color.RGBA{R: 6, G: 11, B: 15, A: 238}, false)
	scene.drawText(screen, "Window too small", 505, 310, 28, color.RGBA{R: 239, G: 220, B: 178, A: 255})
	scene.drawText(screen, "Resize to at least 1280 × 720 to continue", 440, 360, 15, color.White)
}

func (scene *MapScene) drawFrame(screen logicalCanvas, frame *gameapi.Frame, selectedBand gameapi.BandID, preview MigrationPreview, notice string, fieldNote FieldNote, fieldNotesVisible bool, ending EndScene) {
	screen.image.Fill(color.RGBA{R: 15, G: 22, B: 29, A: 255})
	grade := EpochGrade(frame.Climate.AridityIndex)
	scene.drawTimeline(screen, frame, grade)
	scene.drawMapLegend(screen, frame.Climate.AridityIndex)
	scene.drawTerrain(screen, frame)
	scene.drawReachableTiles(screen, frame, selectedBand)
	scene.drawEscarpments(screen, frame)
	for _, passage := range frame.Passages {
		lineColor, visible := passageColorForRender(frame, passage)
		if !visible {
			continue
		}
		from, to := frame.Tiles[passage.From], frame.Tiles[passage.To]
		fromX, fromY := scene.tilePoint(from)
		toX, toY := scene.tilePoint(to)
		vector.StrokeLine(screen, fromX, fromY, toX, toY, 2, lineColor, false)
	}
	var interbreedTiles map[gameapi.TileID]bool
	if actor := selectedBandInFrame(frame, selectedBand); actor != nil {
		interbreedTiles = interbreedCandidateTiles(frame, *actor)
	}
	for _, band := range frame.Bands {
		marker, visible := bandColorForRender(frame, band)
		if !visible {
			continue
		}
		tile := frame.Tiles[band.TileID]
		centreX, centreY := scene.tilePoint(tile)
		vector.FillCircle(screen, centreX, centreY, 3.6, marker, true)
		// An archaic band the selected band can interbreed with gets its own
		// ring, so the option is visible on the map rather than only discovered
		// by pressing the key and hoping.
		if band.Species == gameapi.ArchaicHominin && interbreedTiles[band.TileID] {
			vector.StrokeCircle(screen, centreX, centreY, 6.4, 1.5, interbreedMarkerColor, true)
		}
		if band.ID == selectedBand {
			vector.StrokeCircle(screen, centreX, centreY, 5.2, 1.5, color.White, true)
		}
	}
	scene.drawQueuedMigrations(screen, frame)
	scene.drawMigrationPreview(screen, frame, preview)
	scene.drawHUD(screen, frame, selectedBand, preview, fieldNote, fieldNotesVisible)
	scene.drawResearchKeys(screen, frame, selectedBand)
	scene.drawEndScene(screen, ending)
	scene.drawMenuOverlay(screen)
	if notice != "" {
		vector.FillRect(screen, 28, 610, 650, 30, color.RGBA{R: 26, G: 38, B: 45, A: 240}, false)
		scene.drawText(screen, notice, 40, 617, 14, color.RGBA{R: 239, G: 220, B: 178, A: 255})
	}
}

func (scene *MapScene) drawEscarpments(screen logicalCanvas, frame *gameapi.Frame) {
	for _, edge := range frame.Escarpments {
		if int(edge.First) >= len(frame.Tiles) || int(edge.Second) >= len(frame.Tiles) {
			continue
		}
		first, second := frame.Tiles[edge.First], frame.Tiles[edge.Second]
		if !first.Explored || !second.Explored {
			continue
		}
		fromX, fromY, toX, toY, ok := escarpmentLine(first, second)
		if !ok {
			continue
		}
		vector.StrokeLine(screen, fromX, fromY, toX, toY, 3, color.RGBA{R: 48, G: 31, B: 26, A: 235}, false)
		vector.StrokeLine(screen, fromX, fromY, toX, toY, 1.35, escarpmentColor, false)
	}
}

func escarpmentLine(first, second gameapi.Tile) (float32, float32, float32, float32, bool) {
	dx, dy := second.X-first.X, second.Y-first.Y
	if absRenderInt(dx)+absRenderInt(dy) != 1 {
		return 0, 0, 0, 0, false
	}
	left := mapOriginX + float32(min(first.X, second.X)*mapTileSize)
	top := mapOriginY + float32(min(first.Y, second.Y)*mapTileSize)
	if dx != 0 {
		x := left + mapTileSize
		return x, top, x, top + mapTileSize, true
	}
	y := top + mapTileSize
	return left, y, left + mapTileSize, y, true
}

func absRenderInt(value int) int {
	if value < 0 {
		return -value
	}
	return value
}

func (scene *MapScene) drawMenuOverlay(screen logicalCanvas) {
	if !scene.overlay.Visible {
		return
	}
	const x, y, width, height = float32(340), float32(150), float32(600), float32(420)
	vector.FillRect(screen, x, y, width, height, color.RGBA{R: 10, G: 17, B: 22, A: 248}, false)
	vector.StrokeRect(screen, x, y, width, height, 2, color.RGBA{R: 203, G: 172, B: 104, A: 255}, false)
	scene.drawText(screen, scene.overlay.Heading, x+28, y+24, 25, color.RGBA{R: 239, G: 220, B: 178, A: 255})
	for index := 0; index < scene.overlay.LineCount && index < len(scene.overlay.Lines); index++ {
		lineY := y + 78 + float32(index)*36
		lineColor := color.RGBA{R: 220, G: 225, B: 218, A: 255}
		prefix := "  "
		if index == scene.overlay.Selected {
			vector.FillRect(screen, x+20, lineY-5, width-40, 29, color.RGBA{R: 35, G: 51, B: 58, A: 255}, false)
			lineColor = color.RGBA{R: 245, G: 202, B: 92, A: 255}
			prefix = "› "
		}
		scene.drawText(screen, prefix+scene.overlay.Lines[index], x+36, lineY, 14, lineColor)
	}
	scene.drawText(screen, scene.overlay.Help, x+28, y+height-42, 11, color.RGBA{R: 167, G: 184, B: 181, A: 255})
}

// drawTerrain caches the immutable top-down tile layer until either its coarse
// terrain revision or its continuously graded water color changes. Commands
// that reveal terrain (including a successful split) advance that revision;
// other planning-only frames can reuse it without stale exploration, biome,
// macro-impact, or climate colors.
func (scene *MapScene) drawTerrain(screen logicalCanvas, frame *gameapi.Frame) {
	if !scene.terrainCached || scene.terrainRevision != frame.TerrainRevision || scene.terrainAridity != frame.Climate.AridityIndex || scene.terrainScale != screen.scale {
		if scene.terrainImage != nil {
			scene.terrainImage.Deallocate()
		}
		terrainWidth := max(1, int(math.Ceil(float64(mapPixelWidth)*float64(screen.scale))))
		terrainHeight := max(1, int(math.Ceil(float64(mapPixelHeight)*float64(screen.scale))))
		scene.terrainImage = ebiten.NewImage(terrainWidth, terrainHeight)
		scene.terrainImage.Fill(unexploredTileColor)
		scene.drawFlatTerrain(newLogicalCanvas(scene.terrainImage, float64(screen.scale)), frame)
		scene.terrainRevision = frame.TerrainRevision
		scene.terrainAridity = frame.Climate.AridityIndex
		scene.terrainScale = screen.scale
		scene.terrainCached = true
		scene.terrainRebuilds++
	}
	options := &ebiten.DrawImageOptions{}
	options.GeoM.Translate(float64(mapOriginX)*float64(screen.scale), float64(mapOriginY)*float64(screen.scale))
	screen.image.DrawImage(scene.terrainImage, options)
}

func (scene *MapScene) drawFlatTerrain(screen logicalCanvas, frame *gameapi.Frame) {
	tileExtent := float32(mapTileSize - 0.4)
	for _, tile := range frame.Tiles {
		vector.FillRect(
			screen,
			float32(tile.X*mapTileSize),
			float32(tile.Y*mapTileSize),
			tileExtent,
			tileExtent,
			tileColorForRender(tile, frame.Climate.AridityIndex),
			false,
		)
	}
}

func (scene *MapScene) tilePoint(tile gameapi.Tile) (float32, float32) {
	return mapOriginX + float32(tile.X*mapTileSize) + mapTileSize/2, mapOriginY + float32(tile.Y*mapTileSize) + mapTileSize/2
}

func (scene *MapScene) PickTile(x, y int) (gameapi.TileID, bool) {
	return MapTileAt(x, y)
}

func (scene *MapScene) drawMigrationPreview(screen logicalCanvas, frame *gameapi.Frame, preview MigrationPreview) {
	if !preview.Visible || int(preview.TileID) >= len(frame.Tiles) {
		return
	}
	band := selectedBandInFrame(frame, preview.BandID)
	if band == nil || int(band.TileID) >= len(frame.Tiles) {
		return
	}
	origin, destination := frame.Tiles[band.TileID], frame.Tiles[preview.TileID]
	fromX, fromY := scene.tilePoint(origin)
	toX, toY := scene.tilePoint(destination)
	drawMigrationArrow(screen, fromX, fromY, toX, toY, color.RGBA{R: 255, G: 74, B: 74, A: 255})
	vector.StrokeCircle(screen, toX, toY, 4.2, 1.2, color.RGBA{R: 255, G: 126, B: 106, A: 255}, true)
}

func (scene *MapScene) drawQueuedMigrations(screen logicalCanvas, frame *gameapi.Frame) {
	arrowColor := color.RGBA{R: 232, G: 72, B: 72, A: 255}
	for _, band := range frame.Bands {
		if band.Species != gameapi.HomoSapiens || !band.HasQueuedMigration || int(band.TileID) >= len(frame.Tiles) || int(band.QueuedMigration) >= len(frame.Tiles) {
			continue
		}
		origin, destination := frame.Tiles[band.TileID], frame.Tiles[band.QueuedMigration]
		if !origin.Explored || !destination.Explored {
			continue
		}
		fromX, fromY := scene.tilePoint(origin)
		toX, toY := scene.tilePoint(destination)
		drawMigrationArrow(screen, fromX, fromY, toX, toY, arrowColor)
	}
}

func drawMigrationArrow(screen logicalCanvas, fromX, fromY, toX, toY float32, arrowColor color.Color) {
	dx, dy := toX-fromX, toY-fromY
	length := float32(math.Hypot(float64(dx), float64(dy)))
	if length <= 0 {
		return
	}
	unitX, unitY := dx/length, dy/length
	const headLength, headWidth = float32(3.5), float32(2.4)
	baseX, baseY := toX-unitX*headLength, toY-unitY*headLength
	perpendicularX, perpendicularY := -unitY*headWidth, unitX*headWidth
	vector.StrokeLine(screen, fromX, fromY, toX, toY, 1.8, arrowColor, true)
	vector.StrokeLine(screen, toX, toY, baseX+perpendicularX, baseY+perpendicularY, 1.8, arrowColor, true)
	vector.StrokeLine(screen, toX, toY, baseX-perpendicularX, baseY-perpendicularY, 1.8, arrowColor, true)
}

func (scene *MapScene) drawReachableTiles(screen logicalCanvas, frame *gameapi.Frame, selectedBand gameapi.BandID) {
	band := selectedBandInFrame(frame, selectedBand)
	if band == nil || band.SpatialActionUsed {
		return
	}
	for index, candidate := range band.MigrationCandidates {
		if int(candidate.TileID) >= len(frame.Tiles) {
			continue
		}
		tile := frame.Tiles[candidate.TileID]
		x, y := scene.tilePoint(tile)
		highlight := reachableTileColor(index)
		x -= mapTileSize / 2
		y -= mapTileSize / 2
		vector.FillRect(screen, x+0.7, y+0.7, mapTileSize-1.8, mapTileSize-1.8, color.RGBA{R: highlight.R, G: highlight.G, B: highlight.B, A: 48}, false)
		vector.StrokeRect(screen, x+0.7, y+0.7, mapTileSize-1.8, mapTileSize-1.8, 1.35, highlight, false)
	}
}

func tileColorForRender(tile gameapi.Tile, aridity float64) color.RGBA {
	if !tile.Explored {
		return unexploredTileColor
	}
	if !tile.Land {
		return EpochGrade(aridity).Water
	}
	return climateBiomeColor(tile.Biome, aridity)
}

func passageColorForRender(frame *gameapi.Frame, passage gameapi.Passage) (color.RGBA, bool) {
	if frame == nil || !passage.Explored || int(passage.From) >= len(frame.Tiles) || int(passage.To) >= len(frame.Tiles) {
		return color.RGBA{}, false
	}
	if passage.Status == gameapi.PassageLocked {
		return color.RGBA{R: 124, G: 111, B: 101, A: 180}, true
	}
	return color.RGBA{R: 203, G: 172, B: 104, A: 210}, true
}

func bandColorForRender(frame *gameapi.Frame, band gameapi.Band) (color.RGBA, bool) {
	if frame == nil || int(band.TileID) >= len(frame.Tiles) {
		return color.RGBA{}, false
	}
	if !frame.Tiles[band.TileID].Explored && band.Species != gameapi.HomoSapiens {
		return color.RGBA{}, false
	}
	if band.Species == gameapi.ArchaicHominin {
		return archaicBandMarkerColor, true
	}
	return color.RGBA{R: 245, G: 202, B: 92, A: 255}, true
}

func reachableTileColor(candidateIndex int) color.RGBA {
	if candidateIndex == 0 {
		return color.RGBA{R: 245, G: 202, B: 92, A: 255}
	}
	return color.RGBA{R: 87, G: 211, B: 211, A: 255}
}

func MapTileAt(x, y int) (gameapi.TileID, bool) {
	gridX := (x - mapOriginX) / mapTileSize
	gridY := (y - mapOriginY) / mapTileSize
	if x < mapOriginX || y < mapOriginY || gridX < 0 || gridX >= 96 || gridY < 0 || gridY >= 64 {
		return 0, false
	}
	return gameapi.TileID(gridY*96 + gridX), true
}

func (scene *MapScene) drawTimeline(screen logicalCanvas, frame *gameapi.Frame, grade GradeColors) {
	const left, right, y = float32(20), float32(1260), float32(38)
	state := deriveTimelineState(frame)
	vector.StrokeLine(screen, left, y, right, y, 2, color.RGBA{R: 91, G: 110, B: 117, A: 255}, false)
	progressX := timelinePosition(left, right, state.Progress)
	vector.StrokeLine(screen, left, y, progressX, y, 2.5, grade.HUDChromeAccent, false)
	for _, boundary := range timelineEraBoundaries {
		x := timelinePosition(left, right, boundary.Progress)
		vector.StrokeLine(screen, x, y-4, x, y+4, 0.7, color.RGBA{R: 105, G: 119, B: 122, A: 210}, false)
	}
	for index, tick := range timelineMajorTicks {
		x := timelinePosition(left, right, tick.Progress)
		vector.StrokeLine(screen, x, y-6, x, y+6, 1, color.RGBA{R: 151, G: 165, B: 165, A: 255}, false)
		labelX := x - 24
		if index == 0 {
			labelX = x
		} else if index == len(timelineMajorTicks)-1 {
			labelX = x - 55
		}
		scene.drawText(screen, formatTimelineYear(tick.YearBP), labelX, 14, 8.5, color.RGBA{R: 174, G: 188, B: 185, A: 255})
	}
	markerColor := grade.HUDChromeAccent
	if state.PulseDirection > 0 {
		markerColor = color.RGBA{R: 224, G: 128, B: 83, A: 255}
	} else if state.PulseDirection < 0 {
		markerColor = color.RGBA{R: 117, G: 177, B: 218, A: 255}
	}
	vector.FillCircle(screen, progressX, y, 5, markerColor, true)
	if !state.CurrentOnMajor {
		labelX := min(max(progressX-25, left), right-57)
		scene.drawText(screen, state.CurrentLabel, labelX, 1, 8.5, markerColor)
	}
	for _, episode := range frame.MacroEpisodes {
		if episode.Episode != gameapi.CampanianIgnimbrite {
			continue
		}
		x := timelinePosition(left, right, timelineProgressForYear(39_850))
		glyph := color.RGBA{R: 118, G: 126, B: 128, A: 220}
		if episode.Warned || episode.Current {
			glyph = color.RGBA{R: 213, G: 115, B: 80, A: 255}
		}
		vector.FillCircle(screen, x, y, 3.5, glyph, true)
	}
	if state.ShowTobaContext {
		x := timelinePosition(left, right, timelineProgressForYear(73_880))
		vector.FillCircle(screen, x, y, 2.6, color.RGBA{R: 143, G: 132, B: 123, A: 220}, true)
	}
}

func (scene *MapScene) drawMapLegend(screen logicalCanvas, aridity float64) {
	vector.FillRect(screen, mapOriginX, mapLegendOriginY, 864, mapLegendHeight, color.RGBA{R: 18, G: 27, B: 33, A: 245}, false)
	entries := mapLegendEntries(aridity)
	const entryWidth = float32(96)
	for index, entry := range entries {
		x := float32(mapOriginX) + float32(index)*entryWidth
		if entry.edge {
			vector.StrokeLine(screen, x+4, mapLegendOriginY+8, x+12, mapLegendOriginY+8, 3, color.RGBA{R: 48, G: 31, B: 26, A: 235}, false)
			vector.StrokeLine(screen, x+4, mapLegendOriginY+8, x+12, mapLegendOriginY+8, 1.35, entry.color, false)
		} else {
			vector.FillRect(screen, x+4, mapLegendOriginY+4, 8, 8, entry.color, false)
			vector.StrokeRect(screen, x+4, mapLegendOriginY+4, 8, 8, 0.7, color.RGBA{R: 210, G: 216, B: 210, A: 180}, false)
		}
		scene.drawText(screen, entry.label, x+15, mapLegendOriginY+1, 8.5, color.RGBA{R: 235, G: 236, B: 226, A: 255})
		scene.drawText(screen, entry.meaning, x+4, mapLegendOriginY+13, 7, color.RGBA{R: 167, G: 184, B: 181, A: 255})
	}
}

func (scene *MapScene) drawHUD(screen logicalCanvas, frame *gameapi.Frame, selectedBand gameapi.BandID, preview MigrationPreview, fieldNote FieldNote, fieldNotesVisible bool) {
	const panelX = float32(908)
	vector.FillRect(screen, panelX, 68, 352, 626, color.RGBA{R: 25, G: 35, B: 42, A: 238}, false)
	scene.drawText(screen, "Africa 2 Ice", panelX+18, 88, 24, color.RGBA{R: 239, G: 220, B: 178, A: 255})
	scene.drawText(screen, fmt.Sprintf("%d BP  ·  Turn %d/400", frame.YearBP, frame.Turn), panelX+18, 124, 16, color.White)
	scene.drawText(screen, frame.Season.String()+"  ·  "+frame.Climate.Epoch.String(), panelX+18, 150, 14, color.RGBA{R: 183, G: 199, B: 194, A: 255})
	var totalPopulation uint64
	for _, band := range frame.Bands {
		if band.Species == gameapi.HomoSapiens {
			totalPopulation += uint64(band.Population)
		}
	}
	scene.drawText(screen, fmt.Sprintf("Homo sapiens: %d", totalPopulation), panelX+18, 188, 17, color.RGBA{R: 245, G: 202, B: 92, A: 255})
	bandWindow := visibleSapiensBandWindow(frame.Bands, selectedBand)
	scene.drawText(screen, fmt.Sprintf("%s  ·  Regions: %d", bandWindow.label(), len(frame.SapiensEstablishedRegions)), panelX+18, 215, 14, color.White)
	y := float32(bandListOriginY)
	for row := 0; row < bandWindow.count; row++ {
		band := &frame.Bands[bandWindow.indices[row]]
		prefix := "  "
		if band.ID == selectedBand {
			prefix = "› "
		}
		summary := summarizeBandOutcome(band)
		populationChange, healthChange := "", ""
		if summary.available && summary.populationDelta != 0 {
			populationChange = fmt.Sprintf(" (%+d)", summary.populationDelta)
		}
		if summary.available && math.Abs(summary.healthDeltaPoints) >= 0.005 {
			healthChange = " (" + formatHealthDelta(summary.healthDeltaPoints) + ")"
		}
		scene.drawText(screen, fmt.Sprintf("%sB%d  Pop %d%s  Health %.1f%%%s", prefix, band.ID, band.Population, populationChange, band.Health*100, healthChange), panelX+18, y, 10.5, color.White)
		y += bandRowHeight
	}
	if band := selectedBandInFrame(frame, selectedBand); band != nil {
		summary := summarizeBandOutcome(band)
		outcomeY := float32(bandOutcomeOriginY)
		lossColor := color.RGBA{R: 239, G: 174, B: 151, A: 255}
		if summary.populationDelta < 0 {
			scene.drawText(screen, fmt.Sprintf("Pop %+d: %s", summary.populationDelta, formatOutcomeCauses(summary.populationLossCauses, 2)), panelX+18, outcomeY, 9.5, lossColor)
			outcomeY += 14
		}
		if summary.healthDeltaPoints < -0.005 {
			scene.drawText(screen, fmt.Sprintf("Health %s: %s", formatHealthDelta(summary.healthDeltaPoints), formatOutcomeCauses(summary.healthLossCauses, 2)), panelX+18, outcomeY, 9.5, lossColor)
		}
	}
	scene.drawTileInspector(screen, frame, selectedBand, preview)
	scene.drawWorkforceDraft(screen)
	if fieldNotesVisible {
		panelColor, headingColor := fieldNotePanelColors(fieldNote.Celebration)
		heading := "FIELD NOTES"
		if fieldNote.Topic != "" {
			heading += " · " + fieldNote.Topic
		}
		if fieldNote.Celebration {
			heading = "BREAKTHROUGH · " + fieldNote.Topic
		}
		vector.FillRect(screen, panelX+14, fieldNotesPanelOriginY, 324, fieldNotesPanelHeight, panelColor, false)
		if fieldNote.Celebration {
			vector.StrokeRect(screen, panelX+14, fieldNotesPanelOriginY, 324, fieldNotesPanelHeight, 2, headingColor, false)
		}
		headingSuffix := "  [F to hide]"
		if fieldNote.Celebration {
			headingSuffix = "  [F]"
		}
		scene.drawText(screen, heading+headingSuffix, panelX+28, fieldNotesPanelOriginY+9, 10, headingColor)
		body := fieldNote.Introduction
		if fieldNote.Context != "" {
			body += "\nCONTEXT · " + fieldNote.Context
		}
		if fieldNote.GameEffect != "" {
			body += "\nGAME · " + fieldNote.GameEffect
		}
		if fieldNote.Hint != "" {
			body += "\nHINT · " + fieldNote.Hint
		}
		scene.drawText(screen, body, panelX+28, fieldNotesPanelOriginY+29, 9, color.RGBA{R: 202, G: 210, B: 206, A: 255})
	} else {
		label := "F: show Field Notes"
		labelColor := color.RGBA{R: 203, G: 172, B: 104, A: 255}
		if fieldNote.Celebration {
			label = "BREAKTHROUGH: " + fieldNote.Topic + " · F for details"
			labelColor = color.RGBA{R: 255, G: 213, B: 92, A: 255}
		}
		scene.drawText(screen, label, panelX+18, 574, 12, labelColor)
	}
	scene.drawText(screen, "Click: migrate · Arrows: choose · Enter: queue", panelX+18, 636, 10.5, color.White)
	scene.drawText(screen, "Tab/Shift+Tab: bands · Space: turn", panelX+18, 652, 10.5, color.White)
	spatialHint := "N: split"
	if actor := selectedBandInFrame(frame, selectedBand); actor != nil {
		spatialHint = spatialControlHint(interbreedStatus(*actor))
	}
	scene.drawText(screen, spatialHint+" · G: genetics · P: pause", panelX+18, 668, 9.6, color.White)
	scene.drawText(screen, "Quick-save Ctrl/Cmd+S · Manual F1–F3 · Shift+F1–F3 load", panelX+18, 684, 8.2, color.White)
}

func (scene *MapScene) drawWorkforceDraft(screen logicalCanvas) {
	if !scene.workforce.Visible {
		return
	}
	const panelX = float32(922)
	var total uint32
	labels := [...]string{"F", "H", "T", "M", "S"}
	parts := ""
	for role, points := range scene.workforce.AllocationBP {
		total += uint32(points)
		marker := " "
		if gameapi.WorkforceRole(role) == scene.workforce.SelectedRole {
			marker = "›"
		}
		parts += fmt.Sprintf("%s%s %.0f  ", marker, labels[role], float64(points)/100)
	}
	status := "accepted"
	statusColor := color.RGBA{R: 121, G: 195, B: 137, A: 255}
	if scene.workforce.Dirty {
		status = "DIRTY"
		statusColor = color.RGBA{R: 245, G: 202, B: 92, A: 255}
	}
	if !scene.workforce.Valid {
		status = fmt.Sprintf("%+.0f%%", (float64(total)-10_000)/100)
		statusColor = color.RGBA{R: 232, G: 112, B: 92, A: 255}
	}
	scene.drawText(screen, "WORKFORCE · W role · [/] edit · A apply · D discard", panelX+8, workforcePanelOriginY, 7.2, color.RGBA{R: 167, G: 184, B: 181, A: 255})
	scene.drawText(screen, parts, panelX+8, workforcePanelOriginY+13, 7.8, color.White)
	scene.drawText(screen, status, panelX+276, workforcePanelOriginY+13, 7.8, statusColor)
}

func fieldNotePanelColors(celebration bool) (color.RGBA, color.RGBA) {
	if celebration {
		return color.RGBA{R: 45, G: 39, B: 24, A: 255}, color.RGBA{R: 255, G: 213, B: 92, A: 255}
	}
	return color.RGBA{R: 19, G: 28, B: 34, A: 255}, color.RGBA{R: 203, G: 172, B: 104, A: 255}
}

func (scene *MapScene) drawTileInspector(screen logicalCanvas, frame *gameapi.Frame, selectedBand gameapi.BandID, preview MigrationPreview) {
	const panelX = float32(922)
	const currentX = panelX + 8
	const targetX = panelX + 167

	vector.FillRect(screen, panelX, tileInspectorOriginY, 324, tileInspectorHeight, color.RGBA{R: 18, G: 27, B: 33, A: 255}, false)
	vector.StrokeRect(screen, panelX, tileInspectorOriginY, 324, tileInspectorHeight, 1, color.RGBA{R: 70, G: 91, B: 97, A: 255}, false)
	scene.drawText(screen, "TILE LIVEABILITY · cyan reachable · gold best · red queued", currentX, tileInspectorOriginY+3, 7.8, color.RGBA{R: 203, G: 172, B: 104, A: 255})

	band := selectedBandInFrame(frame, selectedBand)
	current := currentTileSummary(frame, band)
	target := targetTileSummary(frame, band, preview)
	currentHeading, targetHeading := current.heading, target.heading
	if current.showDetails {
		currentHeading += " · " + current.status
	}
	if target.showDetails {
		targetHeading += " · " + target.status
	}
	scene.drawText(screen, currentHeading, currentX, tileInspectorOriginY+15, 8.3, color.RGBA{R: 245, G: 202, B: 92, A: 255})
	targetColor := color.RGBA{R: 167, G: 184, B: 181, A: 255}
	if target.showDetails {
		targetColor = color.RGBA{R: 87, G: 211, B: 211, A: 255}
	}
	if band != nil && band.HasQueuedMigration && (!preview.Visible || preview.BandID != band.ID) {
		targetColor = color.RGBA{R: 255, G: 106, B: 91, A: 255}
	}
	scene.drawText(screen, targetHeading, targetX, tileInspectorOriginY+15, 8.3, targetColor)
	vector.StrokeLine(screen, targetX-8, tileInspectorOriginY+15, targetX-8, tileInspectorOriginY+104, 0.7, color.RGBA{R: 67, G: 82, B: 87, A: 220}, false)

	currentLines, targetLines := liveabilityLines(current), liveabilityLines(target)
	for index := range currentLines {
		y := tileInspectorOriginY + 27 + float32(index)*tileInspectorRowGap
		if currentLines[index] != "" {
			lineColor := color.RGBA{R: 220, G: 225, B: 218, A: 255}
			if index == archaicPresenceLineIndex && current.archaicBandCount > 0 {
				lineColor = archaicBandMarkerColor
			}
			scene.drawText(screen, currentLines[index], currentX, y, tileInspectorTextSize, lineColor)
		}
		if targetLines[index] != "" {
			lineColor := color.RGBA{R: 220, G: 225, B: 218, A: 255}
			if index == archaicPresenceLineIndex && target.archaicBandCount > 0 {
				lineColor = archaicBandMarkerColor
			}
			scene.drawText(screen, targetLines[index], targetX, y, tileInspectorTextSize, lineColor)
		}
	}

	footer := "Arrows update target · Enter queues · Esc clears"
	footerColor := color.RGBA{R: 145, G: 163, B: 161, A: 255}
	if band != nil {
		if line := interbreedPanelLine(interbreedStatus(*band)); line != "" {
			footer, footerColor = line, interbreedMarkerColor
		}
	}
	scene.drawText(screen, footer, currentX, interbreedPanelLineY, 8, footerColor)
}

func formatHealthDelta(points float64) string {
	if math.Abs(points) < 0.1 {
		return fmt.Sprintf("%+.2fpp", points)
	}
	return fmt.Sprintf("%+.1fpp", points)
}

func (scene *MapScene) drawResearchKeys(screen logicalCanvas, frame *gameapi.Frame, selectedBand gameapi.BandID) {
	const top = float32(656)
	vector.FillRect(screen, 20, top, 864, 52, color.RGBA{R: 25, G: 35, B: 42, A: 245}, false)
	scene.drawText(screen, "RESEARCH KEYS  ·  gold current  ·  green learned  ·  grey locked", 30, top+4, 10, color.RGBA{R: 203, G: 172, B: 104, A: 255})
	band := selectedBandInFrame(frame, selectedBand)
	rowOneX := [...]float32{30, 196, 362, 528, 694}
	rowTwoX := [...]float32{30, 238, 446, 654}
	for technology := gameapi.Tech(0); technology < gameapi.TechCount; technology++ {
		x, y := float32(0), top+21
		if technology < 5 {
			x = rowOneX[technology]
		} else {
			x, y = rowTwoX[technology-5], top+37
		}
		labelColor := color.RGBA{R: 116, G: 128, B: 131, A: 255}
		if band != nil {
			option := band.ResearchOptions[technology]
			switch {
			case option.Current:
				labelColor = color.RGBA{R: 245, G: 202, B: 92, A: 255}
			case option.Acquired:
				labelColor = color.RGBA{R: 121, G: 195, B: 137, A: 255}
			case option.Available:
				labelColor = color.RGBA{R: 231, G: 235, B: 229, A: 255}
			}
		}
		scene.drawText(screen, fmt.Sprintf("%d %s", technology+1, technology.String()), x, y, 10, labelColor)
	}
}

func selectedBandInFrame(frame *gameapi.Frame, selectedBand gameapi.BandID) *gameapi.Band {
	for index := range frame.Bands {
		if frame.Bands[index].ID == selectedBand {
			return &frame.Bands[index]
		}
	}
	return nil
}

func (scene *MapScene) drawText(destination logicalCanvas, value string, x, y, size float32, textColor color.Color) {
	options := &text.DrawOptions{}
	scale := float64(destination.scale)
	options.GeoM.Translate(float64(x)*scale, float64(y)*scale)
	options.ColorScale.ScaleWithColor(textColor)
	options.LineSpacing = float64(size) * 1.35 * scale
	text.Draw(destination.image, value, &text.GoTextFace{Source: scene.faceSource, Size: float64(size) * scale}, options)
}

func climateBiomeColor(biome gameapi.Biome, _ float64) color.RGBA {
	return [gameapi.BiomeCount]color.RGBA{
		{R: 55, G: 105, B: 66, A: 255}, {R: 126, G: 137, B: 70, A: 255}, {R: 112, G: 126, B: 79, A: 255},
		{R: 103, G: 104, B: 94, A: 255}, {R: 166, G: 134, B: 77, A: 255}, {R: 150, G: 166, B: 169, A: 255},
	}[biome]
}

func clampRender(value float64) float64 {
	if value < 0 {
		return 0
	}
	if value > 1 {
		return 1
	}
	return value
}
