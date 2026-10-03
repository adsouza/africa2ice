package ui

import "testing"

// Preference files are user-writable bytes. Whatever they hold, decoding must
// not panic; a failure must fall back to the defaults; and accepted settings
// must already be normalized and survive an encode/decode round trip.
func FuzzDecodeUISettings(f *testing.F) {
	current, err := EncodeUISettings(DefaultUISettings())
	if err != nil {
		f.Fatal(err)
	}
	for _, seed := range []string{
		string(current),
		`{"SchemaVersion":1,"FieldNotesVisible":true,"MasterVolume":0.4,"Muted":false}`,
		`{"SchemaVersion":2,"FieldNotesVisible":false,"MasterVolume":2,"Muted":true,"GuideDismissed":true,"FieldNotesExpanded":true}`,
		`{"SchemaVersion":3,"FieldNotesVisible":true,"MasterVolume":-1,"Muted":false,"GuideDismissed":false,"FieldNotesExpanded":false,"ReducedMotion":true}`,
		`{"SchemaVersion":4,"FieldNotesVisible":true,"MasterVolume":0.5,"Muted":false,"GuideDismissed":true,"FieldNotesExpanded":true,"ReducedMotion":false,"EasyMode":true}`,
		`{"SchemaVersion":4,"MasterVolume":1e309}`,
		`{"SchemaVersion":null}`, `{}`, `null`, `[]`, `{"SchemaVersion":1}{}`, ``,
	} {
		f.Add([]byte(seed))
	}
	f.Fuzz(func(t *testing.T, payload []byte) {
		settings, err := DecodeUISettings(payload)
		if err != nil {
			if settings != DefaultUISettings() {
				t.Fatalf("rejected input returned %+v, want the defaults", settings)
			}
			return
		}
		if normalized := NormalizeUISettings(settings); normalized != settings {
			t.Fatalf("accepted settings %+v are not normalized (%+v)", settings, normalized)
		}
		encoded, err := EncodeUISettings(settings)
		if err != nil {
			t.Fatalf("accepted settings do not encode: %v", err)
		}
		again, err := DecodeUISettings(encoded)
		if err != nil || again != settings {
			t.Fatalf("round trip = %+v, %v; want %+v", again, err, settings)
		}
	})
}
