package app

import "github.com/adsouza/africa2ice/pkg/gameapi"

type skeletonPort struct{ frame gameapi.Frame }

func newSkeletonPort() *skeletonPort {
	return &skeletonPort{frame: gameapi.Frame{
		WorldRevision:    1,
		TerrainRevision:  1,
		Turn:             0,
		YearBP:           80_000,
		Era:              gameapi.EraEarly,
		Season:           gameapi.SeasonWarm,
		CampaignResult:   gameapi.Ongoing,
		CalendarProgress: 0,
	}}
}

func (p *skeletonPort) Snapshot() (*gameapi.Frame, error)             { frame := p.frame; return &frame, nil }
func (*skeletonPort) StateHash() (string, error)                      { return "walking-skeleton", nil }
func (p *skeletonPort) NewCampaign() (*gameapi.Frame, error)          { return p.Snapshot() }
func (p *skeletonPort) Apply(gameapi.Command) (*gameapi.Frame, error) { return p.Snapshot() }
func (p *skeletonPort) EndTurn() (*gameapi.Frame, error)              { return p.Snapshot() }
func (p *skeletonPort) BeginSave(int) (gameapi.StorageOpID, error)    { return 1, nil }
func (p *skeletonPort) BeginLoad(int) (gameapi.StorageOpID, error)    { return 1, nil }
func (p *skeletonPort) BeginDelete(int) (gameapi.StorageOpID, error)  { return 1, nil }
func (p *skeletonPort) BeginListSlots() (gameapi.StorageOpID, error)  { return 1, nil }
func (p *skeletonPort) PollStorage() []gameapi.StorageResult          { return nil }

var _ gameapi.Game = (*skeletonPort)(nil)
