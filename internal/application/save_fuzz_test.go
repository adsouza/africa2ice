package application

import (
	"bytes"
	"compress/gzip"
	"encoding/json"
	"io"
	"os"
	"path/filepath"
	"testing"

	"github.com/adsouza/africa2ice/pkg/gameapi"
)

// Save files are user-reachable bytes: a hand-edited file or one written by a
// buggy build passes the storage checksum yet may be malformed. Loading is
// DecodeSaveState then RestoreWorld; neither may panic on any input, and a
// save that restores must reach a fixed point after one export, so saving and
// reloading never drifts.
func FuzzLoadSave(f *testing.F) {
	for _, seed := range saveFuzzSeeds(f) {
		f.Add(seed)
	}
	f.Fuzz(func(t *testing.T, data []byte) {
		save, err := DecodeSaveState(data)
		if err != nil {
			return
		}
		world, err := save.RestoreWorld()
		if err != nil {
			return
		}
		first, err := SaveStateFromWorld(world, save.WorldRevision)
		if err != nil {
			t.Fatalf("restored world does not export: %v", err)
		}
		encoded, err := EncodeSaveState(first)
		if err != nil {
			t.Fatalf("exported save does not encode: %v", err)
		}
		decoded, err := DecodeSaveState(encoded)
		if err != nil {
			t.Fatalf("exported save does not decode: %v", err)
		}
		reloaded, err := decoded.RestoreWorld()
		if err != nil {
			t.Fatalf("exported save does not restore: %v", err)
		}
		second, err := SaveStateFromWorld(reloaded, decoded.WorldRevision)
		if err != nil {
			t.Fatalf("reloaded world does not export: %v", err)
		}
		again, err := EncodeSaveState(second)
		if err != nil {
			t.Fatalf("re-exported save does not encode: %v", err)
		}
		if !bytes.Equal(encoded, again) {
			t.Fatal("save drifted across one export, decode, restore, export cycle")
		}
	})
}

func saveFuzzSeeds(tb testing.TB) [][]byte {
	tb.Helper()
	service, err := NewGameService(1)
	if err != nil {
		tb.Fatal(err)
	}
	encode := func() []byte {
		state, err := service.ExportSaveState()
		if err != nil {
			tb.Fatal(err)
		}
		data, err := EncodeSaveState(state)
		if err != nil {
			tb.Fatal(err)
		}
		return data
	}
	seeds := [][]byte{encode()}
	frame, err := service.Snapshot()
	if err != nil {
		tb.Fatal(err)
	}
	for _, band := range frame.Bands {
		if band.Species == gameapi.HomoSapiens && len(band.MigrationCandidates) > 0 {
			if _, err := service.Apply(gameapi.QueueMigration{BandID: band.ID, TileID: band.MigrationCandidates[0].TileID}); err != nil {
				tb.Fatal(err)
			}
			break
		}
	}
	seeds = append(seeds, encode()) // a queued order mid-planning
	for range 3 {
		if _, err := service.EndTurn(); err != nil {
			tb.Fatal(err)
		}
	}
	seeds = append(seeds, encode())
	compressed, err := os.ReadFile(filepath.Join("..", "adapters", "storage", "testdata", "oldest_supported_save_v1.json.gz"))
	if err != nil {
		tb.Fatal(err)
	}
	reader, err := gzip.NewReader(bytes.NewReader(compressed))
	if err != nil {
		tb.Fatal(err)
	}
	oldest, err := io.ReadAll(reader)
	if err != nil {
		tb.Fatal(err)
	}
	var fixture struct {
		World json.RawMessage `json:"world"`
	}
	if json.Unmarshal(oldest, &fixture) == nil && len(fixture.World) > 0 {
		oldest = fixture.World
	}
	return append(seeds, oldest, []byte(`{}`), []byte(`null`))
}

// FuzzLoadSave's invariant runs only for inputs that restore, so its seed
// corpus must contain real saves; otherwise every plain `go test` run would
// exercise nothing past the decoder.
func TestLoadSaveFuzzSeedsReachRestore(t *testing.T) {
	seeds := saveFuzzSeeds(t)
	restored := 0
	for _, seed := range seeds {
		if save, err := DecodeSaveState(seed); err == nil {
			if _, err := save.RestoreWorld(); err == nil {
				restored++
			}
		}
	}
	if restored != 4 {
		t.Fatalf("%d of %d seeds restore, want the 4 real saves (new, queued, advanced, oldest supported)", restored, len(seeds))
	}
}
