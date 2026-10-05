package render

import (
	"image/color"
	"math"

	"github.com/adsouza/africa2ice/pkg/gameapi"
)

func drawCampSky(c logicalCanvas) {
	for y := 0; y < 720; y += 6 {
		f := float64(y) / 720
		vector.FillRect(c, 0, float32(y), 1280, 6, color.NRGBA{R: uint8(8 + 15*f), G: uint8(16 + 10*f), B: uint8(27 + 8*f), A: 255}, false)
	}
}

// Stable positions and individual phases keep stars shimmering independently.
// The miniature uses the original fixed star field; reduced motion holds the
// full-size stars at time zero along with the rest of the camp illustration.
func drawCampStars(c logicalCanvas, band gameapi.BandID, seconds float64, twinkle bool) {
	for i := 0; i < 75; i++ {
		x := 45 + 1190*unitNoise(hashTile(i, int(band), 1))
		y := 130 + 165*unitNoise(hashTile(i, int(band), 2))
		radius := .6 + unitNoise(hashTile(i, 0, 3))
		if !twinkle {
			vector.FillCircle(c, float32(x), float32(y), float32(radius), color.NRGBA{R: 166, G: 185, B: 194, A: 150}, false)
			continue
		}
		phase := unitNoise(hashTile(i, int(band), 4)) * 2 * math.Pi
		speed := .9 + .7*unitNoise(hashTile(i, int(band), 5))
		brightness := .5 + .35*math.Sin(seconds*speed+phase) + .15*math.Sin(seconds*speed*2.3+phase*1.7)
		col := color.NRGBA{R: 180, G: 199, B: 219, A: uint8(80 + brightness*155)}
		campEllipse(c, x, y, radius*(.8+brightness*.3), radius*(.8+brightness*.3), col)
		if i%13 == 0 {
			flare := math.Max(0, (brightness-.75)*4)
			col.A = uint8(flare * 80)
			campEllipse(c, x, y, 2+flare*1.5, .45, col)
			campEllipse(c, x, y, .45, 2+flare*1.5, col)
		}
	}
}
