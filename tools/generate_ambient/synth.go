package main

import (
	"math"
	"math/rand/v2"
)

const sampleRate = 44_100

// This listening prototype deliberately lives outside the game graph. Its
// private PCG never consumes campaign randomness and it opens no audio device.
type score struct {
	random    *rand.PCG
	frame     int
	frames    int
	nextNote  int
	noteIndex int
	strings   []pluck
	room      [2][4]comb
}

// Eight open voicings and moving bass roots travel around D natural minor.
// The upper voice changes register as well as pitch to keep the bed moving.
var chords = [8][4]float64{
	{50, 57, 64, 65}, {53, 60, 64, 69},
	{45, 55, 60, 76}, {48, 55, 62, 67},
	{46, 53, 57, 74}, {43, 50, 58, 69},
	{50, 57, 62, 77}, {45, 52, 60, 67},
}

var bassNotes = [8]float64{38, 41, 33, 36, 34, 31, 38, 33}
var suspendedNotes = [8]float64{69, 72, 65, 74, 67, 77, 72, 76}

// Wide leaps alternate with nearer replies and change direction within a
// phrase. Steps, thirds, fourths, fifths, sixths, sevenths, and octaves occur.
var motifs = [8][5]float64{
	{62, 69, 77, 76, 72}, {65, 72, 62, 64, 69},
	{69, 60, 67, 76, 72}, {67, 74, 65, 72, 62},
	{70, 62, 77, 74, 65}, {67, 58, 74, 72, 69},
	{74, 65, 72, 62, 69}, {69, 77, 67, 74, 62},
}

var noteOffsets = [5]float64{0, 2.75, 5.5, 8.5, 11}

func newScore(frames int, seed uint64) *score {
	s := &score{random: rand.NewPCG(seed, seed^0x9e3779b97f4a7c15), frames: frames, nextNote: 8 * sampleRate}
	for channel := range s.room {
		for i, seconds := range [4]float64{0.071, 0.089, 0.113, 0.137} {
			s.room[channel][i].buffer = make([]float64, int((seconds+float64(channel)*0.0037)*sampleRate))
		}
	}
	return s
}

func (s *score) randomUnit() float64 {
	return float64(s.random.Uint64()>>11) / (1 << 53)
}

func frequency(midi float64) float64 {
	return 440 * math.Exp2((midi-69)/12)
}

func smooth(value float64) float64 {
	value = min(1, max(0, value))
	return value * value * (3 - 2*value)
}

// next produces one stereo frame with a fixed amount of reverb memory. The
// score clock advances by samples rather than wall time or rendering chunks.
func (s *score) next() (float64, float64) {
	t := float64(s.frame) / sampleRate
	if s.frame == s.nextNote {
		s.scheduleNote()
	}
	var dry [2]float64
	// Crossfade entire sustained voicings over six seconds. Each oscillator's
	// phase is a function of sample time, so chord changes never reset a phase.
	section := int(t / 18)
	blend := smooth(math.Mod(t, 18) / 6)
	for voice := range 4 {
		for side := range 2 {
			midi := chords[(section+side)%len(chords)][voice]
			weight := 1 - blend
			if side == 1 {
				weight = blend
			}
			for channel := range 2 {
				f := frequency(midi) * (1 + float64(channel)*0.0009)
				phase := 2 * math.Pi * f * t
				breath := 0.8 + 0.2*math.Sin(2*math.Pi*t/17+float64(voice))
				dry[channel] += weight * breath * 0.037 * (math.Sin(phase) + 0.17*math.Sin(2*phase))
			}
		}
	}
	// The bass follows the harmony instead of holding a D for the whole piece.
	// A quieter suspended line moves independently every nine seconds. Fading
	// fixed-pitch oscillators preserves clear pitches without a siren-like glide.
	bass := 0.045 * ((1-blend)*math.Sin(2*math.Pi*frequency(bassNotes[section%len(bassNotes)])*t) +
		blend*math.Sin(2*math.Pi*frequency(bassNotes[(section+1)%len(bassNotes)])*t))
	upperSection := int(t / 9)
	upperBlend := smooth(math.Mod(t, 9) / 4)
	for channel := range 2 {
		for side := range 2 {
			weight := 1 - upperBlend
			if side == 1 {
				weight = upperBlend
			}
			f := frequency(suspendedNotes[(upperSection+side)%len(suspendedNotes)]) * (1 + float64(channel)*0.0007)
			dry[channel] += 0.02 * weight * math.Sin(2*math.Pi*f*t)
		}
	}
	for i := range s.strings {
		p := &s.strings[i]
		value := p.next()
		dry[0] += value * math.Cos(p.pan*math.Pi/2)
		dry[1] += value * math.Sin(p.pan*math.Pi/2)
	}
	live := s.strings[:0]
	for _, p := range s.strings {
		if p.age < 8*sampleRate {
			live = append(live, p)
		}
	}
	clear(s.strings[len(live):])
	s.strings = live
	var out [2]float64
	for channel := range 2 {
		var wet float64
		for i := range s.room[channel] {
			wet += s.room[channel][i].next(0.65*dry[channel]+0.35*dry[1-channel]) / 4
		}
		// Headroom, gentle saturation, and long edge fades make a standalone
		// preview safe to audition. The fade includes the reverb tail.
		fade := smooth(t/6) * smooth(float64(s.frames-1-s.frame)/sampleRate/10)
		out[channel] = 0.8 * math.Tanh(1.5*(dry[channel]+bass+0.45*wet)) * fade
	}
	s.frame++
	return out[0], out[1]
}

