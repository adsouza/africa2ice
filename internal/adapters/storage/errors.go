package storage

import (
	"fmt"
	"os"
)

// errSlotEmpty is what both backends report for a slot that holds no save,
// whether it was never written or a tombstone deleted it. It wraps
// os.ErrNotExist so callers can test for emptiness without matching message
// text, and the message is the same on desktop and in the browser.
var errSlotEmpty = fmt.Errorf("save slot is empty: %w", os.ErrNotExist)
