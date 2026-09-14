package logging

import (
	"encoding/json"
	"errors"
	"reflect"
	"strings"
	"testing"

	"github.com/adsouza/africa2ice/pkg/gameapi"
)

type storageGame struct {
	countingGame
	calls   []string
	slot    int
	results []gameapi.StorageResult
}

func (g *storageGame) begin(name string, slot int) (gameapi.StorageOpID, error) {
	g.calls = append(g.calls, name)
	g.slot = slot
	return 42, g.err
}
func (g *storageGame) BeginSave(slot int) (gameapi.StorageOpID, error) { return g.begin("save", slot) }
func (g *storageGame) BeginLoad(slot int) (gameapi.StorageOpID, error) { return g.begin("load", slot) }
func (g *storageGame) BeginDelete(slot int) (gameapi.StorageOpID, error) {
	return g.begin("delete", slot)
}
func (g *storageGame) BeginListSlots() (gameapi.StorageOpID, error) { return g.begin("list_slots", 0) }
func (g *storageGame) PollStorage() []gameapi.StorageResult {
	g.calls = append(g.calls, "poll")
	return g.results
}

func logRecords(t *testing.T, output string) []map[string]any {
	t.Helper()
	var records []map[string]any
	for _, line := range strings.Split(strings.TrimSpace(output), "\n") {
		var record map[string]any
		if err := json.Unmarshal([]byte(line), &record); err != nil {
			t.Fatal(err)
		}
		records = append(records, record)
	}
	return records
}

func TestStorageLoggingPreservesDelegationAndCorrelatesResults(t *testing.T) {
	for _, operation := range []string{"save", "load", "delete", "list_slots"} {
		for _, failure := range []bool{false, true} {
			t.Run(operation+map[bool]string{false: "/success", true: "/failure"}[failure], func(t *testing.T) {
				output := &bufferCloser{}
				session := newSession(strings.NewReader(strings.Repeat("s", 16)), "test", output)
				defer session.Close()
				next := &storageGame{}
				if failure {
					next.err = errors.New("storage unavailable")
				}
				game := DecorateGame(session, next)
				output.Reset()
				var id gameapi.StorageOpID
				var err error
				slot := 7
				switch operation {
				case "save":
					id, err = game.BeginSave(slot)
				case "load":
					id, err = game.BeginLoad(slot)
				case "delete":
					id, err = game.BeginDelete(slot)
				case "list_slots":
					slot = 0
					id, err = game.BeginListSlots()
				}
				if id != 42 || err != next.err || !reflect.DeepEqual(next.calls, []string{operation}) || next.slot != slot || next.snapshots != 0 {
					t.Fatalf("delegation changed: id=%v err=%v next=%#v", id, err, next)
				}
				records := logRecords(t, output.String())
				if len(records) != 2 {
					t.Fatalf("records = %#v", records)
				}
				start, end := records[0], records[1]
				if start["msg"] != "operation.start" || end["msg"] != "operation.end" || start["operation_id"] != end["operation_id"] || start["operation_id"] == nil {
					t.Fatalf("uncorrelated lifecycle: %#v", records)
				}
				if end["operation"] != operation || end["storage_operation_id"] != float64(42) || end["outcome"] != map[bool]string{false: "ok", true: "error"}[failure] {
					t.Fatalf("incorrect result log: %#v", end)
				}
				if operation != "list_slots" && (start["slot"] != float64(slot) || end["slot"] != float64(slot)) {
					t.Fatalf("lost slot: %#v", records)
				}
				if failure && end["error"] != next.err.Error() {
					t.Fatalf("lost error: %#v", end)
				}
			})
		}
	}
}

func TestStorageCompletionLoggingPreservesBatchAndEmptyPoll(t *testing.T) {
	output := &bufferCloser{}
	session := newSession(strings.NewReader(strings.Repeat("p", 16)), "test", output)
	defer session.Close()
	next := &storageGame{results: []gameapi.StorageResult{
		{OperationID: 41, Operation: gameapi.StorageLoad, Slot: 3, ReplacementFrame: &gameapi.Frame{Turn: 8}},
		{OperationID: 42, Operation: gameapi.StorageSave, Slot: 7, Err: errors.New("disk full")},
	}}
	game := DecorateGame(session, next)
	output.Reset()
	if got := game.PollStorage(); !reflect.DeepEqual(got, next.results) {
		t.Fatalf("changed completions: %#v", got)
	}
	records := logRecords(t, output.String())
	if len(records) != 2 {
		t.Fatalf("completion count: %#v", records)
	}
	for i, result := range next.results {
		record := records[i]
		if record["msg"] != "storage.completion" || record["storage_operation_id"] != float64(result.OperationID) || record["slot"] != float64(result.Slot) || record["operation"] != float64(result.Operation) {
			t.Fatalf("completion correlation lost: %#v", record)
		}
	}
	output.Reset()
	next.results = nil
	if game.PollStorage() != nil || output.Len() != 0 || !reflect.DeepEqual(next.calls, []string{"poll", "poll"}) {
		t.Fatal("empty polling added output or changed delegation")
	}
}

func TestGameReadAndCampaignDelegation(t *testing.T) {
	output := &bufferCloser{}
	session := newSession(strings.NewReader(strings.Repeat("c", 16)), "test", output)
	defer session.Close()
	next := &countingGame{frame: gameapi.Frame{Turn: 5, WorldRevision: 9}}
	if DecorateGame(nil, next) != next {
		t.Fatal("nil session should preserve port identity")
	}
	game := DecorateGame(session, next)
	output.Reset()
	frame, err := game.Snapshot()
	hash, hashErr := game.StateHash()
	if err != nil || hashErr != nil || hash != "test-hash" || frame.Turn != 5 || next.snapshots != 1 || output.Len() != 0 {
		t.Fatal("read delegation altered or logged reads")
	}
	for _, failure := range []error{nil, errors.New("campaign unavailable")} {
		next.err = failure
		output.Reset()
		frame, err = game.NewCampaign()
		if err != failure || frame.Turn != 5 {
			t.Fatalf("campaign result changed: %#v %v", frame, err)
		}
		records := logRecords(t, output.String())
		if len(records) != 2 || records[1]["turn"] != float64(5) || records[1]["world_revision"] != float64(9) {
			t.Fatalf("missing campaign metadata: %#v", records)
		}
	}
	if next.newCampaign != 2 || next.snapshots != 1 {
		t.Fatalf("extra calls: %#v", next)
	}
}
