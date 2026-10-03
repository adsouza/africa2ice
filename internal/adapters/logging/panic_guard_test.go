package logging

import (
	"strings"
	"testing"
)

// Composition hands owners the method value session.PanicGuard as a plain
// func(), and each owned goroutine runs `defer guard()`. recover only works
// when called directly by the deferred function, so this exercises exactly
// that shape, on a goroutine the entrypoint guard cannot see.
func TestPanicGuardMethodValueRecordsAndRepanicsOnAnotherGoroutine(t *testing.T) {
	output := &bufferCloser{}
	session := newSession(strings.NewReader(strings.Repeat("g", 16)), "test", output)
	guard := session.PanicGuard // the plain func() composition hands to owners
	original := &struct{ message string }{"worker failed"}
	recovered := make(chan any, 1)
	go func() {
		defer func() { recovered <- recover() }()
		defer guard()
		panic(original)
	}()
	if got := <-recovered; got != original {
		t.Fatalf("re-raised panic = %#v, want the original value", got)
	}
	_ = session.Close()
	var sawPanic, sawEnd bool
	for _, record := range sessionMessages(t, output.String()) {
		sawPanic = sawPanic || record["msg"] == "session.panic"
		sawEnd = sawEnd || record["msg"] == "session.end"
	}
	if !sawPanic || sawEnd {
		t.Fatalf("session.panic recorded %t, session.end written %t; want a panic record and no end marker", sawPanic, sawEnd)
	}
}

func TestPanicGuardIsInertWithoutAPanicOrASession(t *testing.T) {
	output := &bufferCloser{}
	session := newSession(strings.NewReader(strings.Repeat("n", 16)), "test", output)
	func() { defer session.PanicGuard() }()
	_ = session.Close()
	for _, record := range sessionMessages(t, output.String()) {
		if record["msg"] == "session.panic" {
			t.Fatal("PanicGuard recorded a panic that never happened")
		}
	}
	var none *Session
	recovered := make(chan any, 1)
	go func() {
		defer func() { recovered <- recover() }()
		defer none.PanicGuard()
		panic("still raised")
	}()
	if got := <-recovered; got != "still raised" {
		t.Fatalf("nil-session guard re-raised %#v, want the original panic", got)
	}
}
