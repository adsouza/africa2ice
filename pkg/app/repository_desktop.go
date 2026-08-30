//go:build !js

package app

import (
	"os"
	"path/filepath"

	"github.com/adsouza/africa2ice/internal/adapters/storage"
	"github.com/adsouza/africa2ice/internal/application"
)

func newCampaignRepository() (application.CampaignRepository, error) {
	root, err := os.UserConfigDir()
	if err != nil {
		return nil, err
	}
	return storage.NewFileRepository(filepath.Join(root, "africa2ice", "saves"))
}

func shouldResumeSavedGameOnStartup() bool { return true }
