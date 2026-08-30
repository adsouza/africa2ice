package app

import (
	"errors"
	"testing"

	"github.com/adsouza/africa2ice/pkg/gameapi"
	"github.com/adsouza/africa2ice/pkg/ui"
)

type gameStub struct {
	frame          *gameapi.Frame
	results        []gameapi.StorageResult
	nextStorageID  gameapi.StorageOpID
	loadedSlot     int
	savedSlot      int
	appliedCommand gameapi.Command
	newCampaigns   int
}

func (stub *gameStub) Snapshot() (*gameapi.Frame, error) { return stub.frame, nil }

func (stub *gameStub) NewCampaign() (*gameapi.Frame, error) {
	stub.newCampaigns++
	return stub.frame, nil
}

func (stub *gameStub) Apply(command gameapi.Command) (*gameapi.Frame, error) {
	stub.appliedCommand = command
	return stub.frame, nil
}

func (stub *gameStub) EndTurn() (*gameapi.Frame, error) { return stub.frame, nil }

func (stub *gameStub) BeginSave(slot int) (gameapi.StorageOpID, error) {
	stub.savedSlot = slot
	return stub.nextID(), nil
}

func (stub *gameStub) BeginLoad(slot int) (gameapi.StorageOpID, error) {
	stub.loadedSlot = slot
	return stub.nextID(), nil
}

func (stub *gameStub) BeginDelete(int) (gameapi.StorageOpID, error) {
	return 0, errors.New("unexpected delete")
}

func (stub *gameStub) BeginListSlots() (gameapi.StorageOpID, error) { return stub.nextID(), nil }

func (stub *gameStub) PollStorage() []gameapi.StorageResult {
	results := stub.results
	stub.results = nil
	return results
}

func (stub *gameStub) nextID() gameapi.StorageOpID {
	stub.nextStorageID++
	return stub.nextStorageID
}

func TestKeyboardMigrationPreviewCanTurnIntoReachableCorner(t *testing.T) {
	frame := migrationPreviewFrame()
	stub := &gameStub{frame: frame}
	game := New(stub)

	game.handleDirectionalMigration(0, -1)
	if !game.hasMigrationPreview || game.migrationPreviewTile != 1 || stub.appliedCommand != nil {
		t.Fatalf("north preview = (visible %t, tile %d, command %T), want blocked tile 1 without a command", game.hasMigrationPreview, game.migrationPreviewTile, stub.appliedCommand)
	}

	game.handleDirectionalMigration(-1, 0)
	if game.migrationPreviewTile != 2 {
		t.Fatalf("turned preview tile = %d, want northwest corner 2", game.migrationPreviewTile)
	}
	game.confirmMigrationPreview()
	command, ok := stub.appliedCommand.(gameapi.QueueMigration)
	if !ok || command.BandID != 7 || command.TileID != 2 {
		t.Fatalf("applied command = %#v, want QueueMigration for band 7 to tile 2", stub.appliedCommand)
	}
	if game.hasMigrationPreview {
		t.Fatal("confirmed migration left the keyboard preview active")
	}
}

func TestKeyboardMigrationDoesNotReplaceQueuedSpatialAction(t *testing.T) {
	frame := migrationPreviewFrame()
	frame.Bands[0].SpatialActionUsed = true
	stub := &gameStub{frame: frame}
	game := New(stub)

	game.handleDirectionalMigration(0, -1)
	if game.hasMigrationPreview || stub.appliedCommand != nil {
		t.Fatalf("spent spatial action created preview=%t or command=%T", game.hasMigrationPreview, stub.appliedCommand)
	}
	if game.notice != "This band has already used its spatial action this turn." {
		t.Fatalf("spent-action notice = %q", game.notice)
	}
}

func TestBandSelectionCyclesForwardAndBackwardWithWraparound(t *testing.T) {
	frame := migrationPreviewFrame()
	frame.Bands = append(frame.Bands,
		gameapi.Band{ID: 8, Species: gameapi.ArchaicHominin},
		gameapi.Band{ID: 9, Species: gameapi.HomoSapiens},
		gameapi.Band{ID: 10, Species: gameapi.ArchaicHominin},
		gameapi.Band{ID: 11, Species: gameapi.HomoSapiens},
	)
	game := New(&gameStub{frame: frame})

	game.selectPreviousSapiens()
	if game.selectedBand != 11 {
		t.Fatalf("previous selection from first = %d, want last sapiens band 11", game.selectedBand)
	}
	game.selectNextSapiens()
	if game.selectedBand != 7 {
		t.Fatalf("next selection from last = %d, want first sapiens band 7", game.selectedBand)
	}
	game.selectNextSapiens()
	if game.selectedBand != 9 {
		t.Fatalf("next selection = %d, want sapiens band 9", game.selectedBand)
	}
	game.selectPreviousSapiens()
	if game.selectedBand != 7 {
		t.Fatalf("previous selection = %d, want sapiens band 7", game.selectedBand)
	}
}

