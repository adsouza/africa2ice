//go:build !js

package storage

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/adsouza/africa2ice/internal/application"
)

type fileRequest struct {
	op    application.RepositoryOpID
	kind  application.RepositoryOperation
	slot  application.SlotID
	state *application.SaveState
}

type FileRepository struct {
	directory   string
	lease       repositoryLease
	writable    bool
	requests    chan fileRequest
	completions chan application.RepositoryCompletion
	done        chan struct{}
	closeOnce   sync.Once
	// sequence is the last commit sequence this process issued. It is scanned
	// from disk once and then advanced in memory, so a long campaign's
	// per-turn autosaves do not re-read every metadata record on every write.
	// Only the worker goroutine touches it.
	sequence       uint64
	sequenceLoaded bool
}

func NewFileRepository(directory string) (*FileRepository, error) {
	if err := os.MkdirAll(directory, 0o700); err != nil {
		return nil, err
	}
	lease, writable, err := acquireRepositoryLease(filepath.Join(directory, "saves.lock"))
	if err != nil {
		return nil, err
	}
	repository := &FileRepository{directory: directory, lease: lease, writable: writable, requests: make(chan fileRequest, 2), completions: make(chan application.RepositoryCompletion, 8), done: make(chan struct{})}
	go repository.worker()
	return repository, nil
}

func (repository *FileRepository) BeginWrite(op application.RepositoryOpID, slot application.SlotID, state application.SaveState) error {
	if !repository.writable {
		return errors.New("repository is read-only")
	}
	return repository.enqueue(fileRequest{op: op, kind: application.RepositoryWrite, slot: slot, state: &state})
}
func (repository *FileRepository) BeginRead(op application.RepositoryOpID, slot application.SlotID) error {
	return repository.enqueue(fileRequest{op: op, kind: application.RepositoryRead, slot: slot})
}
func (repository *FileRepository) BeginDelete(op application.RepositoryOpID, slot application.SlotID) error {
	if !repository.writable {
		return errors.New("repository is read-only")
	}
	return repository.enqueue(fileRequest{op: op, kind: application.RepositoryDelete, slot: slot})
}
func (repository *FileRepository) BeginList(op application.RepositoryOpID) error {
	return repository.enqueue(fileRequest{op: op, kind: application.RepositoryList})
}

func (repository *FileRepository) enqueue(request fileRequest) error {
	if request.kind != application.RepositoryList && !application.ValidSlot(request.slot) {
		return fmt.Errorf("invalid slot %d", request.slot)
	}
	select {
	case <-repository.done:
		return errors.New("repository closed")
	default:
	}
	select {
	case repository.requests <- request:
		return nil
	default:
		return errors.New("repository queue full")
	}
}

func (repository *FileRepository) Writable() bool { return repository.writable }

func (repository *FileRepository) Poll() []application.RepositoryCompletion {
	var result []application.RepositoryCompletion
	for {
		select {
		case completion := <-repository.completions:
			result = append(result, completion)
		default:
			return result
		}
	}
}

func (repository *FileRepository) Close() error {
	var closeErr error
	repository.closeOnce.Do(func() {
		close(repository.done)
		if repository.lease != nil {
			closeErr = repository.lease.Close()
		}
	})
	return closeErr
}

func (repository *FileRepository) worker() {
	for {
		select {
		case request := <-repository.requests:
			completion := application.RepositoryCompletion{OperationID: request.op, Operation: request.kind, SlotID: request.slot, Writable: repository.writable}
			switch request.kind {
			case application.RepositoryWrite:
				completion.Metadata, completion.Err = repository.write(request.slot, *request.state)
			case application.RepositoryRead:
				completion.State, completion.Metadata, completion.Err = repository.read(request.slot)
			case application.RepositoryDelete:
				completion.Metadata, completion.Err = repository.delete(request.slot)
			case application.RepositoryList:
				completion.Slots, completion.Err = repository.list()
			}
			repository.completions <- completion
		case <-repository.done:
			return
		}
	}
}

