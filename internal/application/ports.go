package application

import "time"

type RepositoryOpID uint64
type SlotID int

const (
	Manual1   SlotID = 1
	Manual2   SlotID = 2
	Manual3   SlotID = 3
	QuickSave SlotID = 99
	Auto1     SlotID = 101
	Auto2     SlotID = 102
	Auto3     SlotID = 103
)

type RepositoryOperation uint8

const (
	RepositoryWrite RepositoryOperation = iota
	RepositoryRead
	RepositoryDelete
	RepositoryList
)

type SaveMetadata struct {
	SlotID                 SlotID    `json:"slot_id"`
	CommitSequence         uint64    `json:"commit_sequence"`
	Deleted                bool      `json:"deleted"`
	Generation             string    `json:"generation,omitempty"`
	SchemaVersion          int       `json:"schema_version,omitempty"`
	CampaignClockAlgorithm string    `json:"campaign_clock_algorithm,omitempty"`
	WorldRevision          uint64    `json:"world_revision,omitempty"`
	Turn                   int       `json:"turn,omitempty"`
	SapiensPopulation      uint64    `json:"sapiens_population,omitempty"`
	ArchaicPopulation      uint64    `json:"archaic_population,omitempty"`
	SavedAt                time.Time `json:"saved_at,omitempty"`
}

type RepositoryCompletion struct {
	OperationID RepositoryOpID
	Operation   RepositoryOperation
	SlotID      SlotID
	State       *SaveState
	Metadata    *SaveMetadata
	Slots       []SaveMetadata
	Writable    bool
	Err         error
}

type CampaignRepository interface {
	BeginWrite(RepositoryOpID, SlotID, SaveState) error
	BeginRead(RepositoryOpID, SlotID) error
	BeginDelete(RepositoryOpID, SlotID) error
	BeginList(RepositoryOpID) error
	Poll() []RepositoryCompletion
	Writable() bool
	Close() error
}

func ValidSlot(slot SlotID) bool {
	switch slot {
	case Manual1, Manual2, Manual3, QuickSave, Auto1, Auto2, Auto3:
		return true
	default:
		return false
	}
}
