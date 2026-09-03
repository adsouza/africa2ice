package app

import (
	"fmt"

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
		Frame:          frame,
		SelectedBand:   g.selectedBand,
		Preview:        render.MigrationPreview{BandID: g.migrationPreviewBand, TileID: g.migrationPreviewTile, Visible: g.hasMigrationPreview},
		Hover:          render.TileHover{TileID: g.hoveredTile, Visible: g.hasHoveredTile},
		OpenRow:        g.openRow,
		ResearchCursor: g.researchCursor,
		DetailsOpen:    g.detailsOpen,
		BandListOpen:   g.bandListOpen,
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
	if band := g.selected(); band != nil {
		state.Camera = hud.CameraState{FocusAvailable: band.Species == gameapi.HomoSapiens && !ui.MoveDone(*band), Focused: g.camera.Mode == render.CameraFocus}
	}
	if frame != nil && frame.CampaignResult != gameapi.Ongoing {
		state.EndTurn = ui.EndTurnGate{}
	}
	state.Overlay = g.overlayState()
	return state
}

// overlayState derives the modal scene's widget content from UI-local
// storage/settings fields; it never stores anything of its own.
func (g *Game) overlayState() hud.OverlayState {
	overlay := hud.OverlayState{Scene: g.scenes.Current(), MasterVolume: g.settings.MasterVolume, Muted: g.settings.Muted, SettingsDisabled: g.settingsLoading}
	if overlay.Scene != ui.SceneStorage {
		return overlay
	}
	overlay.StorageSaving = g.storageMode == storageBrowserSave
	overlay.StorageHeading = "Load / Delete"
	if overlay.StorageSaving {
		overlay.StorageHeading = "Save / Delete"
	}
	for index, slot := range storageBrowserSlots {
		row := hud.StorageRow{Label: storageSlotLabel(slot), Slot: slot, Detail: "Empty", Writable: slot >= 1 && slot <= 3}
		if metadata, exists := g.storageMetadata(slot); exists {
			row.Occupied = true
			row.Detail = fmt.Sprintf("Turn %d · %d BP · sapiens %d", metadata.Turn, metadata.YearBP, metadata.SapiensPopulation)
		}
		overlay.StorageRows[index] = row
	}
	switch {
	case g.storageListID != 0:
		overlay.StorageBusy = "Reading save metadata…"
	case g.storageOperationID != 0:
		overlay.StorageBusy = "Storage operation pending…"
	}
	return overlay
}

// resetDisclosure returns the panel to its defaults for a new selection, a
// load, a new campaign, or a completed turn (spec §5.2).
func (g *Game) resetDisclosure() {
	g.openRow = ui.DefaultOpenRow(g.selected())
	g.rowChosen = false
	g.detailsOpen = false
	g.bandListOpen = false
	g.endTurnArmed = false
	g.cameraOverride = false
}

// advanceOpenRow moves to the next unfinished row after an accepted action
// unless the player chose a row explicitly.
func (g *Game) advanceOpenRow() {
	if !g.rowChosen {
		g.openRow = ui.DefaultOpenRow(g.selected())
	}
}
