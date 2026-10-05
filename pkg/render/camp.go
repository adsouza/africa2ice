package render

import (
	"image"
	"image/color"
	"math"

	"github.com/adsouza/africa2ice/pkg/gameapi"
	"github.com/hajimehoshi/ebiten/v2"
	ebitenvector "github.com/hajimehoshi/ebiten/v2/vector"
)

// CampScene is an illustrative view of a band, not a simulation of its
// individual members. Its own tick clock never consumes campaign randomness.
// Static scenery is cached; motion repaints at 20 Hz, and reduced motion
// holds one reproducible pose. Motion repaints only the illustration's moving
// region; the host repaints its HUD when Draw requests a full repaint.
type CampScene struct {
	tick        int
	background  *ebiten.Image
	sky         *ebiten.Image
	image       *ebiten.Image
	key         campKey
	cached      bool
	imageKey    campKey
	imageCached bool
}

type campKey struct {
	frame   *gameapi.Frame
	band    gameapi.BandID
	chrome  uint64
	phase   int
	width   int
	height  int
	twinkle bool
}

type CampPaint uint8

const (
	CampSkipped CampPaint = iota
	CampMotion
	CampFull
)

func (scene *CampScene) Update(reducedMotion bool) {
	if !reducedMotion {
		scene.tick++
	}
}

// Invalidate ensures re-entering a frozen camp paints over the map even
// when its frame, viewport and caption are identical to the previous visit.
func (scene *CampScene) Invalidate() {
	scene.cached, scene.imageCached = false, false
}

func (scene *CampScene) Draw(screen *ebiten.Image, frame *gameapi.Frame, selected gameapi.BandID, reducedMotion bool, chrome uint64) CampPaint {
	width, height := screen.Bounds().Dx(), screen.Bounds().Dy()
	phase := 0
	if !reducedMotion {
		phase = scene.tick / 3
	}
	key := campKey{frame: frame, band: selected, chrome: chrome, phase: phase, width: width, height: height}
	if scene.cached && scene.key == key {
		return CampSkipped
	}
	staticKey, previousStaticKey := key, scene.key
	staticKey.phase, previousStaticKey.phase = 0, 0
	full := !scene.cached || staticKey != previousStaticKey
	transform := FitPresentation(width, height)
	// Scale the illustration once; HUD text remains at native presentation scale.
	scene.illustration(frame, selected, reducedMotion, int(PresentationWidth), true)
	op := &ebiten.DrawImageOptions{Filter: ebiten.FilterLinear}
	op.GeoM.Scale(transform.Scale, transform.Scale)
	op.GeoM.Translate(transform.OffsetX, transform.OffsetY)
	if full {
		screen.Fill(color.NRGBA{R: 5, G: 9, B: 14, A: 255})
		screen.DrawImage(scene.image, op)
	} else {
		// Replace the star field and gathering while preserving captions and
		// letterbox. The composite keeps mountains, trees and moon over stars.
		for _, bounds := range []image.Rectangle{image.Rect(32, 120, 1248, 304), image.Rect(260, 210, 1020, 600)} {
			clip := image.Rect(
				int(math.Floor(transform.OffsetX+float64(bounds.Min.X)*transform.Scale)),
				int(math.Floor(transform.OffsetY+float64(bounds.Min.Y)*transform.Scale)),
				int(math.Ceil(transform.OffsetX+float64(bounds.Max.X)*transform.Scale)),
				int(math.Ceil(transform.OffsetY+float64(bounds.Max.Y)*transform.Scale)),
			).Intersect(screen.Bounds())
			if !clip.Empty() {
				// Clip the destination, keeping neighbouring source pixels
				// available to linear filtering at high-DPI crop boundaries.
				screen.SubImage(clip).(*ebiten.Image).DrawImage(scene.image, op)
			}
		}
	}
	scene.key, scene.cached = key, true
	if full {
		return CampFull
	}
	return CampMotion
}

