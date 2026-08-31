package ui

import (
	"strings"
	"testing"

	"github.com/adsouza/africa2ice/pkg/gameapi"
)

func TestDiagnoseMigrationGeneralRejections(t *testing.T) {
	frame, band := migrationFixture()
	tests := []struct {
		name        string
		destination gameapi.TileID
		want        MigrationBlockReason
	}{
		{name: "authoritative candidate", destination: 1, want: MigrationAllowed},
		{name: "current tile", destination: 0, want: MigrationBlockedCurrentTile},
		{name: "unexplored water does not leak terrain", destination: 2, want: MigrationBlockedUnexplored},
		{name: "explored water", destination: 3, want: MigrationBlockedWater},
		{name: "uninhabitable land", destination: 4, want: MigrationBlockedUninhabitable},
		{name: "escarpment", destination: 9, want: MigrationBlockedEscarpment},
		{name: "blocked diagonal", destination: 5, want: MigrationBlockedDiagonal},
		{name: "more than one tile away", destination: 6, want: MigrationBlockedTooFar},
		{name: "adjacent without a route", destination: 7, want: MigrationBlockedNoRoute},
		{name: "outside map", destination: 99, want: MigrationBlockedInvalidTile},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if got := DiagnoseMigration(frame, band, test.destination); got.Reason != test.want {
				t.Fatalf("DiagnoseMigration() reason = %v, want %v", got.Reason, test.want)
			}
		})
	}
}

func TestDiagnoseMigrationSpentActionTakesPrecedence(t *testing.T) {
	frame, band := migrationFixture()
	band.SpatialActionUsed = true
	if got := DiagnoseMigration(frame, band, 1); got.Reason != MigrationBlockedActionSpent {
		t.Fatalf("DiagnoseMigration() reason = %v, want action spent", got.Reason)
	}
}

func TestMoveMigrationPreviewReachesCornersThroughBlockedTiles(t *testing.T) {
	frame, band := migrationFixture()

	up, ok := MoveMigrationPreview(frame, band, band.TileID, 0, -1)
	if !ok || up != 4 || DiagnoseMigration(frame, band, up).Reason != MigrationBlockedUninhabitable {
		t.Fatalf("up preview = (%d, %t), want blocked tile 4", up, ok)
	}
	corner, ok := MoveMigrationPreview(frame, band, up, -1, 0)
	if !ok || corner != 8 || DiagnoseMigration(frame, band, corner).Reason != MigrationAllowed {
		t.Fatalf("corner preview = (%d, %t), want reachable tile 8", corner, ok)
	}
	if _, ok := MoveMigrationPreview(frame, band, corner, -1, 0); ok {
		t.Fatal("preview moved beyond the one-turn neighborhood")
	}
	if _, ok := MoveMigrationPreview(frame, band, band.TileID, 1, 1); ok {
		t.Fatal("one input step accepted a diagonal delta")
	}

	water, ok := MoveMigrationPreview(frame, band, band.TileID, -1, 0)
	if !ok || DiagnoseMigration(frame, band, water).Reason != MigrationBlockedWater {
		t.Fatal("left-arrow destination did not retain the standard water rejection")
	}
}

func TestDiagnoseMigrationNamesPassageRequirement(t *testing.T) {
	frame, band := migrationFixture()
	frame.Passages = []gameapi.Passage{{ID: gameapi.NorthWallacea, From: 0, To: 6}}
	band.PassageStatuses[gameapi.NorthWallacea] = gameapi.PassageLocked

	diagnostic := DiagnoseMigration(frame, band, 6)
	if diagnostic.Reason != MigrationBlockedPassageTechnology || diagnostic.Passage != gameapi.NorthWallacea {
		t.Fatalf("DiagnoseMigration() = %+v, want locked North Wallacea", diagnostic)
	}
	if message := MigrationDiagnosticMessage(diagnostic, band); !strings.Contains(message, "Coastal Navigation") {
		t.Fatalf("message %q does not explain the technology requirement", message)
	}

	frame.Passages[0].ID = gameapi.BeringStrait
	band.PassageStatuses[gameapi.BeringStrait] = gameapi.PassageLocked
	diagnostic = DiagnoseMigration(frame, band, 6)
	if diagnostic.Reason != MigrationBlockedPassageClimate {
		t.Fatalf("DiagnoseMigration() reason = %v, want climate lock", diagnostic.Reason)
	}
}

func TestMigrationWaterMessageExplainsNavigationLimit(t *testing.T) {
	frame, band := migrationFixture()
	diagnostic := DiagnoseMigration(frame, band, 3)
	if message := MigrationDiagnosticMessage(diagnostic, band); !strings.Contains(message, "cannot migrate into open water") || !strings.Contains(message, "Coastal Navigation") {
		t.Fatalf("message %q does not explain the water restriction", message)
	}

	band.AcquiredTech |= 1 << gameapi.CoastalNavigation
	if message := MigrationDiagnosticMessage(diagnostic, band); !strings.Contains(message, "cannot occupy open water") || !strings.Contains(message, "land endpoint") {
		t.Fatalf("message %q does not explain named-passage use", message)
	}
}

func migrationFixture() (*gameapi.Frame, *gameapi.Band) {
	frame := &gameapi.Frame{Tiles: []gameapi.Tile{
		{ID: 0, X: 0, Y: 0, Land: true, Explored: true, BaselineK: 100},
		{ID: 1, X: 1, Y: 0, Land: true, Explored: true, BaselineK: 100},
		{ID: 2, X: 0, Y: 1, Land: false, Explored: false},
		{ID: 3, X: -1, Y: 0, Land: false, Explored: true},
		{ID: 4, X: 0, Y: -1, Land: true, Explored: true, BaselineK: 0},
		{ID: 5, X: 1, Y: 1, Land: true, Explored: true, BaselineK: 100},
		{ID: 6, X: 3, Y: 0, Land: true, Explored: true, BaselineK: 100},
		{ID: 7, X: -1, Y: 0, Land: true, Explored: true, BaselineK: 100},
		{ID: 8, X: -1, Y: -1, Land: true, Explored: true, BaselineK: 100},
		{ID: 9, X: 0, Y: 1, Land: true, Explored: true, BaselineK: 100},
	}}
	frame.Escarpments = []gameapi.Escarpment{{Name: "Test Front", First: 0, Second: 9}}
	band := &gameapi.Band{
		ID: 1, Species: gameapi.HomoSapiens, TileID: 0,
		MigrationCandidates: []gameapi.MigrationCandidate{{TileID: 1}, {TileID: 8}},
	}
	return frame, band
}
