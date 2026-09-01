package application

import (
	"errors"
	"math"
	"testing"
	"time"

	"github.com/adsouza/africa2ice/internal/domain"
	"github.com/adsouza/africa2ice/pkg/gameapi"
)

type fakeMonotonicClock struct{ now time.Time }

func (clock *fakeMonotonicClock) Now() time.Time { return clock.now }

func (clock *fakeMonotonicClock) Advance(duration time.Duration) { clock.now = clock.now.Add(duration) }

type repositoryStub struct {
	writable    bool
	written     *SaveState
	writtenSlot SlotID
	completions []RepositoryCompletion
}

func (repository *repositoryStub) BeginWrite(op RepositoryOpID, slot SlotID, state SaveState) error {
	repository.written = &state
	repository.writtenSlot = slot
	repository.completions = append(repository.completions, RepositoryCompletion{OperationID: op, Operation: RepositoryWrite, SlotID: slot, Writable: repository.writable})
	return nil
}
func (repository *repositoryStub) BeginRead(op RepositoryOpID, slot SlotID) error {
	repository.completions = append(repository.completions, RepositoryCompletion{OperationID: op, Operation: RepositoryRead, SlotID: slot, State: repository.written, Writable: repository.writable})
	return nil
}
func (repository *repositoryStub) BeginDelete(op RepositoryOpID, slot SlotID) error {
	repository.completions = append(repository.completions, RepositoryCompletion{OperationID: op, Operation: RepositoryDelete, SlotID: slot, Writable: repository.writable})
	return nil
}
func (repository *repositoryStub) BeginList(op RepositoryOpID) error {
	repository.completions = append(repository.completions, RepositoryCompletion{OperationID: op, Operation: RepositoryList, Writable: repository.writable})
	return nil
}
func (repository *repositoryStub) Poll() []RepositoryCompletion {
	completions := repository.completions
	repository.completions = nil
	return completions
}
func (repository *repositoryStub) Writable() bool { return repository.writable }
func (repository *repositoryStub) Close() error   { return nil }

func TestGameServiceRevisionsAndPlayerAuthority(t *testing.T) {
	service, err := NewGameService(1)
	if err != nil {
		t.Fatal(err)
	}
	initial, err := service.Snapshot()
	if err != nil {
		t.Fatal(err)
	}
	allocation := [gameapi.AssignmentCount]uint16{2000, 2000, 2000, 2000, 2000}
	assigned, err := service.Apply(gameapi.SetAssignment{BandID: 1, AllocationBP: allocation})
	if err != nil {
		t.Fatal(err)
	}
	if assigned.WorldRevision != initial.WorldRevision+1 || assigned.TerrainRevision != initial.TerrainRevision {
		t.Fatalf("assignment revisions = %d/%d", assigned.WorldRevision, assigned.TerrainRevision)
	}
	_, err = service.Apply(gameapi.SetAssignment{BandID: 5, AllocationBP: allocation})
	var gameErr *gameapi.GameError
	if !errors.As(err, &gameErr) || gameErr.Code != gameapi.ErrComputerControlledBand {
		t.Fatalf("authority error = %#v", err)
	}
	if after, _ := service.Snapshot(); after.WorldRevision != assigned.WorldRevision {
		t.Fatal("rejected command advanced revision")
	}
}

