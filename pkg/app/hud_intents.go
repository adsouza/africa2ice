package app

import (
	"github.com/adsouza/africa2ice/pkg/gameapi"
	"github.com/adsouza/africa2ice/pkg/hud"
	"github.com/adsouza/africa2ice/pkg/ui"
)

// handleIntents maps chrome clicks onto the same guarded methods hotkeys use.
func (g *Game) handleIntents(intents []hud.Intent) {
	for _, intent := range intents {
		g.handleIntent(intent)
	}
}

func (g *Game) handleIntent(intent hud.Intent) {
	switch intent.Kind {
	case hud.IntentSelectBand:
		g.selectBandByID(intent.Band)
		g.bandListOpen = false
	case hud.IntentToggleBandList:
		g.bandListOpen = !g.bandListOpen
	case hud.IntentToggleShortcuts:
		g.toggleShortcutSheet()
	case hud.IntentOpenRow:
		g.openRow, g.rowChosen = intent.Row, true
	case hud.IntentToggleDetails:
		g.detailsOpen = !g.detailsOpen
	case hud.IntentSetNotesMode:
		g.setNotesMode(intent.Notes)
	case hud.IntentMoveTo:
		if band := g.selected(); band != nil {
			g.clearMigrationPreview()
			g.tryQueueMigration(band, intent.Tile)
		}
	case hud.IntentMoveToBest:
		g.moveToBestTile()
	case hud.IntentSplit:
		g.splitSelectedBand()
	case hud.IntentInterbreed:
		if intent.Band != 0 {
			g.interbreedFocus = intent.Band
		}
		g.requestInterbreed()
	case hud.IntentChooseResearch:
		g.chooseResearchTechnology(intent.Tech)
	case hud.IntentSelectRole:
		g.assignmentRole = intent.Role
	case hud.IntentAdjustRole:
		g.assignmentRole = intent.Role
		g.editAssignmentDraft(intent.Delta)
	case hud.IntentApplyWorkforce:
		g.applyAssignmentDraft()
	case hud.IntentDiscardWorkforce:
		g.discardAssignmentDraft()
	case hud.IntentEndTurn:
		g.endTurn(intent.Force)
	case hud.IntentGuideNext:
		g.guide = g.guide.Next()
	case hud.IntentGuideDismiss:
		g.dismissGuide()
	case hud.IntentFocusTrait:
		if band := g.selected(); band != nil {
			if note, ok := ui.TraitFieldNote(intent.Trait, band.HeritableState[intent.Trait]); ok {
				g.traitFocus = intent.Trait
				g.setFieldNote(note)
			}
		}
	case hud.IntentFocusEvent:
		if note, ok := ui.EventKindFieldNote(intent.Event); ok {
			g.setFieldNote(note)
		}
	case hud.IntentOpenMenu:
		g.dispatchBatch([]ui.Action{ui.PushSceneAction(ui.SceneMenu)})
	case hud.IntentCameraToggle:
		g.toggleCameraOverride()
	default:
		g.handleOverlayIntent(intent)
	}
}

// moveToBestTile queues a migration to the selected band's first ordinary
// (non-passage) MigrationCandidates entry, sharing one code path between the
// Best tile button and the B hotkey (spec §3.4).
func (g *Game) moveToBestTile() {
	band := g.selected()
	if band == nil {
		return
	}
	for _, candidate := range band.MigrationCandidates {
		if !candidate.RequiresPassage {
			g.clearMigrationPreview()
			g.tryQueueMigration(band, candidate.TileID)
			return
		}
	}
	g.showNotice("No reachable land tile to move to this turn.")
}

// handleOverlayIntent maps modal-scene widget clicks onto the same guarded
// methods the scenes' hotkeys use.
func (g *Game) handleOverlayIntent(intent hud.Intent) {
	switch intent.Kind {
	case hud.IntentBack:
		g.dispatchBatch([]ui.Action{ui.PopSceneAction()})
	case hud.IntentContinue:
		g.dispatchBatch([]ui.Action{ui.PopSceneAction()})
	case hud.IntentNewCampaign:
		g.startNewCampaign()
		if g.scenes.Current() == ui.SceneTitle {
			g.dispatchBatch([]ui.Action{ui.PopSceneAction()})
		}
	case hud.IntentOpenStorage:
		mode := storageBrowserLoad
		if intent.Save {
			mode = storageBrowserSave
		}
		g.openStorageBrowser(mode)
	case hud.IntentOpenSettings:
		g.dispatchBatch([]ui.Action{ui.PushSceneAction(ui.SceneSettings)})
	case hud.IntentReturnToTitle:
		if g.assignmentDraftDirty() {
			g.showNotice("Apply or discard workforce changes before returning to title")
			return
		}
		g.scenes.Reset()
		g.scenes.Push(ui.SceneTitle)
	case hud.IntentSaveSlot:
		g.storageSelection = storageIndexForSlot(intent.Slot)
		g.activateStorageSelection()
	case hud.IntentLoadSlot:
		g.storageSelection = storageIndexForSlot(intent.Slot)
		g.activateStorageSelection()
	case hud.IntentDeleteSlot:
		g.storageSelection = storageIndexForSlot(intent.Slot)
		g.deleteStorageSelection()
	case hud.IntentSetVolume:
		if !g.settingsLoading && intent.Volume != g.settings.MasterVolume {
			settings := g.settings
			settings.MasterVolume = intent.Volume
			g.updateUISettings(settings)
		}
	case hud.IntentToggleMute:
		g.toggleMute()
	case hud.IntentShowGuide:
		g.guide = ui.NewGuideState(false)
		if !g.settingsLoading {
			settings := g.settings
			settings.GuideDismissed = false
			g.updateUISettings(settings)
		}
	}
}

func storageIndexForSlot(slot int) int {
	for index, candidate := range storageBrowserSlots {
		if candidate == slot {
			return index
		}
	}
	return 0
}

// selectBandByID is the chip/list selection path; it applies the dirty-draft
// guard exactly like Tab.
func (g *Game) selectBandByID(id gameapi.BandID) {
	if id == g.selectedBand {
		return
	}
	if g.assignmentDraftDirty() {
		g.showNotice("Apply or discard workforce changes")
		return
	}
	for _, band := range g.frame.Bands {
		if band.ID != id {
			continue
		}
		g.selectedBand = id
		g.clearMigrationPreview()
		g.syncAssignmentDraft(true)
		g.refreshBandFieldNote()
		g.resetDisclosure()
		return
	}
}

// endTurn is the single End-turn path for Space and the button (spec §5.3).
func (g *Game) endTurn(force bool) {
	if g.frame == nil || g.frame.CampaignResult != gameapi.Ongoing {
		return
	}
	if g.assignmentDraftDirty() {
		g.showNotice("Apply or discard workforce changes before ending the turn")
		return
	}
	if g.hasMigrationPreview {
		g.showNotice("Press Enter to queue the migration, or Esc to clear it before ending the turn.")
		return
	}
	if waiting := ui.BandsNeedingMove(g.frame.Bands); waiting > 0 && !force && !g.endTurnArmed {
		g.endTurnArmed = true
		g.showNotice("Some bands have not moved. Click End turn again or press Space to end the turn anyway.")
		return
	}
	g.endTurnArmed = false
	g.dispatchBatch([]ui.Action{ui.EndTurnAction()})
}

func (g *Game) dismissGuide() {
	g.guide = g.guide.Dismiss()
	if g.settingsLoading {
		return
	}
	settings := g.settings
	settings.GuideDismissed = true
	g.updateUISettings(settings)
}
