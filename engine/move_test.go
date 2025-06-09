package engine_test

import (
	"silverfish/engine"
	"testing"
)

func TestGetMovesFromBitboard(t *testing.T) {
	gotMoves := engine.GetMovesFromBitboard(engine.SquareA1, 0x000000000101010e)
	expectedMoves := []engine.Move{
		engine.NewMove(engine.SquareA1, engine.SquareA2),
		engine.NewMove(engine.SquareA1, engine.SquareA3),
		engine.NewMove(engine.SquareA1, engine.SquareA4),
		engine.NewMove(engine.SquareA1, engine.SquareB1),
		engine.NewMove(engine.SquareA1, engine.SquareC1),
		engine.NewMove(engine.SquareA1, engine.SquareD1),
	}

	if !equalSets(gotMoves, expectedMoves) {
		t.Errorf("Expected: %#v\nGot: %#v\n", expectedMoves, gotMoves)
	}
}
