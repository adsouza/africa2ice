package app

import (
	"fmt"
	"hash/maphash"
	"os"
	"sync"

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
	closeResources         func() error
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
	pendingLakeNotes       []render.FieldNote
	openExternalURL        func(string) error
	breakthroughFrames     int
	migrationPreviewBand   gameapi.BandID
	migrationPreviewTile   gameapi.TileID
	hasMigrationPreview    bool
	hoveredTile            gameapi.TileID
	hasHoveredTile         bool
	windowClosingRequested bool
	storage                storageController
	preferences            preferenceController
	workforce              ui.AssignmentDraft
	profileDisplayFrame    *gameapi.Frame
	viewport               render.Viewport
	viewportInitialized    bool
	deviceScaleFactor      func() float64
	scenes                 ui.SceneStack
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
	cameraFocused          bool
}

const (
	fieldNotesHotkey = ebiten.KeyF
	splitBandHotkey  = ebiten.KeyN
)

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
	preferences, preferencesErr := newPreferenceController(settingsStore)
	settings := preferences.value
	game := &Game{
		port: port, sound: sound, frame: frame, scene: render.NewMapScene(),
		panel: hud.New(), notesMode: notesModeFor(settings), guide: ui.NewGuideState(false),
		fieldNote:       ui.CampaignOverviewFieldNote(),
		openExternalURL: openExternalURL,
		notice:          "Outlined tiles are reachable — arrows choose, Enter confirms", noticeFrames: 300,
		storage:           newStorageController(),
		preferences:       preferences,
		deviceScaleFactor: func() float64 { return ebiten.Monitor().DeviceScaleFactor() },
		scenes:            ui.NewSceneStack(),
	}
	if !game.preferences.loading {
		game.applyPresentationSettings()
	}
	if preferencesErr != nil {
		game.notice = "Preferences could not be loaded; using defaults"
	}
	if !game.preferences.loading {
		game.syncEasyMode()
	}
	game.ensureSelection()
	game.syncAssignmentDraft(true)
	game.resetDisclosure()
	return game
}

func NewGame(seed uint64, session *logging.Session) (*Game, error) {
	return newHostedGame(seed, session, true)
}

// NewGameWithoutSound composes the interactive host with a permanently silent
// presentation port, constructing no audio context at all. A machine whose
// sound device cannot be opened needs this: a device that fails to open is
// reported and degraded now, but -no-sound skips the attempt entirely.
func NewGameWithoutSound(seed uint64, session *logging.Session) (*Game, error) {
	return newHostedGame(seed, session, false)
}

func newHostedGame(seed uint64, session *logging.Session, enableSound bool) (*Game, error) {
	return composeHostedGame(seed, session, enableSound, newCampaignRepository, newUISettingsStore, shouldResumeSavedGameOnStartup())
}

func composeHostedGame(seed uint64, session *logging.Session, enableSound bool,
	openRepository func() (application.CampaignRepository, error),
	openSettings func() (ui.UISettingsStore, error), resume bool,
) (*Game, error) {
	repository, err := openRepository()
	if err != nil {
		return nil, err
	}
	// Keep ownership here until construction succeeds, including panic unwinding.
	transferred := false
	defer func() {
		if !transferred {
			_ = repository.Close()
		}
	}()
	repository = logging.DecorateCampaignRepository(session, repository)
	service, err := application.NewGameServiceWithRepository(seed, repository)
	if err != nil {
		return nil, err
	}
	// game is assigned below; the reporter only ever fires from a later Play.
	var game *Game
	var sound gameaudio.SoundManager = gameaudio.NoopManager{}
	if enableSound {
		sound = gameaudio.NewLazyManager(audioReporter(session, os.Stderr, func(notice string) {
			game.showNotice(notice)
		}))
	}
	settingsStore, settingsErr := openSettings()
	settingsStore = logging.DecorateUISettingsStore(session, settingsStore)
	game = newGameWithPresentation(logging.DecorateGame(session, service), sound, settingsStore)
	game.logSession = session
	game.scenes.Push(ui.SceneTitle)
	if settingsErr != nil {
		sound.SetMaster(ui.DefaultUISettings().MasterVolume, ui.DefaultUISettings().Muted)
		game.showNotice("Preferences are unavailable; using defaults")
	}
	if resume {
		game.beginStartupResume()
	}
	game.closeResources = sync.OnceValue(repository.Close)
	transferred = true
	return game, nil
}

