package main

import "errors"

// closable is all the lifetime seam needs from the composed host.
type closable interface{ Close() error }

// runGameLoop owns the composed host's lifetime for both entry points: it closes
// the game when the loop returns, again during panic unwinding, and reports a
// close failure alongside a loop failure. app.Game.Close is idempotent, so the
// two paths cannot double-release the repository lease.
//
// It is a variable, and the only route from an entry point to Close, so a test
// can assert the wiring. Close has unit tests of its own; those stay green if a
// main() stops calling it, which is the failure this seam exists to catch.
var runGameLoop = defaultRunGameLoop

func defaultRunGameLoop(game closable, loop func() error) error {
	defer func() { _ = game.Close() }()
	return errors.Join(loop(), game.Close())
}
