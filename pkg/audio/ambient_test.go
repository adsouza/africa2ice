package audio

import (
	"bytes"
	"encoding/binary"
	"io"
	"math"
	"testing"
)

func TestAmbientChunkSizesDoNotAlterPCM(t *testing.T) {
	expected := make([]byte, 16387)
	if _, err := io.ReadFull(NewAmbient(42), expected); err != nil {
		t.Fatal(err)
	}
	stream := NewAmbient(42)
	if n, err := stream.Read(nil); n != 0 || err != nil || stream.score.frame != 0 {
		t.Fatal("empty read advanced music")
	}
	var actual []byte
	for i := 0; len(actual) < len(expected); i++ {
		size := min([]int{1, 7, 31, 1024}[i%4], len(expected)-len(actual))
		buffer := make([]byte, size)
		if _, err := io.ReadFull(stream, buffer); err != nil {
			t.Fatal(err)
		}
		actual = append(actual, buffer...)
	}
	if !bytes.Equal(actual, expected) {
		t.Fatal("device buffer alignment changed PCM")
	}
}

func TestAmbientContinuesAcrossThreeMusicalCycles(t *testing.T) {
	a := NewAmbient(42)
	var buffer [8192]byte
	var previous [2]float64
	var boundaryEnergy [3]float64
	boundaries := [3]float64{scoreTime(96), scoreTime(192), scoreTime(288)}
	end := int((boundaries[2] + 2) * sampleRate)
	for frame := 0; frame < end; {
		count := min(len(buffer)/8, end-frame)
		if _, err := io.ReadFull(a, buffer[:count*8]); err != nil {
			t.Fatal(err)
		}
		for i := range count {
			for channel := range 2 {
				value := float64(math.Float32frombits(binary.LittleEndian.Uint32(buffer[i*8+channel*4:])))
				if math.IsNaN(value) || math.IsInf(value, 0) || math.Abs(value) >= 0.8 || math.Abs(value-previous[channel]) > 0.1 {
					t.Fatalf("invalid or discontinuous PCM at frame %d: %v after %v", frame+i, value, previous[channel])
				}
				previous[channel] = value
				for boundary, seconds := range boundaries {
					if math.Abs(float64(frame+i)/sampleRate-seconds) < 1 {
						boundaryEnergy[boundary] += value * value
					}
				}
			}
		}
		if len(a.score.strings) > 6 || len(a.score.hits) > 3 {
			t.Fatal("voices accumulated across a cycle")
		}
		frame += count
	}
	for cycle, energy := range boundaryEnergy {
		if rms := math.Sqrt(energy / (4 * sampleRate)); rms < 0.02 {
			t.Fatalf("cycle %d fades or stops: RMS %v", cycle, rms)
		}
	}
	// The quicker introduction must never return on subsequent repetitions.
	if math.Abs(scoreTime(192)-scoreTime(96)-96) > 1e-9 || math.Abs(scoreTime(288)-scoreTime(192)-96) > 1e-9 {
		t.Fatal("loop restarted the tempo ramp")
	}
}