func (g *Game) Update() error {
	g.scene.Update()
	g.pollAudio()
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
		if len(g.storage.pendingQuickSaveIDs) == 0 {
			return ebiten.Termination
		}
		g.showNotice("Finishing quick-save before exit…")
		return nil
	}
	if g.storage.resumePending() {
		return nil
	}
	// Wait for preferences before allowing the first gameplay action.
	if g.frame == nil || g.preferences.loading {
		return nil
	}
	if g.storage.pendingManualLoadID == 0 {
		// Retry after a startup load failed while preferences were arriving.
		g.syncEasyMode()
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
	if g.storage.pendingManualLoadID != 0 {
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
	g.scene.SetViewport(g.viewport)
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
	if !g.firstDrawDone && !g.storage.resumePending() {
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
	g.pendingLakeNotes = append(g.pendingLakeNotes, ui.NearbyLakeHistoryNotes(previous, frame)...)
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
		case len(g.pendingLakeNotes) > 0:
			g.setFieldNote(g.pendingLakeNotes[0])
			g.pendingLakeNotes = g.pendingLakeNotes[1:]
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

// cameraFocusEligible reports whether the selection is one the camera leans
// in for: a player band that can still act on the map. It arms Focus on a
// new selection (resetDisclosure) and keeps the map-corner toggle honest; it
// is deliberately not consulted once Focus is armed, since a mode that kept
// reading it would zoom back out the moment the move was spent.
func (g *Game) cameraFocusEligible() bool {
	band := g.selected()
	return band != nil && band.Species == gameapi.HomoSapiens && !ui.MoveDone(*band)
}

// desiredCameraMode reads the stored zoom (spec §6). Focus is sticky: only Z
// and the map-corner button leave it, so committing a move, changing the open
// row, or selecting another band never zooms the player out mid-plan. The one
// clamp is the selection — stepCamera refreshes CenterTile only while a band
// is selected, so Focus with nothing selected would zoom a stale tile. The
// clamp reads the flag without clearing it, so reselecting restores the zoom.
func (g *Game) desiredCameraMode() render.CameraMode {
	if g.cameraFocused && g.selected() != nil {
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

func (g *Game) toggleCameraFocus() { g.cameraFocused = !g.cameraFocused }

// toggleDetails flips the band details disclosure (spec §8). It is `D`'s
// global meaning; while the Workforce row is open, D is row-owned instead
// (handleRowKey's ui.RowWorkforce case discards the draft), matching the
// row-owned model arrows, Enter, and -/+ already use there.
func (g *Game) toggleDetails() { g.detailsOpen = !g.detailsOpen }

// focusInterbreedPartner moves the partner comparison onto another candidate
// without spending anything. The picker chips used to emit IntentInterbreed,
// so the only way to see a second candidate's genetics was to breed with it —
// which defeated the block that exists to make that choice informed.
//
// Focus is validated against the selected band's own candidate list, so it
// always names a band the player could actually breed with; requestInterbreed
// passes it straight to the domain as the target. The partner comparison
// block is the feedback, so no Field Note is disturbed.
func (g *Game) focusInterbreedPartner(id gameapi.BandID) {
	band := g.selected()
	if band == nil {
		return
	}
	for _, candidate := range band.InterbreedCandidateIDs {
		if candidate != id {
			continue
		}
		g.interbreedFocus = id
		return
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
	if g.workforce.Dirty() {
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
	// Co-located bands share one disc on the map, so a click there cannot say
	// which of them the player meant; it resolves to the band they can
	// actually command. Archaic bands stay selectable only where no sapiens
	// band contests the tile — otherwise every tile offering interbreeding
	// left the player one click from a read-only selection.
	sapiensPresent := false
	for _, band := range g.frame.Bands {
		if band.TileID == tileID && band.Species == gameapi.HomoSapiens {
			sapiensPresent = true
			break
		}
	}
	bandIDs := make([]gameapi.BandID, 0, 2)
	selectedIndex := -1
	for _, band := range g.frame.Bands {
		if band.TileID != tileID {
			continue
		}
		if band.Species == gameapi.ArchaicHominin && (sapiensPresent || !g.frame.Tiles[tileID].Explored) {
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
	if g.workforce.Dirty() {
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
				g.storage.trackSave(result.operationID, action.Slot())
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
			g.showNotice(ui.ErrorMessageForMode(err, g.preferences.value.EasyMode))
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
	g.workforce.Clear()
	g.syncAssignmentDraft(true)
	g.resetDisclosure()
	if !g.preferences.value.GuideDismissed {
		g.guide = ui.NewGuideState(false)
	}
	g.pendingLakeNotes = nil
	g.setFieldNote(ui.CampaignOverviewFieldNote())
	g.setNotesMode(hud.NotesExpanded)
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

func (g *Game) handleSceneInput() bool {
	return g.handleSceneKeyState(ebiten.IsKeyPressed, inpututil.IsKeyJustPressed)
}

func (g *Game) handleSceneKeyState(pressed, justPressed func(ebiten.Key) bool) bool {
	switch g.scenes.Current() {
	case ui.SceneGameplay:
		if justPressed(ebiten.KeyEscape) {
			if !g.escape() {
				g.dispatchBatch([]ui.Action{ui.PushSceneAction(ui.SceneMenu)})
			}
			return true
		}
		return false
	case ui.SceneTitle:
		if justPressed(ebiten.KeyEnter) || justPressed(ebiten.KeyC) {
			g.dispatchBatch([]ui.Action{ui.PopSceneAction()})
			return true
		}
		if justPressed(ebiten.KeyN) {
			g.startNewCampaign()
			g.dispatchBatch([]ui.Action{ui.PopSceneAction()})
			return true
		}
		if justPressed(ebiten.KeyL) {
			g.openStorageBrowser(storageBrowserLoad)
			return true
		}
		// The title overlay's rows are panel buttons now; they arrive as intents.
		return true
	case ui.SceneMenu:
		if justPressed(ebiten.KeyEscape) {
			g.dispatchBatch([]ui.Action{ui.PopSceneAction()})
			return true
		}
		modifier := pressed(ebiten.KeyControl) || pressed(ebiten.KeyMeta)
		if !modifier && justPressed(ebiten.KeyS) {
			g.openStorageBrowser(storageBrowserSave)
			return true
		}
		if !modifier && justPressed(ebiten.KeyL) {
			g.openStorageBrowser(storageBrowserLoad)
			return true
		}
		if justPressed(ebiten.KeyO) {
			g.dispatchBatch([]ui.Action{ui.PushSceneAction(ui.SceneSettings)})
			return true
		}
		if justPressed(ebiten.KeyT) {
			if g.workforce.Dirty() {
				g.showNotice("Apply or discard workforce changes before returning to title")
				return true
			}
			g.scenes.Reset()
			g.scenes.Push(ui.SceneTitle)
			return true
		}
		if justPressed(fieldNotesHotkey) {
			g.toggleFieldNotes()
		}
		if justPressed(ebiten.KeyM) {
			g.toggleMute()
		}
		if justPressed(ebiten.KeyMinus) {
			g.adjustVolume(-0.1)
		}
		if justPressed(ebiten.KeyEqual) {
			g.adjustVolume(0.1)
		}
		return true
	case ui.SceneStorage:
		if justPressed(ebiten.KeyEscape) {
			g.dispatchBatch([]ui.Action{ui.PopSceneAction()})
			return true
		}
		if justPressed(ebiten.KeyArrowUp) {
			g.moveStorageSelection(-1)
		}
		if justPressed(ebiten.KeyArrowDown) {
			g.moveStorageSelection(1)
		}
		if justPressed(ebiten.KeyEnter) {
			g.activateStorageSelection()
		}
		if justPressed(ebiten.KeyDelete) || justPressed(ebiten.KeyBackspace) {
			g.deleteStorageSelection()
		}
		return true
	case ui.SceneSettings:
		// The settings slider and check boxes are panel widgets now.
		if justPressed(ebiten.KeyEscape) || justPressed(ebiten.KeyO) {
			g.dispatchBatch([]ui.Action{ui.PopSceneAction()})
			return true
		}
		if justPressed(fieldNotesHotkey) {
			g.toggleFieldNotes()
		}
		if justPressed(ebiten.KeyM) {
			g.toggleMute()
		}
		if justPressed(ebiten.KeyMinus) {
			g.adjustVolume(-0.1)
		}
		if justPressed(ebiten.KeyEqual) {
			g.adjustVolume(0.1)
		}
		return true
	default:
		return false
	}
}

func (g *Game) setFieldNote(note render.FieldNote) { g.fieldNote = note }

func (g *Game) splitSelectedBand() {
	band := g.selected()
	if band == nil {
		return
	}
	if !band.HasSplitDestination {
		g.showNotice("This band has no eligible adjacent land tile for splitting.")
		return
	}
	if g.apply(gameapi.SplitBand{BandID: band.ID, Destination: band.SplitDestination}) {
		g.clearMigrationPreview()
		g.advanceOpenRow()
	}
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
