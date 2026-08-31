package render

import (
	"image/color"
	"math"
)

// GradeColors is the continuous presentation grade driven by the accepted
// frame's aridity index. Biome colors deliberately are not part of this value:
// the grade changes atmosphere and chrome without changing what a tile is.
type GradeColors struct {
	DirectionalLight          color.RGBA
	DirectionalLightIntensity float64
	AmbientLevel              float64
	Veil                      color.RGBA
	Water                     color.RGBA
	HUDChromeAccent           color.RGBA
}

var epochGradeAnchors = [...]GradeColors{
	{
		DirectionalLight: color.RGBA{R: 0xff, G: 0xe8, B: 0xbc, A: 0xff}, DirectionalLightIntensity: 0.94, AmbientLevel: 0.72,
		Veil: color.RGBA{R: 0x48, G: 0x58, B: 0x60, A: 0xff}, Water: color.RGBA{R: 0x20, G: 0x6c, B: 0x9c, A: 0xff}, HUDChromeAccent: color.RGBA{R: 0x9e, G: 0x9e, B: 0x48, A: 0xff},
	},
	{
		DirectionalLight: color.RGBA{R: 0xf1, G: 0xda, B: 0xb6, A: 0xff}, DirectionalLightIntensity: 0.82, AmbientLevel: 0.60,
		Veil: color.RGBA{R: 0x50, G: 0x5a, B: 0x60, A: 0xff}, Water: color.RGBA{R: 0x38, G: 0x68, B: 0x84, A: 0xff}, HUDChromeAccent: color.RGBA{R: 0xa6, G: 0x80, B: 0x42, A: 0xff},
	},
	{
		DirectionalLight: color.RGBA{R: 0xcf, G: 0xe0, B: 0xf0, A: 0xff}, DirectionalLightIntensity: 0.70, AmbientLevel: 0.48,
		Veil: color.RGBA{R: 0x4c, G: 0x58, B: 0x68, A: 0xff}, Water: color.RGBA{R: 0x34, G: 0x54, B: 0x74, A: 0xff}, HUDChromeAccent: color.RGBA{R: 0x70, G: 0xaa, B: 0xcc, A: 0xff},
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
		DirectionalLight:          interpolateRGBA(from.DirectionalLight, to.DirectionalLight, fraction),
		DirectionalLightIntensity: lerp(from.DirectionalLightIntensity, to.DirectionalLightIntensity, fraction),
		AmbientLevel:              lerp(from.AmbientLevel, to.AmbientLevel, fraction),
		Veil:                      interpolateRGBA(from.Veil, to.Veil, fraction),
		Water:                     interpolateRGBA(from.Water, to.Water, fraction),
		HUDChromeAccent:           interpolateRGBA(from.HUDChromeAccent, to.HUDChromeAccent, fraction),
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
