package audio

import (
	"encoding/binary"
	"math"
)

// Ambient is an endless, single-reader stereo float32 PCM source. Version 5's
// opening tempo plays once; its eight-phrase arrangement then cycles at 60 BPM.
// Instruments and reverb carry over the boundary without a restart or fade.
type Ambient struct {
	score     *score
	frame     [channelCount * bytesPerSample]byte
	remaining []byte
}

func NewAmbient(seed uint64) *Ambient {
	return &Ambient{score: newScore(0, seed)}
}

// Read preserves partial frames, so device buffer sizes cannot change the
// music or discard bytes. It returns a full buffer and never signals EOF.
func (a *Ambient) Read(buffer []byte) (int, error) {
	length := len(buffer)
	for len(buffer) > 0 {
		if len(a.remaining) == 0 {
			left, right := a.score.next()
			binary.LittleEndian.PutUint32(a.frame[:4], math.Float32bits(float32(left)))
			binary.LittleEndian.PutUint32(a.frame[4:], math.Float32bits(float32(right)))
			a.remaining = a.frame[:]
		}
		n := copy(buffer, a.remaining)
		buffer = buffer[n:]
		a.remaining = a.remaining[n:]
	}
	return length, nil
}
