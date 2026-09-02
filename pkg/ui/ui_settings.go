package ui

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math"
)

const UISettingsSchemaVersion = 2

type UISettings struct {
	SchemaVersion      int     `json:"SchemaVersion"`
	FieldNotesVisible  bool    `json:"FieldNotesVisible"`
	MasterVolume       float64 `json:"MasterVolume"`
	Muted              bool    `json:"Muted"`
	GuideDismissed     bool    `json:"GuideDismissed"`
	FieldNotesExpanded bool    `json:"FieldNotesExpanded"`
}

func DefaultUISettings() UISettings {
	return UISettings{SchemaVersion: UISettingsSchemaVersion, FieldNotesVisible: true, MasterVolume: 0.5}
}

func NormalizeUISettings(settings UISettings) UISettings {
	settings.SchemaVersion = UISettingsSchemaVersion
	if math.IsNaN(settings.MasterVolume) || math.IsInf(settings.MasterVolume, 0) {
		settings.MasterVolume = DefaultUISettings().MasterVolume
	}
	settings.MasterVolume = min(max(settings.MasterVolume, 0), 1)
	return settings
}

func DecodeUISettings(payload []byte) (UISettings, error) {
	decoder := json.NewDecoder(bytes.NewReader(payload))
	decoder.UseNumber()
	var fields map[string]json.RawMessage
	if err := decoder.Decode(&fields); err != nil {
		return DefaultUISettings(), fmt.Errorf("decode UI settings: %w", err)
	}
	var trailing any
	if err := decoder.Decode(&trailing); !errors.Is(err, io.EOF) {
		if err == nil {
			err = errors.New("trailing JSON value")
		}
		return DefaultUISettings(), fmt.Errorf("decode UI settings: %w", err)
	}
	var settings UISettings
	if raw, ok := fields["SchemaVersion"]; !ok || bytes.Equal(bytes.TrimSpace(raw), []byte("null")) {
		return DefaultUISettings(), errors.New("decode UI settings: missing or null SchemaVersion")
	} else if err := json.Unmarshal(raw, &settings.SchemaVersion); err != nil {
		return DefaultUISettings(), fmt.Errorf("decode UI settings schema: %w", err)
	}
	required := []string{"FieldNotesVisible", "MasterVolume", "Muted"}
	switch settings.SchemaVersion {
	case 1:
		// Schema 1 predates the guide and drawer height; both default to false.
	case 2:
		required = append(required, "GuideDismissed", "FieldNotesExpanded")
	default:
		return DefaultUISettings(), fmt.Errorf("unsupported UI settings schema %d", settings.SchemaVersion)
	}
	for _, name := range required {
		value, ok := fields[name]
		if !ok || bytes.Equal(bytes.TrimSpace(value), []byte("null")) {
			return DefaultUISettings(), fmt.Errorf("decode UI settings: missing or null %s", name)
		}
	}
	if err := json.Unmarshal(fields["FieldNotesVisible"], &settings.FieldNotesVisible); err != nil {
		return DefaultUISettings(), fmt.Errorf("decode FieldNotesVisible: %w", err)
	}
	if err := json.Unmarshal(fields["MasterVolume"], &settings.MasterVolume); err != nil {
		return DefaultUISettings(), fmt.Errorf("decode MasterVolume: %w", err)
	}
	if math.IsNaN(settings.MasterVolume) || math.IsInf(settings.MasterVolume, 0) {
		return DefaultUISettings(), errors.New("decode MasterVolume: value must be finite")
	}
	if err := json.Unmarshal(fields["Muted"], &settings.Muted); err != nil {
		return DefaultUISettings(), fmt.Errorf("decode Muted: %w", err)
	}
	if settings.SchemaVersion == 2 {
		if err := json.Unmarshal(fields["GuideDismissed"], &settings.GuideDismissed); err != nil {
			return DefaultUISettings(), fmt.Errorf("decode GuideDismissed: %w", err)
		}
		if err := json.Unmarshal(fields["FieldNotesExpanded"], &settings.FieldNotesExpanded); err != nil {
			return DefaultUISettings(), fmt.Errorf("decode FieldNotesExpanded: %w", err)
		}
	}
	return NormalizeUISettings(settings), nil
}

func EncodeUISettings(settings UISettings) ([]byte, error) {
	settings = NormalizeUISettings(settings)
	return json.Marshal(settings)
}

type UISettingsOperation uint8

const (
	UISettingsRead UISettingsOperation = iota
	UISettingsWrite
)

type UISettingsCompletion struct {
	Operation UISettingsOperation
	Revision  uint64
	Settings  UISettings
	Err       error
}

// UISettingsStore is the asynchronous, presentation-only preference port.
// Implementations enqueue completions; they never mutate live UI from a
// goroutine or JavaScript callback.
type UISettingsStore interface {
	BeginRead(revision uint64) error
	BeginWrite(revision uint64, settings UISettings) error
	Poll() []UISettingsCompletion
}
