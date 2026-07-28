package engine_test

import (
	"testing"

	"silverfish/engine"
)

// EvaluateHCE reports the score relative to the side to move, so flipping
// Turn on an otherwise-unchanged position must exactly negate the result.
// This holds regardless of how the material/PST tables are tuned, unlike
// asserting hardcoded centipawn values.
//
// EvaluateHCE is currently unused by Evaluate (which calls EvaluateNNUE) but
// is kept as a manually selectable evaluation, so it stays covered.
func TestEvaluateHCESideToMoveSymmetry(t *testing.T) {
	fens := []string{
		"rnbqkbnr/pppppppp/8/8/8/8/PPPPPPPP/RNBQKBNR w KQkq - 0 1",
		"r1bqkbnr/pppp1ppp/2n5/4p3/4P3/5N2/PPPP1PPP/RNBQKB1R w KQkq - 2 3",
		"8/8/8/4k3/8/4K3/4P3/8 w - - 0 1",
		"r3k2r/p1ppqpb1/bn2pnp1/3PN3/1p2P3/2N2Q1p/PPPBBPPP/R3K2R w KQkq - 0 1",
	}

	for _, fen := range fens {
		pos := engine.FromFEN(fen)

		white := engine.EvaluateHCE(&pos)
		pos.Turn = engine.Black
		black := engine.EvaluateHCE(&pos)

		if white != -black {
			t.Errorf("EvaluateHCE(%q): white-to-move = %d, black-to-move = %d; want negation", fen, white, black)
		}
	}
}
