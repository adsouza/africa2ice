package app

import (
	"fmt"

	"github.com/adsouza/africa2ice/internal/adapters/logging"
	"github.com/adsouza/africa2ice/internal/application"
	gameaudio "github.com/adsouza/africa2ice/pkg/audio"
	"github.com/adsouza/africa2ice/pkg/gameapi"
	"github.com/adsouza/africa2ice/pkg/render"
	"github.com/adsouza/africa2ice/pkg/ui"
	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
)

const (
	LogicalWidth                  = 1280
	LogicalHeight                 = 720
	breakthroughCelebrationFrames = 600
)

type Game struct {
	port                   gameapi.Game
	sound                  gameaudio.SoundManager
	frame                  *gameapi.Frame
	scene                  *render.MapScene
	onFirstDraw            func()
	onFramePublished       func(string)
	firstDrawDone          bool
	selectedBand           gameapi.BandID
	notice                 string
	noticeFrames           int
	fieldNotesVisible      bool
	fieldNote              render.FieldNote
	breakthroughFrames     int
	migrationPreviewBand   gameapi.BandID
	migrationPreviewTile   gameapi.TileID
	hasMigrationPreview    bool
	startupRestorePending  bool
	startupRestoreListID   gameapi.StorageOpID
	startupRestoreLoadID   gameapi.StorageOpID
	startupRestoreSlot     int
	pendingQuickSaveIDs    map[gameapi.StorageOpID]struct{}
	windowClosingRequested bool
	pendingManualLoadID    gameapi.StorageOpID
	settingsStore          ui.UISettingsStore
	settings               ui.UISettings
	settingsLoading        bool
	settingsRevision       uint64
	settingsWriteActive    bool
	pendingSettings        *ui.UISettings
	assignmentDraft        [gameapi.AssignmentCount]uint16
	assignmentBaseline     [gameapi.AssignmentCount]uint16
	assignmentDraftBand    gameapi.BandID
	assignmentRole         gameapi.WorkforceRole
	hasAssignmentDraft     bool
	profileDisplayFrame    *gameapi.Frame
}

func NewWalkingSkeleton() *Game {
	port := newSkeletonPort()
	return New(port)
}

func New(port gameapi.Game) *Game {
	return NewWithSound(port, gameaudio.NoopManager{})
}

func NewWithSound(port gameapi.Game, sound gameaudio.SoundManager) *Game {
	return newGameWithPresentation(port, sound, nil)
}

func newGameWithPresentation(port gameapi.Game, sound gameaudio.SoundManager, settingsStore ui.UISettingsStore) *Game {
	frame, _ := port.Snapshot()
	if sound == nil {
		sound = gameaudio.NoopManager{}
	}
	settings := ui.DefaultUISettings()
	game := &Game{
		port: port, sound: sound, frame: frame, scene: render.NewMapScene(), fieldNotesVisible: true,
		fieldNote: ui.CampaignOverviewFieldNote(),
		notice:    "Outlined tiles are reachable — arrows choose, Enter confirms", noticeFrames: 300,
		pendingQuickSaveIDs: make(map[gameapi.StorageOpID]struct{}),
		settingsStore:       settingsStore, settings: settings,
	}
	if settingsStore == nil {
		sound.SetMaster(settings.MasterVolume, settings.Muted)
	} else if err := settingsStore.BeginRead(1); err != nil {
		sound.SetMaster(settings.MasterVolume, settings.Muted)
		game.notice = "Preferences could not be loaded; using defaults"
	} else {
		game.settingsLoading = true
		game.settingsRevision = 1
	}
	game.ensureSelection()
	game.syncAssignmentDraft(true)
	return game
}

func NewGame(seed uint64, session *logging.Session) (*Game, error) {
	repository, err := newCampaignRepository()
	if err != nil {
		return nil, err
	}
	repository = logging.DecorateCampaignRepository(session, repository)
	service, err := application.NewGameServiceWithRepository(seed, repository)
	if err != nil {
		return nil, err
	}
	sound := gameaudio.NewLazyManager()
	settingsStore, settingsErr := newUISettingsStore()
	settingsStore = logging.DecorateUISettingsStore(session, settingsStore)
	game := newGameWithPresentation(logging.DecorateGame(session, service), sound, settingsStore)
	if settingsErr != nil {
		sound.SetMaster(ui.DefaultUISettings().MasterVolume, ui.DefaultUISettings().Muted)
		game.showNotice("Preferences are unavailable; using defaults")
	}
	if shouldResumeSavedGameOnStartup() {
		game.beginStartupResume()
	}
	return game, nil
}

