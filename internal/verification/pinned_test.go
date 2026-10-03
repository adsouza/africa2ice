package verification

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"sort"
	"testing"
)

const (
	pinnedCheckpoints = "testdata/reference_checkpoints.json"
	regenerateCommand = "go run . -headless -turns 400 -seed 0x9e3779b97f4a7c15 -policy reference -checkpoint-json internal/verification/testdata/reference_checkpoints.json"
)

// The other gates ask whether the campaign stays winnable; this one asks
// whether it changed at all. A model edit that moves outcomes — a split rule,
// a capacity formula — passes every viability margin, so the reference run is
// pinned byte for byte and any change must arrive with a regenerated fixture
// whose diff a reviewer reads. CI's cross-target job already requires these
// bytes to match on Linux, macOS, Windows, and Chromium.
func TestReferenceCampaignMatchesPinnedCheckpoints(t *testing.T) {
	records, err := referenceRecords()
	if err != nil {
		t.Fatal(err)
	}
	got, err := CanonicalJSON(records)
	if err != nil {
		t.Fatal(err)
	}
	got = append(got, '\n') // the CLI's -checkpoint-json writes a trailing newline
	want, err := os.ReadFile(filepath.FromSlash(pinnedCheckpoints))
	if err != nil {
		t.Fatalf("read pinned checkpoints: %v\nCreate them with:\n  %s", err, regenerateCommand)
	}
	if bytes.Equal(got, want) {
		return
	}
	t.Fatalf("reference campaign no longer matches %s: %s\nIf the change is intended, regenerate and review the diff:\n  %s",
		pinnedCheckpoints, firstDifference(got, want), regenerateCommand)
}

// firstDifference names the first checkpoint field that differs, so a failure
// says what moved rather than only that the bytes changed.
func firstDifference(got, want []byte) string {
	var gotRecords, wantRecords []map[string]any
	if json.Unmarshal(got, &gotRecords) != nil || json.Unmarshal(want, &wantRecords) != nil {
		return "the files differ and at least one is not a checkpoint array"
	}
	if len(gotRecords) != len(wantRecords) {
		return "checkpoint count changed"
	}
	for index := range gotRecords {
		for _, key := range canonicalKeys(gotRecords[index], wantRecords[index]) {
			gotValue, _ := json.Marshal(gotRecords[index][key])
			wantValue, _ := json.Marshal(wantRecords[index][key])
			if !bytes.Equal(gotValue, wantValue) {
				return "turn " + string(mustJSON(gotRecords[index]["turn"])) + " " + key + " is " + string(gotValue) + ", pinned " + string(wantValue)
			}
		}
	}
	return "only formatting differs"
}

func canonicalKeys(records ...map[string]any) []string {
	seen := map[string]bool{}
	var keys []string
	for _, record := range records {
		for key := range record {
			if !seen[key] {
				seen[key] = true
				keys = append(keys, key)
			}
		}
	}
	sort.Strings(keys)
	return keys
}

func mustJSON(value any) []byte {
	encoded, _ := json.Marshal(value)
	return encoded
}
