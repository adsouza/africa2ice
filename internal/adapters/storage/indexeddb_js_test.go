//go:build js

package storage

import (
	"bytes"
	"errors"
	"os"
	"syscall/js"
	"testing"
	"time"

	"github.com/adsouza/africa2ice/internal/application"
)

func TestIndexedDBRepositoryBrowserContract(t *testing.T) {
	repository := NewIndexedDBRepository(nil)
	defer func() { _ = repository.Close() }()
	<-repository.ready
	if !repository.Writable() {
		t.Fatal("first browser repository did not acquire the writer lease")
	}
	secondary := NewIndexedDBRepository(nil)
	defer func() { _ = secondary.Close() }()
	<-secondary.ready
	if secondary.Writable() {
		t.Fatal("second browser repository acquired a concurrent writer lease")
	}
	service, err := application.NewGameService(42)
	if err != nil {
		t.Fatal(err)
	}
	state, err := service.ExportSaveState()
	if err != nil {
		t.Fatal(err)
	}
	if err := repository.BeginWrite(1, application.QuickSave, state); err != nil {
		t.Fatal(err)
	}
	write := waitForCompletion(t, repository, 1)
	if write.Err != nil || write.Metadata == nil || write.Metadata.SlotID != application.QuickSave {
		t.Fatalf("write completion = %#v", write)
	}
	if err := secondary.BeginWrite(10, application.QuickSave, state); err == nil {
		t.Fatal("read-only browser repository accepted a write")
	}

	state.Turn++
	if err := repository.BeginWrite(6, application.QuickSave, state); err != nil {
		t.Fatal(err)
	}
	if overwrite := waitForCompletion(t, repository, 6); overwrite.Err != nil {
		t.Fatalf("overwrite completion = %#v", overwrite)
	}
	if count := indexedStoreCount(t, repository, "worlds"); count != 1 {
		t.Fatalf("world generations after overwrite = %d, want 1", count)
	}
	if err := repository.BeginList(2); err != nil {
		t.Fatal(err)
	}
	list := waitForCompletion(t, repository, 2)
	if list.Err != nil || len(list.Slots) != 1 || list.Slots[0].SlotID != application.QuickSave {
		t.Fatalf("list completion = %#v", list)
	}
	if err := repository.BeginRead(3, application.QuickSave); err != nil {
		t.Fatal(err)
	}
	read := waitForCompletion(t, repository, 3)
	if read.Err != nil || read.State == nil {
		t.Fatalf("read completion = %#v", read)
	}
	want, err := application.EncodeSaveState(state)
	if err != nil {
		t.Fatal(err)
	}
	got, err := application.EncodeSaveState(*read.State)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(got, want) {
		t.Fatal("IndexedDB round trip changed the save payload")
	}
	if err := repository.BeginDelete(4, application.QuickSave); err != nil {
		t.Fatal(err)
	}
	deleted := waitForCompletion(t, repository, 4)
	if deleted.Err != nil || deleted.Metadata == nil || !deleted.Metadata.Deleted {
		t.Fatalf("delete completion = %#v", deleted)
	}
	if err := repository.BeginRead(5, application.QuickSave); err != nil {
		t.Fatal(err)
	}
	missing := waitForCompletion(t, repository, 5)
	if missing.Err == nil || missing.State != nil {
		t.Fatalf("deleted slot read = %#v", missing)
	}

	indexedStorePut(t, repository, "worlds", "orphan", "recoverable payload")
	indexedStorePut(t, repository, "metadata", slotKey(application.Manual1), "not json")
	if err := repository.BeginList(7); err != nil {
		t.Fatal(err)
	}
	if corruptList := waitForCompletion(t, repository, 7); corruptList.Err == nil {
		t.Fatal("list accepted malformed metadata")
	}
	if err := repository.collectUnreferencedWorlds(); err == nil {
		t.Fatal("generation collection accepted malformed metadata")
	}
	if count := indexedStoreCount(t, repository, "worlds"); count != 1 {
		t.Fatalf("world generations after unsafe collection = %d, want preserved orphan", count)
	}
}

