//go:build js

package storage

import (
	"bytes"
	"testing"
	"time"

	"github.com/adsouza/africa2ice/internal/application"
)

func TestIndexedDBRepositoryBrowserContract(t *testing.T) {
	repository := NewIndexedDBRepository()
	defer func() { _ = repository.Close() }()
	service, err := application.NewGameService(42)
	if err != nil {
		t.Fatal(err)
	}
	state, err := service.ExportSaveState()
	if err != nil {
		t.Fatal(err)
	}
	if err := repository.BeginWrite(1, application.QuickSave, state); err != nil {
		t.Fatal(err)
	}
	write := waitForCompletion(t, repository, 1)
	if write.Err != nil || write.Metadata == nil || write.Metadata.SlotID != application.QuickSave {
		t.Fatalf("write completion = %#v", write)
	}
	if err := repository.BeginList(2); err != nil {
		t.Fatal(err)
	}
	list := waitForCompletion(t, repository, 2)
	if list.Err != nil || len(list.Slots) != 1 || list.Slots[0].SlotID != application.QuickSave {
		t.Fatalf("list completion = %#v", list)
	}
	if err := repository.BeginRead(3, application.QuickSave); err != nil {
		t.Fatal(err)
	}
	read := waitForCompletion(t, repository, 3)
	if read.Err != nil || read.State == nil {
		t.Fatalf("read completion = %#v", read)
	}
	want, err := application.EncodeSaveState(state)
	if err != nil {
		t.Fatal(err)
	}
	got, err := application.EncodeSaveState(*read.State)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(got, want) {
		t.Fatal("IndexedDB round trip changed the save payload")
	}
	if err := repository.BeginDelete(4, application.QuickSave); err != nil {
		t.Fatal(err)
	}
	deleted := waitForCompletion(t, repository, 4)
	if deleted.Err != nil || deleted.Metadata == nil || !deleted.Metadata.Deleted {
		t.Fatalf("delete completion = %#v", deleted)
	}
	if err := repository.BeginRead(5, application.QuickSave); err != nil {
		t.Fatal(err)
	}
	missing := waitForCompletion(t, repository, 5)
	if missing.Err == nil || missing.State != nil {
		t.Fatalf("deleted slot read = %#v", missing)
	}
}

func waitForCompletion(t *testing.T, repository *IndexedDBRepository, operation application.RepositoryOpID) application.RepositoryCompletion {
	t.Helper()
	deadline := time.Now().Add(5 * time.Second)
	for time.Now().Before(deadline) {
		for _, completion := range repository.Poll() {
			if completion.OperationID == operation {
				return completion
			}
		}
		time.Sleep(10 * time.Millisecond)
	}
	t.Fatalf("operation %d timed out", operation)
	return application.RepositoryCompletion{}
}
