package render

import (
	"image/color"
	"math"
	"testing"

	"github.com/adsouza/africa2ice/pkg/gameapi"
)

// Terrain tiles are drawn at mapTileSize (8) minus the 0.4 gap drawFlatTerrain
// leaves, so every biome is read as a 7.6px square. Below roughly one degree of
// visual angle the eye resolves far less chroma detail than luminance, which
// makes lightness — not hue — the axis that actually separates tiles. These two
// thresholds encode that constraint; they are not an aesthetic preference.
//
// The palette was twice "fixed" by moving a swatch sideways in hue and checking
// CIEDE2000 on large swatches. Both times the map still read as one wash,
// because a large-swatch metric does not model a 7.6px tile and because two
// colors can score a healthy total distance while sharing a lightness. Hence
// the lightness floor is asserted separately rather than folded into the total.
const (
	// minBiomeLightnessGap is the smallest allowed |ΔL*| between any two biome
	// swatches. Lightness is the one axis that survives both tile size and every
	// color-vision deficiency, so it carries the separation on its own.
	minBiomeLightnessGap = 8.0
	// minSwatchColorDifference is the smallest allowed CIEDE2000 between any two
	// swatches that can appear on screen together, under normal vision and under
	// each simulated color-vision deficiency.
	minSwatchColorDifference = 10.0
	// waterGradeSamples walks the aridity range EpochGrade accepts. The three
	// locked anchors are not sufficient on their own: water is interpolated
	// between them, and an intermediate blend can land nearer a biome than
	// either endpoint does.
	waterGradeSamples = 101
)

// visionModels are the filters a swatch pair must stay separable through.
var visionModels = []struct {
	name string
	// apply maps a color to what the viewer perceives; nil is normal vision.
	apply func(color.RGBA) color.RGBA
}{
	{name: "normal", apply: nil},
	{name: "protanopia", apply: simulateProtanopia},
	{name: "deuteranopia", apply: simulateDeuteranopia},
	{name: "tritanopia", apply: simulateTritanopia},
}

func TestBiomeSwatchesStayDistinguishableAtTileSize(t *testing.T) {
	biomes := biomeSwatches()

	t.Run("lightness ladder", func(t *testing.T) {
		worst, first, second := math.Inf(1), "", ""
		for i := range biomes {
			for j := i + 1; j < len(biomes); j++ {
				gap := math.Abs(cieLightness(biomes[i].value) - cieLightness(biomes[j].value))
				if gap < worst {
					worst, first, second = gap, biomes[i].name, biomes[j].name
				}
			}
		}
		if worst < minBiomeLightnessGap {
			t.Fatalf("closest biome pair in lightness is %s/%s at |ΔL*| = %.2f, want >= %.2f; "+
				"at 7.6px lightness is what separates tiles, so hue alone cannot carry this pair",
				first, second, worst, minBiomeLightnessGap)
		}
	})

	t.Run("perceived difference", func(t *testing.T) {
		for _, model := range visionModels {
			t.Run(model.name, func(t *testing.T) {
				worst, first, second, when := math.Inf(1), "", "", 0.0
				for sample := range waterGradeSamples {
					aridity := float64(sample) / float64(waterGradeSamples-1)
					// Water is one color per frame, so each aridity is scored as its
					// own screen. Comparing two grade anchors to each other would be a
					// false positive: they never appear together.
					swatches := coVisibleSwatches(aridity)
					for i := range swatches {
						for j := i + 1; j < len(swatches); j++ {
							difference := colorDifference(
								perceived(swatches[i].value, model.apply),
								perceived(swatches[j].value, model.apply),
							)
							if difference < worst {
								worst, first, second, when = difference, swatches[i].name, swatches[j].name, aridity
							}
						}
					}
				}
				if worst < minSwatchColorDifference {
					t.Fatalf("closest co-visible pair under %s is %s/%s at ΔE2000 = %.2f (aridity %.2f), want >= %.2f",
						model.name, first, second, worst, when, minSwatchColorDifference)
				}
			})
		}
	})
}

type swatch struct {
	name  string
	value color.RGBA
}

func biomeSwatches() []swatch {
	swatches := make([]swatch, 0, gameapi.BiomeCount)
	for biome := range gameapi.Biome(gameapi.BiomeCount) {
		// The aridity argument is deliberately ignored by climateBiomeColor, and
		// TestLegendUsesStableBiomeIdentityAndTheLiveAtmosphericWaterGrade pins
		// that, so one sample defines the whole ladder.
		swatches = append(swatches, swatch{name: biome.String(), value: climateBiomeColor(biome, 0)})
	}
	return swatches
}