func TestOldestSupportedSaveAdvancesAndResavesThroughIndexedDB(t *testing.T) {
	repository := waitForWritableIndexedDBRepository(t)
	defer func() { _ = repository.Close() }()
	clearIndexedStores(t, repository)
	defer clearIndexedStores(t, repository)
	fixture := oldestSupportedSave(t)
	if err := repository.BeginWrite(100, application.Manual2, fixture); err != nil {
		t.Fatal(err)
	}
	if completion := waitForCompletion(t, repository, 100); completion.Err != nil {
		t.Fatal(completion.Err)
	}

	service, err := application.NewGameServiceWithRepository(1, repository)
	if err != nil {
		t.Fatal(err)
	}
	loadID, err := service.BeginLoad(int(application.Manual2))
	if err != nil {
		t.Fatal(err)
	}
	if turn := waitForServiceOperation(t, service, uint64(loadID)); turn != fixture.Turn {
		t.Fatalf("loaded fixture turn = %d, want %d", turn, fixture.Turn)
	}
	advanced, err := service.EndTurn()
	if err != nil {
		t.Fatal(err)
	}
	if advanced.Turn != fixture.Turn+1 {
		t.Fatalf("advanced turn = %d, want %d", advanced.Turn, fixture.Turn+1)
	}
	wantHash, err := service.StateHash()
	if err != nil {
		t.Fatal(err)
	}
	saveID, err := service.BeginSave(int(application.Manual2))
	if err != nil {
		t.Fatal(err)
	}
	waitForServiceOperation(t, service, uint64(saveID))

	reloaded, err := application.NewGameServiceWithRepository(2, repository)
	if err != nil {
		t.Fatal(err)
	}
	reloadID, err := reloaded.BeginLoad(int(application.Manual2))
	if err != nil {
		t.Fatal(err)
	}
	if turn := waitForServiceOperation(t, reloaded, uint64(reloadID)); turn != fixture.Turn+1 {
		t.Fatalf("reloaded turn = %d, want %d", turn, fixture.Turn+1)
	}
	gotHash, err := reloaded.StateHash()
	if err != nil {
		t.Fatal(err)
	}
	if gotHash != wantHash {
		t.Fatalf("resaved fixture hash = %s, want %s", gotHash, wantHash)
	}
	if err := repository.BeginDelete(101, application.Manual2); err != nil {
		t.Fatal(err)
	}
	if completion := waitForCompletion(t, repository, 101); completion.Err != nil {
		t.Fatal(completion.Err)
	}
}

func waitForWritableIndexedDBRepository(t *testing.T) *IndexedDBRepository {
	t.Helper()
	deadline := time.Now().Add(5 * time.Second)
	for time.Now().Before(deadline) {
		repository := NewIndexedDBRepository(nil)
		<-repository.ready
		if repository.Writable() {
			return repository
		}
		_ = repository.Close()
		time.Sleep(10 * time.Millisecond)
	}
	t.Fatal("browser repository did not reacquire the writer lease")
	return nil
}

func clearIndexedStores(t *testing.T, repository *IndexedDBRepository) {
	t.Helper()
	transaction := repository.db.Call("transaction", js.ValueOf([]any{"worlds", "metadata", "control"}), "readwrite")
	for _, store := range []string{"worlds", "metadata", "control"} {
		transaction.Call("objectStore", store).Call("clear")
	}
	result := make(chan bool, 1)
	complete := js.FuncOf(func(this js.Value, args []js.Value) any { result <- true; return nil })
	failure := js.FuncOf(func(this js.Value, args []js.Value) any { result <- false; return nil })
	transaction.Set("oncomplete", complete)
	transaction.Set("onabort", failure)
	defer complete.Release()
	defer failure.Release()
	select {
	case ok := <-result:
		if !ok {
			t.Fatal("clearing IndexedDB stores failed")
		}
	case <-time.After(5 * time.Second):
		t.Fatal("clearing IndexedDB stores timed out")
	}
}

func waitForServiceOperation(t *testing.T, service *application.GameService, operationID uint64) int {
	t.Helper()
	deadline := time.Now().Add(5 * time.Second)
	for time.Now().Before(deadline) {
		for _, result := range service.PollStorage() {
			if uint64(result.OperationID) != operationID {
				continue
			}
			if result.Err != nil {
				t.Fatal(result.Err)
			}
			if result.ReplacementFrame != nil {
				return result.ReplacementFrame.Turn
			}
			return -1
		}
		time.Sleep(10 * time.Millisecond)
	}
	t.Fatalf("service operation %d timed out", operationID)
	return -1
}

