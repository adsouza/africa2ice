package render

import (
	"image/color"
	"math"

	"github.com/adsouza/africa2ice/pkg/gameapi"
)

func departureCrowd(population uint32, seconds float64) []campPerson {
	count := campFigureCount(population)
	front := min(count, campForegroundFigures)
	back := count - front
	people := make([]campPerson, 0, count)
	// Depth order stays stable. Every member is visible at departure; the
	// procession drifts right without reaching an illustrated destination.
	for i := range back {
		x := 96 + 480*float64(i+1)/float64(back+1) + seconds*16
		people = append(people, campPerson{position: [4]float64{x, 120, .49, 1}, pose: front + i})
	}
	for i := range front {
		x := 60 + 540*float64(i+1)/float64(front+1) + seconds*20
		people = append(people, campPerson{position: [4]float64{x, 149 + float64(i%3)*2, .64 + float64(i%3)*.025, 1}, pose: i})
	}
	return people
}

func drawDepartureCompany(c logicalCanvas, band gameapi.Band, tile gameapi.Tile, seconds float64) {
	// Sparse tufts drift left underfoot, suggesting travel without changing
	// or revealing any campaign geography. Their phase is presentation-only.
	plants := departurePalette(tile).plants
	plants.A = 85
	if tile.Biome != gameapi.MountainousHighlands {
		for i := 0; i < 18; i++ {
			x := math.Mod(float64(i)*43+departureWidth-seconds*24, departureWidth)
			y := 147 + float64(i%3)*7
			vector.StrokeLine(c, float32(x), float32(y), float32(x-3), float32(y-5), 1.5, plants, false)
		}
	}
	for _, person := range departureCrowd(band.Population, seconds) {
		drawDeparturePerson(c, person, band, tile, seconds)
	}
}

func drawDeparturePerson(c logicalCanvas, person campPerson, band gameapi.Band, tile gameapi.Tile, seconds float64) {
	x, y, size := person.position[0], person.position[1], person.position[2]
	stride := seconds*6.6 + float64(person.pose)*1.7
	swing := math.Sin(stride)
	bob := math.Cos(stride*2) * (1 + band.Health) * .6
	skin := color.NRGBA{R: uint8(130 + person.pose%4*9), G: uint8(85 + person.pose%4*6), B: uint8(57 + person.pose%3*4), A: 255}
	cloak := color.NRGBA{R: uint8(108 + person.pose%3*14), G: uint8(91 + person.pose%3*9), B: uint8(64 + person.pose%3*8), A: 255}
	if tile.LocalTemperatureC < 5 {
		cloak = color.NRGBA{R: 116, G: 111, B: 99, A: 255}
	}
	hair := color.NRGBA{R: 35, G: 31, B: 26, A: 255}
	point := func(dx, dy float64) (float32, float32) { return float32(x + dx*size), float32(y + dy*size) }
	line := func(ax, ay, bx, by, width float64, col color.NRGBA) {
		x1, y1 := point(ax, ay)
		x2, y2 := point(bx, by)
		vector.StrokeLine(c, x1, y1, x2, y2, float32(width*size), col, false)
	}
	campEllipse(c, x+6*size, y+2, 27*size, 4*size, color.NRGBA{R: 49, G: 53, B: 40, A: 90})
	// Opposing articulated legs and lifted trailing feet make the gait read
	// as walking, rather than translating a stationary camp figure.
	for _, side := range []float64{-1, 1} {
		step := swing * side
		kneeX := step*15 + 4
		footX := step * 29
		lift := math.Max(0, -step) * 9
		line(0, -44+bob, kneeX, -23-lift*.4, 11, cloak)
		line(kneeX, -23-lift*.4, footX, -4-lift, 8, skin)
		line(footX, -3-lift, footX+12, -2-lift, 5, skin)
	}
	// A slung hide bundle sits behind the shoulder; a diagonal strap crosses
	// the tunic. Faces, noses, hands, and toes all point along the procession.
	campEllipse(c, x-18*size, y+(-83+bob)*size, 16*size, 22*size, color.NRGBA{R: 144, G: 111, B: 72, A: 255})
	campPolygon(c, cloak, x-14*size, y+(-43+bob)*size, x-15*size, y+(-91+bob)*size,
		x+3*size, y+(-101+bob)*size, x+16*size, y+(-88+bob)*size, x+15*size, y+(-44+bob)*size)
	line(-12, -92+bob, 12, -55+bob, 3, color.NRGBA{R: 189, G: 161, B: 112, A: 255})
	line(3, -98+bob, 6, -109+bob, 9, skin)
	campEllipse(c, x+6*size, y+(-120+bob)*size, 12*size, 15*size, skin)
	campEllipse(c, x+2*size, y+(-127+bob)*size, 12*size, 10*size, hair)
	campPolygon(c, skin, x+15*size, y+(-123+bob)*size, x+23*size, y+(-117+bob)*size, x+15*size, y+(-115+bob)*size)
	line(15, -123+bob, 17, -123+bob, 1.5, hair)
	armSwing := -swing * 12
	line(11, -87+bob, 14+armSwing, -66+bob, 8, cloak)
	line(14+armSwing, -66+bob, 22+armSwing, -49+bob, 6, skin)
	if person.pose%3 == 0 {
		line(22+armSwing, -51+bob, 34+armSwing, -4, 2.5, color.NRGBA{R: 102, G: 81, B: 51, A: 255})
		if band.AcquiredTech&(1<<gameapi.HaftedTools) != 0 {
			line(22+armSwing, -51+bob, 14+armSwing, -81+bob, 2.5, hair)
		}
	} else {
		campEllipse(c, x+(22+armSwing)*size, y+(-40+bob)*size, 9*size, 12*size, color.NRGBA{R: 160, G: 130, B: 87, A: 255})
	}
}
