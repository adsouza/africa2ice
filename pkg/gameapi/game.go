package gameapi

type CampaignUseCases interface {
	Snapshot() (*Frame, error)
	StateHash() (string, error)
	NewCampaign() (*Frame, error)
	Apply(Command) (*Frame, error)
	EndTurn() (*Frame, error)
}

type StorageUseCases interface {
	BeginSave(slot int) (StorageOpID, error)
	BeginLoad(slot int) (StorageOpID, error)
	BeginDelete(slot int) (StorageOpID, error)
	BeginListSlots() (StorageOpID, error)
	PollStorage() []StorageResult
}

type Game interface {
	CampaignUseCases
	StorageUseCases
}
