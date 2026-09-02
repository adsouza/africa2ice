package app

import (
	"errors"
	"os"
	"strings"
	"testing"

	"github.com/adsouza/africa2ice/internal/application"
	gameaudio "github.com/adsouza/africa2ice/pkg/audio"
	"github.com/adsouza/africa2ice/pkg/gameapi"
	"github.com/adsouza/africa2ice/pkg/render"
	"github.com/adsouza/africa2ice/pkg/ui"
	"github.com/hajimehoshi/ebiten/v2"
)

func TestMaximumProfileFrameRendersWithCompleteHUD(t *testing.T) {
	payload, err := os.ReadFile("../../testdata/performance_profile_save.json")
	if err != nil {
		t.Fatal(err)
	}
	state, err := application.DecodeSaveState(payload)
	if err != nil {
		t.Fatal(err)
	}
	profile, err := application.ProjectSaveState(state)
	if err != nil {
		t.Fatal(err)
	}
	game := New(&gameStub{frame: migrationPreviewFrame()})
	game.SetRenderProfileFrame(profile)
	screen := ebiten.NewImage(1280, 720)
	defer screen.Deallocate()
	game.Draw(screen)
	if len(profile.Bands) != 256 {
		t.Fatal("maximum profile did not produce a complete presentation frame")
	}
}

type gameStub struct {
	frame           *gameapi.Frame
	results         []gameapi.StorageResult
	nextStorageID   gameapi.StorageOpID
	loadedSlot      int
	savedSlot       int
	deletedSlot     int
	appliedCommand  gameapi.Command
	appliedCommands []gameapi.Command
	applyErrorAt    int
	endTurns        int
	newCampaigns    int
}

type soundRecorder struct {
	played  []gameaudio.Sound
	masters []struct {
		volume float64
		muted  bool
	}
}

func (recorder *soundRecorder) Play(sound gameaudio.Sound) {
	recorder.played = append(recorder.played, sound)
}

func (recorder *soundRecorder) SetMaster(volume float64, muted bool) {
	recorder.masters = append(recorder.masters, struct {
		volume float64
		muted  bool
	}{volume: volume, muted: muted})
}

type settingsStoreStub struct {
	reads       []uint64
	writes      []ui.UISettingsCompletion
	completions []ui.UISettingsCompletion
}

func (store *settingsStoreStub) BeginRead(revision uint64) error {
	store.reads = append(store.reads, revision)
	return nil
}

func (store *settingsStoreStub) BeginWrite(revision uint64, settings ui.UISettings) error {
	store.writes = append(store.writes, ui.UISettingsCompletion{Operation: ui.UISettingsWrite, Revision: revision, Settings: settings})
	return nil
}

func (store *settingsStoreStub) Poll() []ui.UISettingsCompletion {
	result := store.completions
	store.completions = nil
	return result
}

func (stub *gameStub) Snapshot() (*gameapi.Frame, error) { return stub.frame, nil }
func (*gameStub) StateHash() (string, error)             { return "test-hash", nil }

func (stub *gameStub) NewCampaign() (*gameapi.Frame, error) {
	stub.newCampaigns++
	return stub.frame, nil
}

func (stub *gameStub) Apply(command gameapi.Command) (*gameapi.Frame, error) {
	stub.appliedCommand = command
	stub.appliedCommands = append(stub.appliedCommands, command)
	if stub.applyErrorAt > 0 && len(stub.appliedCommands) == stub.applyErrorAt {
		return nil, errors.New("semantic rejection")
	}
	return stub.frame, nil
}

func (stub *gameStub) EndTurn() (*gameapi.Frame, error) {
	stub.endTurns++
	return stub.frame, nil
}

func (stub *gameStub) BeginSave(slot int) (gameapi.StorageOpID, error) {
	stub.savedSlot = slot
	return stub.nextID(), nil
}

func (stub *gameStub) BeginLoad(slot int) (gameapi.StorageOpID, error) {
	stub.loadedSlot = slot
	return stub.nextID(), nil
}

