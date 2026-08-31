package ui

import (
	"math"
	"testing"
)

func TestUISettingsRequireCompleteTypedV1Record(t *testing.T) {
	tests := []struct {
		name    string
		payload string
		wantOK  bool
	}{
		{name: "complete", payload: `{"SchemaVersion":1,"FieldNotesVisible":false,"MasterVolume":0.75,"Muted":true,"Ignored":"yes"}`, wantOK: true},
		{name: "old development record", payload: `{"SchemaVersion":1,"FieldNotesVisible":false}`},
		{name: "future", payload: `{"SchemaVersion":2,"FieldNotesVisible":false,"MasterVolume":0.5,"Muted":false}`},
		{name: "null", payload: `{"SchemaVersion":1,"FieldNotesVisible":null,"MasterVolume":0.5,"Muted":false}`},
		{name: "wrong bool", payload: `{"SchemaVersion":1,"FieldNotesVisible":1,"MasterVolume":0.5,"Muted":false}`},
		{name: "wrong number", payload: `{"SchemaVersion":1,"FieldNotesVisible":true,"MasterVolume":"0.5","Muted":false}`},
		{name: "malformed", payload: `{`},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			settings, err := DecodeUISettings([]byte(test.payload))
			if (err == nil) != test.wantOK {
				t.Fatalf("DecodeUISettings error = %v, want success %t", err, test.wantOK)
			}
			if err != nil && settings != DefaultUISettings() {
				t.Fatalf("invalid record default = %#v", settings)
			}
		})
	}
}

func TestUISettingsVolumeIsFiniteAndClamped(t *testing.T) {
	for _, test := range []struct {
		value float64
		want  float64
	}{{value: -2, want: 0}, {value: 0.4, want: 0.4}, {value: 4, want: 1}, {value: math.NaN(), want: 0.5}} {
		settings := NormalizeUISettings(UISettings{MasterVolume: test.value})
		if settings.MasterVolume != test.want || settings.SchemaVersion != UISettingsSchemaVersion {
			t.Fatalf("NormalizeUISettings(%v) = %#v", test.value, settings)
		}
	}
	if _, err := DecodeUISettings([]byte(`{"SchemaVersion":1,"FieldNotesVisible":true,"MasterVolume":1e999,"Muted":false}`)); err == nil {
		t.Fatal("non-finite JSON volume was accepted")
	}
}
