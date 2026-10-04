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

const (
	indexedDBName    = "africa2ice"
	indexedDBVersion = 1
)

var (
	errUpgradeBlocked = errors.New("saved games are unavailable until other africa2ice tabs close or reload")
	errReloadRequired = errors.New("saved games changed in another tab; reload to keep saving")
)

type indexedRequest struct {
	op    application.RepositoryOpID
	kind  application.RepositoryOperation
	slot  application.SlotID
	state *application.SaveState
}

type IndexedDBRepository struct {
	// mu guards the connection state below, which a blocked upgrade's late
	// success or a versionchange event can rewrite after open has returned.
	mu           sync.Mutex
	db           js.Value
	openErr      error
	writable     bool
	availability Availability
	canCollect   bool
	lockRelease  js.Value
	ready        chan struct{}
	requests     chan indexedRequest
	completions  chan application.RepositoryCompletion
	done         chan struct{}
	closeOnce    sync.Once
	// panicGuard is deferred by both goroutines and, through funcOf, by every
	// IndexedDB callback; never nil.
	panicGuard func()
}

// NewIndexedDBRepository opens the browser save store. panicGuard is the
// session's panic hook for the repository's goroutines and JavaScript
// callbacks, which run where the entrypoint guard cannot see; nil means none.
func NewIndexedDBRepository(panicGuard func()) *IndexedDBRepository {
	return newIndexedDBRepository(panicGuard, indexedDBVersion)
}

// newIndexedDBRepository opens at a chosen database version, so a test can
// stage the upgrade a future release will make.
func newIndexedDBRepository(panicGuard func(), version int) *IndexedDBRepository {
	repository := &IndexedDBRepository{ready: make(chan struct{}), requests: make(chan indexedRequest, 2), completions: make(chan application.RepositoryCompletion, 8), done: make(chan struct{}), panicGuard: orNoGuard(panicGuard)}
	go repository.open(version)
	go repository.worker()
	return repository
}

// Availability reports whether another tab is blocking this one's upgrade or
// has taken the database away. It never blocks.
func (repository *IndexedDBRepository) Availability() Availability {
	repository.mu.Lock()
	defer repository.mu.Unlock()
	return repository.availability
}

// connection returns the state an operation needs, read once under the lock.
func (repository *IndexedDBRepository) connection() (js.Value, error, bool) {
	repository.mu.Lock()
	defer repository.mu.Unlock()
	return repository.db, repository.openErr, repository.writable
}

// transaction starts one on the current connection, or fails if versionchange
// has closed it. An operation can yield between transactions, so each checks
// afresh rather than trusting the check the worker made before it began.
func (repository *IndexedDBRepository) transaction(stores any, mode string) (js.Value, error) {
	database, openErr, _ := repository.connection()
	if openErr != nil {
		return js.Value{}, openErr
	}
	return database.Call("transaction", js.ValueOf(stores), mode), nil
}

func (repository *IndexedDBRepository) collecting() bool {
	repository.mu.Lock()
	defer repository.mu.Unlock()
	return repository.canCollect
}

// connectionLost handles versionchange: another tab is upgrading or deleting
// the database, and it waits until every older connection closes. This one
// closes at once, stops writing, and hands the writer lease to whichever tab
// asks next. Operations from then on fail rather than reach a closed database,
// whose transaction() would throw.
func (repository *IndexedDBRepository) connectionLost() {
	repository.mu.Lock()
	database, release := repository.db, repository.lockRelease
	repository.db = js.Undefined()
	repository.openErr = errReloadRequired
	repository.writable = false
	repository.canCollect = false
	repository.availability = AvailabilityReloadRequired
	repository.lockRelease = js.Undefined()
	repository.mu.Unlock()
	if !database.IsUndefined() {
		database.Call("close")
	}
	if release.Type() == js.TypeFunction {
		release.Invoke()
	}
}

