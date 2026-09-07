package gameapi

type Command interface {
	isCommand()
	ActingBandID() BandID
}

type SetAssignment struct {
	BandID       BandID
	AllocationBP [AssignmentCount]uint16
}

func (SetAssignment) isCommand()             {}
func (c SetAssignment) ActingBandID() BandID { return c.BandID }

type QueueMigration struct {
	BandID BandID
	TileID TileID
}

func (QueueMigration) isCommand()             {}
func (c QueueMigration) ActingBandID() BandID { return c.BandID }

type SplitBand struct {
	BandID      BandID
	Destination TileID
}

func (SplitBand) isCommand()             {}
func (c SplitBand) ActingBandID() BandID { return c.BandID }

type ResearchTech struct {
	BandID BandID
	Tech   Tech
}

func (ResearchTech) isCommand()             {}
func (c ResearchTech) ActingBandID() BandID { return c.BandID }

type Interbreed struct {
	BandID       BandID
	TargetBandID BandID
}

func (Interbreed) isCommand()             {}
func (c Interbreed) ActingBandID() BandID { return c.BandID }

// SetEasyMode changes the gameplay difficulty immediately.
type SetEasyMode struct{ Enabled bool }

func (SetEasyMode) isCommand()           {}
func (SetEasyMode) ActingBandID() BandID { return 0 }
