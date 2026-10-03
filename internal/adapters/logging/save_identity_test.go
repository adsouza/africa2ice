package logging

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"strings"
	"testing"

	"github.com/adsouza/africa2ice/internal/application"
)

// A record that involves a save names the save's format, never its contents:
// the schema version and a short digest of its algorithm identifiers. Every
// write in a session carries this build's digest, so a load whose digest
// differs is visibly an older or foreign save.
func TestRepositoryRecordsIdentifySavesWithoutTheirPayload(t *testing.T) {
	versions := application.AlgorithmVersions{RNGAlgorithm: "pcg-splitmix-v1", ClimateAlgorithm: "climate-v9"}
	encoded, err := json.Marshal(versions)
	if err != nil {
		t.Fatal(err)
	}
	sum := sha256.Sum256(encoded)
	wantDigest := hex.EncodeToString(sum[:])[:12]
	state := application.SaveState{SchemaVersion: 2, AlgorithmVersions: versions, WorldSeed: 9}

	output := &bufferCloser{}
	session := newSession(strings.NewReader(strings.Repeat("s", 16)), "test", output)
	next := &countingRepository{completions: []application.RepositoryCompletion{{OperationID: 8, Operation: application.RepositoryRead, SlotID: application.QuickSave, State: &state}}}
	decorated := DecorateCampaignRepository(session, next)
	if err := decorated.BeginWrite(7, application.QuickSave, state); err != nil {
		t.Fatal(err)
	}
	decorated.Poll()
	_ = session.Close()

	identified := map[string]bool{}
	for _, record := range sessionMessages(t, output.String()) {
		if _, leaked := record["world_seed"]; leaked {
			t.Fatalf("save payload leaked: %v", record)
		}
		key, _ := record["msg"].(string)
		if operation, _ := record["operation"].(string); key != "repository.completion" {
			key += ":" + operation
		}
		if record["schema_version"] == float64(2) && record["algorithm_versions"] == wantDigest {
			identified[key] = true
		}
	}
	for _, want := range []string{"operation.start:write", "operation.end:write", "repository.completion"} {
		if !identified[want] {
			t.Fatalf("%s record does not identify the save as schema 2 / %s:\n%s", want, wantDigest, output.String())
		}
	}
}
