package storage

// orNoGuard substitutes a no-op for an absent panic hook, so owned goroutines
// and callbacks can always `defer panicGuard()`.
func orNoGuard(panicGuard func()) func() {
	if panicGuard == nil {
		return func() {}
	}
	return panicGuard
}
