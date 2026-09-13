package render

import (
	"bytes"
	"fmt"
	"image"
	"image/color"
	"math"
	"strings"

	"github.com/adsouza/africa2ice/pkg/gameapi"
	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/text/v2"
	"golang.org/x/image/font/gofont/goregular"
)

const (
	TerrainGridWidth   = 96
	TerrainGridHeight  = 64
	mapOriginX         = 20
	mapOriginY         = 74
	mapTileSize        = 8
	mapPixelWidth      = TerrainGridWidth * mapTileSize
	mapPixelHeight     = TerrainGridHeight * mapTileSize
	mapLegendOriginY   = 48
	mapLegendHeight    = 25
	noticeBoxX         = 28
	noticeBoxY         = 82
	noticeBoxWidth     = 650
	noticeBoxMinHeight = 30
	noticeTextX        = 40
	noticeTextY        = 89
	noticeFontSize     = 14
	noticeTextMaxWidth = noticeBoxWidth - 2*(noticeTextX-noticeBoxX)
	textLineSpacing    = 1.35
	// An 8x8 swatch cannot host a legible pictograph; 12 can, at the em size
	// legendGlyphSize measures out below. The swatch's y offset moved from the
	// original 4 to 2 so its taller 12 DIP body still clears the meaning text
	// drawn below it in the same row.
	legendSwatchX    = float32(4)
	legendSwatchY    = float32(2)
	legendSwatchSize = float32(12)
	legendLabelX     = float32(19)
	legendMeaningY   = float32(15)
	// legendMeaningSize is the meaning line's font size. legend_test.go
	// checks that line's box against the row height, so it has to be a name
	// rather than the same literal spelled in two places.
	legendMeaningSize = float32(7)
	// legendGlyphSize applies the map's measured ink-to-em ratio to the
	// swatch, so the legend glyph clears the swatch's 0.7 stroke the same way
	// a tile glyph clears its neighbours. At the previous 11 the widest ink
	// measured 13.00 DIP inside a 12 DIP swatch -- mountainous highlands
	// painted over the stroke and into the label gutter, and every biome
	// crossed the top edge because ebiten floors a glyph's baseline to a
	// whole physical pixel, which shifts the ink up by up to 1 DIP at DPR 1.
	// At 8.4 the widest ink measures 9.93 DIP, leaving ~1 DIP per side: more
	// than that quantization can spend. TestLegendGlyphInkStaysInsideItsSwatch
	// pins it.
	legendGlyphSize = legendSwatchSize * glyphCellFraction
)

var (
	unexploredTileColor    = color.RGBA{R: 6, G: 11, B: 15, A: 255}
	archaicBandMarkerColor = color.RGBA{R: 201, G: 103, B: 82, A: 255}
	sapiensBandMarkerColor = color.RGBA{R: 245, G: 202, B: 92, A: 255}
	escarpmentColor        = color.RGBA{R: 220, G: 142, B: 88, A: 255}
	queuedMigrationColor   = color.RGBA{R: 232, G: 72, B: 72, A: 255}
)

type MapScene struct {
	faceSource      *text.GoTextFaceSource
	biomeGlyphs     [gameapi.BiomeCount]glyphPainter
	glyphDraws      uint64
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
	viewport        Viewport
	hover           TileHover
	camera          Camera
	visibleHeight   float64
	guideHighlight  bool
	chromeRevision  uint64
	haloDistance    []uint8
	haloWater       [haloRingCount]haloBlendPair
	haloActive      bool
	haloRevision    uint64
	haloAridity     float64
	haloCached      bool
	shimmerTick     int
	// reducedMotion freezes the halo at its ring target instead of animating
	// the shimmer. SetReducedMotion below is the setter; drawHalo reads it.
	reducedMotion bool
	// Paints counts every Draw call that actually painted the screen (i.e.
	// returned true). It exists for pkg/app's tests: unlike pkg/render's own
	// package, pkg/app has no TestMain running inside an ebiten game loop, so
	// (*ebiten.Image).At — the screen-sentinel technique this package's own
	// skip test uses — panics there. Exported so it stays outside
	// golangci-lint's unused check; production code never reads it.
	Paints int
}

