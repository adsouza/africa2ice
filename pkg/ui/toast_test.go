package ui

import "testing"

func TestToastManagerCoalescesBoundsAndLetsErrorsDisplaceSuccess(t *testing.T) {
	manager := ToastManager{}
	manager.Push("one", false, 2)
	manager.Push("one", false, 5)
	if manager.count != 1 || manager.entries[0].RemainingFrames != 5 {
		t.Fatalf("coalesced manager = %#v", manager)
	}
	manager.Push("two", false, 2)
	manager.Push("three", false, 2)
	manager.Push("four", false, 2)
	manager.Push("dropped", false, 2)
	if manager.count != 4 {
		t.Fatalf("bounded manager count = %d", manager.count)
	}
	manager.Push("failure", true, 2)
	if manager.entries[3].Message != "failure" {
		t.Fatalf("error did not displace oldest success: %#v", manager)
	}
	for range 4 {
		manager.Tick()
	}
	if current, ok := manager.Current(); !ok || current.Message != "four" {
		t.Fatalf("FIFO after ticks = (%#v,%t)", current, ok)
	}
}
