package storage

import (
	"bytes"
	"compress/gzip"
	_ "embed"
	"io"
	"testing"

	"github.com/adsouza/africa2ice/internal/application"
)

//go:embed testdata/oldest_supported_save_v1.json.gz
var oldestSupportedSaveGzip []byte

func oldestSupportedSave(t *testing.T) application.SaveState {
	t.Helper()
	reader, err := gzip.NewReader(bytes.NewReader(oldestSupportedSaveGzip))
	if err != nil {
		t.Fatal(err)
	}
	payload, err := io.ReadAll(reader)
	if err != nil {
		t.Fatal(err)
	}
	if err := reader.Close(); err != nil {
		t.Fatal(err)
	}
	state, err := application.DecodeSaveState(payload)
	if err != nil {
		t.Fatal(err)
	}
	if state.SchemaVersion != 1 {
		t.Fatalf("oldest supported fixture schema = %d, want 1", state.SchemaVersion)
	}
	if state.WorldSeed != 0x9e3779b97f4a7c15 || state.Turn != 12 {
		t.Fatalf("oldest supported fixture identity = seed %#x turn %d", state.WorldSeed, state.Turn)
	}
	return state
}
