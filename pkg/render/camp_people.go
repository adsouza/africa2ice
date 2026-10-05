package render

import (
	"image/color"
	"math"

	"github.com/adsouza/africa2ice/pkg/gameapi"
)

func drawCampFigure(c logicalCanvas, person campPerson, band gameapi.Band, tile gameapi.Tile, seconds, flicker float64) {
	if person.standing {
		drawCampStandingPerson(c, person, band, tile, seconds, flicker)
		return
	}
	drawCampPerson(c, person.position, person.pose, band, tile, seconds, flicker)
}

func campPersonColors(index int, tile gameapi.Tile, flicker float64) (skin, cloak color.NRGBA) {
	skin = color.NRGBA{R: uint8(76 + flicker*65), G: uint8(44 + flicker*32), B: 32, A: 255}
	cloak = color.NRGBA{R: uint8(41+index%3*8) + uint8(flicker*17), G: uint8(34+index%3*5) + uint8(flicker*7), B: 29, A: 255}
	if tile.LocalTemperatureC < 5 {
		cloak = color.NRGBA{R: 64 + uint8(flicker*17), G: 61 + uint8(flicker*7), B: 55, A: 255}
	}
	return skin, cloak
}

func drawCampStandingPerson(c logicalCanvas, person campPerson, band gameapi.Band, tile gameapi.Tile, seconds, flicker float64) {
	x, y, size := person.position[0], person.position[1], person.position[2]
	breath := math.Sin(seconds*1.4+float64(person.pose)*1.9) * (1 + band.Health) * .65
	gesture := math.Sin(seconds*.9+float64(person.pose)*2.3) * 3
	skin, cloak := campPersonColors(person.pose, tile, flicker)
	point := func(dx, dy float64) (float32, float32) { return float32(x + dx*size), float32(y + dy*size) }
	line := func(ax, ay, bx, by, width float64, col color.NRGBA) {
		x1, y1 := point(ax, ay)
		x2, y2 := point(bx, by)
		vector.StrokeLine(c, x1, y1, x2, y2, float32(width*size), col, false)
	}
	campEllipse(c, x, y+3*size, 29*size, 7*size, color.NRGBA{R: 8, G: 11, B: 13, A: 150})
	// Straight legs and separate feet anchor an upright, symmetrical pose.
	line(-9, -49, -12, -5, 12, skin)
	line(9, -49, 12, -5, 12, skin)
	line(-12, -3, -21, -1, 7, skin)
	line(12, -3, 21, -1, 7, skin)
	campPolygon(c, cloak, x-16*size, y-47*size, x-22*size, y+(-97+breath)*size,
		x-11*size, y+(-109+breath)*size, x+11*size, y+(-109+breath)*size,
		x+22*size, y+(-97+breath)*size, x+16*size, y-47*size)
	line(0, -108+breath, 0, -119+breath, 12, skin)
	line(-21, -95+breath, -27, -69, 10, cloak)
	line(-27, -69, -24+gesture, -48, 7, skin)
	line(21, -95+breath, 27, -69, 10, cloak)
	line(27, -69, 24+gesture, -48, 7, skin)
	hx, hy := point(0, -132+breath)
	campEllipse(c, float64(hx), float64(hy), 14*size, 16*size, skin)
	hair := color.NRGBA{R: 19, G: 20, B: 21, A: 255}
	campEllipse(c, float64(hx), float64(hy)-10*size, 14*size, 7*size, hair)
	// Two visible eyes and a central nose make the face look toward the viewer.
	ink := color.NRGBA{R: 12, G: 16, B: 19, A: 255}
	line(-6, -133+breath, -4, -133+breath, 2, ink)
	line(4, -133+breath, 6, -133+breath, 2, ink)
	line(0, -131+breath, 0, -126+breath, 1.5, hair)
	line(-3, -122+breath, 3, -122+breath, 1.5, hair)
	rim := color.NRGBA{R: 196, G: uint8(108 + flicker*30), B: 58, A: uint8(80 + flicker*150)}
	line(-19, -91+breath, -23, -73, 1.5, rim)
	line(19, -91+breath, 23, -73, 1.5, rim)
}
