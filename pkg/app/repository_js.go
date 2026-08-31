//go:build js

package app

import (
	"github.com/adsouza/africa2ice/internal/adapters/storage"
	"github.com/adsouza/africa2ice/internal/application"
	"github.com/adsouza/africa2ice/pkg/ui"
)

func newCampaignRepository() (application.CampaignRepository, error) {
	return storage.NewIndexedDBRepository(), nil
}

func shouldResumeSavedGameOnStartup() bool { return true }

func newUISettingsStore() (ui.UISettingsStore, error) {
	return ui.NewIndexedDBUISettingsStore()
}
