package app

import (
	"errors"
	"testing"

	"github.com/adsouza/africa2ice/pkg/gameapi"
)

func TestStorageControllerKeepsBrowserAndStartupOperationsIndependent(t *testing.T) {
	c := newStorageController()
	c.beginResume(10)
	c.openBrowser(storageBrowserLoad, 11)
	effect := c.accept(gameapi.StorageResult{OperationID: 11, Operation: gameapi.StorageList, Slots: []gameapi.SlotMetadata{{SlotID: 1}}})
	if effect.resumeSlot != 0 || !c.resumePending() || c.storageListID != 0 {
		t.Fatal("browser completion advanced startup lifecycle")
	}
	effect = c.accept(gameapi.StorageResult{OperationID: 10, Operation: gameapi.StorageList, Slots: []gameapi.SlotMetadata{
		{SlotID: 99, SlotKind: gameapi.QuickSlot, CommitSequence: 3},
		{SlotID: 102, SlotKind: gameapi.AutoSlot, CommitSequence: 4},
	}})
	if effect.resumeSlot != 102 || !c.resumePending() {
		t.Fatal("startup did not request the newest resume slot")
	}
	if _, ok := c.storageMetadata(1); !ok {
		t.Fatal("startup listing replaced browser metadata")
	}
	c.resumeDispatched(effect.resumeSlot, 12, true)
	effect = c.accept(gameapi.StorageResult{OperationID: 12, Operation: gameapi.StorageLoad, Slot: 102, Err: errors.New("corrupt save")})
	if !effect.startupLoad || !effect.failed || effect.notice == "" || c.resumePending() || c.resume.id != 0 || c.resume.slot != 0 {
		t.Fatal("failed resume did not settle its lifecycle")
	}
}

func TestStorageControllerRejectedResumeAndUnrelatedCompletion(t *testing.T) {
	c := newStorageController()
	c.beginResume(10)
	effect := c.accept(gameapi.StorageResult{OperationID: 9, Operation: gameapi.StorageList})
	if effect.resumeSlot != 0 || !c.resumePending() || c.resume.id != 10 {
		t.Fatal("unrelated list released startup guard")
	}
	effect = c.accept(gameapi.StorageResult{OperationID: 10, Operation: gameapi.StorageList, Slots: []gameapi.SlotMetadata{{SlotID: 99, SlotKind: gameapi.QuickSlot}}})
	c.resumeDispatched(effect.resumeSlot, 0, false)
	if c.resumePending() || c.resume.id != 0 {
		t.Fatal("rejected resume left startup blocked")
	}
}

func TestStorageControllerSaveEffectsFollowTrackedCompletion(t *testing.T) {
	c := newStorageController()
	c.trackSave(20, 99)
	effect := c.accept(gameapi.StorageResult{OperationID: 21, Operation: gameapi.StorageSave, Slot: 101})
	if effect.playSaveSound || len(c.pendingQuickSaveIDs) != 1 {
		t.Fatal("autosave consumed quick-save tracking")
	}
	effect = c.accept(gameapi.StorageResult{OperationID: 20, Operation: gameapi.StorageSave, Slot: 99, Err: errors.New("disk full")})
	if effect.playSaveSound || !effect.failed || len(c.pendingQuickSaveIDs) != 0 || len(c.pendingSaveSoundIDs) != 0 {
		t.Fatal("failed save played success sound or left close guard active")
	}
	c.trackSave(22, 1)
	effect = c.accept(gameapi.StorageResult{OperationID: 22, Operation: gameapi.StorageSave, Slot: 1, Metadata: &gameapi.SlotMetadata{SlotID: 1, Turn: 5}})
	if !effect.playSaveSound || effect.failed {
		t.Fatal("successful manual save lost its completion effect")
	}
	if metadata, ok := c.storageMetadata(1); !ok || metadata.Turn != 5 {
		t.Fatal("save metadata not installed")
	}
	duplicate := c.accept(gameapi.StorageResult{OperationID: 22, Operation: gameapi.StorageSave, Slot: 1})
	if duplicate.playSaveSound {
		t.Fatal("duplicate completion replayed success sound")
	}
}

