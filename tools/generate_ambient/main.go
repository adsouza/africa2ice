// Command generate_ambient renders an instrumental listening prototype without
// opening a sound device. All instruments and reverb are synthesized in Go.
package main

import (
	"encoding/binary"
	"flag"
	"fmt"
	"io"
	"math"
	"os"
	"time"
)

func main() {
	output := flag.String("out", "ambient-prototype.wav", "output WAV path")
	duration := flag.Duration("duration", 2*time.Minute, "piece duration (1s to 30m)")
	seed := flag.Uint64("seed", 20261010, "private instrument seed")
	flag.Parse()
	if err := run(*output, *duration, *seed); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	fmt.Printf("Rendered %s to %s (44.1 kHz, stereo, 16-bit PCM; seed %d)\n", *duration, *output, *seed)
}

func run(path string, duration time.Duration, seed uint64) error {
	if duration < time.Second || duration > 30*time.Minute {
		return fmt.Errorf("duration must be between 1s and 30m")
	}
	file, err := os.Create(path)
	if err != nil {
		return err
	}
	writeErr := writeWAV(file, int(duration.Seconds()*sampleRate), seed)
	closeErr := file.Close()
	if writeErr != nil {
		return writeErr
	}
	return closeErr
}

func writeWAV(writer io.Writer, frames int, seed uint64) error {
	var header [44]byte
	copy(header[:], "RIFF")
	binary.LittleEndian.PutUint32(header[4:], uint32(frames*4+36))
	copy(header[8:], "WAVEfmt ")
	binary.LittleEndian.PutUint32(header[16:], 16)
	binary.LittleEndian.PutUint16(header[20:], 1)
	binary.LittleEndian.PutUint16(header[22:], 2)
	binary.LittleEndian.PutUint32(header[24:], sampleRate)
	binary.LittleEndian.PutUint32(header[28:], sampleRate*4)
	binary.LittleEndian.PutUint16(header[32:], 4)
	binary.LittleEndian.PutUint16(header[34:], 16)
	copy(header[36:], "data")
	binary.LittleEndian.PutUint32(header[40:], uint32(frames*4))
	if err := writeAll(writer, header[:]); err != nil {
		return err
	}
	s := newScore(frames, seed)
	var buffer [4096]byte
	for remaining := frames; remaining > 0; {
		count := min(remaining, len(buffer)/4)
		for i := range count {
			left, right := s.next()
			binary.LittleEndian.PutUint16(buffer[4*i:], uint16(int16(math.Round(left*32767))))
			binary.LittleEndian.PutUint16(buffer[4*i+2:], uint16(int16(math.Round(right*32767))))
		}
		if err := writeAll(writer, buffer[:count*4]); err != nil {
			return err
		}
		remaining -= count
	}
	return nil
}

func writeAll(writer io.Writer, data []byte) error {
	n, err := writer.Write(data)
	if err == nil && n != len(data) {
		return io.ErrShortWrite
	}
	return err
}
