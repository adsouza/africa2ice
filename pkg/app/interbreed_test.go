package app

import (
	"testing"

	"github.com/adsouza/africa2ice/pkg/gameapi"
)

func interbreedFrame(candidates ...gameapi.BandID) *gameapi.Frame {
	return &gameapi.Frame{
		CampaignResult: gameapi.Ongoing,
		Tiles:          []gameapi.Tile{{ID: 0, X: 0, Y: 0, Land: true, Explored: true, BaselineK: 100}},
		Bands: []gameapi.Band{
			{ID: 7, Species: gameapi.HomoSapiens, TileID: 0, Population: 120, InterbreedCandidateIDs: candidates},
			{ID: 9, Species: gameapi.ArchaicHominin, TileID: 0, Population: 90},
		},
	}
}

// Pressing the key with nothing to interbreed with used to do nothing at all:
// the guard short-circuited before apply(), so not even an error notice
// appeared. An advertised control that silently does nothing reads as broken.
func TestInterbreedWithoutCandidatesExplainsWhy(t *testing.T) {
	stub := &gameStub{frame: interbreedFrame()}
	game := New(stub)

	game.requestInterbreed()

	if stub.appliedCommand != nil {
		t.Fatalf("applied %T with no candidate available", stub.appliedCommand)
	}
	if game.notice == "" {
		t.Fatal("pressing interbreed with no candidate produced no notice at all")
	}
}

func TestInterbreedWithCandidateAppliesAndConfirms(t *testing.T) {
	stub := &gameStub{frame: interbreedFrame(9)}
	game := New(stub)

	game.requestInterbreed()

	command, ok := stub.appliedCommand.(gameapi.Interbreed)
	if !ok {
		t.Fatalf("applied command = %T, want gameapi.Interbreed", stub.appliedCommand)
	}
	if command.BandID != 7 || command.TargetBandID != 9 {
		t.Fatalf("applied %#v, want band 7 targeting band 9", command)
	}
	if game.notice == "" {
		t.Fatal("a successful interbreed gave the player no confirmation")
	}
}

// A band that has already moved or split cannot interbreed. Saying so is better
// than letting the command through to be rejected by the domain.
func TestInterbreedRefusesASpentSpatialAction(t *testing.T) {
	frame := interbreedFrame(9)
	frame.Bands[0].SpatialActionUsed = true
	stub := &gameStub{frame: frame}
	game := New(stub)

	game.requestInterbreed()

	if stub.appliedCommand != nil {
		t.Fatalf("applied %T despite a spent spatial action", stub.appliedCommand)
	}
	if game.notice == "" {
		t.Fatal("a refused interbreed produced no notice")
	}
}
