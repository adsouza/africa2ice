//go:build !js

package app

import (
	"os"
	"path/filepath"

	"github.com/adsouza/africa2ice/internal/adapters/storage"
	"github.com/adsouza/africa2ice/internal/application"
	"github.com/adsouza/africa2ice/pkg/ui"
)

func newCampaignRepository() (application.CampaignRepository, error) {
	root, err := os.UserConfigDir()
	if err != nil {
		return nil, err
	}
	return storage.NewFileRepository(filepath.Join(root, "africa2ice", "saves"))
}

func shouldResumeSavedGameOnStartup() bool { return true }

func newUISettingsStore() (ui.UISettingsStore, error) {
	root, err := os.UserConfigDir()
	if err != nil {
		return nil, err
	}
	return ui.NewFileUISettingsStore(filepath.Join(root, "africa2ice", "ui_settings.json"))
}
