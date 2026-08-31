//go:build !js

package logging

import (
	"crypto/rand"
	"fmt"
	"io"
	"os"
	"path/filepath"
)

func NewDefaultSession(announcement io.Writer) (*Session, error) {
	file, err := os.CreateTemp(os.TempDir(), "africa2ice-*.jsonl")
	if err != nil {
		_, _ = fmt.Fprintf(announcement, "Africa 2 Ice session log: stderr (temporary file unavailable)\n")
		session := newSession(rand.Reader, "desktop", noCloseWriteCloser{os.Stderr})
		session.logger.Warn("log.fallback", "reason", "temporary file creation failed", "error", err.Error())
		return session, nil
	}
	path, err := filepathAbs(file.Name())
	if err != nil {
		_ = file.Close()
		_, _ = fmt.Fprintf(announcement, "Africa 2 Ice session log: stderr (path resolution failed)\n")
		session := newSession(rand.Reader, "desktop", noCloseWriteCloser{os.Stderr})
		session.logger.Warn("log.fallback", "reason", "path resolution failed", "error", err.Error())
		return session, nil
	}
	if _, err := fmt.Fprintf(announcement, "Africa 2 Ice session log: %s\n", path); err != nil {
		_, _ = fmt.Fprintf(os.Stderr, "Africa 2 Ice session log: %s\n", path)
	}
	primary := newBoundedSink(file, maxDesktopLogBytes)
	return newSession(rand.Reader, "desktop", newFallbackSink(primary, os.Stderr)), nil
}

func filepathAbs(path string) (string, error) { return filepath.Abs(path) }

type noCloseWriteCloser struct{ io.Writer }

func (noCloseWriteCloser) Close() error { return nil }