func TestCompletedTurnCelebratesNewTechnologyWithoutOverridingPanelVisibility(t *testing.T) {
	before := migrationPreviewFrame()
	game := New(&gameStub{frame: before})
	game.fieldNotesVisible = false
	after := *before
	after.Turn = 1
	after.Bands = append([]gameapi.Band(nil), before.Bands...)
	after.Bands[0].AcquiredTech = 1 << gameapi.Firecraft

	game.acceptCompletedTurn(&after)
	if game.fieldNote.Topic != gameapi.Firecraft.String() || game.breakthroughFrames != breakthroughCelebrationFrames {
		t.Fatalf("breakthrough note = %#v for %d frames", game.fieldNote, game.breakthroughFrames)
	}
	if game.fieldNotesVisible {
		t.Fatal("technology celebration overrode the player's hidden Field Notes choice")
	}
	if game.notice != "Breakthrough! Band 7 learned Firecraft" || game.noticeFrames != 300 {
		t.Fatalf("breakthrough notice = %q for %d frames", game.notice, game.noticeFrames)
	}

	game.breakthroughFrames = 0
	game.acceptCompletedTurn(&after)
	if game.breakthroughFrames != 0 {
		t.Fatal("accepting an unchanged acquired-tech frame replayed the celebration")
	}
}

func TestTechnologyDiscoveryIgnoresArchaicLearningAndInheritedTechOnNewBands(t *testing.T) {
	before := migrationPreviewFrame()
	before.Bands = append(before.Bands,
		gameapi.Band{ID: 8, Species: gameapi.ArchaicHominin},
		gameapi.Band{ID: 9, Species: gameapi.HomoSapiens},
	)
	after := *before
	after.Bands = append([]gameapi.Band(nil), before.Bands...)
	after.Bands[0].AcquiredTech = 1 << gameapi.HaftedTools
	after.Bands[1].AcquiredTech = 1 << gameapi.Firecraft
	after.Bands[2].AcquiredTech = 1 << gameapi.PlantKnowledge
	after.Bands = append(after.Bands, gameapi.Band{ID: 10, Species: gameapi.HomoSapiens, AcquiredTech: 1 << gameapi.Firecraft})

	discoveries := newTechnologyDiscoveries(before, &after, 9)
	if len(discoveries) != 2 || discoveries[0] != (technologyDiscovery{bandID: 9, technology: gameapi.PlantKnowledge}) || discoveries[1] != (technologyDiscovery{bandID: 7, technology: gameapi.HaftedTools}) {
		t.Fatalf("technology discoveries = %#v", discoveries)
	}
}

func TestStartupResumeLoadsNewestQuickOrAutosave(t *testing.T) {
	initial := migrationPreviewFrame()
	restored := migrationPreviewFrame()
	restored.Turn = 3
	restored.Bands[0].TileID = 2
	stub := &gameStub{frame: initial}
	game := New(stub)
	game.fieldNote, _ = ui.TechnologyFieldNote(gameapi.Firecraft, 7, 1)
	game.breakthroughFrames = breakthroughCelebrationFrames

	game.beginStartupResume()
	listID := game.startupRestoreListID
	stub.results = []gameapi.StorageResult{{
		OperationID: listID,
		Operation:   gameapi.StorageList,
		Slots: []gameapi.SlotMetadata{
			{SlotID: 1, SlotKind: gameapi.ManualSlot, CommitSequence: 99, Turn: 8},
			{SlotID: 99, SlotKind: gameapi.QuickSlot, CommitSequence: 10, Turn: 2},
			{SlotID: 101, SlotKind: gameapi.AutoSlot, CommitSequence: 11, Turn: 3},
			{SlotID: 102, SlotKind: gameapi.AutoSlot, CommitSequence: 9, Turn: 1},
		},
	}}
	game.pollStorage()
	if stub.loadedSlot != 101 || game.startupRestoreLoadID == 0 || !game.startupRestorePending {
		t.Fatalf("startup list did not load newest resume slot: slot=%d loadID=%d pending=%t", stub.loadedSlot, game.startupRestoreLoadID, game.startupRestorePending)
	}

	stub.results = []gameapi.StorageResult{{
		OperationID:      game.startupRestoreLoadID,
		Operation:        gameapi.StorageLoad,
		Slot:             101,
		ReplacementFrame: restored,
	}}
	game.pollStorage()
	if game.startupRestorePending || game.frame.Turn != 3 || game.frame.Bands[0].TileID != 2 {
		t.Fatalf("restored state = pending %t, turn %d, tile %d", game.startupRestorePending, game.frame.Turn, game.frame.Bands[0].TileID)
	}
	if game.notice != "Autosave restored — Auto 1" {
		t.Fatalf("restore notice = %q", game.notice)
	}
	if game.fieldNote.Topic != "WELCOME" || game.breakthroughFrames != 0 {
		t.Fatalf("loaded game retained a stale breakthrough: %#v for %d frames", game.fieldNote, game.breakthroughFrames)
	}
}

