package logging

import (
	"errors"
	"strings"
	"testing"
)

// TestLogAudioFailureRecordsTheStageAndError covers the AerynOS report's blind
// spot: the ALSA error reached stderr only, so a JSON session log showed an
// orderly session.end and nothing about audio at all. A device failure must
// leave a record in the log a player can send.
func TestLogAudioFailureRecordsTheStageAndError(t *testing.T) {
	output := &bufferCloser{}
	session := newSession(strings.NewReader(strings.Repeat("q", 16)), "test", output)

	session.LogAudioFailure("init", errors.New("oto: ALSA error at snd_pcm_open: \"default\": No such file or directory"))

	logged := output.String()
	for _, required := range []string{"audio.failure", "init", "snd_pcm_open", "No such file or directory"} {
		if !strings.Contains(logged, required) {
			t.Fatalf("audio log missing %q: %s", required, logged)
		}
	}
	_ = session.Close()
}

// TestLogAudioFailureNilSessionIsSafe covers pkg/app's session-less paths,
// the same discipline every other Session method follows.
func TestLogAudioFailureNilSessionIsSafe(t *testing.T) {
	var session *Session
	session.LogAudioFailure("init", errors.New("no device"))
	session.LogAudioFailure("play", nil)
}
