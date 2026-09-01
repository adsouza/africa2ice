package verification

import (
	"bytes"
	"reflect"
	"sort"
	"strings"
	"testing"
)

func TestReferenceRunIsDeterministic(t *testing.T) {
	first, err := ReferenceRun(42, 10, "reference")
	if err != nil {
		t.Fatal(err)
	}
	second, err := ReferenceRun(42, 10, "reference")
	if err != nil {
		t.Fatal(err)
	}
	firstJSON, err := CanonicalJSON(first)
	if err != nil {
		t.Fatal(err)
	}
	secondJSON, err := CanonicalJSON(second)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(firstJSON, secondJSON) {
		t.Fatalf("same reference run diverged:\n%s\n%s", firstJSON, secondJSON)
	}
	if len(first) != 2 || first[0].Turn != 0 || first[1].Turn != 10 || first[0].StateHash == first[1].StateHash {
		t.Fatalf("checkpoints = %#v", first)
	}
}

func TestReferenceCampaignClearsReleaseMargins(t *testing.T) {
	records, err := ReferenceRun(ReferenceSeed, MaxTurns, "reference")
	if err != nil {
		t.Fatal(err)
	}
	if len(records) != 5 {
		t.Fatalf("checkpoint count = %d, want 5", len(records))
	}
	final := records[len(records)-1]
	if final.Turn != MaxTurns || final.CampaignResult != "Victory" {
		t.Fatalf("terminal checkpoint = %#v", final)
	}
	if final.FirstDestinationTurn < 0 || final.FirstDestinationTurn > 350 || final.FirstDestinationPopulation < 40 {
		t.Fatalf("destination margins = turn %d, population %d", final.FirstDestinationTurn, final.FirstDestinationPopulation)
	}
	if final.SapiensBandCount < 5 || final.SapiensPopulation < 200 {
		t.Fatalf("survival margins = %d bands, %d people", final.SapiensBandCount, final.SapiensPopulation)
	}
}

func TestPolicyCatalogAndValidation(t *testing.T) {
	want := []string{"reference", "toward-south-asia", "toward-yellow-river", "toward-sahul", "toward-beringia"}
	if got := PolicyNames(); !reflect.DeepEqual(got, want) {
		t.Fatalf("PolicyNames() = %q, want %q", got, want)
	}
	for _, name := range want {
		if _, err := ParsePolicy(name); err != nil {
			t.Errorf("ParsePolicy(%q): %v", name, err)
		}
	}
	if _, err := ParsePolicy("unknown"); err == nil {
		t.Fatal("unknown policy accepted")
	}
	if _, err := ReferenceRun(0, -1, "reference"); err == nil {
		t.Fatal("negative turn count accepted")
	}
	if _, err := ReferenceRun(0, MaxTurns+1, "reference"); err == nil {
		t.Fatal("overlong turn count accepted")
	}
}

func TestCheckpointJSONKeysAreDeclaredInCanonicalOrderAndSeedIsLossless(t *testing.T) {
	typeOfRecord := reflect.TypeOf(CheckpointRecord{})
	keys := make([]string, 0, typeOfRecord.NumField())
	for index := 0; index < typeOfRecord.NumField(); index++ {
		key, _, _ := strings.Cut(typeOfRecord.Field(index).Tag.Get("json"), ",")
		keys = append(keys, key)
	}
	sorted := append([]string(nil), keys...)
	sort.Strings(sorted)
	if !reflect.DeepEqual(keys, sorted) {
		t.Fatalf("checkpoint keys are not lexical: %q", keys)
	}
	payload, err := CanonicalJSON([]CheckpointRecord{{Seed: ReferenceSeed}})
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Contains(payload, []byte(`"seed":11400714819323198485`)) {
		t.Fatalf("uint64 seed was not encoded losslessly: %s", payload)
	}
}

func TestDumpMapContainsTwoCompleteLayers(t *testing.T) {
	encoded, err := DumpMap(0)
	if err != nil {
		t.Fatal(err)
	}
	lines := strings.Split(strings.TrimSpace(encoded), "\n")
	if len(lines) != 1+64+1+64 || lines[0] != "biomes 96x64" || lines[65] != "regions" {
		t.Fatalf("map shape = %d lines, headers %q/%q", len(lines), lines[0], lines[65])
	}
	for index, line := range append(append([]string(nil), lines[1:65]...), lines[66:]...) {
		if len(line) != 96 {
			t.Fatalf("map row %d width = %d", index, len(line))
		}
	}
}
