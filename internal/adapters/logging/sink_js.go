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
	return newWebSession(os.Stdout), nil
}

// newWebSession is the browser session over out. Production writes to
// standard output, which wasm_exec.js forwards to console.log; tests inject a
// buffer. The web path writes nowhere else: no file, IndexedDB record,
// download, or network request.
func newWebSession(out io.Writer) *Session {
	return newSession(rand.Reader, "web", noCloseWriter{out})
}
