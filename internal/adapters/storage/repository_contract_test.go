package storage

import (
	"bytes"
	"errors"
	"os"
	"testing"
	"time"

	"github.com/adsouza/africa2ice/internal/application"
)

// This file has no build tag: the desktop tests run the contract against
// FileRepository natively, and tools/run_wasm_go_tests.mjs runs it against
// IndexedDBRepository in Chromium. DESIGN.md step 7 asks for one shared
// repository contract suite rather than two adapter test files that each
// check different things.

// repositoryBackend opens one backend. fresh returns a writable repository
// over empty storage; reopen returns a new writable repository over the
// storage the last one used, after that one is closed.
type repositoryBackend struct {
	fresh  func(t *testing.T) application.CampaignRepository
	reopen func(t *testing.T) application.CampaignRepository
}

func contractState(t *testing.T, turn int) application.SaveState {
	t.Helper()
	service, err := application.NewGameService(42)
	if err != nil {
		t.Fatal(err)
	}
	state, err := service.ExportSaveState()
	if err != nil {
		t.Fatal(err)
	}
	state.Turn = turn
	return state
}

// awaitCompletions polls until count completions arrive and returns them in
// the order the repository reported them.
func awaitCompletions(t *testing.T, repository application.CampaignRepository, count int) []application.RepositoryCompletion {
	t.Helper()
	var got []application.RepositoryCompletion
	deadline := time.Now().Add(10 * time.Second)
	for len(got) < count && time.Now().Before(deadline) {
		got = append(got, repository.Poll()...)
		if len(got) < count {
			time.Sleep(2 * time.Millisecond)
		}
	}
	if len(got) != count {
		t.Fatalf("received %d completions, want %d", len(got), count)
	}
	return got
}

func awaitOne(t *testing.T, repository application.CampaignRepository) application.RepositoryCompletion {
	t.Helper()
	return awaitCompletions(t, repository, 1)[0]
}

func mustWrite(t *testing.T, repository application.CampaignRepository, op application.RepositoryOpID, slot application.SlotID, state application.SaveState) application.SaveMetadata {
	t.Helper()
	if err := repository.BeginWrite(op, slot, state); err != nil {
		t.Fatal(err)
	}
	completion := awaitOne(t, repository)
	if completion.OperationID != op || completion.Err != nil || completion.Metadata == nil {
		t.Fatalf("write %d completion = %+v", op, completion)
	}
	return *completion.Metadata
}

func mustDelete(t *testing.T, repository application.CampaignRepository, op application.RepositoryOpID, slot application.SlotID) application.SaveMetadata {
	t.Helper()
	if err := repository.BeginDelete(op, slot); err != nil {
		t.Fatal(err)
	}
	completion := awaitOne(t, repository)
	if completion.OperationID != op || completion.Err != nil || completion.Metadata == nil || !completion.Metadata.Deleted {
		t.Fatalf("delete %d completion = %+v", op, completion)
	}
	return *completion.Metadata
}

func readSlot(t *testing.T, repository application.CampaignRepository, op application.RepositoryOpID, slot application.SlotID) application.RepositoryCompletion {
	t.Helper()
	if err := repository.BeginRead(op, slot); err != nil {
		t.Fatal(err)
	}
	completion := awaitOne(t, repository)
	if completion.OperationID != op {
		t.Fatalf("read %d completed as op %d", op, completion.OperationID)
	}
	return completion
}

func listSlots(t *testing.T, repository application.CampaignRepository, op application.RepositoryOpID) map[application.SlotID]application.SaveMetadata {
	t.Helper()
	if err := repository.BeginList(op); err != nil {
		t.Fatal(err)
	}
	completion := awaitOne(t, repository)
	if completion.OperationID != op || completion.Err != nil {
		t.Fatalf("list %d completion = %+v", op, completion)
	}
	slots := map[application.SlotID]application.SaveMetadata{}
	for _, metadata := range completion.Slots {
		if _, duplicate := slots[metadata.SlotID]; duplicate {
			t.Fatalf("list reported slot %d twice", metadata.SlotID)
		}
		slots[metadata.SlotID] = metadata
	}
	return slots
}

func encoded(t *testing.T, state application.SaveState) []byte {
	t.Helper()
	data, err := application.EncodeSaveState(state)
	if err != nil {
		t.Fatal(err)
	}
	return data
}

