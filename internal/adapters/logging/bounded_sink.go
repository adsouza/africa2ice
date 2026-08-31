package logging

import (
	"fmt"
	"io"
	"sync"
)

const maxDesktopLogBytes int64 = 32 * 1024 * 1024

type truncatingWriteCloser interface {
	io.WriteCloser
	io.Seeker
	Truncate(int64) error
}

type boundedSink struct {
	mu      sync.Mutex
	file    truncatingWriteCloser
	limit   int64
	written int64
}

func newBoundedSink(file truncatingWriteCloser, limit int64) *boundedSink {
	return &boundedSink{file: file, limit: limit}
}

func (sink *boundedSink) Write(payload []byte) (int, error) {
	sink.mu.Lock()
	defer sink.mu.Unlock()
	if sink.limit <= 0 {
		return 0, fmt.Errorf("invalid log byte limit %d", sink.limit)
	}
	originalLength := len(payload)
	if int64(len(payload)) > sink.limit {
		payload = []byte(fmt.Sprintf("{\"level\":\"WARN\",\"msg\":\"log.record_clipped\",\"original_bytes\":%d}\n", originalLength))
	}
	if sink.written+int64(len(payload)) > sink.limit {
		if err := sink.file.Truncate(0); err != nil {
			return 0, err
		}
		if _, err := sink.file.Seek(0, io.SeekStart); err != nil {
			return 0, err
		}
		marker := []byte("{\"level\":\"WARN\",\"msg\":\"log.wrapped\"}\n")
		written, err := sink.file.Write(marker)
		sink.written = int64(written)
		if err != nil {
			return 0, err
		}
	}
	written, err := sink.file.Write(payload)
	sink.written += int64(written)
	if err != nil {
		return written, err
	}
	// io.Writer reports consumption of the original record even when the sink
	// deliberately replaced an oversize record with bounded metadata.
	return originalLength, nil
}

func (sink *boundedSink) Close() error {
	sink.mu.Lock()
	defer sink.mu.Unlock()
	return sink.file.Close()
}
