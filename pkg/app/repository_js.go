//go:build js

package app

import (
	"github.com/adsouza/africa2ice/internal/adapters/storage"
	"github.com/adsouza/africa2ice/internal/application"
)

func newCampaignRepository() (application.CampaignRepository, error) {
	return storage.NewIndexedDBRepository(), nil
}

func shouldResumeSavedGameOnStartup() bool { return false }