// Illustration returns cached, caption-free camp artwork at the requested width.
// The image belongs to the scene and is updated in place at 20 Hz. The boolean
// reports whether its pixels changed, allowing a host to repaint only this image.
func (scene *CampScene) Illustration(frame *gameapi.Frame, selected gameapi.BandID, reducedMotion bool, width int) (*ebiten.Image, bool) {
	return scene.illustration(frame, selected, reducedMotion, width, false)
}

func (scene *CampScene) illustration(frame *gameapi.Frame, selected gameapi.BandID, reducedMotion bool, width int, twinkle bool) (*ebiten.Image, bool) {
	width = max(1, width)
	height := max(1, int(math.Round(float64(width)*PresentationHeight/PresentationWidth)))
	phase := 0
	if !reducedMotion {
		phase = scene.tick / 3
	}
	key := campKey{frame: frame, band: selected, phase: phase, width: width, height: height, twinkle: twinkle}
	if scene.imageCached && scene.imageKey == key {
		return scene.image, false
	}
	var band gameapi.Band
	var tile gameapi.Tile
	if frame != nil {
		for _, candidate := range frame.Bands {
			if candidate.ID == selected {
				band = candidate
				break
			}
		}
		if int(band.TileID) < len(frame.Tiles) {
			tile = frame.Tiles[band.TileID]
		}
	}
	resize := scene.image == nil || scene.image.Bounds().Dx() != width || scene.image.Bounds().Dy() != height
	if resize {
		if scene.image != nil {
			scene.image.Deallocate()
			scene.background.Deallocate()
			scene.sky.Deallocate()
		}
		scene.image, scene.background = ebiten.NewImage(width, height), ebiten.NewImage(width, height)
		scene.sky = ebiten.NewImage(width, height)
	}
	scale := float64(width) / PresentationWidth
	if resize || !scene.imageCached || scene.imageKey.frame != frame || scene.imageKey.band != selected || scene.imageKey.twinkle != twinkle {
		scene.sky.Clear()
		drawCampSky(newLogicalCanvas(scene.sky, scale))
		if !twinkle {
			drawCampStars(newLogicalCanvas(scene.sky, scale), selected, 0, false)
		}
		scene.background.Clear()
		drawCampLandscape(newLogicalCanvas(scene.background, scale), tile)
	}
	scene.image.Clear()
	scene.image.DrawImage(scene.sky, nil)
	if twinkle {
		drawCampStars(newLogicalCanvas(scene.image, scale), selected, float64(phase)/20, true)
	}
	scene.image.DrawImage(scene.background, nil)
	drawCampCompany(newLogicalCanvas(scene.image, scale), band, tile, float64(phase)/20)
	scene.imageKey, scene.imageCached = key, true
	return scene.image, true
}

func drawCampCompany(c logicalCanvas, band gameapi.Band, tile gameapi.Tile, seconds float64) {
	flicker := campFlicker(seconds)
	spread := .92 + flicker*.16
	for i := 10; i > 0; i-- {
		campEllipse(c, 640, 468, float64(70+i*19)*spread, float64(14+i*5)*spread, color.NRGBA{R: 229, G: 123, B: 43, A: uint8(4 + flicker*12)})
	}
	people := campCrowd(band.Population)
	for _, person := range people {
		if person.position[1] < 500 {
			drawCampFigure(c, person, band, tile, seconds, flicker)
		}
	}
	drawCampFire(c, seconds, flicker)
	for _, person := range people {
		if person.position[1] >= 500 {
			drawCampFigure(c, person, band, tile, seconds, flicker)
		}
	}
}

// Blend short, irregular brightness changes without using campaign randomness.
// The same light drives the ground, faces and stones so each flare reads as firelight.
func campFlicker(seconds float64) float64 {
	step := seconds * 9
	index := int(math.Floor(step))
	fraction := step - float64(index)
	fraction = smoothstep(fraction)
	from := unitNoise(hashTile(index, 17, 31))
	to := unitNoise(hashTile(index+1, 17, 31))
	return .12 + .76*(from+(to-from)*fraction) + .1*math.Sin(seconds*14)
}