func TestSuccessfulSplitAdvancesTerrainRevision(t *testing.T) {
	service, err := NewGameService(25)
	if err != nil {
		t.Fatal(err)
	}
	band := service.world.Bands()[0]
	// Raise settlement stress enough to make the split gate deterministic.
	state, err := service.world.ExportState()
	if err != nil {
		t.Fatal(err)
	}
	state.Bands[0].Population = 10_000
	service.world, err = domain.RestoreWorld(state)
	if err != nil {
		t.Fatal(err)
	}
	var destination domain.TileID
	found := false
	for _, edge := range service.world.Grid().OrdinaryEdges(band.TileID) {
		if service.world.IsExplored(edge.To) && service.world.Habitat()[edge.To].BaselineK > 0 {
			destination, found = edge.To, true
			break
		}
	}
	if !found {
		t.Fatal("starting band has no explored habitable split destination")
	}
	before, _ := service.Snapshot()
	after, err := service.Apply(gameapi.SplitBand{BandID: gameapi.BandID(band.ID), Destination: gameapi.TileID(destination)})
	if err != nil {
		t.Fatal(err)
	}
	if after.WorldRevision != before.WorldRevision+1 || after.TerrainRevision != before.TerrainRevision+1 {
		t.Fatalf("split revisions = %d/%d, want %d/%d", after.WorldRevision, after.TerrainRevision, before.WorldRevision+1, before.TerrainRevision+1)
	}
}

func TestFrameIsolationAcrossProjections(t *testing.T) {
	service, _ := NewGameService(2)
	old, _ := service.Snapshot()
	if len(old.Escarpments) == 0 {
		t.Fatal("initially explored East Africa projects no escarpments")
	}
	old.Bands[0].Population = 999
	old.Tiles[0].ElevationKm = 999
	old.Tiles[0].LocalTemperatureC = 999
	if len(old.Escarpments) > 0 {
		old.Escarpments[0].Name = "mutated"
	}
	if len(old.Bands[0].MigrationCandidates) > 0 {
		old.Bands[0].MigrationCandidates[0].SeasonalMortalityRate = 999
	}
	old.Bands[0].MigrationCandidates = append(old.Bands[0].MigrationCandidates, gameapi.MigrationCandidate{TileID: 6000})
	fresh, _ := service.Snapshot()
	if fresh.Bands[0].Population == 999 || fresh.Tiles[0].ElevationKm == 999 || fresh.Tiles[0].LocalTemperatureC == 999 {
		t.Fatal("frame aliases a prior projection")
	}
	for _, edge := range fresh.Escarpments {
		if edge.Name == "mutated" {
			t.Fatal("escarpment projection aliases a prior frame")
		}
	}
	for _, candidate := range fresh.Bands[0].MigrationCandidates {
		if candidate.TileID == 6000 || candidate.SeasonalMortalityRate == 999 {
			t.Fatal("nested candidate slice aliased")
		}
	}
}

func TestFrameProjectsTileLiveabilityInputsAndBandSpecificRisks(t *testing.T) {
	service, err := NewGameService(29)
	if err != nil {
		t.Fatal(err)
	}
	frame, err := service.Snapshot()
	if err != nil {
		t.Fatal(err)
	}
	for _, tile := range frame.Tiles {
		if !tile.Land {
			continue
		}
		if math.IsNaN(tile.LocalTemperatureC) || math.IsInf(tile.LocalTemperatureC, 0) {
			t.Fatalf("tile %d temperature = %v", tile.ID, tile.LocalTemperatureC)
		}
		if tile.MovementCost < 1 || tile.MovementCost > 2.75 {
			t.Fatalf("tile %d movement cost = %v", tile.ID, tile.MovementCost)
		}
	}
	for _, band := range frame.Bands {
		if band.SeasonalMortalityRate < 0 || band.ChronicMortalityRate < 0 ||
			math.IsNaN(band.SeasonalMortalityRate) || math.IsNaN(band.ChronicMortalityRate) {
			t.Fatalf("band %d mortality preview = %v/%v", band.ID, band.SeasonalMortalityRate, band.ChronicMortalityRate)
		}
		for _, candidate := range band.MigrationCandidates {
			if candidate.SeasonalMortalityRate < 0 || candidate.ChronicMortalityRate < 0 ||
				math.IsNaN(candidate.SeasonalMortalityRate) || math.IsNaN(candidate.ChronicMortalityRate) {
				t.Fatalf("band %d candidate %d mortality preview = %v/%v", band.ID, candidate.TileID, candidate.SeasonalMortalityRate, candidate.ChronicMortalityRate)
			}
		}
	}
}

