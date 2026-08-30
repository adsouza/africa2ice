//go:build js

package storage

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"strconv"
	"sync"
	"syscall/js"
	"time"

	"github.com/adsouza/africa2ice/internal/application"
)

const indexedDBName = "africa2ice"

type indexedRequest struct {
	op    application.RepositoryOpID
	kind  application.RepositoryOperation
	slot  application.SlotID
	state *application.SaveState
}

type IndexedDBRepository struct {
	db          js.Value
	openErr     error
	ready       chan struct{}
	requests    chan indexedRequest
	completions chan application.RepositoryCompletion
	done        chan struct{}
	closeOnce   sync.Once
}

func NewIndexedDBRepository() *IndexedDBRepository {
	repository := &IndexedDBRepository{ready: make(chan struct{}), requests: make(chan indexedRequest, 2), completions: make(chan application.RepositoryCompletion, 8), done: make(chan struct{})}
	go repository.open()
	go repository.worker()
	return repository
}

func (repository *IndexedDBRepository) open() {
	defer close(repository.ready)
	indexedDB := js.Global().Get("indexedDB")
	if indexedDB.IsUndefined() || indexedDB.IsNull() {
		repository.openErr = errors.New("IndexedDB unavailable")
		return
	}
	request := indexedDB.Call("open", indexedDBName, 1)
	result := make(chan error, 1)
	var upgrade, success, failure js.Func
	upgrade = js.FuncOf(func(this js.Value, args []js.Value) any {
		database := request.Get("result")
		for _, name := range []string{"worlds", "metadata", "control"} {
			if !database.Get("objectStoreNames").Call("contains", name).Bool() {
				database.Call("createObjectStore", name)
			}
		}
		return nil
	})
	success = js.FuncOf(func(this js.Value, args []js.Value) any {
		repository.db = request.Get("result")
		result <- nil
		return nil
	})
	failure = js.FuncOf(func(this js.Value, args []js.Value) any {
		result <- jsError(request, "open IndexedDB")
		return nil
	})
	request.Set("onupgradeneeded", upgrade)
	request.Set("onsuccess", success)
	request.Set("onerror", failure)
	request.Set("onblocked", failure)
	repository.openErr = <-result
	upgrade.Release()
	success.Release()
	failure.Release()
}

func (repository *IndexedDBRepository) BeginWrite(op application.RepositoryOpID, slot application.SlotID, state application.SaveState) error {
	return repository.enqueue(indexedRequest{op: op, kind: application.RepositoryWrite, slot: slot, state: &state})
}
func (repository *IndexedDBRepository) BeginRead(op application.RepositoryOpID, slot application.SlotID) error {
	return repository.enqueue(indexedRequest{op: op, kind: application.RepositoryRead, slot: slot})
}
func (repository *IndexedDBRepository) BeginDelete(op application.RepositoryOpID, slot application.SlotID) error {
	return repository.enqueue(indexedRequest{op: op, kind: application.RepositoryDelete, slot: slot})
}
func (repository *IndexedDBRepository) BeginList(op application.RepositoryOpID) error {
	return repository.enqueue(indexedRequest{op: op, kind: application.RepositoryList})
}

