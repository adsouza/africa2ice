package application

import (
	"errors"
	"testing"

	"github.com/adsouza/africa2ice/pkg/gameapi"
)

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

func TestFrameIsolationAcrossProjections(t *testing.T) {
	service, _ := NewGameService(2)
	old, _ := service.Snapshot()
	old.Bands[0].Population = 999
	old.Tiles[0].ElevationKm = 999
	old.Bands[0].MigrationCandidates = append(old.Bands[0].MigrationCandidates, gameapi.MigrationCandidate{TileID: 6000})
	fresh, _ := service.Snapshot()
	if fresh.Bands[0].Population == 999 || fresh.Tiles[0].ElevationKm == 999 {
		t.Fatal("frame aliases a prior projection")
	}
	for _, candidate := range fresh.Bands[0].MigrationCandidates {
		if candidate.TileID == 6000 {
			t.Fatal("nested candidate slice aliased")
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