func (g *Game) Update() error {
	g.scene.Update()
	g.pollUISettings()
	if g.noticeFrames > 0 {
		g.noticeFrames--
		if g.noticeFrames == 0 {
			g.notice = ""
		}
	}
	if g.breakthroughFrames > 0 {
		g.breakthroughFrames--
	}
	g.pollStorage()
	if ebiten.IsWindowBeingClosed() {
		g.windowClosingRequested = true
	}
	if g.windowClosingRequested {
		if len(g.pendingQuickSaveIDs) == 0 {
			return ebiten.Termination
		}
		g.showNotice("Finishing quick-save before exit…")
		return nil
	}
	if g.startupRestorePending {
		return nil
	}
	if g.frame == nil {
		return nil
	}
	modifier := ebiten.IsKeyPressed(ebiten.KeyControl) || ebiten.IsKeyPressed(ebiten.KeyMeta)
	if modifier && inpututil.IsKeyJustPressed(ebiten.KeyS) {
		g.beginQuickSave()
	}
	for index, key := range [...]ebiten.Key{ebiten.KeyF1, ebiten.KeyF2, ebiten.KeyF3} {
		if !inpututil.IsKeyJustPressed(key) {
			continue
		}
		slot := index + 1
		if ebiten.IsKeyPressed(ebiten.KeyShift) {
			g.beginManualLoad(slot)
		} else {
			g.beginManualSave(slot)
		}
		break
	}
	if g.pendingManualLoadID != 0 {
		return nil
	}
	if g.frame.CampaignResult != gameapi.Ongoing {
		mouseStartsCampaign := false
		if inpututil.IsMouseButtonJustPressed(ebiten.MouseButtonLeft) {
			mouseStartsCampaign = render.NewCampaignButtonContains(ebiten.CursorPosition())
		}
		if inpututil.IsKeyJustPressed(ebiten.KeyN) || mouseStartsCampaign {
			g.startNewCampaign()
		}
		return nil
	}
	if inpututil.IsKeyJustPressed(ebiten.KeyTab) {
		if g.assignmentDraftDirty() {
			g.showNotice("Apply or discard workforce changes")
		} else {
			g.clearMigrationPreview()
			if ebiten.IsKeyPressed(ebiten.KeyShift) {
				g.selectPreviousSapiens()
			} else {
				g.selectNextSapiens()
			}
		}
	}
	if inpututil.IsKeyJustPressed(ebiten.KeyF) {
		if g.settingsLoading {
			g.showNotice("Loading preferences…")
		} else {
			settings := g.settings
			settings.FieldNotesVisible = !settings.FieldNotesVisible
			g.updateUISettings(settings)
		}
	}
	if inpututil.IsKeyJustPressed(ebiten.KeyM) {
		if g.settingsLoading {
			g.showNotice("Loading preferences…")
		} else {
			settings := g.settings
			settings.Muted = !settings.Muted
			g.updateUISettings(settings)
			if settings.Muted {
				g.showNotice("Sound muted")
			} else {
				g.showNotice(fmt.Sprintf("Sound unmuted at %.0f%%", settings.MasterVolume*100))
			}
		}
	}
	for _, volumeKey := range [...]struct {
		key   ebiten.Key
		delta float64
	}{{key: ebiten.KeyMinus, delta: -0.1}, {key: ebiten.KeyEqual, delta: 0.1}} {
		if inpututil.IsKeyJustPressed(volumeKey.key) {
			if g.settingsLoading {
				g.showNotice("Loading preferences…")
			} else {
				settings := g.settings
				settings.MasterVolume += volumeKey.delta
				g.updateUISettings(settings)
				g.showNotice(fmt.Sprintf("Master volume %.0f%%", g.settings.MasterVolume*100))
			}
			break
		}
	}
	if inpututil.IsKeyJustPressed(ebiten.KeyW) && g.hasAssignmentDraft {
		g.assignmentRole = (g.assignmentRole + 1) % gameapi.AssignmentCount
	}
	for _, edit := range [...]struct {
		key   ebiten.Key
		delta int
	}{{key: ebiten.KeyBracketLeft, delta: -100}, {key: ebiten.KeyBracketRight, delta: 100}} {
		if inpututil.IsKeyJustPressed(edit.key) {
			g.editAssignmentDraft(edit.delta)
			break
		}
	}
	if inpututil.IsKeyJustPressed(ebiten.KeyA) {
		g.applyAssignmentDraft()
	}
	if inpututil.IsKeyJustPressed(ebiten.KeyD) {
		g.discardAssignmentDraft()
	}
	if inpututil.IsMouseButtonJustPressed(ebiten.MouseButtonLeft) {
		g.handleMapClick()
	}
	for _, directionalKey := range [...]struct {
		key    ebiten.Key
		dx, dy int
	}{
		{key: ebiten.KeyArrowUp, dy: -1},
		{key: ebiten.KeyArrowDown, dy: 1},
		{key: ebiten.KeyArrowLeft, dx: -1},
		{key: ebiten.KeyArrowRight, dx: 1},
	} {
		if inpututil.IsKeyJustPressed(directionalKey.key) {
			g.handleDirectionalMigration(directionalKey.dx, directionalKey.dy)
			break
		}
	}
	if inpututil.IsKeyJustPressed(ebiten.KeyEnter) {
		g.confirmMigrationPreview()
	}
	if inpututil.IsKeyJustPressed(ebiten.KeyEscape) && g.hasMigrationPreview {
		g.clearMigrationPreview()
		g.showNotice("Migration choice cleared")
	}
	if inpututil.IsKeyJustPressed(ebiten.KeyN) {
		if band := g.selected(); band != nil {
			for _, candidate := range band.MigrationCandidates {
				if !candidate.RequiresPassage {
					if g.apply(gameapi.SplitBand{BandID: g.selectedBand, Destination: candidate.TileID}) {
						g.clearMigrationPreview()
					}
					break
				}
			}
		}
	}
	if inpututil.IsKeyJustPressed(ebiten.KeyI) {
		g.requestInterbreed()
	}
	for index, key := range [...]ebiten.Key{ebiten.Key1, ebiten.Key2, ebiten.Key3, ebiten.Key4, ebiten.Key5, ebiten.Key6, ebiten.Key7, ebiten.Key8, ebiten.Key9} {
		if inpututil.IsKeyJustPressed(key) {
			g.apply(gameapi.ResearchTech{BandID: g.selectedBand, Tech: gameapi.Tech(index)})
		}
	}
	if inpututil.IsKeyJustPressed(ebiten.KeySpace) && g.frame.CampaignResult == gameapi.Ongoing {
		if g.assignmentDraftDirty() {
			g.showNotice("Apply or discard workforce changes before ending the turn")
			return nil
		}
		if g.hasMigrationPreview {
			g.showNotice("Press Enter to queue the migration, or Esc to clear it before ending the turn.")
			return nil
		}
		frame, err := g.port.EndTurn()
		if err == nil {
			g.acceptCompletedTurn(frame)
		} else {
			g.showNotice(ui.ErrorMessage(err))
		}
	}
	return nil
}

