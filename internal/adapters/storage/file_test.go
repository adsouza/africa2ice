//go:build !js

package storage

import (
	"os"
	"strings"
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

// Records are immutable while they are reachable. Once a newer commit
// supersedes a generation, its payload is reclaimed: a campaign autosaves every
// turn, and keeping every turn's world forever would grow the save directory
// without bound.
func TestFileRepositoryOverwriteCommitsANewGeneration(t *testing.T) {
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

func slotFiles(t *testing.T, directory string) (meta, world int) {
	t.Helper()
	entries, err := os.ReadDir(directory)
	if err != nil {
		t.Fatal(err)
	}
	for _, entry := range entries {
		switch {
		case strings.Contains(entry.Name(), "_meta_"):
			meta++
		case strings.Contains(entry.Name(), "_world_"):
			world++
		}
	}
	return meta, world
}

// Autosave writes one world payload per turn. Superseded records must be
// reclaimed, and the commit sequence must come from memory rather than a rescan
// of every metadata file, or the cost of a save grows with the campaign.
func TestFileRepositoryReclaimsSupersededRecords(t *testing.T) {
	directory := t.TempDir()
	repository, err := NewFileRepository(directory)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = repository.Close() }()
	for turn := range 6 {
		service, _ := application.NewGameService(uint64(turn) + 1)
		state, _ := service.ExportSaveState()
		if err := repository.BeginWrite(application.RepositoryOpID(turn+1), application.Auto1, state); err != nil {
			t.Fatal(err)
		}
		completion := waitCompletion(t, repository)
		if completion.Err != nil {
			t.Fatalf("turn %d write = %v", turn, completion.Err)
		}
		if got := completion.Metadata.CommitSequence; got != uint64(turn+1) {
			t.Fatalf("turn %d commit sequence = %d, want %d", turn, got, turn+1)
		}
	}
	if meta, world := slotFiles(t, directory); meta != 1 || world != 1 {
		t.Fatalf("after six saves: %d metadata and %d world files, want 1 and 1", meta, world)
	}

	// The newest save is still readable after everything before it went away.
	if err := repository.BeginRead(100, application.Auto1); err != nil {
		t.Fatal(err)
	}
	read := waitCompletion(t, repository)
	if read.Err != nil || read.State == nil || read.State.WorldSeed != 6 {
		t.Fatalf("newest generation not readable after pruning: %#v", read)
	}

	// A delete marker names no generation, so the payload goes too.
	if err := repository.BeginDelete(101, application.Auto1); err != nil {
		t.Fatal(err)
	}
	if deleted := waitCompletion(t, repository); deleted.Err != nil {
		t.Fatal(deleted.Err)
	}
	if meta, world := slotFiles(t, directory); meta != 1 || world != 0 {
		t.Fatalf("after delete: %d metadata and %d world files, want 1 and 0", meta, world)
	}
}

// A fresh process must not reissue a commit sequence an earlier one already
// used, or autosave rotation would compare stale numbers across slots.
func TestFileRepositoryResumesCommitSequenceAcrossProcesses(t *testing.T) {
	directory := t.TempDir()
	first, err := NewFileRepository(directory)
	if err != nil {
		t.Fatal(err)
	}
	service, _ := application.NewGameService(4)
	state, _ := service.ExportSaveState()
	for index := range 3 {
		if err := first.BeginWrite(application.RepositoryOpID(index+1), application.SlotID(int(application.Auto1)+index), state); err != nil {
			t.Fatal(err)
		}
		if completion := waitCompletion(t, first); completion.Err != nil {
			t.Fatal(completion.Err)
		}
	}
	if err := first.Close(); err != nil {
		t.Fatal(err)
	}

	second, err := NewFileRepository(directory)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = second.Close() }()
	if err := second.BeginWrite(9, application.Manual1, state); err != nil {
		t.Fatal(err)
	}
	resumed := waitCompletion(t, second)
	if resumed.Err != nil || resumed.Metadata.CommitSequence != 4 {
		t.Fatalf("resumed commit sequence = %#v, want 4", resumed.Metadata)
	}
}
