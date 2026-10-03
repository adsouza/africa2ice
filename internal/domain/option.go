package domain

// Option holds a value or nothing; its zero value is nothing. It replaces a
// presence flag stored beside its value, which let the value outlive the flag
// and reach saves and state hashes as dead state. Unlike a pointer it copies by
// value, so a Band holding one keeps no reference fields and the copies
// World.Bands() and ExportState return stay isolated from the aggregate.
type Option[T any] struct {
	value T
	ok    bool
}

func Some[T any](value T) Option[T] { return Option[T]{value: value, ok: true} }

// Get returns the value and whether one is present; absent, it returns T's zero.
func (option Option[T]) Get() (T, bool) { return option.value, option.ok }

func (option Option[T]) Present() bool { return option.ok }

// MigrationOrder is a move queued for phase 4: from Origin to Destination,
// across Passage when the move is a passage crossing.
type MigrationOrder struct {
	Destination TileID
	Origin      TileID
	Passage     Option[PassageID]
}