func (g *Game) pollUISettings() {
	if g.settingsStore == nil {
		return
	}
	for _, completion := range g.settingsStore.Poll() {
		switch completion.Operation {
		case ui.UISettingsRead:
			if !g.settingsLoading || completion.Revision != g.settingsRevision {
				continue
			}
			g.settingsLoading = false
			g.settings = ui.NormalizeUISettings(completion.Settings)
			g.fieldNotesVisible = g.settings.FieldNotesVisible
			g.sound.SetMaster(g.settings.MasterVolume, g.settings.Muted)
			if completion.Err != nil {
				g.showNotice("Preferences could not be loaded; using defaults")
			}
		case ui.UISettingsWrite:
			if !g.settingsWriteActive || completion.Revision != g.settingsRevision {
				continue
			}
			g.settingsWriteActive = false
			if completion.Err != nil {
				g.showNotice("Preferences could not be saved")
			}
			if g.pendingSettings != nil {
				pending := *g.pendingSettings
				g.pendingSettings = nil
				if pending != completion.Settings {
					g.startUISettingsWrite(pending)
				}
			}
		}
	}
}

func (g *Game) updateUISettings(settings ui.UISettings) {
	settings = ui.NormalizeUISettings(settings)
	g.settings = settings
	g.fieldNotesVisible = settings.FieldNotesVisible
	g.sound.SetMaster(settings.MasterVolume, settings.Muted)
	if g.settingsStore == nil {
		return
	}
	if g.settingsWriteActive {
		pending := settings
		g.pendingSettings = &pending
		return
	}
	g.startUISettingsWrite(settings)
}

