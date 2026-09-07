package application

import (
	"errors"
	"time"

	"github.com/adsouza/africa2ice/internal/domain"
	"github.com/adsouza/africa2ice/pkg/gameapi"
)

type GameService struct {
	world                   *domain.World
	worldRevision           uint64
	terrainRevision         uint64
	repository              CampaignRepository
	nextStorageID           gameapi.StorageOpID
	activeStorage           *storageRequest
	queuedStorage           *storageRequest
	storageResults          []gameapi.StorageResult
	autoNeeded              bool
	clock                   monotonicClock
	lastAutosaveRevision    uint64
	nextAutosaveFallback    time.Time
	autosaveCommitSequences [3]uint64
}

type storageRequest struct {
	id        gameapi.StorageOpID
	operation gameapi.StorageOperation
	slot      SlotID
	revision  uint64
	state     *SaveState
}

func NewGameService(seed uint64) (*GameService, error) {
	return NewGameServiceWithRepository(seed, nil)
}

func NewGameServiceWithRepository(seed uint64, repository CampaignRepository) (*GameService, error) {
	return newGameServiceWithRepositoryAndClock(seed, repository, systemMonotonicClock{})
}

func newGameServiceWithRepositoryAndClock(seed uint64, repository CampaignRepository, clock monotonicClock) (*GameService, error) {
	if clock == nil {
		return nil, errors.New("autosave clock is required")
	}
	world, err := domain.NewWorld(seed)
	if err != nil {
		return nil, err
	}
	service := &GameService{world: world, worldRevision: 1, terrainRevision: 1, repository: repository, clock: clock}
	service.resetAutosaveClock()
	return service, nil
}

func (service *GameService) Snapshot() (*gameapi.Frame, error) { return service.projectFrame() }

func (service *GameService) NewCampaign() (*gameapi.Frame, error) {
	if service.loadPending() {
		return nil, &gameapi.GameError{Code: gameapi.ErrStoragePending, Message: "load operation pending"}
	}
	_, seed := domain.SplitMix64(service.world.Seed())
	world, err := domain.NewWorld(seed)
	if err != nil {
		return nil, err
	}
	world.SetEasyMode(service.world.EasyMode())
	service.world = world
	service.worldRevision = 1
	service.terrainRevision++
	service.autoNeeded = false
	service.resetAutosaveClock()
	return service.projectFrame()
}

func (service *GameService) Apply(command gameapi.Command) (*gameapi.Frame, error) {
	if service.loadPending() {
		return nil, &gameapi.GameError{Code: gameapi.ErrStoragePending, Message: "load operation pending"}
	}
	var err error
	terrainChanged := false
	switch value := command.(type) {
	case gameapi.SetEasyMode:
		service.world.SetEasyMode(value.Enabled)
		terrainChanged = true
	case gameapi.SetAssignment:
		var allocation [domain.AssignmentCount]domain.AssignmentBP
		for index, item := range value.AllocationBP {
			allocation[index] = domain.AssignmentBP(item)
		}
		err = service.world.SetAssignment(domain.BandID(value.BandID), allocation, true)
	case gameapi.QueueMigration:
		err = service.world.QueueMigration(domain.BandID(value.BandID), domain.TileID(value.TileID), true)
	case gameapi.SplitBand:
		err = service.world.Split(domain.BandID(value.BandID), domain.TileID(value.Destination), true)
		terrainChanged = err == nil
	case gameapi.ResearchTech:
		technology, ok := unmapTech(value.Tech)
		if !ok {
			return nil, &gameapi.GameError{Code: gameapi.ErrInvalidCommand}
		}
		err = service.world.Research(domain.BandID(value.BandID), technology, true)
	case gameapi.Interbreed:
		err = service.world.Interbreed(domain.BandID(value.BandID), domain.BandID(value.TargetBandID), true)
	default:
		return nil, &gameapi.GameError{Code: gameapi.ErrInvalidCommand}
	}
	if err != nil {
		return nil, mapDomainError(err)
	}
	service.worldRevision++
	if terrainChanged {
		service.terrainRevision++
	}
	return service.projectFrame()
}

