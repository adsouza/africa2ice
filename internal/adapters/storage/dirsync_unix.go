//go:build !windows && !js

package storage

import "os"

// syncDirectory flushes a directory's entries, so a rename into it survives a
// power loss. File.Sync makes the record's bytes durable; only this makes its
// name durable.
func syncDirectory(directory string) error {
	handle, err := os.Open(directory)
	if err != nil {
		return err
	}
	syncErr := handle.Sync()
	closeErr := handle.Close()
	if syncErr != nil {
		return syncErr
	}
	return closeErr
}