func TestResumeTransitionsRejectUnidentifiedAndStaleCompletions(t *testing.T) {
	c := newStorageController()
	c.beginResume(0)
	if c.resumePending() {
		t.Fatal("zero operation started resume")
	}
	c.beginResume(10)
	c.beginResume(11)
	if c.resume.id != 10 {
		t.Fatal("second start overwrote pending operation")
	}
	for _, operation := range []gameapi.StorageOperation{gameapi.StorageLoad, gameapi.StorageList} {
		effect := c.accept(gameapi.StorageResult{Operation: operation})
		if effect.startupLoad || effect.resumeSlot != 0 || c.resume.phase != resumeListing {
			t.Fatal("unidentified completion advanced resume")
		}
	}
	effect := c.accept(gameapi.StorageResult{OperationID: 10, Operation: gameapi.StorageList, Slots: []gameapi.SlotMetadata{{SlotID: 99, SlotKind: gameapi.QuickSlot}}})
	if effect.resumeSlot != 99 || c.resume.phase != resumeDispatching {
		t.Fatal("list did not enter dispatch phase")
	}
	c.resumeDispatched(101, 12, true)
	if c.resume.phase != resumeDispatching {
		t.Fatal("wrong slot advanced resume")
	}
	c.resumeDispatched(99, 12, true)
	c.resumeDispatched(99, 13, true)
	c.accept(gameapi.StorageResult{OperationID: 10, Operation: gameapi.StorageList})
	if c.resume.phase != resumeLoading || c.resume.id != 12 {
		t.Fatal("stale callback changed in-flight load")
	}
	effect = c.accept(gameapi.StorageResult{OperationID: 12, Operation: gameapi.StorageLoad, Slot: 99})
	if !effect.startupLoad || effect.notice != "Quick save restored" || c.resume != (resumeState{}) {
		t.Fatal("load did not settle resume completely")
	}
}

// TestAcceptedResumeWithoutOperationIDSettles covers the branch the suite above
// only ever takes one way: an accepted dispatch that yields no usable id. Entering
// resumeLoading with id 0 would wait on a completion no operation can produce, and
// resumePending() blocks input and first-draw for the rest of the session.
func TestAcceptedResumeWithoutOperationIDSettles(t *testing.T) {
	c := newStorageController()
	c.beginResume(10)
	effect := c.accept(gameapi.StorageResult{OperationID: 10, Operation: gameapi.StorageList, Slots: []gameapi.SlotMetadata{{SlotID: 99, SlotKind: gameapi.QuickSlot}}})
	if effect.resumeSlot != 99 || c.resume.phase != resumeDispatching {
		t.Fatal("list did not enter dispatch phase")
	}
	c.resumeDispatched(99, 0, true)
	if c.resumePending() || c.resume != (resumeState{}) {
		t.Fatalf("accepted dispatch without an id left resume %+v", c.resume)
	}
}

// TestResumeIgnoresUndispatchableSlot pins the controller's entry condition to the
// host's dispatch condition: pollStorage only calls resumeDispatched when
// effect.resumeSlot is nonzero, so the controller must not enter the dispatch
// phase for a slot that guard would skip.
func TestResumeIgnoresUndispatchableSlot(t *testing.T) {
	c := newStorageController()
	c.beginResume(10)
	effect := c.accept(gameapi.StorageResult{OperationID: 10, Operation: gameapi.StorageList, Slots: []gameapi.SlotMetadata{{SlotID: 0, SlotKind: gameapi.QuickSlot}}})
	if effect.resumeSlot != 0 {
		t.Fatalf("host would dispatch slot %d", effect.resumeSlot)
	}
	if c.resumePending() {
		t.Fatalf("resume stranded in %v with no dispatch to come", c.resume.phase)
	}
}
