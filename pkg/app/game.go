package app

import (
	"fmt"
	"hash/maphash"

	"github.com/adsouza/africa2ice/internal/adapters/logging"
	"github.com/adsouza/africa2ice/internal/application"
	gameaudio "github.com/adsouza/africa2ice/pkg/audio"
	"github.com/adsouza/africa2ice/pkg/gameapi"
	"github.com/adsouza/africa2ice/pkg/hud"
	"github.com/adsouza/africa2ice/pkg/render"
	"github.com/adsouza/africa2ice/pkg/ui"
	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
)

// chromeRevisionSeed is fixed for the process's lifetime: MapScene compares
// chrome revisions across ticks (via mapFrameKey), so hashing the same
// PresentationKey with a different seed between calls would look like a
// change and force a spurious repaint.
var chromeRevisionSeed = maphash.MakeSeed()

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
	fieldNote              render.FieldNote
	breakthroughFrames     int
	migrationPreviewBand   gameapi.BandID
	migrationPreviewTile   gameapi.TileID
	hasMigrationPreview    bool
	hoveredTile            gameapi.TileID
	hasHoveredTile         bool
	startupRestorePending  bool
	startupRestoreListID   gameapi.StorageOpID
	startupRestoreLoadID   gameapi.StorageOpID
	startupRestoreSlot     int
	pendingQuickSaveIDs    map[gameapi.StorageOpID]struct{}
	pendingSaveSoundIDs    map[gameapi.StorageOpID]struct{}
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
	viewport               render.Viewport
	viewportInitialized    bool
	deviceScaleFactor      func() float64
	scenes                 ui.SceneStack
	storageMode            storageBrowserMode
	storageSlots           []gameapi.SlotMetadata
	storageSelection       int
	storageListID          gameapi.StorageOpID
	storageOperationID     gameapi.StorageOpID
	toasts                 ui.ToastManager
	traitFocus             gameapi.HeritableTrait
	interbreedFocus        gameapi.BandID
	regionalPulseFocused   bool
	logSession             *logging.Session
	panel                  *hud.Panel
	openRow                ui.ChecklistRow
	rowChosen              bool // player opened a row explicitly; auto-advance yields until reset
	detailsOpen            bool
	bandListOpen           bool
	endTurnArmed           bool
	guide                  ui.GuideState
	notesMode              hud.NotesMode
	shortcutsOpen          bool
	researchCursor         gameapi.Tech
	camera                 render.Camera
	cameraOverride         bool
}

const (
	fieldNotesHotkey = ebiten.KeyF
	splitBandHotkey  = ebiten.KeyN
)

type storageBrowserMode uint8

const (
	storageBrowserSave storageBrowserMode = iota
	storageBrowserLoad
)

