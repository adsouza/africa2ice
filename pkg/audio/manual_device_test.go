package audio

import (
	"os"
	"testing"
	"time"
)

// TestManualDevicePlayback is the only test that touches a real sound device,
// and it is skipped unless A2I_AUDIO_MANUAL is set. DESIGN.md §11 notes that
// no CI runner needs a sound device, which is true and also means nothing in
// the gate can catch a break in the oto wiring below. Run it by hand after
// changing Manager:
//
//	A2I_AUDIO_MANUAL=1 go test ./pkg/audio/ -run TestManualDevicePlayback -v
//
// It asserts the whole lifecycle the rest of the package fakes: the device
// opens, a sound plays, the player retires on its own, and prunePlayers drops
// it. The retirement deadline is generous because NewManager deliberately does
// not wait for oto's ready channel, so the first sound also pays the device's
// open latency (44-95ms on the reference Mac, cold to warm).
func TestManualDevicePlayback(t *testing.T) {
	if os.Getenv("A2I_AUDIO_MANUAL") == "" {
		t.Skip("manual: needs a real sound device")
	}
	manager, err := NewManager()
	if err != nil {
		t.Fatalf("NewManager() = %v", err)
	}
	manager.SetMaster(0.5, false)

	manager.Play(SFXChoiceClick)
	if len(manager.players) != 1 {
		t.Fatalf("players after Play = %d, want 1", len(manager.players))
	}
	time.Sleep(30 * time.Millisecond)
	if err := manager.Err(); err != nil {
		t.Fatalf("device error after Play: %v", err)
	}
	if !manager.players[0].IsPlaying() {
		t.Fatal("player is not playing 30ms into a 120ms sound")
	}

	start := time.Now()
	deadline := start.Add(5 * time.Second)
	for manager.players[0].IsPlaying() && time.Now().Before(deadline) {
		time.Sleep(10 * time.Millisecond)
	}
	if manager.players[0].IsPlaying() {
		t.Fatalf("player never retired: still playing after %v", time.Since(start))
	}
	t.Logf("player retired %v after the 30ms mark", time.Since(start))

	manager.Play(SFXSaveComplete)
	if len(manager.players) != 1 {
		t.Fatalf("players after pruning = %d, want the retired one dropped", len(manager.players))
	}
	time.Sleep(250 * time.Millisecond)
	if err := manager.Err(); err != nil {
		t.Fatalf("device error after the second sound: %v", err)
	}
}
