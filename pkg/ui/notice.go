package ui

import "unicode/utf8"

// Notice timing at 60 TPS: a base to catch the eye plus reading time at a
// comfortable pace, clamped so short confirmations do not linger and a long
// diagnostic cannot pin the bar indefinitely.
const (
	noticeBaseFrames    = 90
	noticeFramesPerRune = 4
	noticeMinFrames     = 180
	noticeMaxFrames     = 540
)

// NoticeFrames returns how long a transient HUD notice should stay visible.
func NoticeFrames(message string) int {
	frames := noticeBaseFrames + noticeFramesPerRune*utf8.RuneCountInString(message)
	return min(max(frames, noticeMinFrames), noticeMaxFrames)
}
