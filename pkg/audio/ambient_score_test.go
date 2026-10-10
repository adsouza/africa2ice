package audio

import (
	"math"
	"testing"
)

func TestScoreHasHeadroomStereoAndSilentEdges(t *testing.T) {
	const frames = 60 * sampleRate
	s := newScore(frames, 42)
	var energy, difference, peak, jump float64
	var previous [2]float64
	for i := range frames {
		left, right := s.next()
		for channel, value := range [2]float64{left, right} {
			if math.IsNaN(value) || math.IsInf(value, 0) || math.Abs(value) >= 0.8 {
				t.Fatalf("invalid or clipped sample at %d: %v", i, value)
			}
			if (i == 0 || i == frames-1) && value != 0 {
				t.Fatalf("edge at %d is audible: %v", i, value)
			}
			energy += value * value
			peak = max(peak, math.Abs(value))
			jump = max(jump, math.Abs(value-previous[channel]))
			previous[channel] = value
		}
		difference += (left - right) * (left - right)
		// Denser melody and occasional accompaniment share three-second tails.
		if len(s.strings) > 6 {
			t.Fatalf("unbounded voices at %d: %d", i, len(s.strings))
		}
		if len(s.hits) > 3 {
			t.Fatalf("unbounded percussion at %d: %d", i, len(s.hits))
		}
	}
	rms := math.Sqrt(energy / (frames * 2))
	if rms < 0.03 || rms > 0.2 || difference/energy < 0.01 || peak < 0.1 || jump > 0.1 {
		t.Fatalf("unexpected sound: RMS %.4f, peak %.4f, stereo ratio %.4f, maximum step %.4f", rms, peak, difference/energy, jump)
	}
	t.Logf("RMS %.4f, peak %.4f, stereo ratio %.4f, maximum step %.4f", rms, peak, difference/energy, jump)
}