func (repository *IndexedDBRepository) open(version int) {
	defer close(repository.ready)
	defer repository.panicGuard()
	if err := repository.acquireWriterLease(); err != nil {
		repository.openErr = err
		return
	}
	indexedDB := js.Global().Get("indexedDB")
	if indexedDB.IsUndefined() || indexedDB.IsNull() {
		repository.openErr = errors.New("IndexedDB unavailable")
		return
	}
	request := indexedDB.Call("open", indexedDBName, version)
	result := make(chan error, 1)
	var settleOnce sync.Once
	settle := func(err error) { settleOnce.Do(func() { result <- err }) }
	// A blocked upgrade settles open with an error so startup is not held
	// hostage by another tab, but the request stays live and can still upgrade
	// and succeed once that tab lets go. Its callbacks are therefore retained
	// for the page's lifetime: releasing them here would make that late call
	// panic on a released function.
	upgrade := repository.funcOf(func(this js.Value, args []js.Value) any {
		database := request.Get("result")
		for _, name := range []string{"worlds", "metadata", "control"} {
			if !database.Get("objectStoreNames").Call("contains", name).Bool() {
				database.Call("createObjectStore", name)
			}
		}
		return nil
	})
	success := repository.funcOf(func(this js.Value, args []js.Value) any {
		database := request.Get("result")
		select {
		case <-repository.done:
			// Closed while blocked; nobody will use or close this connection.
			database.Call("close")
			return nil
		default:
		}
		database.Set("onversionchange", repository.funcOf(func(this js.Value, args []js.Value) any {
			repository.connectionLost()
			return nil
		}))
		repository.mu.Lock()
		repository.db = database
		repository.openErr = nil
		repository.availability = AvailabilityReady
		repository.mu.Unlock()
		settle(nil)
		return nil
	})
	failure := repository.funcOf(func(this js.Value, args []js.Value) any {
		err := jsError(request, "open IndexedDB")
		repository.mu.Lock()
		repository.openErr = err
		if repository.availability == AvailabilityUpgradeBlocked {
			// Startup already finished on the blocked error; this is final.
			repository.openErr = errReloadRequired
			repository.availability = AvailabilityReloadRequired
		}
		repository.mu.Unlock()
		settle(err)
		return nil
	})
	blocked := repository.funcOf(func(this js.Value, args []js.Value) any {
		repository.mu.Lock()
		repository.openErr = errUpgradeBlocked
		repository.availability = AvailabilityUpgradeBlocked
		repository.mu.Unlock()
		settle(errUpgradeBlocked)
		return nil
	})
	request.Set("onupgradeneeded", upgrade)
	request.Set("onsuccess", success)
	request.Set("onerror", failure)
	request.Set("onblocked", blocked)
	if err := <-result; err == nil && repository.collecting() {
		_ = repository.collectUnreferencedWorlds()
	}
}

// acquireWriterLease makes a tab the sole browser writer when Web Locks are
// available. Browsers without the API retain IndexedDB's transactional
// last-commit semantics, but skip orphan collection because ownership cannot
// be proven.
func (repository *IndexedDBRepository) acquireWriterLease() error {
	navigator := js.Global().Get("navigator")
	locks := navigator.Get("locks")
	if locks.IsUndefined() || locks.IsNull() {
		repository.writable = true
		return nil
	}

	acquired := make(chan error, 1)
	var signalOnce sync.Once
	signal := func(err error) { signalOnce.Do(func() { acquired <- err }) }
	executor := repository.funcOf(func(this js.Value, args []js.Value) any {
		repository.lockRelease = args[0]
		return nil
	})
	hold := js.Global().Get("Promise").New(executor)
	executor.Release()
	callback := repository.funcOf(func(this js.Value, args []js.Value) any {
		lock := args[0]
		if lock.IsUndefined() || lock.IsNull() {
			repository.writable = false
			signal(nil)
			return nil
		}
		repository.writable = true
		repository.canCollect = true
		signal(nil)
		return hold
	})
	failure := repository.funcOf(func(this js.Value, args []js.Value) any {
		message := "Web Lock request failed"
		if len(args) != 0 && !args[0].IsUndefined() {
			message = args[0].Get("message").String()
		}
		signal(errors.New(message))
		return nil
	})
	request := locks.Call("request", indexedDBName+"/saves", js.ValueOf(map[string]any{"mode": "exclusive", "ifAvailable": true}), callback)
	request.Call("catch", failure)
	// Both callbacks are retained by the lifetime-long lock promise. They are
	// intentionally released only with the page's Go runtime.
	return <-acquired
}

