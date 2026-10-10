package render

import (
	"image"
	"image/color"
	"math"

	"github.com/adsouza/africa2ice/pkg/gameapi"
	"github.com/hajimehoshi/ebiten/v2"
)

const (
	departureWidth  = 720
	departureHeight = 176
	departureTicks  = 4 * 60
)

// departureScene holds a short illustration of setting out, independent of
// the campaign clock. Accepted origin facts survive later planning commands.
type departureScene struct {
	band                     gameapi.Band
	tile                     gameapi.Tile
	turn                     int
	tick                     int
	active                   bool
	westward                 bool
	background, people, mask *ebiten.Image
	backgroundCached         bool
	peoplePhase              int
	peopleCached             bool
	paintKey                 departurePaintKey
	painted                  bool
}

type departurePaintKey struct {
	visible       bool
	phase         int
	reducedMotion bool
	bounds        image.Rectangle
}

// StartDeparture is called only after a migration command succeeds. The band
// and tile describe departure, before any end-turn movement has resolved.
func (scene *MapScene) StartDeparture(band gameapi.Band, tile, destination gameapi.Tile, turn int) {
	scene.departure.band, scene.departure.tile = band, tile
	// Grid columns increase eastward, including the unwrapped Beringian tiles.
	scene.departure.westward = destination.X < tile.X
	scene.departure.turn, scene.departure.tick, scene.departure.active = turn, 0, true
	scene.departure.backgroundCached, scene.departure.peopleCached = false, false
	scene.departure.painted = false
	scene.frameCached = false // refresh the route foreground even if the host reused a frame
}

func (scene *MapScene) ClearDeparture() { scene.departure.active = false }

// DepartureActive reports whether the brief departure presentation is live.
func (scene *MapScene) DepartureActive() bool { return scene.departure.active }

// SetDepartureVisible lets the host suppress decoration beneath HUD windows
// and tooltips. The four-second lifetime continues while it is obscured.
func (scene *MapScene) SetDepartureVisible(visible bool) { scene.departureHidden = !visible }

func (scene *departureScene) update() {
	if scene.active {
		scene.tick++
		if scene.tick >= departureTicks {
			scene.active = false
		}
	}
}

func departureBounds(visibleHeight float64) image.Rectangle {
	left := mapOriginX + int((mapAreaWidth-departureWidth)/2)
	bottom := mapOriginY + int(visibleHeight) - 28 // clear the drawer's tab as well as its body
	return image.Rect(left, bottom-departureHeight, left+departureWidth, bottom)
}

func (scene *departureScene) opacity(reducedMotion bool) float32 {
	if reducedMotion {
		return 1
	}
	// Appear immediately; soften only the final 0.8 seconds.
	return float32(smoothstep(math.Min(1, float64(departureTicks-scene.tick)/48)))
}

func (scene *departureScene) prepare(phase int) {
	if scene.background == nil {
		scene.background = ebiten.NewImage(departureWidth, departureHeight)
		scene.people = ebiten.NewImage(departureWidth, departureHeight)
		scene.mask = ebiten.NewImage(departureWidth, departureHeight)
		pixels := make([]byte, departureWidth*departureHeight*4)
		for y := range departureHeight {
			for x := range departureWidth {
				edgeX := math.Min(float64(x), float64(departureWidth-1-x)) / 42
				edgeY := math.Min(float64(y), float64(departureHeight-1-y)) / 24
				alpha := uint8(255 * smoothstep(math.Min(1, edgeX)) * smoothstep(math.Min(1, edgeY)))
				i := (y*departureWidth + x) * 4
				pixels[i], pixels[i+1], pixels[i+2], pixels[i+3] = alpha, alpha, alpha, alpha
			}
		}
		scene.mask.WritePixels(pixels)
	}
	maskOptions := &ebiten.DrawImageOptions{Blend: ebiten.BlendDestinationIn}
	if !scene.backgroundCached {
		scene.background.Clear()
		drawDepartureLandscape(newLogicalCanvas(scene.background, 1), scene.tile)
		scene.background.DrawImage(scene.mask, maskOptions)
		scene.backgroundCached = true
	}
	if !scene.peopleCached || scene.peoplePhase != phase {
		scene.people.Clear()
		drawDepartureCompany(newLogicalCanvas(scene.people, 1), scene.band, scene.tile, float64(phase)/20)
		scene.people.DrawImage(scene.mask, maskOptions)
		scene.peoplePhase, scene.peopleCached = phase, true
	}
}