func drawCampPerson(c logicalCanvas, position [4]float64, index int, band gameapi.Band, tile gameapi.Tile, seconds, flicker float64) {
	x, y, size, direction := position[0], position[1], position[2], position[3]
	breath := math.Sin(seconds*1.4+float64(index)*1.9) * (1 + band.Health) * .65
	gesture := math.Sin(seconds*.9+float64(index)*2.3) * 5
	skin, cloak := campPersonColors(index, tile, flicker)
	// Local points mirror the pose so every face and hand turns toward the fire.
	point := func(dx, dy float64) (float32, float32) { return float32(x + dx*size*direction), float32(y + dy*size) }
	line := func(ax, ay, bx, by, width float64, col color.NRGBA) {
		x1, y1 := point(ax, ay)
		x2, y2 := point(bx, by)
		vector.StrokeLine(c, x1, y1, x2, y2, float32(width*size), col, false)
	}
	campEllipse(c, x+direction*9, y+9*size, 57*size, 11*size, color.NRGBA{R: 8, G: 11, B: 13, A: 150})
	// Bent legs and a hide wrap make a seated figure rather than a standing icon.
	line(-3, -12, 32, -7, 24, cloak)
	line(32, -7, 45, 4, 14, cloak)
	line(45, 4, 61, 5, 7, skin)
	line(0, -12, -23, 2, 21, cloak)
	line(-23, 2, 27, 6, 13, cloak)
	line(27, 6, 44, 7, 6, skin)
	campPolygon(c, cloak, x-22*size, y-12*size, x-18*size, y+(-63+breath)*size, x+direction*7*size, y+(-69+breath)*size, x+23*size, y-14*size)
	line(4, -62+breath, 7, -73+breath, 13, skin)
	hx, hy := point(7, -88+breath)
	vector.FillCircle(c, hx, hy, float32(15*size), skin, false)
	campEllipse(c, float64(hx)-direction*5*size, float64(hy)-5*size, 13*size, 12*size, color.NRGBA{R: 19, G: 20, B: 21, A: 255})
	campPolygon(c, skin, float64(hx)+direction*11*size, float64(hy)-3*size, float64(hx)+direction*19*size, float64(hy)+3*size, float64(hx)+direction*10*size, float64(hy)+6*size)
	line(16, -88+breath, 16, -87+breath, 2, color.NRGBA{R: 12, G: 16, B: 19, A: 255})
	line(12, -57+breath, 24, -32+gesture, 10, skin)
	line(24, -32+gesture, 48, -26+gesture*.4, 8, skin)
	line(-10, -55+breath, -8, -27, 10, cloak)
	line(-8, -27, 22, -16, 7, skin)
	// Small rim highlights tie the figures to the fire's changing light.
	line(17, -52+breath, 22, -35+gesture, 1.5, color.NRGBA{R: 196, G: uint8(108 + flicker*30), B: 58, A: uint8(80 + flicker*150)})
	if index == 4 {
		line(45, -27+gesture*.4, 100, -15, 2.5, color.NRGBA{R: 103, G: 76, B: 47, A: 255})
	}
	if index == 1 && band.AcquiredTech&(1<<gameapi.HaftedTools) != 0 {
		line(39, -23, 50, -44, 3, color.NRGBA{R: 120, G: 96, B: 68, A: 255})
	}
}