func (service *GameService) EndTurn() (*gameapi.Frame, error) {
	if service.loadPending() {
		return nil, &gameapi.GameError{Code: gameapi.ErrStoragePending, Message: "load operation pending"}
	}
	if err := service.world.AdvanceTurn(); err != nil {
		return nil, mapDomainError(err)
	}
	service.worldRevision++
	service.terrainRevision++
	frame, err := service.projectFrame()
	if err != nil {
		return nil, err
	}
	service.autoNeeded = true
	service.dispatchAutosaveIfIdle()
	return frame, nil
}

func storageUnavailable() error {
	return &gameapi.GameError{Code: gameapi.ErrStorageFailure, Message: "storage adapter is not composed"}
}

func (service *GameService) BeginSave(slot int) (gameapi.StorageOpID, error) {
	state, err := service.ExportSaveState()
	if err != nil {
		return 0, &gameapi.GameError{Code: gameapi.ErrStorageFailure, Message: err.Error()}
	}
	return service.beginStorage(gameapi.StorageSave, SlotID(slot), &state)
}
func (service *GameService) BeginLoad(slot int) (gameapi.StorageOpID, error) {
	return service.beginStorage(gameapi.StorageLoad, SlotID(slot), nil)
}
func (service *GameService) BeginDelete(slot int) (gameapi.StorageOpID, error) {
	return service.beginStorage(gameapi.StorageDelete, SlotID(slot), nil)
}
func (service *GameService) BeginListSlots() (gameapi.StorageOpID, error) {
	return service.beginStorage(gameapi.StorageList, 0, nil)
}

func (service *GameService) beginStorage(operation gameapi.StorageOperation, slot SlotID, state *SaveState) (gameapi.StorageOpID, error) {
	if service.repository == nil {
		return 0, storageUnavailable()
	}
	if operation != gameapi.StorageList && !ValidSlot(slot) {
		return 0, &gameapi.GameError{Code: gameapi.ErrStorageFailure, Message: "invalid storage slot"}
	}
	if (operation == gameapi.StorageSave || operation == gameapi.StorageDelete) && !service.repository.Writable() {
		return 0, &gameapi.GameError{Code: gameapi.ErrStorageReadOnly}
	}
	if service.activeStorage != nil && service.queuedStorage != nil {
		return 0, &gameapi.GameError{Code: gameapi.ErrStoragePending}
	}
	service.nextStorageID++
	request := &storageRequest{id: service.nextStorageID, operation: operation, slot: slot, revision: service.worldRevision, state: state}
	if service.activeStorage != nil {
		service.queuedStorage = request
		return request.id, nil
	}
	if err := service.dispatchStorage(request); err != nil {
		return 0, err
	}
	service.activeStorage = request
	return request.id, nil
}

func (service *GameService) dispatchStorage(request *storageRequest) error {
	op := RepositoryOpID(request.id)
	var err error
	switch request.operation {
	case gameapi.StorageSave:
		err = service.repository.BeginWrite(op, request.slot, *request.state)
	case gameapi.StorageLoad:
		err = service.repository.BeginRead(op, request.slot)
	case gameapi.StorageDelete:
		err = service.repository.BeginDelete(op, request.slot)
	case gameapi.StorageList:
		err = service.repository.BeginList(op)
	default:
		err = storageUnavailable()
	}
	if err != nil {
		return &gameapi.GameError{Code: gameapi.ErrStorageFailure, Message: err.Error()}
	}
	return nil
}

func (service *GameService) loadPending() bool {
	return service.activeStorage != nil && service.activeStorage.operation == gameapi.StorageLoad ||
		service.queuedStorage != nil && service.queuedStorage.operation == gameapi.StorageLoad
}

func (service *GameService) PollStorage() []gameapi.StorageResult {
	if service.repository != nil {
		for _, completion := range service.repository.Poll() {
			service.acceptStorageCompletion(completion)
		}
	}
	service.pollAutosaveClock()
	service.dispatchAutosaveIfIdle()
	results := service.storageResults
	service.storageResults = nil
	return results
}

