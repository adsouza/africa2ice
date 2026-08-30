package domain

// BalanceSeedCorpus is the fixed set of campaign seeds DESIGN.md names as the
// shared input to the step-5e domain viability gate, the step-4 LocalTemperatureC
// sweep, and the step-12 balance pass. It is exported because the verification
// harness and the balance pass both run outside this package and must use the
// same corpus rather than each inventing one.
//
// The corpus is deliberately small and fixed. Its purpose is not to sample the
// uint64 seed space — 2^64 makes that meaningless — but to give every pass that
// claims "the campaign is viable" the same reproducible evidence, so a tuning
// change that helps one seed and ruins another cannot hide behind a fresh draw.
//
// Adding or removing a seed changes what every one of those gates measured, so
// treat this list as frozen: extend it only alongside a re-run and a re-record
// of the margins each gate reports.
var BalanceSeedCorpus = [8]uint64{
	0x9e3779b97f4a7c15, // the golden-ratio constant used as the default campaign seed
	0x0000000000000000, // all-zero: the degenerate seed a splitmix bug would expose
	0xffffffffffffffff, // all-ones, the opposite extreme
	0x0123456789abcdef,
	0xfedcba9876543210,
	0x00000000000003e8,
	0x5851f42d4c957f2d, // the PCG multiplier, a value adjacent to the generator's own state space
	0x2545f4914f6cdd1d,
}
