package main

import (
	"bytes"
	"encoding/binary"
	"errors"
	"io"
	"math"
	"testing"
	"time"
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
	}
	rms := math.Sqrt(energy / (frames * 2))
	if rms < 0.03 || rms > 0.2 || difference/energy < 0.01 || peak < 0.1 || jump > 0.1 {
		t.Fatalf("unexpected sound: RMS %.4f, peak %.4f, stereo ratio %.4f, maximum step %.4f", rms, peak, difference/energy, jump)
	}
	t.Logf("RMS %.4f, peak %.4f, stereo ratio %.4f, maximum step %.4f", rms, peak, difference/energy, jump)
}

func TestWAVIsReproducibleAndSeedVariesTheStrings(t *testing.T) {
	const frames = 12 * sampleRate
	var first, repeated, other bytes.Buffer
	for _, render := range []struct {
		target *bytes.Buffer
		seed   uint64
	}{{&first, 42}, {&repeated, 42}, {&other, 43}} {
		if err := writeWAV(render.target, frames, render.seed); err != nil {
			t.Fatal(err)
		}
	}
	if !bytes.Equal(first.Bytes(), repeated.Bytes()) || bytes.Equal(first.Bytes(), other.Bytes()) {
		t.Fatal("render must repeat with the same seed and vary with a different seed")
	}
	data := first.Bytes()
	if len(data) != 44+frames*4 || string(data[:4]) != "RIFF" || string(data[8:16]) != "WAVEfmt " || string(data[36:40]) != "data" {
		t.Fatal("invalid WAV layout")
	}
	if binary.LittleEndian.Uint32(data[4:]) != uint32(len(data)-8) || binary.LittleEndian.Uint32(data[40:]) != frames*4 || binary.LittleEndian.Uint32(data[24:]) != sampleRate || binary.LittleEndian.Uint16(data[22:]) != 2 || binary.LittleEndian.Uint16(data[34:]) != 16 {
		t.Fatal("WAV header does not describe its PCM payload")
	}
}

type failedWriter struct {
	writes int
	failAt int
	err    error
}

func (w *failedWriter) Write(data []byte) (int, error) {
	w.writes++
	if w.writes == w.failAt {
		return len(data) - 1, w.err
	}
	return len(data), nil
}

func TestWAVPropagatesHeaderAndPayloadWriteFailures(t *testing.T) {
	sentinel := errors.New("disk full")
	for _, failAt := range []int{1, 2} {
		for _, failure := range []error{nil, sentinel} {
			writer := &failedWriter{failAt: failAt, err: failure}
			want := failure
			if want == nil {
				want = io.ErrShortWrite
			}
			if err := writeWAV(writer, sampleRate, 42); !errors.Is(err, want) {
				t.Fatalf("write %d: got %v, want %v", failAt, err, want)
			}
		}
	}
}

func TestInvalidDurationDoesNotCreateOutput(t *testing.T) {
	for _, duration := range []time.Duration{0, -time.Second, time.Millisecond, 31 * time.Minute} {
		// A nonexistent parent distinguishes validation from filesystem failure.
		if err := run(t.TempDir()+"/missing/output.wav", duration, 42); err == nil || err.Error() != "duration must be between 1s and 30m" {
			t.Fatalf("duration %v: %v", duration, err)
		}
	}
}