func (service *GameService) acceptStorageCompletion(completion RepositoryCompletion) {
	request := service.activeStorage
	if request == nil || RepositoryOpID(request.id) != completion.OperationID {
		return
	}
	result := gameapi.StorageResult{OperationID: request.id, Operation: request.operation, Slot: int(request.slot), WorldRevision: request.revision, StorageWritable: completion.Writable}
	if completion.Metadata != nil {
		metadata := mapStorageMetadata(*completion.Metadata)
		result.Metadata = &metadata
	}
	for _, item := range completion.Slots {
		service.observeAutosaveMetadata(item)
		result.Slots = append(result.Slots, mapStorageMetadata(item))
	}
	if completion.Metadata != nil {
		service.observeAutosaveMetadata(*completion.Metadata)
	}
	if completion.Err != nil {
		result.Err = &gameapi.GameError{Code: gameapi.ErrStorageFailure, Message: completion.Err.Error()}
	} else if request.operation == gameapi.StorageLoad {
		if completion.State == nil {
			result.Err = &gameapi.GameError{Code: gameapi.ErrInvalidSave, Message: "repository returned no save state"}
		} else if world, err := completion.State.RestoreWorld(); err != nil {
			result.Err = &gameapi.GameError{Code: gameapi.ErrInvalidSave, Message: err.Error()}
		} else {
			service.world = world
			service.worldRevision = completion.State.WorldRevision
			service.terrainRevision++
			service.autoNeeded = false
			service.resetAutosaveClock()
			result.WorldRevision = service.worldRevision
			result.ReplacementFrame, result.Err = service.projectFrame()
		}
	}
	if result.Err == nil && request.operation == gameapi.StorageSave {
		if _, ok := autosaveSlotIndex(request.slot); ok {
			service.acceptSuccessfulAutosave(request, completion.Metadata)
		}
	}
	service.storageResults = append(service.storageResults, result)
	service.activeStorage = nil
	if service.queuedStorage != nil {
		next := service.queuedStorage
		service.queuedStorage = nil
		if err := service.dispatchStorage(next); err != nil {
			service.storageResults = append(service.storageResults, gameapi.StorageResult{OperationID: next.id, Operation: next.operation, Slot: int(next.slot), WorldRevision: next.revision, StorageWritable: service.repository.Writable(), Err: err})
		} else {
			service.activeStorage = next
		}
	}
	if service.activeStorage == nil {
		service.dispatchAutosaveIfIdle()
	}
}

func (service *GameService) dispatchAutosaveIfIdle() {
	if !service.autoNeeded || service.repository == nil || !service.repository.Writable() || service.activeStorage != nil || service.queuedStorage != nil {
		return
	}
	state, err := service.ExportSaveState()
	if err != nil {
		return
	}
	slot := service.chooseAutosaveSlot()
	service.nextStorageID++
	request := &storageRequest{id: service.nextStorageID, operation: gameapi.StorageSave, slot: slot, revision: service.worldRevision, state: &state}
	service.autoNeeded = false
	if err := service.dispatchStorage(request); err != nil {
		service.storageResults = append(service.storageResults, gameapi.StorageResult{
			OperationID: request.id, Operation: request.operation, Slot: int(request.slot),
			WorldRevision: request.revision, StorageWritable: service.repository.Writable(), Err: err,
		})
		return
	}
	service.activeStorage = request
}

func mapStorageMetadata(metadata SaveMetadata) gameapi.SlotMetadata {
	date, _ := domain.CampaignDate(metadata.Turn)
	kind := gameapi.ManualSlot
	switch metadata.SlotID {
	case QuickSave:
		kind = gameapi.QuickSlot
	case Auto1, Auto2, Auto3:
		kind = gameapi.AutoSlot
	}
	return gameapi.SlotMetadata{SlotID: int(metadata.SlotID), SlotKind: kind, CommitSequence: metadata.CommitSequence, WorldRevision: metadata.WorldRevision, Turn: metadata.Turn, YearBP: date.YearBP, Era: gameapi.CampaignEra(date.Era), SapiensPopulation: metadata.SapiensPopulation, SavedAt: metadata.SavedAt, CampaignClockAlgorithm: metadata.CampaignClockAlgorithm, StateHash: metadata.Generation}
}

var _ gameapi.Game = (*GameService)(nil)
