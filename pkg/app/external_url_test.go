package app

import (
	"errors"
	"strings"
	"testing"

	"github.com/adsouza/africa2ice/pkg/hud"
)

func TestPublicationIntentOpensBrowserWithoutChangingCampaign(t *testing.T) {
	game := New(&gameStub{frame: migrationPreviewFrame()})
	beforeFrame, beforeNote, beforeBand := game.frame, game.fieldNote, game.selectedBand
	var opened []string
	game.openExternalURL = func(url string) error {
		opened = append(opened, url)
		return nil
	}
	const url = "https://doi.org/10.1038/nature09710"
	game.handleIntent(hud.Intent{Kind: hud.IntentOpenPublication, URL: url})
	game.handleIntent(hud.Intent{Kind: hud.IntentOpenPublication, URL: "https://example.com"})
	if len(opened) != 1 || opened[0] != url {
		t.Fatalf("browser requests = %v", opened)
	}
	if game.frame != beforeFrame || game.fieldNote != beforeNote || game.selectedBand != beforeBand {
		t.Fatal("opening a publication changed campaign or note state")
	}
	game.openExternalURL = func(string) error { return errors.New("launcher missing") }
	game.handleIntent(hud.Intent{Kind: hud.IntentOpenPublication, URL: url})
	if !strings.Contains(game.notice, "Could not open") {
		t.Fatalf("missing failure notice: %q", game.notice)
	}
}
