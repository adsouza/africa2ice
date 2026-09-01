package render

import (
	"bytes"
	"fmt"
	"image/color"
	"math"
	"strings"

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
	mapTileSize            = 8
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
	fieldNoteWrapLimit     = 78
	visibleFieldNoteLines  = 5
	workforcePanelOriginY  = 604
	workforceRoleOriginY   = 612
	workforceRoleRowGap    = 9
	controlsDividerY       = 641
	controlsReferenceY     = 645
	controlsReferenceGap   = 13
	bottomInspectorOriginY = 590
	bottomInspectorHeight  = 118
)

var (
	unexploredTileColor    = color.RGBA{R: 6, G: 11, B: 15, A: 255}
	archaicBandMarkerColor = color.RGBA{R: 201, G: 103, B: 82, A: 255}
	escarpmentColor        = color.RGBA{R: 220, G: 142, B: 88, A: 255}
	queuedMigrationColor   = color.RGBA{R: 232, G: 72, B: 72, A: 255}
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
	fieldNoteScroll int
	interbreedFocus gameapi.BandID
	hover           TileHover
}

type mapFrameKey struct {
	frame             *gameapi.Frame
	selectedBand      gameapi.BandID
	preview           MigrationPreview
	hover             TileHover
	notice            string
	fieldNote         FieldNote
	fieldNotesVisible bool
	fieldNoteScroll   int
	interbreedFocus   gameapi.BandID
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

// TileHover is UI-local pointer focus. It never changes selection or campaign
// state, and the inspector gives explicit migration choices precedence over it.
type TileHover struct {
	TileID  gameapi.TileID
	Visible bool
}

// WorkforceDraft is UI-local editor state. Accepted allocations continue to
// come from the frame until the complete draft is explicitly applied.
type WorkforceDraft struct {
	Visible      bool
	BandID       gameapi.BandID
	Population   uint32
	AllocationBP [gameapi.AssignmentCount]uint16
	SelectedRole gameapi.WorkforceRole
	Dirty        bool
	Valid        bool
}

type MenuOverlay struct {
	Visible           bool
	Heading           string
	Help              string
	Lines             [8]string
	LineCount         int
	Selected          int
	Settings          bool
	SettingsDisabled  bool
	MasterVolume      float64
	Muted             bool
	FieldNotesVisible bool
}

// FieldNote is UI-local presentation content. It is never simulation or save
// state; the renderer only lays out the already-selected entry.
type FieldNote struct {
	Topic        string
	Introduction string
	Context      string
	GameEffect   string
	Hint         string
	References   string
	Celebration  bool
}

// FieldNoteMaxScroll returns the greatest meaningful scroll offset for the
// renderer's current Field Notes layout.
func FieldNoteMaxScroll(note FieldNote) int {
	return max(0, len(fieldNoteLines(note))-visibleFieldNoteLines)
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
func (scene *MapScene) SetFieldNoteScroll(scroll int) {
	scene.fieldNoteScroll = max(0, scroll)
}
func (scene *MapScene) SetInterbreedFocus(target gameapi.BandID) { scene.interbreedFocus = target }
func (scene *MapScene) SetTileHover(hover TileHover)             { scene.hover = hover }

func (scene *MapScene) Draw(screen *ebiten.Image, frame *gameapi.Frame, selectedBand gameapi.BandID, preview MigrationPreview, notice string, fieldNote FieldNote, fieldNotesVisible bool, ending EndScene, resizeRequired bool) {
	if frame == nil {
		screen.Fill(color.RGBA{R: 15, G: 22, B: 29, A: 255})
		return
	}
	key := mapFrameKey{
		frame: frame, selectedBand: selectedBand, preview: preview, hover: scene.hover, notice: notice,
		fieldNote: fieldNote, fieldNotesVisible: fieldNotesVisible, ending: ending,
		fieldNoteScroll: scene.fieldNoteScroll,
		interbreedFocus: scene.interbreedFocus,
		workforce:       scene.workforce, resizeRequired: resizeRequired, overlay: scene.overlay,
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
		vector.FillRect(screen, 28, 82, 650, 30, color.RGBA{R: 26, G: 38, B: 45, A: 240}, false)
		scene.drawText(screen, notice, 40, 89, 14, color.RGBA{R: 239, G: 220, B: 178, A: 255})
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
	if scene.overlay.Settings {
		disabledColor := color.RGBA{R: 92, G: 106, B: 109, A: 255}
		activeColor := color.RGBA{R: 203, G: 172, B: 104, A: 255}
		chromeColor := color.RGBA{R: 91, G: 110, B: 117, A: 255}
		if scene.overlay.SettingsDisabled {
			activeColor, chromeColor = disabledColor, disabledColor
		}
		const sliderLeft, sliderRight, sliderY = float32(650), float32(870), float32(239)
		vector.StrokeLine(screen, sliderLeft, sliderY, sliderRight, sliderY, 4, chromeColor, false)
		knobX := sliderLeft + float32(clampRender(scene.overlay.MasterVolume))*(sliderRight-sliderLeft)
		vector.FillCircle(screen, knobX, sliderY, 7, activeColor, true)
		for row, checked := range []bool{scene.overlay.Muted, scene.overlay.FieldNotesVisible} {
			boxY := float32(267 + row*36)
			vector.StrokeRect(screen, 650, boxY, 16, 16, 1.5, chromeColor, false)
			if checked {
				vector.StrokeLine(screen, 653, boxY+8, 657, boxY+13, 2, activeColor, false)
				vector.StrokeLine(screen, 657, boxY+13, 664, boxY+3, 2, activeColor, false)
			}
		}
	}
	scene.drawText(screen, scene.overlay.Help, x+28, y+height-42, 11, color.RGBA{R: 167, G: 184, B: 181, A: 255})
}

func MenuOverlayRowAt(x, y, lineCount int) int {
	const left, top, width = 340, 150, 600
	if x < left+20 || x >= left+width-20 || y < top+73 {
		return -1
	}
	row := (y - (top + 73)) / 36
	if row < 0 || row >= lineCount || y >= top+73+(row+1)*36 {
		return -1
	}
	return row
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
		drawMigrationArrow(screen, fromX, fromY, toX, toY, queuedMigrationColor)
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
	vector.FillRect(screen, panelX+273, 78, 65, 24, color.RGBA{R: 35, G: 51, B: 58, A: 255}, false)
	scene.drawText(screen, "▣ NOTES", panelX+282, 84, 8.5, color.RGBA{R: 203, G: 172, B: 104, A: 255})
	scene.drawText(screen, fmt.Sprintf("%d BP  ·  Turn %d/400", frame.YearBP, frame.Turn), panelX+18, 124, 16, color.White)
	scene.drawText(screen, campaignEraLabel(frame.Era), panelX+18, 147, 11.5, color.RGBA{R: 183, G: 199, B: 194, A: 255})
	scene.drawText(screen, frame.Season.String()+"  ·  "+frame.Climate.Epoch.String(), panelX+18, 164, 11.5, color.RGBA{R: 183, G: 199, B: 194, A: 255})
	if warning := macroWarningLabel(frame.MacroEpisodes); warning != "" {
		scene.drawText(screen, warning, panelX+18, 180, 9.5, color.RGBA{R: 239, G: 151, B: 104, A: 255})
	}
	var totalPopulation uint64
	for _, band := range frame.Bands {
		if band.Species == gameapi.HomoSapiens {
			totalPopulation += uint64(band.Population)
		}
	}
	scene.drawText(screen, fmt.Sprintf("Homo sapiens: %d", totalPopulation), panelX+18, 195, 15, color.RGBA{R: 245, G: 202, B: 92, A: 255})
	bandWindow := visibleSapiensBandWindow(frame.Bands, selectedBand)
	scene.drawText(screen, fmt.Sprintf("%s  ·  Regions %d", bandWindow.label(), len(frame.SapiensEstablishedRegions)), panelX+18, 215, 11.5, color.White)
	scene.drawText(screen, "! DANGER  !! SUFFERING", panelX+178, 217, 7.5, color.RGBA{R: 224, G: 173, B: 112, A: 255})
	scene.drawText(screen, "ACTION", panelX+286, 217, 8, color.RGBA{R: 203, G: 172, B: 104, A: 255})
	y := float32(bandListOriginY)
	for row := 0; row < bandWindow.count; row++ {
		band := &frame.Bands[bandWindow.indices[row]]
		condition := conditionForSapiensBand(*band)
		conditionMarker := ""
		conditionColor := color.RGBA{R: 167, G: 184, B: 181, A: 255}
		switch condition {
		case bandConditionDanger:
			conditionMarker = "!"
			conditionColor = color.RGBA{R: 237, G: 176, B: 84, A: 255}
			vector.FillRect(screen, panelX+14, y-2, 266, 14, color.RGBA{R: 91, G: 64, B: 30, A: 100}, false)
		case bandConditionSuffering:
			conditionMarker = "!!"
			conditionColor = color.RGBA{R: 247, G: 137, B: 119, A: 255}
			vector.FillRect(screen, panelX+14, y-2, 266, 14, color.RGBA{R: 92, G: 38, B: 35, A: 120}, false)
		}
		if band.ID == selectedBand {
			scene.drawText(screen, "›", panelX+18, y, 10.5, color.White)
		}
		if conditionMarker != "" {
			scene.drawText(screen, conditionMarker, panelX+28, y, 8.5, conditionColor)
		}
		summary := summarizeBandOutcome(band)
		populationChange, healthChange := "", ""
		if summary.available && summary.populationDelta != 0 {
			populationChange = fmt.Sprintf(" (%+d)", summary.populationDelta)
		}
		if summary.available && math.Abs(summary.healthDeltaPoints) >= 0.005 {
			healthChange = " (" + formatHealthDelta(summary.healthDeltaPoints) + ")"
		}
		scene.drawText(screen, fmt.Sprintf("B%d  Pop %d%s  Health %.1f%%%s", band.ID, band.Population, populationChange, band.Health*100, healthChange), panelX+42, y, 10.5, color.White)
		actionLabel := sapiensBandActionLabel(*band)
		actionColor := color.RGBA{R: 167, G: 184, B: 181, A: 255}
		switch actionLabel {
		case bandActionMoveSet:
			actionColor = queuedMigrationColor
		case bandActionInterbreed:
			actionColor = interbreedMarkerColor
		case bandActionDone:
			actionColor = color.RGBA{R: 245, G: 202, B: 92, A: 255}
		}
		scene.drawText(screen, actionLabel, panelX+286, y+1, 8, actionColor)
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
		lines := fieldNoteLines(fieldNote)
		scroll := min(scene.fieldNoteScroll, FieldNoteMaxScroll(fieldNote))
		visibleEnd := min(len(lines), scroll+visibleFieldNoteLines)
		scene.drawText(screen, strings.Join(lines[scroll:visibleEnd], "\n"), panelX+28, fieldNotesPanelOriginY+29, 8.2, color.RGBA{R: 202, G: 210, B: 206, A: 255})
		if len(lines) > visibleFieldNoteLines {
			scene.drawText(screen, fmt.Sprintf("SCROLL %d/%d · wheel or PgUp/PgDn", scroll+1, len(lines)-visibleFieldNoteLines+1), panelX+147, fieldNotesPanelOriginY+91, 7, color.RGBA{R: 145, G: 163, B: 161, A: 255})
		}
		scene.drawText(screen, "RECENT EVENTS", panelX+28, fieldNotesPanelOriginY+104, 7.5, headingColor)
		scene.drawText(screen, strings.Join(recentEventLines(frame.Events, 2, 52), "\n"), panelX+28, fieldNotesPanelOriginY+116, 7.2, color.RGBA{R: 184, G: 198, B: 194, A: 255})
	} else {
		label := "F: show Field Notes"
		labelColor := color.RGBA{R: 203, G: 172, B: 104, A: 255}
		if fieldNote.Celebration {
			label = "BREAKTHROUGH: " + fieldNote.Topic + " · F for details"
			labelColor = color.RGBA{R: 255, G: 213, B: 92, A: 255}
		}
		scene.drawText(screen, "RECENT EVENT", panelX+18, 548, 8.5, color.RGBA{R: 167, G: 184, B: 181, A: 255})
		scene.drawText(screen, recentEventLines(frame.Events, 1, 55)[0], panelX+18, 562, 7.6, color.RGBA{R: 202, G: 210, B: 206, A: 255})
		scene.drawText(screen, label, panelX+18, 584, 10.5, labelColor)
	}
	scene.drawControlsReference(screen, frame, selectedBand)
}

func (scene *MapScene) drawControlsReference(screen logicalCanvas, frame *gameapi.Frame, selectedBand gameapi.BandID) {
	const panelX = float32(908)
	vector.StrokeLine(screen, panelX+14, controlsDividerY, panelX+338, controlsDividerY, 1, color.RGBA{R: 58, G: 76, B: 82, A: 210}, false)
	scene.drawText(screen, "Click: migrate · Arrows: choose · Enter: queue", panelX+18, controlsReferenceY, 10.5, color.White)
	scene.drawText(screen, "Tab/Shift+Tab: bands · Space: turn", panelX+18, controlsReferenceY+controlsReferenceGap, 10.5, color.White)
	spatialHint := "N: split"
	if actor := selectedBandInFrame(frame, selectedBand); actor != nil {
		spatialHint = spatialControlHint(interbreedStatus(*actor))
	}
	scene.drawText(screen, spatialHint+" · G: genetics · Esc: menu", panelX+18, controlsReferenceY+2*controlsReferenceGap, 9.6, color.White)
	scene.drawText(screen, "Quick-save Ctrl/Cmd+S · Manual F1–F3 · Shift+F1–F3 load", panelX+18, controlsReferenceY+3*controlsReferenceGap, 8.2, color.White)
}

func fieldNoteLines(note FieldNote) []string {
	body := "SUMMARY · " + note.Introduction
	if note.Context != "" {
		body += "\nHISTORICAL CONTEXT · " + note.Context
	}
	if note.GameEffect != "" {
		body += "\nGAME ABSTRACTION · " + note.GameEffect
	}
	if note.Hint != "" {
		body += "\nHINT · " + note.Hint
	}
	if note.References != "" {
		body += "\nREFERENCES · " + note.References
	}
	return wrapTextLines(body, fieldNoteWrapLimit)
}

func recentEventLines(events []gameapi.Event, limit, maxRunes int) []string {
	if limit <= 0 {
		return nil
	}
	lines := make([]string, 0, limit)
	for index := len(events) - 1; index >= 0 && len(lines) < limit; index-- {
		event := events[index]
		summary := strings.TrimSpace(event.Summary)
		if summary == "" {
			summary = event.Kind.String()
		}
		lines = append(lines, truncateRunes(fmt.Sprintf("T%d · %s · %s", event.Turn, event.Kind, summary), maxRunes))
	}
	if len(lines) == 0 {
		lines = append(lines, "No campaign events yet.")
	}
	return lines
}

func truncateRunes(value string, limit int) string {
	runes := []rune(value)
	if limit <= 0 || len(runes) <= limit {
		return value
	}
	if limit == 1 {
		return "…"
	}
	return string(runes[:limit-1]) + "…"
}

func wrapTextLines(value string, limit int) []string {
	if limit <= 0 {
		return strings.Split(value, "\n")
	}
	lines := make([]string, 0)
	for _, paragraph := range strings.Split(value, "\n") {
		words := strings.Fields(paragraph)
		if len(words) == 0 {
			lines = append(lines, "")
			continue
		}
		line := words[0]
		for _, word := range words[1:] {
			if len([]rune(line))+1+len([]rune(word)) <= limit {
				line += " " + word
				continue
			}
			lines = append(lines, line)
			line = word
		}
		lines = append(lines, line)
	}
	return lines
}

func FieldNotesToggleContains(x, y int) bool {
	return x >= 1181 && x < 1246 && y >= 78 && y < 102
}

func FieldNotesPanelContains(x, y int) bool {
	return x >= 922 && x < 1246 && y >= int(fieldNotesPanelOriginY) && y < int(fieldNotesPanelOriginY+fieldNotesPanelHeight)
}

func (scene *MapScene) drawWorkforceDraft(screen logicalCanvas) {
	if !scene.workforce.Visible {
		return
	}
	const panelX = float32(922)
	labels := [...]string{"Foraging", "Hunt/fish", "Toolcraft", "Megafauna", "Shelter/care"}
	parts := [gameapi.AssignmentCount]string{}
	for role, points := range scene.workforce.AllocationBP {
		marker := " "
		if gameapi.WorkforceRole(role) == scene.workforce.SelectedRole {
			marker = "›"
		}
		workers := float64(scene.workforce.Population) * float64(points) / 10_000
		parts[role] = fmt.Sprintf("%s%s %.0f%% · %.1fp", marker, labels[role], float64(points)/100, workers)
	}
	status, statusColor := workforceDraftStatus(scene.workforce)
	scene.drawText(screen, "WORKFORCE · W role/context · [/] edit · A apply · D discard", panelX+8, workforcePanelOriginY-2, 6.7, color.RGBA{R: 167, G: 184, B: 181, A: 255})
	for role := range parts {
		column := role % 2
		row := role / 2
		x := panelX + 8 + float32(column)*158
		y := float32(workforceRoleOriginY + row*workforceRoleRowGap)
		scene.drawText(screen, parts[role], x, y, 6.8, color.White)
	}
	if status != "" {
		scene.drawText(screen, status, panelX+276, workforceRoleOriginY+2*workforceRoleRowGap, 6.8, statusColor)
	}
}

func workforceDraftStatus(draft WorkforceDraft) (string, color.RGBA) {
	if !draft.Valid {
		var total uint32
		for _, points := range draft.AllocationBP {
			total += uint32(points)
		}
		return fmt.Sprintf("%+.0f%%", (float64(total)-10_000)/100), color.RGBA{R: 232, G: 112, B: 92, A: 255}
	}
	if draft.Dirty {
		return "DIRTY", color.RGBA{R: 245, G: 202, B: 92, A: 255}
	}
	return "", color.RGBA{}
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
	target := targetTileSummary(frame, band, preview, scene.hover)
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
	const left, top, width = float32(20), float32(bottomInspectorOriginY), float32(864)
	vector.FillRect(screen, left, top, width, bottomInspectorHeight, color.RGBA{R: 20, G: 30, B: 36, A: 250}, false)
	vector.StrokeRect(screen, left, top, width, bottomInspectorHeight, 1, color.RGBA{R: 70, G: 91, B: 97, A: 255}, false)
	band := selectedBandInFrame(frame, selectedBand)
	scene.drawResearchDAG(screen, band)
	scene.drawSelectedBandInspector(screen, frame, band)
}

type researchNodePoint struct{ x, y float32 }

var researchNodePoints = [gameapi.TechCount]researchNodePoint{
	gameapi.Firecraft:          {x: 30, y: 610},
	gameapi.HaftedTools:        {x: 196, y: 610},
	gameapi.PlantKnowledge:     {x: 362, y: 610},
	gameapi.TailoredClothing:   {x: 196, y: 635},
	gameapi.CordageAndNets:     {x: 362, y: 635},
	gameapi.Campcraft:          {x: 30, y: 660},
	gameapi.MedicinalKnowledge: {x: 196, y: 660},
	gameapi.Trapping:           {x: 362, y: 660},
	gameapi.CoastalNavigation:  {x: 362, y: 685},
}

func (scene *MapScene) drawResearchDAG(screen logicalCanvas, band *gameapi.Band) {
	scene.drawText(screen, "RESEARCH DAG · keys 1–9 · basic survival remains available", 30, bottomInspectorOriginY+3, 8.5, color.RGBA{R: 203, G: 172, B: 104, A: 255})
	if band != nil {
		for technology := gameapi.Tech(0); technology < gameapi.TechCount; technology++ {
			to := researchNodePoints[technology]
			for prerequisite := gameapi.Tech(0); prerequisite < gameapi.TechCount; prerequisite++ {
				if band.ResearchOptions[technology].PrerequisiteMask&(1<<prerequisite) == 0 {
					continue
				}
				from := researchNodePoints[prerequisite]
				vector.StrokeLine(screen, from.x+76, from.y+14, to.x+76, to.y, 0.8, color.RGBA{R: 83, G: 102, B: 106, A: 210}, false)
			}
		}
	}
	for technology := gameapi.Tech(0); technology < gameapi.TechCount; technology++ {
		point := researchNodePoints[technology]
		option := gameapi.ResearchOption{}
		progress := 0.0
		if band != nil {
			option = band.ResearchOptions[technology]
			progress = band.ResearchProgress[technology]
		}
		border, fill, textColor := researchNodeColors(option)
		vector.FillRect(screen, point.x, point.y, 152, 20, fill, false)
		vector.StrokeRect(screen, point.x, point.y, 152, 20, 0.9, border, false)
		scene.drawText(screen, fmt.Sprintf("%d %s", technology+1, technology), point.x+4, point.y+1, 7.2, textColor)
		status := fmt.Sprintf("%.0f/%.0f", progress, option.Cost)
		switch {
		case option.Acquired:
			status += " · learned"
		case option.Current:
			status += " · current"
		case !option.Available && band != nil:
			status += " · needs " + missingPrerequisiteLabel(option, band.AcquiredTech)
		case band != nil && band.Species == gameapi.ArchaicHominin:
			status += " · computer"
		}
		scene.drawText(screen, status, point.x+4, point.y+10, 6.2, textColor)
	}
}

func researchNodeColors(option gameapi.ResearchOption) (color.RGBA, color.RGBA, color.RGBA) {
	border := color.RGBA{R: 82, G: 95, B: 98, A: 255}
	fill := color.RGBA{R: 28, G: 39, B: 45, A: 255}
	textColor := color.RGBA{R: 122, G: 135, B: 137, A: 255}
	switch {
	case option.Current:
		border, textColor = color.RGBA{R: 245, G: 202, B: 92, A: 255}, color.RGBA{R: 255, G: 225, B: 148, A: 255}
	case option.Acquired:
		border, textColor = color.RGBA{R: 121, G: 195, B: 137, A: 255}, color.RGBA{R: 168, G: 223, B: 178, A: 255}
	case option.Available:
		border, textColor = color.RGBA{R: 190, G: 204, B: 199, A: 255}, color.RGBA{R: 231, G: 235, B: 229, A: 255}
	}
	return border, fill, textColor
}

func missingPrerequisiteLabel(option gameapi.ResearchOption, acquired uint16) string {
	missing := option.PrerequisiteMask &^ acquired
	label := ""
	for technology := gameapi.Tech(0); technology < gameapi.TechCount; technology++ {
		if missing&(1<<technology) == 0 {
			continue
		}
		if label != "" {
			label += "+"
		}
		label += researchShortName(technology)
	}
	return label
}

func researchShortName(technology gameapi.Tech) string {
	return [...]string{"Fire", "Haft", "Plants", "Clothes", "Cordage", "Camp", "Medicine", "Traps", "Navigation"}[technology]
}

func (scene *MapScene) drawSelectedBandInspector(screen logicalCanvas, frame *gameapi.Frame, band *gameapi.Band) {
	const x = float32(536)
	vector.StrokeLine(screen, x, bottomInspectorOriginY, x, bottomInspectorOriginY+bottomInspectorHeight, 1, color.RGBA{R: 70, G: 91, B: 97, A: 255}, false)
	if band == nil {
		scene.drawText(screen, "SELECTED BAND · none", x+10, bottomInspectorOriginY+8, 9, color.White)
		return
	}
	control := "PLAYER CONTROLLED"
	headingColor := color.RGBA{R: 245, G: 202, B: 92, A: 255}
	if band.Species == gameapi.ArchaicHominin {
		control = "COMPUTER CONTROLLED · READ ONLY"
		headingColor = archaicBandMarkerColor
	}
	scene.drawText(screen, fmt.Sprintf("BAND %d · %s", band.ID, control), x+10, bottomInspectorOriginY+5, 8.5, headingColor)
	scene.drawText(screen, fmt.Sprintf("%s · pop %d · health %.1f%% · stored %.1f FU", band.Species, band.Population, band.Health*100, band.StoredFood), x+10, bottomInspectorOriginY+19, 7.7, color.White)
	foodLine := "Last turn food: unavailable"
	if band.LastFoodReport.Turn > 0 {
		foodLine = fmt.Sprintf("Turn %d food: need %.1f · ate %.1f · short %.1f (%.1f%%)", band.LastFoodReport.Turn, band.LastFoodReport.RequiredFU, band.LastFoodReport.ConsumedFU(), band.LastFoodReport.DeficitFU, band.LastFoodReport.DeficitFraction()*100)
	}
	scene.drawText(screen, foodLine, x+10, bottomInspectorOriginY+33, 7.2, color.RGBA{R: 202, G: 210, B: 206, A: 255})
	m := band.LastMortality
	mortalityLine := "Last turn mortality: unavailable"
	if band.LastOutcomeReport.Turn > 0 {
		mortalityLine = fmt.Sprintf("Deaths: starv %.2f · season %.2f · chronic %.2f · macro %.2f · acute %.2f", m.Starvation, m.Seasonal, m.Chronic, m.Macro, m.Acute)
	}
	scene.drawText(screen, mortalityLine, x+10, bottomInspectorOriginY+47, 6.9, color.RGBA{R: 202, G: 210, B: 206, A: 255})
	researchLine := "Research: no target"
	if band.HasResearchTarget {
		option := band.ResearchOptions[band.ResearchTarget]
		researchLine = fmt.Sprintf("Research: %s %.1f/%.0f · own gain +%.1f/turn", band.ResearchTarget, band.ResearchProgress[band.ResearchTarget], option.Cost, band.OriginalResearchGainPreview)
	}
	scene.drawText(screen, researchLine, x+10, bottomInspectorOriginY+61, 7.2, color.RGBA{R: 203, G: 172, B: 104, A: 255})
	traits := band.HeritableState
	temperature, elevation, latitude, biome := 0.0, 0.0, 0.0, "unknown"
	fauna := "none"
	if frame != nil && int(band.TileID) < len(frame.Tiles) {
		tile := frame.Tiles[band.TileID]
		temperature, elevation, latitude, biome = tile.LocalTemperatureC, tile.ElevationKm, tile.Latitude, tile.Biome.String()
		fauna = dominantFaunaOpportunity(tile.Fauna)
	}
	traitHeadingY := float32(bottomInspectorOriginY + 76)
	if len(band.InterbreedCandidateIDs) > 0 || band.HasInterbreedTarget {
		interbreed := "Interbreed targets: "
		for index, candidate := range band.InterbreedCandidateIDs {
			if index > 0 {
				interbreed += ", "
			}
			marker := ""
			if candidate == scene.interbreedFocus {
				marker = "›"
			}
			interbreed += fmt.Sprintf("%sB%d", marker, candidate)
		}
		if band.HasInterbreedTarget {
			interbreed = fmt.Sprintf("Interbreeding accepted with B%d", band.InterbreedTargetID)
		} else {
			interbreed += " · J choose · I accept"
		}
		scene.drawText(screen, interbreed, x+10, traitHeadingY, 6.8, interbreedMarkerColor)
		traitHeadingY += 11
	}
	scene.drawText(screen, "HERITABLE VARIANTS · G cycles scientific context", x+10, traitHeadingY, 7, color.RGBA{R: 167, G: 184, B: 181, A: 255})
	scene.drawText(screen, fmt.Sprintf("Cold %.3f @ %.0f°C · Alt %.3f @ %.1fkm · Immune %.3f @ %s", traits[gameapi.ColdAdaptation], temperature, traits[gameapi.HighAltitudeAdaptation], elevation, traits[gameapi.InnateImmuneReactivity], biome), x+10, traitHeadingY+13, 6.3, color.White)
	scene.drawText(screen, fmt.Sprintf("Arid %.3f @ %.0f°C · Pigment %.3f @ %.0f° · Fat %.3f @ %s", traits[gameapi.AridClimateAdaptation], temperature, traits[gameapi.PigmentationLevel], latitude, traits[gameapi.FattyAcidMetabolism], fauna), x+10, traitHeadingY+26, 6.3, color.White)
}

func campaignEraLabel(era gameapi.CampaignEra) string {
	ranges := [...]string{"80,000–50,000 BP", "50,000–35,000 BP", "35,000–25,000 BP", "25,000–20,000 BP"}
	if era >= gameapi.CampaignEraCount {
		return era.String()
	}
	return era.String() + " era  ·  " + ranges[era]
}

func macroWarningLabel(episodes []gameapi.MacroEpisodeSummary) string {
	for _, episode := range episodes {
		switch {
		case episode.Current:
			return "ACTIVE · " + episode.Episode.String()
		case episode.Warned:
			return "WARNING · " + episode.Episode.String()
		}
	}
	return ""
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