func TestNewestResumeSlotUsesCommitSequenceAndExcludesManualSaves(t *testing.T) {
	tests := []struct {
		name  string
		slots []gameapi.SlotMetadata
		want  int
		ok    bool
	}{
		{name: "empty", ok: false},
		{name: "manual only", slots: []gameapi.SlotMetadata{{SlotID: 3, SlotKind: gameapi.ManualSlot, CommitSequence: 100}}, ok: false},
		{name: "quick newest", slots: []gameapi.SlotMetadata{{SlotID: 101, SlotKind: gameapi.AutoSlot, CommitSequence: 4}, {SlotID: 99, SlotKind: gameapi.QuickSlot, CommitSequence: 5}}, want: 99, ok: true},
		{name: "autosave newest", slots: []gameapi.SlotMetadata{{SlotID: 99, SlotKind: gameapi.QuickSlot, CommitSequence: 5}, {SlotID: 103, SlotKind: gameapi.AutoSlot, CommitSequence: 6}}, want: 103, ok: true},
		{name: "deterministic corrupt tie", slots: []gameapi.SlotMetadata{{SlotID: 102, SlotKind: gameapi.AutoSlot, CommitSequence: 7}, {SlotID: 101, SlotKind: gameapi.AutoSlot, CommitSequence: 7}}, want: 101, ok: true},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got, ok := newestResumeSlot(test.slots)
			if got != test.want || ok != test.ok {
				t.Fatalf("newestResumeSlot() = (%d, %t), want (%d, %t)", got, ok, test.want, test.ok)
			}
		})
	}
}

func TestStartupResumeQuietlyKeepsNewGameWithoutResumeSave(t *testing.T) {
	stub := &gameStub{frame: migrationPreviewFrame()}
	game := New(stub)
	game.beginStartupResume()
	stub.results = []gameapi.StorageResult{{OperationID: game.startupRestoreListID, Operation: gameapi.StorageList}}

	game.pollStorage()
	if game.startupRestorePending || stub.loadedSlot != 0 {
		t.Fatalf("empty slot list = pending %t, loaded slot %d", game.startupRestorePending, stub.loadedSlot)
	}
}

func TestQuickSaveRemainsPendingUntilItsCompletion(t *testing.T) {
	stub := &gameStub{frame: migrationPreviewFrame()}
	game := New(stub)
	game.beginQuickSave()
	if stub.savedSlot != 99 || len(game.pendingQuickSaveIDs) != 1 {
		t.Fatalf("quick-save start = slot %d, pending %d", stub.savedSlot, len(game.pendingQuickSaveIDs))
	}
	operationID := stub.nextStorageID
	stub.results = []gameapi.StorageResult{{OperationID: operationID, Operation: gameapi.StorageSave, Slot: 99}}

	game.pollStorage()
	if len(game.pendingQuickSaveIDs) != 0 {
		t.Fatal("completed quick-save still blocks shutdown")
	}
}

func TestStartingNewCampaignReplacesTerminalPresentationState(t *testing.T) {
	terminal := migrationPreviewFrame()
	terminal.CampaignResult = gameapi.DispersalFailed
	terminal.Turn = 400
	stub := &gameStub{frame: terminal}
	game := New(stub)
	game.fieldNote, _ = ui.TechnologyFieldNote(gameapi.Firecraft, 7, 1)
	game.breakthroughFrames = breakthroughCelebrationFrames
	game.hasMigrationPreview = true

	fresh := migrationPreviewFrame()
	stub.frame = fresh
	game.startNewCampaign()

	if stub.newCampaigns != 1 || game.frame != fresh || game.frame.CampaignResult != gameapi.Ongoing {
		t.Fatalf("new campaign = calls %d, frame %#v", stub.newCampaigns, game.frame)
	}
	if game.fieldNote.Topic != "WELCOME" || game.breakthroughFrames != 0 || game.hasMigrationPreview {
		t.Fatalf("new campaign retained stale presentation: note %#v, breakthrough %d, preview %t", game.fieldNote, game.breakthroughFrames, game.hasMigrationPreview)
	}
	if game.notice != "New campaign begun" || game.selectedBand != 7 {
		t.Fatalf("new campaign notice/selection = %q/%d", game.notice, game.selectedBand)
	}
}

func migrationPreviewFrame() *gameapi.Frame {
	return &gameapi.Frame{
		CampaignResult: gameapi.Ongoing,
		Tiles: []gameapi.Tile{
			{ID: 0, X: 0, Y: 0, Land: true, Explored: true, BaselineK: 100},
			{ID: 1, X: 0, Y: -1, Land: true, Explored: true},
			{ID: 2, X: -1, Y: -1, Land: true, Explored: true, BaselineK: 100},
		},
		Bands: []gameapi.Band{{
			ID: 7, Species: gameapi.HomoSapiens, TileID: 0,
			MigrationCandidates: []gameapi.MigrationCandidate{{TileID: 2}},
		}},
	}
}

var _ gameapi.Game = (*gameStub)(nil)
