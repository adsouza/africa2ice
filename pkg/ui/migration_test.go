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
	if message := MigrationDiagnosticMessage(diagnostic, band); !strings.Contains(message, "cannot migrate into open water") || !strings.Contains(message, "Coastal Navigation") || !strings.Contains(message, "Wallacea") {
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

func TestMigrationWaterMessageAtPassageEndpointNamesTheRealGate(t *testing.T) {
	frame, band := migrationFixture()
	diagnostic := DiagnoseMigration(frame, band, 3)

	// PassageStatuses maps NotAtEndpoint to Unavailable, so a Locked or Open
	// status places the band on that passage's shore.
	band.PassageStatuses[gameapi.BeringStrait] = gameapi.PassageLocked
	if message := MigrationDiagnosticMessage(diagnostic, band); strings.Contains(message, "Coastal Navigation") || !strings.Contains(message, "Beringia") {
		t.Fatalf("Bering endpoint message %q blames technology for a climate gate", message)
	}

	band.PassageStatuses[gameapi.BeringStrait] = gameapi.PassageUnavailable
	band.PassageStatuses[gameapi.SouthWallacea] = gameapi.PassageLocked
	if message := MigrationDiagnosticMessage(diagnostic, band); !strings.Contains(message, "Coastal Navigation") || !strings.Contains(message, "South Wallacea") {
		t.Fatalf("Wallacea endpoint message %q does not name the locked passage and its technology", message)
	}

	band.PassageStatuses[gameapi.SouthWallacea] = gameapi.PassageOpen
	if message := MigrationDiagnosticMessage(diagnostic, band); !strings.Contains(message, "South Wallacea") || !strings.Contains(message, "far endpoint") {
		t.Fatalf("open passage message %q does not direct the player to the far endpoint", message)
	}
}

// splitEligibleFrame is a band that satisfies every split guard: sapiens, its
// spatial action still open, over the stress threshold, at the minimum viable
// population, with one ordinary-land neighbour to establish on.
func splitEligibleFrame() *gameapi.Frame {
	return &gameapi.Frame{CampaignResult: gameapi.Ongoing, Bands: []gameapi.Band{{
		ID: 1, Species: gameapi.HomoSapiens, Population: gameapi.MinSplitSourcePopulation,
		Stress:              gameapi.SplitStressThreshold + 0.01,
		MigrationCandidates: []gameapi.MigrationCandidate{{TileID: 1}},
	}}}
}

// DiagnoseSplit exists so the Split button can be disabled when the command
// would refuse it: it used to look available whatever the band's state, and
// clicking it only produced a notice (user-reported). Every reason returned
// here is a guard domain.World.Split or splitBand actually applies, and each
// already has player copy in errors.go.
func TestDiagnoseSplitNamesTheGuardThatWouldRefuse(t *testing.T) {
	if got := DiagnoseSplit(splitEligibleFrame(), &splitEligibleFrame().Bands[0]); got != "" {
		t.Fatalf("eligible band = %q, want no blocking reason", got)
	}
	for name, test := range map[string]struct {
		mutate func(*gameapi.Frame)
		want   gameapi.ErrorCode
	}{
		"campaign already over": {
			func(f *gameapi.Frame) { f.CampaignResult = gameapi.DispersalFailed },
			gameapi.ErrCampaignComplete,
		},
		"computer-controlled band": {
			func(f *gameapi.Frame) { f.Bands[0].Species = gameapi.ArchaicHominin },
			gameapi.ErrComputerControlledBand,
		},
		"spatial action already spent": {
			func(f *gameapi.Frame) { f.Bands[0].SpatialActionUsed = true },
			gameapi.ErrSpatialActionUsed,
		},
		"a queued migration also spends the action": {
			func(f *gameapi.Frame) { f.Bands[0].HasQueuedMigration = true },
			gameapi.ErrSpatialActionUsed,
		},
		"stress exactly at the threshold is not over it": {
			func(f *gameapi.Frame) { f.Bands[0].Stress = gameapi.SplitStressThreshold },
			gameapi.ErrSplitStressTooLow,
		},
		"one person short of two viable bands": {
			func(f *gameapi.Frame) { f.Bands[0].Population = gameapi.MinSplitSourcePopulation - 1 },
			gameapi.ErrSplitPopulationTooLow,
		},
		"no ordinary-land neighbour": {
			func(f *gameapi.Frame) {
				f.Bands[0].MigrationCandidates = []gameapi.MigrationCandidate{{TileID: 1, RequiresPassage: true}}
			},
			gameapi.ErrSplitDestinationNotAdjacent,
		},
		"campaign is at the band limit": {
			func(f *gameapi.Frame) { f.Bands = append(f.Bands, make([]gameapi.Band, gameapi.MaxBands)...) },
			gameapi.ErrBandLimitReached,
		},
	} {
		t.Run(name, func(t *testing.T) {
			frame := splitEligibleFrame()
			test.mutate(frame)
			// The band pointer is taken after the mutation: appending to
			// frame.Bands can reallocate its backing array.
			if got := DiagnoseSplit(frame, &frame.Bands[0]); got != test.want {
				t.Fatalf("DiagnoseSplit = %q, want %q", got, test.want)
			}
		})
	}
	if got := DiagnoseSplit(splitEligibleFrame(), nil); got != gameapi.ErrBandNotFound {
		t.Fatalf("nil band = %q, want %q", got, gameapi.ErrBandNotFound)
	}
}

// moveActionFrame is a band with every Move action available: sapiens, action
// open, crowded enough and large enough to split, one ordinary-land neighbour,
// and an archaic band sharing its tile to interbreed with.
func moveActionFrame() *gameapi.Frame {
	return &gameapi.Frame{
		CampaignResult: gameapi.Ongoing,
		Tiles: []gameapi.Tile{
			{ID: 0, X: 0, Y: 0, Land: true, Explored: true, BaselineK: 150, EcologicalK: 150},
			{ID: 1, X: 1, Y: 0, Land: true, Explored: true, BaselineK: 150, EcologicalK: 150},
		},
		Bands: []gameapi.Band{
			{
				ID: 1, Species: gameapi.HomoSapiens, TileID: 0,
				Population: gameapi.MinSplitSourcePopulation, Stress: gameapi.SplitStressThreshold + 0.01,
				MigrationCandidates:    []gameapi.MigrationCandidate{{TileID: 1}},
				InterbreedCandidateIDs: []gameapi.BandID{2},
			},
			{ID: 2, Species: gameapi.ArchaicHominin, TileID: 0, Population: 20},
		},
	}
}

// DiagnoseMoveActions is the single source of truth for the Move row's four
// buttons: each is disabled exactly when its explanation is non-empty, so the
// button state and the tooltip cannot disagree (spec §4.1).
func TestDiagnoseMoveActionsExplainsEachUnavailableAction(t *testing.T) {
	diagnose := func(frame *gameapi.Frame) MoveActionBlocks {
		band := &frame.Bands[0]
		target := TargetTileLiveability(frame, band, 1)
		return DiagnoseMoveActions(frame, band, target, TargetHover, 1)
	}
	if blocks := diagnose(moveActionFrame()); blocks != (MoveActionBlocks{}) {
		t.Fatalf("fully eligible band = %+v, want every action available", blocks)
	}
	for name, test := range map[string]struct {
		mutate                                func(*gameapi.Frame)
		moveHere, bestTile, split, interbreed string
	}{
		"campaign over blocks every action": {
			func(f *gameapi.Frame) { f.CampaignResult = gameapi.DispersalFailed },
			ErrorCodeMessage(gameapi.ErrCampaignComplete), ErrorCodeMessage(gameapi.ErrCampaignComplete),
			ErrorCodeMessage(gameapi.ErrCampaignComplete), ErrorCodeMessage(gameapi.ErrCampaignComplete),
		},
		"an archaic band is computer controlled": {
			func(f *gameapi.Frame) { f.Bands[0].Species = gameapi.ArchaicHominin },
			ErrorCodeMessage(gameapi.ErrComputerControlledBand), ErrorCodeMessage(gameapi.ErrComputerControlledBand),
			ErrorCodeMessage(gameapi.ErrComputerControlledBand), ErrorCodeMessage(gameapi.ErrComputerControlledBand),
		},
		"a spent spatial action blocks every action": {
			func(f *gameapi.Frame) { f.Bands[0].SpatialActionUsed = true },
			ErrorCodeMessage(gameapi.ErrSpatialActionUsed), ErrorCodeMessage(gameapi.ErrSpatialActionUsed),
			ErrorCodeMessage(gameapi.ErrSpatialActionUsed), ErrorCodeMessage(gameapi.ErrSpatialActionUsed),
		},
		"too little pressure stops only the split": {
			func(f *gameapi.Frame) { f.Bands[0].Stress = gameapi.SplitStressThreshold },
			"", "", ErrorCodeMessage(gameapi.ErrSplitStressTooLow), "",
		},
		"too few people stops only the split": {
			func(f *gameapi.Frame) { f.Bands[0].Population = gameapi.MinSplitSourcePopulation - 1 },
			"", "", ErrorCodeMessage(gameapi.ErrSplitPopulationTooLow), "",
		},
		"no archaic neighbour stops only interbreeding": {
			func(f *gameapi.Frame) { f.Bands[0].InterbreedCandidateIDs = nil },
			"", "", "", noInterbreedPartnerMessage,
		},
		"only passage routes stop Best tile and the split": {
			func(f *gameapi.Frame) {
				f.Bands[0].MigrationCandidates = []gameapi.MigrationCandidate{{TileID: 1, RequiresPassage: true}}
			},
			"", noOrdinaryLandMessage, ErrorCodeMessage(gameapi.ErrSplitDestinationNotAdjacent), "",
		},
	} {
		t.Run(name, func(t *testing.T) {
			frame := moveActionFrame()
			test.mutate(frame)
			got := diagnose(frame)
			want := MoveActionBlocks{MoveHere: test.moveHere, BestTile: test.bestTile, Split: test.split, Interbreed: test.interbreed}
			if got != want {
				t.Fatalf("DiagnoseMoveActions =\n  %+v\nwant\n  %+v", got, want)
			}
		})
	}
}

// With no tile chosen, Move here is the only action that cannot proceed, and it
// says what to do rather than what went wrong.
func TestDiagnoseMoveActionsAsksForATargetWhenNoneIsChosen(t *testing.T) {
	frame := moveActionFrame()
	band := &frame.Bands[0]
	blocks := DiagnoseMoveActions(frame, band, TileLiveability{}, TargetNone, 0)
	if blocks.MoveHere != noTargetChosenMessage {
		t.Fatalf("Move here = %q, want the prompt to choose a tile", blocks.MoveHere)
	}
	if blocks.BestTile != "" || blocks.Split != "" || blocks.Interbreed != "" {
		t.Fatalf("a missing target blocked more than Move here: %+v", blocks)
	}
	// An unreachable chosen tile falls through to the existing migration
	// diagnostic rather than inventing new copy.
	band.MigrationCandidates = nil
	unreachable := DiagnoseMoveActions(frame, band, TargetTileLiveability(frame, band, 1), TargetHover, 1)
	if want := MigrationDiagnosticMessage(DiagnoseMigration(frame, band, 1), band); unreachable.MoveHere != want || want == "" {
		t.Fatalf("unreachable target = %q, want the migration diagnostic %q", unreachable.MoveHere, want)
	}
}

// The blocked-terrain message used to promise recovery unconditionally. On the
// tiles that matter most — the two-column Caucasus land bridge that carries the
// only eastern route out of the Levant — that promise is false: once long-term
// cooling drives their vegetation under the cold cutoff they stay dead for the
// rest of the campaign, and a player who waits for the thaw waits forever.
func TestUninhabitableMessageDoesNotPromiseRecoveryForAPermanentlyDeadTile(t *testing.T) {
	frame, band := migrationFixture()
	frame.Turn = 200
	frame.Tiles[4].LastHabitableTurn = 72 // died at turn 72 and never recovers

	diagnostic := DiagnoseMigration(frame, band, 4)
	if diagnostic.Reason != MigrationBlockedUninhabitable {
		t.Fatalf("DiagnoseMigration() reason = %v, want MigrationBlockedUninhabitable", diagnostic.Reason)
	}
	message := MigrationDiagnosticMessage(diagnostic, band)
	if strings.Contains(message, "later") {
		t.Errorf("message for a permanently dead tile promises a recovery that never comes: %q", message)
	}
}

func TestUninhabitableMessageStillOffersHopeForATileThatRecovers(t *testing.T) {
	frame, band := migrationFixture()
	frame.Turn = 200
	frame.Tiles[4].LastHabitableTurn = 340 // an interstadial reopens it

	message := MigrationDiagnosticMessage(DiagnoseMigration(frame, band, 4), band)
	if !strings.Contains(message, "later") {
		t.Errorf("message for a tile that does recover hides that fact: %q", message)
	}
}

func TestEasyModeSplitDiagnostics(t *testing.T) {
	frame := &gameapi.Frame{EasyMode: true, CampaignResult: gameapi.Ongoing}
	band := gameapi.Band{Species: gameapi.HomoSapiens, Population: 20, MigrationCandidates: []gameapi.MigrationCandidate{{TileID: 1}}}
	if code := DiagnoseSplit(frame, &band); code != "" {
		t.Fatalf("easy split blocked: %v", code)
	}
	frame.EasyMode = false
	if code := DiagnoseSplit(frame, &band); code != gameapi.ErrSplitStressTooLow {
		t.Fatalf("normal split: %v", code)
	}
	frame.EasyMode = true
	band.Population = 19
	if code := DiagnoseSplit(frame, &band); code != gameapi.ErrSplitPopulationTooLow {
		t.Fatalf("small split: %v", code)
	}
}
