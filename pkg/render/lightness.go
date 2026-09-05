package render

import (
	"image/color"
	"math"
)

// cieLightness is the L* of an 8-bit sRGB color. The halo ladder (halo.go) is
// specified in this space rather than as an alpha over the terrain color:
// blending is linear in sRGB channels and perceived lightness is not, so one
// blend fraction applied across a palette with a wide L* range is not a uniform
// dimming. See the spec's section 4.1 for the measurement that ruled it out.
func cieLightness(value color.RGBA) float64 {
	lightness, _, _ := cieLab(value)
	return lightness
}

// cieLab converts 8-bit sRGB to CIELAB under a D65 white point.
func cieLab(value color.RGBA) (lightness, greenRed, blueYellow float64) {
	red, green, blue := linearize(value.R), linearize(value.G), linearize(value.B)
	x := (red*0.4124564 + green*0.3575761 + blue*0.1804375) / 0.95047
	y := red*0.2126729 + green*0.7151522 + blue*0.0721750
	z := (red*0.0193339 + green*0.1191920 + blue*0.9503041) / 1.08883
	fx, fy, fz := labCurve(x), labCurve(y), labCurve(z)
	return 116*fy - 16, 500 * (fx - fy), 200 * (fy - fz)
}

func labCurve(component float64) float64 {
	if component > 216.0/24389.0 {
		return math.Cbrt(component)
	}
	return (841.0/108.0)*component + 4.0/29.0
}

func linearize(channel uint8) float64 {
	value := float64(channel) / 255
	if value <= 0.04045 {
		return value / 12.92
	}
	return math.Pow((value+0.055)/1.055, 2.4)
}