func runRepositoryContract(t *testing.T, backend repositoryBackend) {
	t.Run("empty slots read as empty and list nothing", func(t *testing.T) {
		repository := backend.fresh(t)
		defer func() { _ = repository.Close() }()
		read := readSlot(t, repository, 1, application.Manual1)
		if !errors.Is(read.Err, os.ErrNotExist) || read.State != nil {
			t.Fatalf("never-written slot read = %+v, want an os.ErrNotExist-wrapping error and no state", read)
		}
		if slots := listSlots(t, repository, 2); len(slots) != 0 {
			t.Fatalf("fresh storage lists %v", slots)
		}
	})

	t.Run("a write reads back byte for byte and lists", func(t *testing.T) {
		repository := backend.fresh(t)
		defer func() { _ = repository.Close() }()
		state := contractState(t, 5)
		metadata := mustWrite(t, repository, 1, application.Manual2, state)
		if metadata.SlotID != application.Manual2 || metadata.Turn != 5 || metadata.Generation == "" || metadata.CommitSequence == 0 {
			t.Fatalf("write metadata = %+v", metadata)
		}
		read := readSlot(t, repository, 2, application.Manual2)
		if read.Err != nil || read.State == nil {
			t.Fatalf("read = %+v", read)
		}
		if !bytes.Equal(encoded(t, *read.State), encoded(t, state)) {
			t.Fatal("read state does not re-encode to the written bytes")
		}
		if read.Metadata == nil || read.Metadata.CommitSequence != metadata.CommitSequence || read.Metadata.Generation != metadata.Generation {
			t.Fatalf("read metadata = %+v, want the committed %+v", read.Metadata, metadata)
		}
		if listed := listSlots(t, repository, 3); len(listed) != 1 || listed[application.Manual2].CommitSequence != metadata.CommitSequence {
			t.Fatalf("list = %v", listed)
		}
	})

	t.Run("commit sequences rise globally and the newest commit wins", func(t *testing.T) {
		repository := backend.fresh(t)
		defer func() { _ = repository.Close() }()
		first := mustWrite(t, repository, 1, application.Auto1, contractState(t, 1))
		second := mustWrite(t, repository, 2, application.Auto2, contractState(t, 2))
		third := mustWrite(t, repository, 3, application.Auto1, contractState(t, 3))
		deleted := mustDelete(t, repository, 4, application.Auto2)
		sequences := []uint64{first.CommitSequence, second.CommitSequence, third.CommitSequence, deleted.CommitSequence}
		for index := 1; index < len(sequences); index++ {
			if sequences[index] <= sequences[index-1] {
				t.Fatalf("commit sequences %v do not rise across slots, overwrites, and deletes", sequences)
			}
		}
		if read := readSlot(t, repository, 5, application.Auto1); read.Err != nil || read.State.Turn != 3 {
			t.Fatalf("overwritten slot read = %+v, want turn 3", read)
		}
		if read := readSlot(t, repository, 6, application.Auto2); !errors.Is(read.Err, os.ErrNotExist) {
			t.Fatalf("deleted slot read = %+v, want empty", read)
		}
		listed := listSlots(t, repository, 7)
		if _, present := listed[application.Auto2]; present || len(listed) != 1 || listed[application.Auto1].Turn != 3 {
			t.Fatalf("list after overwrite and delete = %v", listed)
		}
		again := mustWrite(t, repository, 8, application.Auto2, contractState(t, 4))
		if again.CommitSequence <= deleted.CommitSequence {
			t.Fatal("a write after a delete reused an earlier sequence")
		}
	})

	t.Run("queued operations complete in submission order", func(t *testing.T) {
		repository := backend.fresh(t)
		defer func() { _ = repository.Close() }()
		if err := repository.BeginWrite(1, application.QuickSave, contractState(t, 9)); err != nil {
			t.Fatal(err)
		}
		if err := repository.BeginRead(2, application.QuickSave); err != nil {
			t.Fatal(err)
		}
		completions := awaitCompletions(t, repository, 2)
		if completions[0].OperationID != 1 || completions[1].OperationID != 2 {
			t.Fatalf("completion order = %d, %d; want 1, 2", completions[0].OperationID, completions[1].OperationID)
		}
		if read := completions[1]; read.Err != nil || read.State == nil || read.State.Turn != 9 {
			t.Fatalf("read queued behind a write = %+v, want the write's state", read)
		}
	})

	t.Run("invalid slots are refused at submission", func(t *testing.T) {
		repository := backend.fresh(t)
		defer func() { _ = repository.Close() }()
		const invalid = application.SlotID(4)
		if application.ValidSlot(invalid) {
			t.Fatal("fixture slot is valid")
		}
		if repository.BeginWrite(1, invalid, contractState(t, 1)) == nil || repository.BeginRead(2, invalid) == nil || repository.BeginDelete(3, invalid) == nil {
			t.Fatal("an invalid slot was accepted")
		}
		if completions := repository.Poll(); len(completions) != 0 {
			t.Fatalf("refused operations produced completions %v", completions)
		}
	})

	t.Run("committed saves and deletions survive a reopen", func(t *testing.T) {
		repository := backend.fresh(t)
		kept := mustWrite(t, repository, 1, application.Manual3, contractState(t, 12))
		mustWrite(t, repository, 2, application.Manual1, contractState(t, 13))
		removed := mustDelete(t, repository, 3, application.Manual1)
		_ = repository.Close()

		reopened := backend.reopen(t)
		defer func() { _ = reopened.Close() }()
		if read := readSlot(t, reopened, 4, application.Manual3); read.Err != nil || read.State.Turn != 12 || read.Metadata.CommitSequence != kept.CommitSequence {
			t.Fatalf("reopened save = %+v", read)
		}
		if read := readSlot(t, reopened, 5, application.Manual1); !errors.Is(read.Err, os.ErrNotExist) {
			t.Fatalf("reopened deletion = %+v, want empty", read)
		}
		next := mustWrite(t, reopened, 6, application.Manual2, contractState(t, 14))
		if next.CommitSequence <= removed.CommitSequence {
			t.Fatalf("reopened storage issued sequence %d, not above %d", next.CommitSequence, removed.CommitSequence)
		}
	})
}
