//go:build !js

package ui

import (
	"path/filepath"
	"testing"
	"time"
)

func TestFileUISettingsStoreRoundTripsCompleteRecord(t *testing.T) {
	store, err := NewFileUISettingsStore(filepath.Join(t.TempDir(), "nested", "ui_settings.json"))
	if err != nil {
		t.Fatal(err)
	}
	want := UISettings{SchemaVersion: 1, FieldNotesVisible: false, MasterVolume: 0.8, Muted: true}
	if err := store.BeginWrite(3, want); err != nil {
		t.Fatal(err)
	}
	write := waitForUISettingsCompletion(t, store)
	if write.Err != nil || write.Operation != UISettingsWrite || write.Revision != 3 {
		t.Fatalf("write completion = %#v", write)
	}
	if err := store.BeginRead(4); err != nil {
		t.Fatal(err)
	}
	read := waitForUISettingsCompletion(t, store)
	if read.Err != nil || read.Operation != UISettingsRead || read.Revision != 4 || read.Settings != want {
		t.Fatalf("read completion = %#v, want %#v", read, want)
	}
}

func waitForUISettingsCompletion(t *testing.T, store UISettingsStore) UISettingsCompletion {
	t.Helper()
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		if completions := store.Poll(); len(completions) != 0 {
			return completions[0]
		}
		time.Sleep(time.Millisecond)
	}
	t.Fatal("timed out waiting for UI settings completion")
	return UISettingsCompletion{}
}
