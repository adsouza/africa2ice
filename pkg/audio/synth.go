package audio

import (
	"encoding/binary"
	"math"
)

const (
	sampleRate       = 44_100
	channelCount     = 2
	bytesPerSample   = 4
	soundDurationSec = 0.12
)

// Sound identifies one stable UI event. The generated tones are an adapter
// detail, so asset-backed effects can replace them without changing callers.
type Sound uint8

const (
	SFXChoiceClick Sound = iota
	SFXSaveComplete
	SFXEventTrigger
	soundCount
)

var soundPitchHz = [soundCount]float64{440, 660, 880}

// synthPCM returns interleaved stereo float32 samples. A short attack and
// quadratic release avoid clicks without shipping binary audio assets.
func synthPCM(sound Sound) []byte {
	if sound >= soundCount {
		return nil
	}
	frames := int(sampleRate * soundDurationSec)
	result := make([]byte, frames*channelCount*bytesPerSample)
	frequency := soundPitchHz[sound]
	for frame := 0; frame < frames; frame++ {
		progress := float64(frame) / float64(frames-1)
		attack := min(progress/0.08, 1)
		release := 1 - progress
		envelope := attack * release * release
		phase := 2 * math.Pi * frequency * float64(frame) / sampleRate
		value := float32(0.32 * envelope * (0.82*math.Sin(phase) + 0.18*math.Sin(2*phase)))
		bits := math.Float32bits(value)
		for channel := 0; channel < channelCount; channel++ {
			offset := (frame*channelCount + channel) * bytesPerSample
			binary.LittleEndian.PutUint32(result[offset:offset+bytesPerSample], bits)
		}
	}
	return result
}
