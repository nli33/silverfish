package engine_test

import (
	"testing"

	"silverfish/engine"
)

// The NNUE accumulator is updated incrementally (Add/Remove) on every
// DoMove/UndoMove rather than recomputed from scratch. This test checks that
// incremental updates stay consistent with a from-scratch build: after
// playing a sequence of moves, the position's accumulator-derived evaluation
// must match that of a fresh Position built straight from the resulting FEN
// (which only ever calls Add, never Remove).
func TestNNUEIncrementalMatchesFromScratch(t *testing.T) {
	pos := engine.FromFEN("6k1/5p1p/1q2p1p1/1PnpP3/3N4/1Pr5/P5PP/3QR1K1 w - - 3 37")

	moves := []string{"d1a1", "b6a5", "d4c6"}
	for _, m := range moves {
		move := engine.NewMoveFromStr(m)
		var legalMove engine.Move
		found := false
		for _, candidate := range pos.LegalMoves() {
			if candidate.From() == move.From() && candidate.To() == move.To() {
				legalMove, found = candidate, true
				break
			}
		}
		if !found {
			t.Fatalf("move %s not found in legal moves", m)
		}
		pos.DoMove(legalMove)
	}

	fresh := engine.FromFEN(pos.ToFEN())

	got := engine.Evaluate(&pos)
	want := engine.Evaluate(&fresh)
	if got != want {
		t.Errorf("incrementally updated eval = %d, from-scratch eval = %d; want equal", got, want)
	}
}