// coVisibleSwatches returns every fill that can share the terrain layer at the
// given aridity: the biomes, the unexplored ground, and the single water color
// the grade yields for that frame.
func coVisibleSwatches(aridity float64) []swatch {
	return append(biomeSwatches(),
		swatch{name: "unexplored", value: unexploredTileColor},
		swatch{name: "water", value: EpochGrade(aridity).Water},
	)
}

func perceived(value color.RGBA, apply func(color.RGBA) color.RGBA) color.RGBA {
	if apply == nil {
		return value
	}
	return apply(value)
}

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

func encodeChannel(value float64) uint8 {
	value = math.Max(0, math.Min(1, value))
	if value <= 0.0031308 {
		value *= 12.92
	} else {
		value = 1.055*math.Pow(value, 1/2.4) - 0.055
	}
	return uint8(math.Round(255 * value))
}

// colorDifference is CIEDE2000 with all three weighting factors at 1.
func colorDifference(first, second color.RGBA) float64 {
	l1, a1, b1 := cieLab(first)
	l2, a2, b2 := cieLab(second)
	return labDifference(l1, a1, b1, l2, a2, b2)
}

// labDifference is split out from colorDifference so the metric can be checked
// against the published CIEDE2000 reference pairs, which are given in CIELAB.
// The hue-mean and blue-region rotation terms are the two places this formula
// is usually got wrong, and both are silent: a broken implementation returns
// plausible numbers and every threshold built on it passes for free.
func labDifference(l1, a1, b1, l2, a2, b2 float64) float64 {
	chroma1, chroma2 := math.Hypot(a1, b1), math.Hypot(a2, b2)
	meanChroma := (chroma1 + chroma2) / 2
	// g expands a* near the neutral axis so grey pairs are not over-rewarded.
	g := 0.5 * (1 - math.Sqrt(pow7(meanChroma)/(pow7(meanChroma)+pow7(25))))
	a1p, a2p := (1+g)*a1, (1+g)*a2
	c1p, c2p := math.Hypot(a1p, b1), math.Hypot(a2p, b2)
	h1p, h2p := hueAngle(a1p, b1), hueAngle(a2p, b2)

	deltaL := l2 - l1
	deltaC := c2p - c1p
	deltaHue := 0.0
	if c1p*c2p != 0 {
		deltaHue = h2p - h1p
		switch {
		case deltaHue > 180:
			deltaHue -= 360
		case deltaHue < -180:
			deltaHue += 360
		}
	}
	deltaH := 2 * math.Sqrt(c1p*c2p) * math.Sin(radians(deltaHue/2))

	meanL := (l1 + l2) / 2
	meanCp := (c1p + c2p) / 2
	meanHp := meanHueAngle(h1p, h2p, c1p*c2p != 0)

	tone := 1 - 0.17*math.Cos(radians(meanHp-30)) + 0.24*math.Cos(radians(2*meanHp)) +
		0.32*math.Cos(radians(3*meanHp+6)) - 0.20*math.Cos(radians(4*meanHp-63))
	lightnessWeight := 1 + (0.015*sq(meanL-50))/math.Sqrt(20+sq(meanL-50))
	chromaWeight := 1 + 0.045*meanCp
	hueWeight := 1 + 0.015*meanCp*tone
	// The rotation term cancels the blue region's chroma/hue interaction.
	rotation := -2 * math.Sqrt(pow7(meanCp)/(pow7(meanCp)+pow7(25))) *
		math.Sin(radians(60*math.Exp(-sq((meanHp-275)/25))))

	lightnessTerm := deltaL / lightnessWeight
	chromaTerm := deltaC / chromaWeight
	hueTerm := deltaH / hueWeight
	return math.Sqrt(sq(lightnessTerm) + sq(chromaTerm) + sq(hueTerm) + rotation*chromaTerm*hueTerm)
}

func hueAngle(a, b float64) float64 {
	if a == 0 && b == 0 {
		return 0
	}
	return math.Mod(degrees(math.Atan2(b, a))+360, 360)
}

func meanHueAngle(first, second float64, bothChromatic bool) float64 {
	if !bothChromatic {
		return first + second
	}
	switch difference := math.Abs(first - second); {
	case difference <= 180:
		return (first + second) / 2
	case first+second < 360:
		return (first + second + 360) / 2
	default:
		return (first + second - 360) / 2
	}
}