func TestEndTurnBuildsOneReplacementFrame(t *testing.T) {
	service, _ := NewGameService(3)
	before, _ := service.Snapshot()
	after, err := service.EndTurn()
	if err != nil {
		t.Fatal(err)
	}
	if after.Turn != 1 || after.WorldRevision != before.WorldRevision+1 || after.TerrainRevision != before.TerrainRevision+1 {
		t.Fatalf("turn frame = %#v", after)
	}
}

func TestNewCampaignReplacesWorldWithFreshSeedAndMonotonicTerrainRevision(t *testing.T) {
	service, _ := NewGameService(3)
	before, err := service.EndTurn()
	if err != nil {
		t.Fatal(err)
	}
	oldSeed := service.world.Seed()
	fresh, err := service.NewCampaign()
	if err != nil {
		t.Fatal(err)
	}
	if service.world.Seed() == oldSeed || fresh.Turn != 0 || fresh.CampaignResult != gameapi.Ongoing {
		t.Fatalf("fresh campaign = seed %d, frame %#v", service.world.Seed(), fresh)
	}
	if fresh.WorldRevision != 1 || fresh.TerrainRevision != before.TerrainRevision+1 {
		t.Fatalf("fresh revisions = %d/%d, want 1/%d", fresh.WorldRevision, fresh.TerrainRevision, before.TerrainRevision+1)
	}
}

func TestFrameProjectsQueuedMigrationUntilTurnResolution(t *testing.T) {
	service, err := NewGameService(31)
	if err != nil {
		t.Fatal(err)
	}
	initial, _ := service.Snapshot()
	band := initial.Bands[0]
	for technology := gameapi.Tech(0); technology < gameapi.TechCount; technology++ {
		option := band.ResearchOptions[technology]
		if option.Cost != domain.ResearchCost[domain.Technology(technology)] || option.PrerequisiteMask != domain.TechnologyPrerequisiteMask(domain.Technology(technology)) {
			t.Fatalf("technology %s projection = cost %.0f mask %#x", technology, option.Cost, option.PrerequisiteMask)
		}
	}
	if len(band.MigrationCandidates) == 0 {
		t.Fatal("initial band has no migration candidate")
	}
	destination := band.MigrationCandidates[0].TileID
	queued, err := service.Apply(gameapi.QueueMigration{BandID: band.ID, TileID: destination})
	if err != nil {
		t.Fatal(err)
	}
	if !queued.Bands[0].HasQueuedMigration || queued.Bands[0].QueuedMigration != destination || queued.Bands[0].TileID != band.TileID {
		t.Fatalf("queued frame did not preserve origin and destination: %+v", queued.Bands[0])
	}
	resolved, err := service.EndTurn()
	if err != nil {
		t.Fatal(err)
	}
	if resolved.Bands[0].HasQueuedMigration {
		t.Fatal("resolved frame still exposes a queued migration")
	}
}

func TestFrameProjectsResearchKeyAvailability(t *testing.T) {
	service, _ := NewGameService(23)
	initial, _ := service.Snapshot()
	band := initial.Bands[0]
	for _, technology := range []gameapi.Tech{gameapi.Firecraft, gameapi.HaftedTools, gameapi.PlantKnowledge} {
		if !band.ResearchOptions[technology].Available {
			t.Fatalf("starting technology %s is not shown as available", technology)
		}
	}
	for _, technology := range []gameapi.Tech{gameapi.TailoredClothing, gameapi.CordageAndNets, gameapi.Campcraft, gameapi.MedicinalKnowledge, gameapi.Trapping, gameapi.CoastalNavigation} {
		if band.ResearchOptions[technology].Available {
			t.Fatalf("locked technology %s is shown as available", technology)
		}
	}
	selected, err := service.Apply(gameapi.ResearchTech{BandID: band.ID, Tech: gameapi.Firecraft})
	if err != nil {
		t.Fatal(err)
	}
	if !selected.Bands[0].ResearchOptions[gameapi.Firecraft].Current {
		t.Fatal("selected technology is not projected as current")
	}
}

