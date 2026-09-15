package app

// Close releases resources acquired by the composed host. It is idempotent.
// Games built with New or NewWithSound borrow their ports and do not close them.
// Call after the game loop stops; Close does not drain pending saves.
func (g *Game) Close() error {
	if g == nil || g.closeResources == nil {
		return nil
	}
	return g.closeResources()
}