// drawDeparture restores the cached map before compositing translucent art.
// On animation-only draws it touches just the vignette's pixels, leaving the
// map cache, HUD, and letterbox intact. Route affordances stay above the art.
func (scene *MapScene) drawDeparture(screen *ebiten.Image, frame *gameapi.Frame, selected gameapi.BandID, full bool) {
	d := &scene.departure
	if d.active {
		band := selectedBandInFrame(frame, selected)
		if band == nil || band.ID != d.band.ID || band.TileID != d.tile.ID || frame.Turn != d.turn || frame.CampaignResult != gameapi.Ongoing {
			d.active = false
		}
	}
	key := departurePaintKey{visible: d.active && !scene.departureHidden, reducedMotion: scene.reducedMotion}
	if key.visible {
		key.bounds = departureBounds(scene.effectiveVisibleHeight())
		if !scene.reducedMotion {
			key.phase = d.tick / 3
		}
	}
	if !full && d.painted && d.paintKey == key {
		return
	}
	transform := FitPresentation(screen.Bounds().Dx(), screen.Bounds().Dy())
	logicalClip := key.bounds
	if !full && d.painted && d.paintKey.visible {
		logicalClip = logicalClip.Union(d.paintKey.bounds)
	}
	clip := image.Rect(
		int(math.Floor(transform.OffsetX+float64(logicalClip.Min.X)*transform.Scale)),
		int(math.Floor(transform.OffsetY+float64(logicalClip.Min.Y)*transform.Scale)),
		int(math.Ceil(transform.OffsetX+float64(logicalClip.Max.X)*transform.Scale)),
		int(math.Ceil(transform.OffsetY+float64(logicalClip.Max.Y)*transform.Scale)),
	).Intersect(screen.Bounds())
	if !clip.Empty() {
		target := screen.SubImage(clip).(*ebiten.Image)
		baseOptions := &ebiten.DrawImageOptions{}
		baseOptions.GeoM.Translate(transform.OffsetX, transform.OffsetY)
		if !full {
			target.DrawImage(scene.frameImage, baseOptions)
		}
		if key.visible {
			d.prepare(key.phase)
			options := &ebiten.DrawImageOptions{Filter: ebiten.FilterLinear}
			options.GeoM.Scale(transform.Scale, transform.Scale)
			options.GeoM.Translate(transform.OffsetX+float64(key.bounds.Min.X)*transform.Scale, transform.OffsetY+float64(key.bounds.Min.Y)*transform.Scale)
			options.ColorScale.ScaleAlpha(.34 * d.opacity(scene.reducedMotion))
			target.DrawImage(d.background, options)
			if d.westward {
				// Mirror walkers and underfoot drift within the fixed strip. The
				// daylight landscape and map route retain their original orientation.
				options.GeoM.Reset()
				options.GeoM.Scale(-transform.Scale, transform.Scale)
				options.GeoM.Translate(transform.OffsetX+float64(key.bounds.Max.X)*transform.Scale, transform.OffsetY+float64(key.bounds.Min.Y)*transform.Scale)
			}
			options.ColorScale.Reset()
			options.ColorScale.ScaleAlpha(.94 * d.opacity(scene.reducedMotion))
			target.DrawImage(d.people, options)
			if scene.departureForeground != nil {
				target.DrawImage(scene.departureForeground, baseOptions)
			}
		}
	}
	d.paintKey, d.painted = key, true
}

func (scene *MapScene) prepareDepartureForeground(frame *gameapi.Frame, selected gameapi.BandID, preview MigrationPreview, notice string) {
	if scene.departureForeground != nil {
		scene.departureForeground.Deallocate()
		scene.departureForeground = nil
	}
	if !scene.departure.active {
		return
	}
	scene.departureForeground = ebiten.NewImage(scene.frameImage.Bounds().Dx(), scene.frameImage.Bounds().Dy())
	canvas := newLogicalCanvas(scene.departureForeground, scene.frameScale)
	geometry := scene.geometry(frame)
	// Preserve the selected origin and every migration arrow over the vignette.
	band := selectedBandInFrame(frame, selected)
	if band != nil && int(band.TileID) < len(frame.Tiles) {
		for _, marker := range tileMarkersForRender(frame) {
			if marker.TileID == band.TileID {
				x, y := geometry.TilePoint(frame.Tiles[marker.TileID])
				drawTileMarker(canvas, x, y, geometry.Cell/mapTileSize, marker)
				vector.StrokeCircle(canvas, x, y, 5.2*geometry.Cell/mapTileSize, 1.5, color.White, true)
			}
		}
	}
	scene.drawQueuedMigrations(canvas, geometry, frame)
	scene.drawMigrationPreview(canvas, geometry, frame, preview)
	scene.drawNotice(canvas, notice)
}
