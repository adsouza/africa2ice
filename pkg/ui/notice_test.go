package ui

import "testing"

func TestNoticeFramesGrowWithMessageLength(t *testing.T) {
	short := NoticeFrames("Workforce allocation applied")
	long := NoticeFrames("Bands cannot occupy open water; South Wallacea crosses it from here: select its highlighted far endpoint. Keep using arrows, or press Esc to clear.")
	if short < 180 {
		t.Fatalf("short notice lasts %d frames, under three seconds at 60 TPS", short)
	}
	if long <= short || long < 480 {
		t.Fatalf("long notice lasts %d frames vs short %d; want longer and at least eight seconds", long, short)
	}
	if huge := NoticeFrames(string(make([]rune, 10_000))); huge > 600 {
		t.Fatalf("unbounded notice lasts %d frames", huge)
	}
}
