package ui

import (
	"fmt"

	"github.com/adsouza/africa2ice/pkg/gameapi"
)

// ChecklistRow is one of the three per-band decisions the panel walks the
// player through each turn (spec §5.1). Exactly one row is open at a time.
type ChecklistRow uint8

const (
	RowMove ChecklistRow = iota
	RowResearch
	RowWorkforce
	ChecklistRowCount
)

func (row ChecklistRow) Title() string {
	return [...]string{"Move", "Research", "Workforce"}[row]
}

// ResearchDone is the Research row predicate.
func ResearchDone(band gameapi.Band) bool { return band.HasResearchTarget }

// DefaultOpenRow is the first row that still needs a decision, falling back to
// Move. It is recomputed on selection change, load, new campaign, and completed
// turn; the player may then open any row explicitly.
func DefaultOpenRow(band *gameapi.Band) ChecklistRow {
	if band == nil {
		return RowMove
	}
	switch {
	case !MoveDone(*band):
		return RowMove
	case !ResearchDone(*band):
		return RowResearch
	default:
		return RowMove
	}
}

// MoveSummary is the collapsed Move row's one-line state.
func MoveSummary(frame *gameapi.Frame, band gameapi.Band) string {
	switch {
	case band.HasQueuedMigration:
		label := "Move set"
		if frame != nil && int(band.QueuedMigration) < len(frame.Tiles) {
			label += " → " + frame.Tiles[band.QueuedMigration].Biome.String()
			if direction := CompassDirection(frame, band.TileID, band.QueuedMigration); direction != "" {
				label += ", " + direction
			}
		}
		return label
	case band.HasInterbreedTarget:
		return fmt.Sprintf("Interbreeding with B%d", band.InterbreedTargetID)
	case band.SpatialActionUsed:
		// The frame does not say which action consumed it; a split is the only
		// remaining player action that can (DESIGN.md forbids inferring more).
		return "Split queued"
	default:
		return "Choose a destination"
	}
}

// ResearchSummary is the collapsed Research row's one-line state.
func ResearchSummary(band gameapi.Band) string {
	if !band.HasResearchTarget {
		return "No target · choose one"
	}
	option := band.ResearchOptions[band.ResearchTarget]
	return fmt.Sprintf("%s %.0f/%.0f · %+.1f/turn", band.ResearchTarget, band.ResearchProgress[band.ResearchTarget], option.Cost, band.OriginalResearchGainPreview)
}

// RoleShortLabel is the compact per-role label used in summaries and sliders.
func RoleShortLabel(role gameapi.WorkforceRole) string {
	return [...]string{"Foraging", "Hunt / fish", "Toolcraft", "Megafauna", "Shelter / care"}[role]
}

var roleSummaryLabels = [gameapi.AssignmentCount]string{"F", "H", "T", "M", "S"}

// WorkforceSummary is the collapsed Workforce row's one-line state.
func WorkforceSummary(allocation [gameapi.AssignmentCount]uint16, dirty bool) string {
	if dirty {
		return "Unapplied changes"
	}
	summary := ""
	for role, points := range allocation {
		if role > 0 {
			summary += " · "
		}
		summary += fmt.Sprintf("%s %d", roleSummaryLabels[role], points/100)
	}
	return summary
}

// EndTurnGate is the End turn button's state (spec §5.3). Hard blocks disable
// the button; the soft block keeps it enabled but amber and demands a second
// click (or Space) while unarmed.
type EndTurnGate struct {
	Enabled bool
	Soft    bool
	Label   string
}

func EndTurnGateFor(frame *gameapi.Frame, dirtyDraft, pendingCursor, armed bool) EndTurnGate {
	switch {
	case dirtyDraft:
		return EndTurnGate{Label: "End turn · apply or discard workforce changes"}
	case pendingCursor:
		return EndTurnGate{Label: "End turn · queue or clear the arrow-key choice"}
	}
	waiting := 0
	if frame != nil {
		waiting = BandsNeedingMove(frame.Bands)
	}
	switch {
	case waiting == 0:
		return EndTurnGate{Enabled: true, Label: "End turn · Space"}
	case armed:
		return EndTurnGate{Enabled: true, Label: "End turn now · Space"}
	case waiting == 1:
		return EndTurnGate{Enabled: true, Soft: true, Label: "End turn · 1 band still needs a move"}
	default:
		return EndTurnGate{Enabled: true, Soft: true, Label: fmt.Sprintf("End turn · %d bands still need a move", waiting)}
	}
}

// CompassDirection names the one-step direction from one tile to an adjacent
// tile, or returns "" when they are not neighbours. Y grows southward.
func CompassDirection(frame *gameapi.Frame, from, to gameapi.TileID) string {
	if frame == nil || int(from) >= len(frame.Tiles) || int(to) >= len(frame.Tiles) {
		return ""
	}
	dx := frame.Tiles[to].X - frame.Tiles[from].X
	dy := frame.Tiles[to].Y - frame.Tiles[from].Y
	if dx < -1 || dx > 1 || dy < -1 || dy > 1 || (dx == 0 && dy == 0) {
		return ""
	}
	return [3][3]string{{"NW", "N", "NE"}, {"W", "", "E"}, {"SW", "S", "SE"}}[dy+1][dx+1]
}
