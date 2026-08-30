package app

import (
	"fmt"

	"github.com/adsouza/africa2ice/internal/adapters/logging"
	"github.com/adsouza/africa2ice/internal/application"
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
	frame                  *gameapi.Frame
	scene                  *render.MapScene
	onFirstDraw            func()
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
}

func NewWalkingSkeleton() *Game {
	port := newSkeletonPort()
	return New(port)
}

func New(port gameapi.Game) *Game {
	frame, _ := port.Snapshot()
	game := &Game{
		port: port, frame: frame, scene: render.NewMapScene(), fieldNotesVisible: true,
		fieldNote: ui.CampaignOverviewFieldNote(),
		notice:    "Outlined tiles are reachable — arrows choose, Enter confirms", noticeFrames: 300,
		pendingQuickSaveIDs: make(map[gameapi.StorageOpID]struct{}),
	}
	game.ensureSelection()
	return game
}

func NewGame(seed uint64, session *logging.Session) (*Game, error) {
	repository, err := newCampaignRepository()
	if err != nil {
		return nil, err
	}
	service, err := application.NewGameServiceWithRepository(seed, repository)
	if err != nil {
		return nil, err
	}
	game := New(logging.DecorateGame(session, service))
	if shouldResumeSavedGameOnStartup() {
		game.beginStartupResume()
	}
	return game, nil
}

func (g *Game) Update() error {
	g.scene.Update()
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
		g.clearMigrationPreview()
		if ebiten.IsKeyPressed(ebiten.KeyShift) {
			g.selectPreviousSapiens()
		} else {
			g.selectNextSapiens()
		}
	}
	if inpututil.IsKeyJustPressed(ebiten.KeyF) {
		g.fieldNotesVisible = !g.fieldNotesVisible
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
		if band := g.selected(); band != nil && len(band.InterbreedCandidateIDs) != 0 {
			if g.apply(gameapi.Interbreed{BandID: band.ID, TargetBandID: band.InterbreedCandidateIDs[0]}) {
				g.clearMigrationPreview()
			}
		}
	}
	for index, key := range [...]ebiten.Key{ebiten.Key1, ebiten.Key2, ebiten.Key3, ebiten.Key4, ebiten.Key5, ebiten.Key6, ebiten.Key7, ebiten.Key8, ebiten.Key9} {
		if inpututil.IsKeyJustPressed(key) {
			g.apply(gameapi.ResearchTech{BandID: g.selectedBand, Tech: gameapi.Tech(index)})
		}
	}
	if inpututil.IsKeyJustPressed(ebiten.KeySpace) && g.frame.CampaignResult == gameapi.Ongoing {
		if g.hasMigrationPreview {
			g.showNotice("Press Enter to queue the migration, or Esc to clear it before ending the turn.")
			return nil
		}
		frame, err := g.port.EndTurn()
		if err == nil {
			g.acceptCompletedTurn(frame)
		} else {
			g.showNotice(err.Error())
		}
	}
	return nil
}

func (g *Game) Draw(screen *ebiten.Image) {
	fieldNote := g.fieldNote
	fieldNote.Celebration = g.breakthroughFrames > 0
	g.scene.Draw(screen, g.frame, g.selectedBand, render.MigrationPreview{
		BandID: g.migrationPreviewBand, TileID: g.migrationPreviewTile, Visible: g.hasMigrationPreview,
	}, g.notice, fieldNote, g.fieldNotesVisible, ui.CampaignEndScene(g.frame))
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
	g.frame = frame
	g.ensureSelection()
	if len(discoveries) == 0 {
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
		return
	}
	selectedIndex = (selectedIndex + offset + len(bandIDs)) % len(bandIDs)
	g.selectedBand = bandIDs[selectedIndex]
}

func (g *Game) handleMapClick() {
	tileID, ok := render.MapTileAt(ebiten.CursorPosition())
	if !ok {
		return
	}
	g.clearMigrationPreview()
	for _, band := range g.frame.Bands {
		if band.Species == gameapi.HomoSapiens && band.TileID == tileID {
			g.selectedBand = band.ID
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
		g.ensureSelection()
		return true
	} else {
		g.showNotice(err.Error())
		return false
	}
}

func (g *Game) startNewCampaign() {
	frame, err := g.port.NewCampaign()
	if err != nil {
		g.showNotice("Could not start a new campaign: " + err.Error())
		return
	}
	g.frame = frame
	g.selectedBand = 0
	g.ensureSelection()
	g.fieldNote = ui.CampaignOverviewFieldNote()
	g.breakthroughFrames = 0
	g.clearMigrationPreview()
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
		g.showNotice("Save failed: " + err.Error())
		return
	}
	g.pendingQuickSaveIDs[operationID] = struct{}{}
}

func (g *Game) beginStartupResume() {
	operationID, err := g.port.BeginListSlots()
	if err != nil {
		g.showNotice("Storage failed: " + err.Error())
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
		if isStartupList {
			g.startupRestoreListID = 0
			if result.Err == nil {
				if slot, ok := newestResumeSlot(result.Slots); ok {
					operationID, err := g.port.BeginLoad(slot)
					if err != nil {
						g.startupRestorePending = false
						g.showNotice("Storage failed: " + err.Error())
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
			g.fieldNote = ui.CampaignOverviewFieldNote()
			g.breakthroughFrames = 0
			g.clearMigrationPreview()
			g.ensureSelection()
		}
		if isStartupLoad {
			g.startupRestorePending = false
			g.startupRestoreLoadID = 0
		}
		if result.Err != nil {
			if isStartupLoad {
				g.startupRestoreSlot = 0
			}
			g.showNotice("Storage failed: " + result.Err.Error())
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
		case result.Operation == gameapi.StorageLoad:
			g.showNotice("Game loaded")
		case result.Operation == gameapi.StorageSave && result.Slot == 99:
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
