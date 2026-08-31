package render

import (
	"image/color"
	"math"
)

// GradeColors is the continuous presentation grade driven by the accepted
// frame's aridity index. Biome colors deliberately are not part of this value:
// the grade changes water and chrome without changing what a land tile is.
type GradeColors struct {
	Water           color.RGBA
	HUDChromeAccent color.RGBA
}

var epochGradeAnchors = [...]GradeColors{
	{
		Water: color.RGBA{R: 0x20, G: 0x6c, B: 0x9c, A: 0xff}, HUDChromeAccent: color.RGBA{R: 0x9e, G: 0x9e, B: 0x48, A: 0xff},
	},
	{
		Water: color.RGBA{R: 0x38, G: 0x68, B: 0x84, A: 0xff}, HUDChromeAccent: color.RGBA{R: 0xa6, G: 0x80, B: 0x42, A: 0xff},
	},
	{
		Water: color.RGBA{R: 0x34, G: 0x54, B: 0x74, A: 0xff}, HUDChromeAccent: color.RGBA{R: 0x70, G: 0xaa, B: 0xcc, A: 0xff},
	},
}

// EpochGrade linearly interpolates the locked humid, transitional, and
// glacial anchors in eight-bit sRGB space. A malformed presentation input uses
// the humid endpoint; the domain projection independently guarantees finite
// values in normal play.
func EpochGrade(aridityIndex float64) GradeColors {
	if math.IsNaN(aridityIndex) {
		aridityIndex = 0
	}
	aridityIndex = clampRender(aridityIndex)
	segment, fraction := 0, aridityIndex*2
	if aridityIndex >= 0.5 {
		segment, fraction = 1, (aridityIndex-0.5)*2
	}
	return interpolateGrade(epochGradeAnchors[segment], epochGradeAnchors[segment+1], fraction)
}

func interpolateGrade(from, to GradeColors, fraction float64) GradeColors {
	return GradeColors{
		Water:           interpolateRGBA(from.Water, to.Water, fraction),
		HUDChromeAccent: interpolateRGBA(from.HUDChromeAccent, to.HUDChromeAccent, fraction),
	}
}

func interpolateRGBA(from, to color.RGBA, fraction float64) color.RGBA {
	return color.RGBA{
		R: interpolateChannel(from.R, to.R, fraction),
		G: interpolateChannel(from.G, to.G, fraction),
		B: interpolateChannel(from.B, to.B, fraction),
		A: interpolateChannel(from.A, to.A, fraction),
	}
}

func interpolateChannel(from, to uint8, fraction float64) uint8 {
	value := lerp(float64(from), float64(to), fraction)
	return uint8(math.Floor(value + 0.5))
}

func lerp(from, to, fraction float64) float64 {
	return from + (to-from)*fraction
}