var storageBrowserSlots = [...]int{1, 2, 3, 99, 101, 102, 103}

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
		port: port, sound: sound, frame: frame, scene: render.NewMapScene(),
		panel: hud.New(), notesMode: hud.NotesCompact, guide: ui.NewGuideState(false),
		fieldNote: ui.CampaignOverviewFieldNote(),
		notice:    "Outlined tiles are reachable — arrows choose, Enter confirms", noticeFrames: 300,
		pendingQuickSaveIDs: make(map[gameapi.StorageOpID]struct{}),
		pendingSaveSoundIDs: make(map[gameapi.StorageOpID]struct{}),
		settingsStore:       settingsStore, settings: settings,
		deviceScaleFactor: func() float64 { return ebiten.Monitor().DeviceScaleFactor() },
		scenes:            ui.NewSceneStack(),
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
	game.resetDisclosure()
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
	game.logSession = session
	game.scenes.Push(ui.SceneTitle)
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
	g.advanceToasts()
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
	if g.viewportInitialized && !g.viewport.SupportsGameplay() {
		return nil
	}
	g.stepCamera()
	intents := g.panel.Update(g.hudState())
	// One record per left-button press (never per frame): the cursor
	// position, whether the chrome claimed the pointer, and how many
	// intents this tick produced. This is the seam that let the New
	// Campaign click go undiagnosable from a session log — the gap between
	// a pointer going down and an action dispatch.
	if inpututil.IsMouseButtonJustPressed(ebiten.MouseButtonLeft) {
		cursorX, cursorY := ebiten.CursorPosition()
		g.logSession.LogUIPointer(cursorX, cursorY, g.panel.Hovered(), len(intents))
	}
	// Overlay intents are handled in every scene: the panel column's own
	// buttons are gameplay-only, but a modal window (title/menu/storage/
	// settings, or the shortcut sheet) blocks pointer input to whatever sits
	// beneath it, so the widgets that can actually emit an intent while a
	// non-gameplay scene is showing are exactly the overlay's own.
	g.handleIntents(intents)
	modifier := ebiten.IsKeyPressed(ebiten.KeyControl) || ebiten.IsKeyPressed(ebiten.KeyMeta)
	if modifier && inpututil.IsKeyJustPressed(ebiten.KeyS) && g.scenes.Current() == ui.SceneGameplay {
		g.beginQuickSave()
	}
	for index, key := range [...]ebiten.Key{ebiten.KeyF1, ebiten.KeyF2, ebiten.KeyF3} {
		if g.scenes.Current() != ui.SceneGameplay {
			break
		}
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
	if g.scenes.Current() != ui.SceneGameplay {
		g.hasHoveredTile = false
		g.handleSceneInput()
		return nil
	}
	g.syncTileHover()
	if g.frame.CampaignResult != gameapi.Ongoing {
		// The terminal scene's New campaign button belongs to the panel now; the
		// key remains the application's own path for it.
		if inpututil.IsKeyJustPressed(ebiten.KeyN) {
			g.startNewCampaign()
		}
		return nil
	}
	if g.handleSceneInput() {
		return nil
	}
	if inpututil.IsMouseButtonJustPressed(ebiten.MouseButtonLeft) && !g.panel.Hovered() {
		g.handleMapClick()
	}
	g.handleGameplayKeys()
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
			// A completed read is the one install that may seed UI-local state
			// from preferences; later writes must not rewind the live guide.
			g.notesMode = notesModeFor(g.settings)
			g.guide = ui.NewGuideState(g.settings.GuideDismissed)
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
	g.notesMode = notesModeFor(settings)
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
	g.scene.SetTileHover(render.TileHover{TileID: g.hoveredTile, Visible: g.hasHoveredTile})
	g.scene.SetCamera(g.camera, g.mapVisibleHeight())
	g.scene.SetGuideHighlight(g.guide.Step == ui.GuideMove && g.frame != nil && g.frame.CampaignResult == gameapi.Ongoing)
	// The chrome (pkg/hud) draws over this image and can change what it
	// looks like — a rebuild, or an in-place refresh — without any of the
	// fields above changing, so pkg/render's cache key must see it too:
	// hash the panel's PresentationKey into the opaque revision MapScene
	// accepts (pkg/render must not import pkg/hud) and fold it into the
	// map's own key. That is what makes the map repaint when only the
	// chrome changed (details collapsing, the drawer shrinking, a settings
	// window closing) even though the map itself did not.
	g.scene.SetChromeRevision(maphash.Comparable(chromeRevisionSeed, g.panel.PresentationKey()))
	displayFrame := g.displayFrame()
	painted := g.scene.Draw(screen, displayFrame, g.selectedBand, render.MigrationPreview{
		BandID: g.migrationPreviewBand, TileID: g.migrationPreviewTile, Visible: g.hasMigrationPreview,
	}, g.notice, g.endScene(displayFrame), g.viewportInitialized && !g.viewport.SupportsGameplay())
	// The chrome draws over the map image rather than into it, so the two
	// layers must always paint together and never separately: painting the
	// panel alone over a stale map (or vice versa) leaves stale pixels
	// exactly like the bug this replaces. The too-small overlay can only
	// stay the topmost thing if the panel yields.
	if painted && (!g.viewportInitialized || g.viewport.SupportsGameplay()) {
		g.panel.Draw(screen)
	}
	// Browser readiness means the first frame is visible and input is accepted.
	// IndexedDB discovery may still be resolving on earlier draws; announcing
	// readiness there lets the first gesture disappear into the startup guard.
	// This must still run on a skipped frame: readiness is about whether a
	// frame has ever been shown, not whether this particular tick painted.
	if !g.firstDrawDone && !g.startupRestorePending {
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
	previous := g.frame
	discoveries := newTechnologyDiscoveries(previous, frame, g.selectedBand)
	newestEvent, hasNewEvent := newestAddedEvent(previous, frame)
	newRegion, hasNewRegion := newlyEstablishedRegion(previous, frame)
	if completedTurnAddedAcuteEvent(previous, frame) {
		g.sound.Play(gameaudio.SFXEventTrigger)
	}
	g.frame = frame
	// A completed turn refreshes the attention order from its new actuals and
	// projections. Start the next planning period at the band most in need.
	g.selectedBand = 0
	g.ensureSelection()
	pulseNote, hasPulse := currentRegionalPulseFieldNote(frame, g.selectedBand)
	previousPulseNote, hadPulse := currentRegionalPulseFieldNote(previous, g.selectedBand)
	if !hasPulse || !hadPulse || pulseNote.Topic != previousPulseNote.Topic {
		g.regionalPulseFocused = false
	}
	g.publishFrame()
	g.syncAssignmentDraft(false)
	g.resetDisclosure()
	g.guide = g.guide.ObserveTurnCompleted()
	if len(discoveries) == 0 {
		switch {
		case hasNewRegion:
			if note, ok := ui.RegionEstablishedFieldNote(newRegion); ok {
				g.setFieldNote(note)
			}
		case hasCurrentMacroContext(frame):
			note, _ := currentMacroFieldNote(frame)
			g.setFieldNote(note)
		case crossedTobaMarker(previous, frame):
			g.setFieldNote(ui.TobaFieldNote())
		case previous != nil && previous.Climate.Epoch != frame.Climate.Epoch:
			if note, ok := ui.ClimateEpochFieldNote(frame.Climate.Epoch); ok {
				g.setFieldNote(note)
			}
		case hasNewEvent:
			g.setFieldNote(ui.EventFieldNote(newestEvent))
		case hasPulse && !g.regionalPulseFocused:
			g.setFieldNote(pulseNote)
			g.regionalPulseFocused = true
		}
		return
	}
	first := discoveries[0]
	fieldNote, ok := ui.TechnologyFieldNote(first.technology, first.bandID, len(discoveries))
	if !ok {
		return
	}
	g.setFieldNote(fieldNote)
	g.breakthroughFrames = breakthroughCelebrationFrames
	message := fmt.Sprintf("Breakthrough! Band %d learned %s", first.bandID, first.technology)
	if len(discoveries) > 1 {
		message += fmt.Sprintf(" (+%d more)", len(discoveries)-1)
	}
	g.showNoticeFor(message, 300)
}

func hasCurrentMacroContext(frame *gameapi.Frame) bool {
	_, ok := currentMacroFieldNote(frame)
	return ok
}

func currentRegionalPulseFieldNote(frame *gameapi.Frame, selectedBand gameapi.BandID) (render.FieldNote, bool) {
	var band *gameapi.Band
	if frame != nil {
		for index := range frame.Bands {
			if frame.Bands[index].ID == selectedBand {
				band = &frame.Bands[index]
				break
			}
		}
	}
	if band == nil || int(band.TileID) >= len(frame.Tiles) {
		return render.FieldNote{}, false
	}
	region := frame.Tiles[band.TileID].Region
	if region >= gameapi.RegionCount || frame.Climate.RegionalAbrupt[region] == 0 {
		return render.FieldNote{}, false
	}
	return ui.AbruptClimateFieldNote(region, frame.Climate.RegionalAbrupt[region])
}

func crossedTobaMarker(before, after *gameapi.Frame) bool {
	return before != nil && after != nil && before.YearBP > 73_880 && after.YearBP <= 73_880
}

func (g *Game) focusNextTraitNote() {
	band := g.selected()
	if band == nil {
		return
	}
	trait := g.traitFocus % gameapi.HeritableTraitCount
	if note, ok := ui.TraitFieldNote(trait, band.HeritableState[trait]); ok {
		g.setFieldNote(note)
		g.showNotice("Genetics: " + trait.String())
	}
	g.traitFocus = (trait + 1) % gameapi.HeritableTraitCount
}

func newlyEstablishedRegion(before, after *gameapi.Frame) (gameapi.Region, bool) {
	if after == nil {
		return 0, false
	}
	existing := make(map[gameapi.Region]struct{})
	if before != nil {
		for _, region := range before.SapiensEstablishedRegions {
			existing[region] = struct{}{}
		}
	}
	for _, region := range after.SapiensEstablishedRegions {
		if _, ok := existing[region]; !ok {
			return region, true
		}
	}
	return 0, false
}

func currentMacroFieldNote(frame *gameapi.Frame) (render.FieldNote, bool) {
	if frame == nil {
		return render.FieldNote{}, false
	}
	for _, episode := range frame.MacroEpisodes {
		if episode.Warned || episode.Current {
			return ui.MacroEpisodeFieldNote(episode)
		}
	}
	return render.FieldNote{}, false
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
	seen := make(map[gameapi.Event]struct{})
	if previous != nil {
		for _, event := range previous.Events {
			seen[event] = struct{}{}
		}
	}
	for index := len(current.Events) - 1; index >= 0; index-- {
		event := current.Events[index]
		if event.Kind == gameapi.EventAcuteIncident {
			if _, existed := seen[event]; !existed {
				return true
			}
		}
	}
	return false
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

// endScene is the terminal presentation, suppressed while the title is up:
// the title is the application's front door, and a finished campaign behind
// it reads as two competing dialogs rather than one. Game.Draw and
// hudState() both call this instead of ui.CampaignEndScene directly, so the
// render layer and the chrome always agree. The dialog reappears as soon as
// the player leaves the title with Continue; the menu, storage and settings
// scenes are modal windows the player opened deliberately over a visible
// dialog, ordinary layering that is left alone.
func (g *Game) endScene(frame *gameapi.Frame) render.EndScene {
	if g.scenes.Current() == ui.SceneTitle {
		return render.EndScene{}
	}
	return ui.CampaignEndScene(frame)
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
	if len(band.InterbreedCandidateIDs) > 0 {
		g.setFieldNote(ui.InterbreedingFieldNote(len(band.InterbreedCandidateIDs)))
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
		target := g.interbreedFocus
		if target == 0 {
			target = band.InterbreedCandidateIDs[0]
		}
		if g.apply(gameapi.Interbreed{BandID: band.ID, TargetBandID: target}) {
			g.clearMigrationPreview()
			g.showNotice(fmt.Sprintf("Interbreeding with archaic band %d — gene flow resolves when the turn ends.", target))
			g.advanceOpenRow()
		}
	}
}

func (g *Game) selectNextInterbreedTarget() {
	band := g.selected()
	if band == nil || len(band.InterbreedCandidateIDs) == 0 {
		g.showNotice("No eligible archaic interbreeding target shares this tile.")
		return
	}
	index := -1
	for candidateIndex, candidate := range band.InterbreedCandidateIDs {
		if candidate == g.interbreedFocus {
			index = candidateIndex
			break
		}
	}
	g.interbreedFocus = band.InterbreedCandidateIDs[(index+1)%len(band.InterbreedCandidateIDs)]
	g.setFieldNote(ui.InterbreedingFieldNote(len(band.InterbreedCandidateIDs)))
	g.showNotice(fmt.Sprintf("Interbreeding target: archaic band %d", g.interbreedFocus))
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

// desiredCameraMode applies spec §6: focus while the Move row is open and the
// selected sapiens band still has its spatial action, inverted by Z.
func (g *Game) desiredCameraMode() render.CameraMode {
	band := g.selected()
	auto := g.openRow == ui.RowMove && band != nil && band.Species == gameapi.HomoSapiens && !ui.MoveDone(*band)
	if g.cameraOverride {
		auto = !auto
	}
	if auto {
		return render.CameraFocus
	}
	return render.CameraOverview
}

// stepCamera runs once per Update: retarget, then advance the transition.
func (g *Game) stepCamera() {
	g.camera.Mode = g.desiredCameraMode()
	if band := g.selected(); band != nil {
		g.camera.CenterTile = band.TileID
	}
	g.camera = g.camera.Step()
}

// mapVisibleHeight is the map area's height above the Field Notes drawer, so
// hover/click picking and the camera geometry agree with what the drawer
// leaves on screen (spec §6).
func (g *Game) mapVisibleHeight() float64 {
	switch g.notesMode {
	case hud.NotesCompact:
		return 626 - hud.DrawerCompactHeight
	case hud.NotesExpanded:
		return 626 - hud.DrawerExpandedHeight
	default:
		return 626 - hud.DrawerHiddenHeight
	}
}

func (g *Game) toggleCameraOverride() { g.cameraOverride = !g.cameraOverride }

// toggleDetails flips the band details disclosure (spec §8). It is `D`'s
// global meaning; while the Workforce row is open, D is row-owned instead
// (handleRowKey's ui.RowWorkforce case discards the draft), matching the
// row-owned model arrows, Enter, and -/+ already use there.
func (g *Game) toggleDetails() { g.detailsOpen = !g.detailsOpen }

func (g *Game) syncAssignmentDraft(force bool) {
	band := g.selected()
	g.syncInterbreedFocus(band)
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

func (g *Game) syncInterbreedFocus(band *gameapi.Band) {
	if band == nil || len(band.InterbreedCandidateIDs) == 0 {
		g.interbreedFocus = 0
		return
	}
	for _, candidate := range band.InterbreedCandidateIDs {
		if candidate == g.interbreedFocus {
			return
		}
	}
	g.interbreedFocus = band.InterbreedCandidateIDs[0]
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

func (g *Game) ensureSelection() {
	if selected := g.selected(); selected != nil && selected.Species == gameapi.HomoSapiens && selected.Population > 0 {
		return
	}
	g.selectedBand = 0
	if g.frame == nil {
		return
	}
	bandIDs := ui.SapiensBandIDsByAttention(g.frame.Bands)
	if len(bandIDs) > 0 {
		g.selectedBand = bandIDs[0]
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
	bandIDs := ui.SapiensBandIDsByAttention(g.frame.Bands)
	selectedIndex := -1
	for index, bandID := range bandIDs {
		if bandID == g.selectedBand {
			selectedIndex = index
			break
		}
	}
	if len(bandIDs) == 0 {
		g.selectedBand = 0
		return
	}
	if selectedIndex < 0 {
		g.selectedBand = bandIDs[0]
		g.syncAssignmentDraft(true)
		g.refreshBandFieldNote()
		g.resetDisclosure()
		return
	}
	selectedIndex = (selectedIndex + offset + len(bandIDs)) % len(bandIDs)
	g.selectedBand = bandIDs[selectedIndex]
	g.syncAssignmentDraft(true)
	g.refreshBandFieldNote()
	g.resetDisclosure()
}

func (g *Game) refreshBandFieldNote() {
	if band := g.selected(); band != nil {
		g.setFieldNote(ui.BandContextFieldNote(g.frame, band))
	}
}

func (g *Game) handleMapClick() {
	x, y, inside := g.logicalCursorPosition()
	if !inside {
		return
	}
	tileID, ok := g.pickTile(x, y)
	if !ok {
		return
	}
	g.clearMigrationPreview()
	if g.selectBandAtTile(tileID) {
		return
	}
	band := g.selected()
	if band == nil {
		return
	}
	g.tryQueueMigration(band, tileID)
}

func (g *Game) syncTileHover() {
	g.hasHoveredTile = false
	// The chrome sits over the map's right edge and bottom drawer; a pointer
	// there must not also light a tile underneath it.
	if g.panel.Hovered() {
		return
	}
	x, y, inside := g.logicalCursorPosition()
	tileID, ok := g.exploredHoverTile(x, y, inside)
	if !ok {
		return
	}
	g.hoveredTile = tileID
	g.hasHoveredTile = true
}

// exploredHoverTile is the camera-aware, drawer-aware pick shared by hover
// and clicks (spec §6): it reads whatever the map is currently showing, not
// a fixed overview grid.
func (g *Game) exploredHoverTile(x, y int, inside bool) (gameapi.TileID, bool) {
	if g.frame == nil || !inside {
		return 0, false
	}
	tileID, ok := g.pickTile(x, y)
	if !ok || !g.frame.Tiles[tileID].Explored {
		return 0, false
	}
	return tileID, true
}

// pickTile resolves a logical pointer position against the live camera and
// drawer-aware visible height (spec §6). It is the one geometry lookup
// shared by hover (which then filters to explored tiles above) and clicks
// (which must still reach fogged tiles so tryQueueMigration's diagnostic
// fires) — a stale copy of the camera, refreshed only in Draw, would let a
// click resolve a different tile than the hover highlight mid-transition.
func (g *Game) pickTile(x, y int) (gameapi.TileID, bool) {
	if g.frame == nil {
		return 0, false
	}
	tileID, ok := render.MapTileAt(g.camera, g.frame, g.mapVisibleHeight(), x, y)
	if !ok || int(tileID) >= len(g.frame.Tiles) {
		return 0, false
	}
	return tileID, true
}

func (g *Game) selectBandAtTile(tileID gameapi.TileID) bool {
	if g.frame == nil || int(tileID) >= len(g.frame.Tiles) {
		return false
	}
	bandIDs := make([]gameapi.BandID, 0, 2)
	selectedIndex := -1
	for _, band := range g.frame.Bands {
		if band.TileID != tileID {
			continue
		}
		if band.Species == gameapi.ArchaicHominin && !g.frame.Tiles[tileID].Explored {
			continue
		}
		if band.ID == g.selectedBand {
			selectedIndex = len(bandIDs)
		}
		bandIDs = append(bandIDs, band.ID)
	}
	if len(bandIDs) == 0 {
		return false
	}
	next := 0
	if selectedIndex >= 0 {
		next = (selectedIndex + 1) % len(bandIDs)
	}
	if bandIDs[next] == g.selectedBand {
		return true
	}
	if g.assignmentDraftDirty() {
		g.showNotice("Apply or discard workforce changes")
		return true
	}
	g.selectedBand = bandIDs[next]
	g.clearMigrationPreview()
	g.syncAssignmentDraft(true)
	g.refreshBandFieldNote()
	g.resetDisclosure()
	return true
}

func (g *Game) chooseResearchTechnology(technology gameapi.Tech) {
	band := g.selected()
	if band == nil {
		g.showNotice("Select a Homo sapiens band to choose research.")
		return
	}
	if note, ok := ui.TechnologyContextFieldNote(technology, band); ok {
		g.setFieldNote(note)
	}
	if band.Species != gameapi.HomoSapiens {
		g.showNotice("Archaic research is computer controlled; its DAG is read only.")
		return
	}
	if g.apply(gameapi.ResearchTech{BandID: band.ID, Tech: technology}) {
		g.advanceOpenRow()
	}
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
		g.focusPassageForCandidate(band, tileID)
		g.showNotice("Destination selected — press Enter to queue migration.")
		return
	}
	g.showNotice(ui.MigrationDiagnosticMessage(diagnostic, band) + " Keep using arrows, or press Esc to clear.")
}

func (g *Game) focusPassageForCandidate(band *gameapi.Band, tileID gameapi.TileID) {
	if band == nil {
		return
	}
	for _, candidate := range band.MigrationCandidates {
		if candidate.TileID != tileID || !candidate.RequiresPassage {
			continue
		}
		if note, ok := ui.PassageFieldNote(candidate.Passage, band.PassageStatuses[candidate.Passage]); ok {
			g.setFieldNote(note)
		}
		return
	}
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
		if g.apply(gameapi.QueueMigration{BandID: band.ID, TileID: tileID}) {
			g.advanceOpenRow()
			return true
		}
		return false
	}
	g.showNotice(ui.MigrationDiagnosticMessage(diagnostic, band))
	return false
}

func (g *Game) apply(command gameapi.Command) bool {
	_, accepted := g.dispatchBatch([]ui.Action{ui.SimulationAction(command)})
	return accepted
}

type actionBatchResult struct {
	operationID gameapi.StorageOpID
}

// dispatchBatch is the only UI-to-use-case dispatch boundary. It validates the
// complete typed batch before invoking any member and preserves accepted
// earlier campaign commands if a later command is rejected semantically.
func (g *Game) dispatchBatch(actions []ui.Action) (actionBatchResult, bool) {
	if _, err := ui.ValidateActionBatch(actions); err != nil {
		g.logSession.LogActionRejected(len(actions), err)
		g.showNotice("Internal input error: " + err.Error())
		return actionBatchResult{}, false
	}
	result := actionBatchResult{}
	for _, action := range actions {
		g.logSession.LogActionDispatch(action)
		var err error
		switch action.Kind() {
		case ui.ActionSimulationCommand:
			var frame *gameapi.Frame
			frame, err = g.port.Apply(action.Command())
			if err == nil {
				g.frame = frame
				g.publishFrame()
				g.ensureSelection()
				g.guide = g.guide.Observe(g.selected())
				g.sound.Play(gameaudio.SFXChoiceClick)
			}
		case ui.ActionEndTurn:
			var frame *gameapi.Frame
			frame, err = g.port.EndTurn()
			if err == nil {
				g.acceptCompletedTurn(frame)
			}
		case ui.ActionSave:
			result.operationID, err = g.port.BeginSave(action.Slot())
			if err == nil {
				if action.Slot() == 99 {
					g.pendingQuickSaveIDs[result.operationID] = struct{}{}
				}
				if action.Slot() == 99 || action.Slot() >= 1 && action.Slot() <= 3 {
					g.pendingSaveSoundIDs[result.operationID] = struct{}{}
				}
			}
		case ui.ActionLoad:
			result.operationID, err = g.port.BeginLoad(action.Slot())
		case ui.ActionDelete:
			result.operationID, err = g.port.BeginDelete(action.Slot())
		case ui.ActionListSlots:
			result.operationID, err = g.port.BeginListSlots()
		case ui.ActionNavigate:
			navigation, scene := action.Navigation()
			switch navigation {
			case ui.NavigationPush:
				if !g.scenes.Push(scene) {
					err = fmt.Errorf("cannot push scene %d", scene)
				}
			case ui.NavigationPop:
				if !g.scenes.Pop() {
					err = fmt.Errorf("cannot pop the root scene")
				}
			case ui.NavigationReset:
				g.scenes.Reset()
			}
			if err == nil && navigation != ui.NavigationReset {
				g.sound.Play(gameaudio.SFXChoiceClick)
			}
		}
		if err != nil {
			g.showNotice(ui.ErrorMessage(err))
			return result, false
		}
	}
	return result, true
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
	g.resetDisclosure()
	if !g.settings.GuideDismissed {
		g.guide = ui.NewGuideState(false)
	}
	g.setFieldNote(ui.CampaignOverviewFieldNote())
	g.breakthroughFrames = 0
	g.regionalPulseFocused = false
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
	g.dispatchBatch([]ui.Action{ui.SaveAction(99)})
}

func (g *Game) beginManualSave(slot int) {
	if _, accepted := g.dispatchBatch([]ui.Action{ui.SaveAction(slot)}); !accepted {
		return
	}
	g.showNotice(fmt.Sprintf("Saving Manual %d…", slot))
}

func (g *Game) beginManualLoad(slot int) {
	if g.assignmentDraftDirty() {
		g.showNotice("Apply or discard workforce changes before loading")
		return
	}
	result, accepted := g.dispatchBatch([]ui.Action{ui.LoadAction(slot)})
	if !accepted {
		return
	}
	g.pendingManualLoadID = result.operationID
	g.showNotice(fmt.Sprintf("Loading Manual %d…", slot))
}

func (g *Game) handleSceneInput() bool {
	switch g.scenes.Current() {
	case ui.SceneGameplay:
		if inpututil.IsKeyJustPressed(ebiten.KeyEscape) {
			if !g.escape() {
				g.dispatchBatch([]ui.Action{ui.PushSceneAction(ui.SceneMenu)})
			}
			return true
		}
		return false
	case ui.SceneTitle:
		if inpututil.IsKeyJustPressed(ebiten.KeyEnter) || inpututil.IsKeyJustPressed(ebiten.KeyC) {
			g.dispatchBatch([]ui.Action{ui.PopSceneAction()})
			return true
		}
		if inpututil.IsKeyJustPressed(ebiten.KeyN) {
			g.startNewCampaign()
			g.dispatchBatch([]ui.Action{ui.PopSceneAction()})
			return true
		}
		if inpututil.IsKeyJustPressed(ebiten.KeyL) {
			g.openStorageBrowser(storageBrowserLoad)
			return true
		}
		// The title overlay's rows are panel buttons now; they arrive as intents.
		return true
	case ui.SceneMenu:
		if inpututil.IsKeyJustPressed(ebiten.KeyEscape) {
			g.dispatchBatch([]ui.Action{ui.PopSceneAction()})
			return true
		}
		modifier := ebiten.IsKeyPressed(ebiten.KeyControl) || ebiten.IsKeyPressed(ebiten.KeyMeta)
		if !modifier && inpututil.IsKeyJustPressed(ebiten.KeyS) {
			g.openStorageBrowser(storageBrowserSave)
			return true
		}
		if !modifier && inpututil.IsKeyJustPressed(ebiten.KeyL) {
			g.openStorageBrowser(storageBrowserLoad)
			return true
		}
		if inpututil.IsKeyJustPressed(ebiten.KeyO) {
			g.dispatchBatch([]ui.Action{ui.PushSceneAction(ui.SceneSettings)})
			return true
		}
		if inpututil.IsKeyJustPressed(ebiten.KeyT) {
			if g.assignmentDraftDirty() {
				g.showNotice("Apply or discard workforce changes before returning to title")
				return true
			}
			g.scenes.Reset()
			g.scenes.Push(ui.SceneTitle)
			return true
		}
		if inpututil.IsKeyJustPressed(fieldNotesHotkey) {
			g.toggleFieldNotes()
		}
		if inpututil.IsKeyJustPressed(ebiten.KeyM) {
			g.toggleMute()
		}
		if inpututil.IsKeyJustPressed(ebiten.KeyMinus) {
			g.adjustVolume(-0.1)
		}
		if inpututil.IsKeyJustPressed(ebiten.KeyEqual) {
			g.adjustVolume(0.1)
		}
		return true
	case ui.SceneStorage:
		if inpututil.IsKeyJustPressed(ebiten.KeyEscape) {
			g.dispatchBatch([]ui.Action{ui.PopSceneAction()})
			return true
		}
		if inpututil.IsKeyJustPressed(ebiten.KeyArrowUp) {
			g.moveStorageSelection(-1)
		}
		if inpututil.IsKeyJustPressed(ebiten.KeyArrowDown) {
			g.moveStorageSelection(1)
		}
		if inpututil.IsKeyJustPressed(ebiten.KeyEnter) {
			g.activateStorageSelection()
		}
		if inpututil.IsKeyJustPressed(ebiten.KeyDelete) || inpututil.IsKeyJustPressed(ebiten.KeyBackspace) {
			g.deleteStorageSelection()
		}
		return true
	case ui.SceneSettings:
		// The settings slider and check boxes are panel widgets now.
		if inpututil.IsKeyJustPressed(ebiten.KeyEscape) || inpututil.IsKeyJustPressed(ebiten.KeyO) {
			g.dispatchBatch([]ui.Action{ui.PopSceneAction()})
			return true
		}
		if inpututil.IsKeyJustPressed(fieldNotesHotkey) {
			g.toggleFieldNotes()
		}
		if inpututil.IsKeyJustPressed(ebiten.KeyM) {
			g.toggleMute()
		}
		if inpututil.IsKeyJustPressed(ebiten.KeyMinus) {
			g.adjustVolume(-0.1)
		}
		if inpututil.IsKeyJustPressed(ebiten.KeyEqual) {
			g.adjustVolume(0.1)
		}
		return true
	default:
		return false
	}
}

// notesModeFor derives the drawer height state from the persisted preference
// pair, which remains the single source of truth across sessions.
func notesModeFor(settings ui.UISettings) hud.NotesMode {
	switch {
	case !settings.FieldNotesVisible:
		return hud.NotesHidden
	case settings.FieldNotesExpanded:
		return hud.NotesExpanded
	default:
		return hud.NotesCompact
	}
}

// setNotesMode is the one path that changes the drawer, so the F key and the
// drawer's own tab cannot disagree about what gets persisted.
func (g *Game) setNotesMode(mode hud.NotesMode) {
	if g.settingsLoading {
		g.showNotice("Loading preferences…")
		return
	}
	settings := g.settings
	settings.FieldNotesVisible = mode != hud.NotesHidden
	if mode != hud.NotesHidden {
		settings.FieldNotesExpanded = mode == hud.NotesExpanded
	}
	g.updateUISettings(settings)
}

// toggleFieldNotes hides a visible drawer and restores the height the player
// last chose when showing it again.
func (g *Game) toggleFieldNotes() {
	if g.notesMode != hud.NotesHidden {
		g.setNotesMode(hud.NotesHidden)
		return
	}
	mode := hud.NotesCompact
	if g.settings.FieldNotesExpanded {
		mode = hud.NotesExpanded
	}
	g.setNotesMode(mode)
}

func (g *Game) setFieldNote(note render.FieldNote) { g.fieldNote = note }

func (g *Game) splitSelectedBand() {
	band := g.selected()
	if band == nil {
		return
	}
	for _, candidate := range band.MigrationCandidates {
		if candidate.RequiresPassage {
			continue
		}
		if g.apply(gameapi.SplitBand{BandID: g.selectedBand, Destination: candidate.TileID}) {
			g.clearMigrationPreview()
			g.advanceOpenRow()
		}
		return
	}
	g.showNotice("This band has no eligible adjacent land tile for splitting.")
}

func (g *Game) toggleMute() {
	if g.settingsLoading {
		g.showNotice("Loading preferences…")
		return
	}
	settings := g.settings
	settings.Muted = !settings.Muted
	g.updateUISettings(settings)
	if settings.Muted {
		g.showNotice("Sound muted")
	} else {
		g.showNotice(fmt.Sprintf("Sound unmuted at %.0f%%", settings.MasterVolume*100))
	}
}

func (g *Game) adjustVolume(delta float64) {
	if g.settingsLoading {
		g.showNotice("Loading preferences…")
		return
	}
	settings := g.settings
	settings.MasterVolume += delta
	g.updateUISettings(settings)
	g.showNotice(fmt.Sprintf("Master volume %.0f%%", g.settings.MasterVolume*100))
}

func (g *Game) openStorageBrowser(mode storageBrowserMode) {
	result, accepted := g.dispatchBatch([]ui.Action{ui.ListSlotsAction()})
	if !accepted {
		return
	}
	g.storageMode = mode
	g.storageSelection = 0
	g.storageSlots = nil
	g.storageListID = result.operationID
	g.dispatchBatch([]ui.Action{ui.PushSceneAction(ui.SceneStorage)})
}

func (g *Game) moveStorageSelection(delta int) {
	count := len(storageBrowserSlots)
	g.storageSelection = (g.storageSelection + delta + count) % count
}

func (g *Game) activateStorageSelection() {
	if g.storageListID != 0 || g.storageOperationID != 0 {
		return
	}
	slot := storageBrowserSlots[g.storageSelection]
	if g.storageMode == storageBrowserSave {
		if slot < 1 || slot > 3 {
			g.showNotice("Only Manual 1–3 can be overwritten from the Save list")
			return
		}
		result, accepted := g.dispatchBatch([]ui.Action{ui.SaveAction(slot)})
		if !accepted {
			return
		}
		g.storageOperationID = result.operationID
		g.showNotice(fmt.Sprintf("Saving Manual %d…", slot))
		return
	}
	if _, ok := g.storageMetadata(slot); !ok {
		g.showNotice("That slot is empty")
		return
	}
	if g.assignmentDraftDirty() {
		g.showNotice("Apply or discard workforce changes before loading")
		return
	}
	result, accepted := g.dispatchBatch([]ui.Action{ui.LoadAction(slot)})
	if !accepted {
		return
	}
	g.pendingManualLoadID = result.operationID
	g.storageOperationID = result.operationID
	g.showNotice("Loading " + storageSlotLabel(slot) + "…")
}

func (g *Game) deleteStorageSelection() {
	if g.storageListID != 0 || g.storageOperationID != 0 {
		return
	}
	slot := storageBrowserSlots[g.storageSelection]
	if _, ok := g.storageMetadata(slot); !ok {
		g.showNotice("That slot is already empty")
		return
	}
	result, accepted := g.dispatchBatch([]ui.Action{ui.DeleteAction(slot)})
	if !accepted {
		return
	}
	g.storageOperationID = result.operationID
	g.showNotice("Deleting " + storageSlotLabel(slot) + "…")
}

func (g *Game) storageMetadata(slot int) (gameapi.SlotMetadata, bool) {
	for _, metadata := range g.storageSlots {
		if metadata.SlotID == slot {
			return metadata, true
		}
	}
	return gameapi.SlotMetadata{}, false
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
	result, accepted := g.dispatchBatch([]ui.Action{ui.ListSlotsAction()})
	if !accepted {
		return
	}
	g.startupRestorePending = true
	g.startupRestoreListID = result.operationID
}

func (g *Game) pollStorage() {
	for _, result := range g.port.PollStorage() {
		_, saveSoundPending := g.pendingSaveSoundIDs[result.OperationID]
		delete(g.pendingSaveSoundIDs, result.OperationID)
		if result.Operation == gameapi.StorageSave && result.Slot == 99 {
			delete(g.pendingQuickSaveIDs, result.OperationID)
		}

		isStartupList := g.startupRestorePending && result.Operation == gameapi.StorageList && result.OperationID == g.startupRestoreListID
		isStartupLoad := g.startupRestorePending && result.Operation == gameapi.StorageLoad && result.OperationID == g.startupRestoreLoadID
		isManualLoad := result.Operation == gameapi.StorageLoad && result.OperationID == g.pendingManualLoadID
		isBrowserList := result.Operation == gameapi.StorageList && result.OperationID == g.storageListID
		isBrowserOperation := result.OperationID != 0 && result.OperationID == g.storageOperationID
		if isBrowserList {
			g.storageListID = 0
			if result.Err == nil {
				g.storageSlots = append(g.storageSlots[:0], result.Slots...)
			}
		}
		if isBrowserOperation {
			g.storageOperationID = 0
		}
		if isStartupList {
			g.startupRestoreListID = 0
			if result.Err == nil {
				if slot, ok := newestResumeSlot(result.Slots); ok {
					loadResult, accepted := g.dispatchBatch([]ui.Action{ui.LoadAction(slot)})
					if !accepted {
						g.startupRestorePending = false
					} else {
						g.startupRestoreLoadID = loadResult.operationID
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
			g.setFieldNote(ui.CampaignOverviewFieldNote())
			g.breakthroughFrames = 0
			g.regionalPulseFocused = false
			g.clearMigrationPreview()
			g.selectedBand = 0
			g.ensureSelection()
			g.hasAssignmentDraft = false
			g.syncAssignmentDraft(true)
			g.resetDisclosure()
			g.scenes.Reset()
			if isStartupLoad {
				g.scenes.Push(ui.SceneTitle)
			}
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
			g.queueToast("Storage failed: "+ui.ErrorMessage(result.Err), true)
			continue
		}
		if result.Metadata != nil {
			if result.Operation == gameapi.StorageDelete {
				g.removeStorageMetadata(result.Slot)
			} else {
				g.installStorageMetadata(*result.Metadata)
			}
		}
		switch {
		case isStartupLoad:
			if g.startupRestoreSlot == 99 {
				g.queueToast("Quick save restored", false)
			} else {
				g.queueToast(fmt.Sprintf("Autosave restored — Auto %d", g.startupRestoreSlot-100), false)
			}
			g.startupRestoreSlot = 0
		case isManualLoad:
			g.queueToast("Loaded "+storageSlotLabel(result.Slot), false)
		case result.Operation == gameapi.StorageLoad:
			g.queueToast("Game loaded", false)
		case result.Operation == gameapi.StorageSave && result.Slot >= 1 && result.Slot <= 3:
			if saveSoundPending {
				g.sound.Play(gameaudio.SFXSaveComplete)
			}
			g.queueToast(fmt.Sprintf("Saved Manual %d", result.Slot), false)
		case result.Operation == gameapi.StorageSave && result.Slot == 99:
			if saveSoundPending {
				g.sound.Play(gameaudio.SFXSaveComplete)
			}
			g.queueToast("Quick-saved", false)
		case result.Operation == gameapi.StorageSave && result.Slot >= 101 && result.Slot <= 103:
			g.queueToast("Autosaved — "+storageSlotLabel(result.Slot), false)
		case result.Operation == gameapi.StorageDelete:
			g.queueToast("Deleted "+storageSlotLabel(result.Slot), false)
		}
	}
}

func (g *Game) installStorageMetadata(metadata gameapi.SlotMetadata) {
	for index := range g.storageSlots {
		if g.storageSlots[index].SlotID != metadata.SlotID {
			continue
		}
		g.storageSlots[index] = metadata
		return
	}
	g.storageSlots = append(g.storageSlots, metadata)
}

func (g *Game) removeStorageMetadata(slot int) {
	for index := range g.storageSlots {
		if g.storageSlots[index].SlotID != slot {
			continue
		}
		g.storageSlots = append(g.storageSlots[:index], g.storageSlots[index+1:]...)
		return
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
	g.showNoticeFor(message, ui.NoticeFrames(message))
}

func (g *Game) showNoticeFor(message string, frames int) {
	g.toasts = ui.ToastManager{}
	g.notice = message
	g.noticeFrames = frames
}

func (g *Game) queueToast(message string, isError bool) {
	wasEmpty := false
	if _, ok := g.toasts.Current(); !ok {
		wasEmpty = true
	}
	g.toasts.Push(message, isError, ui.NoticeFrames(message))
	if wasEmpty {
		if current, ok := g.toasts.Current(); ok {
			g.notice = current.Message
			g.noticeFrames = current.RemainingFrames
		}
	}
}

func (g *Game) advanceToasts() {
	if _, ok := g.toasts.Current(); !ok {
		if g.noticeFrames > 0 {
			g.noticeFrames--
			if g.noticeFrames == 0 {
				g.notice = ""
			}
		}
		return
	}
	g.toasts.Tick()
	if current, ok := g.toasts.Current(); ok {
		g.notice = current.Message
		g.noticeFrames = current.RemainingFrames
	} else {
		g.notice = ""
		g.noticeFrames = 0
	}
}

func (g *Game) LayoutF(outsideWidth, outsideHeight float64) (float64, float64) {
	scale := 1.0
	if g.deviceScaleFactor != nil {
		scale = g.deviceScaleFactor()
	}
	g.viewport = render.NextViewport(g.viewport, outsideWidth, outsideHeight, scale)
	if outsideWidth > 0 && outsideHeight > 0 {
		g.viewportInitialized = true
	}
	return float64(g.viewport.RenderWidthPx), float64(g.viewport.RenderHeightPx)
}

func (g *Game) Layout(_, _ int) (int, int) { return LogicalWidth, LogicalHeight }

var _ ebiten.LayoutFer = (*Game)(nil)
var _ ebiten.Game = (*Game)(nil)

func (g *Game) logicalCursorPosition() (int, int, bool) {
	x, y := ebiten.CursorPosition()
	if !g.viewportInitialized {
		return x, y, true
	}
	logicalX, logicalY, inside := render.FitPresentation(g.viewport.RenderWidthPx, g.viewport.RenderHeightPx).RenderToLogical(float64(x), float64(y))
	return int(logicalX), int(logicalY), inside
}
