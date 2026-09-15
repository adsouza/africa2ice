package app

import (
	"fmt"

	gameaudio "github.com/adsouza/africa2ice/pkg/audio"
	"github.com/adsouza/africa2ice/pkg/gameapi"
	"github.com/adsouza/africa2ice/pkg/ui"
)

type storageBrowserMode uint8

const (
	storageBrowserSave storageBrowserMode = iota
	storageBrowserLoad
)

var storageBrowserSlots = [...]int{1, 2, 3, 99, 101, 102, 103}

func (g *Game) beginQuickSave() {
	g.dispatchBatch([]ui.Action{ui.SaveAction(99)})
}

func (g *Game) beginManualSave(slot int) {
	if _, accepted := g.dispatchBatch([]ui.Action{ui.SaveAction(slot)}); !accepted {
		return
	}
	g.showNotice(fmt.Sprintf("Saving Manual %d…", slot))
}

func (g *Game) beginManualLoad(slot int) {
	if g.workforce.Dirty() {
		g.showNotice("Apply or discard workforce changes before loading")
		return
	}
	result, accepted := g.dispatchBatch([]ui.Action{ui.LoadAction(slot)})
	if !accepted || result.operationID == 0 {
		return
	}
	g.storage.beginManualLoad(result.operationID)
	g.showNotice(fmt.Sprintf("Loading Manual %d…", slot))
}

func (g *Game) openStorageBrowser(mode storageBrowserMode) {
	result, accepted := g.dispatchBatch([]ui.Action{ui.ListSlotsAction()})
	if !accepted {
		return
	}
	g.storage.openBrowser(mode, result.operationID)
	g.dispatchBatch([]ui.Action{ui.PushSceneAction(ui.SceneStorage)})
}

func (g *Game) moveStorageSelection(delta int) {
	g.storage.moveSelection(delta)
}

func (g *Game) activateStorageSelection() {
	if g.storage.busy() {
		return
	}
	slot := storageBrowserSlots[g.storage.storageSelection]
	if g.storage.storageMode == storageBrowserSave {
		if slot < 1 || slot > 3 {
			g.showNotice("Only Manual 1–3 can be overwritten from the Save list")
			return
		}
		result, accepted := g.dispatchBatch([]ui.Action{ui.SaveAction(slot)})
		if !accepted {
			return
		}
		g.storage.beginBrowserOperation(result.operationID, false)
		g.showNotice(fmt.Sprintf("Saving Manual %d…", slot))
		return
	}
	if _, ok := g.storage.storageMetadata(slot); !ok {
		g.showNotice("That slot is empty")
		return
	}
	if g.workforce.Dirty() {
		g.showNotice("Apply or discard workforce changes before loading")
		return
	}
	result, accepted := g.dispatchBatch([]ui.Action{ui.LoadAction(slot)})
	if !accepted || result.operationID == 0 {
		return
	}
	g.storage.beginBrowserOperation(result.operationID, true)
	g.showNotice("Loading " + storageSlotLabel(slot) + "…")
}

func (g *Game) deleteStorageSelection() {
	if g.storage.busy() {
		return
	}
	slot := storageBrowserSlots[g.storage.storageSelection]
	if _, ok := g.storage.storageMetadata(slot); !ok {
		g.showNotice("That slot is already empty")
		return
	}
	result, accepted := g.dispatchBatch([]ui.Action{ui.DeleteAction(slot)})
	if !accepted {
		return
	}
	g.storage.beginBrowserOperation(result.operationID, false)
	g.showNotice("Deleting " + storageSlotLabel(slot) + "…")
}

func storageSlotLabel(slot int) string {
	switch slot {
	case 1, 2, 3:
		return fmt.Sprintf("Manual %d", slot)
	case 99:
		return "Quick Save"
	case 101, 102, 103:
		return fmt.Sprintf("Auto %d", slot-100)
	default:
		return fmt.Sprintf("Slot %d", slot)
	}
}

func (g *Game) beginStartupResume() {
	// Ask before spending the operation: beginResume declines a second
	// resume, and a listing it declines would run untracked.
	if g.storage.resumePending() {
		return
	}
	result, accepted := g.dispatchBatch([]ui.Action{ui.ListSlotsAction()})
	if !accepted || result.operationID == 0 {
		return
	}
	g.storage.beginResume(result.operationID)
}

func (g *Game) pollStorage() {
	for _, result := range g.port.PollStorage() {
		effect := g.storage.accept(result)
		if effect.resumeSlot != 0 {
			loadResult, accepted := g.dispatchBatch([]ui.Action{ui.LoadAction(effect.resumeSlot)})
			g.storage.resumeDispatched(effect.resumeSlot, loadResult.operationID, accepted)
		}
		if result.ReplacementFrame != nil {
			g.installStoredFrame(result.ReplacementFrame, effect.startupLoad)
		}
		if effect.playSaveSound {
			g.sound.Play(gameaudio.SFXSaveComplete)
		}
		if effect.notice != "" {
			g.queueToast(effect.notice, effect.failed)
		}
	}
}

func (g *Game) installStoredFrame(frame *gameapi.Frame, startup bool) {
	g.frame = frame
	g.syncEasyMode()
	g.publishFrame()
	g.pendingLakeNotes = nil
	g.setFieldNote(ui.CampaignOverviewFieldNote())
	g.breakthroughFrames = 0
	g.regionalPulseFocused = false
	g.clearMigrationPreview()
	g.selectedBand = 0
	g.ensureSelection()
	g.workforce.Clear()
	g.syncAssignmentDraft(true)
	g.resetDisclosure()
	g.scenes.Reset()
	if startup {
		g.scenes.Push(ui.SceneTitle)
	}
}
