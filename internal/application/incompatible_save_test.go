package application

import (
	"errors"
	"strings"
	"testing"

	"github.com/adsouza/africa2ice/pkg/gameapi"
)

func currentSave(t *testing.T) SaveState {
	t.Helper()
	service, err := NewGameService(3)
	if err != nil {
		t.Fatal(err)
	}
	state, err := service.ExportSaveState()
	if err != nil {
		t.Fatal(err)
	}
	return state
}

// A save another build wrote is incompatible, not damaged: the schema outside
// the supported range and any differing algorithm identifier say so, naming
// what differs, while a structurally invalid save stays invalid.
func TestRestoreWorldNamesAnIncompatibleSave(t *testing.T) {
	superseded := currentSave(t)
	superseded.GeographyAlgorithm = "dispersal-map-v5"
	newer := currentSave(t)
	newer.SchemaVersion = SaveSchemaVersion + 1
	older := currentSave(t)
	older.SchemaVersion = OldestSupportedSaveSchemaVersion - 1
	damaged := currentSave(t)
	damaged.GridWidth++
	for _, test := range []struct {
		name         string
		save         SaveState
		incompatible bool
		mentions     string
	}{
		{"superseded algorithm", superseded, true, `GeographyAlgorithm is "dispersal-map-v5"`},
		{"newer schema", newer, true, "schema"},
		{"older schema", older, true, "schema"},
		{"invalid grid", damaged, false, "grid"},
	} {
		_, err := test.save.RestoreWorld()
		if err == nil {
			t.Fatalf("%s: restored", test.name)
		}
		if errors.Is(err, errIncompatibleSave) != test.incompatible || !strings.Contains(err.Error(), test.mentions) {
			t.Fatalf("%s: error %q, want incompatible=%t mentioning %q", test.name, err, test.incompatible, test.mentions)
		}
	}
}

// A newer build's save can carry a field this build has never heard of. The
// strict decode rejects it before RestoreWorld can look at its identity, so
// the decoder must recognise the newer schema itself. An unknown field in a
// save that claims this build's identity is still damage.
func TestDecodeRecognisesANewerBuildsSave(t *testing.T) {
	encode := func(save SaveState) []byte {
		payload, err := EncodeSaveState(save)
		if err != nil {
			t.Fatal(err)
		}
		return append(payload[:len(payload)-1], []byte(`,"field_from_the_future":1}`)...)
	}
	newer := currentSave(t)
	newer.SchemaVersion = SaveSchemaVersion + 1
	if _, err := DecodeSaveState(encode(newer)); !errors.Is(err, errIncompatibleSave) {
		t.Fatalf("newer save with an unknown field: %v, want incompatible", err)
	}
	if _, err := DecodeSaveState(encode(currentSave(t))); err == nil || errors.Is(err, errIncompatibleSave) {
		t.Fatalf("current save with an unknown field: %v, want a plain decode error", err)
	}
	if _, err := DecodeSaveState([]byte(`{"schema_version":`)); err == nil || errors.Is(err, errIncompatibleSave) {
		t.Fatalf("truncated save: %v, want a plain decode error", err)
	}
}

// failingReadRepository fails every read with one error, the way an adapter
// does when its strict decode rejects the stored payload.
type failingReadRepository struct {
	repositoryStub
	err error
}

func (repository *failingReadRepository) BeginRead(op RepositoryOpID, slot SlotID) error {
	repository.completions = append(repository.completions, RepositoryCompletion{OperationID: op, Operation: RepositoryRead, SlotID: slot, Writable: true, Err: repository.err})
	return nil
}

func loadResult(t *testing.T, repository CampaignRepository) gameapi.StorageResult {
	t.Helper()
	service, err := NewGameServiceWithRepository(4, repository)
	if err != nil {
		t.Fatal(err)
	}
	loadID, err := service.BeginLoad(int(Manual1))
	if err != nil {
		t.Fatal(err)
	}
	for _, result := range service.PollStorage() {
		if result.OperationID == loadID {
			return result
		}
	}
	t.Fatal("load did not complete")
	return gameapi.StorageResult{}
}

// The player sees "created by an incompatible game version" for a save from
// another build, whichever layer caught it, and "incomplete or damaged" or a
// storage failure only for saves that really are.
func TestLoadReportsAnotherBuildsSaveAsIncompatible(t *testing.T) {
	superseded := currentSave(t)
	superseded.GeographyAlgorithm = "dispersal-map-v5"
	damaged := currentSave(t)
	damaged.GridWidth++
	_, decodeErr := DecodeSaveState([]byte(`{"schema_version":99,"field_from_the_future":1}`))
	for _, test := range []struct {
		name       string
		repository CampaignRepository
		want       gameapi.ErrorCode
	}{
		{"superseded identifier", &repositoryStub{writable: true, written: &superseded}, gameapi.ErrIncompatibleSave},
		{"newer build's payload", &failingReadRepository{err: decodeErr}, gameapi.ErrIncompatibleSave},
		{"damaged save", &repositoryStub{writable: true, written: &damaged}, gameapi.ErrInvalidSave},
		{"unreadable storage", &failingReadRepository{err: errors.New("disk on fire")}, gameapi.ErrStorageFailure},
	} {
		result := loadResult(t, test.repository)
		var gameErr *gameapi.GameError
		if !errors.As(result.Err, &gameErr) || gameErr.Code != test.want {
			t.Fatalf("%s: load error %#v, want code %s", test.name, result.Err, test.want)
		}
		if result.ReplacementFrame != nil {
			t.Fatalf("%s: a rejected load replaced the world", test.name)
		}
	}
}
