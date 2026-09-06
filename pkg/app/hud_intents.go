package app

import (
	"github.com/adsouza/africa2ice/pkg/gameapi"
	"github.com/adsouza/africa2ice/pkg/hud"
	"github.com/adsouza/africa2ice/pkg/ui"
)

// handleIntents maps chrome clicks onto the same guarded methods hotkeys use.
func (g *Game) handleIntents(intents []hud.Intent) {
	for _, intent := range intents {
		g.logSession.LogUIIntent(intent.Kind.String(), uiIntentLogAttributes(intent)...)
		g.handleIntent(intent)
	}
}

// uiIntentLogAttributes returns only the payload fields the given kind
// actually uses (see handleIntent/handleOverlayIntent below), each a
// bounded scalar, for internal/adapters/logging's LogUIIntent. It must not
// grow to include a frame, a slice, or anything per-frame.
func uiIntentLogAttributes(intent hud.Intent) []any {
	switch intent.Kind {
	case hud.IntentSelectBand:
		return []any{"band", uint64(intent.Band)}
	case hud.IntentOpenRow:
		return []any{"row", uint8(intent.Row)}
	case hud.IntentSetNotesMode:
		return []any{"notes", uint8(intent.Notes)}
	case hud.IntentMoveTo:
		return []any{"tile", uint64(intent.Tile)}
	case hud.IntentInterbreed:
		return []any{"band", uint64(intent.Band)}
	case hud.IntentChooseResearch:
		return []any{"tech", uint8(intent.Tech)}
	case hud.IntentAdjustRole:
		return []any{"role", uint8(intent.Role), "delta", intent.Delta}
	case hud.IntentEndTurn:
		return []any{"force", intent.Force}
	case hud.IntentFocusTrait:
		return []any{"trait", uint8(intent.Trait)}
	case hud.IntentFocusEvent:
		return []any{"event", uint8(intent.Event)}
	case hud.IntentFocusInterbreedPartner:
		return []any{"band", uint64(intent.Band)}
	case hud.IntentOpenStorage:
		return []any{"save", intent.Save}
	case hud.IntentSaveSlot, hud.IntentLoadSlot, hud.IntentDeleteSlot:
		return []any{"slot", intent.Slot}
	case hud.IntentSetVolume:
		return []any{"volume", intent.Volume}
	default:
		return nil
	}
}

// bandActionIntent reports whether a kind asks the domain to change the
// campaign. Every one of these is refused once the campaign is over (spec
// §5.3); overlay navigation and IntentNewCampaign still work, so the
// epilogue's own controls keep responding.
func bandActionIntent(kind hud.IntentKind) bool {
	switch kind {
	case hud.IntentMoveTo, hud.IntentMoveToBest, hud.IntentSplit, hud.IntentInterbreed,
		hud.IntentChooseResearch, hud.IntentAdjustRole, hud.IntentApplyWorkforce, hud.IntentEndTurn:
		return true
	default:
		return false
	}
}

func (g *Game) handleIntent(intent hud.Intent) {
	if g.frame != nil && g.frame.CampaignResult != gameapi.Ongoing && bandActionIntent(intent.Kind) {
		g.logSession.LogUIIntentRefused(intent.Kind.String(), "campaign-over")
		return
	}
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
		g.toggleDetails()
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
				// traitFocus is always "the next variant G will show", the
				// same meaning focusNextTraitNote leaves it in after
				// displaying a note — so a click sets it one past the trait
				// just displayed, not to that trait itself.
				g.traitFocus = (intent.Trait + 1) % gameapi.HeritableTraitCount
				g.setFieldNote(note)
			}
		}
	case hud.IntentFocusEvent:
		if note, ok := ui.EventKindFieldNote(intent.Event); ok {
			g.setFieldNote(note)
		}
	case hud.IntentFocusInterbreedPartner:
		g.focusInterbreedPartner(intent.Band)
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
			if g.tryQueueMigration(band, candidate.TileID) {
				// The game chose this destination, not the player; leave the
				// Move row open and pinned so its queued summary stays
				// visible instead of auto-advancing to the next row.
				g.openRow, g.rowChosen = ui.RowMove, true
			}
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
	case hud.IntentToggleReducedMotion:
		g.toggleReducedMotion()
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