func (g *Game) startUISettingsWrite(settings ui.UISettings) {
	g.settingsRevision++
	if err := g.settingsStore.BeginWrite(g.settingsRevision, settings); err != nil {
		g.showNotice("Preferences could not be saved")
		return
	}
	g.settingsWriteActive = true
}

func (g *Game) Draw(screen *ebiten.Image) {
	fieldNote := g.fieldNote
	fieldNote.Celebration = g.breakthroughFrames > 0
	g.scene.SetWorkforceDraft(g.workforceDraftForRender())
	displayFrame := g.displayFrame()
	g.scene.Draw(screen, displayFrame, g.selectedBand, render.MigrationPreview{
		BandID: g.migrationPreviewBand, TileID: g.migrationPreviewTile, Visible: g.hasMigrationPreview,
	}, g.notice, fieldNote, g.fieldNotesVisible, ui.CampaignEndScene(displayFrame))
	if !g.firstDrawDone {
		g.firstDrawDone = true
		if g.onFirstDraw != nil {
			g.onFirstDraw()
		}
	}
}

type technologyDiscovery struct {
	bandID     gameapi.BandID
	technology gameapi.Tech
}

func (g *Game) acceptCompletedTurn(frame *gameapi.Frame) {
	discoveries := newTechnologyDiscoveries(g.frame, frame, g.selectedBand)
	newestEvent, hasNewEvent := newestAddedEvent(g.frame, frame)
	if completedTurnAddedAcuteEvent(g.frame, frame) {
		g.sound.Play(gameaudio.SFXEventTrigger)
	}
	g.frame = frame
	g.publishFrame()
	g.ensureSelection()
	g.syncAssignmentDraft(false)
	if len(discoveries) == 0 {
		if hasNewEvent {
			g.fieldNote = ui.EventFieldNote(newestEvent)
		}
		return
	}
	first := discoveries[0]
	fieldNote, ok := ui.TechnologyFieldNote(first.technology, first.bandID, len(discoveries))
	if !ok {
		return
	}
	g.fieldNote = fieldNote
	g.breakthroughFrames = breakthroughCelebrationFrames
	message := fmt.Sprintf("Breakthrough! Band %d learned %s", first.bandID, first.technology)
	if len(discoveries) > 1 {
		message += fmt.Sprintf(" (+%d more)", len(discoveries)-1)
	}
	g.notice = message
	g.noticeFrames = 300
}

func newestAddedEvent(previous, current *gameapi.Frame) (gameapi.Event, bool) {
	if current == nil || len(current.Events) == 0 {
		return gameapi.Event{}, false
	}
	seen := make(map[gameapi.Event]struct{})
	if previous != nil {
		for _, event := range previous.Events {
			seen[event] = struct{}{}
		}
	}
	for index := len(current.Events) - 1; index >= 0; index-- {
		event := current.Events[index]
		if _, existed := seen[event]; !existed {
			return event, true
		}
	}
	return gameapi.Event{}, false
}

func completedTurnAddedAcuteEvent(previous, current *gameapi.Frame) bool {
	if current == nil {
		return false
	}
	previousCount := 0
	if previous != nil {
		for _, event := range previous.Events {
			if event.Kind == gameapi.EventAcuteIncident {
				previousCount++
			}
		}
	}
	currentCount := 0
	for _, event := range current.Events {
		if event.Kind == gameapi.EventAcuteIncident {
			currentCount++
		}
	}
	return currentCount > previousCount
}

