//go:build windows

package storage

import (
	"errors"
	"os"

	"golang.org/x/sys/windows"
)

type repositoryLease interface{ Close() error }

type windowsLease struct {
	file       *os.File
	overlapped windows.Overlapped
}

func acquireRepositoryLease(path string) (repositoryLease, bool, error) {
	file, err := os.OpenFile(path, os.O_CREATE|os.O_RDWR, 0o600)
	if err != nil {
		return nil, false, err
	}
	lease := &windowsLease{file: file}
	handle := windows.Handle(file.Fd())
	err = windows.LockFileEx(handle, windows.LOCKFILE_EXCLUSIVE_LOCK|windows.LOCKFILE_FAIL_IMMEDIATELY, 0, 1, 0, &lease.overlapped)
	if err != nil {
		_ = file.Close()
		if errors.Is(err, windows.ERROR_LOCK_VIOLATION) {
			return nil, false, nil
		}
		return nil, false, err
	}
	return lease, true, nil
}

func (lease *windowsLease) Close() error {
	unlockErr := windows.UnlockFileEx(windows.Handle(lease.file.Fd()), 0, 1, 0, &lease.overlapped)
	closeErr := lease.file.Close()
	if unlockErr != nil {
		return unlockErr
	}
	return closeErr
}