func indexedStorePut(t *testing.T, repository *IndexedDBRepository, store, key, value string) {
	t.Helper()
	transaction := repository.db.Call("transaction", store, "readwrite")
	transaction.Call("objectStore", store).Call("put", value, key)
	result := make(chan bool, 1)
	complete := js.FuncOf(func(this js.Value, args []js.Value) any { result <- true; return nil })
	failure := js.FuncOf(func(this js.Value, args []js.Value) any { result <- false; return nil })
	transaction.Set("oncomplete", complete)
	transaction.Set("onabort", failure)
	defer complete.Release()
	defer failure.Release()
	select {
	case ok := <-result:
		if !ok {
			t.Fatalf("putting %s/%s failed", store, key)
		}
	case <-time.After(5 * time.Second):
		t.Fatalf("putting %s/%s timed out", store, key)
	}
}

func indexedStoreCount(t *testing.T, repository *IndexedDBRepository, store string) int {
	t.Helper()
	request := repository.db.Call("transaction", store, "readonly").Call("objectStore", store).Call("count")
	result := make(chan int, 1)
	callback := js.FuncOf(func(this js.Value, args []js.Value) any {
		result <- request.Get("result").Int()
		return nil
	})
	failure := js.FuncOf(func(this js.Value, args []js.Value) any {
		result <- -1
		return nil
	})
	request.Set("onsuccess", callback)
	request.Set("onerror", failure)
	defer callback.Release()
	defer failure.Release()
	select {
	case count := <-result:
		if count < 0 {
			t.Fatal("counting IndexedDB records failed")
		}
		return count
	case <-time.After(5 * time.Second):
		t.Fatal("counting IndexedDB records timed out")
		return 0
	}
}

func waitForCompletion(t *testing.T, repository *IndexedDBRepository, operation application.RepositoryOpID) application.RepositoryCompletion {
	t.Helper()
	deadline := time.Now().Add(5 * time.Second)
	for time.Now().Before(deadline) {
		for _, completion := range repository.Poll() {
			if completion.OperationID == operation {
				return completion
			}
		}
		time.Sleep(10 * time.Millisecond)
	}
	t.Fatalf("operation %d timed out", operation)
	return application.RepositoryCompletion{}
}

// funcOf is the only way this repository creates a JavaScript callback, and
// each invocation must run the session's panic hook on the way out.
func TestIndexedDBRepositoryCallbacksDeferThePanicGuard(t *testing.T) {
	calls := 0
	repository := NewIndexedDBRepository(func() { calls++ })
	defer func() { _ = repository.Close() }()
	<-repository.ready
	before := calls
	callback := repository.funcOf(func(js.Value, []js.Value) any { return "handled" })
	defer callback.Release()
	if result := callback.Invoke(); result.String() != "handled" {
		t.Fatalf("callback returned %v, want its handler's result", result)
	}
	if calls != before+1 {
		t.Fatalf("one callback invocation ran the panic guard %d times, want once", calls-before)
	}
}

func TestIndexedDBRepositoryContract(t *testing.T) {
	runRepositoryContract(t, repositoryBackend{
		fresh: func(t *testing.T) application.CampaignRepository {
			repository := waitForWritableIndexedDBRepository(t)
			clearIndexedStores(t, repository)
			return repository
		},
		reopen: func(t *testing.T) application.CampaignRepository {
			return waitForWritableIndexedDBRepository(t)
		},
	})
}

// idbFaultScript wraps every IDBObjectStore request method. While armed it
// counts requests and, on the failAt-th, aborts that request's transaction in
// a microtask: after the adapter has queued the rest of its synchronous
// requests, the way a quota failure or request error arrives as a later event.
// An abort that lands after the transaction finished is recorded as late.
const idbFaultScript = `(() => {
  if (globalThis.__idbFaults) return;
  const proto = IDBObjectStore.prototype;
  const methods = ["get", "put", "delete", "getAll", "getAllKeys"];
  const state = { armed: false, failAt: 0, count: 0, fired: false, late: false };
  for (const name of methods) {
    const original = proto[name];
    proto[name] = function (...args) {
      const request = original.apply(this, args);
      if (state.armed && ++state.count === state.failAt) {
        const transaction = this.transaction;
        state.fired = true;
        queueMicrotask(() => { try { transaction.abort(); } catch (error) { state.late = true; } });
      }
      return request;
    };
  }
  globalThis.__idbFaults = state;
})();`