func (s *score) scheduleNote() {
	phrase, step := s.noteIndex/len(noteOffsets), s.noteIndex%len(noteOffsets)
	// Leave space in every fourth phrase, but retain its opening leap and reply.
	if phrase%4 != 3 || step < 2 || step == len(noteOffsets)-1 {
		midi := motifs[phrase%len(motifs)][step]
		if phrase%4 == 3 {
			midi -= 12
		}
		s.strings = append(s.strings, s.newPluck(midi))
	}
	s.noteIndex++
	phrase, step = s.noteIndex/len(noteOffsets), s.noteIndex%len(noteOffsets)
	s.nextNote = int((8 + float64(phrase)*14 + noteOffsets[step]) * sampleRate)
}

type pluck struct {
	buffer   []float64
	position int
	age      int
	pan      float64
	filtered float64
}

// Karplus-Strong: a noise excitation circulates around a damped delay line.
// Averaging costs half a sample of delay, accounted for in the string tuning.
func (s *score) newPluck(midi float64) pluck {
	p := pluck{buffer: make([]float64, int(float64(sampleRate)/frequency(midi)-0.5)), pan: 0.2 + 0.6*s.randomUnit()}
	var mean float64
	for i := range p.buffer {
		p.buffer[i] = (2*s.randomUnit() - 1) * 0.4
		mean += p.buffer[i] / float64(len(p.buffer))
	}
	for i := range p.buffer {
		p.buffer[i] -= mean
	}
	// Soften the noise transient before the string is audible.
	for range 16 {
		for i := range p.buffer {
			p.buffer[i] = (p.buffer[i] + p.buffer[(i+1)%len(p.buffer)]) * 0.5
		}
	}
	return p
}

func (p *pluck) next() float64 {
	value := p.buffer[p.position]
	next := (p.position + 1) % len(p.buffer)
	p.buffer[p.position] = 0.996 * (value + p.buffer[next]) * 0.5
	p.position = next
	p.filtered += 0.3 * (value - p.filtered)
	attack := smooth(float64(p.age) / sampleRate / 0.012)
	release := smooth(float64(8*sampleRate-1-p.age) / sampleRate)
	p.age++
	return p.filtered * attack * release * 0.7
}

type comb struct {
	buffer   []float64
	position int
	filtered float64
}

func (c *comb) next(input float64) float64 {
	value := c.buffer[c.position]
	c.filtered += 0.22 * (value - c.filtered)
	c.buffer[c.position] = input + 0.84*c.filtered
	c.position = (c.position + 1) % len(c.buffer)
	return value
}
