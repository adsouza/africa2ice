package ui

import (
	"fmt"

	"github.com/adsouza/africa2ice/pkg/gameapi"
)

// MigrationBlockReason classifies a destination rejected by the authoritative
// migration-candidate projection.
type MigrationBlockReason uint8

const (
	MigrationAllowed MigrationBlockReason = iota
	MigrationBlockedInvalidTile
	MigrationBlockedActionSpent
	MigrationBlockedCurrentTile
	MigrationBlockedUnexplored
	MigrationBlockedWater
	MigrationBlockedUninhabitable
	MigrationBlockedPassageTechnology
	MigrationBlockedPassageClimate
	MigrationBlockedPassageUnavailable
	MigrationBlockedEscarpment
	MigrationBlockedDiagonal
	MigrationBlockedTooFar
	MigrationBlockedNoRoute
)

// MigrationDiagnostic contains the presentation-safe reason for a rejected
// map destination and, when relevant, the named passage involved.
type MigrationDiagnostic struct {
	Reason  MigrationBlockReason
	Passage gameapi.PassageID
}

// MoveMigrationPreview moves a keyboard destination cursor one cardinal tile
// while keeping it inside the band's one-turn 3x3 neighborhood. The cursor
// may cross an ineligible tile so two arrow presses can still reach a valid
// diagonal destination; confirmation performs authoritative validation.
func MoveMigrationPreview(frame *gameapi.Frame, band *gameapi.Band, cursor gameapi.TileID, dx, dy int) (gameapi.TileID, bool) {
	if frame == nil || band == nil || int(band.TileID) >= len(frame.Tiles) || int(cursor) >= len(frame.Tiles) || absInt(dx)+absInt(dy) != 1 {
		return 0, false
	}
	origin := frame.Tiles[band.TileID]
	current := frame.Tiles[cursor]
	targetX, targetY := current.X+dx, current.Y+dy
	if absInt(targetX-origin.X) > 1 || absInt(targetY-origin.Y) > 1 {
		return 0, false
	}
	for index := range frame.Tiles {
		if frame.Tiles[index].X == targetX && frame.Tiles[index].Y == targetY {
			return gameapi.TileID(index), true
		}
	}
	return 0, false
}

// DiagnoseMigration explains why a clicked tile is absent from the selected
// band's authoritative candidate list. Unexplored tiles are classified before
// their terrain is inspected so the diagnostic cannot leak hidden geography.
func DiagnoseMigration(frame *gameapi.Frame, band *gameapi.Band, destination gameapi.TileID) MigrationDiagnostic {
	if frame == nil || band == nil || int(destination) >= len(frame.Tiles) || int(band.TileID) >= len(frame.Tiles) {
		return MigrationDiagnostic{Reason: MigrationBlockedInvalidTile}
	}
	if band.SpatialActionUsed {
		return MigrationDiagnostic{Reason: MigrationBlockedActionSpent}
	}
	if destination == band.TileID {
		return MigrationDiagnostic{Reason: MigrationBlockedCurrentTile}
	}
	tile := frame.Tiles[destination]
	if !tile.Explored {
		return MigrationDiagnostic{Reason: MigrationBlockedUnexplored}
	}
	for _, candidate := range band.MigrationCandidates {
		if candidate.TileID == destination {
			return MigrationDiagnostic{Reason: MigrationAllowed, Passage: candidate.Passage}
		}
	}
	if !tile.Land {
		return MigrationDiagnostic{Reason: MigrationBlockedWater}
	}
	if tile.BaselineK <= 0 {
		return MigrationDiagnostic{Reason: MigrationBlockedUninhabitable}
	}
	if passage, ok := framePassageBetween(frame, band.TileID, destination); ok {
		status := gameapi.PassageUnavailable
		if passage.ID < gameapi.PassageCount {
			status = band.PassageStatuses[passage.ID]
		}
		switch {
		case status == gameapi.PassageLocked && passage.ID == gameapi.BeringStrait:
			return MigrationDiagnostic{Reason: MigrationBlockedPassageClimate, Passage: passage.ID}
		case status == gameapi.PassageLocked:
			return MigrationDiagnostic{Reason: MigrationBlockedPassageTechnology, Passage: passage.ID}
		default:
			return MigrationDiagnostic{Reason: MigrationBlockedPassageUnavailable, Passage: passage.ID}
		}
	}
	if gameapi.EscarpmentBlocks(frame, band.TileID, destination) {
		return MigrationDiagnostic{Reason: MigrationBlockedEscarpment}
	}
	origin := frame.Tiles[band.TileID]
	dx, dy := absInt(tile.X-origin.X), absInt(tile.Y-origin.Y)
	if dx == 1 && dy == 1 {
		return MigrationDiagnostic{Reason: MigrationBlockedDiagonal}
	}
	if dx > 1 || dy > 1 {
		return MigrationDiagnostic{Reason: MigrationBlockedTooFar}
	}
	return MigrationDiagnostic{Reason: MigrationBlockedNoRoute}
}