func (repository *FileRepository) write(slot application.SlotID, state application.SaveState) (*application.SaveMetadata, error) {
	worldBytes, err := application.EncodeSaveState(state)
	if err != nil {
		return nil, err
	}
	hash := sha256.Sum256(worldBytes)
	generation := hex.EncodeToString(hash[:])
	worldPath := filepath.Join(repository.directory, fmt.Sprintf("slot_%d_world_%s.json", slot, generation))
	if err := writeImmutable(worldPath, worldBytes); err != nil {
		return nil, err
	}
	sequence, err := repository.nextSequence()
	if err != nil {
		return nil, err
	}
	metadata := metadataFor(slot, sequence, generation, state)
	metadataBytes, err := json.Marshal(metadata)
	if err != nil {
		return nil, err
	}
	metaPath := filepath.Join(repository.directory, fmt.Sprintf("slot_%d_meta_%020d.json", slot, sequence))
	if err := writeImmutable(metaPath, metadataBytes); err != nil {
		return nil, err
	}
	repository.pruneSupersededRecords(slot, sequence, generation)
	return &metadata, nil
}

func metadataFor(slot application.SlotID, sequence uint64, generation string, state application.SaveState) application.SaveMetadata {
	metadata := application.SaveMetadata{SlotID: slot, CommitSequence: sequence, Generation: generation, SchemaVersion: state.SchemaVersion, CampaignClockAlgorithm: state.CampaignClockAlgorithm, WorldRevision: state.WorldRevision, Turn: state.Turn, SavedAt: time.Now().UTC()}
	metadata.SapiensPopulation, metadata.ArchaicPopulation = state.PopulationBySpecies()
	return metadata
}

func writeImmutable(path string, data []byte) error {
	if existing, err := os.ReadFile(path); err == nil {
		if !bytes.Equal(existing, data) {
			return errors.New("immutable record collision")
		}
		return nil
	} else if !errors.Is(err, os.ErrNotExist) {
		return err
	}
	directory := filepath.Dir(path)
	temporary, err := os.CreateTemp(directory, ".africa2ice-*.tmp")
	if err != nil {
		return err
	}
	temporaryPath := temporary.Name()
	cleanup := func() { _ = temporary.Close(); _ = os.Remove(temporaryPath) }
	if _, err := temporary.Write(data); err != nil {
		cleanup()
		return err
	}
	if err := temporary.Sync(); err != nil {
		cleanup()
		return err
	}
	if err := temporary.Close(); err != nil {
		_ = os.Remove(temporaryPath)
		return err
	}
	if err := os.Rename(temporaryPath, path); err != nil {
		_ = os.Remove(temporaryPath)
		return err
	}
	return nil
}

func (repository *FileRepository) latestMetadata(slot application.SlotID) (*application.SaveMetadata, error) {
	entries, err := filepath.Glob(filepath.Join(repository.directory, fmt.Sprintf("slot_%d_meta_*.json", slot)))
	if err != nil {
		return nil, err
	}
	var latest *application.SaveMetadata
	for _, path := range entries {
		data, err := os.ReadFile(path)
		if err != nil {
			continue
		}
		var metadata application.SaveMetadata
		decoder := json.NewDecoder(bytes.NewReader(data))
		decoder.DisallowUnknownFields()
		if decoder.Decode(&metadata) != nil || metadata.SlotID != slot {
			continue
		}
		if latest == nil || metadata.CommitSequence > latest.CommitSequence {
			copyValue := metadata
			latest = &copyValue
		}
	}
	if latest == nil {
		return nil, os.ErrNotExist
	}
	return latest, nil
}

func (repository *FileRepository) read(slot application.SlotID) (*application.SaveState, *application.SaveMetadata, error) {
	metadata, err := repository.latestMetadata(slot)
	if err != nil {
		return nil, nil, err
	}
	if metadata.Deleted {
		return nil, metadata, os.ErrNotExist
	}
	path := filepath.Join(repository.directory, fmt.Sprintf("slot_%d_world_%s.json", slot, metadata.Generation))
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, metadata, err
	}
	hash := sha256.Sum256(data)
	if hex.EncodeToString(hash[:]) != metadata.Generation {
		return nil, metadata, errors.New("save checksum mismatch")
	}
	state, err := application.DecodeSaveState(data)
	if err != nil {
		return nil, metadata, err
	}
	return &state, metadata, nil
}