func newTechnologyDiscoveries(previous, current *gameapi.Frame, preferredBand gameapi.BandID) []technologyDiscovery {
	if previous == nil || current == nil {
		return nil
	}
	previousBands := make(map[gameapi.BandID]gameapi.Band, len(previous.Bands))
	for _, band := range previous.Bands {
		previousBands[band.ID] = band
	}
	result := make([]technologyDiscovery, 0)
	appendBandDiscoveries := func(band gameapi.Band) {
		before, existed := previousBands[band.ID]
		if !existed || before.Species != gameapi.HomoSapiens || band.Species != gameapi.HomoSapiens {
			return
		}
		newlyAcquired := band.AcquiredTech &^ before.AcquiredTech
		for technology := gameapi.Tech(0); technology < gameapi.TechCount; technology++ {
			if newlyAcquired&(1<<technology) != 0 {
				result = append(result, technologyDiscovery{bandID: band.ID, technology: technology})
			}
		}
	}
	for _, band := range current.Bands {
		if band.ID == preferredBand {
			appendBandDiscoveries(band)
			break
		}
	}
	for _, band := range current.Bands {
		if band.ID != preferredBand {
			appendBandDiscoveries(band)
		}
	}
	return result
}

func (g *Game) SetFirstDrawCallback(callback func()) { g.onFirstDraw = callback }

func (g *Game) SetTerrainDetail(detail render.TerrainDetailMode) {
	g.scene.SetTerrainDetail(detail)
}

// SetRenderProfileFrame installs the browser's validated maximum-render
// fixture only at the drawing seam. Simulation commands, storage, hashes, and
// the semantic E2E observer continue to use the accepted live frame.
func (g *Game) SetRenderProfileFrame(frame *gameapi.Frame) { g.profileDisplayFrame = frame }

func (g *Game) displayFrame() *gameapi.Frame {
	if g.profileDisplayFrame != nil {
		return g.profileDisplayFrame
	}
	return g.frame
}

// SetFramePublishedCallback installs the opt-in semantic browser-test
// observer. The callback receives only a bounded encoding of the frame already
// held by the host; it cannot issue commands or request another snapshot.
func (g *Game) SetFramePublishedCallback(callback func(string)) {
	g.onFramePublished = callback
	g.publishFrame()
}

func (g *Game) publishFrame() {
	if g.onFramePublished != nil {
		g.onFramePublished(e2eSummaryJSON(g.frame))
	}
}

// requestInterbreed answers the interbreed key in every case. It previously
// short-circuited on an empty candidate list, so the advertised control did
// nothing at all and gave no reason — and a successful one was equally silent.
func (g *Game) requestInterbreed() {
	band := g.selected()
	if band == nil {
		return
	}
	switch {
	case band.Species != gameapi.HomoSapiens:
		g.showNotice("Only a Homo sapiens band can initiate interbreeding.")
	case band.HasInterbreedTarget:
		g.showNotice(fmt.Sprintf("This band is already interbreeding with archaic band %d this turn.", band.InterbreedTargetID))
	case band.SpatialActionUsed:
		g.showNotice("This band has already used its spatial action this turn.")
	case len(band.InterbreedCandidateIDs) == 0:
		g.showNotice("No archaic band shares this tile — move onto one first to interbreed.")
	default:
		target := band.InterbreedCandidateIDs[0]
		if g.apply(gameapi.Interbreed{BandID: band.ID, TargetBandID: target}) {
			g.clearMigrationPreview()
			g.showNotice(fmt.Sprintf("Interbreeding with archaic band %d — gene flow resolves when the turn ends.", target))
		}
	}
}

func (g *Game) selected() *gameapi.Band {
	if g.frame == nil {
		return nil
	}
	for index := range g.frame.Bands {
		if g.frame.Bands[index].ID == g.selectedBand {
			return &g.frame.Bands[index]
		}
	}
	return nil
}

func (g *Game) syncAssignmentDraft(force bool) {
	band := g.selected()
	if band == nil || band.Species != gameapi.HomoSapiens {
		g.hasAssignmentDraft = false
		return
	}
	if !force && g.hasAssignmentDraft && g.assignmentDraftBand == band.ID && g.assignmentDraftDirty() {
		return
	}
	g.assignmentDraftBand = band.ID
	g.assignmentDraft = band.AllocationBP
	g.assignmentBaseline = band.AllocationBP
	g.hasAssignmentDraft = true
	if g.assignmentRole >= gameapi.AssignmentCount {
		g.assignmentRole = gameapi.Foraging
	}
}