func radians(degrees float64) float64 { return degrees * math.Pi / 180 }
func degrees(radians float64) float64 { return radians * 180 / math.Pi }
func sq(value float64) float64        { return value * value }
func pow7(value float64) float64 {
	cube := value * value * value
	return cube * cube * value
}

// The three simulations below share the Viénot/Brettel long-medium-short cone
// space. Protanopia and deuteranopia use the Viénot (1999) single-plane
// projection, which is exact for dichromats; tritanopia uses Brettel's.
func simulateProtanopia(value color.RGBA) color.RGBA {
	_, medium, short := toCones(value)
	return fromCones(2.02344*medium-2.52581*short, medium, short)
}

func simulateDeuteranopia(value color.RGBA) color.RGBA {
	long, _, short := toCones(value)
	return fromCones(long, 0.494207*long+1.24827*short, short)
}

func simulateTritanopia(value color.RGBA) color.RGBA {
	long, medium, _ := toCones(value)
	return fromCones(long, medium, -0.395913*long+0.801109*medium)
}

func toCones(value color.RGBA) (long, medium, short float64) {
	red, green, blue := linearize(value.R), linearize(value.G), linearize(value.B)
	return red*17.8824 + green*43.5161 + blue*4.11935,
		red*3.45565 + green*27.1554 + blue*3.86714,
		red*0.0299566 + green*0.184309 + blue*1.46709
}

func fromCones(long, medium, short float64) color.RGBA {
	return color.RGBA{
		R: encodeChannel(0.0809444479*long - 0.130504409*medium + 0.116721066*short),
		G: encodeChannel(-0.0102485335*long + 0.0540193266*medium - 0.113614708*short),
		B: encodeChannel(-0.000365296938*long - 0.00412161469*medium + 0.693511405*short),
		A: 0xff,
	}
}

// TestColorDifferenceMatchesCIEDE2000ReferenceData pins labDifference against
// standard CIEDE2000 pairs, cross-checked against an independent implementation
// (scikit-image), which agrees with this one to 0 decimal places over 20k random
// Lab pairs. Without this the thresholds in the test above would rest on an
// unverified metric, and a formula error that inflated distances would let that
// test pass no matter what the palette did.
//
// The two wraparound rows differ by 0.0002 in b2 and must produce *different*
// answers; they straddle the mean-hue branch and are the usual place this
// formula is got wrong. The blue rows are the only place the rotation term is
// load-bearing. Both errors are silent: they return plausible numbers.
func TestColorDifferenceMatchesCIEDE2000ReferenceData(t *testing.T) {
	tests := []struct {
		name                   string
		l1, a1, b1, l2, a2, b2 float64
		want                   float64
	}{
		{name: "near-neutral hue wrap", l1: 50, a1: 2.5, b1: 0, l2: 50, a2: 3.1736, b2: 0.5854, want: 1.0000},
		{name: "neutral to slight green", l1: 50, a1: 0, b1: 0, l2: 50, a2: -1, b2: 2, want: 2.3669},
		{name: "argument order is symmetric", l1: 50, a1: -1, b1: 2, l2: 50, a2: 0, b2: 0, want: 2.3669},
		{name: "hue wraparound, below the branch", l1: 50, a1: 2.49, b1: -0.001, l2: 50, a2: -2.49, b2: 0.0009, want: 7.1792},
		{name: "hue wraparound, above the branch", l1: 50, a1: 2.49, b1: -0.001, l2: 50, a2: -2.49, b2: 0.0011, want: 7.2195},
		{name: "green pair", l1: 60.2574, a1: -34.0099, b1: 36.2677, l2: 60.4626, a2: -34.1751, b2: 39.4387, want: 1.2644},
		{name: "blue rotation, teal", l1: 63.0109, a1: -31.0961, b1: -5.8663, l2: 62.8187, a2: -29.7946, b2: -4.0864, want: 1.2630},
		{name: "blue rotation, deep", l1: 22.7233, a1: 20.0904, b1: -46.694, l2: 23.0331, a2: 14.973, b2: -42.5619, want: 2.0373},
		{name: "blue rotation, saturated", l1: 50, a1: 2.6772, b1: -79.7751, l2: 50, a2: 0, b2: -82.7485, want: 2.0425},
		{name: "blue rotation, mid", l1: 50, a1: 2.8361, b1: -74.02, l2: 50, a2: 0, b2: -82.7485, want: 3.4412},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got := labDifference(test.l1, test.a1, test.b1, test.l2, test.a2, test.b2)
			if math.Abs(got-test.want) > 0.0001 {
				t.Fatalf("labDifference = %.4f, want %.4f", got, test.want)
			}
		})
	}
}
