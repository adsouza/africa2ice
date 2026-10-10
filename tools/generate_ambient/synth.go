package main

import (
	"math"
	"math/rand/v2"
)

const sampleRate = 44_100

// This listening prototype deliberately lives outside the game graph. Its
// private PCG never consumes campaign randomness and it opens no audio device.
type score struct {
	random     *rand.PCG
	frame      int
	frames     int
	nextNote   int
	noteIndex  int
	strings    []pluck
	nextBeat   int
	beatIndex  int
	hits       []percussionHit
	percussion [3][]float64
	room       [2][4]comb
}

// Question-and-answer phrases bring the melody forward. The opening theme
// returns with a changed ending; small runs connect its wider leaps.
var motifs = [8][10]float64{
	{62, 69, 65, 64, 62, 69, 67, 65, 64, 62},
	{65, 69, 72, 69, 67, 65, 64, 62, 60, 65},
	{67, 72, 76, 74, 72, 67, 69, 72, 67, 64},
	{62, 69, 65, 64, 62, 74, 72, 69, 65, 62},
	{65, 70, 74, 72, 70, 69, 65, 62, 65, 70},
	{67, 74, 70, 69, 67, 62, 65, 69, 67, 62},
	{69, 76, 72, 70, 69, 67, 65, 64, 62, 64},
	{74, 69, 65, 64, 62, 65, 67, 69, 65, 62},
}

var bassNotes = [8]float64{50, 53, 48, 50, 46, 43, 45, 50}
var noteOffsets = [10]float64{0, 1, 2, 2.5, 3.5, 5, 6, 7, 8, 9.5}

const (
	melodyStart   = 2.5
	phraseSeconds = 12
	stringFrames  = 3 * sampleRate
)

func newScore(frames int, seed uint64) *score {
	s := &score{random: rand.NewPCG(seed, seed^0x9e3779b97f4a7c15), frames: frames, nextNote: int(melodyStart * sampleRate)}
	s.nextBeat = int((melodyStart + phraseSeconds) * sampleRate)
	s.percussion = synthPercussion(seed)
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
	if s.frame == s.nextBeat {
		s.scheduleBeat()
	}
	var dry [2]float64
	for i := range s.strings {
		p := &s.strings[i]
		value := p.next()
		dry[0] += value * math.Cos(p.pan*math.Pi/2)
		dry[1] += value * math.Sin(p.pan*math.Pi/2)
	}
	live := s.strings[:0]
	for _, p := range s.strings {
		if p.age < stringFrames {
			live = append(live, p)
		}
	}
	clear(s.strings[len(live):])
	s.strings = live
	// The rhythm enters after the opening phrase, below the lead in level.
	percussionFade := smooth((t - melodyStart - phraseSeconds) / 4)
	liveHits := s.hits[:0]
	for _, hit := range s.hits {
		value := s.percussion[hit.kind][hit.position] * hit.gain * percussionFade
		dry[0] += value * math.Cos(hit.pan*math.Pi/2)
		dry[1] += value * math.Sin(hit.pan*math.Pi/2)
		hit.position++
		if hit.position < len(s.percussion[hit.kind]) {
			liveHits = append(liveHits, hit)
		}
	}
	clear(s.hits[len(liveHits):])
	s.hits = liveHits
	var out [2]float64
	for channel := range 2 {
		var wet float64
		for i := range s.room[channel] {
			wet += s.room[channel][i].next(0.65*dry[channel]+0.35*dry[1-channel]) / 4
		}
		// Headroom, gentle saturation, and long edge fades make a standalone
		// preview safe to audition. The fade includes the reverb tail.
		fade := smooth(t/2) * smooth(float64(s.frames-1-s.frame)/sampleRate/8)
		out[channel] = 0.8 * math.Tanh(1.5*(dry[channel]+0.25*wet)) * fade
	}
	s.frame++
	return out[0], out[1]
}

// A sparse 60 BPM hand-drum pattern fits the melody's half-second grid.
// Offbeat shaker accents add movement; every fourth phrase leaves more space.
func (s *score) scheduleBeat() {
	step := s.beatIndex % 8
	sparse := (s.beatIndex/24+1)%4 == 3
	if step == 0 {
		s.hits = append(s.hits, percussionHit{kind: 0, gain: 0.10, pan: 0.45})
	}
	if !sparse {
		switch step {
		case 3, 4:
			s.hits = append(s.hits, percussionHit{kind: 1, gain: 0.055, pan: 0.62})
		case 6:
			s.hits = append(s.hits, percussionHit{kind: 0, gain: 0.045, pan: 0.45})
		}
		if step%2 == 1 {
			gain := 0.022
			if step == 3 || step == 7 {
				gain = 0.028
			}
			s.hits = append(s.hits, percussionHit{kind: 2, gain: gain, pan: 0.72})
		}
	}
	s.beatIndex++
	s.nextBeat = int((melodyStart + phraseSeconds + float64(s.beatIndex)*0.5) * sampleRate)
}

func (s *score) scheduleNote() {
	phrase, step := s.noteIndex/len(noteOffsets), s.noteIndex%len(noteOffsets)
	lead := s.newPluck(motifs[phrase%len(motifs)][step])
	// Keep the tune near the centre and give each phrase a natural accent.
	lead.pan = 0.45 + 0.1*s.randomUnit()
	lead.gain = 0.9
	if step == 0 || step == 5 {
		lead.gain = 1.05
	}
	s.strings = append(s.strings, lead)
	// Two quiet, decaying accompaniment notes per phrase replace the drone.
	if step == 0 || step == 5 {
		midi := bassNotes[phrase%len(bassNotes)]
		pan := 0.25
		if step == 5 {
			midi += 7
			pan = 0.75
		}
		accompaniment := s.newPluck(midi)
		accompaniment.gain = 0.22
		accompaniment.pan = pan
		s.strings = append(s.strings, accompaniment)
	}
	s.noteIndex++
	phrase, step = s.noteIndex/len(noteOffsets), s.noteIndex%len(noteOffsets)
	s.nextNote = int((melodyStart + float64(phrase)*phraseSeconds + noteOffsets[step]) * sampleRate)
}

type pluck struct {
	buffer   []float64
	position int
	age      int
	pan      float64
	filtered float64
	gain     float64
}

// Karplus-Strong: a pitched excitation circulates around a damped delay line.
// Averaging costs half a sample of delay, accounted for in the string tuning.
func (s *score) newPluck(midi float64) pluck {
	p := pluck{buffer: make([]float64, int(float64(sampleRate)/frequency(midi)-0.5)), pan: 0.2 + 0.6*s.randomUnit(), gain: 1}
	var mean float64
	for i := range p.buffer {
		phase := 2 * math.Pi * float64(i) / float64(len(p.buffer))
		// A pitched excitation makes the fundamental reliable. A little noise
		// retains the wooden string attack and seed-dependent texture.
		p.buffer[i] = 0.28*math.Sin(phase) + 0.09*math.Sin(2*phase) + 0.05*(2*s.randomUnit()-1)
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
	release := smooth(float64(stringFrames-1-p.age) / sampleRate / 0.5)
	p.age++
	return p.filtered * attack * release * 0.7 * p.gain
}

type comb struct {
	buffer   []float64
	position int
	filtered float64
}

func (c *comb) next(input float64) float64 {
	value := c.buffer[c.position]
	c.filtered += 0.22 * (value - c.filtered)
	c.buffer[c.position] = input + 0.72*c.filtered
	c.position = (c.position + 1) % len(c.buffer)
	return value
}
