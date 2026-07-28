package engine_test

import (
	"testing"
	"time"

	"silverfish/engine"
)

// Structural invariants that must hold regardless of how the evaluation is
// tuned: the search returns a legal move, doesn't mutate the position it
// started from, and explores at least one node.
func TestSearchInvariants(t *testing.T) {
	fen := "r3k2r/p1ppqpb1/bn2pnp1/3PN3/1p2P3/2N2Q1p/PPPBBPPP/R3K2R w KQkq - 0 1"
	pos := engine.FromFEN(fen)
	before := pos.ToFEN()

	search := engine.Search{
		MaxDepth:  3,
		TimeLimit: engine.InfiniteMovetime,
	}
	search.Init(&pos)

	_, bestMove := search.Search()

	if bestMove == engine.Move(0) {
		t.Fatalf("Search() returned a null move")
	}
	if !pos.MoveIsLegal(bestMove) {
		t.Errorf("Search() returned illegal move %s", bestMove.ToString())
	}
	if pos.ToFEN() != before {
		t.Errorf("Search() mutated the position: before %q, after %q", before, pos.ToFEN())
	}
	if search.Nodes == 0 {
		t.Errorf("Search() explored 0 nodes")
	}
}

// A fixed-depth search with no time pressure must be reproducible.
func TestSearchReproducible(t *testing.T) {
	fen := "r1bqkbnr/pppp1ppp/2n5/4p3/4P3/5N2/PPPP1PPP/RNBQKB1R w KQkq - 2 3"

	run := func() (int32, engine.Move) {
		pos := engine.FromFEN(fen)
		search := engine.Search{
			MaxDepth:  3,
			TimeLimit: engine.InfiniteMovetime,
		}
		search.Init(&pos)
		return search.Search()
	}

	wantScore, wantMove := run()
	for i := 0; i < 3; i++ {
		gotScore, gotMove := run()
		if gotScore != wantScore || gotMove != wantMove {
			t.Errorf("run %d: got (%d, %s), want (%d, %s)", i, gotScore, gotMove.ToString(), wantScore, wantMove.ToString())
		}
	}
}

// Forced mates: any correct search must find them, independent of eval tuning.
func TestSearchFindsMateInOne(t *testing.T) {
	cases := []struct {
		name string
		fen  string
	}{
		{"back rank mate", "6k1/5ppp/8/8/8/8/8/R5K1 w - - 0 1"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			pos := engine.FromFEN(tc.fen)
			search := engine.Search{
				MaxDepth:  2,
				TimeLimit: 4 * time.Second,
			}
			search.Init(&pos)

			score, bestMove := search.Search()
			if score < engine.Infinity-10 {
				t.Errorf("%s: score = %d, want a mate score near +Infinity", tc.name, score)
			}

			pos.DoMove(bestMove)
			if len(pos.LegalMoves()) != 0 || pos.Checkers(pos.Turn) == 0 {
				t.Errorf("%s: move %s is not checkmate", tc.name, bestMove.ToString())
			}
		})
	}
}

// A free, undefended piece with no counterplay must be captured by any
// correct search -- independent of eval tuning, since forfeiting it is a
// clear material loss under any reasonable evaluation.
func TestSearchCapturesHangingPiece(t *testing.T) {
	pos := engine.FromFEN("4k3/8/8/7q/5N2/8/8/4K3 w - - 0 1")
	search := engine.Search{
		MaxDepth:  3,
		TimeLimit: 4 * time.Second,
	}
	search.Init(&pos)

	_, bestMove := search.Search()
	if bestMove.To() != engine.SquareH5 {
		t.Errorf("Search() = %s, want a move to h5 capturing the undefended queen", bestMove.ToString())
	}
}
