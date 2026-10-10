package main

import (
	"math"
	"math/rand/v2"
)

var fluteReplies = [4][2]float64{{69, 65}, {67, 62}, {70, 67}, {64, 62}}

// Two short notes answer every second plucked phrase. Their beat positions
// fall into the closing gap, leaving the plucked melody as the lead voice.
func fluteBeat(index int) float64 {
	phrase := 1 + 2*(index/2)
	return float64(phrase)*phraseBeats + 10.25 + float64(index%2)
}

func (s *score) scheduleFlute() {
	beat := fluteBeat(s.fluteIndex)
	frames := int(scoreTime(beat+0.75)*sampleRate) - s.frame
	midi := fluteReplies[(s.fluteIndex/2)%len(fluteReplies)][s.fluteIndex%2]
	s.flute = &fluteVoice{frequency: frequency(midi), frames: frames}
	s.fluteIndex++
	s.nextFlute = int(scoreTime(fluteBeat(s.fluteIndex)) * sampleRate)
}

type fluteVoice struct {
	frequency float64
	frames    int
	age       int
	phase     float64
	low       float64
	high      float64
}

func (f *fluteVoice) next(random *rand.PCG) float64 {
	t := float64(f.age) / sampleRate
	// Gentle delayed vibrato, a rounded harmonic spectrum, and filtered breath
	// distinguish the flute from the string's sharper attack and decay.
	vibrato := 0.0025 * smooth(t/0.3) * math.Sin(2*math.Pi*5*t)
	f.phase += 2 * math.Pi * f.frequency * (1 + vibrato) / sampleRate
	noise := 2*float64(random.Uint64()>>11)/(1<<53) - 1
	f.low += 0.025 * (noise - f.low)
	f.high += 0.18 * (noise - f.high)
	envelope := smooth(t/0.12) * smooth(float64(f.frames-1-f.age)/sampleRate/0.18)
	value := math.Sin(f.phase) + 0.15*math.Sin(2*f.phase) + 0.035*math.Sin(3*f.phase) + 0.11*(f.high-f.low)
	f.age++
	return 0.065 * envelope * value
}
