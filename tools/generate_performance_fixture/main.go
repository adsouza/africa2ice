package main

import (
	"fmt"
	"os"

	"github.com/adsouza/africa2ice/internal/application"
	"github.com/adsouza/africa2ice/internal/verification"
)

const referenceSeed = 0x9e3779b97f4a7c15

func main() {
	game, _, err := verification.PrepareGame(referenceSeed, 300, "reference")
	if err != nil {
		panic(err)
	}
	service, ok := game.(*application.GameService)
	if !ok {
		panic("verification port is not a GameService")
	}
	state, err := service.ExportSaveState()
	if err != nil {
		panic(err)
	}
	state, err = application.MaximumRenderFixture(state)
	if err != nil {
		panic(err)
	}
	payload, err := application.EncodeSaveState(state)
	if err != nil {
		panic(err)
	}
	if err := os.WriteFile("testdata/performance_profile_save.json", append(payload, '\n'), 0o644); err != nil {
		panic(err)
	}
	fmt.Printf("wrote %d-byte turn-%d fixture with %d bands\n", len(payload)+1, state.Turn, len(state.Bands))
}
