//go:build js

package ui

import (
	"testing"
	"time"
)

func TestIndexedDBUISettingsStoreBrowserContract(t *testing.T) {
	store, err := NewIndexedDBUISettingsStore()
	if err != nil {
		t.Fatal(err)
	}
	want := UISettings{SchemaVersion: 1, FieldNotesVisible: false, MasterVolume: 0.65, Muted: true}
	if err := store.BeginWrite(1, want); err != nil {
		t.Fatal(err)
	}
	write := waitForIndexedDBSettings(t, store)
	if write.Err != nil || write.Operation != UISettingsWrite || write.Settings != want {
		t.Fatalf("write completion = %#v", write)
	}
	if err := store.BeginRead(2); err != nil {
		t.Fatal(err)
	}
	read := waitForIndexedDBSettings(t, store)
	if read.Err != nil || read.Operation != UISettingsRead || read.Revision != 2 || read.Settings != want {
		t.Fatalf("read completion = %#v, want %#v", read, want)
	}
}

func waitForIndexedDBSettings(t *testing.T, store UISettingsStore) UISettingsCompletion {
	t.Helper()
	deadline := time.Now().Add(5 * time.Second)
	for time.Now().Before(deadline) {
		if completions := store.Poll(); len(completions) != 0 {
			return completions[0]
		}
		time.Sleep(10 * time.Millisecond)
	}
	t.Fatal("timed out waiting for IndexedDB UI settings")
	return UISettingsCompletion{}
}
