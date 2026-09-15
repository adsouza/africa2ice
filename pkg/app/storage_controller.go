package app

import (
	"fmt"

	"github.com/adsouza/africa2ice/pkg/gameapi"
	"github.com/adsouza/africa2ice/pkg/ui"
)

// storageController owns host-side storage correlation and metadata. It never
// installs frames, navigates scenes, or plays sounds; completions describe those
// effects for the host to apply.
type storageController struct {
	resume              resumeState
	pendingQuickSaveIDs map[gameapi.StorageOpID]struct{}
	pendingSaveSoundIDs map[gameapi.StorageOpID]struct{}
	pendingManualLoadID gameapi.StorageOpID
	storageMode         storageBrowserMode
	storageSlots        []gameapi.SlotMetadata
	storageSelection    int
	storageListID       gameapi.StorageOpID
	storageOperationID  gameapi.StorageOpID
}

type resumePhase uint8

const (
	resumeIdle resumePhase = iota
	resumeListing
	resumeDispatching
	resumeLoading
)

type resumeState struct {
	phase resumePhase
	id    gameapi.StorageOpID
	slot  int
}

func (c *storageController) resumePending() bool { return c.resume.phase != resumeIdle }

func newStorageController() storageController {
	return storageController{pendingQuickSaveIDs: make(map[gameapi.StorageOpID]struct{}), pendingSaveSoundIDs: make(map[gameapi.StorageOpID]struct{})}
}

func (c *storageController) storageMetadata(slot int) (gameapi.SlotMetadata, bool) {
	for _, metadata := range c.storageSlots {
		if metadata.SlotID == slot {
			return metadata, true
		}
	}
	return gameapi.SlotMetadata{}, false
}

func (c *storageController) installStorageMetadata(metadata gameapi.SlotMetadata) {
	for index := range c.storageSlots {
		if c.storageSlots[index].SlotID != metadata.SlotID {
			continue
		}
		c.storageSlots[index] = metadata
		return
	}
	c.storageSlots = append(c.storageSlots, metadata)
}

func (c *storageController) removeStorageMetadata(slot int) {
	for index := range c.storageSlots {
		if c.storageSlots[index].SlotID != slot {
			continue
		}
		c.storageSlots = append(c.storageSlots[:index], c.storageSlots[index+1:]...)
		return
	}
}

// storageEffect contains only decisions that require host integration.
type storageEffect struct {
	startupLoad   bool
	resumeSlot    int
	notice        string
	failed        bool
	playSaveSound bool
}

func (effect *storageEffect) toast(notice string, failed bool) {
	effect.notice = notice
	effect.failed = failed
}