func idbFaults(t *testing.T) js.Value {
	t.Helper()
	js.Global().Call("eval", idbFaultScript)
	state := js.Global().Get("__idbFaults")
	if state.IsUndefined() {
		t.Fatal("IndexedDB fault shim did not install")
	}
	return state
}

func armIDBFaults(state js.Value, failAt int) {
	state.Set("armed", failAt > 0)
	state.Set("failAt", failAt)
	state.Set("count", 0)
	state.Set("fired", false)
	state.Set("late", false)
}

// DESIGN.md §9: failure injection stops at every IndexedDB request and
// transaction event; the slot must then resolve to the complete old save, the
// complete new save, or a committed deletion, never a mixture. A failed
// transaction must not consume a commit sequence. Each case commits a save,
// arms an abort at request k, runs one operation, and reads the slot back. k
// grows until the operation completes without the abort firing, so every
// request is covered, including the post-commit orphan collection.
func TestIndexedDBProtocolSurvivesAnAbortAtEveryRequest(t *testing.T) {
	faults := idbFaults(t)
	defer armIDBFaults(faults, 0)
	const slot = application.QuickSave
	operations := map[string]struct {
		run      func(*IndexedDBRepository, application.RepositoryOpID) error
		newTurn  int
		deletion bool
	}{
		"overwrite": {run: func(repository *IndexedDBRepository, op application.RepositoryOpID) error {
			return repository.BeginWrite(op, slot, contractState(t, 8))
		}, newTurn: 8},
		"delete": {run: func(repository *IndexedDBRepository, op application.RepositoryOpID) error {
			return repository.BeginDelete(op, slot)
		}, deletion: true},
	}
	for name, operation := range operations {
		sawFailure, sawSuccess := false, false
		for failAt := 1; ; failAt++ {
			repository := waitForWritableIndexedDBRepository(t)
			clearIndexedStores(t, repository)
			committed := mustWrite(t, repository, 1, slot, contractState(t, 7))

			armIDBFaults(faults, failAt)
			if err := operation.run(repository, 2); err != nil {
				t.Fatal(err)
			}
			completion := awaitOne(t, repository)
			fired := faults.Get("fired").Bool() && !faults.Get("late").Bool()
			armIDBFaults(faults, 0)

			read := readSlot(t, repository, 3, slot)
			switch {
			case completion.Err != nil:
				sawFailure = true
				if read.Err != nil || read.State == nil || read.State.Turn != 7 || read.Metadata.CommitSequence != committed.CommitSequence {
					t.Fatalf("%s aborted at request %d (%v) but the slot is not the complete old save: %+v", name, failAt, completion.Err, read)
				}
				// The aborted transaction must have rolled the counter back.
				next := mustWrite(t, repository, 4, application.Manual1, contractState(t, 1))
				if next.CommitSequence != committed.CommitSequence+1 {
					t.Fatalf("%s aborted at request %d consumed a sequence: next commit is %d, want %d", name, failAt, next.CommitSequence, committed.CommitSequence+1)
				}
			case operation.deletion:
				sawSuccess = true
				if !errors.Is(read.Err, os.ErrNotExist) {
					t.Fatalf("delete reported success at request %d but the slot reads %+v", failAt, read)
				}
			default:
				sawSuccess = true
				if read.Err != nil || read.State == nil || read.State.Turn != operation.newTurn {
					t.Fatalf("overwrite reported success at request %d but the slot reads %+v", failAt, read)
				}
			}
			_ = repository.Close()
			if !fired {
				break // no abort landed: every request of the operation is covered
			}
		}
		if !sawFailure || !sawSuccess {
			t.Fatalf("%s: the abort sweep saw failure %t, success %t; it did not cross the commit", name, sawFailure, sawSuccess)
		}
	}
}
