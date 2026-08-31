package logging

import (
	"bytes"
	"errors"
	"strings"
	"testing"
)

type failingWriteCloser struct {
	writes int
	closed bool
}

func (writer *failingWriteCloser) Write([]byte) (int, error) {
	writer.writes++
	return 0, errors.New("disk full")
}

func (writer *failingWriteCloser) Close() error {
	writer.closed = true
	return nil
}

func TestFallbackSinkSwitchesOnceWithoutFailingTheCaller(t *testing.T) {
	primary := &failingWriteCloser{}
	fallback := &bytes.Buffer{}
	sink := newFallbackSink(primary, fallback)
	for _, payload := range [][]byte{[]byte("first\n"), []byte("second\n")} {
		if written, err := sink.Write(payload); err != nil || written != len(payload) {
			t.Fatalf("Write() = (%d, %v), want (%d, nil)", written, err, len(payload))
		}
	}
	if primary.writes != 1 {
		t.Fatalf("primary writes = %d, want one before permanent fallback", primary.writes)
	}
	logged := fallback.String()
	if strings.Count(logged, "log.fallback") != 1 || !strings.Contains(logged, "first\nsecond\n") {
		t.Fatalf("fallback output = %q", logged)
	}
	if err := sink.Close(); err != nil || !primary.closed {
		t.Fatalf("Close() = %v, closed=%t", err, primary.closed)
	}
}