func drawCampFire(c logicalCanvas, seconds, flicker float64) {
	// Thin, drifting smoke wisps and rising embers are clock-derived, so a
	// frozen scene and fixed-tick screenshots always reproduce the same image.
	for i := 0; i < 7; i++ {
		p := math.Mod(seconds*.13+float64(i)/7, 1)
		x := 640 + math.Sin(p*7+seconds*.4)*17 + p*28
		campEllipse(c, x, 405-p*167, 10+p*20, 6+p*10, color.NRGBA{R: 124, G: 132, B: 135, A: uint8((1 - p) * 17)})
	}
	for i := 0; i < 11; i++ {
		p := math.Mod(seconds*.31+float64(i)/11, 1)
		x := 640 + math.Sin(float64(i)*3+p*6)*p*36
		vector.FillCircle(c, float32(x), float32(434-p*115), float32(1.3-p*.7), color.NRGBA{R: 255, G: 180, B: 75, A: uint8((1 - p) * 220)}, false)
	}
	for i := 0; i < 10; i++ {
		a := float64(i) * math.Pi / 5
		campEllipse(c, 640+math.Cos(a)*53, 465+math.Sin(a)*12, 11, 6, color.NRGBA{R: uint8(57 + flicker*55), G: uint8(51 + flicker*32), B: uint8(42 + flicker*16), A: 255})
	}
	vector.StrokeLine(c, 609, 470, 670, 452, 10, color.NRGBA{R: 62, G: 36, B: 22, A: 255}, false)
	vector.StrokeLine(c, 615, 451, 669, 471, 9, color.NRGBA{R: 82, G: 44, B: 23, A: 255}, false)
	for i := 0; i < 5; i++ {
		x := 614 + float64(i)*13
		height := max(30, 38+22*math.Sin(seconds*7+float64(i)*1.7)+flicker*15)
		sway := math.Sin(seconds*5+float64(i)) * 10
		campPolygon(c, color.NRGBA{R: 226, G: 86, B: 29, A: 210}, x-11, 460, x-13, 436, x+sway, 460-height, x+15, 438, x+10, 461)
		campPolygon(c, color.NRGBA{R: 255, G: 174, B: 54, A: 240}, x-7, 461, x-7, 443, x+sway*.5, 460-height*.70, x+9, 445, x+6, 461)
		campPolygon(c, color.NRGBA{R: 255, G: 226, B: 142, A: 255}, x-3, 461, x, 440-height*.13, x+4, 461)
	}
}

func campPolygon(c logicalCanvas, col color.NRGBA, points ...float64) {
	// Moving shapes are small convex polygons and can use ordinary batched
	// triangles. Only the cached, concave ridgelines need a vector stencil.
	if len(points) <= 10 {
		r, g, b, a := col.RGBA()
		vertices := make([]ebiten.Vertex, len(points)/2)
		indices := make([]uint16, 0, (len(vertices)-2)*3)
		for i := range vertices {
			vertices[i] = ebiten.Vertex{
				DstX: float32(points[i*2]) * c.scale, DstY: float32(points[i*2+1]) * c.scale,
				SrcX: 32, SrcY: 32,
				ColorR: float32(r) / 65535, ColorG: float32(g) / 65535, ColorB: float32(b) / 65535, ColorA: float32(a) / 65535,
			}
			if i >= 2 {
				indices = append(indices, 0, uint16(i-1), uint16(i))
			}
		}
		c.image.DrawTriangles(vertices, indices, campDisc, &ebiten.DrawTrianglesOptions{ColorScaleMode: ebiten.ColorScaleModePremultipliedAlpha})
		return
	}
	var path ebitenvector.Path
	path.MoveTo(float32(points[0])*c.scale, float32(points[1])*c.scale)
	for i := 2; i < len(points); i += 2 {
		path.LineTo(float32(points[i])*c.scale, float32(points[i+1])*c.scale)
	}
	path.Close()
	options := &ebitenvector.DrawPathOptions{AntiAlias: false}
	options.ColorScale.ScaleWithColor(col)
	ebitenvector.FillPath(c.image, &path, nil, options)
}

// A single antialiased disc is shared by smoke, shadows and firelight. These
// transformed sprites batch together instead of rebuilding vector stencils
// for dozens of ellipses on every high-DPI animation frame.
var campDisc = func() *ebiten.Image {
	disc := ebiten.NewImage(64, 64)
	ebitenvector.FillCircle(disc, 32, 32, 31, color.White, true)
	return disc
}()

func campEllipse(c logicalCanvas, x, y, rx, ry float64, col color.NRGBA) {
	op := &ebiten.DrawImageOptions{Filter: ebiten.FilterLinear}
	s := float64(c.scale)
	op.GeoM.Translate(-32, -32)
	op.GeoM.Scale(rx*s/31, ry*s/31)
	op.GeoM.Translate(x*s, y*s)
	op.ColorScale.ScaleWithColor(col)
	c.image.DrawImage(campDisc, op)
}
