package ui

type Toast struct {
	Message         string
	Error           bool
	RemainingFrames int
}

// ToastManager is a four-entry, coalescing FIFO. Storage completion bursts are
// bounded, and an error can displace the oldest queued success when full.
type ToastManager struct {
	entries [4]Toast
	count   int
}

func (manager *ToastManager) Push(message string, isError bool, frames int) {
	if manager == nil || message == "" || frames <= 0 {
		return
	}
	for index := 0; index < manager.count; index++ {
		if manager.entries[index].Message == message && manager.entries[index].Error == isError {
			manager.entries[index].RemainingFrames = max(manager.entries[index].RemainingFrames, frames)
			return
		}
	}
	toast := Toast{Message: message, Error: isError, RemainingFrames: frames}
	if manager.count < len(manager.entries) {
		manager.entries[manager.count] = toast
		manager.count++
		return
	}
	if !isError {
		return
	}
	for index := 0; index < manager.count; index++ {
		if manager.entries[index].Error {
			continue
		}
		copy(manager.entries[index:], manager.entries[index+1:manager.count])
		manager.entries[manager.count-1] = toast
		return
	}
}

func (manager *ToastManager) Tick() {
	if manager == nil || manager.count == 0 {
		return
	}
	manager.entries[0].RemainingFrames--
	if manager.entries[0].RemainingFrames > 0 {
		return
	}
	copy(manager.entries[:], manager.entries[1:manager.count])
	manager.count--
	manager.entries[manager.count] = Toast{}
}

func (manager *ToastManager) Current() (Toast, bool) {
	if manager == nil || manager.count == 0 {
		return Toast{}, false
	}
	return manager.entries[0], true
}