func TestEndTurnRequestsRotatingAutosave(t *testing.T) {
	repository := &repositoryStub{writable: true}
	service, _ := NewGameServiceWithRepository(31, repository)
	for turn, expected := range []SlotID{Auto1, Auto2, Auto3, Auto1} {
		if _, err := service.EndTurn(); err != nil {
			t.Fatal(err)
		}
		if repository.writtenSlot != expected || repository.written == nil || repository.written.Turn != turn+1 {
			t.Fatalf("turn %d autosave = slot %d state %#v; want %d", turn+1, repository.writtenSlot, repository.written, expected)
		}
		results := service.PollStorage()
		if len(results) != 1 || results[0].Slot != int(expected) || results[0].Err != nil {
			t.Fatalf("turn %d autosave result = %#v", turn+1, results)
		}
	}
}

func TestFiveMinuteAutosaveFallbackRequiresANewerRevision(t *testing.T) {
	clock := &fakeMonotonicClock{now: time.Unix(1_000, 0)}
	repository := &repositoryStub{writable: true}
	service, err := newGameServiceWithRepositoryAndClock(31, repository, clock)
	if err != nil {
		t.Fatal(err)
	}
	clock.Advance(autosaveFallbackInterval)
	service.PollStorage()
	if repository.written != nil {
		t.Fatal("time fallback saved an unchanged initial revision")
	}

	initial, _ := service.Snapshot()
	allocation := initial.Bands[0].AllocationBP
	allocation[0]++
	allocation[1]--
	if _, err := service.Apply(gameapi.SetAssignment{BandID: initial.Bands[0].ID, AllocationBP: allocation}); err != nil {
		t.Fatal(err)
	}
	clock.Advance(autosaveFallbackInterval - time.Second)
	service.PollStorage()
	if repository.written != nil {
		t.Fatal("fallback saved before a fresh five-minute interval")
	}
	clock.Advance(time.Second)
	service.PollStorage()
	if repository.written == nil || repository.writtenSlot != Auto1 || repository.written.WorldRevision != initial.WorldRevision+1 {
		t.Fatalf("fallback write = slot %d state %#v", repository.writtenSlot, repository.written)
	}
}

func TestGameServiceRejectsNilAutosaveClock(t *testing.T) {
	service, err := newGameServiceWithRepositoryAndClock(31, nil, nil)
	if err == nil || service != nil {
		t.Fatalf("nil clock constructor = service %#v, error %v", service, err)
	}
	if err.Error() != "autosave clock is required" {
		t.Fatalf("nil clock error = %q", err)
	}
}

func TestAutosaveChoosesEmptyThenOldestCommittedSlot(t *testing.T) {
	clock := &fakeMonotonicClock{now: time.Unix(1_000, 0)}
	repository := &repositoryStub{writable: true}
	service, err := newGameServiceWithRepositoryAndClock(31, repository, clock)
	if err != nil {
		t.Fatal(err)
	}
	service.observeAutosaveMetadata(SaveMetadata{SlotID: Auto1, CommitSequence: 8})
	service.observeAutosaveMetadata(SaveMetadata{SlotID: Auto2, CommitSequence: 3})
	if got := service.chooseAutosaveSlot(); got != Auto3 {
		t.Fatalf("first empty autosave slot = %d, want %d", got, Auto3)
	}
	service.observeAutosaveMetadata(SaveMetadata{SlotID: Auto3, CommitSequence: 5})
	if got := service.chooseAutosaveSlot(); got != Auto2 {
		t.Fatalf("oldest autosave slot = %d, want %d", got, Auto2)
	}
	service.observeAutosaveMetadata(SaveMetadata{SlotID: Auto1, CommitSequence: 3})
	if got := service.chooseAutosaveSlot(); got != Auto1 {
		t.Fatalf("tie-broken autosave slot = %d, want %d", got, Auto1)
	}
}

