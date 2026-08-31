package gameapi

import "testing"

func TestEscarpmentBlocksCardinalAndDiagonalMoves(t *testing.T) {
	frame := &Frame{
		Tiles: []Tile{
			{ID: 0, X: 0, Y: 0}, {ID: 1, X: 1, Y: 0},
			{ID: 2, X: 0, Y: 1}, {ID: 3, X: 1, Y: 1},
		},
		Escarpments: []Escarpment{{Name: "Test Front", First: 0, Second: 1}},
	}
	if !EscarpmentBlocks(frame, 0, 1) || !EscarpmentBlocks(frame, 1, 0) {
		t.Fatal("cardinal escarpment is not symmetric")
	}
	if !EscarpmentBlocks(frame, 0, 3) {
		t.Fatal("diagonal move cut across an enclosing escarpment")
	}
	if EscarpmentBlocks(frame, 0, 2) || EscarpmentBlocks(frame, 0, 99) {
		t.Fatal("unblocked or invalid move was reported as an escarpment")
	}
}
