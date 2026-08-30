//go:build !js

package storage

import (
	"os"
	"testing"
	"time"

	"github.com/adsouza/africa2ice/internal/application"
)

func waitCompletion(t *testing.T, repository *FileRepository) application.RepositoryCompletion {
	t.Helper()
	deadline := time.Now().Add(3 * time.Second)
	for time.Now().Before(deadline) {
		if completions := repository.Poll(); len(completions) != 0 {
			return completions[0]
		}
		time.Sleep(time.Millisecond)
	}
	t.Fatal("repository completion timed out")
	return application.RepositoryCompletion{}
}

func TestFileRepositoryWriteReadListDelete(t *testing.T) {
	directory := t.TempDir()
	repository, err := NewFileRepository(directory)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = repository.Close() }()
	service, _ := application.NewGameService(9)
	state, _ := service.ExportSaveState()
	if err := repository.BeginWrite(1, application.Manual1, state); err != nil {
		t.Fatal(err)
	}
	written := waitCompletion(t, repository)
	if written.Err != nil || written.Metadata == nil || written.Metadata.CommitSequence != 1 {
		t.Fatalf("write = %#v", written)
	}
	if err := repository.BeginRead(2, application.Manual1); err != nil {
		t.Fatal(err)
	}
	read := waitCompletion(t, repository)
	if read.Err != nil || read.State == nil || read.State.WorldSeed != 9 {
		t.Fatalf("read = %#v", read)
	}
	if err := repository.BeginList(3); err != nil {
		t.Fatal(err)
	}
	listed := waitCompletion(t, repository)
	if listed.Err != nil || len(listed.Slots) != 1 || listed.Slots[0].SlotID != application.Manual1 {
		t.Fatalf("list = %#v", listed)
	}
	if err := repository.BeginDelete(4, application.Manual1); err != nil {
		t.Fatal(err)
	}
	deleted := waitCompletion(t, repository)
	if deleted.Err != nil || deleted.Metadata == nil || !deleted.Metadata.Deleted || deleted.Metadata.CommitSequence != 2 {
		t.Fatalf("delete = %#v", deleted)
	}
	if err := repository.BeginRead(5, application.Manual1); err != nil {
		t.Fatal(err)
	}
	if readDeleted := waitCompletion(t, repository); !os.IsNotExist(readDeleted.Err) {
		t.Fatalf("deleted read = %#v", readDeleted)
	}
}

func TestFileRepositoryOverwriteKeepsImmutableGenerations(t *testing.T) {
	repository, _ := NewFileRepository(t.TempDir())
	defer func() { _ = repository.Close() }()
	firstService, _ := application.NewGameService(1)
	first, _ := firstService.ExportSaveState()
	secondService, _ := application.NewGameService(2)
	second, _ := secondService.ExportSaveState()
	_ = repository.BeginWrite(1, application.QuickSave, first)
	firstCompletion := waitCompletion(t, repository)
	_ = repository.BeginWrite(2, application.QuickSave, second)
	secondCompletion := waitCompletion(t, repository)
	if firstCompletion.Metadata.Generation == secondCompletion.Metadata.Generation || secondCompletion.Metadata.CommitSequence != 2 {
		t.Fatalf("overwrite metadata = %#v / %#v", firstCompletion.Metadata, secondCompletion.Metadata)
	}
	_ = repository.BeginRead(3, application.QuickSave)
	loaded := waitCompletion(t, repository)
	if loaded.State == nil || loaded.State.WorldSeed != 2 {
		t.Fatalf("latest generation not loaded: %#v", loaded)
	}
}

func TestFileRepositorySecondProcessViewIsReadOnly(t *testing.T) {
	directory := t.TempDir()
	first, err := NewFileRepository(directory)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = first.Close() }()
	second, err := NewFileRepository(directory)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = second.Close() }()
	if !first.Writable() || second.Writable() {
		t.Fatalf("writable capabilities = %t/%t", first.Writable(), second.Writable())
	}
	service, _ := application.NewGameService(1)
	state, _ := service.ExportSaveState()
	if err := second.BeginWrite(1, application.Manual1, state); err == nil {
		t.Fatal("read-only repository accepted write")
	}
}
