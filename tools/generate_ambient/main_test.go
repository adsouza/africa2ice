package main

import (
	"bytes"
	"encoding/binary"
	"errors"
	"io"
	"testing"
	"time"

	"github.com/adsouza/africa2ice/pkg/audio"
)

func TestWAVIsReproducibleAndSeedVariesTheStrings(t *testing.T) {
	const frames = 12 * audio.SampleRate
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
	if binary.LittleEndian.Uint32(data[4:]) != uint32(len(data)-8) || binary.LittleEndian.Uint32(data[40:]) != frames*4 || binary.LittleEndian.Uint32(data[24:]) != audio.SampleRate || binary.LittleEndian.Uint16(data[22:]) != 2 || binary.LittleEndian.Uint16(data[34:]) != 16 {
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
			if err := writeWAV(writer, audio.SampleRate, 42); !errors.Is(err, want) {
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