// MigrationDiagnosticMessage renders a concise player-facing explanation.
func MigrationDiagnosticMessage(diagnostic MigrationDiagnostic, band *gameapi.Band) string {
	switch diagnostic.Reason {
	case MigrationAllowed:
		return ""
	case MigrationBlockedInvalidTile:
		return "That tile is outside the playable map."
	case MigrationBlockedActionSpent:
		return "This band has already used its spatial action this turn."
	case MigrationBlockedCurrentTile:
		return "The selected band is already on that tile."
	case MigrationBlockedUnexplored:
		return "That area is unexplored; move into an outlined frontier tile first."
	case MigrationBlockedWater:
		if passage, status, ok := localPassage(band); ok {
			switch {
			case status == gameapi.PassageOpen:
				return fmt.Sprintf("Bands cannot occupy open water; %s crosses it from here: select its highlighted far endpoint.", passage)
			case passage == gameapi.BeringStrait:
				return "Bands cannot occupy open water; Beringia is closed until long-term cooling exposes the land bridge."
			default:
				return fmt.Sprintf("Bands cannot occupy open water; research Coastal Navigation to cross %s.", passage)
			}
		}
		if band != nil && band.AcquiredTech&(1<<gameapi.CoastalNavigation) != 0 {
			return "Bands cannot occupy open water; use a land endpoint of a named passage."
		}
		return "Bands cannot migrate into open water; only named passages cross it, and Coastal Navigation unlocks the Wallacea crossings."
	case MigrationBlockedUninhabitable:
		return "This terrain is uninhabitable now; climate change may make it viable later."
	case MigrationBlockedPassageTechnology:
		return fmt.Sprintf("%s is locked; research Coastal Navigation to cross it.", diagnostic.Passage)
	case MigrationBlockedPassageClimate:
		return "Beringia is closed; long-term cooling must expose the land bridge first."
	case MigrationBlockedPassageUnavailable:
		return fmt.Sprintf("%s cannot be used from here right now.", diagnostic.Passage)
	case MigrationBlockedEscarpment:
		return "A steep escarpment blocks entry from this direction."
	case MigrationBlockedDiagonal:
		return "No traversable route: diagonal moves cannot cut across a water corner."
	case MigrationBlockedTooFar:
		return "Too far away: move one outlined tile or use an eligible named passage."
	default:
		return "There is no traversable route to that tile from this band."
	}
}

func framePassageBetween(frame *gameapi.Frame, origin, destination gameapi.TileID) (gameapi.Passage, bool) {
	for _, passage := range frame.Passages {
		if passage.From == origin && passage.To == destination || passage.To == origin && passage.From == destination {
			return passage, true
		}
	}
	return gameapi.Passage{}, false
}

func absInt(value int) int {
	if value < 0 {
		return -value
	}
	return value
}

// localPassage reports the named passage whose endpoint the band occupies,
// preferring an open crossing over a locked one. The projection maps
// NotAtEndpoint (and an uninhabitable far shore) to Unavailable, so any other
// status places the band on that passage's shore.
func localPassage(band *gameapi.Band) (gameapi.PassageID, gameapi.PassageStatus, bool) {
	if band == nil {
		return 0, gameapi.PassageUnavailable, false
	}
	found, foundStatus, ok := gameapi.PassageID(0), gameapi.PassageUnavailable, false
	for id, status := range band.PassageStatuses {
		switch status {
		case gameapi.PassageOpen:
			return gameapi.PassageID(id), status, true
		case gameapi.PassageLocked:
			if !ok {
				found, foundStatus, ok = gameapi.PassageID(id), status, true
			}
		}
	}
	return found, foundStatus, ok
}

// HasOrdinaryLandCandidate reports whether the band can reach an adjacent land
// tile without crossing a passage. Both spatial actions that pick their own
// destination need one: Game.splitSelectedBand and Game.moveToBestTile each
// take the first candidate with RequiresPassage false, and a passage crossing
// is not an eligible neighbour for a new band under Split's adjacency guard.
// A non-empty candidate list is not enough on its own.
func HasOrdinaryLandCandidate(band gameapi.Band) bool {
	for _, candidate := range band.MigrationCandidates {
		if !candidate.RequiresPassage {
			return true
		}
	}
	return false
}

