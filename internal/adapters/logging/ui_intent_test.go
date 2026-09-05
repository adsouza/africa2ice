package logging

import (
	"strings"
	"testing"
)

// TestUIIntentLoggingRecordsIntentsRefusalsAndPointerPresses covers Wave I
// item I0: the gap between a pointer going down and an action dispatch,
// which is why the New Campaign click could not be diagnosed from a session
// log. Each method must write one bounded-scalar record; a nil session must
// stay safe, the same discipline LogActionDispatch/LogActionRejected already
// follow, since pkg/app has a session-less path callers rely on.
func TestUIIntentLoggingRecordsIntentsRefusalsAndPointerPresses(t *testing.T) {
	output := &bufferCloser{}
	session := newSession(strings.NewReader(strings.Repeat("q", 16)), "test", output)

	session.LogUIIntent("move-to-best", "band", uint64(3))
	session.LogUIIntentRefused("split", "campaign-over")
	session.LogUIPointer(452, 517, true, 1)

	logged := output.String()
	for _, required := range []string{
		"ui.intent", "move-to-best", "band",
		"ui.intent_refused", "split", "campaign-over",
		"ui.pointer", "452", "517", "over_chrome", "intents",
	} {
		if !strings.Contains(logged, required) {
			t.Fatalf("UI log missing %q: %s", required, logged)
		}
	}
	_ = session.Close()
}

// TestUIIntentLoggingNilSessionIsSafe covers the session-less path pkg/app's
// tests construct games without: every existing Session method begins with
// a session == nil guard, and callers rely on it.
func TestUIIntentLoggingNilSessionIsSafe(t *testing.T) {
	var session *Session
	session.LogUIIntent("split")
	session.LogUIIntentRefused("split", "campaign-over")
	session.LogUIPointer(0, 0, false, 0)
}
