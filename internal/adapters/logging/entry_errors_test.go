package logging

import (
	"encoding/json"
	"errors"
	"strings"
	"testing"
)

// A failure that ends the process before or during the game loop must reach
// the session log, not only stderr, which a double-clicked desktop build has
// nowhere to show.
func TestEntryPointErrorsAreRecorded(t *testing.T) {
	output := &bufferCloser{}
	session := newSession(strings.NewReader(strings.Repeat("e", 16)), "test", output)
	session.LogInitError(errors.New("no graphics device"))
	session.LogRunError(errors.New("loop failed"))
	_ = session.Close()
	want := map[string]string{"init.error": "no graphics device", "run.error": "loop failed"}
	for _, line := range strings.Split(strings.TrimSpace(output.String()), "\n") {
		var record map[string]any
		if err := json.Unmarshal([]byte(line), &record); err != nil {
			t.Fatal(err)
		}
		message, _ := record["msg"].(string)
		if expected, ok := want[message]; ok {
			if record["error"] != expected || record["level"] != "ERROR" {
				t.Fatalf("%s record = %v", message, record)
			}
			delete(want, message)
		}
	}
	if len(want) != 0 {
		t.Fatalf("missing records: %v", want)
	}
}

func sessionMessages(t *testing.T, output string) []map[string]any {
	t.Helper()
	var records []map[string]any
	for _, line := range strings.Split(strings.TrimSpace(output), "\n") {
		var record map[string]any
		if err := json.Unmarshal([]byte(line), &record); err != nil {
			t.Fatal(err)
		}
		records = append(records, record)
	}
	return records
}

// session.end means the process shut down normally, so it carries whether an
// entry-point error was recorded, and a crash never writes one: the log of a
// panicked session ends at session.panic.
func TestSessionEndReportsOutcomeAndNeverFollowsAPanic(t *testing.T) {
	for _, tc := range []struct {
		name    string
		act     func(*Session)
		outcome string // "" means session.end must be absent
	}{
		{"clean", func(*Session) {}, "success"},
		{"run error", func(s *Session) { s.LogRunError(errors.New("loop failed")) }, "error"},
		{"panic", func(s *Session) {
			defer func() { _ = recover() }()
			defer GuardPanic(s)
			panic("boom")
		}, ""},
	} {
		t.Run(tc.name, func(t *testing.T) {
			output := &bufferCloser{}
			session := newSession(strings.NewReader(strings.Repeat("o", 16)), "test", output)
			tc.act(session)
			_ = session.Close()
			var end map[string]any
			for _, record := range sessionMessages(t, output.String()) {
				if record["msg"] == "session.end" {
					end = record
				}
			}
			switch {
			case tc.outcome == "" && end != nil:
				t.Fatalf("session.end written after a panic: %v", end)
			case tc.outcome != "" && (end == nil || end["outcome"] != tc.outcome):
				t.Fatalf("session.end = %v, want outcome %q", end, tc.outcome)
			}
		})
	}
}
