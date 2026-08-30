package domain

type CampaignResult uint8

const (
	CampaignOngoing CampaignResult = iota
	CampaignVictory
	CampaignExtinction
	CampaignDispersalFailed
)

const ExplorationWordCount = (TileCount + 63) / 64

type State struct {
	Seed               uint64
	Turn               int
	Result             CampaignResult
	NextBandID         BandID
	Bands              []Band
	Tiles              [TileCount]TileState
	ExploredTiles      [ExplorationWordCount]uint64
	EstablishedRegions uint16
	RNGState           []byte
}
