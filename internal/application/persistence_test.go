package application

import (
	"bytes"
	"encoding/json"
	"reflect"
	"strconv"
	"testing"

	"github.com/adsouza/africa2ice/internal/domain"
	"github.com/adsouza/africa2ice/pkg/gameapi"
)

func TestSaveStateRoundTrip(t *testing.T) {
	service, _ := NewGameService(0x9e3779b97f4a7c15)
	_, _ = service.EndTurn()
	save, err := service.ExportSaveState()
	if err != nil {
		t.Fatal(err)
	}
	encoded, err := EncodeSaveState(save)
	if err != nil {
		t.Fatal(err)
	}
	decoded, err := DecodeSaveState(encoded)
	if err != nil {
		t.Fatal(err)
	}
	world, err := decoded.RestoreWorld()
	if err != nil {
		t.Fatal(err)
	}
	restored, err := SaveStateFromWorld(world, decoded.WorldRevision)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(save, restored) {
		t.Fatal("save changed across JSON/domain round trip")
	}
}

func TestProjectedFrameIncludesPersistedEvents(t *testing.T) {
	service, _ := NewGameService(31)
	state, _ := service.world.ExportState()
	state.Events = []domain.Event{{Turn: 0, Kind: domain.EventAchievement, Region: domain.EastAfrica, LegacySummary: "A test achievement."}}
	world, err := domain.RestoreWorld(state)
	if err != nil {
		t.Fatal(err)
	}
	service.world = world
	frame, err := service.Snapshot()
	if err != nil {
		t.Fatal(err)
	}
	if len(frame.Events) != 1 || frame.Events[0].Kind != gameapi.EventAchievement || frame.Events[0].Summary != "A test achievement." {
		t.Fatalf("projected events = %#v", frame.Events)
	}
}

func TestSaveDecoderRejectsUnknownAndUnsupportedFields(t *testing.T) {
	service, _ := NewGameService(1)
	save, _ := service.ExportSaveState()
	encoded, _ := EncodeSaveState(save)
	unknown := bytes.Replace(encoded, []byte(`"schema_version":2`), []byte(`"schema_version":2,"mystery":true`), 1)
	if _, err := DecodeSaveState(unknown); err == nil {
		t.Fatal("unknown field accepted")
	}
	save.ClimateAlgorithm = "future"
	if _, err := save.RestoreWorld(); err == nil {
		t.Fatal("unsupported algorithm accepted")
	}
	save = SaveState{SchemaVersion: SaveSchemaVersion + 1}
	if _, err := save.RestoreWorld(); err == nil {
		t.Fatal("future schema accepted")
	}
}

func TestSaveStateRejectsNonUint32Population(t *testing.T) {
	service, _ := NewGameService(29)
	save, _ := service.ExportSaveState()
	encoded, _ := EncodeSaveState(save)
	populationToken := []byte(`"population":` + strconv.FormatUint(uint64(domain.StartingAnchors[0].Population), 10))
	if !bytes.Contains(encoded, populationToken) {
		t.Fatalf("first starting population token %q is absent", populationToken)
	}
	for _, invalid := range []string{"99.5", "-1", "4294967296"} {
		payload := bytes.Replace(encoded, populationToken, []byte(`"population":`+invalid), 1)
		if _, err := DecodeSaveState(payload); err == nil {
			t.Fatalf("saved population %s was accepted", invalid)
		}
	}
}

func TestWholeNumberPopulationJSONRemainsCompatible(t *testing.T) {
	service, _ := NewGameService(30)
	save, _ := service.ExportSaveState()
	encoded, _ := EncodeSaveState(save)
	decoded, err := DecodeSaveState(encoded)
	if err != nil {
		t.Fatal(err)
	}
	if decoded.Bands[0].Population != uint32(domain.StartingAnchors[0].Population) {
		t.Fatalf("population = %d", decoded.Bands[0].Population)
	}
}

func TestDevelopmentSaveWithoutOutcomeReportRemainsCompatible(t *testing.T) {
	service, _ := NewGameService(32)
	_, _ = service.EndTurn()
	save, _ := service.ExportSaveState()
	encoded, _ := EncodeSaveState(save)
	var document map[string]any
	if err := json.Unmarshal(encoded, &document); err != nil {
		t.Fatal(err)
	}
	for _, rawBand := range document["bands"].([]any) {
		delete(rawBand.(map[string]any), "last_outcome_report")
	}
	legacy, err := json.Marshal(document)
	if err != nil {
		t.Fatal(err)
	}
	decoded, err := DecodeSaveState(legacy)
	if err != nil {
		t.Fatal(err)
	}
	world, err := decoded.RestoreWorld()
	if err != nil {
		t.Fatal(err)
	}
	for _, band := range world.Bands() {
		if band.LastOutcomeReport != (domain.OutcomeReport{}) {
			t.Fatalf("missing historical report was fabricated: %#v", band.LastOutcomeReport)
		}
	}
}

func TestAllAlgorithmFieldsArePopulated(t *testing.T) {
	value := reflect.ValueOf(supportedAlgorithms)
	if value.NumField() != 34 {
		t.Fatalf("algorithm field count = %d", value.NumField())
	}
	for index := 0; index < value.NumField(); index++ {
		if value.Field(index).String() == "" {
			t.Fatalf("algorithm field %d is empty", index)
		}
	}
}
