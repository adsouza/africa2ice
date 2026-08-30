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
	0x0000000000000000,
	0x0000000000000001,
	0x0000000000000002,
	0x0000000000000003,
	0x9e3779b97f4a7c15,
	0xd1b54a32d192ed03,
	0x94d049bb133111eb,
	0xffffffffffffffff,
}
