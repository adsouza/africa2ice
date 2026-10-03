//go:build !js

package app

import (
	"os"
	"path/filepath"

	"github.com/adsouza/africa2ice/internal/adapters/storage"
	"github.com/adsouza/africa2ice/internal/application"
	"github.com/adsouza/africa2ice/pkg/ui"
)

func newCampaignRepository(panicGuard func()) (application.CampaignRepository, error) {
	root, err := os.UserConfigDir()
	if err != nil {
		return nil, err
	}
	return storage.NewFileRepository(filepath.Join(root, "africa2ice", "saves"), panicGuard)
}

func shouldResumeSavedGameOnStartup() bool { return true }

func newUISettingsStore(panicGuard func()) (ui.UISettingsStore, error) {
	root, err := os.UserConfigDir()
	if err != nil {
		return nil, err
	}
	store, err := ui.NewFileUISettingsStore(filepath.Join(root, "africa2ice", "ui_settings.json"), panicGuard)
	if err != nil {
		return nil, err
	}
	return store, nil
}
