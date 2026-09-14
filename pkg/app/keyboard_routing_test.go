package app

import (
	"testing"

	"github.com/adsouza/africa2ice/pkg/gameapi"
	"github.com/adsouza/africa2ice/pkg/ui"
	"github.com/hajimehoshi/ebiten/v2"
)

func keysDown(keys ...ebiten.Key) func(ebiten.Key) bool {
	return func(key ebiten.Key) bool {
		for _, candidate := range keys {
			if key == candidate {
				return true
			}
		}
		return false
	}
}

func TestKeyboardModalBlocksRealCommandCombinations(t *testing.T) {
	stub := &gameStub{frame: migrationPreviewFrame()}
	game := New(stub)
	game.shortcutsOpen = true
	game.openRow = ui.RowWorkforce
	game.handleGameplayKeyState(keysDown(), keysDown(ebiten.KeySpace, ebiten.KeyTab, ebiten.KeyB, ebiten.KeyArrowRight))
	if stub.endTurns != 0 || len(stub.appliedCommands) != 0 || game.workforce.Dirty() {
		t.Fatal("commands escaped shortcut modal")
	}
	game.handleGameplayKeyState(keysDown(ebiten.KeyShift), keysDown(ebiten.KeySlash, ebiten.KeySpace))
	if game.shortcutsOpen || stub.endTurns != 0 {
		t.Fatal("closing shortcut modal also dispatched a turn")
	}
	game.handleGameplayKeyState(keysDown(), keysDown(ebiten.KeySpace))
	if stub.endTurns != 1 {
		t.Fatal("Space did not resume after modal closed")
	}
}

func TestKeyboardDraftOwnershipAndGlobalPriority(t *testing.T) {
	stub := &gameStub{frame: migrationPreviewFrame()}
	game := New(stub)
	game.openRow = ui.RowWorkforce
	game.workforce.Role = gameapi.Foraging
	game.handleGameplayKeyState(keysDown(ebiten.KeyShift), keysDown(ebiten.KeyArrowRight))
	if game.workforce.Allocation()[gameapi.Foraging] != 500 {
		t.Fatal("Shift+Right did not edit row by 500 BP")
	}
	draft, band := game.workforce.Allocation(), game.selectedBand
	game.handleGameplayKeyState(keysDown(), keysDown(ebiten.KeyTab, ebiten.KeySpace))
	if game.workforce.Allocation() != draft || game.selectedBand != band || stub.endTurns != 0 {
		t.Fatal("dirty Tab discarded draft or leaked into turn")
	}
	game.handleGameplayKeyState(keysDown(), keysDown(ebiten.KeyD))
	if game.workforce.Dirty() || game.detailsOpen {
		t.Fatal("D did not belong exclusively to workforce")
	}
	game.handleGameplayKeyState(keysDown(ebiten.KeyShift), keysDown(ebiten.KeyArrowUp))
	if game.openRow != ui.RowResearch || game.workforce.Role != gameapi.Foraging {
		t.Fatal("Shift+Up was not owned by global row navigation")
	}
	game.handleGameplayKeyState(keysDown(), keysDown(ebiten.KeyD))
	if !game.detailsOpen {
		t.Fatal("D did not toggle details outside workforce")
	}
	game.handleGameplayKeyState(keysDown(), keysDown(ebiten.Key2, ebiten.KeyEnter))
	if len(stub.appliedCommands) != 1 {
		t.Fatalf("simultaneous keys dispatched %d commands", len(stub.appliedCommands))
	}
	if command, ok := stub.appliedCommand.(gameapi.ResearchTech); !ok || command.Tech != gameapi.HaftedTools {
		t.Fatalf("research shortcut = %#v", stub.appliedCommand)
	}
}

func TestSceneKeyboardEscapeAndModalConsumption(t *testing.T) {
	game := New(&gameStub{frame: migrationPreviewFrame()})
	if game.handleSceneKeyState(keysDown(), keysDown()) {
		t.Fatal("gameplay swallowed ordinary input")
	}
	game.shortcutsOpen = true
	if !game.handleSceneKeyState(keysDown(), keysDown(ebiten.KeyEscape)) || game.shortcutsOpen || game.scenes.Current() != ui.SceneGameplay {
		t.Fatal("Escape did not peel shortcut sheet first")
	}
	game.handleSceneKeyState(keysDown(), keysDown(ebiten.KeyEscape))
	if game.scenes.Current() != ui.SceneMenu {
		t.Fatal("Escape did not open menu")
	}
	game.handleSceneKeyState(keysDown(), keysDown(ebiten.KeyO))
	if game.scenes.Current() != ui.SceneSettings {
		t.Fatal("O did not open settings")
	}
	before := game.settings.Muted
	if !game.handleSceneKeyState(keysDown(), keysDown(ebiten.KeyM)) || game.settings.Muted == before {
		t.Fatal("settings did not own M")
	}
	game.handleSceneKeyState(keysDown(), keysDown(ebiten.KeyEscape, ebiten.KeyM))
	if game.scenes.Current() != ui.SceneMenu || game.settings.Muted == before {
		t.Fatal("Escape did not take priority over settings action")
	}
	game.handleSceneKeyState(keysDown(), keysDown(ebiten.KeyEscape))
	if game.scenes.Current() != ui.SceneGameplay {
		t.Fatal("menu did not return to gameplay")
	}
}