func (repository *FileRepository) delete(slot application.SlotID) (*application.SaveMetadata, error) {
	sequence, err := repository.nextSequence()
	if err != nil {
		return nil, err
	}
	metadata := application.SaveMetadata{SlotID: slot, CommitSequence: sequence, Deleted: true}
	data, err := json.Marshal(metadata)
	if err != nil {
		return nil, err
	}
	path := filepath.Join(repository.directory, fmt.Sprintf("slot_%d_meta_%020d.json", slot, sequence))
	if err := writeImmutable(path, data); err != nil {
		return nil, err
	}
	// The delete marker names no generation, so every world payload for this
	// slot is now unreachable.
	repository.pruneSupersededRecords(slot, sequence, "")
	return &metadata, nil
}

func (repository *FileRepository) list() ([]application.SaveMetadata, error) {
	result := make([]application.SaveMetadata, 0, 7)
	for _, slot := range []application.SlotID{application.Manual1, application.Manual2, application.Manual3, application.QuickSave, application.Auto1, application.Auto2, application.Auto3} {
		metadata, err := repository.latestMetadata(slot)
		if err == nil && !metadata.Deleted {
			result = append(result, *metadata)
		}
	}
	return result, nil
}

func (repository *FileRepository) nextSequence() (uint64, error) {
	if !repository.sequenceLoaded {
		highest, err := repository.highestRecordedSequence()
		if err != nil {
			return 0, err
		}
		repository.sequence, repository.sequenceLoaded = highest, true
	}
	if repository.sequence == ^uint64(0) {
		return 0, errors.New("commit sequence exhausted")
	}
	repository.sequence++
	return repository.sequence, nil
}

// highestRecordedSequence reads what previous processes committed. Sequences
// are global across slots because autosave rotation compares them between
// slots to pick the oldest.
func (repository *FileRepository) highestRecordedSequence() (uint64, error) {
	entries, err := filepath.Glob(filepath.Join(repository.directory, "slot_*_meta_*.json"))
	if err != nil {
		return 0, err
	}
	maximum := uint64(0)
	for _, path := range entries {
		data, err := os.ReadFile(path)
		if err != nil {
			continue
		}
		var metadata application.SaveMetadata
		if json.Unmarshal(data, &metadata) == nil && metadata.CommitSequence > maximum {
			maximum = metadata.CommitSequence
		}
	}
	return maximum, nil
}

// pruneSupersededRecords drops the records a slot no longer needs: metadata
// below the newest commit, and every world payload except the one that commit
// names. Records stay immutable while they are reachable; without this a
// campaign that autosaves every turn leaves a full world payload per turn on
// disk forever. Best effort, and never fatal to a save that already committed.
func (repository *FileRepository) pruneSupersededRecords(slot application.SlotID, keepSequence uint64, keepGeneration string) {
	metaPrefix := fmt.Sprintf("slot_%d_meta_", slot)
	keepMeta := fmt.Sprintf("%s%020d.json", metaPrefix, keepSequence)
	keepWorld := ""
	if keepGeneration != "" {
		keepWorld = fmt.Sprintf("slot_%d_world_%s.json", slot, keepGeneration)
	}
	for _, pattern := range []string{metaPrefix + "*.json", fmt.Sprintf("slot_%d_world_*.json", slot)} {
		entries, err := filepath.Glob(filepath.Join(repository.directory, pattern))
		if err != nil {
			continue
		}
		for _, path := range entries {
			switch filepath.Base(path) {
			case keepMeta, keepWorld:
				continue
			}
			_ = os.Remove(path)
		}
	}
}

var _ application.CampaignRepository = (*FileRepository)(nil)