func (g *Game) assignmentDraftDirty() bool {
	return g.hasAssignmentDraft && g.assignmentDraft != g.assignmentBaseline
}

func (g *Game) assignmentDraftValid() bool {
	if !g.hasAssignmentDraft {
		return false
	}
	var total uint32
	for _, points := range g.assignmentDraft {
		total += uint32(points)
	}
	return total == 10_000
}

func (g *Game) editAssignmentDraft(delta int) {
	if !g.hasAssignmentDraft || g.assignmentRole >= gameapi.AssignmentCount {
		return
	}
	value := int(g.assignmentDraft[g.assignmentRole]) + delta
	value = min(max(value, 0), 10_000)
	g.assignmentDraft[g.assignmentRole] = uint16(value)
	if g.assignmentDraftDirty() {
		g.showNotice("Workforce draft changed — total must equal 100%; A applies, D discards")
	}
}

func (g *Game) applyAssignmentDraft() {
	if !g.assignmentDraftDirty() {
		return
	}
	if !g.assignmentDraftValid() {
		g.showNotice("Workforce allocation must total exactly 100%")
		return
	}
	command := gameapi.SetAssignment{BandID: g.assignmentDraftBand, AllocationBP: g.assignmentDraft}
	if g.apply(command) {
		g.syncAssignmentDraft(true)
		g.showNotice("Workforce allocation applied")
	}
}

func (g *Game) discardAssignmentDraft() {
	if !g.assignmentDraftDirty() {
		return
	}
	g.assignmentDraft = g.assignmentBaseline
	g.showNotice("Workforce changes discarded")
}

func (g *Game) workforceDraftForRender() render.WorkforceDraft {
	return render.WorkforceDraft{
		Visible: g.hasAssignmentDraft, BandID: g.assignmentDraftBand, AllocationBP: g.assignmentDraft,
		SelectedRole: g.assignmentRole, Dirty: g.assignmentDraftDirty(), Valid: g.assignmentDraftValid(),
	}
}

func (g *Game) ensureSelection() {
	if selected := g.selected(); selected != nil && selected.Species == gameapi.HomoSapiens {
		return
	}
	g.selectedBand = 0
	if g.frame == nil {
		return
	}
	for _, band := range g.frame.Bands {
		if band.Species == gameapi.HomoSapiens {
			g.selectedBand = band.ID
			return
		}
	}
}

func (g *Game) selectNextSapiens() {
	g.selectSapiens(1)
}

func (g *Game) selectPreviousSapiens() {
	g.selectSapiens(-1)
}

func (g *Game) selectSapiens(offset int) {
	if g.frame == nil {
		return
	}
	if g.assignmentDraftDirty() {
		g.showNotice("Apply or discard workforce changes")
		return
	}
	bandIDs := make([]gameapi.BandID, 0, len(g.frame.Bands))
	selectedIndex := -1
	for _, band := range g.frame.Bands {
		if band.Species != gameapi.HomoSapiens {
			continue
		}
		if band.ID == g.selectedBand {
			selectedIndex = len(bandIDs)
		}
		bandIDs = append(bandIDs, band.ID)
	}
	if len(bandIDs) == 0 {
		g.selectedBand = 0
		return
	}
	if selectedIndex < 0 {
		g.selectedBand = bandIDs[0]
		g.syncAssignmentDraft(true)
		g.refreshBandFieldNote()
		return
	}
	selectedIndex = (selectedIndex + offset + len(bandIDs)) % len(bandIDs)
	g.selectedBand = bandIDs[selectedIndex]
	g.syncAssignmentDraft(true)
	g.refreshBandFieldNote()
}

func (g *Game) refreshBandFieldNote() {
	if band := g.selected(); band != nil {
		g.fieldNote = ui.BandContextFieldNote(g.frame, band)
	}
}

