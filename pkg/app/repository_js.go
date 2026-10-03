//go:build js

package app

import (
	"github.com/adsouza/africa2ice/internal/adapters/storage"
	"github.com/adsouza/africa2ice/internal/application"
	"github.com/adsouza/africa2ice/pkg/ui"
)

func newCampaignRepository(panicGuard func()) (application.CampaignRepository, error) {
	return storage.NewIndexedDBRepository(panicGuard), nil
}

func shouldResumeSavedGameOnStartup() bool { return true }

func newUISettingsStore(panicGuard func()) (ui.UISettingsStore, error) {
	store, err := ui.NewIndexedDBUISettingsStore(panicGuard)
	if err != nil {
		return nil, err
	}
	return store, nil
}