func (stub *gameStub) BeginDelete(slot int) (gameapi.StorageOpID, error) {
	stub.deletedSlot = slot
	return stub.nextID(), nil
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

func TestFieldNotesAndSplitHotkeysRemainDistinct(t *testing.T) {
	if fieldNotesHotkey != ebiten.KeyF {
		t.Fatalf("Field Notes hotkey = %v, want F", fieldNotesHotkey)
	}
	if splitBandHotkey != ebiten.KeyN {
		t.Fatalf("split-band hotkey = %v, want N", splitBandHotkey)
	}
	if fieldNotesHotkey == splitBandHotkey {
		t.Fatal("Field Notes and split-band hotkeys overlap")
	}

	stub := &gameStub{frame: migrationPreviewFrame()}
	game := New(stub)
	if !game.handleGameplayHotkey(fieldNotesHotkey) || game.fieldNotesVisible || stub.appliedCommand != nil {
		t.Fatalf("F binding = visible %t, command %T", game.fieldNotesVisible, stub.appliedCommand)
	}
	if !game.handleGameplayHotkey(splitBandHotkey) {
		t.Fatal("N binding was not handled")
	}
	command, ok := stub.appliedCommand.(gameapi.SplitBand)
	if !ok || command.BandID != 7 || command.Destination != 2 || game.fieldNotesVisible {
		t.Fatalf("N binding = command %#v, Field Notes visible %t", stub.appliedCommand, game.fieldNotesVisible)
	}
}

func TestSplitHotkeyExplainsWhenNoOrdinaryDestinationExists(t *testing.T) {
	frame := migrationPreviewFrame()
	frame.Bands[0].MigrationCandidates[0].RequiresPassage = true
	stub := &gameStub{frame: frame}
	game := New(stub)

	game.handleGameplayHotkey(splitBandHotkey)
	if stub.appliedCommand != nil || game.notice != "This band has no eligible adjacent land tile for splitting." {
		t.Fatalf("unavailable split = command %T, notice %q", stub.appliedCommand, game.notice)
	}
}

func TestAcceptedPlanningSaveCompletionAndNewAcuteEventRequestDistinctSounds(t *testing.T) {
	frame := migrationPreviewFrame()
	stub := &gameStub{frame: frame}
	sounds := &soundRecorder{}
	game := NewWithSound(stub, sounds)

	if !game.apply(gameapi.ResearchTech{BandID: 7, Tech: gameapi.Firecraft}) {
		t.Fatal("accepted planning command was rejected")
	}
	game.beginQuickSave()
	stub.results = []gameapi.StorageResult{{Operation: gameapi.StorageSave, OperationID: stub.nextStorageID, Slot: 99}}
	game.pollStorage()
	after := cloneAppFrame(frame)
	after.Turn++
	after.Events = append(after.Events, gameapi.Event{Turn: after.Turn, Kind: gameapi.EventAcuteIncident})
	game.acceptCompletedTurn(after)

	want := []gameaudio.Sound{gameaudio.SFXChoiceClick, gameaudio.SFXSaveComplete, gameaudio.SFXEventTrigger}
	if len(sounds.played) != len(want) {
		t.Fatalf("played sounds = %v, want %v", sounds.played, want)
	}
	for index := range want {
		if sounds.played[index] != want[index] {
			t.Fatalf("played sounds = %v, want %v", sounds.played, want)
		}
	}

	game.acceptCompletedTurn(after)
	if len(sounds.played) != len(want) {
		t.Fatalf("unchanged event feed replayed a sound: %v", sounds.played)
	}
}

func TestPresentationSettingsInstallAtomicallyAndCoalesceWrites(t *testing.T) {
	store := &settingsStoreStub{}
	sounds := &soundRecorder{}
	game := newGameWithPresentation(&gameStub{frame: migrationPreviewFrame()}, sounds, store)
	if !game.settingsLoading || len(store.reads) != 1 || len(sounds.masters) != 0 {
		t.Fatalf("initial settings state = loading %t reads %v masters %v", game.settingsLoading, store.reads, sounds.masters)
	}
	loaded := ui.UISettings{SchemaVersion: 1, FieldNotesVisible: false, MasterVolume: 0.8, Muted: true}
	store.completions = []ui.UISettingsCompletion{{Operation: ui.UISettingsRead, Revision: 1, Settings: loaded}}
	game.pollUISettings()
	if game.settingsLoading || game.settings != loaded || game.fieldNotesVisible || len(sounds.masters) != 1 || sounds.masters[0].volume != 0.8 || !sounds.masters[0].muted {
		t.Fatalf("installed settings = loading %t record %#v visible %t masters %v", game.settingsLoading, game.settings, game.fieldNotesVisible, sounds.masters)
	}

	first := loaded
	first.MasterVolume = 0.4
	game.updateUISettings(first)
	latest := first
	latest.MasterVolume = 0.2
	latest.Muted = false
	game.updateUISettings(latest)
	if len(store.writes) != 1 || game.pendingSettings == nil || *game.pendingSettings != latest {
		t.Fatalf("coalesced writes = starts %v pending %#v", store.writes, game.pendingSettings)
	}
	store.completions = []ui.UISettingsCompletion{store.writes[0]}
	game.pollUISettings()
	if len(store.writes) != 2 || store.writes[1].Settings != latest || !game.settingsWriteActive {
		t.Fatalf("latest write was not started after completion: %v", store.writes)
	}
	if game.settings != latest || sounds.masters[len(sounds.masters)-1].volume != 0.2 || sounds.masters[len(sounds.masters)-1].muted {
		t.Fatalf("live settings rolled back = %#v masters %v", game.settings, sounds.masters)
	}
}

func TestRenderProfileFrameDoesNotReplaceAcceptedApplicationFrame(t *testing.T) {
	accepted := migrationPreviewFrame()
	profile := &gameapi.Frame{Turn: 300, Bands: make([]gameapi.Band, 256)}
	game := New(&gameStub{frame: accepted})
	game.SetRenderProfileFrame(profile)
	if game.displayFrame() != profile || game.frame != accepted {
		t.Fatalf("render profile crossed drawing seam: display %p accepted %p", game.displayFrame(), game.frame)
	}
}

func TestLayoutFUsesCappedDeviceScaleAndOnlyChangesViewportState(t *testing.T) {
	stub := &gameStub{frame: migrationPreviewFrame()}
	game := New(stub)
	game.deviceScaleFactor = func() float64 { return 3 }

	width, height := game.LayoutF(1000.25, 600.25)
	if width != 2001 || height != 1201 || game.viewport.RenderScale != 2 || game.viewport.ViewportRevision != 1 {
		t.Fatalf("first layout = %vx%v, viewport %#v", width, height, game.viewport)
	}
	game.LayoutF(1000.25, 600.25)
	if game.viewport.ViewportRevision != 1 {
		t.Fatalf("identical layout advanced revision to %d", game.viewport.ViewportRevision)
	}
	game.deviceScaleFactor = func() float64 { return 1.25 }
	game.LayoutF(1000.25, 600.25)
	if game.viewport.ViewportRevision != 2 || game.viewport.RenderWidthPx != 1251 {
		t.Fatalf("monitor-scale change = %#v", game.viewport)
	}
	if stub.appliedCommand != nil || stub.savedSlot != 0 || stub.loadedSlot != 0 || stub.nextStorageID != 0 {
		t.Fatalf("layout touched game port: %#v", stub)
	}
}

func TestActionBatchPreflightRejectsTheWholeMixedBatch(t *testing.T) {
	stub := &gameStub{frame: migrationPreviewFrame()}
	game := New(stub)
	_, accepted := game.dispatchBatch([]ui.Action{
		ui.SimulationAction(gameapi.ResearchTech{BandID: 7, Tech: gameapi.Firecraft}),
		ui.SaveAction(1),
	})
	if accepted || len(stub.appliedCommands) != 0 || stub.savedSlot != 0 || stub.endTurns != 0 {
		t.Fatalf("rejected mixed batch invoked port: accepted=%t commands=%d save=%d turns=%d", accepted, len(stub.appliedCommands), stub.savedSlot, stub.endTurns)
	}
}

func TestCampaignBatchStopsAfterSemanticFailureAndKeepsEarlierFrame(t *testing.T) {
	initial := migrationPreviewFrame()
	firstAccepted := cloneAppFrame(initial)
	firstAccepted.WorldRevision++
	stub := &gameStub{frame: firstAccepted, applyErrorAt: 2}
	game := New(stub)
	game.frame = initial
	_, accepted := game.dispatchBatch([]ui.Action{
		ui.SimulationAction(gameapi.ResearchTech{BandID: 7, Tech: gameapi.Firecraft}),
		ui.SimulationAction(gameapi.SetAssignment{BandID: 7, AllocationBP: [gameapi.AssignmentCount]uint16{2_000, 2_000, 2_000, 2_000, 2_000}}),
		ui.EndTurnAction(),
	})
	if accepted || len(stub.appliedCommands) != 2 || stub.endTurns != 0 {
		t.Fatalf("semantic failure sequence = accepted %t commands %d turns %d", accepted, len(stub.appliedCommands), stub.endTurns)
	}
	if game.frame != firstAccepted {
		t.Fatal("semantic failure discarded the frame returned by an earlier accepted command")
	}
}

func cloneAppFrame(frame *gameapi.Frame) *gameapi.Frame {
	clone := *frame
	clone.Tiles = append([]gameapi.Tile(nil), frame.Tiles...)
	clone.Bands = append([]gameapi.Band(nil), frame.Bands...)
	clone.Events = append([]gameapi.Event(nil), frame.Events...)
	return &clone
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
		gameapi.Band{ID: 9, Species: gameapi.HomoSapiens, Population: 90},
		gameapi.Band{ID: 10, Species: gameapi.ArchaicHominin},
		gameapi.Band{ID: 11, Species: gameapi.HomoSapiens, Population: 90},
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

func TestBandSelectionUsesAttentionOrder(t *testing.T) {
	frame := migrationPreviewFrame()
	frame.Bands[0].Health = 1
	frame.Bands = append(frame.Bands,
		gameapi.Band{ID: 9, Species: gameapi.HomoSapiens, Population: 90, Health: 0.7},
		gameapi.Band{ID: 11, Species: gameapi.HomoSapiens, Population: 90, Health: 0.9, LastFoodReport: gameapi.FoodTurnReport{Turn: 2, RequiredFU: 90, DeficitFU: 1}},
	)
	game := New(&gameStub{frame: frame})

	if game.selectedBand != 11 {
		t.Fatalf("initial priority selection = %d, want suffering band 11", game.selectedBand)
	}
	game.selectNextSapiens()
	if game.selectedBand != 9 {
		t.Fatalf("next priority selection = %d, want danger band 9", game.selectedBand)
	}
	game.selectNextSapiens()
	if game.selectedBand != 7 {
		t.Fatalf("final priority selection = %d, want stable band 7", game.selectedBand)
	}
	game.selectNextSapiens()
	if game.selectedBand != 11 {
		t.Fatalf("wrapped priority selection = %d, want band 11", game.selectedBand)
	}
}

func TestCompletedTurnSelectsHighestAttentionBand(t *testing.T) {
	before := migrationPreviewFrame()
	before.Bands[0].Health = 1
	before.Bands = append(before.Bands, gameapi.Band{ID: 9, Species: gameapi.HomoSapiens, Population: 90, Health: 1})
	game := New(&gameStub{frame: before})
	if game.selectedBand != 7 {
		t.Fatalf("initial selection = %d, want band 7", game.selectedBand)
	}

	after := cloneAppFrame(before)
	after.Turn = 1
	after.Bands[1].Health = 0.95
	after.Bands[1].LastOutcomeReport = gameapi.OutcomeReport{
		Turn: 1, StartingPopulation: 90, EndingPopulation: 90, StartingHealth: 1, EndingHealth: 0.95,
	}
	game.acceptCompletedTurn(after)

	if game.selectedBand != 9 || game.assignmentDraftBand != 9 {
		t.Fatalf("completed-turn priority selection/draft = %d/%d, want band 9", game.selectedBand, game.assignmentDraftBand)
	}
}

func TestInitialSelectionSkipsArchaicAndExtinctBands(t *testing.T) {
	frame := migrationPreviewFrame()
	frame.Bands = []gameapi.Band{
		{ID: 0, Species: gameapi.ArchaicHominin, Population: 90},
		{ID: 6, Species: gameapi.HomoSapiens},
		{ID: 7, Species: gameapi.HomoSapiens, Population: 120},
	}
	game := New(&gameStub{frame: frame})

	if game.selectedBand != 7 || !game.hasAssignmentDraft {
		t.Fatalf("initial selection = band %d draft %t, want living sapiens band 7", game.selectedBand, game.hasAssignmentDraft)
	}
}

func TestFieldNoteScrollNormalizesBeforeApplyingInput(t *testing.T) {
	game := New(&gameStub{frame: migrationPreviewFrame()})
	game.fieldNote = render.FieldNote{
		Introduction: strings.Repeat("A deliberately verbose field note sentence. ", 20),
		Context:      strings.Repeat("Additional historical context. ", 20),
	}
	maximum := render.FieldNoteMaxScroll(game.fieldNote)
	if maximum <= 3 {
		t.Fatalf("test Field Note max scroll = %d, want more than 3", maximum)
	}

	game.fieldNoteScroll = maximum + 12
	game.scrollFieldNotes(-3)
	if game.fieldNoteScroll != maximum-3 {
		t.Fatalf("PageUp-equivalent scroll = %d, want %d", game.fieldNoteScroll, maximum-3)
	}
	game.scrollFieldNotes(maximum + 12)
	if game.fieldNoteScroll != maximum {
		t.Fatalf("PageDown-equivalent scroll = %d, want capped %d", game.fieldNoteScroll, maximum)
	}
}

func TestWorkforceDraftPreservesExplicitSharesUntilValidApplyOrDiscard(t *testing.T) {
	frame := migrationPreviewFrame()
	frame.Bands[0].AllocationBP = [gameapi.AssignmentCount]uint16{2_000, 2_000, 2_000, 2_000, 2_000}
	frame.Bands = append(frame.Bands, gameapi.Band{ID: 9, Species: gameapi.HomoSapiens, AllocationBP: frame.Bands[0].AllocationBP})
	stub := &gameStub{frame: frame}
	game := New(stub)

	game.editAssignmentDraft(100)
	if !game.assignmentDraftDirty() || game.assignmentDraftValid() || stub.appliedCommand != nil {
		t.Fatalf("first independent edit = dirty %t valid %t command %T", game.assignmentDraftDirty(), game.assignmentDraftValid(), stub.appliedCommand)
	}
	game.applyAssignmentDraft()
	if stub.appliedCommand != nil || game.notice != "Workforce allocation must total exactly 100%" {
		t.Fatalf("invalid apply = command %T notice %q", stub.appliedCommand, game.notice)
	}
	game.selectNextSapiens()
	if game.selectedBand != 7 {
		t.Fatalf("dirty draft allowed selection to change to %d", game.selectedBand)
	}

	game.assignmentRole = gameapi.HuntingAndFishing
	game.editAssignmentDraft(-100)
	if !game.assignmentDraftValid() {
		t.Fatalf("balanced explicit draft = %v", game.assignmentDraft)
	}
	game.applyAssignmentDraft()
	command, ok := stub.appliedCommand.(gameapi.SetAssignment)
	if !ok || command.AllocationBP != ([gameapi.AssignmentCount]uint16{2_100, 1_900, 2_000, 2_000, 2_000}) {
		t.Fatalf("applied assignment = %#v", stub.appliedCommand)
	}
	if game.assignmentDraftDirty() {
		t.Fatal("successful Apply left the draft dirty")
	}

	game.editAssignmentDraft(100)
	game.discardAssignmentDraft()
	if game.assignmentDraftDirty() || game.assignmentDraft != game.assignmentBaseline {
		t.Fatalf("discarded assignment = draft %v baseline %v", game.assignmentDraft, game.assignmentBaseline)
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

func TestCompletedTurnPrioritizesNewRegionContextWhenThereIsNoBreakthrough(t *testing.T) {
	before := migrationPreviewFrame()
	before.SapiensEstablishedRegions = []gameapi.Region{gameapi.EastAfrica}
	after := cloneAppFrame(before)
	after.Turn++
	after.SapiensEstablishedRegions = []gameapi.Region{gameapi.EastAfrica, gameapi.Arabia}
	after.Events = append(after.Events, gameapi.Event{Turn: after.Turn, Kind: gameapi.EventAchievement, Region: gameapi.Arabia, Summary: "Arabia established"})
	game := New(&gameStub{frame: before})

	game.acceptCompletedTurn(after)
	if !strings.Contains(game.fieldNote.Topic, "Arabia") || !strings.Contains(game.fieldNote.Introduction, "founder") {
		t.Fatalf("new-region note = %#v", game.fieldNote)
	}
	if region, ok := newlyEstablishedRegion(after, after); ok || region != 0 {
		t.Fatalf("latched region replayed = (%d,%t)", region, ok)
	}
}

func TestCompletedTurnFocusesTobaAndClimateContextOnlyWhenCrossed(t *testing.T) {
	before := migrationPreviewFrame()
	before.YearBP = 74_000
	after := cloneAppFrame(before)
	after.Turn++
	after.YearBP = 73_700
	game := New(&gameStub{frame: before})
	game.acceptCompletedTurn(after)
	if !strings.Contains(game.fieldNote.Topic, "TOBA") || !strings.Contains(game.fieldNote.GameEffect, "no effect") {
		t.Fatalf("Toba crossing note = %#v", game.fieldNote)
	}

	climateBefore := cloneAppFrame(after)
	climateAfter := cloneAppFrame(after)
	climateAfter.Turn++
	climateAfter.YearBP -= 300
	climateAfter.Climate.Epoch = gameapi.AridTransition
	game.frame = climateBefore
	game.acceptCompletedTurn(climateAfter)
	if !strings.Contains(game.fieldNote.Topic, "Arid Transition") {
		t.Fatalf("climate transition note = %#v", game.fieldNote)
	}
}

func TestCompletedTurnPrioritizesEpochTransitionOverRegionalPulse(t *testing.T) {
	before := migrationPreviewFrame()
	after := cloneAppFrame(before)
	after.Turn++
	after.YearBP -= 300
	after.Climate.Epoch = gameapi.AridTransition
	region := after.Tiles[after.Bands[0].TileID].Region
	after.Climate.RegionalAbrupt[region] = 0.2
	game := New(&gameStub{frame: before})

	game.acceptCompletedTurn(after)
	if !strings.Contains(game.fieldNote.Topic, "Arid Transition") {
		t.Fatalf("regional pulse displaced epoch transition note: %#v", game.fieldNote)
	}
}

func TestRegionalPulseFieldNoteFocusesOnlyOncePerContinuousRun(t *testing.T) {
	before := migrationPreviewFrame()
	first := cloneAppFrame(before)
	first.Turn++
	region := first.Tiles[first.Bands[0].TileID].Region
	first.Climate.RegionalAbrupt[region] = 0.2
	game := New(&gameStub{frame: before})

	game.acceptCompletedTurn(first)
	if !strings.Contains(game.fieldNote.Topic, "REGIONAL CLIMATE PULSE") || !game.regionalPulseFocused {
		t.Fatalf("first pulse note = %#v, focused %t", game.fieldNote, game.regionalPulseFocused)
	}
	game.fieldNoteScroll = 4
	second := cloneAppFrame(first)
	second.Turn++
	second.Climate.RegionalAbrupt[region] = 0.3
	game.acceptCompletedTurn(second)
	if game.fieldNoteScroll != 4 {
		t.Fatalf("continuous pulse reset Field Notes scroll to %d", game.fieldNoteScroll)
	}

	between := cloneAppFrame(second)
	between.Turn++
	between.Climate.RegionalAbrupt[region] = 0
	game.acceptCompletedTurn(between)
	if game.regionalPulseFocused {
		t.Fatal("ended pulse retained its focus marker")
	}
	later := cloneAppFrame(between)
	later.Turn++
	later.Climate.RegionalAbrupt[region] = 0.2
	game.fieldNoteScroll = 3
	game.acceptCompletedTurn(later)
	if game.fieldNoteScroll != 0 || !game.regionalPulseFocused {
		t.Fatalf("later pulse did not refocus: scroll %d focused %t", game.fieldNoteScroll, game.regionalPulseFocused)
	}
}

func TestStartupResumeLoadsNewestQuickOrAutosave(t *testing.T) {
	initial := migrationPreviewFrame()
	restored := migrationPreviewFrame()
	restored.Turn = 3
	restored.Bands[0].TileID = 2
	restored.Bands[0].Health = 1
	restored.Bands = append(restored.Bands, gameapi.Band{ID: 9, Species: gameapi.HomoSapiens, Population: 90, Health: 0.7})
	stub := &gameStub{frame: initial}
	game := New(stub)
	game.fieldNote, _ = ui.TechnologyFieldNote(gameapi.Firecraft, 7, 1)
	game.breakthroughFrames = breakthroughCelebrationFrames
	game.regionalPulseFocused = true

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
	if game.fieldNote.Topic != "WELCOME" || game.breakthroughFrames != 0 || game.regionalPulseFocused {
		t.Fatalf("loaded game retained stale presentation: note %#v, breakthrough %d, pulse focus %t", game.fieldNote, game.breakthroughFrames, game.regionalPulseFocused)
	}
	if game.selectedBand != 9 || game.assignmentDraftBand != 9 {
		t.Fatalf("loaded priority selection/draft = %d/%d, want band 9", game.selectedBand, game.assignmentDraftBand)
	}
}

func TestManualSlotShortcutsUseExplicitSlotsAndFreezeARequestedLoad(t *testing.T) {
	stub := &gameStub{frame: migrationPreviewFrame()}
	sounds := &soundRecorder{}
	game := NewWithSound(stub, sounds)
	game.beginManualSave(2)
	if stub.savedSlot != 2 || game.notice != "Saving Manual 2…" {
		t.Fatalf("manual save = slot %d notice %q", stub.savedSlot, game.notice)
	}
	stub.results = []gameapi.StorageResult{{Operation: gameapi.StorageSave, OperationID: stub.nextStorageID, Slot: 2}}
	game.pollStorage()
	if len(sounds.played) != 1 || sounds.played[0] != gameaudio.SFXSaveComplete {
		t.Fatalf("manual-save sounds = %v", sounds.played)
	}
	stub.results = []gameapi.StorageResult{{Operation: gameapi.StorageSave, OperationID: stub.nextStorageID, Slot: 2}}
	game.pollStorage()
	if len(sounds.played) != 1 {
		t.Fatalf("duplicate completion replayed manual-save sound: %v", sounds.played)
	}
	game.beginManualLoad(3)
	if stub.loadedSlot != 3 || game.pendingManualLoadID == 0 || game.notice != "Loading Manual 3…" {
		t.Fatalf("manual load = slot %d pending %d notice %q", stub.loadedSlot, game.pendingManualLoadID, game.notice)
	}
	operationID := game.pendingManualLoadID
	stub.results = []gameapi.StorageResult{{Operation: gameapi.StorageLoad, OperationID: operationID, Slot: 3, ReplacementFrame: migrationPreviewFrame()}}
	game.pollStorage()
	if game.pendingManualLoadID != 0 || game.notice != "Loaded Manual 3" {
		t.Fatalf("manual load completion = pending %d notice %q", game.pendingManualLoadID, game.notice)
	}

	game.editAssignmentDraft(100)
	stub.loadedSlot = 0
	game.beginManualLoad(1)
	if stub.loadedSlot != 0 || game.notice != "Apply or discard workforce changes before loading" {
		t.Fatalf("dirty draft load = slot %d notice %q", stub.loadedSlot, game.notice)
	}
}

func TestClickingSelectedBandPreservesDirtyAssignmentDraft(t *testing.T) {
	stub := &gameStub{frame: migrationPreviewFrame()}
	game := New(stub)
	game.editAssignmentDraft(100)
	game.notice = "existing notice"
	want := game.assignmentDraft
	if !game.assignmentDraftDirty() {
		t.Fatal("test setup did not create a dirty assignment draft")
	}
	if !game.selectBandAtTile(game.frame.Bands[0].TileID) {
		t.Fatal("selected band marker was not recognized")
	}
	if !game.assignmentDraftDirty() || game.assignmentDraft != want {
		t.Fatalf("same-band click changed draft: got %#v want %#v", game.assignmentDraft, want)
	}
	if game.notice != "existing notice" {
		t.Fatalf("same-band click replaced notice with %q", game.notice)
	}
}

func TestResearchKeyWithoutSelectionRequestsSapiensBand(t *testing.T) {
	stub := &gameStub{frame: &gameapi.Frame{CampaignResult: gameapi.Ongoing}}
	game := New(stub)

	game.chooseResearchTechnology(gameapi.Firecraft)

	if stub.appliedCommand != nil {
		t.Fatalf("no-selection research applied %T", stub.appliedCommand)
	}
	if game.notice != "Select a Homo sapiens band to choose research." {
		t.Fatalf("no-selection research notice = %q", game.notice)
	}
}

func TestRepeatedTileClicksCycleVisibleSapiensAndArchaicBands(t *testing.T) {
	frame := migrationPreviewFrame()
	frame.Bands = append(frame.Bands, gameapi.Band{ID: 12, Species: gameapi.ArchaicHominin, TileID: 0, Population: 90})
	game := New(&gameStub{frame: frame})
	if game.selectedBand != 7 {
		t.Fatalf("initial selection = %d", game.selectedBand)
	}
	if !game.selectBandAtTile(0) || game.selectedBand != 12 || game.hasAssignmentDraft {
		t.Fatalf("first repeated click did not select read-only archaic band: selected %d draft %t", game.selectedBand, game.hasAssignmentDraft)
	}
	if !strings.Contains(game.fieldNote.Introduction, "Computer controlled") {
		t.Fatalf("archaic Field Notes = %#v", game.fieldNote)
	}
	if !game.selectBandAtTile(0) || game.selectedBand != 7 || !game.hasAssignmentDraft {
		t.Fatalf("second repeated click did not wrap to sapiens: selected %d draft %t", game.selectedBand, game.hasAssignmentDraft)
	}
}

func TestTitleAndGameMenuExposeCampaignNavigation(t *testing.T) {
	game := New(&gameStub{frame: migrationPreviewFrame()})
	if !game.scenes.Push(ui.SceneTitle) {
		t.Fatal("could not install title scene")
	}
	title := game.menuOverlayForRender()
	if title.Heading != "Africa 2 Ice: Paleolithic Dispersal" || title.LineCount != 3 || !strings.Contains(title.Lines[1], "New Campaign") {
		t.Fatalf("title overlay = %#v", title)
	}
	game.scenes.Reset()
	game.scenes.Push(ui.SceneMenu)
	menu := game.menuOverlayForRender()
	if menu.LineCount != 6 || !strings.Contains(menu.Lines[5], "title") {
		t.Fatalf("game menu navigation = %#v", menu)
	}
}

func TestGameMenuDescribesTurnBasedBehavior(t *testing.T) {
	game := New(&gameStub{frame: migrationPreviewFrame()})
	game.scenes.Push(ui.SceneMenu)

	overlay := game.menuOverlayForRender()
	if overlay.Heading != "Game Menu" || overlay.Lines[0] != "Esc  Back to game" {
		t.Fatalf("game menu identity = %#v", overlay)
	}
	if strings.Contains(strings.ToLower(overlay.Help), "pause") || !strings.Contains(overlay.Help, "explicitly end") {
		t.Fatalf("game menu turn guidance = %q", overlay.Help)
	}
}

func TestStorageBrowserListsAllGroupsAndActivatesExplicitOperations(t *testing.T) {
	stub := &gameStub{frame: migrationPreviewFrame()}
	game := New(stub)
	game.scenes.Push(ui.SceneMenu)
	game.openStorageBrowser(storageBrowserLoad)
	if game.scenes.Current() != ui.SceneStorage || game.storageListID == 0 {
		t.Fatalf("opened browser = scene %d list %d", game.scenes.Current(), game.storageListID)
	}
	stub.results = []gameapi.StorageResult{{
		OperationID: game.storageListID, Operation: gameapi.StorageList,
		Slots: []gameapi.SlotMetadata{
			{SlotID: 1, SlotKind: gameapi.ManualSlot, Turn: 4, YearBP: 78_800, SapiensPopulation: 490},
			{SlotID: 102, SlotKind: gameapi.AutoSlot, Turn: 8, YearBP: 77_600, SapiensPopulation: 510},
		},
	}}
	game.pollStorage()
	overlay := game.menuOverlayForRender()
	if overlay.LineCount != 7 || !strings.Contains(overlay.Lines[0], "Turn 4") || !strings.Contains(overlay.Lines[5], "Turn 8") || !strings.Contains(overlay.Lines[3], "Empty") {
		t.Fatalf("storage overlay = %#v", overlay)
	}

	game.storageSelection = 5
	game.activateStorageSelection()
	if stub.loadedSlot != 102 || game.pendingManualLoadID == 0 {
		t.Fatalf("autosave load = slot %d pending %d", stub.loadedSlot, game.pendingManualLoadID)
	}
	loadID := game.pendingManualLoadID
	stub.results = []gameapi.StorageResult{{OperationID: loadID, Operation: gameapi.StorageLoad, Slot: 102, ReplacementFrame: migrationPreviewFrame()}}
	game.pollStorage()
	if game.scenes.Current() != ui.SceneGameplay || game.notice != "Loaded Auto 2" {
		t.Fatalf("completed browser load = scene %d notice %q", game.scenes.Current(), game.notice)
	}
}

func TestStorageBrowserRestrictsWritesButCanDeleteAnyOccupiedGroup(t *testing.T) {
	stub := &gameStub{frame: migrationPreviewFrame()}
	game := New(stub)
	game.scenes.Push(ui.SceneMenu)
	game.openStorageBrowser(storageBrowserSave)
	game.storageListID = 0
	game.storageSlots = []gameapi.SlotMetadata{{SlotID: 99, SlotKind: gameapi.QuickSlot}}

	game.storageSelection = 3
	game.activateStorageSelection()
	if stub.savedSlot != 0 || !strings.Contains(game.notice, "Only Manual 1–3") {
		t.Fatalf("quick-slot overwrite = saved %d notice %q", stub.savedSlot, game.notice)
	}
	game.deleteStorageSelection()
	if stub.deletedSlot != 99 || game.storageOperationID == 0 {
		t.Fatalf("quick-slot delete = slot %d operation %d", stub.deletedSlot, game.storageOperationID)
	}
}

func TestSettingsSceneReportsLivePreferences(t *testing.T) {
	game := New(&gameStub{frame: migrationPreviewFrame()})
	game.scenes.Push(ui.SceneMenu)
	game.scenes.Push(ui.SceneSettings)
	game.settings.MasterVolume = 0.7
	game.settings.Muted = true
	game.fieldNotesVisible = false

	overlay := game.menuOverlayForRender()
	joined := strings.Join(overlay.Lines[:overlay.LineCount], " ")
	for _, required := range []string{"70%", "Muted  On", "Field Notes  Hidden"} {
		if !strings.Contains(joined, required) {
			t.Fatalf("settings overlay missing %q: %#v", required, overlay)
		}
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
	game.regionalPulseFocused = true

	fresh := migrationPreviewFrame()
	stub.frame = fresh
	game.startNewCampaign()

	if stub.newCampaigns != 1 || game.frame != fresh || game.frame.CampaignResult != gameapi.Ongoing {
		t.Fatalf("new campaign = calls %d, frame %#v", stub.newCampaigns, game.frame)
	}
	if game.fieldNote.Topic != "WELCOME" || game.breakthroughFrames != 0 || game.hasMigrationPreview || game.regionalPulseFocused {
		t.Fatalf("new campaign retained stale presentation: note %#v, breakthrough %d, preview %t, pulse focus %t", game.fieldNote, game.breakthroughFrames, game.hasMigrationPreview, game.regionalPulseFocused)
	}
	if game.notice != "New campaign begun" || game.selectedBand != 7 {
		t.Fatalf("new campaign notice/selection = %q/%d", game.notice, game.selectedBand)
	}
}

func TestExploredHoverTileRejectsFogAndCoordinatesOutsideTheMap(t *testing.T) {
	frame := &gameapi.Frame{Tiles: make([]gameapi.Tile, 2)}
	frame.Tiles[0] = gameapi.Tile{ID: 0, Explored: true}
	frame.Tiles[1] = gameapi.Tile{ID: 1, Explored: false}

	if tileID, ok := exploredHoverTile(frame, 24, 78, true); !ok || tileID != 0 {
		t.Fatalf("explored hover = (%d, %t), want tile 0", tileID, ok)
	}
	if _, ok := exploredHoverTile(frame, 32, 78, true); ok {
		t.Fatal("pointer hover exposed a fogged tile")
	}
	if _, ok := exploredHoverTile(frame, 19, 78, true); ok {
		t.Fatal("pointer hover accepted a coordinate outside the map")
	}
	if _, ok := exploredHoverTile(frame, 24, 78, false); ok {
		t.Fatal("pointer hover ignored the viewport boundary")
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
			Population:          120,
			MigrationCandidates: []gameapi.MigrationCandidate{{TileID: 2}},
		}},
	}
}

var _ gameapi.Game = (*gameStub)(nil)

func TestShowNoticeScalesDurationWithLength(t *testing.T) {
	game := New(&gameStub{frame: migrationPreviewFrame()})
	message := "Too far away: move one outlined tile or use an eligible named passage. Keep using arrows, or press Esc to clear."
	game.showNotice(message)
	if game.notice != message || game.noticeFrames != ui.NoticeFrames(message) || game.noticeFrames <= 120 {
		t.Fatalf("showNotice = %q for %d frames, want ui.NoticeFrames %d (> 120)", game.notice, game.noticeFrames, ui.NoticeFrames(message))
	}
}