type mapFrameKey struct {
	frame          *gameapi.Frame
	selectedBand   gameapi.BandID
	preview        MigrationPreview
	hover          TileHover
	notice         string
	ending         EndScene
	resizeRequired bool
	windowWidth    float64
	windowHeight   float64
	camera         Camera
	visibleHeight  float64
	guideHighlight bool
	chromeRevision uint64
	shimmerStep    int
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

// FieldNote is UI-local presentation content. It is never simulation or save
// state; the renderer only lays out the already-selected entry.
type FieldNote struct {
	Topic        string
	Introduction string
	Context      string
	GameEffect   string
	Hint         string
	References   string
	Instructions [4]string
	Celebration  bool
	// Trait and HasTrait identify the heritable variant this note is about,
	// so pkg/hud's details grid can highlight the matching cell. Only
	// ui.TraitFieldNote sets HasTrait true; every other constructor leaves
	// it false and clears the highlight by construction.
	Trait    gameapi.HeritableTrait
	HasTrait bool
}

func NewMapScene() *MapScene {
	source, err := text.NewGoTextFaceSource(bytes.NewReader(goregular.TTF))
	if err != nil {
		panic(err)
	}
	painters, err := newBiomeGlyphs()
	if err != nil {
		panic(err)
	}
	return &MapScene{faceSource: source, biomeGlyphs: painters}
}

// Update advances the fog halo's shimmer clock. It is the scene's only
// per-tick state; everything else the map draws comes from the accepted frame
// or from an explicit setter.
func (scene *MapScene) Update() { scene.shimmerTick++ }

// shimmerPhase is the current phase step, or zero when nothing would move:
// a frame with no fringe has no halo to animate, and holding the step at zero
// there keeps Draw on its idle path instead of repainting fifteen times a
// second for nothing.
func (scene *MapScene) shimmerPhase() int {
	if !scene.haloActive || scene.reducedMotion {
		return 0
	}
	return scene.shimmerTick / shimmerTickStride
}

func (scene *MapScene) SetTileHover(hover TileHover) { scene.hover = hover }

// SetCamera records the presentation camera and the map area's visible height
// (in DIPs, above the drawer) for the next Draw.
func (scene *MapScene) SetCamera(camera Camera, visibleHeight float64) {
	scene.camera = camera
	scene.visibleHeight = visibleHeight
}

// SetGuideHighlight toggles the dashed rectangle drawn around the selected
// band's migration candidates.
func (scene *MapScene) SetGuideHighlight(on bool) { scene.guideHighlight = on }

// SetReducedMotion freezes the fog halo's shimmer at each ring's target,
// keeping the terrain hint and dropping the animation. It also returns the
// scene to the idle-paint path, because a frozen halo has nothing to advance.
func (scene *MapScene) SetReducedMotion(on bool) { scene.reducedMotion = on }

// ReducedMotion reports the current setting. It exists for pkg/app's tests,
// which cannot observe this scene the way this package's own tests do: pkg/app
// has no TestMain running inside an ebiten game loop, so reading rendered
// pixels panics there. Exported so it stays outside golangci-lint's unused
// check; production code never reads it.
func (scene *MapScene) ReducedMotion() bool { return scene.reducedMotion }

// SetChromeRevision records an opaque revision of pkg/hud's chrome for the
// next Draw. pkg/render must not import pkg/hud, so the caller (pkg/app)
// hashes whatever it knows changes the chrome's appearance into this
// uint64. Including it in mapFrameKey is what makes Draw repaint when only
// the chrome changed (details collapsing, the drawer shrinking, a settings
// window closing) even though nothing about the map itself did — those
// changes vacate pixels that only this scene's frame image can restore,
// since pkg/hud draws over it and production leaves an unpainted screen
// exactly as ebiten last left it.
func (scene *MapScene) SetChromeRevision(revision uint64) { scene.chromeRevision = revision }

// SetViewport supplies the measured logical window size for the resize overlay.
func (scene *MapScene) SetViewport(viewport Viewport) { scene.viewport = viewport }

// effectiveVisibleHeight defaults an unset visible height to the full map
// area, so a scene that never called SetCamera behaves as it always has.
func (scene *MapScene) effectiveVisibleHeight() float64 {
	if scene.visibleHeight <= 0 {
		return mapAreaHeight
	}
	return scene.visibleHeight
}

// geometry resolves this scene's camera against one frame's tiles.
func (scene *MapScene) geometry(frame *gameapi.Frame) MapGeometry {
	return CameraGeometry(scene.camera, frame, scene.effectiveVisibleHeight())
}

// Draw renders the map, its overlays, and the terminal scene, returning
// whether it painted the screen this call. Every piece of interactive
// chrome now belongs to pkg/hud, which draws over this image — so the
// screen must be repainted whenever the map's own key changed OR the
// chrome changed (SetChromeRevision, folded into frameKey), and skipped
// only when neither did. Production disables Ebitengine's automatic screen
// clear (SetScreenClearedEveryFrame(false)) precisely so that skip is safe:
// an idle frame does nothing, which is the performance floor (DESIGN.md
// §8). Repainting on a chrome-only change matters because pkg/hud draws
// over this image; when chrome shrinks or closes (details collapsing, the
// drawer compacting, a settings window closing) the vacated region needs
// this frame's pixels blitted back over it, and nothing else will.
func (scene *MapScene) Draw(screen *ebiten.Image, frame *gameapi.Frame, selectedBand gameapi.BandID, preview MigrationPreview, notice string, ending EndScene, resizeRequired bool) bool {
	if frame == nil {
		screen.Fill(color.RGBA{R: 15, G: 22, B: 29, A: 255})
		scene.Paints++
		return true
	}
	scene.refreshHaloCache(frame)
	key := mapFrameKey{
		frame: frame, selectedBand: selectedBand, preview: preview, hover: scene.hover, notice: notice,
		ending: ending, resizeRequired: resizeRequired,
		windowWidth: scene.viewport.LogicalWidthDIP, windowHeight: scene.viewport.LogicalHeightDIP,
		camera: scene.camera, visibleHeight: scene.visibleHeight, guideHighlight: scene.guideHighlight,
		chromeRevision: scene.chromeRevision, shimmerStep: scene.shimmerPhase(),
	}
	width, height := screen.Bounds().Dx(), screen.Bounds().Dy()
	if scene.frameCached && scene.frameKey == key && scene.frameWidth == width && scene.frameHeight == height {
		return false
	}
	if scene.frameImage != nil {
		scene.frameImage.Deallocate()
	}
	transform := FitPresentation(width, height)
	contentWidth := max(1, int(math.Ceil(PresentationWidth*transform.Scale)))
	contentHeight := max(1, int(math.Ceil(PresentationHeight*transform.Scale)))
	scene.frameImage = ebiten.NewImage(contentWidth, contentHeight)
	canvas := newLogicalCanvas(scene.frameImage, transform.Scale)
	scene.drawFrame(canvas, frame, selectedBand, preview, notice, ending)
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
	scene.Paints++
	return true
}

func (scene *MapScene) drawResizeOverlay(screen logicalCanvas) {
	vector.FillRect(screen, 0, 0, PresentationWidth, PresentationHeight, color.RGBA{R: 6, G: 11, B: 15, A: 238}, false)
	drawCentered := func(value string, y, size float32, textColor color.Color) {
		width, _ := text.Measure(value, &text.GoTextFace{Source: scene.faceSource, Size: float64(size)}, float64(size)*textLineSpacing)
		scene.drawText(screen, value, float32((PresentationWidth-width)/2), y, size, textColor)
	}
	drawCentered("Window too small", 220, 84, color.RGBA{R: 239, G: 220, B: 178, A: 255})
	drawCentered("Resize to at least 1280 × 720 to continue", 350, 45, color.White)
	size := fmt.Sprintf("Current window: %g × %g", scene.viewport.LogicalWidthDIP, scene.viewport.LogicalHeightDIP)
	drawCentered(size, 425, 45, color.White)
}

func (scene *MapScene) drawFrame(screen logicalCanvas, frame *gameapi.Frame, selectedBand gameapi.BandID, preview MigrationPreview, notice string, ending EndScene) {
	screen.image.Fill(color.RGBA{R: 15, G: 22, B: 29, A: 255})
	grade := EpochGrade(frame.Climate.AridityIndex)
	scene.drawTimeline(screen, frame, grade)
	scene.drawMapLegend(screen, frame.Climate.AridityIndex)

	geometry := scene.geometry(frame)
	visibleHeight := float64(geometry.visibleHeight)
	s := float64(screen.scale)
	clip := image.Rect(round(mapOriginX*s), round(mapOriginY*s), round((mapOriginX+mapAreaWidth)*s), round((mapOriginY+visibleHeight)*s)).Intersect(screen.image.Bounds())
	mapCanvas := logicalCanvas{image: screen.image.SubImage(clip).(*ebiten.Image), scale: screen.scale}

	markerScale := geometry.Cell / mapTileSize
	// This reset only fires on a recompute: Draw returns early on a
	// frame-cache hit, before drawFrame (and this line) ever runs. So a
	// cache-hit Draw leaves glyphDraws holding the prior composition's
	// count. That is fine -- a cache hit means nothing was redrawn, so the
	// stale count still describes exactly what is on screen.
	scene.glyphDraws = 0
	scene.drawTerrain(mapCanvas, geometry, frame)
	scene.drawHalo(mapCanvas, geometry, frame)
	// Glyphs must land here: above terrain/halo (so they're visible) but
	// below the reachable-tile overlay, guide highlight, markers, and band
	// discs drawn next (so those selection affordances stay readable over a
	// biome's pictograph rather than getting obscured by it).
	scene.drawBiomeGlyphs(mapCanvas, geometry, frame)
	scene.drawLakes(mapCanvas, geometry, frame)
	scene.drawReachableTiles(mapCanvas, geometry, frame, selectedBand)
	scene.drawGuideHighlight(mapCanvas, geometry, frame, selectedBand)
	// The pointer's tile is tinted on the map itself now that the bottom
	// inspector is gone; the panel reads the same hover for its detail lines.
	if scene.hover.Visible && int(scene.hover.TileID) < len(frame.Tiles) && frame.Tiles[scene.hover.TileID].Explored {
		x, y := geometry.TilePoint(frame.Tiles[scene.hover.TileID])
		vector.FillRect(mapCanvas, x-geometry.Cell/2+0.7, y-geometry.Cell/2+0.7, geometry.Cell-1.8, geometry.Cell-1.8, color.RGBA{R: 87, G: 211, B: 211, A: 70}, false)
	}
	scene.drawEscarpments(mapCanvas, geometry, frame)
	for _, passage := range frame.Passages {
		lineColor, visible := passageColorForRender(frame, passage)
		if !visible {
			continue
		}
		switch kind, anchor := passageOverlayForRender(frame, passage); kind {
		case passageOverlayLine:
			from, to := frame.Tiles[passage.From], frame.Tiles[passage.To]
			fromX, fromY := geometry.TilePoint(from)
			toX, toY := geometry.TilePoint(to)
			vector.StrokeLine(mapCanvas, fromX, fromY, toX, toY, 2, lineColor, false)
		case passageOverlayGlyph:
			// A lone explored shore marks that a crossing starts here without
			// drawing a line into fog toward the hidden far endpoint.
			x, y := geometry.TilePoint(frame.Tiles[anchor])
			drawPassageGlyph(mapCanvas, x, y, markerScale, lineColor)
		}
	}
	var interbreedTiles map[gameapi.TileID]bool
	if actor := selectedBandInFrame(frame, selectedBand); actor != nil {
		interbreedTiles = interbreedCandidateTiles(frame, *actor)
	}
	// Markers belong to tiles rather than to bands: co-located bands share one
	// centre, so a circle per band painted the same disc repeatedly and only
	// the last one drawn survived. Rings follow the disc for the same reason —
	// stroking them per band stacked identical circles on identical pixels.
	var selectedTile gameapi.TileID
	hasSelectedTile := false
	if actor := selectedBandInFrame(frame, selectedBand); actor != nil {
		selectedTile, hasSelectedTile = actor.TileID, true
	}
	for _, marker := range tileMarkersForRender(frame) {
		centreX, centreY := geometry.TilePoint(frame.Tiles[marker.TileID])
		drawTileMarker(mapCanvas, centreX, centreY, markerScale, marker)
		// Interbreeding requires co-location, so a candidate always stands on
		// the ringed tile: the option is visible on the map rather than only
		// discovered by pressing the key and hoping.
		if interbreedTiles[marker.TileID] {
			vector.StrokeCircle(mapCanvas, centreX, centreY, 6.4*markerScale, 1.5, interbreedMarkerColor, true)
		}
		if hasSelectedTile && marker.TileID == selectedTile {
			vector.StrokeCircle(mapCanvas, centreX, centreY, 5.2*markerScale, 1.5, color.White, true)
		}
	}
	scene.drawQueuedMigrations(mapCanvas, geometry, frame)
	scene.drawMigrationPreview(mapCanvas, geometry, frame, preview)
	scene.drawEndScene(screen, ending)
	if notice != "" {
		// Long diagnostics wrap and grow the box downward over the map rather
		// than running past its right edge.
		lines := scene.wrapTextToWidth(notice, noticeFontSize, noticeTextMaxWidth)
		vector.FillRect(screen, noticeBoxX, noticeBoxY, noticeBoxWidth, noticeBoxHeight(len(lines)), color.RGBA{R: 26, G: 38, B: 45, A: 240}, false)
		scene.drawText(screen, strings.Join(lines, "\n"), noticeTextX, noticeTextY, noticeFontSize, color.RGBA{R: 239, G: 220, B: 178, A: 255})
	}
}

// round rounds to the nearest physical pixel for SubImage clip rectangles.
func round(value float64) int {
	return int(math.Round(value))
}

// drawGuideHighlight strokes a dashed rectangle around the selected band's
// migration candidates. Task 16 is the first caller to turn it on.
func (scene *MapScene) drawGuideHighlight(screen logicalCanvas, geometry MapGeometry, frame *gameapi.Frame, selectedBand gameapi.BandID) {
	if !scene.guideHighlight {
		return
	}
	band := selectedBandInFrame(frame, selectedBand)
	if band == nil || len(band.MigrationCandidates) == 0 {
		return
	}
	var minX, minY, maxX, maxY float32
	found := false
	for _, candidate := range band.MigrationCandidates {
		if int(candidate.TileID) >= len(frame.Tiles) {
			continue
		}
		x, y := geometry.TilePoint(frame.Tiles[candidate.TileID])
		if !found {
			minX, maxX, minY, maxY = x, x, y, y
			found = true
			continue
		}
		minX, maxX = min(minX, x), max(maxX, x)
		minY, maxY = min(minY, y), max(maxY, y)
	}
	if !found {
		return
	}
	margin := geometry.Cell
	left, top := minX-margin, minY-margin
	right, bottom := maxX+margin, maxY+margin
	drawDashedRect(screen, left, top, right-left, bottom-top, color.RGBA{R: 245, G: 202, B: 92, A: 220})
}

// drawDashedRect strokes a rectangle's outline as alternating 6px-on/4px-off
// segments so a guide highlight reads as an overlay rather than solid chrome.
func drawDashedRect(screen logicalCanvas, x, y, width, height float32, dashColor color.Color) {
	corners := [][4]float32{
		{x, y, x + width, y},
		{x + width, y, x + width, y + height},
		{x + width, y + height, x, y + height},
		{x, y + height, x, y},
	}
	for _, edge := range corners {
		drawDashedLine(screen, edge[0], edge[1], edge[2], edge[3], dashColor)
	}
}

func drawDashedLine(screen logicalCanvas, fromX, fromY, toX, toY float32, dashColor color.Color) {
	const dashOn, dashOff = float32(6), float32(4)
	dx, dy := toX-fromX, toY-fromY
	length := float32(math.Hypot(float64(dx), float64(dy)))
	if length <= 0 {
		return
	}
	unitX, unitY := dx/length, dy/length
	for travelled := float32(0); travelled < length; travelled += dashOn + dashOff {
		segmentEnd := min(travelled+dashOn, length)
		startX, startY := fromX+unitX*travelled, fromY+unitY*travelled
		endX, endY := fromX+unitX*segmentEnd, fromY+unitY*segmentEnd
		vector.StrokeLine(screen, startX, startY, endX, endY, 1.2, dashColor, false)
	}
}

func (scene *MapScene) drawEscarpments(screen logicalCanvas, geometry MapGeometry, frame *gameapi.Frame) {
	for _, edge := range frame.Escarpments {
		if int(edge.First) >= len(frame.Tiles) || int(edge.Second) >= len(frame.Tiles) {
			continue
		}
		first, second := frame.Tiles[edge.First], frame.Tiles[edge.Second]
		if !first.Explored || !second.Explored {
			continue
		}
		fromX, fromY, toX, toY, ok := escarpmentLine(geometry, first, second)
		if !ok {
			continue
		}
		vector.StrokeLine(screen, fromX, fromY, toX, toY, 3, color.RGBA{R: 48, G: 31, B: 26, A: 235}, false)
		vector.StrokeLine(screen, fromX, fromY, toX, toY, 1.35, escarpmentColor, false)
	}
}

func escarpmentLine(geometry MapGeometry, first, second gameapi.Tile) (float32, float32, float32, float32, bool) {
	dx, dy := second.X-first.X, second.Y-first.Y
	if absRenderInt(dx)+absRenderInt(dy) != 1 {
		return 0, 0, 0, 0, false
	}
	left := geometry.OriginX + float32(min(first.X, second.X))*geometry.Cell
	top := geometry.OriginY + float32(min(first.Y, second.Y))*geometry.Cell
	if dx != 0 {
		x := left + geometry.Cell
		return x, top, x, top + geometry.Cell, true
	}
	y := top + geometry.Cell
	return left, y, left + geometry.Cell, y, true
}

func absRenderInt(value int) int {
	if value < 0 {
		return -value
	}
	return value
}

// refreshHaloCache recomputes the fringe distance field and the water blend
// pairs when exploration or the climate grade changes. It runs at the top of
// Draw rather than inside drawTerrain because shimmerPhase reads haloActive
// while the frame key is built before any drawing happens: leaving it in the
// terrain cache let the key's shimmer step disagree with the one drawHalo
// rendered with on the tick the halo first became active. Unlike the terrain
// image this cache is scale-free, so a resize no longer rebuilds it.
func (scene *MapScene) refreshHaloCache(frame *gameapi.Frame) {
	if scene.haloCached && scene.haloRevision == frame.TerrainRevision && scene.haloAridity == frame.Climate.AridityIndex {
		return
	}
	scene.haloDistance = haloDistances(frame)
	scene.haloWater = haloWaterBlend(EpochGrade(frame.Climate.AridityIndex).Water)
	scene.haloActive = false
	for _, distance := range scene.haloDistance {
		if distance >= 1 && distance <= haloRingCount {
			scene.haloActive = true
			break
		}
	}
	scene.haloRevision = frame.TerrainRevision
	scene.haloAridity = frame.Climate.AridityIndex
	scene.haloCached = true
}

// drawTerrain caches the immutable top-down tile layer until either its coarse
// terrain revision or its continuously graded water color changes. Commands
// that reveal terrain (including a successful split) advance that revision;
// other planning-only frames can reuse it without stale exploration, biome,
// macro-impact, or climate colors.
func (scene *MapScene) drawTerrain(screen logicalCanvas, geometry MapGeometry, frame *gameapi.Frame) {
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
	// The cache stays a fixed 8 px-per-tile image keyed only by revision,
	// aridity, and physical scale; the camera's zoom is applied here, at draw
	// time, by scaling and translating it into place.
	cellScale := float64(geometry.Cell) / float64(mapTileSize)
	options := &ebiten.DrawImageOptions{}
	options.GeoM.Scale(cellScale, cellScale)
	options.GeoM.Translate(float64(geometry.OriginX)*float64(screen.scale), float64(geometry.OriginY)*float64(screen.scale))
	options.Filter = ebiten.FilterNearest
	screen.image.DrawImage(scene.terrainImage, options)
}

// drawHalo tints unexplored tiles within haloRingCount of the explored set.
// It runs immediately after the terrain blit so the selection overlays drawn
// below it -- reachable highlights, the guide rectangle, a migration preview --
// land on top rather than under. None of those three tests Explored itself,
// but none can reach a halo tile either: the projection drops every candidate
// whose destination is unexplored, so a band's candidate list cannot name one.
// Markers, rings, escarpments, passage lines and queued migrations cannot
// either, each being gated on the real Explored bit here.
// Unlike the terrain layer the halo is not baked into the cached image -- that
// image is keyed on exploration and would have to redraw all 6,144 cells on
// every shimmer step.
func (scene *MapScene) drawHalo(screen logicalCanvas, geometry MapGeometry, frame *gameapi.Frame) {
	if !scene.haloActive || len(scene.haloDistance) != len(frame.Tiles) {
		return
	}
	step := scene.shimmerPhase()
	extent := geometry.Cell * (mapTileSize - 0.4) / mapTileSize
	for id := range frame.Tiles {
		ring := int(scene.haloDistance[id]) - 1
		if ring < 0 || ring >= haloRingCount {
			continue
		}
		tile := &frame.Tiles[id]
		noise := 1.0
		if !scene.reducedMotion {
			noise = shimmerNoise(tile.X, tile.Y, step)
		}
		base, blend := climateBiomeColor(tile.Biome, frame.Climate.AridityIndex), haloLandBlend[tile.Biome][ring]
		if !tile.Land {
			base, blend = EpochGrade(frame.Climate.AridityIndex).Water, scene.haloWater[ring]
		}
		x, y := geometry.TilePoint(*tile)
		vector.FillRect(screen, x-geometry.Cell/2, y-geometry.Cell/2, extent, extent, haloColor(base, blend, noise), false)
	}
}

// drawBiomeGlyphs paints one pictograph per explored land tile, giving biome
// identity a shape channel alongside the L* fill ladder. It is deliberately not
// baked into the terrain cache: that image is a fixed 8 px-per-tile raster the
// camera scales up with FilterNearest, so a baked glyph would be upscaled 3x
// into mush at exactly the zoom where it is meant to be legible.
func (scene *MapScene) drawBiomeGlyphs(screen logicalCanvas, geometry MapGeometry, frame *gameapi.Frame) {
	alpha := glyphAlpha(geometry.Cell)
	if alpha <= 0 {
		return
	}
	size := geometry.Cell * glyphCellFraction
	// The ink is a pure function of biome, so hoist it out of the tile loop.
	// glyphInk costs six math.Pow calls through relativeLuminance; paying that
	// per tile priced the whole grid for six distinct answers.
	var ink [gameapi.BiomeCount]color.RGBA
	for biome := gameapi.Biome(0); biome < gameapi.BiomeCount; biome++ {
		ink[biome] = glyphInk(climateBiomeColor(biome, frame.Climate.AridityIndex))
	}
	for _, tile := range frame.Tiles {
		if !tile.Explored || !tile.Land || int(tile.Biome) >= len(scene.biomeGlyphs) {
			continue
		}
		x, y := geometry.TilePoint(tile)
		// The SubImage clip makes an off-screen glyph harmless, but not free:
		// text.Draw still shapes, rasterizes and submits it. At focus the grid
		// is 96x64 while the map rectangle shows about 36x27 cells, so relying
		// on the clip alone paints roughly six tiles for every one visible.
		// Skip them here instead. One em of slack on each edge is far more
		// than the widest glyph's half-ink (9.93 DIP against a 16.8 DIP em),
		// so nothing partly on screen is dropped.
		if x < mapOriginX-size || x > mapOriginX+mapAreaWidth+size ||
			y < mapOriginY-size || y > geometry.visibleBottom()+size {
			continue
		}
		scene.biomeGlyphs[tile.Biome].paint(screen, x, y, size, ink[tile.Biome], alpha)
		scene.glyphDraws++
	}
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

func (scene *MapScene) drawMigrationPreview(screen logicalCanvas, geometry MapGeometry, frame *gameapi.Frame, preview MigrationPreview) {
	if !preview.Visible || int(preview.TileID) >= len(frame.Tiles) {
		return
	}
	band := selectedBandInFrame(frame, preview.BandID)
	if band == nil || int(band.TileID) >= len(frame.Tiles) {
		return
	}
	origin, destination := frame.Tiles[band.TileID], frame.Tiles[preview.TileID]
	fromX, fromY := geometry.TilePoint(origin)
	toX, toY := geometry.TilePoint(destination)
	drawMigrationArrow(screen, fromX, fromY, toX, toY, color.RGBA{R: 255, G: 74, B: 74, A: 255})
	vector.StrokeCircle(screen, toX, toY, 4.2*geometry.Cell/mapTileSize, 1.2, color.RGBA{R: 255, G: 126, B: 106, A: 255}, true)
}

func (scene *MapScene) drawQueuedMigrations(screen logicalCanvas, geometry MapGeometry, frame *gameapi.Frame) {
	for _, band := range frame.Bands {
		if band.Species != gameapi.HomoSapiens || !band.HasQueuedMigration || int(band.TileID) >= len(frame.Tiles) || int(band.QueuedMigration) >= len(frame.Tiles) {
			continue
		}
		origin, destination := frame.Tiles[band.TileID], frame.Tiles[band.QueuedMigration]
		if !origin.Explored || !destination.Explored {
			continue
		}
		fromX, fromY := geometry.TilePoint(origin)
		toX, toY := geometry.TilePoint(destination)
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

func (scene *MapScene) drawReachableTiles(screen logicalCanvas, geometry MapGeometry, frame *gameapi.Frame, selectedBand gameapi.BandID) {
	band := selectedBandInFrame(frame, selectedBand)
	if band == nil || band.SpatialActionUsed {
		return
	}
	for index, candidate := range band.MigrationCandidates {
		if int(candidate.TileID) >= len(frame.Tiles) {
			continue
		}
		tile := frame.Tiles[candidate.TileID]
		x, y := geometry.TilePoint(tile)
		highlight := reachableTileColor(index)
		x -= geometry.Cell / 2
		y -= geometry.Cell / 2
		vector.FillRect(screen, x+0.7, y+0.7, geometry.Cell-1.8, geometry.Cell-1.8, color.RGBA{R: highlight.R, G: highlight.G, B: highlight.B, A: 48}, false)
		vector.StrokeRect(screen, x+0.7, y+0.7, geometry.Cell-1.8, geometry.Cell-1.8, 1.35, highlight, false)
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

// drawTileMarker paints a tile's disc: one species fills it outright, and two
// or more split it into wedges. The undivided case stays on FillCircle because
// a whole-turn sweep is not a sector — see scaledVector.FillPie.
func drawTileMarker(canvas logicalCanvas, centreX, centreY, markerScale float32, marker tileMarker) {
	radius := 3.6 * markerScale
	wedges, count := markerWedges(marker)
	if count == 1 {
		vector.FillCircle(canvas, centreX, centreY, radius, wedges[0].Color, true)
		return
	}
	for _, wedge := range wedges[:count] {
		vector.FillPie(canvas, centreX, centreY, radius, wedge.StartAngle, wedge.SweepAngle, wedge.Color, true)
	}
}

// markerWedge is one species' slice of a tile's disc.
type markerWedge struct {
	Color      color.RGBA
	StartAngle float32
	SweepAngle float32
}

// markerWedges splits a tile's disc into one wedge per species standing on it,
// running clockwise from twelve o'clock in species order so a tile keeps its
// slice arrangement as bands are founded and die. Angles accumulate in float64
// and narrow once, because summing float32 sweeps drifts a wedge off the
// previous one's edge. A tile holding a single species yields one whole-turn
// wedge, which drawTileMarker draws as a plain circle rather than an arc.
func markerWedges(marker tileMarker) ([gameapi.SpeciesCount]markerWedge, int) {
	var wedges [gameapi.SpeciesCount]markerWedge
	total := 0
	for _, bands := range marker.Counts {
		total += bands
	}
	if total == 0 {
		return wedges, 0
	}
	start, count := -math.Pi/2, 0
	for species, bands := range marker.Counts {
		if bands == 0 {
			continue
		}
		sweep := 2 * math.Pi * float64(bands) / float64(total)
		wedges[count] = markerWedge{
			Color:      speciesMarkerColor(gameapi.Species(species)),
			StartAngle: float32(start),
			SweepAngle: float32(sweep),
		}
		start += sweep
		count++
	}
	return wedges, count
}

func speciesMarkerColor(species gameapi.Species) color.RGBA {
	if species == gameapi.ArchaicHominin {
		return archaicBandMarkerColor
	}
	return sapiensBandMarkerColor
}

// tileMarker is the consolidated species breakdown of every visible band
// standing on one tile.
type tileMarker struct {
	TileID gameapi.TileID
	Counts [gameapi.SpeciesCount]int
}

// tileMarkersForRender groups every visible band onto the tile it stands on,
// in first-appearance order so discs keep their identity between frames. It
// buckets species by the same predicate bandColorForRender uses, so a marker's
// slices and its colours can never disagree.
func tileMarkersForRender(frame *gameapi.Frame) []tileMarker {
	if frame == nil {
		return nil
	}
	var markers []tileMarker
	positions := make(map[gameapi.TileID]int, len(frame.Bands))
	for _, band := range frame.Bands {
		if _, visible := bandColorForRender(frame, band); !visible {
			continue
		}
		position, seen := positions[band.TileID]
		if !seen {
			position = len(markers)
			positions[band.TileID] = position
			markers = append(markers, tileMarker{TileID: band.TileID})
		}
		species := gameapi.HomoSapiens
		if band.Species == gameapi.ArchaicHominin {
			species = gameapi.ArchaicHominin
		}
		markers[position].Counts[species]++
	}
	return markers
}

func bandColorForRender(frame *gameapi.Frame, band gameapi.Band) (color.RGBA, bool) {
	if frame == nil || int(band.TileID) >= len(frame.Tiles) {
		return color.RGBA{}, false
	}
	if !frame.Tiles[band.TileID].Explored && band.Species != gameapi.HomoSapiens {
		return color.RGBA{}, false
	}
	return speciesMarkerColor(band.Species), true
}

func reachableTileColor(candidateIndex int) color.RGBA {
	if candidateIndex == 0 {
		return color.RGBA{R: 245, G: 202, B: 92, A: 255}
	}
	return color.RGBA{R: 87, G: 211, B: 211, A: 255}
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
	entries := scene.mapLegendEntries(aridity)
	const entryWidth = float32(96)
	for index, entry := range entries {
		x := float32(mapOriginX) + float32(index)*entryWidth
		if entry.edge {
			// Spelled with the swatch constants rather than the literals the
			// 8x8 era left behind: the rule now spans the same box every
			// other entry's swatch fills, and its centre line follows
			// legendSwatchY instead of coincidentally matching it.
			ruleY := mapLegendOriginY + legendSwatchY + legendSwatchSize/2
			vector.StrokeLine(screen, x+legendSwatchX, ruleY, x+legendSwatchX+legendSwatchSize, ruleY, 3, color.RGBA{R: 48, G: 31, B: 26, A: 235}, false)
			vector.StrokeLine(screen, x+legendSwatchX, ruleY, x+legendSwatchX+legendSwatchSize, ruleY, 1.35, entry.color, false)
		} else {
			vector.FillRect(screen, x+legendSwatchX, mapLegendOriginY+legendSwatchY, legendSwatchSize, legendSwatchSize, entry.color, false)
			vector.StrokeRect(screen, x+legendSwatchX, mapLegendOriginY+legendSwatchY, legendSwatchSize, legendSwatchSize, 0.7, color.RGBA{R: 210, G: 216, B: 210, A: 180}, false)
			if entry.glyph != nil {
				centre := x + legendSwatchX + legendSwatchSize/2
				entry.glyph.paint(screen, centre, mapLegendOriginY+legendSwatchY+legendSwatchSize/2, legendGlyphSize, glyphInk(entry.color), 1)
			}
		}
		scene.drawText(screen, entry.label, x+legendLabelX, mapLegendOriginY+1, 8.5, color.RGBA{R: 235, G: 236, B: 226, A: 255})
		scene.drawText(screen, entry.meaning, x+legendSwatchX, mapLegendOriginY+legendMeaningY, legendMeaningSize, color.RGBA{R: 167, G: 184, B: 181, A: 255})
	}
}

// wrapTextToWidth wraps at measured pixel widths for the given font size, so
// proportional glyphs cannot push a line past its box the way a rune count can.
func (scene *MapScene) wrapTextToWidth(value string, size, maxWidth float32) []string {
	face := &text.GoTextFace{Source: scene.faceSource, Size: float64(size)}
	return wrapWords(value, func(line string) bool {
		width, _ := text.Measure(line, face, 0)
		return float32(width) <= maxWidth
	})
}

// wrapWords greedily packs each paragraph's words into lines that satisfy fits.
// A single word that never fits stands on its own line rather than being split.
func wrapWords(value string, fits func(string) bool) []string {
	lines := make([]string, 0)
	for _, paragraph := range strings.Split(value, "\n") {
		words := strings.Fields(paragraph)
		if len(words) == 0 {
			lines = append(lines, "")
			continue
		}
		line := words[0]
		for _, word := range words[1:] {
			if candidate := line + " " + word; fits(candidate) {
				line = candidate
				continue
			}
			lines = append(lines, line)
			line = word
		}
		lines = append(lines, line)
	}
	return lines
}

// noticeBoxHeight fits one line in the original 30 px bar and adds the
// drawText line spacing for each further wrapped line.
func noticeBoxHeight(lines int) float32 {
	return noticeBoxMinHeight + float32(max(lines, 1)-1)*noticeFontSize*textLineSpacing
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
	options.LineSpacing = float64(size) * textLineSpacing * scale
	text.Draw(destination.image, value, &text.GoTextFace{Source: scene.faceSource, Size: float64(size) * scale}, options)
}

func climateBiomeColor(biome gameapi.Biome, _ float64) color.RGBA {
	// Terrain tiles are 7.6px, where chroma discrimination is weak and lightness
	// is not, so these are spaced as an L* ladder -- riverine 29, highlands 39,
	// shrubland 51, savanna 60, desert 72, tundra 80 -- rather than by hue alone.
	// That keeps the desert margin and tundra line legible at tile size and for
	// red-green color blindness, where hue-only separation collapses.
	return [gameapi.BiomeCount]color.RGBA{
		{R: 54, G: 75, B: 41, A: 255}, {R: 154, G: 145, B: 69, A: 255}, {R: 97, G: 129, B: 110, A: 255},
		{R: 96, G: 91, B: 88, A: 255}, {R: 204, G: 170, B: 125, A: 255}, {R: 182, G: 201, B: 213, A: 255},
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

// passageOverlayKind is how much of a named passage the map may draw.
type passageOverlayKind uint8

const (
	passageOverlayHidden passageOverlayKind = iota
	passageOverlayGlyph
	passageOverlayLine
)

// passageOverlayForRender reads the endpoint tiles' own exploration bits rather
// than Passage.Explored, which the projection sets when either endpoint is
// known and so cannot separate one reached shore from both. With exactly one
// endpoint explored it returns that tile as the glyph anchor.
func passageOverlayForRender(frame *gameapi.Frame, passage gameapi.Passage) (passageOverlayKind, gameapi.TileID) {
	if frame == nil || int(passage.From) >= len(frame.Tiles) || int(passage.To) >= len(frame.Tiles) {
		return passageOverlayHidden, 0
	}
	fromExplored, toExplored := frame.Tiles[passage.From].Explored, frame.Tiles[passage.To].Explored
	switch {
	case fromExplored && toExplored:
		return passageOverlayLine, passage.From
	case fromExplored:
		return passageOverlayGlyph, passage.From
	case toExplored:
		return passageOverlayGlyph, passage.To
	default:
		return passageOverlayHidden, 0
	}
}

// drawPassageGlyph strokes a small diamond at a passage endpoint. It is
// symmetric so it hints at nothing about the far shore's direction, and its
// shape keeps it apart from the round band markers and selection rings.
func drawPassageGlyph(screen logicalCanvas, x, y, markerScale float32, tint color.RGBA) {
	half := 4.2 * markerScale
	vector.StrokeLine(screen, x, y-half, x+half, y, 1.5, tint, true)
	vector.StrokeLine(screen, x+half, y, x, y+half, 1.5, tint, true)
	vector.StrokeLine(screen, x, y+half, x-half, y, 1.5, tint, true)
	vector.StrokeLine(screen, x-half, y, x, y-half, 1.5, tint, true)
}
