//go:build js

package ui

import (
	"errors"
	"fmt"
	"sync"
	"syscall/js"
)

const (
	uiSettingsDatabaseName = "africa2ice-ui"
	uiSettingsStoreName    = "settings"
	uiSettingsRecordKey    = "preferences"
)

type IndexedDBUISettingsStore struct {
	mu             sync.Mutex
	active         bool
	completions    []UISettingsCompletion
	pendingRelease []js.Func
}

func NewIndexedDBUISettingsStore() (*IndexedDBUISettingsStore, error) {
	if js.Global().Get("indexedDB").IsUndefined() {
		return nil, errors.New("IndexedDB is unavailable")
	}
	return &IndexedDBUISettingsStore{}, nil
}

func (store *IndexedDBUISettingsStore) BeginRead(revision uint64) error {
	if !store.begin() {
		return errors.New("UI settings operation already active")
	}
	store.open(UISettingsRead, revision, DefaultUISettings())
	return nil
}

func (store *IndexedDBUISettingsStore) BeginWrite(revision uint64, settings UISettings) error {
	if !store.begin() {
		return errors.New("UI settings operation already active")
	}
	store.open(UISettingsWrite, revision, NormalizeUISettings(settings))
	return nil
}

func (store *IndexedDBUISettingsStore) Poll() []UISettingsCompletion {
	store.mu.Lock()
	result := append([]UISettingsCompletion(nil), store.completions...)
	store.completions = store.completions[:0]
	callbacks := store.pendingRelease
	store.pendingRelease = nil
	store.mu.Unlock()
	for _, callback := range callbacks {
		callback.Release()
	}
	return result
}

func (store *IndexedDBUISettingsStore) begin() bool {
	store.mu.Lock()
	defer store.mu.Unlock()
	if store.active {
		return false
	}
	store.active = true
	return true
}

func (store *IndexedDBUISettingsStore) open(operation UISettingsOperation, revision uint64, settings UISettings) {
	callbacks := make([]js.Func, 0, 8)
	finished := false
	finish := func(result UISettings, err error) {
		if finished {
			return
		}
		finished = true
		store.mu.Lock()
		store.active = false
		store.completions = append(store.completions, UISettingsCompletion{Operation: operation, Revision: revision, Settings: result, Err: err})
		store.pendingRelease = append(store.pendingRelease, callbacks...)
		store.mu.Unlock()
	}
	callback := func(handler func(js.Value, []js.Value)) js.Func {
		function := js.FuncOf(func(this js.Value, arguments []js.Value) any {
			handler(this, arguments)
			return nil
		})
		callbacks = append(callbacks, function)
		return function
	}
	request := js.Global().Get("indexedDB").Call("open", uiSettingsDatabaseName, 1)
	request.Set("onblocked", callback(func(js.Value, []js.Value) {
		finish(DefaultUISettings(), errors.New("UI settings database upgrade is blocked by another tab"))
	}))
	request.Set("onerror", callback(func(js.Value, []js.Value) {
		finish(DefaultUISettings(), indexedDBError("open UI settings", request))
	}))
	request.Set("onupgradeneeded", callback(func(_ js.Value, arguments []js.Value) {
		database := arguments[0].Get("target").Get("result")
		if !database.Get("objectStoreNames").Call("contains", uiSettingsStoreName).Bool() {
			database.Call("createObjectStore", uiSettingsStoreName)
		}
	}))
	request.Set("onsuccess", callback(func(_ js.Value, arguments []js.Value) {
		database := arguments[0].Get("target").Get("result")
		database.Set("onversionchange", callback(func(js.Value, []js.Value) { database.Call("close") }))
		mode := "readonly"
		if operation == UISettingsWrite {
			mode = "readwrite"
		}
		transaction := database.Call("transaction", uiSettingsStoreName, mode)
		transaction.Set("onabort", callback(func(js.Value, []js.Value) {
			database.Call("close")
			finish(DefaultUISettings(), indexedDBError("UI settings transaction", transaction))
		}))
		transaction.Set("onerror", callback(func(js.Value, []js.Value) {}))
		objectStore := transaction.Call("objectStore", uiSettingsStoreName)
		result := settings
		if operation == UISettingsRead {
			get := objectStore.Call("get", uiSettingsRecordKey)
			get.Set("onerror", callback(func(js.Value, []js.Value) {
				finish(DefaultUISettings(), indexedDBError("read UI settings", get))
			}))
			get.Set("onsuccess", callback(func(_ js.Value, arguments []js.Value) {
				value := arguments[0].Get("target").Get("result")
				if value.IsUndefined() {
					result = DefaultUISettings()
					return
				}
				decoded, err := DecodeUISettings([]byte(value.String()))
				if err != nil {
					result = DefaultUISettings()
					return
				}
				result = decoded
			}))
		} else {
			payload, err := EncodeUISettings(settings)
			if err != nil {
				database.Call("close")
				finish(DefaultUISettings(), err)
				return
			}
			put := objectStore.Call("put", string(payload), uiSettingsRecordKey)
			put.Set("onerror", callback(func(js.Value, []js.Value) {
				finish(DefaultUISettings(), indexedDBError("write UI settings", put))
			}))
		}
		transaction.Set("oncomplete", callback(func(js.Value, []js.Value) {
			database.Call("close")
			finish(result, nil)
		}))
	}))
}

func indexedDBError(context string, source js.Value) error {
	value := source.Get("error")
	if value.IsNull() || value.IsUndefined() {
		return errors.New(context)
	}
	message := value.Get("message")
	if !message.IsUndefined() && message.String() != "" {
		return fmt.Errorf("%s: %s", context, message.String())
	}
	return fmt.Errorf("%s: %s", context, value.String())
}
