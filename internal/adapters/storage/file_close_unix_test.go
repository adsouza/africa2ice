//go:build !js && !windows

package storage

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"os"
	"path/filepath"
	"syscall"
	"testing"
	"time"

	"github.com/adsouza/africa2ice/internal/application"
)

func TestFileRepositoryCloseKeepsLeaseUntilActiveWriteExits(t *testing.T) {
	directory := t.TempDir()
	repository, err := NewFileRepository(directory)
	if err != nil {
		t.Fatal(err)
	}
	service, _ := application.NewGameService(91)
	state, _ := service.ExportSaveState()
	payload, err := application.EncodeSaveState(state)
	if err != nil {
		t.Fatal(err)
	}
	hash := sha256.Sum256(payload)
	worldPath := filepath.Join(directory, fmt.Sprintf("slot_%d_world_%s.json", application.Manual1, hex.EncodeToString(hash[:])))
	if err := syscall.Mkfifo(worldPath, 0o600); err != nil {
		t.Fatal(err)
	}
	if err := repository.BeginWrite(1, application.Manual1, state); err != nil {
		t.Fatal(err)
	}

	// Opening the writer proves the repository worker has opened the FIFO for
	// reading and is now blocked in its active write until this handle closes.
	writerReady := make(chan *os.File, 1)
	writerErrors := make(chan error, 1)
	go func() {
		writer, openErr := os.OpenFile(worldPath, os.O_WRONLY, 0)
		if openErr != nil {
			writerErrors <- openErr
			return
		}
		writerReady <- writer
	}()
	var writer *os.File
	select {
	case writer = <-writerReady:
	case openErr := <-writerErrors:
		t.Fatal(openErr)
	case <-time.After(3 * time.Second):
		t.Fatal("worker did not enter active write")
	}

	closed := make(chan error, 1)
	go func() { closed <- repository.Close() }()
	select {
	case closeErr := <-closed:
		t.Fatalf("Close returned while write was active: %v", closeErr)
	case <-time.After(50 * time.Millisecond):
	}

	competitor, err := NewFileRepository(directory)
	if err != nil {
		t.Fatal(err)
	}
	if competitor.Writable() {
		t.Fatal("competitor acquired lease while closing repository still had an active write")
	}
	if err := competitor.Close(); err != nil {
		t.Fatal(err)
	}
	if err := writer.Close(); err != nil {
		t.Fatal(err)
	}
	select {
	case closeErr := <-closed:
		if closeErr != nil {
			t.Fatal(closeErr)
		}
	case <-time.After(3 * time.Second):
		t.Fatal("Close did not return after active write exited")
	}

	after, err := NewFileRepository(directory)
	if err != nil {
		t.Fatal(err)
	}
	if !after.Writable() {
		t.Fatal("lease was not released after worker exit")
	}
	if err := after.Close(); err != nil {
		t.Fatal(err)
	}
}
