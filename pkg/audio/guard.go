package audio

// orNoGuard substitutes a no-op for an absent panic hook, so owned goroutines
// can always `defer panicGuard()`.
func orNoGuard(panicGuard func()) func() {
	if panicGuard == nil {
		return func() {}
	}
	return panicGuard
}