func (repository *IndexedDBRepository) BeginWrite(op application.RepositoryOpID, slot application.SlotID, state application.SaveState) error {
	if !repository.Writable() {
		return errors.New("repository is read-only")
	}
	return repository.enqueue(indexedRequest{op: op, kind: application.RepositoryWrite, slot: slot, state: &state})
}
func (repository *IndexedDBRepository) BeginRead(op application.RepositoryOpID, slot application.SlotID) error {
	return repository.enqueue(indexedRequest{op: op, kind: application.RepositoryRead, slot: slot})
}
func (repository *IndexedDBRepository) BeginDelete(op application.RepositoryOpID, slot application.SlotID) error {
	if !repository.Writable() {
		return errors.New("repository is read-only")
	}
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

func (repository *IndexedDBRepository) Writable() bool {
	select {
	case <-repository.ready:
		_, openErr, writable := repository.connection()
		return openErr == nil && writable
	default:
		// Composition asks before the asynchronous open completes. The eventual
		// operation still checks the authoritative capability after readiness.
		return true
	}
}

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
		repository.mu.Lock()
		database, release := repository.db, repository.lockRelease
		repository.lockRelease = js.Undefined()
		repository.mu.Unlock()
		if !database.IsUndefined() {
			database.Call("close")
		}
		if release.Type() == js.TypeFunction {
			release.Invoke()
		}
	})
	return nil
}

