//go:build js

package logging

import (
	"bytes"
	"encoding/json"
	"errors"
	"strings"
	"syscall/js"
	"testing"

	"github.com/adsouza/africa2ice/pkg/ui"
)

// sinkSpyScript records every way a browser page could persist or send data:
// network requests, beacons, sockets, IndexedDB, downloads, and file opens
// through wasm_exec.js's fs shim. Calls still go through; only the count
// matters.
const sinkSpyScript = `(() => {
  const calls = [];
  const wrap = (owner, name, label) => {
    if (!owner || typeof owner[name] !== "function") return;
    const original = owner[name];
    owner[name] = function (...args) { calls.push(label); return original.apply(this, args); };
  };
  wrap(globalThis, "fetch", "fetch");
  wrap(XMLHttpRequest.prototype, "open", "xhr");
  wrap(navigator, "sendBeacon", "beacon");
  wrap(globalThis, "WebSocket", "websocket");
  wrap(globalThis, "EventSource", "eventsource");
  wrap(indexedDB, "open", "indexeddb");
  wrap(indexedDB, "deleteDatabase", "indexeddb");
  wrap(URL, "createObjectURL", "download");
  wrap(HTMLAnchorElement.prototype, "click", "download");
  wrap(globalThis.fs, "open", "file");
  globalThis.__sinkSpy = calls;
})();`

// DESIGN.md step 11: the build-tagged web sink writes only to its injected
// stdout. Every line is one JSON object, and the sink touches no file,
// IndexedDB record, download, or network endpoint.
func TestWebSinkWritesOnlyJSONLinesToItsOutput(t *testing.T) {
	js.Global().Call("eval", sinkSpyScript)
	var out bytes.Buffer
	session := newWebSession(&out)
	session.LogActionDispatch(ui.EndTurnAction())
	session.LogActionRejected(1, errors.New("induced rejection"))
	session.LogUIPointer(10, 20, true, 0)
	session.LogInitError(errors.New("induced init error"))
	if err := session.Close(); err != nil {
		t.Fatal(err)
	}

	lines := strings.Split(strings.TrimSuffix(out.String(), "\n"), "\n")
	seen := map[string]bool{}
	for _, line := range lines {
		var record map[string]any
		decoder := json.NewDecoder(strings.NewReader(line))
		if err := decoder.Decode(&record); err != nil || decoder.More() {
			t.Fatalf("sink line is not exactly one JSON object: %q", line)
		}
		if record["target"] != "web" || record["session_id"] == "" {
			t.Fatalf("record lacks the web session fields: %v", record)
		}
		message, _ := record["msg"].(string)
		seen[message] = true
	}
	for _, want := range []string{"session.start", "action.dispatch", "action.rejected", "ui.pointer", "init.error"} {
		if !seen[want] {
			t.Errorf("sink emitted no %s record; saw %v", want, seen)
		}
	}
	if calls := js.Global().Get("__sinkSpy"); calls.Length() != 0 {
		t.Fatalf("the web sink reached outside its output: %v", calls.Call("join", ", ").String())
	}
}
