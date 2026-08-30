//go:build js

package logging

import (
	"crypto/rand"
	"io"
	"os"
)

type noCloseWriter struct{ io.Writer }

func (noCloseWriter) Close() error { return nil }

func NewDefaultSession(io.Writer) (*Session, error) {
	return newSession(rand.Reader, "web", noCloseWriter{os.Stdout}), nil
}