func (g *Game) handleMapClick() {
	tileID, ok := render.MapTileAt(ebiten.CursorPosition())
	if !ok {
		return
	}
	g.clearMigrationPreview()
	for _, band := range g.frame.Bands {
		if band.Species == gameapi.HomoSapiens && band.TileID == tileID {
			if band.ID != g.selectedBand && g.assignmentDraftDirty() {
				g.showNotice("Apply or discard workforce changes")
				return
			}
			g.selectedBand = band.ID
			g.syncAssignmentDraft(true)
			g.refreshBandFieldNote()
			return
		}
	}
	band := g.selected()
	if band == nil {
		return
	}
	g.tryQueueMigration(band, tileID)
}

func (g *Game) handleDirectionalMigration(dx, dy int) {
	band := g.selected()
	if band == nil {
		return
	}
	if band.SpatialActionUsed {
		g.showNotice(ui.MigrationDiagnosticMessage(ui.MigrationDiagnostic{Reason: ui.MigrationBlockedActionSpent}, band))
		return
	}
	cursor := band.TileID
	if g.hasMigrationPreview && g.migrationPreviewBand == band.ID {
		cursor = g.migrationPreviewTile
	}
	tileID, ok := ui.MoveMigrationPreview(g.frame, band, cursor, dx, dy)
	if !ok {
		g.showNotice("The migration cursor must stay within this band's one-turn neighborhood.")
		return
	}
	if tileID == band.TileID {
		g.clearMigrationPreview()
		g.showNotice("Migration choice cleared")
		return
	}
	g.migrationPreviewBand = band.ID
	g.migrationPreviewTile = tileID
	g.hasMigrationPreview = true
	diagnostic := ui.DiagnoseMigration(g.frame, band, tileID)
	if diagnostic.Reason == ui.MigrationAllowed {
		g.showNotice("Destination selected — press Enter to queue migration.")
		return
	}
	g.showNotice(ui.MigrationDiagnosticMessage(diagnostic, band) + " Keep using arrows, or press Esc to clear.")
}

func (g *Game) confirmMigrationPreview() {
	if !g.hasMigrationPreview {
		return
	}
	band := g.selected()
	if band == nil || band.ID != g.migrationPreviewBand {
		g.clearMigrationPreview()
		return
	}
	if g.tryQueueMigration(band, g.migrationPreviewTile) {
		g.clearMigrationPreview()
	}
}

func (g *Game) tryQueueMigration(band *gameapi.Band, tileID gameapi.TileID) bool {
	diagnostic := ui.DiagnoseMigration(g.frame, band, tileID)
	if diagnostic.Reason == ui.MigrationAllowed {
		return g.apply(gameapi.QueueMigration{BandID: band.ID, TileID: tileID})
	}
	g.showNotice(ui.MigrationDiagnosticMessage(diagnostic, band))
	return false
}

func (g *Game) apply(command gameapi.Command) bool {
	if frame, err := g.port.Apply(command); err == nil {
		g.frame = frame
		g.publishFrame()
		g.ensureSelection()
		g.sound.Play(gameaudio.SFXChoiceClick)
		return true
	} else {
		g.showNotice(ui.ErrorMessage(err))
		return false
	}
}

func (g *Game) startNewCampaign() {
	frame, err := g.port.NewCampaign()
	if err != nil {
		g.showNotice("Could not start a new campaign: " + ui.ErrorMessage(err))
		return
	}
	g.frame = frame
	g.publishFrame()
	g.selectedBand = 0
	g.ensureSelection()
	g.hasAssignmentDraft = false
	g.syncAssignmentDraft(true)
	g.fieldNote = ui.CampaignOverviewFieldNote()
	g.breakthroughFrames = 0
	g.clearMigrationPreview()
	g.sound.Play(gameaudio.SFXChoiceClick)
	g.showNotice("New campaign begun")
}

func (g *Game) clearMigrationPreview() {
	g.hasMigrationPreview = false
	g.migrationPreviewBand = 0
	g.migrationPreviewTile = 0
}

func (g *Game) beginQuickSave() {
	operationID, err := g.port.BeginSave(99)
	if err != nil {
		g.showNotice("Save failed: " + ui.ErrorMessage(err))
		return
	}
	g.pendingQuickSaveIDs[operationID] = struct{}{}
}

func (g *Game) beginManualSave(slot int) {
	if _, err := g.port.BeginSave(slot); err != nil {
		g.showNotice("Save failed: " + ui.ErrorMessage(err))
		return
	}
	g.showNotice(fmt.Sprintf("Saving Manual %d…", slot))
}