func (c *storageController) accept(result gameapi.StorageResult) storageEffect {
	effect := storageEffect{}
	_, saveSoundPending := c.pendingSaveSoundIDs[result.OperationID]
	delete(c.pendingSaveSoundIDs, result.OperationID)
	if result.Operation == gameapi.StorageSave && result.Slot == 99 {
		delete(c.pendingQuickSaveIDs, result.OperationID)
	}

	isStartupList := c.resume.phase == resumeListing && result.Operation == gameapi.StorageList && result.OperationID == c.resume.id
	isStartupLoad := c.resume.phase == resumeLoading && result.Operation == gameapi.StorageLoad && result.OperationID == c.resume.id
	// Both IDs use 0 for "nothing pending", so an unidentified completion must
	// not correlate with an idle slot the way isBrowserOperation already guards.
	isManualLoad := c.pendingManualLoadID != 0 && result.Operation == gameapi.StorageLoad && result.OperationID == c.pendingManualLoadID
	isBrowserList := c.storageListID != 0 && result.Operation == gameapi.StorageList && result.OperationID == c.storageListID
	isBrowserOperation := result.OperationID != 0 && result.OperationID == c.storageOperationID
	if isBrowserList {
		c.storageListID = 0
		if result.Err == nil {
			c.storageSlots = append(c.storageSlots[:0], result.Slots...)
		}
	}
	if isBrowserOperation {
		c.storageOperationID = 0
	}
	if isStartupList {
		c.resume = resumeState{}
		if result.Err == nil {
			// slot != 0 mirrors the host's own dispatch guard: entering the
			// dispatch phase for a slot nobody will dispatch strands resume,
			// and resumePending() then blocks input and first-draw forever.
			if slot, ok := newestResumeSlot(result.Slots); ok && slot != 0 {
				effect.resumeSlot = slot
				c.resume = resumeState{phase: resumeDispatching, slot: slot}
			}
		}
	}

	effect.startupLoad = isStartupLoad
	restoredSlot := c.resume.slot

	if isStartupLoad {
		c.resume = resumeState{}
	}
	if isManualLoad {
		c.pendingManualLoadID = 0
	}
	if result.Err != nil {
		effect.toast("Storage failed: "+ui.ErrorMessage(result.Err), true)
		return effect
	}
	if result.Metadata != nil {
		if result.Operation == gameapi.StorageDelete {
			c.removeStorageMetadata(result.Slot)
		} else {
			c.installStorageMetadata(*result.Metadata)
		}
	}
	switch {
	case isStartupLoad:
		if restoredSlot == 99 {
			effect.toast("Quick save restored", false)
		} else {
			effect.toast(fmt.Sprintf("Autosave restored — Auto %d", restoredSlot-100), false)
		}
	case isManualLoad:
		effect.toast("Loaded "+storageSlotLabel(result.Slot), false)
	case result.Operation == gameapi.StorageLoad:
		effect.toast("Game loaded", false)
	case result.Operation == gameapi.StorageSave && result.Slot >= 1 && result.Slot <= 3:
		if saveSoundPending {
			effect.playSaveSound = true
		}
		effect.toast(fmt.Sprintf("Saved Manual %d", result.Slot), false)
	case result.Operation == gameapi.StorageSave && result.Slot == 99:
		if saveSoundPending {
			effect.playSaveSound = true
		}
		effect.toast("Quick-saved", false)
	case result.Operation == gameapi.StorageSave && result.Slot >= 101 && result.Slot <= 103:
		effect.toast("Autosaved — "+storageSlotLabel(result.Slot), false)
	case result.Operation == gameapi.StorageDelete:
		effect.toast("Deleted "+storageSlotLabel(result.Slot), false)
	}
	return effect
}

func (c *storageController) resumeDispatched(slot int, id gameapi.StorageOpID, accepted bool) {
	if c.resume.phase != resumeDispatching || c.resume.slot != slot {
		return
	}
	c.resume = resumeState{}
	if accepted && id != 0 {
		c.resume = resumeState{phase: resumeLoading, id: id, slot: slot}
	}
}

func (c *storageController) trackSave(id gameapi.StorageOpID, slot int) {
	if slot == 99 {
		c.pendingQuickSaveIDs[id] = struct{}{}
	}
	if slot == 99 || slot >= 1 && slot <= 3 {
		c.pendingSaveSoundIDs[id] = struct{}{}
	}
}

func newestResumeSlot(slots []gameapi.SlotMetadata) (int, bool) {
	var newest gameapi.SlotMetadata
	found := false
	for _, slot := range slots {
		if slot.SlotKind != gameapi.QuickSlot && slot.SlotKind != gameapi.AutoSlot {
			continue
		}
		if !found || slot.CommitSequence > newest.CommitSequence || slot.CommitSequence == newest.CommitSequence && slot.SlotID < newest.SlotID {
			newest = slot
			found = true
		}
	}
	return newest.SlotID, found
}

func (c *storageController) beginResume(id gameapi.StorageOpID) {
	if id != 0 && !c.resumePending() {
		c.resume = resumeState{phase: resumeListing, id: id}
	}
}

func (c *storageController) openBrowser(mode storageBrowserMode, id gameapi.StorageOpID) {
	c.storageMode = mode
	c.storageSelection = 0
	c.storageSlots = nil
	c.storageListID = id
}

func (c *storageController) moveSelection(delta int) {
	count := len(storageBrowserSlots)
	c.storageSelection = (c.storageSelection + delta + count) % count
}

func (c *storageController) busy() bool { return c.storageListID != 0 || c.storageOperationID != 0 }

func (c *storageController) beginManualLoad(id gameapi.StorageOpID) { c.pendingManualLoadID = id }
func (c *storageController) beginBrowserOperation(id gameapi.StorageOpID, load bool) {
	c.storageOperationID = id
	if load {
		c.beginManualLoad(id)
	}
}
func (c *storageController) selectSlot(slot int) { c.storageSelection = storageIndexForSlot(slot) }
