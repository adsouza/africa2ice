package application

import "time"

const autosaveFallbackInterval = 5 * time.Minute

type monotonicClock interface {
	Now() time.Time
}

type systemMonotonicClock struct{}

func (systemMonotonicClock) Now() time.Time { return time.Now() }

func (service *GameService) resetAutosaveClock() {
	now := service.clock.Now()
	service.lastAutosaveRevision = service.worldRevision
	service.nextAutosaveFallback = now.Add(autosaveFallbackInterval)
}

func (service *GameService) pollAutosaveClock() {
	now := service.clock.Now()
	if now.Before(service.nextAutosaveFallback) {
		return
	}
	// Advance the deadline when the fallback is requested, not on every idle
	// poll. A failed write therefore retries only after a fresh five-minute
	// interval (or a completed turn), never every Ebitengine update.
	service.nextAutosaveFallback = now.Add(autosaveFallbackInterval)
	if service.worldRevision > service.lastAutosaveRevision {
		service.autoNeeded = true
	}
}

func (service *GameService) observeAutosaveMetadata(metadata SaveMetadata) {
	index, ok := autosaveSlotIndex(metadata.SlotID)
	if !ok {
		return
	}
	if metadata.Deleted {
		service.autosaveCommitSequences[index] = 0
		return
	}
	service.autosaveCommitSequences[index] = metadata.CommitSequence
}

func (service *GameService) chooseAutosaveSlot() SlotID {
	selected := 0
	for index := 1; index < len(service.autosaveCommitSequences); index++ {
		if service.autosaveCommitSequences[index] < service.autosaveCommitSequences[selected] {
			selected = index
		}
	}
	return SlotID(int(Auto1) + selected)
}

func autosaveSlotIndex(slot SlotID) (int, bool) {
	if slot < Auto1 || slot > Auto3 {
		return 0, false
	}
	return int(slot - Auto1), true
}

func (service *GameService) acceptSuccessfulAutosave(request *storageRequest, metadata *SaveMetadata) {
	if metadata != nil {
		service.observeAutosaveMetadata(*metadata)
	} else if index, ok := autosaveSlotIndex(request.slot); ok {
		// Repository adapters return metadata. Keeping a deterministic fallback
		// makes test doubles and future minimal adapters preserve oldest-first
		// rotation instead of repeatedly selecting Auto 1.
		var maximum uint64
		for _, sequence := range service.autosaveCommitSequences {
			maximum = max(maximum, sequence)
		}
		service.autosaveCommitSequences[index] = maximum + 1
	}
	service.lastAutosaveRevision = request.revision
	service.nextAutosaveFallback = service.clock.Now().Add(autosaveFallbackInterval)
	if service.worldRevision > request.revision {
		service.autoNeeded = true
	}
}
