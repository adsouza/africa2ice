package gameapi

import "time"

type StorageOpID uint64

type StorageOperation uint8

const (
	StorageSave StorageOperation = iota
	StorageLoad
	StorageDelete
	StorageList
)

type SlotKind uint8

const (
	ManualSlot SlotKind = iota
	QuickSlot
	AutoSlot
)

type SlotMetadata struct {
	SlotID                 int
	SlotKind               SlotKind
	CommitSequence         uint64
	WorldRevision          uint64
	Turn                   int
	YearBP                 int
	Era                    CampaignEra
	SapiensPopulation      uint64
	SavedAt                time.Time
	CampaignClockAlgorithm string
	StateHash              string
}

type StorageResult struct {
	OperationID      StorageOpID
	Operation        StorageOperation
	Slot             int
	WorldRevision    uint64
	Metadata         *SlotMetadata
	Slots            []SlotMetadata
	StorageWritable  bool
	ReplacementFrame *Frame
	Err              error
}