func (repository *IndexedDBRepository) enqueue(request indexedRequest) error {
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

func (repository *IndexedDBRepository) Writable() bool { return true }

func (repository *IndexedDBRepository) Poll() []application.RepositoryCompletion {
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

func (repository *IndexedDBRepository) Close() error {
	repository.closeOnce.Do(func() {
		close(repository.done)
		<-repository.ready
		if repository.openErr == nil && !repository.db.IsUndefined() {
			repository.db.Call("close")
		}
	})
	return nil
}

func (repository *IndexedDBRepository) worker() {
	<-repository.ready
	for {
		select {
		case request := <-repository.requests:
			completion := application.RepositoryCompletion{OperationID: request.op, Operation: request.kind, SlotID: request.slot, Writable: true}
			if repository.openErr != nil {
				completion.Err = repository.openErr
			} else {
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
			}
			repository.completions <- completion
		case <-repository.done:
			return
		}
	}
}

func (repository *IndexedDBRepository) write(slot application.SlotID, state application.SaveState) (*application.SaveMetadata, error) {
	worldBytes, err := application.EncodeSaveState(state)
	if err != nil {
		return nil, err
	}
	hash := sha256.Sum256(worldBytes)
	generation := hex.EncodeToString(hash[:])
	return repository.commit(slot, func(sequence uint64, transaction js.Value) application.SaveMetadata {
		metadata := indexedMetadataFor(slot, sequence, generation, state)
		metadataBytes, _ := json.Marshal(metadata)
		transaction.Call("objectStore", "worlds").Call("put", string(worldBytes), generation)
		transaction.Call("objectStore", "metadata").Call("put", string(metadataBytes), slotKey(slot))
		return metadata
	})
}

func (repository *IndexedDBRepository) delete(slot application.SlotID) (*application.SaveMetadata, error) {
	return repository.commit(slot, func(sequence uint64, transaction js.Value) application.SaveMetadata {
		metadata := application.SaveMetadata{SlotID: slot, CommitSequence: sequence, Deleted: true, SavedAt: time.Now().UTC()}
		metadataBytes, _ := json.Marshal(metadata)
		transaction.Call("objectStore", "metadata").Call("put", string(metadataBytes), slotKey(slot))
		return metadata
	})
}

func (repository *IndexedDBRepository) commit(slot application.SlotID, prepare func(uint64, js.Value) application.SaveMetadata) (*application.SaveMetadata, error) {
	transaction := repository.db.Call("transaction", js.ValueOf([]any{"worlds", "metadata", "control"}), "readwrite")
	result := make(chan error, 1)
	var metadata application.SaveMetadata
	var abortErr error
	counterRequest := transaction.Call("objectStore", "control").Call("get", "sequence")
	var counterSuccess, complete, failure js.Func
	counterSuccess = js.FuncOf(func(this js.Value, args []js.Value) any {
		sequence := uint64(1)
		if value := counterRequest.Get("result"); !value.IsUndefined() {
			current, err := indexedSequence(value)
			if err != nil || current == ^uint64(0) {
				if err == nil {
					err = errors.New("save commit sequence exhausted")
				}
				abortErr = err
				transaction.Call("abort")
				return nil
			}
			sequence = current + 1
		}
		metadata = prepare(sequence, transaction)
		// JavaScript numbers cannot exactly represent all uint64 values. Persist the
		// monotonic counter as decimal text so its ordering contract survives long
		// campaigns and repeated overwrites without a hidden 2^53 ceiling.
		transaction.Call("objectStore", "control").Call("put", strconv.FormatUint(sequence, 10), "sequence")
		return nil
	})
	complete = js.FuncOf(func(this js.Value, args []js.Value) any { result <- nil; return nil })
	failure = js.FuncOf(func(this js.Value, args []js.Value) any {
		if abortErr != nil {
			result <- abortErr
			return nil
		}
		result <- jsError(transaction, "commit IndexedDB transaction")
		return nil
	})
	counterRequest.Set("onsuccess", counterSuccess)
	transaction.Set("oncomplete", complete)
	transaction.Set("onabort", failure)
	err := <-result
	counterSuccess.Release()
	complete.Release()
	failure.Release()
	if err != nil {
		return nil, err
	}
	return &metadata, nil
}

func (repository *IndexedDBRepository) read(slot application.SlotID) (*application.SaveState, *application.SaveMetadata, error) {
	transaction := repository.db.Call("transaction", js.ValueOf([]any{"metadata", "worlds"}), "readonly")
	result := make(chan error, 1)
	var metadata application.SaveMetadata
	var state application.SaveState
	metadataRequest := transaction.Call("objectStore", "metadata").Call("get", slotKey(slot))
	var metadataSuccess, worldSuccess, failure js.Func
	worldSuccess = js.FuncOf(func(this js.Value, args []js.Value) any { return nil })
	metadataSuccess = js.FuncOf(func(this js.Value, args []js.Value) any {
		value := metadataRequest.Get("result")
		if value.IsUndefined() {
			result <- errors.New("save slot not found")
			return nil
		}
		if err := json.Unmarshal([]byte(value.String()), &metadata); err != nil || metadata.Deleted {
			if err == nil {
				err = errors.New("save slot deleted")
			}
			result <- err
			return nil
		}
		worldRequest := transaction.Call("objectStore", "worlds").Call("get", metadata.Generation)
		worldSuccess.Release()
		worldSuccess = js.FuncOf(func(this js.Value, args []js.Value) any {
			worldValue := worldRequest.Get("result")
			if worldValue.IsUndefined() {
				result <- errors.New("save generation missing")
				return nil
			}
			worldBytes := []byte(worldValue.String())
			hash := sha256.Sum256(worldBytes)
			if hex.EncodeToString(hash[:]) != metadata.Generation {
				result <- errors.New("save checksum mismatch")
				return nil
			}
			decoded, err := application.DecodeSaveState(worldBytes)
			if err == nil {
				state = decoded
			}
			result <- err
			return nil
		})
		worldRequest.Set("onsuccess", worldSuccess)
		worldRequest.Set("onerror", failure)
		return nil
	})
	failure = js.FuncOf(func(this js.Value, args []js.Value) any {
		result <- jsError(transaction, "read IndexedDB save")
		return nil
	})
	metadataRequest.Set("onsuccess", metadataSuccess)
	metadataRequest.Set("onerror", failure)
	err := <-result
	metadataSuccess.Release()
	worldSuccess.Release()
	failure.Release()
	if err != nil {
		return nil, &metadata, err
	}
	return &state, &metadata, nil
}

func (repository *IndexedDBRepository) list() ([]application.SaveMetadata, error) {
	transaction := repository.db.Call("transaction", "metadata", "readonly")
	result := make(chan error, 1)
	slots := []application.SlotID{application.Manual1, application.Manual2, application.Manual3, application.QuickSave, application.Auto1, application.Auto2, application.Auto3}
	metadata := make([]application.SaveMetadata, len(slots))
	remaining := len(slots)
	functions := make([]js.Func, 0, len(slots)+1)
	failure := js.FuncOf(func(this js.Value, args []js.Value) any {
		result <- jsError(transaction, "list IndexedDB saves")
		return nil
	})
	functions = append(functions, failure)
	for index, slot := range slots {
		request := transaction.Call("objectStore", "metadata").Call("get", slotKey(slot))
		callback := js.FuncOf(func(this js.Value, args []js.Value) any {
			if value := request.Get("result"); !value.IsUndefined() {
				_ = json.Unmarshal([]byte(value.String()), &metadata[index])
			}
			remaining--
			if remaining == 0 {
				result <- nil
			}
			return nil
		})
		functions = append(functions, callback)
		request.Set("onsuccess", callback)
		request.Set("onerror", failure)
	}
	err := <-result
	for _, function := range functions {
		function.Release()
	}
	if err != nil {
		return nil, err
	}
	compact := make([]application.SaveMetadata, 0, len(metadata))
	for _, item := range metadata {
		if item.SlotID != 0 && !item.Deleted {
			compact = append(compact, item)
		}
	}
	return compact, nil
}

func indexedMetadataFor(slot application.SlotID, sequence uint64, generation string, state application.SaveState) application.SaveMetadata {
	metadata := application.SaveMetadata{SlotID: slot, CommitSequence: sequence, Generation: generation, SchemaVersion: state.SchemaVersion, CampaignClockAlgorithm: state.CampaignClockAlgorithm, WorldRevision: state.WorldRevision, Turn: state.Turn, SavedAt: time.Now().UTC()}
	for _, band := range state.Bands {
		if band.Species == 0 {
			metadata.SapiensPopulation += uint64(band.Population)
		} else {
			metadata.ArchaicPopulation += uint64(band.Population)
		}
	}
	return metadata
}

func slotKey(slot application.SlotID) string { return fmt.Sprintf("slot_%d", slot) }

func indexedSequence(value js.Value) (uint64, error) {
	switch value.Type() {
	case js.TypeString:
		sequence, err := strconv.ParseUint(value.String(), 10, 64)
		if err != nil {
			return 0, fmt.Errorf("invalid save commit sequence: %w", err)
		}
		return sequence, nil
	case js.TypeNumber:
		// Accept the early-development numeric representation only while it is
		// still an exact non-negative JavaScript integer, then rewrite it as text.
		sequence := value.Float()
		if math.IsNaN(sequence) || math.IsInf(sequence, 0) || sequence < 0 || sequence > 1<<53 || math.Trunc(sequence) != sequence {
			return 0, errors.New("invalid numeric save commit sequence")
		}
		return uint64(sequence), nil
	default:
		return 0, errors.New("invalid save commit sequence type")
	}
}

func jsError(value js.Value, prefix string) error {
	detail := value.Get("error")
	if detail.IsUndefined() || detail.IsNull() {
		return errors.New(prefix)
	}
	return fmt.Errorf("%s: %s", prefix, detail.Get("message").String())
}

var _ application.CampaignRepository = (*IndexedDBRepository)(nil)
