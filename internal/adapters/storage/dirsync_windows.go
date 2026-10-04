//go:build windows

package storage

// syncDirectory is a no-op on Windows, where a directory handle opened
// through the os package cannot be flushed. DESIGN.md §9 asks for the
// directory sync "where the platform supports it".
func syncDirectory(string) error { return nil }
