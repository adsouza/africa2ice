package main

import (
	"math"
	"math/rand/v2"
)

type percussionHit struct {
	kind     int
	position int
	gain     float64
	pan      float64
}

// Three short, reusable synthesized hits. A separate random stream leaves
// the existing melody's string excitation and stereo placement unchanged.
func synthPercussion(seed uint64) [3][]float64 {
	random := rand.NewPCG(seed^0xd1b54a32d192ed03, seed^0x94d049bb133111eb)
	var samples [3][]float64
	for kind := range samples {
		duration := 0.6
		if kind == 2 {
			duration = 0.14
		}
		samples[kind] = make([]float64, int(duration*sampleRate))
		var phase, low, high, peak float64
		for i := range samples[kind] {
			t := float64(i) / sampleRate
			noise := 2*float64(random.Uint64()>>11)/(1<<53) - 1
			low += 0.07 * (noise - low)
			high += 0.55 * (noise - high)
			attack := smooth(t / 0.004)
			release := smooth((duration - t) / 0.025)
			var value float64
			if kind == 2 {
				// Difference of two low-pass filters gives a soft band of rattle.
				value = (high - low) * math.Exp(-t/0.032)
			} else {
				base := 72.0
				decay := 0.14
				if kind == 1 {
					base = 155
					decay = 0.09
				}
				// A falling fundamental and inharmonic membrane modes make a
				// hand-drum body; a short filtered-noise transient adds skin.
				phase += 2 * math.Pi * (base + 45*math.Exp(-t/0.025)) / sampleRate
				value = math.Sin(phase)*math.Exp(-t/decay) +
					0.3*math.Sin(1.59*phase)*math.Exp(-t/0.055) +
					0.12*math.Sin(2.14*phase)*math.Exp(-t/0.035) +
					0.18*high*math.Exp(-t/0.018)
			}
			value *= attack * release
			samples[kind][i] = value
			peak = max(peak, math.Abs(value))
		}
		for i := range samples[kind] {
			samples[kind][i] /= peak
		}
	}
	return samples
}