func (repository *IndexedDBRepository) worker() {
	defer repository.panicGuard()
	<-repository.ready
	for {
		select {
		case request := <-repository.requests:
			_, openErr, writable := repository.connection()
			completion := application.RepositoryCompletion{OperationID: request.op, Operation: request.kind, SlotID: request.slot, Writable: writable}
			switch {
			case openErr != nil:
				completion.Err = openErr
			case (request.kind == application.RepositoryWrite || request.kind == application.RepositoryDelete) && !writable:
				completion.Err = errors.New("repository is read-only")
			case request.kind == application.RepositoryWrite:
				completion.Metadata, completion.Err = repository.write(request.slot, *request.state)
			case request.kind == application.RepositoryRead:
				completion.State, completion.Metadata, completion.Err = repository.read(request.slot)
			case request.kind == application.RepositoryDelete:
				completion.Metadata, completion.Err = repository.delete(request.slot)
			case request.kind == application.RepositoryList:
				completion.Slots, completion.Err = repository.list()
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
	metadata, err := repository.commit(slot, func(sequence uint64, transaction js.Value) application.SaveMetadata {
		metadata := indexedMetadataFor(slot, sequence, generation, state)
		metadataBytes, _ := json.Marshal(metadata)
		transaction.Call("objectStore", "worlds").Call("put", string(worldBytes), generation)
		transaction.Call("objectStore", "metadata").Call("put", string(metadataBytes), slotKey(slot))
		return metadata
	})
	if err == nil && repository.collecting() {
		_ = repository.collectUnreferencedWorlds()
	}
	return metadata, err
}

func (repository *IndexedDBRepository) delete(slot application.SlotID) (*application.SaveMetadata, error) {
	metadata, err := repository.commit(slot, func(sequence uint64, transaction js.Value) application.SaveMetadata {
		metadata := application.SaveMetadata{SlotID: slot, CommitSequence: sequence, Deleted: true, SavedAt: time.Now().UTC()}
		metadataBytes, _ := json.Marshal(metadata)
		transaction.Call("objectStore", "metadata").Call("put", string(metadataBytes), slotKey(slot))
		return metadata
	})
	if err == nil && repository.collecting() {
		_ = repository.collectUnreferencedWorlds()
	}
	return metadata, err
}

func (repository *IndexedDBRepository) collectUnreferencedWorlds() error {
	transaction, err := repository.transaction([]any{"worlds", "metadata"}, "readwrite")
	if err != nil {
		return err
	}
	metadataRequest := transaction.Call("objectStore", "metadata").Call("getAll")
	result := make(chan error, 1)
	var finishOnce sync.Once
	finish := func(err error) { finishOnce.Do(func() { result <- err }) }
	referenced := make(map[string]struct{})
	var abortErr error
	var worldKeysRequest js.Value
	var metadataSuccess, keysSuccess, complete, failure js.Func
	keysSuccess = repository.funcOf(func(this js.Value, args []js.Value) any {
		keys := worldKeysRequest.Get("result")
		worlds := transaction.Call("objectStore", "worlds")
		for index := 0; index < keys.Length(); index++ {
			key := keys.Index(index).String()
			if _, live := referenced[key]; !live {
				worlds.Call("delete", key)
			}
		}
		return nil
	})
	metadataSuccess = repository.funcOf(func(this js.Value, args []js.Value) any {
		values := metadataRequest.Get("result")
		for index := 0; index < values.Length(); index++ {
			var metadata application.SaveMetadata
			if err := json.Unmarshal([]byte(values.Index(index).String()), &metadata); err != nil {
				// A malformed live pointer is a recovery problem, not evidence that
				// every world generation is unreferenced. Preserve all payloads.
				abortErr = fmt.Errorf("decode IndexedDB save metadata: %w", err)
				transaction.Call("abort")
				return nil
			}
			if !metadata.Deleted && metadata.Generation != "" {
				referenced[metadata.Generation] = struct{}{}
			}
		}
		worldKeysRequest = transaction.Call("objectStore", "worlds").Call("getAllKeys")
		worldKeysRequest.Set("onsuccess", keysSuccess)
		return nil
	})
	complete = repository.funcOf(func(this js.Value, args []js.Value) any { finish(nil); return nil })
	failure = repository.funcOf(func(this js.Value, args []js.Value) any {
		if abortErr != nil {
			finish(abortErr)
			return nil
		}
		finish(jsError(transaction, "collect IndexedDB save generations"))
		return nil
	})
	metadataRequest.Set("onsuccess", metadataSuccess)
	transaction.Set("oncomplete", complete)
	transaction.Set("onabort", failure)
	err = <-result
	metadataRequest.Set("onsuccess", js.Null())
	if worldKeysRequest.Type() == js.TypeObject {
		worldKeysRequest.Set("onsuccess", js.Null())
	}
	transaction.Set("oncomplete", js.Null())
	transaction.Set("onabort", js.Null())
	metadataSuccess.Release()
	keysSuccess.Release()
	complete.Release()
	failure.Release()
	return err
}

func (repository *IndexedDBRepository) commit(slot application.SlotID, prepare func(uint64, js.Value) application.SaveMetadata) (*application.SaveMetadata, error) {
	transaction, err := repository.transaction([]any{"worlds", "metadata", "control"}, "readwrite")
	if err != nil {
		return nil, err
	}
	result := make(chan error, 1)
	var metadata application.SaveMetadata
	var abortErr error
	counterRequest := transaction.Call("objectStore", "control").Call("get", "sequence")
	var counterSuccess, complete, failure js.Func
	counterSuccess = repository.funcOf(func(this js.Value, args []js.Value) any {
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
	complete = repository.funcOf(func(this js.Value, args []js.Value) any { result <- nil; return nil })
	failure = repository.funcOf(func(this js.Value, args []js.Value) any {
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
	err = <-result
	counterSuccess.Release()
	complete.Release()
	failure.Release()
	if err != nil {
		return nil, err
	}
	return &metadata, nil
}

func (repository *IndexedDBRepository) read(slot application.SlotID) (*application.SaveState, *application.SaveMetadata, error) {
	transaction, err := repository.transaction([]any{"metadata", "worlds"}, "readonly")
	if err != nil {
		return nil, nil, err
	}
	result := make(chan error, 1)
	var metadata application.SaveMetadata
	var state application.SaveState
	metadataRequest := transaction.Call("objectStore", "metadata").Call("get", slotKey(slot))
	var metadataSuccess, worldSuccess, failure js.Func
	worldSuccess = repository.funcOf(func(this js.Value, args []js.Value) any { return nil })
	metadataSuccess = repository.funcOf(func(this js.Value, args []js.Value) any {
		value := metadataRequest.Get("result")
		if value.IsUndefined() {
			result <- errSlotEmpty
			return nil
		}
		if err := json.Unmarshal([]byte(value.String()), &metadata); err != nil || metadata.Deleted {
			if err == nil {
				err = errSlotEmpty
			}
			result <- err
			return nil
		}
		worldRequest := transaction.Call("objectStore", "worlds").Call("get", metadata.Generation)
		worldSuccess.Release()
		worldSuccess = repository.funcOf(func(this js.Value, args []js.Value) any {
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
	failure = repository.funcOf(func(this js.Value, args []js.Value) any {
		result <- jsError(transaction, "read IndexedDB save")
		return nil
	})
	metadataRequest.Set("onsuccess", metadataSuccess)
	metadataRequest.Set("onerror", failure)
	err = <-result
	metadataSuccess.Release()
	worldSuccess.Release()
	failure.Release()
	if err != nil {
		return nil, &metadata, err
	}
	return &state, &metadata, nil
}

func (repository *IndexedDBRepository) list() ([]application.SaveMetadata, error) {
	transaction, err := repository.transaction("metadata", "readonly")
	if err != nil {
		return nil, err
	}
	result := make(chan error, 1)
	slots := []application.SlotID{application.Manual1, application.Manual2, application.Manual3, application.QuickSave, application.Auto1, application.Auto2, application.Auto3}
	metadata := make([]application.SaveMetadata, len(slots))
	requests := make([]js.Value, 0, len(slots))
	functions := make([]js.Func, 0, len(slots)+3)
	var operationErr error
	var finishOnce sync.Once
	finish := func(err error) { finishOnce.Do(func() { result <- err }) }
	complete := repository.funcOf(func(this js.Value, args []js.Value) any {
		finish(nil)
		return nil
	})
	abort := repository.funcOf(func(this js.Value, args []js.Value) any {
		if operationErr == nil {
			operationErr = jsError(transaction, "list IndexedDB saves")
		}
		finish(operationErr)
		return nil
	})
	requestFailure := repository.funcOf(func(this js.Value, args []js.Value) any {
		if operationErr == nil {
			operationErr = jsError(transaction, "list IndexedDB saves")
		}
		return nil
	})
	functions = append(functions, complete, abort, requestFailure)
	transaction.Set("oncomplete", complete)
	transaction.Set("onabort", abort)
	transaction.Set("onerror", requestFailure)
	for index, slot := range slots {
		request := transaction.Call("objectStore", "metadata").Call("get", slotKey(slot))
		callback := repository.funcOf(func(this js.Value, args []js.Value) any {
			if value := request.Get("result"); !value.IsUndefined() {
				if err := json.Unmarshal([]byte(value.String()), &metadata[index]); err != nil {
					operationErr = fmt.Errorf("decode IndexedDB metadata for slot %d: %w", slot, err)
					transaction.Call("abort")
				}
			}
			return nil
		})
		functions = append(functions, callback)
		requests = append(requests, request)
		request.Set("onsuccess", callback)
		request.Set("onerror", requestFailure)
	}
	err = <-result
	transaction.Set("oncomplete", js.Null())
	transaction.Set("onabort", js.Null())
	transaction.Set("onerror", js.Null())
	for _, request := range requests {
		request.Set("onsuccess", js.Null())
		request.Set("onerror", js.Null())
	}
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
	metadata.SapiensPopulation, metadata.ArchaicPopulation = state.PopulationBySpecies()
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

// funcOf is js.FuncOf for this repository's callbacks: each invocation runs
// the session's panic hook, because a callback runs on the JavaScript event
// loop where the entrypoint guard cannot see a panic.
func (repository *IndexedDBRepository) funcOf(handler func(this js.Value, args []js.Value) any) js.Func {
	return js.FuncOf(func(this js.Value, args []js.Value) any {
		defer repository.panicGuard()
		return handler(this, args)
	})
}