func TestMenuStorageShortcutsRespectModifiersAndDirtyDraft(t *testing.T) {
	for _, key := range []ebiten.Key{ebiten.KeyS, ebiten.KeyL} {
		for _, modifier := range []ebiten.Key{ebiten.KeyControl, ebiten.KeyMeta} {
			game := New(&gameStub{frame: migrationPreviewFrame()})
			game.scenes.Push(ui.SceneMenu)
			if !game.handleSceneKeyState(keysDown(modifier), keysDown(key)) || game.scenes.Current() != ui.SceneMenu {
				t.Fatal("modified shortcut opened storage")
			}
			game.handleSceneKeyState(keysDown(), keysDown(key))
			if game.scenes.Current() != ui.SceneStorage {
				t.Fatal("unmodified shortcut did not open storage")
			}
			game.handleSceneKeyState(keysDown(), keysDown(ebiten.KeyEscape))
			if game.scenes.Current() != ui.SceneMenu {
				t.Fatal("storage Escape did not restore menu")
			}
		}
	}
	game := New(&gameStub{frame: migrationPreviewFrame()})
	game.scenes.Push(ui.SceneMenu)
	game.editAssignmentDraft(100)
	draft := game.workforce.Allocation()
	game.handleSceneKeyState(keysDown(), keysDown(ebiten.KeyT))
	if game.scenes.Current() != ui.SceneMenu || game.workforce.Allocation() != draft {
		t.Fatal("title shortcut lost dirty draft")
	}
	game.discardAssignmentDraft()
	game.handleSceneKeyState(keysDown(), keysDown(ebiten.KeyT))
	if game.scenes.Current() != ui.SceneTitle {
		t.Fatal("clean title shortcut refused")
	}
	game.handleSceneKeyState(keysDown(), keysDown(ebiten.KeyEnter, ebiten.KeyN))
	if game.scenes.Current() != ui.SceneGameplay {
		t.Fatal("continue did not return to gameplay")
	}
}

func TestTitleKeyboardNewCampaignAndLoad(t *testing.T) {
	stub := &gameStub{frame: migrationPreviewFrame()}
	game := New(stub)
	game.scenes.Push(ui.SceneTitle)
	if !game.handleSceneKeyState(keysDown(), keysDown(ebiten.KeySpace)) || stub.endTurns != 0 {
		t.Fatal("title failed to consume gameplay key")
	}
	game.handleSceneKeyState(keysDown(), keysDown(ebiten.KeyN))
	if stub.newCampaigns != 1 || game.scenes.Current() != ui.SceneGameplay {
		t.Fatal("title did not begin one campaign")
	}
	game.scenes.Push(ui.SceneTitle)
	game.handleSceneKeyState(keysDown(), keysDown(ebiten.KeyL))
	if game.scenes.Current() != ui.SceneStorage {
		t.Fatal("title load did not open browser")
	}
}

func TestStorageKeyboardSerializesOperations(t *testing.T) {
	stub := &gameStub{frame: migrationPreviewFrame()}
	game := New(stub)
	game.openStorageBrowser(storageBrowserSave)
	game.handleSceneKeyState(keysDown(), keysDown(ebiten.KeyEnter, ebiten.KeyDelete))
	if stub.savedSlot != 0 || stub.deletedSlot != 0 {
		t.Fatal("storage action ran before list completion")
	}
	stub.results = []gameapi.StorageResult{{OperationID: game.storageListID, Operation: gameapi.StorageList, Slots: []gameapi.SlotMetadata{{SlotID: 2}}}}
	game.pollStorage()
	game.handleSceneKeyState(keysDown(), keysDown(ebiten.KeyArrowUp))
	if game.storageSelection != len(storageBrowserSlots)-1 {
		t.Fatal("Up did not wrap selection")
	}
	game.handleSceneKeyState(keysDown(), keysDown(ebiten.KeyArrowDown))
	game.handleSceneKeyState(keysDown(), keysDown(ebiten.KeyArrowDown))
	game.handleSceneKeyState(keysDown(), keysDown(ebiten.KeyEnter, ebiten.KeyDelete))
	if stub.savedSlot != 2 || stub.deletedSlot != 0 || game.storageOperationID == 0 {
		t.Fatal("Enter+Delete did not serialize behind save")
	}
	stub.results = []gameapi.StorageResult{{OperationID: game.storageOperationID, Operation: gameapi.StorageSave, Slot: 2}}
	game.pollStorage()
	game.handleSceneKeyState(keysDown(), keysDown(ebiten.KeyBackspace))
	if stub.deletedSlot != 2 {
		t.Fatal("Backspace did not delete selected occupied slot after completion")
	}
}

func TestGameplayKeyboardDispatchesBestTileAndCyclesBands(t *testing.T) {
	frame := migrationPreviewFrame()
	second := frame.Bands[0]
	second.ID = 8
	frame.Bands = append(frame.Bands, second)
	stub := &gameStub{frame: frame}
	game := New(stub)
	game.hasMigrationPreview = true
	game.handleGameplayKeyState(keysDown(), keysDown(ebiten.KeyTab))
	if game.selectedBand != 8 || game.hasMigrationPreview {
		t.Fatal("Tab did not select next band and clear preview")
	}
	game.handleGameplayKeyState(keysDown(ebiten.KeyShift), keysDown(ebiten.KeyTab))
	if game.selectedBand != 7 {
		t.Fatal("Shift+Tab did not select previous band")
	}
	game.handleGameplayKeyState(keysDown(), keysDown(ebiten.KeyB))
	command, ok := stub.appliedCommand.(gameapi.QueueMigration)
	if !ok || command.BandID != 7 || command.TileID != 2 {
		t.Fatalf("B applied %#v", stub.appliedCommand)
	}
}
