//go:build !js

package storage

import (
	"os"
	"path/filepath"
)

// fileSystem is the desktop commit protocol's view of the disk. Production
// goes straight to the operating system; failure-injection tests substitute
// one that stops after any step, the way a crash or power loss would.
type fileSystem interface {
	ReadFile(path string) ([]byte, error)
	Glob(pattern string) ([]string, error)
	CreateTemp(directory, pattern string) (temporaryFile, error)
	Rename(from, to string) error
	Remove(path string) error
	// SyncDir makes completed renames in directory durable.
	SyncDir(directory string) error
}

// temporaryFile is the subset of *os.File the protocol writes through.
type temporaryFile interface {
	Name() string
	Write(data []byte) (int, error)
	Sync() error
	Close() error
}

type osFileSystem struct{}

func (osFileSystem) ReadFile(path string) ([]byte, error)  { return os.ReadFile(path) }
func (osFileSystem) Glob(pattern string) ([]string, error) { return filepath.Glob(pattern) }
func (osFileSystem) Rename(from, to string) error          { return os.Rename(from, to) }
func (osFileSystem) Remove(path string) error              { return os.Remove(path) }
func (osFileSystem) SyncDir(directory string) error        { return syncDirectory(directory) }
func (osFileSystem) CreateTemp(directory, pattern string) (temporaryFile, error) {
	return os.CreateTemp(directory, pattern)
}