func (g *Game) beginManualLoad(slot int) {
	if g.assignmentDraftDirty() {
		g.showNotice("Apply or discard workforce changes before loading")
		return
	}
	operationID, err := g.port.BeginLoad(slot)
	if err != nil {
		g.showNotice("Load failed: " + ui.ErrorMessage(err))
		return
	}
	g.pendingManualLoadID = operationID
	g.showNotice(fmt.Sprintf("Loading Manual %d…", slot))
}

func (g *Game) beginStartupResume() {
	operationID, err := g.port.BeginListSlots()
	if err != nil {
		g.showNotice("Storage failed: " + ui.ErrorMessage(err))
		return
	}
	g.startupRestorePending = true
	g.startupRestoreListID = operationID
}

func (g *Game) pollStorage() {
	for _, result := range g.port.PollStorage() {
		if result.Operation == gameapi.StorageSave && result.Slot == 99 {
			delete(g.pendingQuickSaveIDs, result.OperationID)
		}

		isStartupList := g.startupRestorePending && result.Operation == gameapi.StorageList && result.OperationID == g.startupRestoreListID
		isStartupLoad := g.startupRestorePending && result.Operation == gameapi.StorageLoad && result.OperationID == g.startupRestoreLoadID
		isManualLoad := result.Operation == gameapi.StorageLoad && result.OperationID == g.pendingManualLoadID
		if isStartupList {
			g.startupRestoreListID = 0
			if result.Err == nil {
				if slot, ok := newestResumeSlot(result.Slots); ok {
					operationID, err := g.port.BeginLoad(slot)
					if err != nil {
						g.startupRestorePending = false
						g.showNotice("Storage failed: " + ui.ErrorMessage(err))
					} else {
						g.startupRestoreLoadID = operationID
						g.startupRestoreSlot = slot
					}
				}
				if g.startupRestoreLoadID == 0 {
					g.startupRestorePending = false
				}
			} else {
				g.startupRestorePending = false
			}
		}

		if result.ReplacementFrame != nil {
			g.frame = result.ReplacementFrame
			g.publishFrame()
			g.fieldNote = ui.CampaignOverviewFieldNote()
			g.breakthroughFrames = 0
			g.clearMigrationPreview()
			g.ensureSelection()
			g.hasAssignmentDraft = false
			g.syncAssignmentDraft(true)
		}
		if isStartupLoad {
			g.startupRestorePending = false
			g.startupRestoreLoadID = 0
		}
		if isManualLoad {
			g.pendingManualLoadID = 0
		}
		if result.Err != nil {
			if isStartupLoad {
				g.startupRestoreSlot = 0
			}
			g.showNotice("Storage failed: " + ui.ErrorMessage(result.Err))
			continue
		}
		switch {
		case isStartupLoad:
			if g.startupRestoreSlot == 99 {
				g.showNotice("Quick save restored")
			} else {
				g.showNotice(fmt.Sprintf("Autosave restored — Auto %d", g.startupRestoreSlot-100))
			}
			g.startupRestoreSlot = 0
		case isManualLoad:
			g.showNotice(fmt.Sprintf("Loaded Manual %d", result.Slot))
		case result.Operation == gameapi.StorageLoad:
			g.showNotice("Game loaded")
		case result.Operation == gameapi.StorageSave && result.Slot >= 1 && result.Slot <= 3:
			g.showNotice(fmt.Sprintf("Saved Manual %d", result.Slot))
		case result.Operation == gameapi.StorageSave && result.Slot == 99:
			g.sound.Play(gameaudio.SFXSaveComplete)
			g.showNotice("Quick-saved")
		case result.Operation == gameapi.StorageSave && result.Slot >= 101 && result.Slot <= 103:
			g.showNotice("Autosaved")
		}
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

func (g *Game) showNotice(message string) {
	g.notice = message
	g.noticeFrames = 120
}

func (g *Game) LayoutF(_, _ float64) (float64, float64) {
	return LogicalWidth, LogicalHeight
}

func (g *Game) Layout(_, _ int) (int, int) { return LogicalWidth, LogicalHeight }

var _ ebiten.LayoutFer = (*Game)(nil)
var _ ebiten.Game = (*Game)(nil)
