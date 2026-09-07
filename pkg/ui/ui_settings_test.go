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

func TestUISettingsSchemaTwoRoundTripsAndUpgradesSchemaOne(t *testing.T) {
	want := UISettings{SchemaVersion: UISettingsSchemaVersion, FieldNotesVisible: false, MasterVolume: 0.3, Muted: true, GuideDismissed: true, FieldNotesExpanded: true}
	payload, err := EncodeUISettings(want)
	if err != nil {
		t.Fatal(err)
	}
	got, err := DecodeUISettings(payload)
	if err != nil || got != want {
		t.Fatalf("round trip = %+v, %v; want %+v", got, err, want)
	}

	v1 := []byte(`{"SchemaVersion":1,"FieldNotesVisible":false,"MasterVolume":0.3,"Muted":true}`)
	upgraded, err := DecodeUISettings(v1)
	if err != nil {
		t.Fatalf("schema 1 payload rejected: %v", err)
	}
	if upgraded.SchemaVersion != UISettingsSchemaVersion || upgraded.GuideDismissed || upgraded.FieldNotesExpanded || upgraded.MasterVolume != 0.3 || upgraded.FieldNotesVisible || !upgraded.Muted {
		t.Fatalf("upgraded schema 1 = %+v", upgraded)
	}

	missing := []byte(`{"SchemaVersion":2,"FieldNotesVisible":true,"MasterVolume":0.5,"Muted":false,"GuideDismissed":false}`)
	if _, err := DecodeUISettings(missing); err == nil {
		t.Fatal("schema 2 payload without FieldNotesExpanded was accepted")
	}
	if _, err := DecodeUISettings([]byte(`{"SchemaVersion":5,"FieldNotesVisible":true,"MasterVolume":0.5,"Muted":false,"GuideDismissed":false,"FieldNotesExpanded":false,"ReducedMotion":false}`)); err == nil {
		t.Fatal("unknown schema 5 was accepted")
	}
}

func TestSchemaTwoDefaultsReducedMotionOff(t *testing.T) {
	settings, err := DecodeUISettings([]byte(`{"SchemaVersion":2,"FieldNotesVisible":true,"MasterVolume":0.5,"Muted":false,"GuideDismissed":false,"FieldNotesExpanded":false}`))
	if err != nil {
		t.Fatalf("decode: %v", err)
	}
	if settings.ReducedMotion {
		t.Fatal("a schema-2 record must not arrive with reduced motion on")
	}
	if settings.SchemaVersion != UISettingsSchemaVersion {
		t.Fatalf("SchemaVersion = %d, want %d", settings.SchemaVersion, UISettingsSchemaVersion)
	}
}

func TestSchemaThreeRoundTripsReducedMotion(t *testing.T) {
	settings, err := DecodeUISettings([]byte(`{"SchemaVersion":3,"FieldNotesVisible":true,"MasterVolume":0.5,"Muted":false,"GuideDismissed":false,"FieldNotesExpanded":false,"ReducedMotion":true}`))
	if err != nil {
		t.Fatalf("decode: %v", err)
	}
	if !settings.ReducedMotion {
		t.Fatal("ReducedMotion did not survive the round trip")
	}

	missing := []byte(`{"SchemaVersion":3,"FieldNotesVisible":true,"MasterVolume":0.5,"Muted":false,"GuideDismissed":false,"FieldNotesExpanded":false}`)
	if _, err := DecodeUISettings(missing); err == nil {
		t.Fatal("schema 3 payload without ReducedMotion was accepted")
	}
}

func TestEasyModePreferenceDefaultsMigrationAndOptOut(t *testing.T) {
	if !DefaultUISettings().EasyMode {
		t.Fatal("easy mode should default on")
	}
	old := []byte(`{"SchemaVersion":3,"FieldNotesVisible":true,"MasterVolume":0.5,"Muted":false,"GuideDismissed":false,"FieldNotesExpanded":false,"ReducedMotion":true}`)
	settings, err := DecodeUISettings(old)
	if err != nil || !settings.EasyMode || !settings.ReducedMotion {
		t.Fatalf("upgrade: %+v %v", settings, err)
	}
	settings.EasyMode = false
	payload, err := EncodeUISettings(settings)
	if err != nil {
		t.Fatal(err)
	}
	restored, err := DecodeUISettings(payload)
	if err != nil || restored != settings {
		t.Fatalf("opt-out round trip: %+v %v", restored, err)
	}
}
