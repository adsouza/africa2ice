package logging

import (
	"fmt"
	"io"
	"sync"
)

// fallbackSink makes operational logging observational: after the first file
// write failure it switches permanently to stderr and reports the transition
// once, but presents successful writes to slog so gameplay is unaffected.
type fallbackSink struct {
	mu       sync.Mutex
	primary  io.WriteCloser
	fallback io.Writer
	failed   bool
}

func newFallbackSink(primary io.WriteCloser, fallback io.Writer) *fallbackSink {
	return &fallbackSink{primary: primary, fallback: fallback}
}

func (sink *fallbackSink) Write(payload []byte) (int, error) {
	sink.mu.Lock()
	defer sink.mu.Unlock()
	if !sink.failed {
		if written, err := sink.primary.Write(payload); err == nil && written == len(payload) {
			return written, nil
		}
		sink.failed = true
		_, _ = fmt.Fprintln(sink.fallback, `{"level":"WARN","msg":"log.fallback","reason":"primary write failed"}`)
	}
	_, _ = sink.fallback.Write(payload)
	return len(payload), nil
}

func (sink *fallbackSink) Close() error {
	sink.mu.Lock()
	defer sink.mu.Unlock()
	return sink.primary.Close()
}
