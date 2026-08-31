//go:build !js

package ui

import (
	"errors"
	"os"
	"path/filepath"
	"sync"
)

type FileUISettingsStore struct {
	path        string
	mu          sync.Mutex
	active      bool
	completions []UISettingsCompletion
}

func NewFileUISettingsStore(path string) (*FileUISettingsStore, error) {
	if path == "" {
		return nil, errors.New("UI settings path is empty")
	}
	return &FileUISettingsStore{path: path}, nil
}

func (store *FileUISettingsStore) BeginRead(revision uint64) error {
	if !store.begin() {
		return errors.New("UI settings operation already active")
	}
	go func() {
		settings := DefaultUISettings()
		payload, err := os.ReadFile(store.path)
		if errors.Is(err, os.ErrNotExist) {
			err = nil
		} else if err == nil {
			settings, err = DecodeUISettings(payload)
		}
		store.complete(UISettingsCompletion{Operation: UISettingsRead, Revision: revision, Settings: settings, Err: err})
	}()
	return nil
}

func (store *FileUISettingsStore) BeginWrite(revision uint64, settings UISettings) error {
	if !store.begin() {
		return errors.New("UI settings operation already active")
	}
	settings = NormalizeUISettings(settings)
	go func() {
		payload, err := EncodeUISettings(settings)
		if err == nil {
			err = os.MkdirAll(filepath.Dir(store.path), 0o755)
		}
		if err == nil {
			var temporary *os.File
			temporary, err = os.CreateTemp(filepath.Dir(store.path), ".ui-settings-*")
			if err == nil {
				temporaryPath := temporary.Name()
				if _, err = temporary.Write(payload); err == nil {
					err = temporary.Sync()
				}
				if closeErr := temporary.Close(); err == nil {
					err = closeErr
				}
				if err == nil {
					err = os.Rename(temporaryPath, store.path)
				}
				if err != nil {
					_ = os.Remove(temporaryPath)
				}
			}
		}
		store.complete(UISettingsCompletion{Operation: UISettingsWrite, Revision: revision, Settings: settings, Err: err})
	}()
	return nil
}

func (store *FileUISettingsStore) Poll() []UISettingsCompletion {
	store.mu.Lock()
	defer store.mu.Unlock()
	result := append([]UISettingsCompletion(nil), store.completions...)
	store.completions = store.completions[:0]
	return result
}

func (store *FileUISettingsStore) begin() bool {
	store.mu.Lock()
	defer store.mu.Unlock()
	if store.active {
		return false
	}
	store.active = true
	return true
}

func (store *FileUISettingsStore) complete(completion UISettingsCompletion) {
	store.mu.Lock()
	defer store.mu.Unlock()
	store.active = false
	store.completions = append(store.completions, completion)
}
