package app

import (
	"github.com/adsouza/africa2ice/pkg/gameapi"
	"github.com/adsouza/africa2ice/pkg/hud"
	"github.com/adsouza/africa2ice/pkg/render"
	"github.com/adsouza/africa2ice/pkg/ui"
)

// hudState derives everything the chrome draws from the accepted frame and
// UI-local fields. Nothing here is stored.
func (g *Game) hudState() hud.State {
	frame := g.displayFrame()
	note := g.fieldNote
	note.Celebration = g.breakthroughFrames > 0
	state := hud.State{
		Frame:        frame,
		SelectedBand: g.selectedBand,
		Preview:      render.MigrationPreview{BandID: g.migrationPreviewBand, TileID: g.migrationPreviewTile, Visible: g.hasMigrationPreview},
		Hover:        render.TileHover{TileID: g.hoveredTile, Visible: g.hasHoveredTile},
		OpenRow:      g.openRow,
		DetailsOpen:  g.detailsOpen,
		BandListOpen: g.bandListOpen,
		Workforce: hud.WorkforceDraft{
			Visible: g.hasAssignmentDraft, AllocationBP: g.assignmentDraft, SelectedRole: g.assignmentRole,
			Dirty: g.assignmentDraftDirty(), Valid: g.assignmentDraftValid(),
		},
		InterbreedFocus: g.interbreedFocus,
		EndTurn:         ui.EndTurnGateFor(frame, g.assignmentDraftDirty(), g.hasMigrationPreview, g.endTurnArmed),
		Note:            note,
		NotesMode:       g.notesMode,
		Guide:           g.guide,
		ShortcutsOpen:   g.shortcutsOpen,
		Ending:          ui.CampaignEndScene(frame),
		Viewport:        g.viewport,
		Transform:       render.FitPresentation(g.viewport.RenderWidthPx, g.viewport.RenderHeightPx),
	}
	if band := g.selected(); band != nil {
		state.Workforce.Population = band.Population
	}
	if frame != nil && frame.CampaignResult != gameapi.Ongoing {
		state.EndTurn = ui.EndTurnGate{}
	}
	state.Overlay = g.overlayState()
	return state
}

// overlayState is a stub until Task 13 gives the modal scenes their widgets.
func (g *Game) overlayState() hud.OverlayState {
	return hud.OverlayState{Scene: g.scenes.Current()}
}

// resetDisclosure returns the panel to its defaults for a new selection, a
// load, a new campaign, or a completed turn (spec §5.2).
func (g *Game) resetDisclosure() {
	g.openRow = ui.DefaultOpenRow(g.selected())
	g.rowChosen = false
	g.detailsOpen = false
	g.bandListOpen = false
	g.endTurnArmed = false
}

// advanceOpenRow moves to the next unfinished row after an accepted action
// unless the player chose a row explicitly.
func (g *Game) advanceOpenRow() {
	if !g.rowChosen {
		g.openRow = ui.DefaultOpenRow(g.selected())
	}
}
