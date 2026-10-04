package storage

// Availability is a save store's standing condition, apart from any one
// operation's result. Only the browser store ever leaves AvailabilityReady:
// another tab can hold its database at an older version, or upgrade or delete
// it from under this one (DESIGN.md §9).
type Availability uint8

const (
	AvailabilityReady Availability = iota
	// AvailabilityUpgradeBlocked: another tab holds the database open at an
	// older version. Operations fail until that tab closes or reloads, and
	// then the store becomes ready without a reload.
	AvailabilityUpgradeBlocked
	// AvailabilityReloadRequired: the database was upgraded or deleted from
	// elsewhere, or could not be opened after a blocked upgrade. The
	// connection is closed and the writer lease released; only a reload
	// recovers.
	AvailabilityReloadRequired
)
