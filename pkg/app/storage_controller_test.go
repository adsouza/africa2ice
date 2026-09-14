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
	if effect.resumeSlot != 0 || !c.startupRestorePending || c.storageListID != 0 {
		t.Fatal("browser completion advanced startup lifecycle")
	}
	effect = c.accept(gameapi.StorageResult{OperationID: 10, Operation: gameapi.StorageList, Slots: []gameapi.SlotMetadata{
		{SlotID: 99, SlotKind: gameapi.QuickSlot, CommitSequence: 3},
		{SlotID: 102, SlotKind: gameapi.AutoSlot, CommitSequence: 4},
	}})
	if effect.resumeSlot != 102 || !c.startupRestorePending {
		t.Fatal("startup did not request the newest resume slot")
	}
	if _, ok := c.storageMetadata(1); !ok {
		t.Fatal("startup listing replaced browser metadata")
	}
	c.resumeDispatched(effect.resumeSlot, 12, true)
	effect = c.accept(gameapi.StorageResult{OperationID: 12, Operation: gameapi.StorageLoad, Slot: 102, Err: errors.New("corrupt save")})
	if !effect.startupLoad || !effect.failed || effect.notice == "" || c.startupRestorePending || c.startupRestoreLoadID != 0 || c.startupRestoreSlot != 0 {
		t.Fatal("failed resume did not settle its lifecycle")
	}
}

func TestStorageControllerRejectedResumeAndUnrelatedCompletion(t *testing.T) {
	c := newStorageController()
	c.beginResume(10)
	effect := c.accept(gameapi.StorageResult{OperationID: 9, Operation: gameapi.StorageList})
	if effect.resumeSlot != 0 || !c.startupRestorePending || c.startupRestoreListID != 10 {
		t.Fatal("unrelated list released startup guard")
	}
	effect = c.accept(gameapi.StorageResult{OperationID: 10, Operation: gameapi.StorageList, Slots: []gameapi.SlotMetadata{{SlotID: 99, SlotKind: gameapi.QuickSlot}}})
	c.resumeDispatched(effect.resumeSlot, 0, false)
	if c.startupRestorePending || c.startupRestoreLoadID != 0 {
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
