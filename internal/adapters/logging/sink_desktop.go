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
		return nil, err
	}
	path, err := filepathAbs(file.Name())
	if err != nil {
		_ = file.Close()
		return nil, err
	}
	if _, err := fmt.Fprintf(announcement, "Africa 2 Ice session log: %s\n", path); err != nil {
		_ = file.Close()
		return nil, err
	}
	return newSession(rand.Reader, "desktop", file), nil
}

func filepathAbs(path string) (string, error) { return filepath.Abs(path) }