// DiagnoseSplit reports the code domain.World.Split would refuse this band
// with, or "" when a split would be accepted. The Split button used to look
// available whatever the band's state, so clicking it produced a notice
// instead of an action (user-reported).
//
// Like DiagnoseMigration this reads only projected data — Band.Stress carries
// the authoritative domain.World.BandStress value — and never recomputes a
// domain rule. Recomputing stress here is not possible anyway: BandStress
// divides by BaselineK times the band's technology capacity multiplier, and
// that multiplier is deliberately not part of the public frame. A tile's
// displayed occupancy (see liveability.go) divides by EcologicalK instead, so
// it is close to stress but not equal to it, and must not stand in for it:
// the two disagree on a degraded tile, which would disable the button for a
// split the command would have allowed.
//
// The guards run in the order World.Split applies them, so the reason matches
// the error a click would produce. Population comes last because splitBand
// checks it only after Split has cleared the destination guards.
func DiagnoseSplit(frame *gameapi.Frame, band *gameapi.Band) gameapi.ErrorCode {
	if frame == nil || band == nil {
		return gameapi.ErrBandNotFound
	}
	if frame.CampaignResult != gameapi.Ongoing {
		return gameapi.ErrCampaignComplete
	}
	if band.Species != gameapi.HomoSapiens {
		return gameapi.ErrComputerControlledBand
	}
	// MoveDone rather than SpatialActionUsed alone: a queued migration or a
	// chosen interbreeding partner has already committed the same action,
	// and the domain rejects the split on the flag they set.
	if MoveDone(*band) {
		return gameapi.ErrSpatialActionUsed
	}
	if len(frame.Bands) >= gameapi.MaxBands {
		return gameapi.ErrBandLimitReached
	}
	if band.Stress <= gameapi.SplitStressThreshold {
		return gameapi.ErrSplitStressTooLow
	}
	if !HasOrdinaryLandCandidate(*band) {
		return gameapi.ErrSplitDestinationNotAdjacent
	}
	if band.Population < gameapi.MinSplitSourcePopulation {
		return gameapi.ErrSplitPopulationTooLow
	}
	return ""
}

// Copy for the Move row blocks that have no matching gameapi.ErrorCode,
// because nothing rejects them at the command boundary: the player is stopped
// before a command is ever built.
const (
	noOrdinaryLandMessage      = "No adjacent land tile is reachable this turn; only passage crossings are open."
	noInterbreedPartnerMessage = "No archaic band shares this tile, so there is no one to interbreed with."
	noTargetChosenMessage      = "Choose a destination first: hover or click an outlined tile, or use the arrow keys."
)

// MoveActionBlocks explains why each of the Move row's four actions cannot be
// sent, or holds "" for one that can. A button is disabled exactly when its
// entry is non-empty, so its enabled state and the reason it shows on hover
// are one fact rather than two that can disagree.
type MoveActionBlocks struct {
	MoveHere   string
	BestTile   string
	Split      string
	Interbreed string
}

// AllBlocked stops every Move action for one shared reason. pkg/hud uses it to
// honour hud.State.CampaignOver, which is the panel's own authority on a
// finished campaign, without duplicating the copy or the shape of this struct.
func AllBlocked(reason string) MoveActionBlocks {
	return MoveActionBlocks{MoveHere: reason, BestTile: reason, Split: reason, Interbreed: reason}
}

// DiagnoseMoveActions explains each of the Move row's four actions. It is the
// only place those conditions live: pkg/hud disables a button exactly when its
// entry is non-empty and shows that entry as the button's tooltip, so a
// control never looks live while doing nothing, and never goes dead without
// saying why (spec §4.1).
func DiagnoseMoveActions(frame *gameapi.Frame, band *gameapi.Band, target TileLiveability, source TargetSource, targetTile gameapi.TileID) MoveActionBlocks {
	if frame == nil || band == nil {
		return MoveActionBlocks{}
	}
	// Three conditions stop all four actions at once. They are checked in the
	// order the domain applies them, so the copy matches the error a command
	// would have returned.
	shared := ""
	switch {
	case frame.CampaignResult != gameapi.Ongoing:
		shared = ErrorCodeMessage(gameapi.ErrCampaignComplete)
	case band.Species != gameapi.HomoSapiens:
		shared = ErrorCodeMessage(gameapi.ErrComputerControlledBand)
	case MoveDone(*band):
		// Covers a queued migration and a chosen interbreeding partner as well
		// as a spent flag, which is why TargetQueued needs no branch of its
		// own: a queued target implies MoveDone.
		shared = ErrorCodeMessage(gameapi.ErrSpatialActionUsed)
	}
	if shared != "" {
		return AllBlocked(shared)
	}

	blocks := MoveActionBlocks{Split: ErrorCodeMessage(DiagnoseSplit(frame, band))}
	switch {
	case source == TargetNone:
		// Nothing has gone wrong yet, so this says what to do instead.
		blocks.MoveHere = noTargetChosenMessage
	case !target.Reachable:
		blocks.MoveHere = MigrationDiagnosticMessage(DiagnoseMigration(frame, band, targetTile), band)
	}
	if !HasOrdinaryLandCandidate(*band) {
		blocks.BestTile = noOrdinaryLandMessage
	}
	if len(band.InterbreedCandidateIDs) == 0 {
		blocks.Interbreed = noInterbreedPartnerMessage
	}
	return blocks
}
