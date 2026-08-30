//go:build !windows && !js

package storage

import (
	"errors"
	"os"
	"syscall"
)

type repositoryLease interface{ Close() error }

type unixLease struct{ file *os.File }

func acquireRepositoryLease(path string) (repositoryLease, bool, error) {
	file, err := os.OpenFile(path, os.O_CREATE|os.O_RDWR, 0o600)
	if err != nil {
		return nil, false, err
	}
	if err := syscall.Flock(int(file.Fd()), syscall.LOCK_EX|syscall.LOCK_NB); err != nil {
		_ = file.Close()
		if errors.Is(err, syscall.EWOULDBLOCK) || errors.Is(err, syscall.EAGAIN) {
			return nil, false, nil
		}
		return nil, false, err
	}
	return &unixLease{file: file}, true, nil
}

func (lease *unixLease) Close() error {
	unlockErr := syscall.Flock(int(lease.file.Fd()), syscall.LOCK_UN)
	closeErr := lease.file.Close()
	if unlockErr != nil {
		return unlockErr
	}
	return closeErr
}