func TestStorageQueueFreezesForQueuedLoadAndReplacesAtomically(t *testing.T) {
	repository := &repositoryStub{writable: true}
	service, err := NewGameServiceWithRepository(4, repository)
	if err != nil {
		t.Fatal(err)
	}
	saveID, err := service.BeginSave(int(Manual1))
	if err != nil {
		t.Fatal(err)
	}
	loadID, err := service.BeginLoad(int(Manual1))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := service.BeginListSlots(); err == nil {
		t.Fatal("third storage operation was accepted")
	}
	if _, err := service.EndTurn(); err == nil {
		t.Fatal("queued load did not freeze simulation")
	}
	first := service.PollStorage()
	if len(first) != 1 || first[0].OperationID != saveID {
		t.Fatalf("save completion = %#v", first)
	}
	second := service.PollStorage()
	if len(second) != 1 || second[0].OperationID != loadID || second[0].ReplacementFrame == nil {
		t.Fatalf("load completion = %#v", second)
	}
	if _, err := service.EndTurn(); err != nil {
		t.Fatalf("simulation remained frozen: %v", err)
	}
}

func TestBeginSaveCapturesImmutableWorldRevision(t *testing.T) {
	repository := &repositoryStub{writable: true}
	service, _ := NewGameServiceWithRepository(5, repository)
	initial, _ := service.Snapshot()
	if _, err := service.BeginSave(int(QuickSave)); err != nil {
		t.Fatal(err)
	}
	if _, err := service.EndTurn(); err != nil {
		t.Fatal(err)
	}
	if repository.written == nil || repository.written.WorldRevision != initial.WorldRevision || repository.written.Turn != initial.Turn {
		t.Fatalf("captured save changed: %#v", repository.written)
	}
}

func TestStorageMetadataProjectsCommitSequence(t *testing.T) {
	metadata := mapStorageMetadata(SaveMetadata{SlotID: Auto2, CommitSequence: 42, Turn: 3})
	if metadata.SlotID != int(Auto2) || metadata.SlotKind != gameapi.AutoSlot || metadata.CommitSequence != 42 {
		t.Fatalf("projected metadata = %#v", metadata)
	}
}

// The renderer cannot show a projected value the frame does not carry. This is
// the middle link of that path: the domain computes the arrival crowding decline
// per candidate, and it has to survive projection into gameapi.
//
// Asserting only that the projected value is finite and non-negative would pass
// against a field that is never assigned, since zero satisfies both. So this
// compares every projected candidate against the domain's own value, and
// separately requires at least one of them to be non-zero, which is what makes
// the comparison mean anything.
func TestFrameCarriesTheProjectedCrowdingDecline(t *testing.T) {
	service, err := NewGameService(29)
	if err != nil {
		t.Fatal(err)
	}
	sawNonZero, compared := false, 0
	for turn := 0; turn < 120 && !sawNonZero; turn++ {
		frame, err := service.Snapshot()
		if err != nil {
			t.Fatal(err)
		}
		for _, band := range frame.Bands {
			expected := map[domain.TileID]float64{}
			for _, candidate := range service.world.MigrationCandidates(domain.BandID(band.ID)) {
				expected[candidate.TileID] = candidate.CrowdingDecline
			}
			for _, candidate := range band.MigrationCandidates {
				want, ok := expected[domain.TileID(candidate.TileID)]
				if !ok {
					t.Fatalf("band %d projected candidate %d the domain does not offer", band.ID, candidate.TileID)
				}
				if candidate.CrowdingDecline != want {
					t.Fatalf("band %d candidate %d crowding decline = %v, domain says %v",
						band.ID, candidate.TileID, candidate.CrowdingDecline, want)
				}
				compared++
				if want > 0 {
					sawNonZero = true
				}
			}
		}
		if _, err := service.EndTurn(); err != nil {
			t.Fatal(err)
		}
	}
	if compared == 0 {
		t.Fatal("no candidates compared")
	}
	if !sawNonZero {
		t.Fatal("no candidate ever projected a crowding decline, so the comparison above proves nothing")
	}
}
