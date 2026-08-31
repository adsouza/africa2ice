//go:build !js

package logging

import (
	"os"
	"strings"
	"testing"
)

func TestBoundedSinkWrapsInPlaceAndClipsOversizeRecords(t *testing.T) {
	file, err := os.CreateTemp(t.TempDir(), "bounded-log-*.jsonl")
	if err != nil {
		t.Fatal(err)
	}
	path := file.Name()
	sink := newBoundedSink(file, 180)
	for range 8 {
		if _, err := sink.Write([]byte("{\"level\":\"INFO\",\"msg\":\"ordinary-record-with-padding-1234567890\"}\n")); err != nil {
			t.Fatal(err)
		}
	}
	if err := sink.Close(); err != nil {
		t.Fatal(err)
	}
	payload, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if int64(len(payload)) > 180 || !strings.Contains(string(payload), "log.wrapped") {
		t.Fatalf("wrapped payload (%d bytes) = %s", len(payload), payload)
	}

	file, err = os.CreateTemp(t.TempDir(), "clipped-log-*.jsonl")
	if err != nil {
		t.Fatal(err)
	}
	path = file.Name()
	sink = newBoundedSink(file, 120)
	secret := strings.Repeat("private-payload", 30)
	if _, err := sink.Write([]byte(secret)); err != nil {
		t.Fatal(err)
	}
	if err := sink.Close(); err != nil {
		t.Fatal(err)
	}
	payload, err = os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(payload), "log.record_clipped") || strings.Contains(string(payload), "private-payload") {
		t.Fatalf("clipped payload leaked the record: %s", payload)
	}
}
